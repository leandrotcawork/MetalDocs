//go:build integration
// +build integration

package scenarios_test

import (
	"context"
	"encoding/json"
	"testing"

	"metaldocs/tests/integration/fixtures"
	"metaldocs/tests/integration/testdb"
)

func TestOutbox_ApprovalInstanceInsertHasGovernanceEvent(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tenantID := testdb.DeterministicID(t, "tenant")
	authorID := testdb.DeterministicID(t, "author")
	docID := testdb.DeterministicID(t, "doc")
	routeID := testdb.DeterministicID(t, "route")
	instanceID := testdb.DeterministicID(t, "instance")

	fixtures.SeedUser(t, ctx, db, "metaldocs", authorID, "Outbox Author")
	fixtures.SeedDocument(t, ctx, db, "metaldocs", docID, tenantID, authorID)
	fixtures.SeedRouteConfig(t, ctx, db, "metaldocs", routeID, tenantID, "OUTBOX_FLOW")

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.approval_instances WHERE id = $1::uuid`, instanceID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.governance_events WHERE tenant_id = $1::uuid AND event_type = 'doc.submitted' AND resource_id = $2`, tenantID, docID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.approval_routes WHERE id = $1::uuid`, routeID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.documents WHERE id = $1::uuid`, docID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.iam_users WHERE tenant_id = $1::uuid`, tenantID)
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	if _, err := tx.ExecContext(ctx, `SELECT set_config('metaldocs.bypass_authz', 'scheduler', true)`); err != nil {
		_ = tx.Rollback()
		t.Fatalf("set bypass_authz: %v", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO metaldocs.approval_instances
			(id, tenant_id, document_v2_id, route_id, route_version_snapshot, status, submitted_by, submitted_at, content_hash_at_submit, idempotency_key)
		VALUES
			($1::uuid, $2::uuid, $3::uuid, $4::uuid, 1, 'in_progress', $5, now(), 'outbox-hash', 'outbox-idem')`,
		instanceID, tenantID, docID, routeID, authorID,
	); err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert approval_instance: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"instance_id": instanceID,
		"document_id": docID,
	})
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO metaldocs.governance_events
			(tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
		VALUES
			($1::uuid, 'doc.submitted', $2, 'document', $3, 'integration outbox pairing', $4::jsonb)`,
		tenantID, authorID, docID, string(payload),
	); err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert governance_event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	var instanceTenant string
	if err := db.QueryRowContext(ctx, `
		SELECT tenant_id::text
		  FROM metaldocs.approval_instances
		 WHERE id = $1::uuid`,
		instanceID,
	).Scan(&instanceTenant); err != nil {
		t.Fatalf("read approval_instance: %v", err)
	}

	var eventTenant string
	if err := db.QueryRowContext(ctx, `
		SELECT tenant_id::text
		  FROM metaldocs.governance_events
		 WHERE tenant_id = $1::uuid
		   AND event_type = 'doc.submitted'
		   AND resource_type = 'document'
		   AND resource_id = $2`,
		tenantID, docID,
	).Scan(&eventTenant); err != nil {
		t.Fatalf("read governance_event: %v", err)
	}

	if instanceTenant != eventTenant {
		t.Fatalf("tenant mismatch: approval_instance=%s governance_event=%s", instanceTenant, eventTenant)
	}
}

func TestOutbox_RollbackOmitsEvent(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tenantID := testdb.DeterministicID(t, "tenant-rollback")
	resourceID := testdb.DeterministicID(t, "resource-rollback")
	actorID := "outbox-rollback-user"

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO metaldocs.governance_events
			(tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
		VALUES
			($1::uuid, 'doc.submitted', $2, 'document', $3, 'rollback-test', '{"rollback":true}'::jsonb)`,
		tenantID, actorID, resourceID,
	); err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert event in tx: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		  FROM metaldocs.governance_events
		 WHERE tenant_id = $1::uuid
		   AND event_type = 'doc.submitted'
		   AND resource_id = $2`,
		tenantID, resourceID,
	).Scan(&count); err != nil {
		t.Fatalf("count events after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to remove event row; found %d row(s)", count)
	}
}

func TestOutbox_DedupeKey(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	var dedupeColumnExists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM information_schema.columns
			 WHERE table_schema = 'metaldocs'
			   AND table_name = 'governance_events'
			   AND column_name = 'dedupe_key'
		)`,
	).Scan(&dedupeColumnExists); err != nil {
		t.Fatalf("check dedupe_key column: %v", err)
	}
	if !dedupeColumnExists {
		t.Skip("governance_events.dedupe_key not present in this database")
	}

	tenantID := testdb.DeterministicID(t, "tenant-dedupe")
	actorID := "outbox-dedupe-user"
	resourceID := "outbox-dedupe-resource"
	dedupeKey := "test-dedup-key-1"
	eventType := "doc.submitted"

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `
			DELETE FROM metaldocs.governance_events
			 WHERE tenant_id = $1::uuid
			   AND event_type = $2
			   AND dedupe_key = $3`,
			tenantID, eventType, dedupeKey,
		)
	})

	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.governance_events
			(tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json, dedupe_key)
		VALUES
			($1::uuid, $2, $3, 'document', $4, 'dedupe-1', '{}'::jsonb, $5)`,
		tenantID, eventType, actorID, resourceID, dedupeKey,
	); err != nil {
		t.Fatalf("seed dedupe event: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.governance_events
			(tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json, dedupe_key)
		VALUES
			($1::uuid, $2, $3, 'document', $4, 'dedupe-2', '{}'::jsonb, $5)
		ON CONFLICT DO NOTHING`,
		tenantID, eventType, actorID, resourceID, dedupeKey,
	); err != nil {
		t.Fatalf("insert duplicate dedupe key: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		  FROM metaldocs.governance_events
		 WHERE tenant_id = $1::uuid
		   AND event_type = $2
		   AND dedupe_key = $3`,
		tenantID, eventType, dedupeKey,
	).Scan(&count); err != nil {
		t.Fatalf("count dedupe rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one deduped governance_event row, got %d", count)
	}
}
