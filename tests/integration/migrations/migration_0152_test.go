//go:build integration

package migrations_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"metaldocs/tests/integration/fixtures"
	"metaldocs/tests/integration/testdb"
)

// TestMigration0152_Columns verifies the snapshot columns were added to documents.
func TestMigration0152_Columns(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	wantCols := []string{
		"placeholder_schema_snapshot",
		"placeholder_schema_hash",
		"composition_config_snapshot",
		"composition_config_hash",
		"editable_zones_schema_snapshot",
		"body_docx_snapshot_s3_key",
		"body_docx_hash",
		"values_frozen_at",
		"values_hash",
		"final_docx_s3_key",
		"final_pdf_s3_key",
		"pdf_hash",
		"pdf_generated_at",
		"reconstruction_attempts",
	}

	for _, col := range wantCols {
		var found bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = $1
				  AND table_name   = 'documents'
				  AND column_name  = $2
			)`, schema, col).Scan(&found)
		if err != nil {
			t.Fatalf("query column %s: %v", col, err)
		}
		if !found {
			t.Errorf("documents column %q not found in schema %q", col, schema)
		}
	}
}

// TestMigration0152_Tables verifies document_placeholder_values and
// document_editable_zone_content tables exist.
func TestMigration0152_Tables(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	for _, tbl := range []string{"document_placeholder_values", "document_editable_zone_content"} {
		var found bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = $1
				  AND table_name   = $2
			)`, schema, tbl).Scan(&found)
		if err != nil {
			t.Fatalf("query table %s: %v", tbl, err)
		}
		if !found {
			t.Errorf("table %q not found in schema %q", tbl, schema)
		}
	}
}

// TestMigration0152_SnapshotTrigger verifies enforce_snapshot_on_submit blocks
// transitions to guarded statuses (under_review, approved, scheduled, published)
// when snapshot columns are NULL.
func TestMigration0152_SnapshotTrigger(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tenantID := testdb.DeterministicID(t, "tenant")
	userID := testdb.DeterministicID(t, "user")

	fixtures.SeedUser(t, ctx, db, schema, userID, "Snapshot Test User")

	guardedStatuses := []string{"under_review", "approved", "scheduled", "published"}

	for _, status := range guardedStatuses {
		t.Run(status, func(t *testing.T) {
			docID := testdb.DeterministicID(t, fmt.Sprintf("doc-%s", status))
			fixtures.SeedDocument(t, ctx, db, schema, docID, tenantID, userID)

			_, err := db.ExecContext(ctx, fmt.Sprintf(`
				UPDATE %s SET status = $2
				 WHERE id = $1::uuid`,
				testdb.Qualified(schema, "documents")),
				docID,
				status,
			)
			if err == nil {
				t.Fatalf("expected error from enforce_snapshot_on_submit trigger for status=%s, got nil", status)
			}
			if !strings.Contains(err.Error(), "snapshot columns required") &&
				!strings.Contains(err.Error(), "check_violation") &&
				!strings.Contains(err.Error(), "23514") {
				t.Fatalf("unexpected error for status=%s (wanted snapshot enforcement): %v", status, err)
			}
		})
	}
}

// TestMigration0152_PlaceholderValueInsert verifies a row can be inserted into
// document_placeholder_values when revision_id references a valid document.
func TestMigration0152_PlaceholderValueInsert(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tenantID := testdb.DeterministicID(t, "tenant")
	docID := testdb.DeterministicID(t, "doc")
	userID := testdb.DeterministicID(t, "user")

	fixtures.SeedUser(t, ctx, db, schema, userID, "PV Insert User")
	fixtures.SeedDocument(t, ctx, db, schema, docID, tenantID, userID)

	_, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (tenant_id, revision_id, placeholder_id, source)
		VALUES ($1::uuid, $2::uuid, 'ph_title', 'user')`,
		testdb.Qualified(schema, "document_placeholder_values")),
		tenantID, docID,
	)
	if err != nil {
		t.Fatalf("insert document_placeholder_values: %v", err)
	}
}
