package analyzers_test

import (
	"os"
	"path/filepath"
	"testing"

	"metaldocs/tools/cilint/internal/analyzers"
)

func writeFixture(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// ─── TxOwnership ────────────────────────────────────────────────────────────

func TestTxOwnership_Positive(t *testing.T) {
	src := `package foo
import "database/sql"
func Bad(db *sql.DB) {
	tx, _ := db.BeginTx(nil, nil) // violation
	_ = tx
}
`
	path := writeFixture(t, "bad_tx.go", src)
	findings := analyzers.TxOwnership([]string{path})
	if len(findings) == 0 {
		t.Fatal("expected finding for BeginTx outside allowed package")
	}
}

func TestTxOwnership_Negative_AllowedPackage(t *testing.T) {
	src := `package application
import "database/sql"
func Good(db *sql.DB) {
	tx, _ := db.BeginTx(nil, nil)
	_ = tx
}
`
	// Place in allowed package path
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "approval", "application")
	_ = os.MkdirAll(pkgDir, 0o755)
	path := filepath.Join(pkgDir, "good.go")
	_ = os.WriteFile(path, []byte(src), 0o644)

	findings := analyzers.TxOwnership([]string{path})
	if len(findings) != 0 {
		t.Fatalf("expected no findings for allowed package, got %d", len(findings))
	}
}

// ─── LegacyVocab ────────────────────────────────────────────────────────────

func TestLegacyVocab_Positive(t *testing.T) {
	src := `package foo
const badName = "finalized" // legacy
`
	path := writeFixture(t, "legacy.go", src)
	findings := analyzers.LegacyVocab([]string{path})
	if len(findings) == 0 {
		t.Fatal("expected finding for 'finalized'")
	}
}

func TestLegacyVocab_Negative_AllowDirective(t *testing.T) {
	src := `package foo
const ok = "finalized" // cilint:allow-legacy historical constant
`
	path := writeFixture(t, "legacy_allowed.go", src)
	findings := analyzers.LegacyVocab([]string{path})
	if len(findings) != 0 {
		t.Fatalf("expected no findings with allow directive, got %d", len(findings))
	}
}

// ─── AuthzRequire ────────────────────────────────────────────────────────────

func TestAuthzRequire_Positive(t *testing.T) {
	src := `package application
type Svc struct{}
func (s *Svc) BadMethod(ctx interface{}) error {
	return nil // no authz.Require
}
`
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "approval", "application")
	_ = os.MkdirAll(pkgDir, 0o755)
	path := filepath.Join(pkgDir, "svc.go")
	_ = os.WriteFile(path, []byte(src), 0o644)

	findings := analyzers.AuthzRequire([]string{path})
	if len(findings) == 0 {
		t.Fatal("expected finding for missing authz.Require")
	}
}

func TestAuthzRequire_Negative_HasRequire(t *testing.T) {
	src := `package application
type Svc struct{}
func (s *Svc) GoodMethod(ctx interface{}) error {
	if err := authz.Require(ctx, "perm"); err != nil {
		return err
	}
	return nil
}
`
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "approval", "application")
	_ = os.MkdirAll(pkgDir, 0o755)
	path := filepath.Join(pkgDir, "svc_good.go")
	_ = os.WriteFile(path, []byte(src), 0o644)

	findings := analyzers.AuthzRequire([]string{path})
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d: %+v", len(findings), findings)
	}
}
