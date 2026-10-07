// Synthetic benchmark AST annotations; no repository-wide evidence or scoring.
package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
)

type symbol struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func main() {
	b, e := io.ReadAll(os.Stdin)
	if e != nil {
		panic(e)
	}
	fs := token.NewFileSet()
	f, e := parser.ParseFile(fs, "input.go", b, 0)
	out := []symbol{}
	if e == nil {
		for _, d := range f.Decls {
			name := ""
			switch x := d.(type) {
			case *ast.FuncDecl:
				name = x.Name.Name
			case *ast.GenDecl:
				for _, s := range x.Specs {
					switch v := s.(type) {
					case *ast.ValueSpec:
						for _, n := range v.Names {
							out = append(out, symbol{n.Name, fs.Position(v.Pos()).Offset, fs.Position(v.End()).Offset})
						}
					case *ast.TypeSpec:
						out = append(out, symbol{v.Name.Name, fs.Position(v.Pos()).Offset, fs.Position(v.End()).Offset})
					}
				}
			}
			if name != "" {
				out = append(out, symbol{name, fs.Position(d.Pos()).Offset, fs.Position(d.End()).Offset})
			}
		}
	}
	json.NewEncoder(os.Stdout).Encode(out)
}
