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
// Fake driver for obsolete tests.
//
// MarkObsolete issues (in order):
//   1. SELECT … FOR UPDATE  (returns the configured status + revision_version)
//   2. UPDATE documents SET status='obsolete' …  (OCC; returns 1 or 0)
//   3. UPDATE approval_instances SET status='cancelled' …  (always 1 in tests)
//   4. INSERT INTO governance_events …  (always succeeds)
//
// obsoleteTestConn tracks the query sequence and returns caller-configured
// values for each step.
// ---------------------------------------------------------------------------

type obsoleteTestResult struct{ rowsAffected int64 }

func (r obsoleteTestResult) LastInsertId() (int64, error) { return 0, nil }
func (r obsoleteTestResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

// obsoleteTestRows returns a single row with (status, revision_version).
type obsoleteTestRows struct {
	status          string
	revisionVersion int
	done            bool
}

func (r *obsoleteTestRows) Columns() []string { return []string{"status", "revision_version"} }
func (r *obsoleteTestRows) Close() error      { return nil }
func (r *obsoleteTestRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.status
	dest[1] = int64(r.revisionVersion)
	return nil
}

// obsoleteEmptyRows is used when the SELECT finds no document.
type obsoleteEmptyRows struct{}

func (obsoleteEmptyRows) Columns() []string         { return []string{"status", "revision_version"} }
func (obsoleteEmptyRows) Close() error              { return nil }
func (obsoleteEmptyRows) Next([]driver.Value) error { return io.EOF }

type obsoleteTestStmt struct {
	conn  *obsoleteTestConn
	query string
}

func (s *obsoleteTestStmt) Close() error  { return nil }
func (s *obsoleteTestStmt) NumInput() int { return -1 }

func (s *obsoleteTestStmt) Exec(_ []driver.Value) (driver.Result, error) {
	lower := strings.ToLower(s.query)
	if strings.Contains(lower, "update documents") {
		// OCC UPDATE for the document itself.
		return obsoleteTestResult{rowsAffected: s.conn.docUpdateRowsAffected}, nil
	}
	// Everything else (approval_instances cancel, governance_events INSERT) succeeds.
	return obsoleteTestResult{rowsAffected: 1}, nil
}

func (s *obsoleteTestStmt) Query(_ []driver.Value) (driver.Rows, error) {
	if s.conn.notFound {
		return &obsoleteEmptyRows{}, nil
	}
	return &obsoleteTestRows{
		status:          s.conn.docStatus,
		revisionVersion: s.conn.docRevisionVersion,
	}, nil
}

type obsoleteTestConn struct {
	// SELECT results
	docStatus          string
	docRevisionVersion int
	notFound           bool

	// UPDATE documents result
	docUpdateRowsAffected int64
}

func (c *obsoleteTestConn) Prepare(query string) (driver.Stmt, error) {
	return &obsoleteTestStmt{conn: c, query: query}, nil
}
func (c *obsoleteTestConn) Close() error              { return nil }
func (c *obsoleteTestConn) Begin() (driver.Tx, error) { return c, nil }
func (c *obsoleteTestConn) Commit() error             { return nil }
func (c *obsoleteTestConn) Rollback() error           { return nil }

type obsoleteTestDriver struct{ conn *obsoleteTestConn }

func (d *obsoleteTestDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

// newObsoleteTestDB registers a unique driver name and returns a *sql.DB.
func newObsoleteTestDB(t *testing.T, conn *obsoleteTestConn) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("obsolete_test_%p", conn)
	sql.Register(name, &obsoleteTestDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open obsolete test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMarkObsolete_FromPublished(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: now}

	conn := &obsoleteTestConn{
		docStatus:             "published",
		docRevisionVersion:    3,
		docUpdateRowsAffected: 1,
	}
	db := newObsoleteTestDB(t, conn)

	svc := &ObsoleteService{emitter: emitter, clock: clock}
	req := MarkObsoleteRequest{
		TenantID:        "tenant-1",
		DocumentID:      "doc-pub-1",
		MarkedBy:        "user-1",
		RevisionVersion: 3,
		Reason:          "product line discontinued",
	}

	result, err := svc.MarkObsolete(context.Background(), db, req)
	if err != nil {
		t.Fatalf("MarkObsolete: unexpected error: %v", err)
	}
	if result.PriorStatus != "published" {
		t.Errorf("PriorStatus = %q; want %q", result.PriorStatus, "published")
	}

	if len(emitter.Events) != 1 {
		t.Fatalf("expected 1 governance event; got %d", len(emitter.Events))
	}
	ev := emitter.Events[0]
	if ev.EventType != "document_obsoleted" {
		t.Errorf("event type = %q; want %q", ev.EventType, "document_obsoleted")
	}
	if ev.ResourceID != "doc-pub-1" {
		t.Errorf("event resource_id = %q; want %q", ev.ResourceID, "doc-pub-1")
	}
	if ev.ActorUserID != "user-1" {
		t.Errorf("event actor_user_id = %q; want %q", ev.ActorUserID, "user-1")
	}
	if ev.TenantID != "tenant-1" {
		t.Errorf("event tenant_id = %q; want %q", ev.TenantID, "tenant-1")
	}
}

func TestMarkObsolete_FromSuperseded(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: now}

	conn := &obsoleteTestConn{
		docStatus:             "superseded",
		docRevisionVersion:    7,
		docUpdateRowsAffected: 1,
	}
	db := newObsoleteTestDB(t, conn)

	svc := &ObsoleteService{emitter: emitter, clock: clock}
	req := MarkObsoleteRequest{
		TenantID:        "tenant-2",
		DocumentID:      "doc-sup-1",
		MarkedBy:        "user-2",
		RevisionVersion: 7,
		Reason:          "replaced by v3",
	}

	result, err := svc.MarkObsolete(context.Background(), db, req)
	if err != nil {
		t.Fatalf("MarkObsolete: unexpected error: %v", err)
	}
	if result.PriorStatus != "superseded" {
		t.Errorf("PriorStatus = %q; want %q", result.PriorStatus, "superseded")
	}

	if len(emitter.Events) != 1 {
		t.Fatalf("expected 1 governance event; got %d", len(emitter.Events))
	}
	ev := emitter.Events[0]
	if ev.EventType != "document_obsoleted" {
		t.Errorf("event type = %q; want %q", ev.EventType, "document_obsoleted")
	}
}

func TestMarkObsolete_InvalidSource(t *testing.T) {
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)}

	conn := &obsoleteTestConn{
		docStatus:             "draft",
		docRevisionVersion:    1,
		docUpdateRowsAffected: 0, // not reached
	}
	db := newObsoleteTestDB(t, conn)

	svc := &ObsoleteService{emitter: emitter, clock: clock}
	req := MarkObsoleteRequest{
		TenantID:        "tenant-3",
		DocumentID:      "doc-draft-1",
		MarkedBy:        "user-3",
		RevisionVersion: 1,
		Reason:          "wrong state test",
	}

	_, err := svc.MarkObsolete(context.Background(), db, req)
	if err == nil {
		t.Fatal("expected ErrInvalidObsoleteSource; got nil")
	}
	if !errors.Is(err, ErrInvalidObsoleteSource) {
		t.Errorf("expected errors.Is(err, ErrInvalidObsoleteSource); got %v", err)
	}
	if len(emitter.Events) != 0 {
		t.Errorf("no governance event should be emitted on invalid source; got %d", len(emitter.Events))
	}
}

func TestMarkObsolete_StaleRevision(t *testing.T) {
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)}

	conn := &obsoleteTestConn{
		docStatus:             "published",
		docRevisionVersion:    5,
		docUpdateRowsAffected: 0, // OCC conflict
	}
	db := newObsoleteTestDB(t, conn)

	svc := &ObsoleteService{emitter: emitter, clock: clock}
	req := MarkObsoleteRequest{
		TenantID:        "tenant-4",
		DocumentID:      "doc-stale-1",
		MarkedBy:        "user-4",
		RevisionVersion: 5,
		Reason:          "stale test",
	}

	_, err := svc.MarkObsolete(context.Background(), db, req)
	if err == nil {
		t.Fatal("expected ErrStaleRevision; got nil")
	}
	if !errors.Is(err, repository.ErrStaleRevision) {
		t.Errorf("expected errors.Is(err, ErrStaleRevision); got %v", err)
	}
	if len(emitter.Events) != 0 {
		t.Errorf("no governance event should be emitted on stale revision; got %d", len(emitter.Events))
	}
}
