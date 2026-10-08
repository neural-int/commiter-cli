// Observe lexical statement scope; this does not infer shared intent.
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

type observation struct {
	Start int      `json:"start"`
	End   int      `json:"end"`
	Kind  string   `json:"kind"`
	Calls []string `json:"calls"`
}

func observe(src string) ([]observation, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "source.go", src, 0)
	if err != nil {
		return nil, err
	}
	out := []observation{}
	ast.Inspect(f, func(n ast.Node) bool {
		st, ok := n.(ast.Stmt)
		if !ok {
			return true
		}
		if _, block := st.(*ast.BlockStmt); block {
			return true
		}
		calls := []string{}
		ast.Inspect(st, func(x ast.Node) bool {
			c, ok := x.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch v := c.Fun.(type) {
			case *ast.Ident:
				calls = append(calls, v.Name)
			case *ast.SelectorExpr:
				if a, ok := v.X.(*ast.Ident); ok {
					calls = append(calls, a.Name+"."+v.Sel.Name)
				}
			}
			return true
		})
		sort.Strings(calls)
		out = append(out, observation{fs.Position(st.Pos()).Offset, fs.Position(st.End()).Offset, "lexical_statement", calls})
		return true
	})
	return out, nil
}
func main() {
	b, err := io.ReadAll(io.LimitReader(os.Stdin, 1048577))
	if err != nil || len(b) > 1048576 {
		panic("source_budget")
	}
	out, err := observe(string(b))
	if err != nil {
		panic("parse_unavailable")
	}
	json.NewEncoder(os.Stdout).Encode(out)
}
