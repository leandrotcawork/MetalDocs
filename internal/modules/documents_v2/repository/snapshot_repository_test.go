//go:build integration
// +build integration

package repository_test

import (
	"context"
	"testing"

	"metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/tests/integration/testdb"
)

const snapshotTestTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

func TestSnapshotRepository_WriteAndRead(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	docID, tenant := testdb.InsertDraftDocument(t, db, schema, snapshotTestTenantID)

	repo := repository.NewSnapshotRepositoryWithSchema(db, schema)
	snap := domain.TemplateSnapshot{
		PlaceholderSchemaJSON: []byte(`{}`),
		CompositionJSON:       []byte(`{}`),
		ZonesSchemaJSON:       []byte(`{}`),
		BodyDocxBytes:         []byte("x"),
		BodyDocxS3Key:         "s3://bucket/key",
	}

	if err := repo.WriteSnapshot(ctx, tenant, docID, snap); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	got, err := repo.ReadSnapshot(ctx, tenant, docID)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if got.BodyDocxS3Key != "s3://bucket/key" {
		t.Fatalf("got BodyDocxS3Key = %q, want s3://bucket/key", got.BodyDocxS3Key)
	}
	if string(got.PlaceholderSchemaJSON) != `{}` {
		t.Fatalf("got PlaceholderSchemaJSON = %q, want {}", got.PlaceholderSchemaJSON)
	}
}
