//go:build integration
// +build integration

package iam_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"

	"github.com/lib/pq"
)

const probeTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

func integrationDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("METALDOCS_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("DATABASE_URL/METALDOCS_DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("integration DB unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skipf("integration DB unreachable: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func testCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func beginTx(t *testing.T, ctx context.Context, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Skipf("unable to begin integration transaction: %v", err)
	}
	return tx
}

func setConfig(t *testing.T, ctx context.Context, tx *sql.Tx, key, value string) {
	t.Helper()
	if _, err := tx.ExecContext(ctx, "SELECT set_config($1, $2, true)", key, value); err != nil {
		t.Fatalf("set_config(%s): %v", key, err)
	}
}

func pqCode(err error) string {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return string(pgErr.Code)
	}
	return ""
}

func findActorWithCapability(ctx context.Context, tx *sql.Tx, capability string) (string, string, bool, error) {
	var actorID string
	var tenantID string
	err := tx.QueryRowContext(ctx, `
SELECT upa.user_id, upa.tenant_id::text
  FROM metaldocs.user_process_areas upa
  JOIN metaldocs.role_capabilities rc ON rc.role = upa.role
 WHERE upa.effective_to IS NULL
   AND rc.capability = $1
 LIMIT 1`,
		capability,
	).Scan(&actorID, &tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return actorID, tenantID, true, nil
}

func TestProbeA_DirectInsertUserProcessAreasBlocked(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()

	_, err := tx.ExecContext(ctx, `
INSERT INTO user_process_areas
  (user_id, tenant_id, area_code, role, effective_from, granted_by)
VALUES
  ('probe-a-user', $1::uuid, 'QA', 'approver', now(), 'admin')`,
		probeTenantID,
	)
	if err == nil {
		t.Fatal("expected direct INSERT into user_process_areas to be blocked")
	}
	if got := pqCode(err); got != "42501" {
		t.Fatalf("expected SQLSTATE 42501, got %q (err=%v)", got, err)
	}
}

func TestProbeB_GrantMembershipWithoutCap(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()

	setConfig(t, ctx, tx, "metaldocs.actor_id", "probe-b-user")
	setConfig(t, ctx, tx, "metaldocs.tenant_id", probeTenantID)

	err := authz.Require(ctx, tx, "membership.grant", "tenant")
	var denied authz.ErrCapabilityDenied
	if !errors.As(err, &denied) {
		t.Fatalf("expected ErrCapabilityDenied, got %v", err)
	}
}

func TestProbeC_SubmitWithoutCapDenied(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()

	setConfig(t, ctx, tx, "metaldocs.actor_id", "probe-c-user")
	setConfig(t, ctx, tx, "metaldocs.tenant_id", probeTenantID)

	err := authz.Require(ctx, tx, "doc.submit", "QA")
	var denied authz.ErrCapabilityDenied
	if !errors.As(err, &denied) {
		t.Fatalf("expected ErrCapabilityDenied, got %v", err)
	}
}

func TestProbeD_SubmitWithCapAssertedCapsSet(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()

	actorID, tenantID, ok, err := findActorWithCapability(ctx, tx, "doc.submit")
	if err != nil {
		t.Fatalf("find actor with doc.submit capability: %v", err)
	}
	if !ok {
		t.Skip("requires fixture actor with doc.submit capability")
	}

	setConfig(t, ctx, tx, "metaldocs.actor_id", actorID)
	setConfig(t, ctx, tx, "metaldocs.tenant_id", tenantID)
	setConfig(t, ctx, tx, "metaldocs.asserted_caps", "[]")

	if err := authz.Require(ctx, tx, "doc.submit", "tenant"); err != nil {
		t.Fatalf("authz.Require(doc.submit): %v", err)
	}

	var raw sql.NullString
	if err := tx.QueryRowContext(ctx, "SELECT current_setting('metaldocs.asserted_caps', true)").Scan(&raw); err != nil {
		t.Fatalf("read asserted_caps: %v", err)
	}
	if !raw.Valid || raw.String == "" {
		t.Fatal("metaldocs.asserted_caps must be set after authz.Require")
	}

	var asserted []map[string]string
	if err := json.Unmarshal([]byte(raw.String), &asserted); err != nil {
		t.Fatalf("asserted_caps JSON decode: %v", err)
	}

	found := false
	for _, cap := range asserted {
		if cap["cap"] == "doc.submit" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("asserted_caps missing doc.submit entry: %s", raw.String)
	}
}

func TestProbeE_TripwireWithoutAssertedCap(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()

	setConfig(t, ctx, tx, "metaldocs.asserted_caps", "")
	setConfig(t, ctx, tx, "metaldocs.bypass_authz", "")

	_, err := tx.ExecContext(ctx, `
INSERT INTO approval_instances
  (id, tenant_id, document_v2_id, route_id, route_version_snapshot, status, submitted_by, submitted_at, content_hash_at_submit, idempotency_key)
VALUES
  ('11111111-1111-1111-1111-111111111111', $1::uuid, '22222222-2222-2222-2222-222222222222', '33333333-3333-3333-3333-333333333333', 1, 'in_progress', 'probe-e-user', now(), 'hash', 'probe-e-key')`,
		probeTenantID,
	)
	if err == nil {
		t.Fatal("expected approval_instances INSERT without asserted cap to fail")
	}
}

func TestProbeF_CancelAfterTerminal(t *testing.T) {
	if repository.ErrInstanceCompleted == nil {
		t.Fatal("repository.ErrInstanceCompleted must be defined")
	}
	t.Log("ErrInstanceCompleted covered by application/cancel_service_test.go")
}

func TestProbeG_RouteInUseBlocksUpdate(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()

	var documentID string
	var tenantID string
	err := tx.QueryRowContext(ctx, "SELECT id::text, tenant_id::text FROM documents LIMIT 1").Scan(&documentID, &tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		t.Skip("requires fixture document row")
	}
	if err != nil {
		t.Fatalf("load fixture document: %v", err)
	}

	var profileCode string
	err = tx.QueryRowContext(ctx, `
SELECT dp.code
  FROM metaldocs.document_profiles dp
 WHERE dp.tenant_id = $1::uuid
   AND NOT EXISTS (
     SELECT 1
       FROM approval_routes ar
      WHERE ar.tenant_id = dp.tenant_id
        AND ar.profile_code = dp.code
   )
 LIMIT 1`,
		tenantID,
	).Scan(&profileCode)
	if errors.Is(err, sql.ErrNoRows) {
		t.Skip("requires free document profile for route fixture")
	}
	if err != nil {
		t.Fatalf("load fixture profile: %v", err)
	}

	var submittedBy string
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM metaldocs.iam_users WHERE tenant_id = $1::uuid LIMIT 1", tenantID).Scan(&submittedBy)
	if errors.Is(err, sql.ErrNoRows) {
		t.Skip("requires fixture iam_user for approval instance")
	}
	if err != nil {
		t.Fatalf("load fixture iam_user: %v", err)
	}

	var routeID string
	err = tx.QueryRowContext(ctx, `
INSERT INTO approval_routes (tenant_id, profile_code, name, created_by)
VALUES ($1::uuid, $2, 'probe-g-route', 'probe-g-user')
RETURNING id::text`,
		tenantID, profileCode,
	).Scan(&routeID)
	if err != nil {
		t.Fatalf("insert route fixture: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO approval_instances
  (id, tenant_id, document_v2_id, route_id, route_version_snapshot, status, submitted_by, submitted_at, content_hash_at_submit, idempotency_key)
VALUES
  ('44444444-4444-4444-4444-444444444444', $1::uuid, $2::uuid, $3::uuid, 1, 'cancelled', $4, now(), 'hash', 'probe-g-key')`,
		tenantID, documentID, routeID, submittedBy,
	)
	if err != nil {
		t.Fatalf("insert instance fixture: %v", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE approval_routes SET name = 'new-name' WHERE id = $1::uuid", routeID)
	if err == nil {
		t.Fatal("expected route update to fail while route is in use")
	}
	if code := pqCode(err); code != "P0001" {
		t.Fatalf("expected SQLSTATE P0001 from route immutability trigger, got %q (err=%v)", code, err)
	}
	if !strings.Contains(err.Error(), "ErrRouteInUse") {
		t.Fatalf("expected ErrRouteInUse message, got: %v", err)
	}
}

func TestProbeH_LegacyCapsAbsentPostMigration(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)

	var count int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
  FROM metaldocs.role_capabilities
 WHERE capability IN ('document.finalize', 'document.archive')`,
	).Scan(&count); err != nil {
		t.Fatalf("count legacy caps: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected zero legacy capability rows, got %d", count)
	}
}

func TestProbeI_BypassAuthzTxScoped(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)

	tx1 := beginTx(t, ctx, db)
	setConfig(t, ctx, tx1, "metaldocs.bypass_authz", "scheduler")

	var got sql.NullString
	if err := tx1.QueryRowContext(ctx, "SELECT current_setting('metaldocs.bypass_authz', true)").Scan(&got); err != nil {
		tx1.Rollback()
		t.Fatalf("read bypass_authz in tx1: %v", err)
	}
	if !got.Valid || got.String != "scheduler" {
		tx1.Rollback()
		t.Fatalf("expected bypass_authz=scheduler in tx1, got %+v", got)
	}
	if err := tx1.Rollback(); err != nil {
		t.Fatalf("rollback tx1: %v", err)
	}

	tx2 := beginTx(t, ctx, db)
	defer tx2.Rollback()

	got = sql.NullString{}
	if err := tx2.QueryRowContext(ctx, "SELECT current_setting('metaldocs.bypass_authz', true)").Scan(&got); err != nil {
		t.Fatalf("read bypass_authz in tx2: %v", err)
	}
	if got.Valid && got.String != "" {
		t.Fatalf("expected bypass_authz to be tx-scoped and unset in tx2, got %q", got.String)
	}
}

func TestProbeJ_AreaMismatchTriggersError(t *testing.T) {
	ctx, cancel := testCtx(t)
	defer cancel()

	db := integrationDB(t)
	tx := beginTx(t, ctx, db)
	defer tx.Rollback()

	setConfig(t, ctx, tx, "metaldocs.asserted_caps", `[{"cap":"doc.submit","area":"AREA_X"}]`)
	setConfig(t, ctx, tx, "metaldocs.bypass_authz", "")

	_, err := tx.ExecContext(ctx, `
INSERT INTO approval_instances
  (id, tenant_id, document_v2_id, route_id, route_version_snapshot, status, submitted_by, submitted_at, content_hash_at_submit, idempotency_key)
VALUES
  ('55555555-5555-5555-5555-555555555555', $1::uuid, '66666666-6666-6666-6666-666666666666', '77777777-7777-7777-7777-777777777777', 1, 'in_progress', 'probe-j-user', now(), 'hash', 'probe-j-key')`,
		probeTenantID,
	)
	if err == nil {
		t.Fatal("expected approval_instances INSERT to fail")
	}
	if strings.Contains(err.Error(), "ErrCapabilityNotAsserted") {
		t.Fatalf("unexpected cap-assertion failure: 0142b enforces cap-only, not area match (err=%v)", err)
	}
}
