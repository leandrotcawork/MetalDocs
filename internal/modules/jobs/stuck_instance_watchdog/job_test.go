package stuck_instance_watchdog

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"metaldocs/internal/modules/documents_v2/approval/application"
)

type mockCancelService struct {
	mu      sync.Mutex
	calls   []application.CancelInput
	results []error
}

func (m *mockCancelService) CancelInstance(_ context.Context, _ *sql.DB, in application.CancelInput) (application.CancelResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, in)
	if len(m.results) > 0 {
		err := m.results[0]
		m.results = m.results[1:]
		if err != nil {
			return application.CancelResult{}, err
		}
	}
	return application.CancelResult{DocumentID: "doc"}, nil
}

func (m *mockCancelService) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

type recordingEmitter struct {
	mu     sync.Mutex
	events []application.GovernanceEvent
}

func (r *recordingEmitter) Emit(_ context.Context, _ *sql.Tx, e application.GovernanceEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}

func (r *recordingEmitter) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func (r *recordingEmitter) first() application.GovernanceEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.events) == 0 {
		return application.GovernanceEvent{}
	}
	return r.events[0]
}

type watchdogDBState struct {
	mu         sync.Mutex
	stuckRows  []StuckInstance
	setConfigN int
}

type watchdogDriver struct {
	state *watchdogDBState
}

func (d *watchdogDriver) Open(_ string) (driver.Conn, error) {
	return &watchdogConn{state: d.state}, nil
}

type watchdogConn struct {
	state *watchdogDBState
}

func (c *watchdogConn) Prepare(query string) (driver.Stmt, error) {
	return &watchdogStmt{state: c.state, query: query}, nil
}

func (c *watchdogConn) Close() error { return nil }

func (c *watchdogConn) Begin() (driver.Tx, error) { return watchdogTx{}, nil }

func (c *watchdogConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return watchdogTx{}, nil
}

type watchdogTx struct{}

func (watchdogTx) Commit() error   { return nil }
func (watchdogTx) Rollback() error { return nil }

type watchdogStmt struct {
	state *watchdogDBState
	query string
}

func (s *watchdogStmt) Close() error  { return nil }
func (s *watchdogStmt) NumInput() int { return -1 }

func (s *watchdogStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if strings.Contains(strings.ToLower(s.query), "set_config('metaldocs.bypass_authz'") {
		s.state.mu.Lock()
		s.state.setConfigN++
		s.state.mu.Unlock()
	}
	return watchdogResult(1), nil
}

func (s *watchdogStmt) Query(_ []driver.Value) (driver.Rows, error) {
	q := strings.ToLower(s.query)
	if strings.Contains(q, "from approval_instances") {
		s.state.mu.Lock()
		rows := append([]StuckInstance(nil), s.state.stuckRows...)
		s.state.mu.Unlock()
		out := make([][]driver.Value, 0, len(rows))
		for _, row := range rows {
			out = append(out, []driver.Value{
				row.ID,
				row.TenantID,
				row.DocumentID,
				row.SubmittedBy,
				row.DriftPolicy,
			})
		}
		return &watchdogRows{
			cols: []string{"id", "tenant_id", "document_v2_id", "submitted_by", "drift_policy"},
			rows: out,
		}, nil
	}
	return &watchdogRows{cols: []string{"ok"}, rows: [][]driver.Value{}}, nil
}

type watchdogRows struct {
	cols []string
	rows [][]driver.Value
	idx  int
}

func (r *watchdogRows) Columns() []string { return r.cols }
func (r *watchdogRows) Close() error      { return nil }

func (r *watchdogRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	for i := range r.rows[r.idx] {
		dest[i] = r.rows[r.idx][i]
	}
	r.idx++
	return nil
}

type watchdogResult int64

func (r watchdogResult) LastInsertId() (int64, error) { return 0, nil }
func (r watchdogResult) RowsAffected() (int64, error) { return int64(r), nil }

func newWatchdogDB(t *testing.T, state *watchdogDBState) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("watchdog_%p", state)
	sql.Register(name, &watchdogDriver{state: state})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open watchdog db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestWatchdog_NoStuck(t *testing.T) {
	t.Parallel()

	state := &watchdogDBState{}
	db := newWatchdogDB(t, state)
	cancelSvc := &mockCancelService{}
	emitter := &recordingEmitter{}

	fn := New(db, cancelSvc, emitter)
	if err := fn(context.Background(), 1); err != nil {
		t.Fatalf("job returned error: %v", err)
	}

	if got := cancelSvc.callCount(); got != 0 {
		t.Fatalf("cancel calls = %d, want 0", got)
	}
	if got := emitter.count(); got != 0 {
		t.Fatalf("alerts emitted = %d, want 0", got)
	}
}

func TestWatchdog_AutoCancel(t *testing.T) {
	t.Parallel()

	state := &watchdogDBState{
		stuckRows: []StuckInstance{
			{ID: "inst-1", TenantID: "tenant-1", DocumentID: "doc-1", SubmittedBy: "u1", DriftPolicy: "auto_cancel"},
			{ID: "inst-2", TenantID: "tenant-1", DocumentID: "doc-2", SubmittedBy: "u2", DriftPolicy: "auto_cancel"},
			{ID: "inst-3", TenantID: "tenant-2", DocumentID: "doc-3", SubmittedBy: "u3", DriftPolicy: "auto_cancel"},
		},
	}
	db := newWatchdogDB(t, state)
	cancelSvc := &mockCancelService{}
	emitter := &recordingEmitter{}

	fn := New(db, cancelSvc, emitter)
	if err := fn(context.Background(), 2); err != nil {
		t.Fatalf("job returned error: %v", err)
	}

	if got := cancelSvc.callCount(); got != 3 {
		t.Fatalf("cancel calls = %d, want 3", got)
	}
	if got := emitter.count(); got != 0 {
		t.Fatalf("alerts emitted = %d, want 0", got)
	}

	for i, call := range cancelSvc.calls {
		if call.ExpectedRevisionVersion != 0 {
			t.Fatalf("call[%d] ExpectedRevisionVersion = %d, want 0", i, call.ExpectedRevisionVersion)
		}
		if call.ActorUserID != SystemActor {
			t.Fatalf("call[%d] ActorUserID = %q, want %q", i, call.ActorUserID, SystemActor)
		}
		if call.Reason != "stuck_watchdog_auto_cancel" {
			t.Fatalf("call[%d] Reason = %q", i, call.Reason)
		}
	}
}

func TestWatchdog_AlertOnly(t *testing.T) {
	t.Parallel()

	state := &watchdogDBState{
		stuckRows: []StuckInstance{
			{ID: "inst-1", TenantID: "tenant-1", DocumentID: "doc-1", SubmittedBy: "u1", DriftPolicy: "reduce_quorum"},
		},
	}
	db := newWatchdogDB(t, state)
	cancelSvc := &mockCancelService{}
	emitter := &recordingEmitter{}

	fn := New(db, cancelSvc, emitter)
	if err := fn(context.Background(), 3); err != nil {
		t.Fatalf("job returned error: %v", err)
	}

	if got := cancelSvc.callCount(); got != 0 {
		t.Fatalf("cancel calls = %d, want 0", got)
	}
	if got := emitter.count(); got != 1 {
		t.Fatalf("alerts emitted = %d, want 1", got)
	}
	ev := emitter.first()
	if ev.EventType != "approval.instance.stuck_alert" {
		t.Fatalf("event type = %q, want approval.instance.stuck_alert", ev.EventType)
	}
	if ev.ResourceType != "approval_instance" {
		t.Fatalf("resource type = %q, want approval_instance", ev.ResourceType)
	}
	if ev.ActorUserID != SystemActor {
		t.Fatalf("actor_user_id = %q, want %q", ev.ActorUserID, SystemActor)
	}
}

func TestWatchdog_CancelError(t *testing.T) {
	t.Parallel()

	state := &watchdogDBState{
		stuckRows: []StuckInstance{
			{ID: "inst-1", TenantID: "tenant-1", DocumentID: "doc-1", SubmittedBy: "u1", DriftPolicy: "auto_cancel"},
			{ID: "inst-2", TenantID: "tenant-1", DocumentID: "doc-2", SubmittedBy: "u2", DriftPolicy: "auto_cancel"},
		},
	}
	db := newWatchdogDB(t, state)
	cancelSvc := &mockCancelService{
		results: []error{errors.New("boom"), nil},
	}
	emitter := &recordingEmitter{}

	fn := New(db, cancelSvc, emitter)
	if err := fn(context.Background(), 4); err != nil {
		t.Fatalf("job returned error: %v", err)
	}

	if got := cancelSvc.callCount(); got != 2 {
		t.Fatalf("cancel calls = %d, want 2", got)
	}
	if got := emitter.count(); got != 0 {
		t.Fatalf("alerts emitted = %d, want 0", got)
	}
}
