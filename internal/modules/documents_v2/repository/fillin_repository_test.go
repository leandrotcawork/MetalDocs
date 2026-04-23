//go:build integration
// +build integration

package repository_test

import (
	"context"
	"testing"

	"metaldocs/internal/modules/documents_v2/repository"
	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
	"metaldocs/tests/integration/testdb"
)

const fillInTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

func TestFillInRepository_SeedDefaults_RequiredOnly(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	docID, tenant := testdb.InsertDraftDocument(t, db, schema, fillInTenantID)

	// Get revisionID from document.
	var revID string
	if err := db.QueryRowContext(ctx,
		`SELECT current_revision_id::text FROM `+testdb.Qualified(schema, "documents")+` WHERE id=$1::uuid`,
		docID,
	).Scan(&revID); err != nil {
		t.Fatalf("get revision: %v", err)
	}

	placeholders := []templatesdomain.Placeholder{
		{ID: "ph-required", Label: "Required Field", Type: templatesdomain.PHText, Required: true},
		{ID: "ph-optional", Label: "Optional Field", Type: templatesdomain.PHText, Required: false},
	}

	repo := repository.NewFillInRepositoryWithSchema(db, schema)
	if err := repo.SeedDefaults(ctx, tenant, revID, placeholders); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}

	// Assert: exactly one row for the required placeholder, none for optional.
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM `+testdb.Qualified(schema, "document_placeholder_values")+
			` WHERE tenant_id=$1::uuid AND revision_id=$2::uuid`,
		tenant, revID,
	).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row for required placeholder, got %d", count)
	}

	// Assert source = 'default'.
	var source string
	if err := db.QueryRowContext(ctx,
		`SELECT source FROM `+testdb.Qualified(schema, "document_placeholder_values")+
			` WHERE tenant_id=$1::uuid AND revision_id=$2::uuid AND placeholder_id=$3`,
		tenant, revID, "ph-required",
	).Scan(&source); err != nil {
		t.Fatalf("get source: %v", err)
	}
	if source != "default" {
		t.Fatalf("expected source=default, got %q", source)
	}

	// Idempotency: calling again should not fail or create duplicates.
	if err := repo.SeedDefaults(ctx, tenant, revID, placeholders); err != nil {
		t.Fatalf("SeedDefaults idempotent call: %v", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM `+testdb.Qualified(schema, "document_placeholder_values")+
			` WHERE tenant_id=$1::uuid AND revision_id=$2::uuid`,
		tenant, revID,
	).Scan(&count); err != nil {
		t.Fatalf("count rows after idempotent: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after idempotent call, got %d", count)
	}
}
