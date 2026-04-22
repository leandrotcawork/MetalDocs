//go:build integration
// +build integration

package scenarios_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoLegacyStatusInGoSource(t *testing.T) {
	root := repoRootForIntegrationTests()
	base := filepath.Join(root, "internal")

	files, err := collectSourceFiles(base, map[string]bool{".go": true})
	if err != nil {
		t.Fatalf("collect go files: %v", err)
	}

	var violations []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if containsFixturePath(file) {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}

		for i, line := range strings.Split(string(content), "\n") {
			if legacyLiteralViolationInGoLine(line) {
				violations = append(violations, fmt.Sprintf("%s:%d", file, i+1))
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("legacy status literals found in Go source:\n%s", strings.Join(violations, "\n"))
	}
}

func TestNoLegacyStatusInTSSource(t *testing.T) {
	root := repoRootForIntegrationTests()
	base := filepath.Join(root, "frontend", "apps", "web", "src")

	if _, err := os.Stat(base); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("frontend source dir not found: %s", base)
		}
		t.Fatalf("stat frontend source dir: %v", err)
	}

	files, err := collectSourceFiles(base, map[string]bool{".ts": true, ".tsx": true})
	if err != nil {
		t.Fatalf("collect ts/tsx files: %v", err)
	}

	var violations []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}

		for i, line := range strings.Split(string(content), "\n") {
			if legacyLiteralViolationInTSLine(line) {
				violations = append(violations, fmt.Sprintf("%s:%d", file, i+1))
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("legacy status literals found in TS source:\n%s", strings.Join(violations, "\n"))
	}
}

func TestGoVetPasses(t *testing.T) {
	root := repoRootForIntegrationTests()
	cmd := exec.Command("go", "vet", "./internal/...")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet ./internal/... failed: %v\n%s", err, strings.TrimSpace(string(out)))
	}
}

func TestStaticcheckInstalled(t *testing.T) {
	if _, err := exec.LookPath("staticcheck"); err != nil {
		t.Skip("staticcheck not found in PATH")
	}

	root := repoRootForIntegrationTests()
	cmd := exec.Command("staticcheck", "./internal/modules/documents_v2/approval/...")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		t.Fatalf("staticcheck failed: %v\n%s", err, output)
	}
	if output != "" {
		t.Fatalf("staticcheck reported findings:\n%s", output)
	}
}

func collectSourceFiles(dir string, exts map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if shouldSkipDirName(name) && entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, name)
		if entry.IsDir() {
			child, err := collectSourceFiles(path, exts)
			if err != nil {
				return nil, err
			}
			out = append(out, child...)
			continue
		}

		if exts[strings.ToLower(filepath.Ext(name))] {
			out = append(out, path)
		}
	}
	return out, nil
}

func shouldSkipDirName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "vendor", "node_modules", "testdata":
		return true
	default:
		return false
	}
}

func containsFixturePath(path string) bool {
	normalized := strings.ToLower(filepath.ToSlash(path))
	return strings.Contains(normalized, "/fixture/") || strings.Contains(normalized, "/fixtures/")
}

func legacyLiteralViolationInGoLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	if strings.HasPrefix(trimmed, "//") {
		return commentLegacyViolation(trimmed)
	}

	codePart := line
	if idx := strings.Index(codePart, "//"); idx >= 0 {
		codePart = codePart[:idx]
	}

	return hasLegacyQuotedLiteral(codePart)
}

func legacyLiteralViolationInTSLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	if strings.Contains(strings.ToLower(trimmed), "event_type") {
		return false
	}

	codePart := line
	if idx := strings.Index(codePart, "//"); idx >= 0 {
		codePart = codePart[:idx]
	}
	return hasLegacyQuotedLiteral(codePart)
}

func commentLegacyViolation(comment string) bool {
	lower := strings.ToLower(comment)
	if !(strings.Contains(lower, "finalized") || strings.Contains(lower, "archived")) {
		return false
	}
	if strings.Contains(lower, "legacy removed") ||
		strings.Contains(lower, "legacy status") ||
		strings.Contains(lower, "legacy") ||
		strings.Contains(lower, "historical") {
		return false
	}
	return true
}

func hasLegacyQuotedLiteral(s string) bool {
	return strings.Contains(s, "'finalized'") ||
		strings.Contains(s, "'archived'") ||
		strings.Contains(s, "\"finalized\"") ||
		strings.Contains(s, "\"archived\"") ||
		strings.Contains(s, "`finalized`") ||
		strings.Contains(s, "`archived`")
}
