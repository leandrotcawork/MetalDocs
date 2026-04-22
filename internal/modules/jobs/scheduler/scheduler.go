package scheduler

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"sync"
	"time"
)

type BackpressurePolicy int

const (
	SkipOnPressure BackpressurePolicy = iota
	DegradeOnPressure
)

type JobFunc func(ctx context.Context, epoch int64) error

type JobConfig struct {
	Name     string
	Interval time.Duration
	Fn       JobFunc
	Policy   BackpressurePolicy
}

type Metrics struct {
	mu sync.Mutex

	RunsTotal   map[string]int64
	ErrorsTotal map[string]int64
	SkipsTotal  map[string]int64
}

func newMetrics() Metrics {
	return Metrics{
		RunsTotal:   map[string]int64{},
		ErrorsTotal: map[string]int64{},
		SkipsTotal:  map[string]int64{},
	}
}

func (m *Metrics) incRun(job string) {
	m.mu.Lock()
	m.RunsTotal[job]++
	m.mu.Unlock()
}

func (m *Metrics) incError(job string) {
	m.mu.Lock()
	m.ErrorsTotal[job]++
	m.mu.Unlock()
}

func (m *Metrics) incSkip(job string) {
	m.mu.Lock()
	m.SkipsTotal[job]++
	m.mu.Unlock()
}

type inFlightJob struct {
	job    string
	epoch  int64
	cancel context.CancelFunc
	done   chan struct{}
}

type Scheduler struct {
	db       *sql.DB
	leaderID string
	jobs     []JobConfig

	pressureCount int
	inPressure    bool
	quietCount    int

	Metrics Metrics

	mu             sync.Mutex
	inFlight       map[*inFlightJob]struct{}
	tickers        []*time.Ticker
	heartbeatEvery time.Duration
	drainWait      time.Duration
	forceWait      time.Duration
	maxSkipStreak  int
	logger         *slog.Logger
}

func New(db *sql.DB, leaderID string) *Scheduler {
	return &Scheduler{
		db:             db,
		leaderID:       leaderID,
		Metrics:        newMetrics(),
		inFlight:       map[*inFlightJob]struct{}{},
		heartbeatEvery: time.Minute,
		drainWait:      30 * time.Second,
		forceWait:      5 * time.Second,
		maxSkipStreak:  10,
		logger:         slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (s *Scheduler) Register(cfg JobConfig) {
	s.jobs = append(s.jobs, cfg)
}

func (s *Scheduler) Start(ctx context.Context) {
	var loops sync.WaitGroup
	for _, cfg := range s.jobs {
		cfg := cfg
		if cfg.Name == "" || cfg.Interval <= 0 || cfg.Fn == nil {
			s.logger.Warn("scheduler_invalid_job_config", "job", cfg.Name)
			continue
		}
		loops.Add(1)
		go func() {
			defer loops.Done()
			s.runJobLoop(ctx, cfg)
		}()
	}

	<-ctx.Done()
	s.stopAllTickers()
	s.drain()

	loopsDone := make(chan struct{})
	go func() {
		defer close(loopsDone)
		loops.Wait()
	}()

	select {
	case <-loopsDone:
	case <-time.After(s.forceWait):
		s.logger.Warn("scheduler_loops_shutdown_timeout")
	}
}

func (s *Scheduler) runJobLoop(ctx context.Context, cfg JobConfig) {
	ticker := time.NewTicker(cfg.Interval)
	s.registerTicker(ticker)
	defer func() {
		ticker.Stop()
		s.unregisterTicker(ticker)
	}()

	skipNext := false
	skipStreak := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			inPressure := s.probePressure(ctx)
			if !inPressure {
				skipNext = false
				skipStreak = 0
			}

			if inPressure {
				shouldSkip := false
				switch cfg.Policy {
				case SkipOnPressure:
					shouldSkip = true
				case DegradeOnPressure:
					if skipNext {
						shouldSkip = true
						skipNext = false
					} else {
						skipNext = true
					}
				}
				if shouldSkip {
					skipStreak++
					s.Metrics.incSkip(cfg.Name)
					s.logger.Warn("scheduler_job_skipped", "job", cfg.Name, "reason", "backpressure", "streak", skipStreak)
					if skipStreak >= s.maxSkipStreak {
						s.logger.Warn("scheduler_backpressure_skip_streak_max", "job", cfg.Name, "streak", skipStreak)
					}
					continue
				}
			}

			acquired, epoch, err := s.acquireLease(ctx, cfg.Name)
			if err != nil {
				s.logger.Error("scheduler_acquire_lease_failed", "job", cfg.Name, "error", err)
				continue
			}
			if !acquired {
				s.logger.Debug("scheduler_lease_held_by_other", "job", cfg.Name)
				continue
			}

			jobBaseCtx := context.WithoutCancel(ctx)
			jobCtx, jobCancel := context.WithCancel(jobBaseCtx)
			inFlight := &inFlightJob{
				job:    cfg.Name,
				epoch:  epoch,
				cancel: jobCancel,
				done:   make(chan struct{}),
			}
			s.addInFlight(inFlight)

			hbStop := make(chan struct{})
			hbDone := make(chan struct{})
			go func() {
				defer close(hbDone)
				s.heartbeatLoop(jobCtx, jobCancel, cfg.Name, s.leaderID, epoch, hbStop)
			}()

			started := time.Now().UTC()
			err = cfg.Fn(jobCtx, epoch)
			duration := time.Since(started)
			if err != nil {
				s.Metrics.incError(cfg.Name)
				s.logger.Error("scheduler_job_failed", "job", cfg.Name, "epoch", epoch, "duration", duration, "error", err)
			}
			s.Metrics.incRun(cfg.Name)
			s.logger.Info("scheduler_job_completed", "job", cfg.Name, "epoch", epoch, "duration", duration)

			close(hbStop)
			<-hbDone
			jobCancel()

			if releaseErr := s.releaseLease(context.Background(), cfg.Name, epoch); releaseErr != nil {
				s.logger.Error("scheduler_release_lease_failed", "job", cfg.Name, "epoch", epoch, "error", releaseErr)
			}

			close(inFlight.done)
			s.removeInFlight(inFlight)
		}
	}
}

func (s *Scheduler) heartbeatLoop(ctx context.Context, cancel context.CancelFunc, job, leader string, epoch int64, stop <-chan struct{}) {
	ticker := time.NewTicker(s.heartbeatEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			ok, err := s.heartbeatLease(ctx, job, leader, epoch)
			if err != nil {
				s.logger.Error("scheduler_heartbeat_failed", "job", job, "epoch", epoch, "error", err)
				continue
			}
			if !ok {
				s.logger.Warn("scheduler_lease_stolen", "job", job, "epoch", epoch)
				cancel()
				return
			}
		}
	}
}

func (s *Scheduler) probePressure(ctx context.Context) bool {
	current := s.currentPressure()

	var active float64
	var maxConn float64
	err := s.db.QueryRowContext(ctx, `
SELECT
    COALESCE((SELECT count(*)::float8 FROM pg_stat_activity WHERE state = 'active'), 0),
    COALESCE(current_setting('max_connections')::float8, 1)
`).Scan(&active, &maxConn)
	if err != nil {
		s.logger.Debug("scheduler_pressure_probe_failed", "error", err)
		return current
	}

	if maxConn <= 0 {
		maxConn = 1
	}
	ratio := active / maxConn

	s.mu.Lock()
	defer s.mu.Unlock()

	if ratio > 0.70 {
		s.pressureCount++
		s.quietCount = 0
		if !s.inPressure && s.pressureCount >= 3 {
			s.inPressure = true
			s.pressureCount = 0
			s.logger.Warn("scheduler_backpressure_enter", "ratio", ratio)
		}
	} else if ratio < 0.60 {
		s.quietCount++
		s.pressureCount = 0
		if s.inPressure && s.quietCount >= 3 {
			s.inPressure = false
			s.quietCount = 0
			s.logger.Info("scheduler_backpressure_exit", "ratio", ratio)
		}
	} else {
		s.pressureCount = 0
		s.quietCount = 0
	}

	return current
}

func (s *Scheduler) currentPressure() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inPressure
}

func (s *Scheduler) acquireLease(ctx context.Context, job string) (bool, int64, error) {
	var acquired bool
	var epoch int64
	err := s.db.QueryRowContext(ctx,
		"SELECT * FROM metaldocs.acquire_lease($1, $2, '5 minutes')",
		job,
		s.leaderID,
	).Scan(&acquired, &epoch)
	return acquired, epoch, err
}

func (s *Scheduler) heartbeatLease(ctx context.Context, job, leader string, epoch int64) (bool, error) {
	var ok bool
	err := s.db.QueryRowContext(ctx,
		"SELECT metaldocs.heartbeat_lease($1, $2, $3)",
		job,
		leader,
		epoch,
	).Scan(&ok)
	return ok, err
}

func (s *Scheduler) releaseLease(ctx context.Context, job string, epoch int64) error {
	_, err := s.db.ExecContext(ctx,
		"SELECT metaldocs.release_lease($1, $2, $3)",
		job,
		s.leaderID,
		epoch,
	)
	return err
}

func (s *Scheduler) drain() {
	if s.waitForInFlight(s.drainWait) {
		return
	}

	s.logger.Warn("scheduler_drain_timeout", "timeout", s.drainWait)
	runs := s.snapshotInFlight()
	for _, r := range runs {
		r.cancel()
	}

	if s.waitForInFlight(s.forceWait) {
		return
	}

	s.logger.Warn("scheduler_force_release", "timeout", s.forceWait)
	runs = s.snapshotInFlight()
	for _, r := range runs {
		if err := s.releaseLease(context.Background(), r.job, r.epoch); err != nil {
			s.logger.Error("scheduler_force_release_failed", "job", r.job, "epoch", r.epoch, "error", err)
			continue
		}
		s.logger.Warn("scheduler_force_release_done", "job", r.job, "epoch", r.epoch)
	}
}

func (s *Scheduler) waitForInFlight(timeout time.Duration) bool {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()

	for {
		if s.inFlightCount() == 0 {
			return true
		}
		select {
		case <-deadline.C:
			return s.inFlightCount() == 0
		case <-tick.C:
		}
	}
}

func (s *Scheduler) inFlightCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.inFlight)
}

func (s *Scheduler) snapshotInFlight() []*inFlightJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	runs := make([]*inFlightJob, 0, len(s.inFlight))
	for r := range s.inFlight {
		runs = append(runs, r)
	}
	return runs
}

func (s *Scheduler) addInFlight(r *inFlightJob) {
	s.mu.Lock()
	s.inFlight[r] = struct{}{}
	s.mu.Unlock()
}

func (s *Scheduler) removeInFlight(r *inFlightJob) {
	s.mu.Lock()
	delete(s.inFlight, r)
	s.mu.Unlock()
}

func (s *Scheduler) registerTicker(t *time.Ticker) {
	s.mu.Lock()
	s.tickers = append(s.tickers, t)
	s.mu.Unlock()
}

func (s *Scheduler) unregisterTicker(t *time.Ticker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.tickers {
		if s.tickers[i] == t {
			s.tickers = append(s.tickers[:i], s.tickers[i+1:]...)
			return
		}
	}
}

func (s *Scheduler) stopAllTickers() {
	s.mu.Lock()
	tickers := append([]*time.Ticker(nil), s.tickers...)
	s.mu.Unlock()
	for _, t := range tickers {
		t.Stop()
	}
}
