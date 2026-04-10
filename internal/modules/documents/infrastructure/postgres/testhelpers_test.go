package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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

var testDocumentSeq atomic.Int64

// newTestDocument inserts a new test document row and returns its id.
// Each call produces unique id + document_code + (profile, sequence) tuple.
// Rows are cleaned up via ON DELETE CASCADE when the document is removed
// — callers that need cleanup should DELETE the document explicitly.
func newTestDocument(t *testing.T, db *sql.DB) string {
	t.Helper()
	seq := testDocumentSeq.Add(1)
	// Add nanoseconds to reduce collision risk across parallel test runs
	unique := fmt.Sprintf("%d-%d", time.Now().UnixNano(), seq)
	id := "test-doc-" + unique
	code := "TEST-" + unique
	sequence := int(seq + time.Now().UnixNano()%1000000)

	_, err := db.Exec(`
		INSERT INTO metaldocs.documents (
			id, title, owner_id, classification, status,
			created_at, updated_at,
			document_type_code, business_unit, department,
			document_profile_code, document_family_code,
			document_sequence, document_code, document_type_key
		) VALUES (
			$1, $2, $3, $4, $5,
			now(), now(),
			$6, $7, $8,
			$9, $10,
			$11, $12, $13
		)
	`,
		id, "Test Document "+unique, "test-owner", "internal", "draft",
		"po", "test-bu", "test-dept",
		"po", "procedure",
		sequence, code, "test-key",
	)
	if err != nil {
		t.Fatalf("newTestDocument: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM metaldocs.documents WHERE id = $1`, id)
	})
	return id
}
