package application

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"testing"
)

// --- minimal in-memory SQL driver for testing SET LOCAL ordering ---

type recordingDriver struct{}
type recordingConn struct {
	executed []string
	failAt   string
}

type noopResult struct{}
func (noopResult) LastInsertId() (int64, error) { return 0, nil }
func (noopResult) RowsAffected() (int64, error) { return 1, nil }

type emptyRows struct{}
func (emptyRows) Columns() []string              { return nil }
func (emptyRows) Close() error                   { return nil }
func (emptyRows) Next([]driver.Value) error      { return io.EOF }

func (d *recordingDriver) Open(_ string) (driver.Conn, error) {
	return &recordingConn{}, nil
}

func (c *recordingConn) Prepare(query string) (driver.Stmt, error) {
	return &recordingStmt{conn: c, query: query}, nil
}
func (c *recordingConn) Close() error  { return nil }
func (c *recordingConn) Begin() (driver.Tx, error) { return c, nil }
func (c *recordingConn) Commit() error   { return nil }
func (c *recordingConn) Rollback() error { return nil }

type recordingStmt struct {
	conn  *recordingConn
	query string
}
func (s *recordingStmt) Close() error                                    { return nil }
func (s *recordingStmt) NumInput() int                                   { return -1 }
func (s *recordingStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.conn.executed = append(s.conn.executed, s.query)
	if s.conn.failAt != "" && s.query == s.conn.failAt {
		return nil, fmt.Errorf("injected error: %s", s.query)
	}
	return noopResult{}, nil
}
func (s *recordingStmt) Query(args []driver.Value) (driver.Rows, error) {
	return emptyRows{}, nil
}

var driverOnce bool

func newTestDB(t *testing.T, conn *recordingConn) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("recording_%p", conn)
	sql.Register(name, &recordingDriverInstance{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

type recordingDriverInstance struct{ conn *recordingConn }
func (d *recordingDriverInstance) Open(_ string) (driver.Conn, error) { return d.conn, nil }

func TestMembershipTxGUCOrder(t *testing.T) {
	conn := &recordingConn{}
	db := newTestDB(t, conn)

	err := WithMembershipContext(context.Background(), db, "actor-1", "workflow.route.edit", func(tx *sql.Tx) error {
		return nil
	})
	if err != nil {
		t.Fatalf("WithMembershipContext: %v", err)
	}

	want := []string{
		"SET LOCAL ROLE metaldocs_membership_writer",
		"SET LOCAL metaldocs.actor_id = $1",
		"SET LOCAL metaldocs.verified_capability = $1",
	}
	if len(conn.executed) < len(want) {
		t.Fatalf("executed %d statements; want at least %d: %v", len(conn.executed), len(want), conn.executed)
	}
	for i, w := range want {
		if conn.executed[i] != w {
			t.Errorf("stmt[%d] = %q; want %q", i, conn.executed[i], w)
		}
	}
}

func TestMembershipTxRollbackOnError(t *testing.T) {
	conn := &recordingConn{}
	db := newTestDB(t, conn)

	sentinel := errors.New("fn error")
	err := WithMembershipContext(context.Background(), db, "actor-1", "cap", func(tx *sql.Tx) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("want sentinel error; got %v", err)
	}
}

func TestMembershipTxEmptyActorRejected(t *testing.T) {
	conn := &recordingConn{}
	db := newTestDB(t, conn)

	err := WithMembershipContext(context.Background(), db, "", "cap", func(tx *sql.Tx) error {
		return nil
	})
	if !errors.Is(err, ErrNoActor) {
		t.Errorf("want ErrNoActor; got %v", err)
	}
	// No statements should have executed.
	if len(conn.executed) > 0 {
		t.Error("no SQL should execute when actor is empty")
	}
}
