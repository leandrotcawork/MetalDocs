package effective_date_publisher

import (
	"context"
	"database/sql"
	"log/slog"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/jobs/scheduler"
)

const (
	DefaultBatchSize = 100
	JobName          = "effective_date_publisher"
)

// publishService is the subset of application.SchedulerService used by this job.
type publishService interface {
	RunDuePublishes(ctx context.Context, db *sql.DB) (application.RunDuePublishesResult, error)
}

// New returns a scheduler.JobFunc for the effective_date_publisher job.
func New(db *sql.DB, svc publishService) scheduler.JobFunc {
	return func(ctx context.Context, epoch int64) error {
		result, err := svc.RunDuePublishes(ctx, db)
		if err != nil {
			slog.ErrorContext(ctx, "effective_date_publisher: run failed",
				"job", JobName, "epoch", epoch, "error", err)
			return err
		}

		slog.InfoContext(ctx, "effective_date_publisher: tick complete",
			"job", JobName, "epoch", epoch, "processed", result.Processed)

		if result.Processed >= DefaultBatchSize {
			slog.WarnContext(ctx, "effective_date_publisher: backlog likely - batch full",
				"job", JobName, "processed", result.Processed, "batch_size", DefaultBatchSize)
		}

		return nil
	}
}
