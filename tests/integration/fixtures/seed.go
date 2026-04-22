//go:build integration
// +build integration

package fixtures

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"metaldocs/tests/integration/testdb"
)

// SeedTenant is a no-op placeholder; tenant IDs are inserted in dependent rows.
func SeedTenant(t *testing.T, ctx context.Context, db *sql.DB, schema, tenantID string) {
	t.Helper()
	_ = ctx
	_ = db
	_ = schema
	_ = tenantID
}

// SeedUser inserts an iam_users row.
func SeedUser(t *testing.T, ctx context.Context, db *sql.DB, schema, userID, displayName string) {
	t.Helper()
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (user_id, display_name, is_active, created_at, updated_at)
		VALUES ($1, $2, true, now(), now())
		ON CONFLICT (user_id) DO NOTHING`,
		testdb.Qualified(schema, "iam_users")),
		userID, displayName,
	); err != nil {
		t.Fatalf("SeedUser: %v", err)
	}
}

// SeedDocument inserts a minimal documents row.
func SeedDocument(t *testing.T, ctx context.Context, db *sql.DB, schema, docID, tenantID, createdBy string) {
	t.Helper()
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, tenant_id, name, status, created_by, revision_version, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, 'Test Document', 'draft', $3, 1, now(), now())
		ON CONFLICT (id) DO NOTHING`,
		testdb.Qualified(schema, "documents")),
		docID, tenantID, createdBy,
	); err != nil {
		t.Fatalf("SeedDocument: %v", err)
	}
}

// SeedRouteConfig inserts a minimal approval_routes row.
func SeedRouteConfig(t *testing.T, ctx context.Context, db *sql.DB, schema, routeID, tenantID, profileCode string) {
	t.Helper()
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, tenant_id, name, profile_code, active, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, 'Test Route', $3, true, now(), now())
		ON CONFLICT (id) DO NOTHING`,
		testdb.Qualified(schema, "approval_routes")),
		routeID, tenantID, profileCode,
	); err != nil {
		t.Fatalf("SeedRouteConfig: %v", err)
	}
}

// Cleanup removes approval-related rows for the tenant.
func Cleanup(t *testing.T, ctx context.Context, db *sql.DB, schema, tenantID string) {
	t.Helper()
	tables := []string{
		"approval_signoffs",
		"approval_stage_instances",
		"approval_instances",
		"approval_route_stages",
		"approval_routes",
		"governance_events",
		"idempotency_keys",
		"job_leases",
	}
	for _, tbl := range tables {
		_, _ = db.ExecContext(ctx,
			fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", testdb.Qualified(schema, tbl)),
			tenantID,
		)
	}
}

// FakeClock returns a fixed time for deterministic tests.
func FakeClock() time.Time {
	return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
}
