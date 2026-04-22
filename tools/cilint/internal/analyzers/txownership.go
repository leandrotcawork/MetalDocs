package analyzers

import (
	"go/ast"
	"go/token"
	"strings"
)

// txCalls is the set of sql.DB/sql.Tx methods that indicate direct tx management.
var txCalls = map[string]bool{
	"BeginTx":  true,
	"Begin":    true,
	"Commit":   true,
	"Rollback": true,
}

// TxOwnership reports BeginTx/Commit/Rollback outside allowed packages.
func TxOwnership(files []string) []Finding {
	var out []Finding
	fset := token.NewFileSet()

	for _, path := range files {
		if inAllowedPackage(path) {
			continue
		}
		tf, raw := parseFile(fset, path)
		if raw == nil {
			continue
		}
		f := raw.(*ast.File)

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			method := sel.Sel.Name
			if !txCalls[method] {
				return true
			}
			// Allow if comment directive present
			pos := fset.Position(call.Pos())
			src := readSource(path)
			line := getLine(src, pos.Line)
			if strings.Contains(line, "//cilint:allow-tx") {
				return true
			}
			if tf != nil {
				_ = tf
			}
			out = append(out, Finding{
				Analyzer: "txownership",
				File:     path,
				Line:     pos.Line,
				Message:  "tx management (" + method + ") outside allowed package; use service layer",
			})
			return true
		})
	}
	return out
}

func getLine(src string, lineNum int) string {
	lines := strings.Split(src, "\n")
	if lineNum < 1 || lineNum > len(lines) {
		return ""
	}
	return lines[lineNum-1]
}
