package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

type assertionFact struct {
	File       string   `json:"test_file_id"`
	Callee     string   `json:"callee_file_id,omitempty"`
	Function   string   `json:"function"`
	Arguments  []string `json:"arguments"`
	Expected   string   `json:"expected"`
	Comparison string   `json:"comparison"`
	Version    string   `json:"version"`
}

func expressionText(e ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, token.NewFileSet(), e)
	return b.String()
}

// This extracts observed test assertions, not a hidden gold intent or an
// execution result. Unsupported shapes remain in the literal diff input.
func assertionFacts(f fixture) []assertionFact {
	facts := []assertionFact{}
	calls := callFacts(f)
	for _, after := range []bool{false, true} {
		version := "before"
		if after {
			version = "after"
		}
		for _, file := range f.Files {
			if file.NewPath == nil || !strings.HasSuffix(*file.NewPath, "_test.go") {
				continue
			}
			tree, e := parser.ParseFile(token.NewFileSet(), *file.NewPath, versionSource(file.RawDiff, after), 0)
			if e != nil {
				continue
			}
			ast.Inspect(tree, func(n ast.Node) bool {
				stmt, ok := n.(*ast.IfStmt)
				if !ok {
					return true
				}
				fatal := false
				if len(stmt.Body.List) == 1 {
					if expression, ok := stmt.Body.List[0].(*ast.ExprStmt); ok {
						if call, ok := expression.X.(*ast.CallExpr); ok {
							if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
								id, ok := sel.X.(*ast.Ident)
								fatal = ok && id.Name == "t" && (sel.Sel.Name == "Fatal" || sel.Sel.Name == "Fatalf")
							}
						}
					}
				}

				if !fatal {
					return true
				}
				var call *ast.CallExpr
				expected, comparison := "", "equal"
				switch x := stmt.Cond.(type) {
				case *ast.CallExpr:
					call = x
					expected = "false"
				case *ast.UnaryExpr:
					if x.Op == token.NOT {
						call, _ = x.X.(*ast.CallExpr)
						expected = "true"
					}
				case *ast.BinaryExpr:
					if x.Op == token.NEQ || x.Op == token.EQL {
						if c, ok := x.X.(*ast.CallExpr); ok {
							call = c
							expected = expressionText(x.Y)
						} else if c, ok := x.Y.(*ast.CallExpr); ok {
							call = c
							expected = expressionText(x.X)
						}
						if x.Op == token.EQL {
							comparison = "not_equal"
						}
					}
				}
				if call == nil {
					return true
				}
				symbol := ""
				switch x := call.Fun.(type) {
				case *ast.Ident:
					symbol = x.Name
				case *ast.SelectorExpr:
					symbol = x.Sel.Name
				}
				if symbol == "" {
					return true
				}
				args := []string{}
				for _, a := range call.Args {
					args = append(args, expressionText(a))
				}
				target := ""
				ambiguous := false
				for _, c := range calls {
					if c.Caller == file.ID && c.Symbol == symbol && c.Version == version {
						if target != "" && target != c.Callee {
							ambiguous = true
						}
						target = c.Callee
					}
				}
				if ambiguous {
					target = ""
				}
				facts = append(facts, assertionFact{file.ID, target, symbol, args, expected, comparison, version})
				return true
			})
		}
	}
	return facts
}
