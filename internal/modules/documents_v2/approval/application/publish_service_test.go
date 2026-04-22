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

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// ---------------------------------------------------------------------------
// Fake repo for publish tests — only LoadInstance is called.
// ---------------------------------------------------------------------------

type fakePublishRepo struct {
	instance    *domain.Instance
	loadErr     error
	repository.ApprovalRepository // no-op embed for unused methods
}

func (r *fakePublishRepo) LoadInstance(_ context.Context, _ *sql.Tx, _, _ string) (*domain.Instance, error) {
	return r.instance, r.loadErr
}

// ---------------------------------------------------------------------------
// Fake driver for publish tests.
// The driver handles:
//   1. BEGIN             — from db.BeginTx
//   2. UPDATE documents  — OCC transition (exec, returns rowsAffected)
//   3. COMMIT/ROLLBACK
// ---------------------------------------------------------------------------

type publishTestResult struct {
	rowsAffected int64
}

func (r publishTestResult) LastInsertId() (int64, error) { return 0, nil }
func (r publishTestResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type publishEmptyRows struct{}

func (publishEmptyRows) Columns() []string         { return nil }
func (publishEmptyRows) Close() error              { return nil }
func (publishEmptyRows) Next([]driver.Value) error { return io.EOF }

type publishTestStmt struct {
	query        string
	rowsAffected int64 // injected by the conn
}

func (s *publishTestStmt) Close() error  { return nil }
func (s *publishTestStmt) NumInput() int { return -1 }

func (s *publishTestStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return publishTestResult{rowsAffected: s.rowsAffected}, nil
}

func (s *publishTestStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return publishEmptyRows{}, nil
}

type publishTestConn struct {
	// rowsAffected controls what the UPDATE returns.
	rowsAffected int64
}

func (c *publishTestConn) Prepare(query string) (driver.Stmt, error) {
	ra := c.rowsAffected
	if !strings.Contains(strings.ToLower(query), "update") {
		// Non-UPDATE statements (governance_events INSERT, etc.) always succeed with 1.
		ra = 1
	}
	return &publishTestStmt{query: query, rowsAffected: ra}, nil
}

func (c *publishTestConn) Close() error              { return nil }
func (c *publishTestConn) Begin() (driver.Tx, error) { return c, nil }
func (c *publishTestConn) Commit() error             { return nil }
func (c *publishTestConn) Rollback() error           { return nil }

type publishTestDriver struct{ conn *publishTestConn }

func (d *publishTestDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

// newPublishTestDB registers a unique driver per test and returns a *sql.DB.
func newPublishTestDB(t *testing.T, rowsAffected int64) *sql.DB {
	t.Helper()
	conn := &publishTestConn{rowsAffected: rowsAffected}
	name := fmt.Sprintf("publish_test_%p", conn)
	sql.Register(name, &publishTestDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open publish test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPublishApproved_HappyPath(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	inst := &domain.Instance{
		ID:              "inst-uuid-1",
		TenantID:        "tenant-uuid-1",
		DocumentID:      "doc-uuid-1",
		Status:          domain.InstanceApproved,
		RevisionVersion: 3,
	}

	repo := &fakePublishRepo{instance: inst}
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: now}

	svc := &PublishService{repo: repo, emitter: emitter, clock: clock}
	// rowsAffected=1 → UPDATE matched one document row.
	db := newPublishTestDB(t, 1)

	req := PublishRequest{
		TenantID:    "tenant-uuid-1",
		InstanceID:  "inst-uuid-1",
		PublishedBy: "user-1",
	}

	result, err := svc.PublishApproved(context.Background(), db, req)
	if err != nil {
		t.Fatalf("PublishApproved: unexpected error: %v", err)
	}
	if result.DocumentID != "doc-uuid-1" {
		t.Errorf("result.DocumentID = %q; want %q", result.DocumentID, "doc-uuid-1")
	}
	if result.NewStatus != "published" {
		t.Errorf("result.NewStatus = %q; want %q", result.NewStatus, "published")
	}
	if len(emitter.Events) != 1 {
		t.Fatalf("expected 1 governance event; got %d", len(emitter.Events))
	}
	ev := emitter.Events[0]
	if ev.EventType != "document_published" {
		t.Errorf("event type = %q; want %q", ev.EventType, "document_published")
	}
	if ev.ResourceID != "doc-uuid-1" {
		t.Errorf("event resource_id = %q; want %q", ev.ResourceID, "doc-uuid-1")
	}
	if ev.ActorUserID != "user-1" {
		t.Errorf("event actor_user_id = %q; want %q", ev.ActorUserID, "user-1")
	}
}

func TestPublishApproved_NotApprovedInstance(t *testing.T) {
	tests := []struct {
		name   string
		status domain.InstanceStatus
	}{
		{"in_progress", domain.InstanceInProgress},
		{"rejected", domain.InstanceRejected},
		{"cancelled", domain.InstanceCancelled},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inst := &domain.Instance{
				ID:         "inst-uuid-2",
				TenantID:   "tenant-uuid-1",
				DocumentID: "doc-uuid-2",
				Status:     tc.status,
			}

			repo := &fakePublishRepo{instance: inst}
			emitter := &MemoryEmitter{}
			clock := fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)}

			svc := &PublishService{repo: repo, emitter: emitter, clock: clock}
			// rowsAffected irrelevant — should never reach UPDATE.
			db := newPublishTestDB(t, 0)

			req := PublishRequest{
				TenantID:    "tenant-uuid-1",
				InstanceID:  "inst-uuid-2",
				PublishedBy: "user-1",
			}

			_, err := svc.PublishApproved(context.Background(), db, req)
			if err == nil {
				t.Fatal("expected ErrInstanceNotApproved; got nil")
			}
			if !errors.Is(err, ErrInstanceNotApproved) {
				t.Errorf("expected errors.Is(err, ErrInstanceNotApproved); got %v", err)
			}
			if len(emitter.Events) != 0 {
				t.Errorf("no governance event should be emitted; got %d", len(emitter.Events))
			}
		})
	}
}
