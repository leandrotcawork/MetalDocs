package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
)

func RunLeaseReaper(db *sql.DB) JobFunc {
	return func(ctx context.Context, epoch int64) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		rows, err := tx.QueryContext(ctx, `
DELETE FROM metaldocs.job_leases
WHERE job_name IN (
	SELECT job_name FROM metaldocs.job_leases
	WHERE expires_at < now() - interval '10 minutes'
	FOR UPDATE SKIP LOCKED
)
RETURNING job_name, leader_id, lease_epoch
`)
		if err != nil {
			return err
		}
		defer rows.Close()

		reclaimed := 0
		for rows.Next() {
			var jobName string
			var leaderID string
			var leaseEpoch int64
			if err := rows.Scan(&jobName, &leaderID, &leaseEpoch); err != nil {
				return err
			}

			payloadJSON, err := json.Marshal(map[string]any{
				"job_name":    jobName,
				"leader_id":   leaderID,
				"lease_epoch": leaseEpoch,
			})
			if err != nil {
				return err
			}

			if _, err := tx.ExecContext(ctx, `
INSERT INTO governance_events
	(tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json, occurred_at)
VALUES ('system', 'lease.reaped', 'system:reaper', 'job_lease', $1, 'expired_lease_reclaimed', $2, now())
`, jobName, payloadJSON); err != nil {
				return err
			}

			reclaimed++
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}

		slog.InfoContext(ctx, "lease_reaper: tick complete",
			"job", "lease_reaper",
			"epoch", epoch,
			"reclaimed", reclaimed)
		return nil
	}
}
