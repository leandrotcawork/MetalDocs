//go:build integration
// +build integration

package application_test

import (
	"context"
	"testing"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/tests/integration/testdb"
)

const snapshotSvcTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

// fakeTemplateReader implements application.SnapshotTemplateReader.
type fakeTemplateReader struct {
	snap domain.TemplateSnapshot
}

func (f fakeTemplateReader) LoadForSnapshot(_ context.Context, _, _ string) (domain.TemplateSnapshot, error) {
	return f.snap, nil
}

func TestSnapshotService_CopiesTemplateToRevision(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	docID, tenant := testdb.InsertDraftDocument(t, db, schema, snapshotSvcTenantID)

	tmpl := domain.TemplateSnapshot{
		PlaceholderSchemaJSON: []byte(`{"placeholders":[]}`),
		CompositionJSON:       []byte(`{"header_sub_blocks":[]}`),
		BodyDocxBytes:         []byte("DOCX"),
		BodyDocxS3Key:         "s3://t/k",
	}

	repo := repository.NewSnapshotRepositoryWithSchema(db, schema)
	svc := application.NewSnapshotService(fakeTemplateReader{tmpl}, repo)

	if err := svc.SnapshotFromTemplate(ctx, tenant, docID, docID, "tmpl-1"); err != nil {
		t.Fatalf("SnapshotFromTemplate: %v", err)
	}

	got, err := repository.NewSnapshotRepositoryWithSchema(db, schema).ReadSnapshot(ctx, tenant, docID)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if string(got.PlaceholderSchemaJSON) != `{"placeholders":[]}` {
		t.Fatalf("mismatch: got %q", got.PlaceholderSchemaJSON)
	}
}
