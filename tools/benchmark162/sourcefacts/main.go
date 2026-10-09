// Source-only facts for the fixed, same-package Go benchmark. Does not execute source.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
)

type File struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type Facts struct {
	Package     string   `json:"package"`
	Definitions []string `json:"definitions"`
	Calls       []string `json:"calls"`
}
type Row struct {
	ID     string `json:"id"`
	Before Facts  `json:"before"`
	After  Facts  `json:"after"`
}

func facts(name, source string) (Facts, error) {
	tree, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
	if err != nil {
		return Facts{}, err
	}
	out := Facts{Package: tree.Name.Name, Definitions: []string{}, Calls: []string{}}
	for _, d := range tree.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			if f.Recv != nil {
				return Facts{}, fmt.Errorf("unsupported method")
			}
			out.Definitions = append(out.Definitions, f.Name.Name)
		}
	}
	seen := map[string]bool{}
	ast.Inspect(tree, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				seen[id.Name] = true
			}
		}
		return true
	})
	for name := range seen {
		out.Calls = append(out.Calls, name)
	}
	sort.Strings(out.Definitions)
	sort.Strings(out.Calls)
	return out, nil
}
func main() {
	var files []File
	if err := json.NewDecoder(os.Stdin).Decode(&files); err != nil {
		panic(err)
	}
	rows := []Row{}
	for _, f := range files {
		b, err := facts(f.Path, f.Before)
		if err != nil {
			panic(err)
		}
		a, err := facts(f.Path, f.After)
		if err != nil {
			panic(err)
		}
		if b.Package != a.Package {
			panic("unsupported package change")
		}
		rows = append(rows, Row{f.ID, b, a})
	}
	if err := json.NewEncoder(os.Stdout).Encode(rows); err != nil {
		panic(err)
	}
}
