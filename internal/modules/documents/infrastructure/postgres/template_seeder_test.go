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
	defer db.Close()

	templateID := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	initialCount, initialHash := loadTemplateSeedState(t, ctx, db, templateID)
	if initialCount > 1 {
		t.Fatalf("expected at most one existing seeded row for template %s, got %d", templateID, initialCount)
	}

	seeder := NewTemplateSeeder(db)

	if err := seeder.SeedPOTemplate(ctx, templateID); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := seeder.SeedPOTemplate(ctx, templateID); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	finalCount, finalHash := loadTemplateSeedState(t, ctx, db, templateID)
	if finalCount != 1 {
		t.Fatalf("expected exactly one seeded row after repeated seed calls, got %d", finalCount)
	}
	if initialCount == 1 {
		if finalHash != initialHash {
			t.Fatalf("expected existing seed hash to remain unchanged, got %q want %q", finalHash, initialHash)
		}
		return
	}

	envelope := mddm.POTemplateMDDM()
	canonicalEnvelope, err := mddm.CanonicalizeMDDM(envelope)
	if err != nil {
		t.Fatalf("canonicalize template: %v", err)
	}
	canonicalBytes, err := mddm.MarshalCanonical(canonicalEnvelope)
	if err != nil {
		t.Fatalf("marshal canonical template: %v", err)
	}
	expectedHashBytes := sha256.Sum256(canonicalBytes)
	expectedHash := hex.EncodeToString(expectedHashBytes[:])

	var contentBytes []byte
	var storedHash string
	if err := db.QueryRowContext(ctx, `
		SELECT content_blocks::text, content_hash
		FROM metaldocs.document_template_versions_mddm
		WHERE template_id = $1 AND version = 1
	`, templateID).Scan(&contentBytes, &storedHash); err != nil {
		t.Fatalf("load seeded template row: %v", err)
	}

	if storedHash != expectedHash {
		t.Fatalf("stored hash = %q, want %q", storedHash, expectedHash)
	}
	if !jsonEqualCanonicalBytes(t, contentBytes, canonicalBytes) {
		t.Fatalf("stored canonical content does not match expected canonical template")
	}
}

func loadTemplateSeedState(t *testing.T, ctx context.Context, db *sql.DB, templateID uuid.UUID) (int, string) {
	t.Helper()

	var count int
	var hash sql.NullString
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(MAX(content_hash), '')
		FROM metaldocs.document_template_versions_mddm
		WHERE template_id = $1 AND version = 1
	`, templateID).Scan(&count, &hash); err != nil {
		t.Fatalf("load template seed state: %v", err)
	}
	return count, hash.String
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
