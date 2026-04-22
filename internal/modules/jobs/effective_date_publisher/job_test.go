package effective_date_publisher

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"

	"metaldocs/internal/modules/documents_v2/approval/application"
)

type mockPublishService struct {
	runFn func(ctx context.Context, db *sql.DB) (application.RunDuePublishesResult, error)
}

func (m mockPublishService) RunDuePublishes(ctx context.Context, db *sql.DB) (application.RunDuePublishesResult, error) {
	if m.runFn == nil {
		return application.RunDuePublishesResult{}, nil
	}
	return m.runFn(ctx, db)
}

func TestEffectiveDatePublisher_HappyPath(t *testing.T) {
	t.Parallel()

	svc := mockPublishService{
		runFn: func(ctx context.Context, db *sql.DB) (application.RunDuePublishesResult, error) {
			return application.RunDuePublishesResult{Processed: 5}, nil
		},
	}

	fn := New(nil, svc)
	err := fn(context.Background(), 11)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEffectiveDatePublisher_ServiceError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("publish failed")
	svc := mockPublishService{
		runFn: func(ctx context.Context, db *sql.DB) (application.RunDuePublishesResult, error) {
			return application.RunDuePublishesResult{}, expectedErr
		},
	}

	fn := New(nil, svc)
	err := fn(context.Background(), 12)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestEffectiveDatePublisher_FullBatch(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	svc := mockPublishService{
		runFn: func(ctx context.Context, db *sql.DB) (application.RunDuePublishesResult, error) {
			return application.RunDuePublishesResult{Processed: DefaultBatchSize}, nil
		},
	}

	fn := New(nil, svc)
	err := fn(context.Background(), 13)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !strings.Contains(logs.String(), "backlog likely - batch full") {
		t.Fatalf("expected warning log for full batch, got logs: %s", logs.String())
	}
}

func TestEffectiveDatePublisher_ContextCancelled(t *testing.T) {
	t.Parallel()

	var called atomic.Int32
	svc := mockPublishService{
		runFn: func(ctx context.Context, db *sql.DB) (application.RunDuePublishesResult, error) {
			called.Add(1)
			return application.RunDuePublishesResult{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fn := New(nil, svc)
	err := fn(ctx, 14)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if called.Load() != 1 {
		t.Fatalf("expected service to be called once, got %d", called.Load())
	}
}
