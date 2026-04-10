package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"metaldocs/internal/modules/documents/domain/mddm"
)

func TestPostgresByteaStorage_PutGetExists(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()
	db := newTestDB(t)
	defer db.Close()

	store := NewPostgresByteaStorage(db)

	bytes := []byte("hello world image bytes")
	sum := sha256.Sum256(bytes)
	hash := hex.EncodeToString(sum[:])

	// Clean up any previous leftovers for this hash before testing
	_, _ = db.ExecContext(ctx, `DELETE FROM metaldocs.document_images WHERE sha256 = $1`, hash)

	// First put
	id1, err := store.Put(ctx, hash, "image/png", bytes)
	if err != nil {
		t.Fatal(err)
	}

	// Same content put again — should return same id
	id2, err := store.Put(ctx, hash, "image/png", bytes)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Errorf("expected dedup, got different ids: %s vs %s", id1, id2)
	}

	// Get
	gotBytes, gotMime, err := store.Get(ctx, id1)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBytes) != string(bytes) {
		t.Errorf("bytes mismatch")
	}
	if gotMime != "image/png" {
		t.Errorf("mime mismatch: %s", gotMime)
	}

	// Exists
	existsID, exists, err := store.Exists(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !exists || existsID != id1 {
		t.Errorf("Exists should return id1")
	}

	// Delete
	if err := store.Delete(ctx, id1); err != nil {
		t.Fatal(err)
	}

	// Get after delete should error
	if _, _, err := store.Get(ctx, id1); err != mddm.ErrImageNotFound {
		t.Errorf("expected ErrImageNotFound, got %v", err)
	}
}
