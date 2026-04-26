//go:build integration
// +build integration

package application_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"metaldocs/internal/modules/documents_v2/application"
	docrepo "metaldocs/internal/modules/documents_v2/repository"
	registrydomain "metaldocs/internal/modules/registry/domain"
	"metaldocs/internal/platform/docgenv2"
	"metaldocs/tests/integration/testdb"
)

const createSnapshotTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

func TestCreateDocument_PopulatesAllSnapshotColumns(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)
	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(ctx, fmt.Sprintf(`SET search_path TO %q`, schema)); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	tenantID := createSnapshotTenantID
	actorID := testdb.DeterministicID(t, "actor")
	templateID := testdb.DeterministicID(t, "template")
	templateVersionID := testdb.DeterministicID(t, "template-version")
	controlledDocumentID := testdb.DeterministicID(t, "controlled-document")

	seedCreateDocumentSnapshotRows(t, ctx, db, tenantID, actorID, templateID, templateVersionID, controlledDocumentID)

	cd := &registrydomain.ControlledDocument{
		ID:              controlledDocumentID,
		TenantID:        tenantID,
		ProfileCode:     "po",
		ProcessAreaCode: "quality",
		Code:            "PO-TEST-001",
		Title:           "Snapshot Test Controlled Document",
		OwnerUserID:     actorID,
		Status:          registrydomain.CDStatusActive,
	}

	snapshotSvc := application.NewSnapshotService(
		docgenv2.NewTemplatesV2SnapshotReader(db),
		docrepo.NewSnapshotRepository(db),
	)
	svc := application.NewServiceWithSnapshot(
		docrepo.New(db),
		nil,
		nil,
		docgenv2.NewTemplatesV2TemplateReader(db),
		fakeFormVal{valid: true},
		&noopAudit{},
		&fakeRegistryReader{cd: cd},
		&fakeAuthzChecker{},
		&fakeProfileDefaultTemplateReader{id: strptr(templateVersionID), status: strptr("published")},
		snapshotSvc,
	)

	res, err := svc.CreateDocument(ctx, application.CreateDocumentInput{
		TenantID:             tenantID,
		ActorUserID:          actorID,
		ControlledDocumentID: controlledDocumentID,
		Name:                 "Snapshot Integration Test",
		FormData:             json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}

	var (
		placeholderSchemaSnapshot sql.NullString
		placeholderSchemaHash     []byte
		compositionConfigSnapshot sql.NullString
		compositionConfigHash     []byte
		bodyDocxSnapshotS3Key     sql.NullString
		bodyDocxHash              []byte
		profileCodeSnapshot       sql.NullString
		processAreaCodeSnapshot   sql.NullString
	)
	if err := db.QueryRowContext(ctx, `
		SELECT placeholder_schema_snapshot::text,
		       placeholder_schema_hash,
		       composition_config_snapshot::text,
		       composition_config_hash,
		       body_docx_snapshot_s3_key,
		       body_docx_hash,
		       profile_code_snapshot,
		       process_area_code_snapshot
		  FROM documents
		 WHERE id = $1::uuid`,
		res.DocumentID,
	).Scan(
		&placeholderSchemaSnapshot,
		&placeholderSchemaHash,
		&compositionConfigSnapshot,
		&compositionConfigHash,
		&bodyDocxSnapshotS3Key,
		&bodyDocxHash,
		&profileCodeSnapshot,
		&processAreaCodeSnapshot,
	); err != nil {
		t.Fatalf("query snapshot columns: %v", err)
	}

	assertNotNullString(t, "placeholder_schema_snapshot", placeholderSchemaSnapshot)
	assertNotNullBytes(t, "placeholder_schema_hash", placeholderSchemaHash)
	assertNotNullString(t, "composition_config_snapshot", compositionConfigSnapshot)
	assertNotNullBytes(t, "composition_config_hash", compositionConfigHash)
	assertNotNullString(t, "body_docx_snapshot_s3_key", bodyDocxSnapshotS3Key)
	assertNotNullBytes(t, "body_docx_hash", bodyDocxHash)
	assertNotNullString(t, "profile_code_snapshot", profileCodeSnapshot)
	assertNotNullString(t, "process_area_code_snapshot", processAreaCodeSnapshot)
}

func seedCreateDocumentSnapshotRows(t *testing.T, ctx context.Context, db *sql.DB, tenantID, actorID, templateID, templateVersionID, controlledDocumentID string) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO templates_v2_template (
			id, tenant_id, doc_type_code, key, name, visibility, latest_version, published_version_id, created_by
		) VALUES (
			$1::uuid, $2, 'po', 'snapshot-integration-template', 'Snapshot Integration Template',
			'public', 1, NULL, $3
		)`,
		templateID, tenantID, actorID,
	); err != nil {
		t.Fatalf("seed templates_v2_template: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO templates_v2_template_version (
			id, template_id, version_number, status, docx_storage_key, content_hash,
			metadata_schema, placeholder_schema, author_id, published_at
		) VALUES (
			$1::uuid, $2::uuid, 1, 'published', 'templates/snapshot/body.docx', 'body-hash',
			'{}'::jsonb, '{"placeholders":[]}'::jsonb, $3, now()
		)`,
		templateVersionID, templateID, actorID,
	); err != nil {
		t.Fatalf("seed templates_v2_template_version: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE templates_v2_template
		   SET published_version_id = $1::uuid
		 WHERE id = $2::uuid`,
		templateVersionID, templateID,
	); err != nil {
		t.Fatalf("seed template published_version_id: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO controlled_documents (
			id, tenant_id, profile_code, process_area_code, code, title, owner_user_id, status
		) VALUES (
			$1::uuid, $2::uuid, 'po', 'quality', 'PO-TEST-001',
			'Snapshot Test Controlled Document', $3, 'active'
		)`,
		controlledDocumentID, tenantID, actorID,
	); err != nil {
		t.Fatalf("seed controlled_documents: %v", err)
	}
}

func assertNotNullString(t *testing.T, name string, got sql.NullString) {
	t.Helper()
	if !got.Valid {
		t.Fatalf("%s is NULL", name)
	}
}

func assertNotNullBytes(t *testing.T, name string, got []byte) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is NULL", name)
	}
}
