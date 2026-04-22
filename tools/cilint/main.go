// cilint — MetalDocs custom Go/TS linters.
// Runs all analyzers and emits SARIF to stdout.
//
// Usage:
//
//	go run ./tools/cilint [--sarif] [./...]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"metaldocs/tools/cilint/internal/analyzers"
)

func main() {
	sarif := flag.Bool("sarif", false, "emit SARIF output for GitHub code-scanning")
	flag.Parse()
	targets := flag.Args()
	if len(targets) == 0 {
		targets = []string{"./..."}
	}

	results := analyzers.RunAll(targets)

	if *sarif {
		out := buildSARIF(results)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		for _, r := range results {
			fmt.Fprintf(os.Stderr, "%s:%d: [%s] %s\n", r.File, r.Line, r.Analyzer, r.Message)
		}
	}

	if len(results) > 0 {
		os.Exit(1)
	}
}

// sarifResult is a minimal SARIF 2.1.0 structure for GitHub code-scanning.
func buildSARIF(results []analyzers.Finding) map[string]any {
	rules := map[string]bool{}
	sarfRules := []map[string]any{}
	sarfResults := []map[string]any{}

	for _, r := range results {
		if !rules[r.Analyzer] {
			rules[r.Analyzer] = true
			sarfRules = append(sarfRules, map[string]any{
				"id": r.Analyzer,
				"shortDescription": map[string]any{"text": r.Analyzer},
			})
		}
		sarfResults = append(sarfResults, map[string]any{
			"ruleId":  r.Analyzer,
			"message": map[string]any{"text": r.Message},
			"locations": []map[string]any{
				{
					"physicalLocation": map[string]any{
						"artifactLocation": map[string]any{"uri": r.File},
						"region":           map[string]any{"startLine": r.Line},
					},
				},
			},
		})
	}

	return map[string]any{
		"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Documents/CommitteeSpecifications/2.1.0/sarif-schema-2.1.0.json",
		"version": "2.1.0",
		"runs": []map[string]any{
			{
				"tool": map[string]any{
					"driver": map[string]any{
						"name":  "cilint",
						"rules": sarfRules,
					},
				},
				"results": sarfResults,
			},
		},
	}
}
