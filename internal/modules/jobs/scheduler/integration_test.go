//go:build integration
// +build integration

package scheduler_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/jobs/scheduler"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("METALDOCS_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("no DATABASE_URL set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open(pgx): %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM metaldocs.job_leases WHERE job_name LIKE 'test-%'")
		_ = db.Close()
	})
	return db
}

func testCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func beginTx(t *testing.T, ctx context.Context, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	return tx
}

func setSearchPath(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('search_path', 'metaldocs,public', true)"); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
}

func acquireLease(ctx context.Context, q queryer, job, leader string) (bool, int64, error) {
	var acquired bool
	var epoch int64
	err := q.QueryRowContext(ctx, `
SELECT (r).acquired, (r).epoch
FROM (SELECT acquire_lease($1, $2, $3) AS r) x`,
		job,
		leader,
		"30 seconds",
	).Scan(&acquired, &epoch)
	return acquired, epoch, err
}

func heartbeatLease(ctx context.Context, q queryer, job, leader string, epoch int64) (bool, error) {
	var ok bool
	err := q.QueryRowContext(ctx, "SELECT heartbeat_lease($1, $2, $3)", job, leader, epoch).Scan(&ok)
	return ok, err
}

func releaseLease(ctx context.Context, e execer, job, leader string, epoch int64) error {
	_, err := e.ExecContext(ctx, "SELECT release_lease($1, $2, $3)", job, leader, epoch)
	return err
}

func assertLeaseEpoch(ctx context.Context, e execer, job string, epoch int64) error {
	_, err := e.ExecContext(ctx, "SELECT assert_lease_epoch($1, $2)", job, epoch)
	return err
}

func testJobName(probe string) string {
	return fmt.Sprintf("test-%s-%d", probe, time.Now().UnixNano())
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func TestProbe1_AcquireLease_NewJob(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("acquire-new")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquired=true")
	}
	if epoch != 1 {
		t.Fatalf("expected epoch=1, got %d", epoch)
	}

	var leader string
	var leaseEpoch int64
	if err := tx.QueryRowContext(ctx, "SELECT leader_id, lease_epoch FROM metaldocs.job_leases WHERE job_name = $1", job).Scan(&leader, &leaseEpoch); err != nil {
		t.Fatalf("select job_leases: %v", err)
	}
	if leader != "leader-a" {
		t.Fatalf("expected leader_id=leader-a, got %s", leader)
	}
	if leaseEpoch != 1 {
		t.Fatalf("expected lease_epoch=1 in table, got %d", leaseEpoch)
	}
}

func TestProbe2_AcquireLease_Takeover(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("acquire-takeover")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("initial acquire_lease: %v", err)
	}
	if !acquired || epoch != 1 {
		t.Fatalf("expected first acquire true epoch=1, got acquired=%v epoch=%d", acquired, epoch)
	}

	if _, err := tx.ExecContext(ctx, "UPDATE metaldocs.job_leases SET expires_at = now() - interval '1 minute' WHERE job_name = $1", job); err != nil {
		t.Fatalf("expire lease fixture: %v", err)
	}

	acquired, epoch, err = acquireLease(ctx, tx, job, "leader-b")
	if err != nil {
		t.Fatalf("takeover acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected second leader to acquire expired lease")
	}
	if epoch != 2 {
		t.Fatalf("expected epoch=2 after takeover, got %d", epoch)
	}
}

func TestProbe3_AcquireLease_Reentrant(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("acquire-reentrant")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("first acquire_lease: %v", err)
	}
	if !acquired || epoch != 1 {
		t.Fatalf("expected first acquire true epoch=1, got acquired=%v epoch=%d", acquired, epoch)
	}

	acquired, epoch2, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("reentrant acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected reentrant acquire to succeed")
	}
	if epoch2 != epoch {
		t.Fatalf("expected same epoch=%d on reentrant acquire, got %d", epoch, epoch2)
	}
}

func TestProbe4_HeartbeatLease_MatchingEpoch(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("heartbeat-match")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected lease acquisition")
	}

	var before time.Time
	if err := tx.QueryRowContext(ctx, "SELECT expires_at FROM metaldocs.job_leases WHERE job_name = $1", job).Scan(&before); err != nil {
		t.Fatalf("read expires_at before heartbeat: %v", err)
	}

	ok, err := heartbeatLease(ctx, tx, job, "leader-a", epoch)
	if err != nil {
		t.Fatalf("heartbeat_lease: %v", err)
	}
	if !ok {
		t.Fatal("expected heartbeat_lease=true for matching leader+epoch")
	}

	var after time.Time
	if err := tx.QueryRowContext(ctx, "SELECT expires_at FROM metaldocs.job_leases WHERE job_name = $1", job).Scan(&after); err != nil {
		t.Fatalf("read expires_at after heartbeat: %v", err)
	}
	if !after.After(before) {
		t.Fatalf("expected expires_at to extend, before=%s after=%s", before, after)
	}
}

func TestProbe5_HeartbeatLease_StalEpoch(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("heartbeat-stale")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected lease acquisition")
	}

	ok, err := heartbeatLease(ctx, tx, job, "leader-a", epoch+1)
	if err != nil {
		t.Fatalf("heartbeat_lease stale epoch: %v", err)
	}
	if ok {
		t.Fatal("expected heartbeat_lease=false for stale epoch")
	}
}

// TestProbe6_ReleaseLease_ExpiresRow verifies that release_lease marks the lease
// as expired (expires_at in the past) rather than deleting it, preserving epoch
// monotonicity per migration 0149.
func TestProbe6_ReleaseLease_ExpiresRow(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("release-expire")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected lease acquisition")
	}

	if err := releaseLease(ctx, tx, job, "leader-a", epoch); err != nil {
		t.Fatalf("release_lease: %v", err)
	}

	// Row must still exist (not deleted) but be expired.
	var count int
	var expired bool
	if err := tx.QueryRowContext(ctx,
		"SELECT count(*), bool_and(expires_at < now()) FROM metaldocs.job_leases WHERE job_name = $1",
		job,
	).Scan(&count, &expired); err != nil {
		t.Fatalf("query after release: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected row to remain after release, got count=%d", count)
	}
	if !expired {
		t.Fatal("expected expires_at < now() after release_lease")
	}
}

func TestProbe7_ReleaseLease_WrongEpoch(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("release-wrong-epoch")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected lease acquisition")
	}

	if err := releaseLease(ctx, tx, job, "leader-a", epoch+99); err != nil {
		t.Fatalf("release_lease wrong epoch: %v", err)
	}

	var count int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM metaldocs.job_leases WHERE job_name = $1", job).Scan(&count); err != nil {
		t.Fatalf("count job_leases: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected row to remain, got count=%d", count)
	}
}

func TestProbe8_AssertLeaseEpoch_Valid(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("assert-valid")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected lease acquisition")
	}

	if err := assertLeaseEpoch(ctx, tx, job, epoch); err != nil {
		t.Fatalf("assert_lease_epoch valid should pass, got: %v", err)
	}
}

func TestProbe9_AssertLeaseEpoch_Stale(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("assert-stale")
	acquired, epoch, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("acquire_lease: %v", err)
	}
	if !acquired {
		t.Fatal("expected lease acquisition")
	}

	err = assertLeaseEpoch(ctx, tx, job, epoch+1)
	if err == nil {
		t.Fatal("expected stale assert_lease_epoch to fail")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("expected pgconn.PgError, got %T (%v)", err, err)
	}
	if pgErr.Code != "P0001" {
		t.Fatalf("expected SQLSTATE P0001, got %q", pgErr.Code)
	}
	if !strings.Contains(pgErr.Message, "ErrLeaseEpochStale") {
		t.Fatalf("expected ErrLeaseEpochStale in message, got %q", pgErr.Message)
	}
}

func TestProbe10_LeaseReaper_ReclaimeStale(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	job := testJobName("lease-reaper")

	if _, err := db.ExecContext(ctx, `
INSERT INTO metaldocs.job_leases (job_name, leader_id, lease_epoch, acquired_at, heartbeat_at, expires_at)
VALUES ($1, $2, $3, now() - interval '15 minutes', now() - interval '15 minutes', now() - interval '11 minutes')
`, job, "leader-reaper", int64(7)); err != nil {
		t.Fatalf("insert stale lease fixture: %v", err)
	}

	fn := scheduler.RunLeaseReaper(db)
	if err := fn(ctx, 1); err != nil {
		t.Fatalf("RunLeaseReaper: %v", err)
	}

	var leaseCount int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM metaldocs.job_leases WHERE job_name = $1", job).Scan(&leaseCount); err != nil {
		t.Fatalf("count leases after reaper: %v", err)
	}
	if leaseCount != 0 {
		t.Fatalf("expected stale lease deleted, got count=%d", leaseCount)
	}

	var eventCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*)
FROM governance_events
WHERE event_type = 'lease.reaped'
  AND resource_type = 'job_lease'
  AND resource_id = $1
`, job).Scan(&eventCount); err != nil {
		t.Fatalf("count governance events: %v", err)
	}
	if eventCount == 0 {
		t.Fatal("expected at least one governance_events row for lease.reaped")
	}
}

// TestProbe11_EpochMonotonic_AcrossReleaseReacquire verifies that after
// release_lease, a subsequent acquire_lease increments the epoch rather than
// resetting it to 1 (regression guard for migration 0149).
func TestProbe11_EpochMonotonic_AcrossReleaseReacquire(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := testDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()
	setSearchPath(t, ctx, tx)

	job := testJobName("epoch-monotonic")

	// First acquisition: epoch must be 1.
	ok, epoch1, err := acquireLease(ctx, tx, job, "leader-a")
	if err != nil {
		t.Fatalf("first acquire_lease: %v", err)
	}
	if !ok || epoch1 != 1 {
		t.Fatalf("expected acquired=true epoch=1, got acquired=%v epoch=%d", ok, epoch1)
	}

	// Release: marks lease expired (0149 behaviour — row stays).
	if err := releaseLease(ctx, tx, job, "leader-a", epoch1); err != nil {
		t.Fatalf("release_lease: %v", err)
	}

	// Second acquisition (same or different leader on expired lease): epoch must be 2.
	ok, epoch2, err := acquireLease(ctx, tx, job, "leader-b")
	if err != nil {
		t.Fatalf("second acquire_lease: %v", err)
	}
	if !ok {
		t.Fatal("expected second acquire_lease to succeed on expired lease")
	}
	if epoch2 != epoch1+1 {
		t.Fatalf("expected epoch to increment from %d to %d, got %d", epoch1, epoch1+1, epoch2)
	}
}
