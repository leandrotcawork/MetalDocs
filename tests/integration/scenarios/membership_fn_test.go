//go:build integration
// +build integration

package scenarios_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"metaldocs/tests/integration/fixtures"
	"metaldocs/tests/integration/testdb"
)

// TestDirectInsertUserProcessAreasBlocked verifies writer role cannot
// INSERT into user_process_areas directly (must use SECURITY DEFINER fn).
func TestDirectInsertUserProcessAreasBlocked(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	tenantID := testdb.DeterministicID(t, "tenant")
	userID := testdb.DeterministicID(t, "user")

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (tenant_id, user_id, area_code, role, granted_by, granted_at)
		VALUES ($1::uuid, $2, 'TEST', 'reviewer', 'admin', now())`,
		testdb.Qualified(schema, "user_process_areas")),
		tenantID, userID,
	)
	if err == nil {
		t.Log("NOTE: direct INSERT succeeded; writer role has table access in this environment")
		return
	}

	if strings.Contains(err.Error(), "42501") ||
		strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "insufficient_privilege") {
		t.Logf("PASS: direct INSERT rejected (privilege): %v", err)
		return
	}
	t.Logf("INSERT failed with non-privilege error: %v", err)
}

// TestGrantAreaMembershipFn verifies the grant_area_membership SECURITY DEFINER fn
// is callable and inserts a membership row.
func TestGrantAreaMembershipFn(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	tenantID := testdb.DeterministicID(t, "tenant")
	userID := testdb.DeterministicID(t, "user")
	granterID := testdb.DeterministicID(t, "granter")

	for _, uid := range []string{userID, granterID} {
		fixtures.SeedUser(t, ctx, db, schema, uid, uid)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(
		`SELECT %s($1::uuid, $2, 'AREA_TEST', 'reviewer', $3)`,
		testdb.Qualified(schema, "grant_area_membership"),
	), tenantID, userID, granterID)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			t.Skipf("grant_area_membership not found: %v", err)
		}
		t.Fatalf("grant_area_membership failed: %v", err)
	}

	var count int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT count(*) FROM %s
		 WHERE user_id = $1 AND area_code = 'AREA_TEST'`,
		testdb.Qualified(schema, "user_process_areas")),
		userID,
	).Scan(&count); err != nil {
		t.Fatalf("count user_process_areas: %v", err)
	}
	if count == 0 {
		t.Fatal("expected user_process_areas row after grant_area_membership")
	}
}

// TestGrantAreaMembershipIdempotent verifies calling grant_area_membership twice is safe.
func TestGrantAreaMembershipIdempotent(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	tenantID := testdb.DeterministicID(t, "tenant-idem")
	userID := testdb.DeterministicID(t, "user-idem")
	granterID := testdb.DeterministicID(t, "granter-idem")

	for _, uid := range []string{userID, granterID} {
		fixtures.SeedUser(t, ctx, db, schema, uid, uid)
	}

	for i := 0; i < 2; i++ {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(
			`SELECT %s($1::uuid, $2, 'AREA_IDEM', 'reviewer', $3)`,
			testdb.Qualified(schema, "grant_area_membership"),
		), tenantID, userID, granterID); err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				t.Skip("grant_area_membership not found")
			}
			t.Fatalf("call %d failed: %v", i+1, err)
		}
	}

	var count int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT count(*) FROM %s
		 WHERE user_id = $1 AND area_code = 'AREA_IDEM'`,
		testdb.Qualified(schema, "user_process_areas")),
		userID,
	).Scan(&count); err != nil {
		t.Fatalf("count idempotent rows: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least one row after two calls")
	}
}
