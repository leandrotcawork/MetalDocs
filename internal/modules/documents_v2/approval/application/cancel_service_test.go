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
	"metaldocs/internal/modules/iam/authz"
)

// ---------------------------------------------------------------------------
// Fake repo for cancel tests — only implements LoadInstance + UpdateInstanceStatus.
// ---------------------------------------------------------------------------

type cancelFakeRepo struct {
	repository.ApprovalRepository
	instance          *domain.Instance
	loadErr           error
	updateInstErr     error
}

func (r *cancelFakeRepo) LoadInstance(_ context.Context, _ *sql.Tx, _, _ string) (*domain.Instance, error) {
	return r.instance, r.loadErr
}

func (r *cancelFakeRepo) UpdateInstanceStatus(_ context.Context, _ *sql.Tx, _, _ string, _, _ domain.InstanceStatus, _ *time.Time) error {
	return r.updateInstErr
}

// ---------------------------------------------------------------------------
// Fake SQL driver for cancel tests.
//
// CancelInstance issues (in order):
//   1. SELECT area_code FROM documents
//   2. SELECT EXISTS … role_capabilities (authz)
//   3. SELECT current_setting('metaldocs.asserted_caps', true)
//   4. SELECT current_setting('metaldocs.actor_id', false)       (authz actor read)
//   5. SELECT set_config('metaldocs.asserted_caps', ...)
//   6. SELECT set_config('metaldocs.cancel_in_progress', ...)
//   7. UPDATE approval_stage_instances SET status='cancelled'
//   8. UPDATE documents SET status='draft' ...
//   9. INSERT INTO governance_events
// ---------------------------------------------------------------------------

type cancelTestRows struct {
	values []driver.Value
	cols   []string
	done   bool
}

func (r *cancelTestRows) Columns() []string              { return r.cols }
func (r *cancelTestRows) Close() error                   { return nil }
func (r *cancelTestRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	for i, v := range r.values {
		dest[i] = v
	}
	return nil
}

type cancelEmptyRows struct{ cols []string }

func (r cancelEmptyRows) Columns() []string              { return r.cols }
func (r cancelEmptyRows) Close() error                   { return nil }
func (r cancelEmptyRows) Next([]driver.Value) error      { return io.EOF }

type cancelTestResult struct{ rowsAffected int64 }

func (r cancelTestResult) LastInsertId() (int64, error) { return 0, nil }
func (r cancelTestResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type cancelTestStmt struct {
	conn  *cancelTestConn
	query string
}

func (s *cancelTestStmt) Close() error  { return nil }
func (s *cancelTestStmt) NumInput() int { return -1 }

func (s *cancelTestStmt) Exec(_ []driver.Value) (driver.Result, error) {
	lower := strings.ToLower(s.query)
	if strings.Contains(lower, "update documents") {
		return cancelTestResult{rowsAffected: s.conn.docUpdateRows}, nil
	}
	// stage cancel, governance_events INSERT — always succeed
	return cancelTestResult{rowsAffected: 1}, nil
}

func (s *cancelTestStmt) Query(_ []driver.Value) (driver.Rows, error) {
	lower := strings.ToLower(s.query)

	// area_code fetch
	if strings.Contains(lower, "select area_code") {
		if s.conn.areaCode == "" {
			return cancelEmptyRows{cols: []string{"area_code"}}, nil
		}
		return &cancelTestRows{cols: []string{"area_code"}, values: []driver.Value{s.conn.areaCode}}, nil
	}
	// authz: SELECT EXISTS
	if strings.Contains(lower, "select exists") && strings.Contains(lower, "role_capabilities") {
		return &cancelTestRows{cols: []string{"exists"}, values: []driver.Value{s.conn.authzGranted}}, nil
	}
	// asserted_caps current value
	if strings.Contains(lower, "current_setting('metaldocs.asserted_caps'") ||
		strings.Contains(lower, `current_setting('metaldocs.asserted_caps'`) {
		return &cancelTestRows{cols: []string{"v"}, values: []driver.Value{nil}}, nil
	}
	// actor_id
	if strings.Contains(lower, "current_setting('metaldocs.actor_id'") ||
		strings.Contains(lower, `current_setting('metaldocs.actor_id'`) {
		return &cancelTestRows{cols: []string{"v"}, values: []driver.Value{s.conn.actorID}}, nil
	}
	// set_config calls — return the value being set
	if strings.Contains(lower, "set_config") {
		return &cancelTestRows{cols: []string{"v"}, values: []driver.Value{"ok"}}, nil
	}
	return cancelEmptyRows{cols: []string{"v"}}, nil
}

type cancelTestConn struct {
	areaCode      string
	authzGranted  bool
	actorID       string
	docUpdateRows int64
}

func (c *cancelTestConn) Prepare(query string) (driver.Stmt, error) {
	return &cancelTestStmt{conn: c, query: query}, nil
}
func (c *cancelTestConn) Close() error              { return nil }
func (c *cancelTestConn) Begin() (driver.Tx, error) { return c, nil }
func (c *cancelTestConn) Commit() error             { return nil }
func (c *cancelTestConn) Rollback() error           { return nil }

type cancelTestDriver struct{ conn *cancelTestConn }

func (d *cancelTestDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

func newCancelTestDB(t *testing.T, conn *cancelTestConn) *sql.DB {
	t.Helper()
	if conn.areaCode == "" {
		conn.areaCode = "QA"
	}
	if conn.actorID == "" {
		conn.actorID = "user-1"
	}
	name := fmt.Sprintf("cancel_test_%p", conn)
	sql.Register(name, &cancelTestDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open cancel test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func buildCancelInstance() *domain.Instance {
	return &domain.Instance{
		ID:         "inst-1",
		DocumentID: "doc-1",
		TenantID:   "tenant-1",
		Status:     domain.InstanceInProgress,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCancelInstance_HappyPath(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: now}

	conn := &cancelTestConn{
		authzGranted:  true,
		docUpdateRows: 1,
	}
	db := newCancelTestDB(t, conn)

	repo := &cancelFakeRepo{instance: buildCancelInstance()}
	svc := &CancelService{repo: repo, emitter: emitter, clock: clock}

	result, err := svc.CancelInstance(context.Background(), db, CancelInput{
		TenantID:                "tenant-1",
		InstanceID:              "inst-1",
		ExpectedRevisionVersion: 2,
		ActorUserID:             "user-1",
		Reason:                  "stakeholder withdrew",
	})
	if err != nil {
		t.Fatalf("CancelInstance: unexpected error: %v", err)
	}
	if result.DocumentID != "doc-1" {
		t.Errorf("DocumentID = %q; want %q", result.DocumentID, "doc-1")
	}
	if len(emitter.Events) != 1 || emitter.Events[0].EventType != "approval.instance_cancelled" {
		t.Errorf("expected 1 approval.instance_cancelled event; got %v", emitter.Events)
	}
}

func TestCancelInstance_InstanceCompleted(t *testing.T) {
	inst := buildCancelInstance()
	inst.Status = domain.InstanceApproved

	conn := &cancelTestConn{authzGranted: true, docUpdateRows: 1}
	db := newCancelTestDB(t, conn)

	repo := &cancelFakeRepo{instance: inst}
	svc := &CancelService{repo: repo, emitter: &MemoryEmitter{}, clock: fixedClock{t: time.Now()}}

	_, err := svc.CancelInstance(context.Background(), db, CancelInput{
		TenantID: "t", InstanceID: "i", ActorUserID: "u", Reason: "reason",
	})
	if !errors.Is(err, repository.ErrInstanceCompleted) {
		t.Errorf("expected ErrInstanceCompleted; got %v", err)
	}
}

func TestCancelInstance_CapDenied(t *testing.T) {
	conn := &cancelTestConn{authzGranted: false, docUpdateRows: 0}
	db := newCancelTestDB(t, conn)

	repo := &cancelFakeRepo{instance: buildCancelInstance()}
	svc := &CancelService{repo: repo, emitter: &MemoryEmitter{}, clock: fixedClock{t: time.Now()}}

	_, err := svc.CancelInstance(context.Background(), db, CancelInput{
		TenantID: "t", InstanceID: "i", ActorUserID: "u", Reason: "reason",
	})
	var denied authz.ErrCapabilityDenied
	if !errors.As(err, &denied) {
		t.Errorf("expected ErrCapabilityDenied; got %v", err)
	}
}

func TestCancelInstance_EmptyReason(t *testing.T) {
	conn := &cancelTestConn{authzGranted: true}
	db := newCancelTestDB(t, conn)

	repo := &cancelFakeRepo{instance: buildCancelInstance()}
	svc := &CancelService{repo: repo, emitter: &MemoryEmitter{}, clock: fixedClock{t: time.Now()}}

	_, err := svc.CancelInstance(context.Background(), db, CancelInput{
		TenantID: "t", InstanceID: "i", ActorUserID: "u", Reason: "",
	})
	if !errors.Is(err, ErrReasonRequired) {
		t.Errorf("expected ErrReasonRequired; got %v", err)
	}
}
