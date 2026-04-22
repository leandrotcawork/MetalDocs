package application

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// ---------------------------------------------------------------------------
// Fake driver for supersede tests.
//
// Two UPDATE statements are issued per call to PublishSuperseding:
//   1. UPDATE documents SET status='published'  (new doc OCC)
//   2. UPDATE documents SET status='superseded' (prior doc OCC)
//
// supersedeTestConn tracks which UPDATE is being executed (via a call counter)
// and returns the caller-configured rowsAffected for each.
// Non-UPDATE statements (BEGIN, COMMIT, governance event INSERT) always succeed.
// ---------------------------------------------------------------------------

type supersedeTestResult struct{ rowsAffected int64 }

func (r supersedeTestResult) LastInsertId() (int64, error) { return 0, nil }
func (r supersedeTestResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type supersedeEmptyRows struct{}

func (supersedeEmptyRows) Columns() []string         { return nil }
func (supersedeEmptyRows) Close() error              { return nil }
func (supersedeEmptyRows) Next([]driver.Value) error { return io.EOF }

type supersedeTestStmt struct {
	conn  *supersedeTestConn
	query string
}

func (s *supersedeTestStmt) Close() error  { return nil }
func (s *supersedeTestStmt) NumInput() int { return -1 }

func (s *supersedeTestStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if !strings.Contains(strings.ToLower(s.query), "update") {
		// Non-UPDATE (governance_events INSERT, etc.) always succeed with 1.
		return supersedeTestResult{rowsAffected: 1}, nil
	}
	// Track which UPDATE call this is.
	s.conn.updateCount++
	switch s.conn.updateCount {
	case 1:
		return supersedeTestResult{rowsAffected: s.conn.newDocRowsAffected}, nil
	default:
		return supersedeTestResult{rowsAffected: s.conn.priorDocRowsAffected}, nil
	}
}

func (s *supersedeTestStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return supersedeEmptyRows{}, nil
}

type supersedeTestConn struct {
	newDocRowsAffected   int64
	priorDocRowsAffected int64
	updateCount          int // incremented on each UPDATE exec
}

func (c *supersedeTestConn) Prepare(query string) (driver.Stmt, error) {
	return &supersedeTestStmt{conn: c, query: query}, nil
}

func (c *supersedeTestConn) Close() error              { return nil }
func (c *supersedeTestConn) Begin() (driver.Tx, error) { return c, nil }
func (c *supersedeTestConn) Commit() error             { return nil }
func (c *supersedeTestConn) Rollback() error           { return nil }

type supersedeTestDriver struct{ conn *supersedeTestConn }

func (d *supersedeTestDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

// newSupersedeTestDB registers a unique driver and returns a *sql.DB.
// newRowsAffected controls the first UPDATE; priorRowsAffected controls the second.
func newSupersedeTestDB(t *testing.T, newRowsAffected, priorRowsAffected int64) *sql.DB {
	t.Helper()
	conn := &supersedeTestConn{
		newDocRowsAffected:   newRowsAffected,
		priorDocRowsAffected: priorRowsAffected,
	}
	name := fmt.Sprintf("supersede_test_%p", conn)
	sql.Register(name, &supersedeTestDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open supersede test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPublishSuperseding_HappyPath(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: now}

	svc := &SupersedeService{emitter: emitter, clock: clock}
	// Both UPDATE statements match one row each.
	db := newSupersedeTestDB(t, 1, 1)

	req := SupersedeRequest{
		TenantID:             "tenant-uuid-1",
		NewDocumentID:        "doc-new-uuid-1",
		PriorDocumentID:      "doc-prior-uuid-1",
		SupersededBy:         "user-1",
		NewRevisionVersion:   4,
		PriorRevisionVersion: 7,
	}

	result, err := svc.PublishSuperseding(context.Background(), db, req)
	if err != nil {
		t.Fatalf("PublishSuperseding: unexpected error: %v", err)
	}
	if result.NewDocumentStatus != "published" {
		t.Errorf("NewDocumentStatus = %q; want %q", result.NewDocumentStatus, "published")
	}
	if result.PriorDocumentStatus != "superseded" {
		t.Errorf("PriorDocumentStatus = %q; want %q", result.PriorDocumentStatus, "superseded")
	}

	if len(emitter.Events) != 1 {
		t.Fatalf("expected 1 governance event; got %d", len(emitter.Events))
	}
	ev := emitter.Events[0]
	if ev.EventType != "document_superseded" {
		t.Errorf("event type = %q; want %q", ev.EventType, "document_superseded")
	}
	if ev.ResourceID != "doc-new-uuid-1" {
		t.Errorf("event resource_id = %q; want %q", ev.ResourceID, "doc-new-uuid-1")
	}
	if ev.ActorUserID != "user-1" {
		t.Errorf("event actor_user_id = %q; want %q", ev.ActorUserID, "user-1")
	}
	if ev.TenantID != "tenant-uuid-1" {
		t.Errorf("event tenant_id = %q; want %q", ev.TenantID, "tenant-uuid-1")
	}
}

func TestPublishSuperseding_OCC_NewConflict(t *testing.T) {
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)}

	svc := &SupersedeService{emitter: emitter, clock: clock}
	// First UPDATE (new doc) returns 0 — OCC conflict.
	db := newSupersedeTestDB(t, 0, 1)

	req := SupersedeRequest{
		TenantID:             "tenant-uuid-1",
		NewDocumentID:        "doc-new-uuid-2",
		PriorDocumentID:      "doc-prior-uuid-2",
		SupersededBy:         "user-1",
		NewRevisionVersion:   3,
		PriorRevisionVersion: 5,
	}

	_, err := svc.PublishSuperseding(context.Background(), db, req)
	if err == nil {
		t.Fatal("expected ErrStaleRevision; got nil")
	}
	if !errors.Is(err, repository.ErrStaleRevision) {
		t.Errorf("expected errors.Is(err, ErrStaleRevision); got %v", err)
	}
	if len(emitter.Events) != 0 {
		t.Errorf("no governance event should be emitted on OCC conflict; got %d", len(emitter.Events))
	}
}

func TestPublishSuperseding_OCC_PriorConflict(t *testing.T) {
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)}

	svc := &SupersedeService{emitter: emitter, clock: clock}
	// First UPDATE (new doc) succeeds; second UPDATE (prior doc) returns 0 — OCC conflict.
	db := newSupersedeTestDB(t, 1, 0)

	req := SupersedeRequest{
		TenantID:             "tenant-uuid-1",
		NewDocumentID:        "doc-new-uuid-3",
		PriorDocumentID:      "doc-prior-uuid-3",
		SupersededBy:         "user-1",
		NewRevisionVersion:   2,
		PriorRevisionVersion: 9,
	}

	_, err := svc.PublishSuperseding(context.Background(), db, req)
	if err == nil {
		t.Fatal("expected ErrStaleRevision; got nil")
	}
	if !errors.Is(err, repository.ErrStaleRevision) {
		t.Errorf("expected errors.Is(err, ErrStaleRevision); got %v", err)
	}
	if len(emitter.Events) != 0 {
		t.Errorf("no governance event should be emitted on OCC conflict; got %d", len(emitter.Events))
	}
}
