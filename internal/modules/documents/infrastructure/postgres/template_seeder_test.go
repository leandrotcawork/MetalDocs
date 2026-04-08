package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain/mddm"
)

func TestTemplateSeeder_IsIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}

	ctx := context.Background()
	db := newTestDB(t)
	t.Cleanup(func() { _ = db.Close() })

	templateID := newTestTemplateID(t, ctx, db)
	seeder := NewTemplateSeeder(db)
	expectedContent, expectedHash := expectedCanonicalTemplateSeed(t)

	seedStaleTemplateRow(t, ctx, db, templateID)

	if err := seeder.SeedPOTemplate(ctx, templateID); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := seeder.SeedPOTemplate(ctx, templateID); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	finalCount, contentBytes, storedHash := loadTemplateSeedState(t, ctx, db, templateID)
	if finalCount != 1 {
		t.Fatalf("expected exactly one seeded row after repeated seed calls, got %d", finalCount)
	}
	if storedHash != expectedHash {
		t.Fatalf("stored hash = %q, want %q", storedHash, expectedHash)
	}
	if !jsonEqualCanonicalBytes(t, contentBytes, expectedContent) {
		t.Fatalf("stored canonical content does not match expected canonical template")
	}
}

func TestTemplateSeeder_PreservesPublishedState(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}

	ctx := context.Background()
	db := newTestDB(t)
	t.Cleanup(func() { _ = db.Close() })

	templateID := newTestTemplateID(t, ctx, db)
	seeder := NewTemplateSeeder(db)

	if err := seeder.SeedPOTemplate(ctx, templateID); err != nil {
		t.Fatalf("initial seed: %v", err)
	}
	if !loadTemplatePublishedState(t, ctx, db, templateID) {
		t.Fatal("expected initial canonical seed to be published")
	}

	if err := seeder.SeedPOTemplate(ctx, templateID); err != nil {
		t.Fatalf("repeat seed: %v", err)
	}
	if !loadTemplatePublishedState(t, ctx, db, templateID) {
		t.Fatal("expected seeded template row to be published")
	}
}

func TestTemplateSeeder_RejectsEmptyCanonicalBlocks(t *testing.T) {
	ctx := context.Background()
	seeder := NewTemplateSeeder(nil)

	err := seeder.seedTemplateVersion(ctx, uuid.New(), map[string]any{
		"mddm_version": 1,
		"template_ref": nil,
		"blocks":       []any{},
	})
	if err == nil {
		t.Fatal("expected empty canonical blocks to fail")
	}
}

func newTestTemplateID(t *testing.T, ctx context.Context, db *sql.DB) uuid.UUID {
	t.Helper()

	templateID := uuid.New()
	t.Cleanup(func() {
		if _, err := db.ExecContext(ctx, `
			DELETE FROM metaldocs.document_template_versions_mddm
			WHERE template_id = $1
		`, templateID); err != nil {
			t.Fatalf("cleanup template seed row: %v", err)
		}
	})
	return templateID
}

func loadTemplateSeedState(t *testing.T, ctx context.Context, db *sql.DB, templateID uuid.UUID) (int, []byte, string) {
	t.Helper()

	var count int
	var content sql.NullString
	var hash sql.NullString
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(MAX(content_blocks::text), ''), COALESCE(MAX(content_hash), '')
		FROM metaldocs.document_template_versions_mddm
		WHERE template_id = $1 AND version = 1
	`, templateID).Scan(&count, &content, &hash); err != nil {
		t.Fatalf("load template seed state: %v", err)
	}
	return count, []byte(content.String), hash.String
}

func loadTemplatePublishedState(t *testing.T, ctx context.Context, db *sql.DB, templateID uuid.UUID) bool {
	t.Helper()

	var published bool
	if err := db.QueryRowContext(ctx, `
		SELECT is_published
		FROM metaldocs.document_template_versions_mddm
		WHERE template_id = $1 AND version = 1
	`, templateID).Scan(&published); err != nil {
		t.Fatalf("load template published state: %v", err)
	}
	return published
}

func seedStaleTemplateRow(t *testing.T, ctx context.Context, db *sql.DB, templateID uuid.UUID) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.document_template_versions_mddm
		  (template_id, version, mddm_version, content_blocks, content_hash, is_published)
		VALUES ($1, 1, 1, $2::jsonb, $3, false)
		ON CONFLICT (template_id, version) DO UPDATE
		SET mddm_version = EXCLUDED.mddm_version,
		    content_blocks = EXCLUDED.content_blocks,
		    content_hash = EXCLUDED.content_hash,
		    is_published = EXCLUDED.is_published
	`, templateID, `{"mddm_version":1,"template_ref":null,"blocks":[]}`, "stale-hash"); err != nil {
		t.Fatalf("seed stale template row: %v", err)
	}
}

func expectedCanonicalTemplateSeed(t *testing.T) ([]byte, string) {
	t.Helper()

	envelope, err := normalizeTemplateEnvelope(mddm.POTemplateMDDM())
	if err != nil {
		t.Fatalf("normalize template: %v", err)
	}
	canonicalEnvelope, err := mddm.CanonicalizeMDDM(envelope)
	if err != nil {
		t.Fatalf("canonicalize template: %v", err)
	}
	canonicalBytes, err := mddm.MarshalCanonical(canonicalEnvelope)
	if err != nil {
		t.Fatalf("marshal canonical template: %v", err)
	}
	expectedHashBytes := sha256.Sum256(canonicalBytes)
	return canonicalBytes, hex.EncodeToString(expectedHashBytes[:])
}

func jsonEqualCanonicalBytes(t *testing.T, got []byte, want []byte) bool {
	t.Helper()

	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("unmarshal stored canonical content: %v", err)
	}

	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("unmarshal expected canonical content: %v", err)
	}

	return jsonEqualValues(gotValue, wantValue)
}

func jsonEqualValues(a any, b any) bool {
	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)
	return string(aBytes) == string(bBytes)
}
