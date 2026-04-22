package idempotency_janitor

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

type janitorDBState struct {
	mu sync.Mutex

	rowsAffected []int64
	execCalls    int
	execErr      error
}

func (s *janitorDBState) nextExec() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.execCalls++
	if s.execErr != nil {
		return 0, s.execErr
	}
	if len(s.rowsAffected) == 0 {
		return 0, nil
	}
	n := s.rowsAffected[0]
	s.rowsAffected = s.rowsAffected[1:]
	return n, nil
}

func (s *janitorDBState) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.execCalls
}

type janitorDriver struct {
	state *janitorDBState
}

func (d *janitorDriver) Open(_ string) (driver.Conn, error) {
	return &janitorConn{state: d.state}, nil
}

type janitorConn struct {
	state *janitorDBState
}

func (c *janitorConn) Prepare(query string) (driver.Stmt, error) {
	return &janitorStmt{state: c.state, query: query}, nil
}

func (c *janitorConn) Close() error { return nil }

func (c *janitorConn) Begin() (driver.Tx, error) { return janitorTx{}, nil }

type janitorTx struct{}

func (janitorTx) Commit() error   { return nil }
func (janitorTx) Rollback() error { return nil }

type janitorStmt struct {
	state *janitorDBState
	query string
}

func (s *janitorStmt) Close() error  { return nil }
func (s *janitorStmt) NumInput() int { return -1 }

func (s *janitorStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if !strings.Contains(strings.ToLower(s.query), "delete from metaldocs.idempotency_keys") {
		return janitorResult(0), nil
	}
	n, err := s.state.nextExec()
	if err != nil {
		return nil, err
	}
	return janitorResult(n), nil
}

func (s *janitorStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return nil, errors.New("query not supported")
}

type janitorResult int64

func (r janitorResult) LastInsertId() (int64, error) { return 0, nil }
func (r janitorResult) RowsAffected() (int64, error) { return int64(r), nil }

func newJanitorDB(t *testing.T, state *janitorDBState) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("idempotency_janitor_%p", state)
	sql.Register(name, &janitorDriver{state: state})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open janitor db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestJanitor_NoExpired(t *testing.T) {
	t.Parallel()

	state := &janitorDBState{rowsAffected: []int64{0}}
	db := newJanitorDB(t, state)

	fn := New(db)
	if err := fn(context.Background(), 1); err != nil {
		t.Fatalf("job returned error: %v", err)
	}

	if got := state.callCount(); got != 1 {
		t.Fatalf("ExecContext calls = %d, want 1", got)
	}
}

func TestJanitor_SomeExpired(t *testing.T) {
	t.Parallel()

	state := &janitorDBState{rowsAffected: []int64{100, 0}}
	db := newJanitorDB(t, state)

	fn := New(db)
	if err := fn(context.Background(), 2); err != nil {
		t.Fatalf("job returned error: %v", err)
	}

	if got := state.callCount(); got != 2 {
		t.Fatalf("ExecContext calls = %d, want 2", got)
	}
}

func TestJanitor_ExecError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("delete failed")
	state := &janitorDBState{execErr: expectedErr}
	db := newJanitorDB(t, state)

	fn := New(db)
	err := fn(context.Background(), 3)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("err = %v, want %v", err, expectedErr)
	}
}
