package analyzers

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// legacyPattern matches deprecated vocabulary in Go and TypeScript sources.
var legacyPattern = regexp.MustCompile(`(?i)\b(finalized|archived|document\.finalize|document\.archive)\b`)

// legacyExcludeDirs are excluded from legacy vocab checks.
var legacyExcludeDirs = []string{
	"migrations/",
	"fixtures/",
	"testdata/",
}

// LegacyVocab reports legacy vocabulary in .go, .ts, and .tsx files
// excluding test fixtures and historical event type enums.
func LegacyVocab(goFiles []string) []Finding {
	// Expand to include .ts/.tsx in addition to .go
	var allFiles []string
	allFiles = append(allFiles, goFiles...)

	// Walk frontend sources too
	_ = filepath.WalkDir("frontend/apps/web/src", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") {
			allFiles = append(allFiles, path)
		}
		return nil
	})

	var out []Finding

	for _, path := range allFiles {
		if isLegacyExcluded(path) {
			continue
		}

		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Skip lines with allow directive
			if strings.Contains(line, "cilint:allow-legacy") {
				continue
			}
			// Skip historical event type constant definitions (e.g., EventTypeArchived = "archived")
			if strings.Contains(line, "EventType") && strings.Contains(line, "=") {
				continue
			}

			if legacyPattern.MatchString(line) {
				matches := legacyPattern.FindAllString(line, -1)
				out = append(out, Finding{
					Analyzer: "legacyvocab",
					File:     path,
					Line:     lineNum,
					Message:  "legacy vocabulary found: " + strings.Join(matches, ", ") + " — use current terminology (published/cancelled/obsolete)",
				})
			}
		}
		_ = f.Close()
	}
	return out
}

func isLegacyExcluded(path string) bool {
	slash := strings.ReplaceAll(path, "\\", "/")
	for _, excl := range legacyExcludeDirs {
		if strings.Contains(slash, excl) {
			return true
		}
	}
	return false
}
