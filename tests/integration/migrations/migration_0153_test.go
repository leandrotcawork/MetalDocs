//go:build integration

package migrations_test

import (
	"context"
	"strings"
	"testing"

	"metaldocs/tests/integration/fixtures"
	"metaldocs/tests/integration/testdb"
)

// TestMigration0153_TenantMismatchRejected verifies that inserting a
// document_placeholder_values row with a tenant_id that differs from the
// owning document's tenant_id is rejected by the trigger.
func TestMigration0153_TenantMismatchRejected(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tenantA := testdb.DeterministicID(t, "tenant-a")
	tenantB := testdb.DeterministicID(t, "tenant-b")
	docID := testdb.DeterministicID(t, "doc")
	userID := testdb.DeterministicID(t, "user")

	fixtures.SeedUser(t, ctx, db, schema, userID, "Tenant Mismatch User")
	fixtures.SeedDocument(t, ctx, db, schema, docID, tenantA, userID)

	_, err := db.ExecContext(ctx,
		`INSERT INTO `+testdb.Qualified(schema, "document_placeholder_values")+`
		 (tenant_id, revision_id, placeholder_id, source)
		 VALUES ($1::uuid, $2::uuid, 'ph_title', 'user')`,
		tenantB, docID,
	)
	if err == nil {
		t.Fatal("expected tenant mismatch error from trigger, got nil")
	}
	if !strings.Contains(err.Error(), "tenant mismatch") &&
		!strings.Contains(err.Error(), "check_violation") &&
		!strings.Contains(err.Error(), "23514") {
		t.Fatalf("unexpected error (wanted tenant mismatch): %v", err)
	}
}

// TestMigration0153_TenantMatchAccepted verifies that inserting with the
// correct tenant_id succeeds.
func TestMigration0153_TenantMatchAccepted(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tenantA := testdb.DeterministicID(t, "tenant-a")
	docID := testdb.DeterministicID(t, "doc")
	userID := testdb.DeterministicID(t, "user")

	fixtures.SeedUser(t, ctx, db, schema, userID, "Tenant Match User")
	fixtures.SeedDocument(t, ctx, db, schema, docID, tenantA, userID)

	_, err := db.ExecContext(ctx,
		`INSERT INTO `+testdb.Qualified(schema, "document_placeholder_values")+`
		 (tenant_id, revision_id, placeholder_id, source)
		 VALUES ($1::uuid, $2::uuid, 'ph_body', 'user')`,
		tenantA, docID,
	)
	if err != nil {
		t.Fatalf("insert with matching tenant should succeed: %v", err)
	}
}

// TestMigration0153_ZoneContentTenantMismatchRejected verifies the same
// trigger fires on document_editable_zone_content.
func TestMigration0153_ZoneContentTenantMismatchRejected(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tenantA := testdb.DeterministicID(t, "tenant-a")
	tenantB := testdb.DeterministicID(t, "tenant-b")
	docID := testdb.DeterministicID(t, "doc")
	userID := testdb.DeterministicID(t, "user")

	fixtures.SeedUser(t, ctx, db, schema, userID, "Zone Mismatch User")
	fixtures.SeedDocument(t, ctx, db, schema, docID, tenantA, userID)

	_, err := db.ExecContext(ctx,
		`INSERT INTO `+testdb.Qualified(schema, "document_editable_zone_content")+`
		 (tenant_id, revision_id, zone_id, content_ooxml, content_hash)
		 VALUES ($1::uuid, $2::uuid, 'zone_1', '<w:p/>', '\x0102')`,
		tenantB, docID,
	)
	if err == nil {
		t.Fatal("expected tenant mismatch error on zone content insert, got nil")
	}
	if !strings.Contains(err.Error(), "tenant mismatch") &&
		!strings.Contains(err.Error(), "check_violation") &&
		!strings.Contains(err.Error(), "23514") {
		t.Fatalf("unexpected error (wanted tenant mismatch): %v", err)
	}
}
