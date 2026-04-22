//go:build integration
// +build integration

package scenarios_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReflect_RepositoryNoBeginTx(t *testing.T) {
	root := repoRootForIntegrationTests()
	repoDir := filepath.Join(root, "internal", "modules", "documents_v2", "approval", "repository")
	assertNoForbiddenTxCalls(t, repoDir, []string{"db.BeginTx(", "tx.Commit(", "tx.Rollback("})
}

func TestHTTPHandlers_NoBeginTx(t *testing.T) {
	root := repoRootForIntegrationTests()
	httpDir := filepath.Join(root, "internal", "modules", "documents_v2", "approval", "http")
	assertNoForbiddenTxCalls(t, httpDir, []string{"db.BeginTx("})
}

func assertNoForbiddenTxCalls(t *testing.T, dir string, forbidden []string) {
	t.Helper()
	var violations []string

	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(b)
		for _, token := range forbidden {
			if strings.Contains(content, token) {
				violations = append(violations, path+": contains "+token)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk %s: %v", dir, walkErr)
	}

	if len(violations) > 0 {
		t.Fatalf("transaction ownership violation(s):\n%s", strings.Join(violations, "\n"))
	}
}

func repoRootForIntegrationTests() string {
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
