// Bounded syntactic expression dependencies, not an intent classifier.
package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"sort"
)

type row struct {
	Start  int      `json:"start"`
	End    int      `json:"end"`
	Calls  []string `json:"calls"`
	Status string   `json:"status"`
}

func observe(src string) ([]row, error) {
	fs := token.NewFileSet()
	f, e := parser.ParseFile(fs, "source.go", src, 0)
	if e != nil {
		return nil, e
	}
	// Any non-definition write makes the initial definition insufficient.
	unstable := map[*ast.Object]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		mark := func(x ast.Expr) {
			if id, ok := x.(*ast.Ident); ok && id.Obj != nil {
				unstable[id.Obj] = true
			}
		}
		switch v := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range v.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && id.Obj != nil && (v.Tok != token.DEFINE || id.Obj.Decl != v) {
					mark(lhs)
				}
			}
		case *ast.IncDecStmt:
			mark(v.X)
		case *ast.UnaryExpr:
			if v.Op == token.AND {
				mark(v.X)
			}
		}
		return true
	})
	out := []row{}
	ast.Inspect(f, func(n ast.Node) bool {
		var roots []ast.Expr
		switch v := n.(type) {
		case *ast.BinaryExpr:
			roots = []ast.Expr{v}
		case *ast.CallExpr:
			roots = v.Args
		}
		for _, root := range roots {
			calls := map[string]bool{}
			seen := map[*ast.Object]bool{}
			unknown := false
			var visit func(ast.Node, int)
			visit = func(x ast.Node, depth int) {
				if depth > 8 {
					unknown = true
					return
				}
				ast.Inspect(x, func(z ast.Node) bool {
					switch v := z.(type) {
					case *ast.CallExpr:
						switch c := v.Fun.(type) {
						case *ast.Ident:
							calls[c.Name] = true
						case *ast.SelectorExpr:
							if a, ok := c.X.(*ast.Ident); ok {
								calls[a.Name+"."+c.Sel.Name] = true
							} else {
								unknown = true
							}
						}
					case *ast.Ident:
						if v.Obj != nil && v.Obj.Kind == ast.Var && !seen[v.Obj] {
							seen[v.Obj] = true
							if unstable[v.Obj] {
								unknown = true
								return false
							}
							switch d := v.Obj.Decl.(type) {
							case *ast.AssignStmt:
								if len(d.Lhs) == 1 && len(d.Rhs) == 1 {
									visit(d.Rhs[0], depth+1)
								} else {
									unknown = true
								}
							case *ast.ValueSpec:
								if len(d.Values) == 1 {
									visit(d.Values[0], depth+1)
								} else {
									unknown = true
								}
							default:
								unknown = true
							}
						}
					}
					return true
				})
			}
			visit(root, 0)
			names := []string{}
			for c := range calls {
				names = append(names, c)
			}
			sort.Strings(names)
			status := "observed_syntactic"
			if unknown {
				status = "partial_unknown"
				names = []string{}
			}
			out = append(out, row{fs.Position(root.Pos()).Offset, fs.Position(root.End()).Offset, names, status})
		}
		return true
	})
	return out, nil
}
func main() {
	b, e := io.ReadAll(io.LimitReader(os.Stdin, 1048577))
	if e != nil || len(b) > 1048576 {
		panic("source_budget")
	}
	out, e := observe(string(b))
	if e != nil {
		panic("parse_unavailable")
	}
	json.NewEncoder(os.Stdout).Encode(out)
}
