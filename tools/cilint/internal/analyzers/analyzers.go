// Package analyzers runs all cilint analyzers and aggregates findings.
package analyzers

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Finding is a lint violation.
type Finding struct {
	Analyzer string
	File     string
	Line     int
	Message  string
}

// allowedTxPackages are packages allowed to call BeginTx/Commit/Rollback.
var allowedTxPackages = []string{
	"approval/application",
	"jobs/",
	"iam/area_membership",
}

// RunAll runs every analyzer over the given patterns and aggregates findings.
func RunAll(targets []string) []Finding {
	files := collectGoFiles(targets)
	var out []Finding
	out = append(out, TxOwnership(files)...)
	out = append(out, AuthzRequire(files)...)
	out = append(out, LegacyVocab(files)...)
	out = append(out, OutboxPair(files)...)
	return out
}

func collectGoFiles(patterns []string) []string {
	var files []string
	for _, pat := range patterns {
		pat = strings.TrimSuffix(pat, "/...")
		if pat == "./..." || pat == "." {
			pat = "."
		}
		_ = filepath.WalkDir(pat, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".go") &&
				!strings.Contains(path, "_test.go") &&
				!strings.Contains(path, "vendor/") {
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}

func parseFile(fset *token.FileSet, path string) (*token.File, any) {
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, nil
	}
	tf := fset.File(f.Pos())
	return tf, f
}

func readSource(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}

func inAllowedPackage(path string) bool {
	for _, pkg := range allowedTxPackages {
		if strings.Contains(filepath.ToSlash(path), pkg) {
			return true
		}
	}
	return false
}
