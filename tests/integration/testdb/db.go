//go:build integration
// +build integration

package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// testNamespace is a fixed UUID v5 namespace for deterministic fixture IDs.
var testNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

// DeterministicID returns a deterministic UUID v5 based on test name + suffix.
func DeterministicID(t *testing.T, suffix string) string {
	t.Helper()
	return uuid.NewSHA1(testNamespace, []byte(t.Name()+":"+suffix)).String()
}

// DSN returns the test database connection string from env, or skips.
func DSN(t *testing.T) string {
	t.Helper()
	if v := strings.TrimSpace(os.Getenv("METALDOCS_DATABASE_URL")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("DATABASE_URL")); v != "" {
		return v
	}
	t.Skip("DATABASE_URL/METALDOCS_DATABASE_URL not set")
	return ""
}

// Open returns a *sql.DB connected to the test database and a unique schema
// for this test. The schema is dropped automatically at test cleanup.
func Open(t *testing.T) (*sql.DB, string) {
	t.Helper()

	db, err := sql.Open("pgx", DSN(t))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("integration DB unreachable: %v", err)
	}

	schema := "metaldocs_test_" + randomSuffix(t)
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quoteIdent(schema)); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	ApplyMigrations(t, db, schema)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+quoteIdent(schema)+" CASCADE")
	})
	return db, schema
}

// Qualified returns a fully-qualified table/function name in the test schema.
func Qualified(schema, object string) string {
	return quoteIdent(schema) + "." + quoteIdent(object)
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return fmt.Sprintf("%x", b)
}

func quoteIdent(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

// repoRoot finds the repo root by walking up from this file.
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find repo root")
		}
		dir = parent
	}
}

var (
	schemaPattern = regexp.MustCompile(`(?i)\bSCHEMA\s+metaldocs\b`)
	dotPattern    = regexp.MustCompile(`\bmetaldocs\.`)
	searchPathEq  = regexp.MustCompile(`(?i)(search_path\s*=\s*)metaldocs\b`)
	searchPathTo  = regexp.MustCompile(`(?i)(search_path\s+to\s+)metaldocs\b`)
)

func rewriteSchema(sqlText, schema string) string {
	qSchema := quoteIdent(schema)
	out := dotPattern.ReplaceAllString(sqlText, qSchema+".")
	out = schemaPattern.ReplaceAllString(out, "SCHEMA "+qSchema)
	out = searchPathEq.ReplaceAllString(out, "${1}"+qSchema)
	out = searchPathTo.ReplaceAllString(out, "${1}"+qSchema)
	return out
}

// ApplyMigrations runs all migration SQL files (excluding *_down.sql) in order.
func ApplyMigrations(t *testing.T, db *sql.DB, schema string) {
	t.Helper()

	root := repoRoot()
	migrationsDir := filepath.Join(root, "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") || strings.HasSuffix(e.Name(), "_down.sql") {
			continue
		}
		files = append(files, filepath.Join(migrationsDir, e.Name()))
	}
	sort.Strings(files)

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read migration %s: %v", f, err)
		}
		sqlText := rewriteSchema(string(sqlBytes), schema)
		if _, err := db.Exec(sqlText); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(f), err)
		}
	}
}
