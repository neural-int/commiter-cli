// Source-only assertion inventory. It neither evaluates code nor infers intent.
package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type File struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type Node struct {
	File     string `json:"file"`
	Version  string `json:"version"`
	Function string `json:"function"`
	Span     [2]int `json:"span"`
	Text     string `json:"text"`
	Kind     string `json:"kind"`
}

func main() {
	var req struct {
		Files []File `json:"files"`
	}
	if json.NewDecoder(os.Stdin).Decode(&req) != nil {
		os.Exit(1)
	}
	nodes := []Node{}
	for _, f := range req.Files {
		if !strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		for _, version := range []string{"before", "after"} {
			s := f.Before
			if version == "after" {
				s = f.After
			}
			fs := token.NewFileSet()
			tree, err := parser.ParseFile(fs, f.Path, s, 0)
			if err != nil {
				os.Exit(2)
			}
			for _, d := range tree.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				add := func(n ast.Node, kind string) {
					a, b := fs.Position(n.Pos()).Offset, fs.Position(n.End()).Offset
					nodes = append(nodes, Node{f.ID, version, fn.Name.Name, [2]int{a, b}, s[a:b], kind})
				}
				add(fn, "function")
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					if a, ok := n.(*ast.AssignStmt); ok {
						add(a, "assignment_syntax")
					}
					if b, ok := n.(*ast.BranchStmt); ok {
						add(b, "branch_syntax")
					}
					branch, ok := n.(*ast.IfStmt)
					if !ok {
						return true
					}
					add(branch.Cond, "control_condition_syntax")
					failure := false
					ast.Inspect(branch.Body, func(x ast.Node) bool {
						call, ok := x.(*ast.CallExpr)
						if !ok {
							return true
						}
						sel, ok := call.Fun.(*ast.SelectorExpr)
						if ok {
							switch sel.Sel.Name {
							case "Error", "Errorf", "Fatal", "Fatalf":
								failure = true
							}
						}
						return true
					})
					if failure {
						add(branch.Cond, "failure_condition_syntax")
					}
					return true
				})
			}
		}
	}
	if json.NewEncoder(os.Stdout).Encode(nodes) != nil {
		os.Exit(3)
	}
}
