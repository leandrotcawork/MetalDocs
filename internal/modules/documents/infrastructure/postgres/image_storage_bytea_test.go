package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"metaldocs/internal/modules/documents/domain/mddm"
)

// newTestDB opens a connection to the dev Postgres container for integration tests.
// Reads TEST_DATABASE_URL from env; falls back to the default dev-container DSN.
// Callers must close the returned *sql.DB.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://metaldocs_app:Lepa12%3C%3E%21@localhost:5433/metaldocs?sslmode=disable&search_path=metaldocs"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("dev Postgres not reachable at %s: %v", dsn, err)
	}
	return db
}

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
