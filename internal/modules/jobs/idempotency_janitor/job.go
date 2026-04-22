package idempotency_janitor

import (
	"context"
	"database/sql"
	"log/slog"

	"metaldocs/internal/modules/jobs/scheduler"
)

const (
	JobName       = "idempotency_janitor"
	BatchSize     = 5000
	MaxIterations = 10
)

func New(db *sql.DB) scheduler.JobFunc {
	return func(ctx context.Context, epoch int64) error {
		totalDeleted := 0
		for i := 0; i < MaxIterations; i++ {
			result, err := db.ExecContext(ctx, `
DELETE FROM metaldocs.idempotency_keys
WHERE ctid IN (
	SELECT ctid FROM metaldocs.idempotency_keys
	WHERE expires_at < now() AND status = 'completed'
	LIMIT $1
)`, BatchSize)
			if err != nil {
				slog.ErrorContext(ctx, "idempotency_janitor: delete failed", "error", err)
				return err
			}

			n, _ := result.RowsAffected()
			totalDeleted += int(n)
			if n == 0 {
				break
			}
		}

		slog.InfoContext(ctx, "idempotency_janitor: tick complete",
			"job", JobName,
			"epoch", epoch,
			"deleted", totalDeleted)
		return nil
	}
}
