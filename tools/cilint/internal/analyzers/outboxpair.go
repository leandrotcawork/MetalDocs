package analyzers

import (
	"go/ast"
	"go/token"
	"strings"
)

// mutatingRepos are repo method prefixes that indicate state mutation on approval tables.
var mutatingRepos = []string{
	"UpdateApproval",
	"UpdateDocument",
	"UpdateSignoff",
	"UpdateInstance",
	"CreateSignoff",
	"CreateInstance",
}

// OutboxPair reports methods in approval/application that mutate approval state
// without a paired events.Emit call (outbox pairing requirement).
func OutboxPair(files []string) []Finding {
	var out []Finding
	fset := token.NewFileSet()

	for _, path := range files {
		if !strings.Contains(strings.ReplaceAll(path, "\\", "/"), "approval/application") {
			continue
		}
		_, raw := parseFile(fset, path)
		if raw == nil {
			continue
		}
		f := raw.(*ast.File)

		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// Only check exported methods with receivers
			if fn.Recv == nil || fn.Name == nil || !fn.Name.IsExported() {
				continue
			}

			hasMutation := false
			hasEmit := false

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				name := sel.Sel.Name

				// Check for repo mutation calls
				for _, prefix := range mutatingRepos {
					if strings.HasPrefix(name, prefix) {
						hasMutation = true
					}
				}

				// Check for events.Emit call
				if name == "Emit" {
					if pkg, ok := sel.X.(*ast.Ident); ok && (pkg.Name == "events" || pkg.Name == "emitter" || pkg.Name == "e") {
						hasEmit = true
					}
				}
				return true
			})

			if hasMutation && !hasEmit {
				pos := fset.Position(fn.Pos())
				src := readSource(path)
				line := getLine(src, pos.Line)
				// Skip if explicitly allowed
				if strings.Contains(line, "cilint:allow-no-outbox") ||
					(fn.Doc != nil && funcDocContains(fn.Doc, "cilint:allow-no-outbox")) {
					continue
				}
				out = append(out, Finding{
					Analyzer: "outboxpair",
					File:     path,
					Line:     pos.Line,
					Message:  "method " + fn.Name.Name + " mutates approval state without paired events.Emit call; add governance event or //cilint:allow-no-outbox with justification",
				})
			}
		}
	}
	return out
}

func funcDocContains(doc *ast.CommentGroup, substr string) bool {
	for _, c := range doc.List {
		if strings.Contains(c.Text, substr) {
			return true
		}
	}
	return false
}
