package scheduler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type leaseAcquireResult struct {
	acquired bool
	epoch    int64
}

type pressureSample struct {
	active float64
	max    float64
}

type fakeSchedulerDB struct {
	mu sync.Mutex

	acquireResults   []leaseAcquireResult
	heartbeatResults []bool
	pressureResults  []pressureSample

	acquireCalls   int
	heartbeatCalls int
	pressureCalls  int
	releaseCalls   int

	onPressureProbe func(call int)
}

func (f *fakeSchedulerDB) nextAcquire() leaseAcquireResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.acquireCalls++
	if len(f.acquireResults) == 0 {
		return leaseAcquireResult{}
	}
	out := f.acquireResults[0]
	f.acquireResults = f.acquireResults[1:]
	return out
}

func (f *fakeSchedulerDB) nextHeartbeat() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heartbeatCalls++
	if len(f.heartbeatResults) == 0 {
		return true
	}
	out := f.heartbeatResults[0]
	f.heartbeatResults = f.heartbeatResults[1:]
	return out
}

func (f *fakeSchedulerDB) nextPressure() pressureSample {
	f.mu.Lock()
	f.pressureCalls++
	call := f.pressureCalls
	hook := f.onPressureProbe
	var out pressureSample
	if len(f.pressureResults) == 0 {
		out = pressureSample{active: 1, max: 100}
	} else {
		out = f.pressureResults[0]
		f.pressureResults = f.pressureResults[1:]
	}
	f.mu.Unlock()

	if hook != nil {
		hook(call)
	}
	return out
}

func (f *fakeSchedulerDB) incRelease() {
	f.mu.Lock()
	f.releaseCalls++
	f.mu.Unlock()
}

type fakeSchedulerDriver struct{ conn *fakeSchedulerDB }

func (d *fakeSchedulerDriver) Open(_ string) (driver.Conn, error) {
	return &fakeSchedulerConn{state: d.conn}, nil
}

type fakeSchedulerConn struct{ state *fakeSchedulerDB }

func (c *fakeSchedulerConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeSchedulerStmt{state: c.state, query: query}, nil
}

func (c *fakeSchedulerConn) Close() error { return nil }

func (c *fakeSchedulerConn) Begin() (driver.Tx, error) { return fakeSchedulerTx{}, nil }

type fakeSchedulerTx struct{}

func (fakeSchedulerTx) Commit() error   { return nil }
func (fakeSchedulerTx) Rollback() error { return nil }

type fakeSchedulerStmt struct {
	state *fakeSchedulerDB
	query string
}

func (s *fakeSchedulerStmt) Close() error  { return nil }
func (s *fakeSchedulerStmt) NumInput() int { return -1 }

func (s *fakeSchedulerStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if strings.Contains(strings.ToLower(s.query), "release_lease") {
		s.state.incRelease()
	}
	return fakeSchedulerResult(1), nil
}

func (s *fakeSchedulerStmt) Query(_ []driver.Value) (driver.Rows, error) {
	lower := strings.ToLower(s.query)
	switch {
	case strings.Contains(lower, "acquire_lease"):
		res := s.state.nextAcquire()
		return &fakeSchedulerRows{cols: []string{"acquired", "epoch"}, rows: [][]driver.Value{{res.acquired, res.epoch}}}, nil
	case strings.Contains(lower, "heartbeat_lease"):
		ok := s.state.nextHeartbeat()
		return &fakeSchedulerRows{cols: []string{"heartbeat_lease"}, rows: [][]driver.Value{{ok}}}, nil
	case strings.Contains(lower, "pg_stat_activity"):
		sample := s.state.nextPressure()
		return &fakeSchedulerRows{cols: []string{"active", "max_connections"}, rows: [][]driver.Value{{sample.active, sample.max}}}, nil
	default:
		return &fakeSchedulerRows{cols: []string{"ok"}, rows: [][]driver.Value{{int64(1)}}}, nil
	}
}

type fakeSchedulerRows struct {
	cols []string
	rows [][]driver.Value
	idx  int
}

func (r *fakeSchedulerRows) Columns() []string { return r.cols }
func (r *fakeSchedulerRows) Close() error      { return nil }

func (r *fakeSchedulerRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	for i := range r.rows[r.idx] {
		dest[i] = r.rows[r.idx][i]
	}
	r.idx++
	return nil
}

type fakeSchedulerResult int64

func (r fakeSchedulerResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeSchedulerResult) RowsAffected() (int64, error) { return int64(r), nil }

func newFakeSchedulerDB(t *testing.T, state *fakeSchedulerDB) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("fake_scheduler_%p", state)
	sql.Register(name, &fakeSchedulerDriver{conn: state})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open fake scheduler db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newTestScheduler(db *sql.DB) *Scheduler {
	s := New(db, "test-leader")
	s.heartbeatEvery = 5 * time.Millisecond
	s.drainWait = 200 * time.Millisecond
	s.forceWait = 100 * time.Millisecond
	s.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	return s
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("condition was not met within %v", timeout)
}

func TestScheduler_LeaseAcquired_JobRuns(t *testing.T) {
	state := &fakeSchedulerDB{
		acquireResults:   []leaseAcquireResult{{acquired: true, epoch: 11}},
		heartbeatResults: []bool{true},
		pressureResults:  []pressureSample{{active: 1, max: 100}},
	}
	db := newFakeSchedulerDB(t, state)
	s := newTestScheduler(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var runs atomic.Int64
	s.Register(JobConfig{
		Name:     "lease-acquired",
		Interval: 5 * time.Millisecond,
		Policy:   SkipOnPressure,
		Fn: func(_ context.Context, _ int64) error {
			runs.Add(1)
			cancel()
			return nil
		},
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Start(ctx)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop in time")
	}

	if got := runs.Load(); got != 1 {
		t.Fatalf("runs = %d; want 1", got)
	}
}

func TestScheduler_LeaseNotAcquired_JobSkipped(t *testing.T) {
	state := &fakeSchedulerDB{
		acquireResults:  []leaseAcquireResult{{acquired: false, epoch: -1}, {acquired: false, epoch: -1}},
		pressureResults: []pressureSample{{active: 1, max: 100}, {active: 1, max: 100}},
	}
	db := newFakeSchedulerDB(t, state)
	s := newTestScheduler(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var runs atomic.Int64
	s.Register(JobConfig{
		Name:     "lease-not-acquired",
		Interval: 5 * time.Millisecond,
		Policy:   SkipOnPressure,
		Fn: func(_ context.Context, _ int64) error {
			runs.Add(1)
			return nil
		},
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Start(ctx)
	}()

	waitUntil(t, time.Second, func() bool {
		state.mu.Lock()
		defer state.mu.Unlock()
		return state.acquireCalls >= 1
	})
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop in time")
	}

	if got := runs.Load(); got != 0 {
		t.Fatalf("runs = %d; want 0", got)
	}
}

func TestScheduler_EpochStolenMidRun(t *testing.T) {
	state := &fakeSchedulerDB{
		acquireResults:   []leaseAcquireResult{{acquired: true, epoch: 7}},
		heartbeatResults: []bool{true, false},
		pressureResults:  []pressureSample{{active: 1, max: 100}},
	}
	db := newFakeSchedulerDB(t, state)
	s := newTestScheduler(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cancelled := make(chan struct{})
	s.Register(JobConfig{
		Name:     "stolen",
		Interval: 5 * time.Millisecond,
		Policy:   SkipOnPressure,
		Fn: func(ctx context.Context, _ int64) error {
			<-ctx.Done()
			close(cancelled)
			return ctx.Err()
		},
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Start(ctx)
	}()

	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("job context was not cancelled by heartbeat")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop in time")
	}
}

func TestScheduler_BackpressureSkip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := &fakeSchedulerDB{
		acquireResults:  []leaseAcquireResult{{acquired: true, epoch: 1}, {acquired: true, epoch: 2}, {acquired: true, epoch: 3}, {acquired: true, epoch: 4}},
		pressureResults: []pressureSample{{active: 80, max: 100}, {active: 80, max: 100}, {active: 80, max: 100}, {active: 80, max: 100}},
		onPressureProbe: func(call int) {
			if call == 4 {
				cancel()
			}
		},
	}
	db := newFakeSchedulerDB(t, state)
	s := newTestScheduler(db)

	var runs atomic.Int64
	s.Register(JobConfig{
		Name:     "bp-skip",
		Interval: 5 * time.Millisecond,
		Policy:   SkipOnPressure,
		Fn: func(_ context.Context, _ int64) error {
			runs.Add(1)
			return nil
		},
	})

	s.Start(ctx)

	if got := runs.Load(); got != 3 {
		t.Fatalf("runs = %d; want 3 (tick 4 skipped)", got)
	}
	if got := s.Metrics.SkipsTotal["bp-skip"]; got < 1 {
		t.Fatalf("SkipsTotal = %d; want >= 1", got)
	}
}

func TestScheduler_DrainWaitsForInFlight(t *testing.T) {
	state := &fakeSchedulerDB{
		acquireResults:  []leaseAcquireResult{{acquired: true, epoch: 5}},
		pressureResults: []pressureSample{{active: 1, max: 100}},
	}
	db := newFakeSchedulerDB(t, state)
	s := newTestScheduler(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	finished := make(chan struct{})
	releaseJob := make(chan struct{})

	s.Register(JobConfig{
		Name:     "drain",
		Interval: 5 * time.Millisecond,
		Policy:   SkipOnPressure,
		Fn: func(_ context.Context, _ int64) error {
			close(started)
			<-releaseJob
			close(finished)
			return nil
		},
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Start(ctx)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("job did not start")
	}

	cancel()

	select {
	case <-done:
		t.Fatal("scheduler returned before in-flight job finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseJob)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("job did not finish")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after in-flight completion")
	}
}

func TestScheduler_Metrics_IncrementOnRun(t *testing.T) {
	state := &fakeSchedulerDB{
		acquireResults:  []leaseAcquireResult{{acquired: true, epoch: 9}},
		pressureResults: []pressureSample{{active: 1, max: 100}},
	}
	db := newFakeSchedulerDB(t, state)
	s := newTestScheduler(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Register(JobConfig{
		Name:     "metrics",
		Interval: 5 * time.Millisecond,
		Policy:   SkipOnPressure,
		Fn: func(_ context.Context, _ int64) error {
			cancel()
			return nil
		},
	})

	s.Start(ctx)

	if got := s.Metrics.RunsTotal["metrics"]; got != 1 {
		t.Fatalf("RunsTotal[metrics] = %d; want 1", got)
	}
}
