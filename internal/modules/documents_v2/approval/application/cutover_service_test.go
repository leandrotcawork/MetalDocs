package application

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Minimal fake driver for cutover tests.
//
// ValidateLegacyCutoverReady issues exactly one query:
//   SELECT COUNT(*) FROM documents WHERE status IN ('finalized','archived')
//
// cutoverTestConn returns a configurable count for that query.
// ---------------------------------------------------------------------------

type cutoverCountRows struct {
	count int64
	done  bool
}

func (r *cutoverCountRows) Columns() []string { return []string{"count"} }
func (r *cutoverCountRows) Close() error      { return nil }
func (r *cutoverCountRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.count
	return nil
}

type cutoverTestStmt struct {
	conn *cutoverTestConn
}

func (s *cutoverTestStmt) Close() error                              { return nil }
func (s *cutoverTestStmt) NumInput() int                             { return -1 }
func (s *cutoverTestStmt) Exec(_ []driver.Value) (driver.Result, error) { return nil, nil }
func (s *cutoverTestStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &cutoverCountRows{count: s.conn.legacyCount}, nil
}

type cutoverTestConn struct {
	legacyCount int64
}

func (c *cutoverTestConn) Prepare(query string) (driver.Stmt, error) {
	return &cutoverTestStmt{conn: c}, nil
}
func (c *cutoverTestConn) Close() error              { return nil }
func (c *cutoverTestConn) Begin() (driver.Tx, error) { return c, nil }
func (c *cutoverTestConn) Commit() error             { return nil }
func (c *cutoverTestConn) Rollback() error           { return nil }

type cutoverTestDriver struct{ conn *cutoverTestConn }

func (d *cutoverTestDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

func newCutoverTestDB(t *testing.T, conn *cutoverTestConn) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("cutover_test_%p", conn)
	sql.Register(name, &cutoverTestDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open cutover test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestValidateLegacyCutoverReady_NoLegacyDocs verifies that the service
// returns nil when no documents carry a legacy status.
func TestValidateLegacyCutoverReady_NoLegacyDocs(t *testing.T) {
	conn := &cutoverTestConn{legacyCount: 0}
	db := newCutoverTestDB(t, conn)

	svc := NewCutoverService(&MemoryEmitter{}, fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)})
	err := svc.ValidateLegacyCutoverReady(context.Background(), db)
	if err != nil {
		t.Fatalf("ValidateLegacyCutoverReady: unexpected error: %v", err)
	}
}

// TestValidateLegacyCutoverReady_LegacyDocsRemain verifies that the service
// returns ErrLegacyDocumentsRemain (unwrappable via errors.Is) when legacy
// documents are still present, and that the error message includes the count.
func TestValidateLegacyCutoverReady_LegacyDocsRemain(t *testing.T) {
	conn := &cutoverTestConn{legacyCount: 7}
	db := newCutoverTestDB(t, conn)

	svc := NewCutoverService(&MemoryEmitter{}, fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)})
	err := svc.ValidateLegacyCutoverReady(context.Background(), db)
	if err == nil {
		t.Fatal("ValidateLegacyCutoverReady: expected error; got nil")
	}
	if !errors.Is(err, ErrLegacyDocumentsRemain) {
		t.Errorf("errors.Is(err, ErrLegacyDocumentsRemain) = false; err = %v", err)
	}
	// The count should appear in the error message.
	errMsg := err.Error()
	const wantSubstr = "7"
	if len(errMsg) == 0 {
		t.Error("error message is empty")
	}
	found := false
	for i := 0; i+len(wantSubstr) <= len(errMsg); i++ {
		if errMsg[i:i+len(wantSubstr)] == wantSubstr {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("error message %q does not contain count %q", errMsg, wantSubstr)
	}
}
