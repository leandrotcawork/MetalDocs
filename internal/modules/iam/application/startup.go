package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"metaldocs/internal/modules/iam/domain"
)

func CheckRoleCapabilitiesVersion(ctx context.Context, db *sql.DB, tenantID string) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return fmt.Errorf("tenant id is required")
	}

	var lastVersion sql.NullInt64
	err := db.QueryRowContext(
		ctx,
		`
SELECT MAX((payload_json->>'version')::int)
FROM governance_events
WHERE tenant_id::text = $1
  AND event_type = 'role.capability_map.version_bump'
`,
		tenantID,
	).Scan(&lastVersion)
	if err != nil {
		return fmt.Errorf("load last role capability map version bump: %w", err)
	}

	if lastVersion.Valid && int(lastVersion.Int64) == domain.RoleCapabilitiesVersion {
		return nil
	}

	payloadJSON, err := json.Marshal(map[string]int{
		"version": domain.RoleCapabilitiesVersion,
	})
	if err != nil {
		return fmt.Errorf("marshal governance event payload: %w", err)
	}

	_, err = db.ExecContext(
		ctx,
		`
INSERT INTO governance_events
  (tenant_id, event_type, actor_user_id, resource_type, resource_id, payload_json)
VALUES
  ($1::uuid, 'role.capability_map.version_bump', 'system', 'role_capability_map', $1, $2::jsonb)
`,
		tenantID,
		string(payloadJSON),
	)
	if err != nil {
		if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "development") {
			return fmt.Errorf("insert role capability map version bump event: %w", err)
		}
		slog.Warn(
			"failed to persist role capability map version bump event",
			"tenant_id",
			tenantID,
			"target_version",
			domain.RoleCapabilitiesVersion,
			"error",
			err,
		)
	}

	return nil
}
