//go:build integration

package domain

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"metaldocs/internal/modules/iam/application"

	_ "github.com/lib/pq"
)

func TestRoleCapabilities_VersionBumpEmitsGovernanceEvent(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("METALDOCS_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("integration test skipped: DATABASE_URL and METALDOCS_DATABASE_URL are unset")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("integration test skipped: open db failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("integration test skipped: db unreachable: %v", err)
	}

	const tenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	const eventType = "role.capability_map.version_bump"

	if _, err := db.ExecContext(ctx, `
DELETE FROM governance_events
WHERE tenant_id::text = $1
  AND event_type = $2
`, tenantID, eventType); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if err := application.CheckRoleCapabilitiesVersion(ctx, db, tenantID); err != nil {
		t.Fatalf("first CheckRoleCapabilitiesVersion failed: %v", err)
	}
	count, err := countVersionBumpEvents(ctx, db, tenantID)
	if err != nil {
		t.Fatalf("count after first call failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 governance event after first call, got %d", count)
	}

	if err := application.CheckRoleCapabilitiesVersion(ctx, db, tenantID); err != nil {
		t.Fatalf("second CheckRoleCapabilitiesVersion failed: %v", err)
	}
	count, err = countVersionBumpEvents(ctx, db, tenantID)
	if err != nil {
		t.Fatalf("count after second call failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected idempotent 1 governance event after second call, got %d", count)
	}
}

func countVersionBumpEvents(ctx context.Context, db *sql.DB, tenantID string) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM governance_events
WHERE tenant_id::text = $1
  AND event_type = 'role.capability_map.version_bump'
`, tenantID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
