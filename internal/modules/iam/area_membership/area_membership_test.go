package area_membership

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// ─── Fake driver ─────────────────────────────────────────────────────────────

// connRegistry is seeded before each test via openDB.
// Tests run sequentially (no t.Parallel) so a package-level var is safe.
var connRegistry = struct {
	result driver.Value // returned by query (e.g. UUID string)
	err    error        // returned instead of executing the statement
}{}

type fakeDriverRegistry struct{}

func (fakeDriverRegistry) Open(_ string) (driver.Conn, error) {
	return &fakeConn{result: connRegistry.result, err: connRegistry.err}, nil
}

func init() {
	sql.Register("fake_area_membership", fakeDriverRegistry{})
}

// fakeConn holds per-connection preset state.
type fakeConn struct {
	result driver.Value
	err    error
}

func (c *fakeConn) Prepare(_ string) (driver.Stmt, error) { return &fakeStmt{conn: c}, nil }
func (c *fakeConn) Close() error                          { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)             { return fakeTx{}, nil }

type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }

type fakeStmt struct{ conn *fakeConn }

func (s *fakeStmt) Close() error    { return nil }
func (s *fakeStmt) NumInput() int   { return -1 } // variadic: driver skips arg count check
func (s *fakeStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if s.conn.err != nil {
		return nil, s.conn.err
	}
	return driver.RowsAffected(1), nil
}
func (s *fakeStmt) Query(_ []driver.Value) (driver.Rows, error) {
	if s.conn.err != nil {
		return nil, s.conn.err
	}
	return &fakeRows{val: s.conn.result}, nil
}

type fakeRows struct {
	val      driver.Value
	consumed bool
}

func (r *fakeRows) Columns() []string { return []string{"col"} }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.consumed {
		return io.EOF
	}
	r.consumed = true
	dest[0] = r.val
	return nil
}

// ─── Helper ───────────────────────────────────────────────────────────────────

// openDB returns an *sql.DB backed by the fake driver seeded with result/err.
func openDB(result driver.Value, err error) *sql.DB {
	connRegistry.result = result
	connRegistry.err = err
	db, _ := sql.Open("fake_area_membership", "")
	db.SetMaxOpenConns(1)
	return db
}

func beginTx(db *sql.DB) *sql.Tx {
	tx, _ := db.BeginTx(context.Background(), nil)
	return tx
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestGrant_HappyPath(t *testing.T) {
	db := openDB("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", nil)
	defer db.Close()

	tx := beginTx(db)
	id, err := Grant(context.Background(), tx, "tid", "user@x", "AREA", "viewer", "admin@x")
	_ = tx.Rollback()

	if err != nil {
		t.Fatalf("Grant returned unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("Grant returned empty correlationID")
	}
}

func TestGrant_InsufficientPrivilege(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "42501", Message: "permission denied"}
	db := openDB(nil, pgErr)
	defer db.Close()

	tx := beginTx(db)
	_, err := Grant(context.Background(), tx, "tid", "user@x", "AREA", "viewer", "admin@x")
	_ = tx.Rollback()

	if !errors.Is(err, ErrInsufficientPrivilege) {
		t.Fatalf("expected ErrInsufficientPrivilege, got: %v", err)
	}
}

func TestRevoke_HappyPath(t *testing.T) {
	db := openDB(nil, nil)
	defer db.Close()

	tx := beginTx(db)
	err := Revoke(context.Background(), tx, "tid", "user@x", "AREA", "viewer", "admin@x")
	_ = tx.Rollback()

	if err != nil {
		t.Fatalf("Revoke returned unexpected error: %v", err)
	}
}

func TestRevoke_NotFound(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "P0002", Message: "membership not found"}
	db := openDB(nil, pgErr)
	defer db.Close()

	tx := beginTx(db)
	err := Revoke(context.Background(), tx, "tid", "user@x", "AREA", "viewer", "admin@x")
	_ = tx.Rollback()

	if !errors.Is(err, ErrMembershipNotFound) {
		t.Fatalf("expected ErrMembershipNotFound, got: %v", err)
	}
}
