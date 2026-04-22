package analyzers

import (
	"go/ast"
	"go/token"
	"strings"
)

// AuthzRequire reports exported methods in approval/application that don't start
// with authz.Require(...) as their first statement.
func AuthzRequire(files []string) []Finding {
	var out []Finding
	fset := token.NewFileSet()

	for _, path := range files {
		slash := strings.ReplaceAll(path, "\\", "/")
		if !strings.Contains(slash, "approval/application") {
			continue
		}
		_, raw := parseFile(fset, path)
		if raw == nil {
			continue
		}
		f := raw.(*ast.File)

		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}
			// Only exported methods (receiver + exported name)
			if fn.Recv == nil || !fn.Name.IsExported() {
				continue
			}
			pos := fset.Position(fn.Pos())

			// Check for //cilint:allow-noauthz directive in function comment
			if fn.Doc != nil {
				for _, c := range fn.Doc.List {
					if strings.Contains(c.Text, "cilint:allow-noauthz") {
						goto next
					}
				}
			}

			if !firstStmtIsAuthzRequire(fn) {
				out = append(out, Finding{
					Analyzer: "authzrequire",
					File:     path,
					Line:     pos.Line,
					Message:  "exported method " + fn.Name.Name + " must call authz.Require(...) as first statement (or add //cilint:allow-noauthz with justification)",
				})
			}
		next:
		}
	}
	return out
}

func firstStmtIsAuthzRequire(fn *ast.FuncDecl) bool {
	if fn.Body == nil || len(fn.Body.List) == 0 {
		return false
	}
	first := fn.Body.List[0]
	// Accept: authz.Require(...) or _ = authz.Require(...)
	switch s := first.(type) {
	case *ast.ExprStmt:
		return isAuthzRequireCall(s.X)
	case *ast.AssignStmt:
		if len(s.Rhs) == 1 {
			return isAuthzRequireCall(s.Rhs[0])
		}
	case *ast.IfStmt:
		// if err := authz.Require(...); err != nil { ... }
		if s.Init != nil {
			if as, ok := s.Init.(*ast.AssignStmt); ok && len(as.Rhs) == 1 {
				return isAuthzRequireCall(as.Rhs[0])
			}
		}
	}
	return false
}

func isAuthzRequireCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkg.Name == "authz" && sel.Sel.Name == "Require"
}

