//go:build integration
// +build integration

package repository_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

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

func TestSnapshotRepository_WriteFreeze_PersistsHashAndFrozenAt(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	docID, tenant := testdb.InsertDraftDocument(t, db, schema, snapshotTestTenantID)
	repo := repository.NewSnapshotRepositoryWithSchema(db, schema)

	hash, err := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("decode hash: %v", err)
	}
	frozenAt := time.Date(2026, 4, 23, 18, 0, 0, 0, time.UTC)

	if err := repo.WriteFreeze(ctx, tenant, docID, hash, frozenAt); err != nil {
		t.Fatalf("WriteFreeze: %v", err)
	}

	var gotHash []byte
	var gotFrozenAt *time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT values_hash, values_frozen_at
		  FROM `+`"`+schema+`"`+`.documents
		 WHERE tenant_id=$1::uuid AND id=$2::uuid`,
		tenant, docID,
	).Scan(&gotHash, &gotFrozenAt); err != nil {
		t.Fatalf("read freeze columns: %v", err)
	}
	if hex.EncodeToString(gotHash) != hex.EncodeToString(hash) {
		t.Fatalf("values_hash mismatch: got %x want %x", gotHash, hash)
	}
	if gotFrozenAt == nil || !gotFrozenAt.Equal(frozenAt) {
		t.Fatalf("values_frozen_at mismatch: got %v want %v", gotFrozenAt, frozenAt)
	}
}
