package application

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// ---------------------------------------------------------------------------
// Fake repo — only ListScheduledDue is exercised by the scheduler.
// ---------------------------------------------------------------------------

type fakeSchedulerRepo struct {
	rows    []repository.ScheduledPublishRow
	fetchErr error
	repository.ApprovalRepository // no-op embed for unused methods
}

func (r *fakeSchedulerRepo) ListScheduledDue(
	_ context.Context, _ *sql.Tx, _ time.Time, _ int,
) ([]repository.ScheduledPublishRow, error) {
	return r.rows, r.fetchErr
}

// ---------------------------------------------------------------------------
// Custom sql driver for scheduler tests.
//
// The driver must handle (per row):
//   1. BEGIN             — from db.BeginTx (fetch tx + per-row tx)
//   2. UPDATE documents  — returns configurable rowsAffected
//   3. INSERT governance_events — always succeeds
//   4. COMMIT / ROLLBACK
//
// rowsAffectedPerUpdate is a slice; each UPDATE call pops the next value.
// ---------------------------------------------------------------------------

type schedulerTestResult struct{ rowsAffected int64 }

func (r schedulerTestResult) LastInsertId() (int64, error) { return 0, nil }
func (r schedulerTestResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type schedulerEmptyRows struct{}

func (schedulerEmptyRows) Columns() []string         { return nil }
func (schedulerEmptyRows) Close() error              { return nil }
func (schedulerEmptyRows) Next([]driver.Value) error { return io.EOF }

type schedulerTestStmt struct {
	query        string
	conn         *schedulerTestConn
}

func (s *schedulerTestStmt) Close() error  { return nil }
func (s *schedulerTestStmt) NumInput() int { return -1 }

func (s *schedulerTestStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if strings.Contains(strings.ToLower(s.query), "update") {
		ra := s.conn.nextRowsAffected()
		return schedulerTestResult{rowsAffected: ra}, nil
	}
	// INSERT governance_events, etc.
	return schedulerTestResult{rowsAffected: 1}, nil
}

func (s *schedulerTestStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return schedulerEmptyRows{}, nil
}

type schedulerTestConn struct {
	// updateResults is consumed left-to-right; one value per UPDATE call.
	updateResults []int64
	updateIdx     int
}

func (c *schedulerTestConn) nextRowsAffected() int64 {
	if c.updateIdx >= len(c.updateResults) {
		return 1 // default: success
	}
	ra := c.updateResults[c.updateIdx]
	c.updateIdx++
	return ra
}

func (c *schedulerTestConn) Prepare(query string) (driver.Stmt, error) {
	return &schedulerTestStmt{query: query, conn: c}, nil
}

func (c *schedulerTestConn) Close() error              { return nil }
func (c *schedulerTestConn) Begin() (driver.Tx, error) { return c, nil }

// BeginTx implements driver.ConnBeginTx so that non-default isolation levels
// (e.g. sql.LevelReadCommitted used by the fetch transaction) are accepted.
func (c *schedulerTestConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return c, nil
}

func (c *schedulerTestConn) Commit() error   { return nil }
func (c *schedulerTestConn) Rollback() error { return nil }

type schedulerTestDriver struct{ conn *schedulerTestConn }

func (d *schedulerTestDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

// newSchedulerTestDB registers a unique driver per test.
// updateResults controls what each successive UPDATE call returns for RowsAffected.
func newSchedulerTestDB(t *testing.T, updateResults []int64) *sql.DB {
	t.Helper()
	conn := &schedulerTestConn{updateResults: updateResults}
	name := fmt.Sprintf("scheduler_test_%p", conn)
	sql.Register(name, &schedulerTestDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open scheduler test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRunDuePublishes_HappyPath(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)

	rows := []repository.ScheduledPublishRow{
		{
			DocumentID:      "doc-1",
			TenantID:        "tenant-1",
			EffectiveFrom:   now.Add(-time.Hour),
			RevisionVersion: 3,
		},
		{
			DocumentID:      "doc-2",
			TenantID:        "tenant-1",
			EffectiveFrom:   now.Add(-2 * time.Hour),
			RevisionVersion: 7,
		},
	}

	repo := &fakeSchedulerRepo{rows: rows}
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: now}

	svc := &SchedulerService{repo: repo, emitter: emitter, clock: clock}
	// Both UPDATEs return rowsAffected=1 (successful publish).
	db := newSchedulerTestDB(t, []int64{1, 1})

	result, err := svc.RunDuePublishes(context.Background(), db)
	if err != nil {
		t.Fatalf("RunDuePublishes: unexpected top-level error: %v", err)
	}
	if result.Processed != 2 {
		t.Errorf("result.Processed = %d; want 2", result.Processed)
	}
	if len(result.Errors) != 0 {
		t.Errorf("result.Errors = %v; want empty", result.Errors)
	}

	// Two "document_published" governance events, one per row.
	if len(emitter.Events) != 2 {
		t.Fatalf("expected 2 governance events; got %d", len(emitter.Events))
	}
	for i, ev := range emitter.Events {
		if ev.EventType != "document_published" {
			t.Errorf("event[%d].EventType = %q; want %q", i, ev.EventType, "document_published")
		}
		wantDoc := rows[i].DocumentID
		if ev.ResourceID != wantDoc {
			t.Errorf("event[%d].ResourceID = %q; want %q", i, ev.ResourceID, wantDoc)
		}
	}
}

func TestRunDuePublishes_OtherRunnerWon(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)

	rows := []repository.ScheduledPublishRow{
		{
			DocumentID:      "doc-contested",
			TenantID:        "tenant-1",
			EffectiveFrom:   now.Add(-time.Minute),
			RevisionVersion: 2,
		},
	}

	repo := &fakeSchedulerRepo{rows: rows}
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: now}

	svc := &SchedulerService{repo: repo, emitter: emitter, clock: clock}
	// UPDATE returns rowsAffected=0 — another runner already published.
	db := newSchedulerTestDB(t, []int64{0})

	result, err := svc.RunDuePublishes(context.Background(), db)
	if err != nil {
		t.Fatalf("RunDuePublishes: unexpected top-level error: %v", err)
	}
	if result.Processed != 0 {
		t.Errorf("result.Processed = %d; want 0", result.Processed)
	}
	if len(result.Errors) != 0 {
		t.Errorf("result.Errors = %v; want empty", result.Errors)
	}
	if len(emitter.Events) != 0 {
		t.Errorf("expected 0 governance events; got %d", len(emitter.Events))
	}
}
