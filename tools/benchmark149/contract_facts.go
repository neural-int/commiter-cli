package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

// Fixture module is the actual module written by executable contract tests.
// This bounded prototype resolves only unique selected Go declarations. It is
// soft syntactic evidence, not type checking or proof of a shared commit intent.
type callFact struct {
	Caller  string `json:"caller_file_id"`
	Callee  string `json:"callee_file_id"`
	Symbol  string `json:"symbol"`
	Class   string `json:"class"`
	Version string `json:"version"`
}

func versionSource(diff string, after bool) string {
	lines := []string{}
	for _, l := range strings.Split(diff, "\n") {
		if strings.HasPrefix(l, " ") {
			lines = append(lines, l[1:])
		} else if after && strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			lines = append(lines, l[1:])
		} else if !after && strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			lines = append(lines, l[1:])
		}
	}
	return strings.Join(lines, "\n")
}
func callFacts(f fixture) []callFact {
	out := []callFact{}
	for _, after := range []bool{false, true} {
		trees := map[string]*ast.File{}
		declarations := map[string][]string{}
		for _, file := range f.Files {
			if file.NewPath == nil || !strings.HasSuffix(*file.NewPath, ".go") {
				continue
			}
			tree, e := parser.ParseFile(token.NewFileSet(), *file.NewPath, versionSource(file.RawDiff, after), 0)
			if e != nil {
				continue
			}
			trees[file.ID] = tree
			for _, decl := range tree.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
					key := path.Dir(*file.NewPath) + "/" + fn.Name.Name
					declarations[key] = append(declarations[key], file.ID)
				}
			}
		}
		for _, file := range f.Files {
			tree := trees[file.ID]
			if tree == nil {
				continue
			}
			imports := map[string]string{}
			for _, i := range tree.Imports {
				p, e := strconv.Unquote(i.Path.Value)
				if e != nil || !strings.HasPrefix(p, "fixture/") {
					continue
				}
				alias := path.Base(p)
				if i.Name != nil {
					alias = i.Name.Name
				}
				if alias != "." && alias != "_" {
					imports[alias] = strings.TrimPrefix(p, "fixture/")
				}
			}
			ast.Inspect(tree, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				pkg, symbol := "", ""
				switch x := call.Fun.(type) {
				case *ast.Ident:
					if x.Obj == nil || x.Obj.Kind == ast.Fun {
						pkg = path.Dir(*file.NewPath)
						symbol = x.Name
					}
				case *ast.SelectorExpr:
					if id, ok := x.X.(*ast.Ident); ok && id.Obj == nil {
						pkg = imports[id.Name]
						symbol = x.Sel.Name
					}
				}
				if pkg == "" || symbol == "" {
					return true
				}
				targets := declarations[pkg+"/"+symbol]
				if len(targets) == 1 && targets[0] != file.ID {
					version := "before"
					if after {
						version = "after"
					}
					out = append(out, callFact{file.ID, targets[0], symbol, "soft", version})
				}
				return true
			})
		}
	}
	return out
}
