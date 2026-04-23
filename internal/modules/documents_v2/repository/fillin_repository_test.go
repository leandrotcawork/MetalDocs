//go:build integration
// +build integration

package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestFillInRepository_UpsertValueAndListValues(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	docID, tenant := testdb.InsertDraftDocument(t, db, schema, fillInTenantID)

	var revID string
	if err := db.QueryRowContext(ctx,
		`SELECT current_revision_id::text FROM `+testdb.Qualified(schema, "documents")+` WHERE id=$1::uuid`,
		docID,
	).Scan(&revID); err != nil {
		t.Fatalf("get revision: %v", err)
	}

	repo := repository.NewFillInRepositoryWithSchema(db, schema)

	v1 := repository.PlaceholderValue{
		TenantID:      tenant,
		RevisionID:    revID,
		PlaceholderID: "ph-1",
		ValueText:     strPtr("A"),
		ValueTyped:    map[string]any{"raw": "A"},
		Source:        "user",
	}
	if err := repo.UpsertValue(ctx, v1); err != nil {
		t.Fatalf("upsert first: %v", err)
	}

	var createdAt, updatedAt time.Time
	if err := db.QueryRowContext(ctx,
		`SELECT created_at, updated_at FROM `+testdb.Qualified(schema, "document_placeholder_values")+`
		  WHERE tenant_id=$1::uuid AND revision_id=$2::uuid AND placeholder_id=$3`,
		tenant, revID, "ph-1",
	).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("timestamps first: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	v2 := repository.PlaceholderValue{
		TenantID:      tenant,
		RevisionID:    revID,
		PlaceholderID: "ph-1",
		ValueText:     strPtr("B"),
		ValueTyped:    map[string]any{"raw": "B", "n": float64(2)},
		Source:        "user",
	}
	if err := repo.UpsertValue(ctx, v2); err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	var createdAt2, updatedAt2 time.Time
	var typedJSON []byte
	if err := db.QueryRowContext(ctx,
		`SELECT created_at, updated_at, value_typed FROM `+testdb.Qualified(schema, "document_placeholder_values")+`
		  WHERE tenant_id=$1::uuid AND revision_id=$2::uuid AND placeholder_id=$3`,
		tenant, revID, "ph-1",
	).Scan(&createdAt2, &updatedAt2, &typedJSON); err != nil {
		t.Fatalf("timestamps second: %v", err)
	}
	if !createdAt2.Equal(createdAt) {
		t.Fatalf("created_at changed: first=%v second=%v", createdAt, createdAt2)
	}
	if !updatedAt2.After(updatedAt) {
		t.Fatalf("updated_at did not advance: first=%v second=%v", updatedAt, updatedAt2)
	}

	values, err := repo.ListValues(ctx, tenant, revID)
	if err != nil {
		t.Fatalf("list values: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if values[0].PlaceholderID != "ph-1" {
		t.Fatalf("placeholder_id = %q, want ph-1", values[0].PlaceholderID)
	}
	if values[0].ValueText == nil || *values[0].ValueText != "B" {
		t.Fatalf("value_text = %v, want B", values[0].ValueText)
	}
	if values[0].Source != "user" {
		t.Fatalf("source = %q, want user", values[0].Source)
	}

	var typed map[string]any
	if err := json.Unmarshal(typedJSON, &typed); err != nil {
		t.Fatalf("unmarshal typed json: %v", err)
	}
	if typed["raw"] != "B" {
		t.Fatalf("typed raw = %v, want B", typed["raw"])
	}
}

func strPtr(v string) *string { return &v }

func TestFillInRepository_UpsertZoneContentAndList(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	docID, tenant := testdb.InsertDraftDocument(t, db, schema, fillInTenantID)

	var revID string
	if err := db.QueryRowContext(ctx,
		`SELECT current_revision_id::text FROM `+testdb.Qualified(schema, "documents")+` WHERE id=$1::uuid`,
		docID,
	).Scan(&revID); err != nil {
		t.Fatalf("get revision: %v", err)
	}

	repo := repository.NewFillInRepositoryWithSchema(db, schema)
	if err := repo.UpsertZoneContent(ctx, repository.ZoneContent{
		TenantID:     tenant,
		RevisionID:   revID,
		ZoneID:       "zone-1",
		ContentOOXML: "<w:p>first</w:p>",
	}); err != nil {
		t.Fatalf("upsert first: %v", err)
	}

	var firstHash []byte
	if err := db.QueryRowContext(ctx,
		`SELECT content_hash FROM `+testdb.Qualified(schema, "document_editable_zone_content")+`
		  WHERE tenant_id=$1::uuid AND revision_id=$2::uuid AND zone_id=$3`,
		tenant, revID, "zone-1",
	).Scan(&firstHash); err != nil {
		t.Fatalf("select first hash: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	if err := repo.UpsertZoneContent(ctx, repository.ZoneContent{
		TenantID:     tenant,
		RevisionID:   revID,
		ZoneID:       "zone-1",
		ContentOOXML: "<w:p>second</w:p>",
	}); err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	var secondHash []byte
	if err := db.QueryRowContext(ctx,
		`SELECT content_hash FROM `+testdb.Qualified(schema, "document_editable_zone_content")+`
		  WHERE tenant_id=$1::uuid AND revision_id=$2::uuid AND zone_id=$3`,
		tenant, revID, "zone-1",
	).Scan(&secondHash); err != nil {
		t.Fatalf("select second hash: %v", err)
	}
	if string(firstHash) == string(secondHash) {
		t.Fatalf("content_hash did not change")
	}

	zones, err := repo.ListZoneContent(ctx, tenant, revID)
	if err != nil {
		t.Fatalf("list zone content: %v", err)
	}
	if len(zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(zones))
	}
	if zones[0].ZoneID != "zone-1" {
		t.Fatalf("zone_id = %q, want zone-1", zones[0].ZoneID)
	}
	if zones[0].ContentOOXML != "<w:p>second</w:p>" {
		t.Fatalf("content_ooxml = %q", zones[0].ContentOOXML)
	}
	if string(zones[0].ContentHash) != string(secondHash) {
		t.Fatalf("list hash mismatch")
	}
}
