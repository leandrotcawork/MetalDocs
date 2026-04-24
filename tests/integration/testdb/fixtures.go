//go:build integration
// +build integration

package testdb

import (
	"context"
	"database/sql"
	"testing"
)

// InsertDraftDocument inserts a minimal draft document with an initial revision
// into the test schema. Returns (docID, tenantID).
// tenantID must be a valid UUID string.
func InsertDraftDocument(t *testing.T, db *sql.DB, schema, tenantID string) (docID, tenant string) {
	t.Helper()
	ctx := context.Background()

	userID := DeterministicID(t, "user")

	// Insert minimal stub template.
	var tplID string
	if err := db.QueryRowContext(ctx,
		`INSERT INTO `+Qualified(schema, "templates")+
			` (tenant_id, key, name, created_by)
		 VALUES ($1::uuid, 'test-template', 'Test Template', $2::uuid)
		 RETURNING id::text`,
		tenantID, userID,
	).Scan(&tplID); err != nil {
		t.Fatalf("InsertDraftDocument: insert template: %v", err)
	}

	// Insert minimal published template version.
	var tvID string
	if err := db.QueryRowContext(ctx,
		`INSERT INTO `+Qualified(schema, "template_versions")+
			` (template_id, version_num, status, docx_storage_key, schema_storage_key,
			   docx_content_hash, schema_content_hash, created_by)
		 VALUES ($1::uuid, 1, 'published', 'key/tpl.docx', 'key/schema.json',
			   'aabbcc', 'ddeeff', $2::uuid)
		 RETURNING id::text`,
		tplID, userID,
	).Scan(&tvID); err != nil {
		t.Fatalf("InsertDraftDocument: insert template_version: %v", err)
	}

	// Insert document.
	if err := db.QueryRowContext(ctx,
		`INSERT INTO `+Qualified(schema, "documents")+
			` (tenant_id, template_version_id, name, status, form_data_json, created_by)
		 VALUES ($1::uuid, $2::uuid, 'Test Doc', 'draft', '{}', $3::uuid)
		 RETURNING id::text`,
		tenantID, tvID, userID,
	).Scan(&docID); err != nil {
		t.Fatalf("InsertDraftDocument: insert document: %v", err)
	}

	// Insert editor session.
	var sessID string
	if err := db.QueryRowContext(ctx,
		`INSERT INTO `+Qualified(schema, "editor_sessions")+
			` (document_id, user_id, expires_at, last_acknowledged_revision_id, status)
		 VALUES ($1::uuid, $2::uuid, now() + interval '1 hour',
		         '00000000-0000-0000-0000-000000000000', 'active')
		 RETURNING id::text`,
		docID, userID,
	).Scan(&sessID); err != nil {
		t.Fatalf("InsertDraftDocument: insert session: %v", err)
	}

	// Insert initial revision.
	var revID string
	if err := db.QueryRowContext(ctx,
		`INSERT INTO `+Qualified(schema, "document_revisions")+
			` (document_id, parent_revision_id, session_id, storage_key, content_hash, form_data_snapshot)
		 VALUES ($1::uuid, NULL, $2::uuid, '', 'aabbcc', '{}')
		 RETURNING id::text`,
		docID, sessID,
	).Scan(&revID); err != nil {
		t.Fatalf("InsertDraftDocument: insert revision: %v", err)
	}

	// Update document pointers.
	if _, err := db.ExecContext(ctx,
		`UPDATE `+Qualified(schema, "documents")+
			` SET current_revision_id=$1::uuid, active_session_id=$2::uuid, updated_at=now()
		 WHERE id=$3::uuid`,
		revID, sessID, docID,
	); err != nil {
		t.Fatalf("InsertDraftDocument: update document pointers: %v", err)
	}

	// Update session ack.
	if _, err := db.ExecContext(ctx,
		`UPDATE `+Qualified(schema, "editor_sessions")+
			` SET last_acknowledged_revision_id=$1::uuid WHERE id=$2::uuid`,
		revID, sessID,
	); err != nil {
		t.Fatalf("InsertDraftDocument: update session ack: %v", err)
	}

	return docID, tenantID
}
