// Bounded syntactic repository evidence; structural calls are soft signals.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
)

type source struct{ ID, Path, Content string }
type edge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Caller string `json:"caller"`
	Callee string `json:"callee"`
}
type parsed struct {
	src       source
	tree      *ast.File
	imports   map[string]string
	namespace string
}

func main() {
	var src []source
	dec := json.NewDecoder(io.LimitReader(os.Stdin, 4*1024*1024+1))
	if err := dec.Decode(&src); err != nil {
		panic("invalid_input")
	}
	if len(src) > 64 {
		panic("file_budget")
	}
	size := 0
	files := []parsed{}
	defs := map[string][]string{}
	unknown := 0
	for _, s := range src {
		size += len(s.Content)
		if size > 4*1024*1024 {
			panic("byte_budget")
		}
		tree, e := parser.ParseFile(token.NewFileSet(), s.Path, s.Content, 0)
		if e != nil {
			unknown++
			continue
		}
		ns := path.Dir(s.Path)
		im := map[string]string{}
		for _, x := range tree.Imports {
			v, e := strconv.Unquote(x.Path.Value)
			if e != nil {
				continue
			}
			name := path.Base(v)
			if x.Name != nil {
				name = x.Name.Name
			}
			im[name] = v
		}
		files = append(files, parsed{s, tree, im, ns})
		for _, d := range tree.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil {
				key := ns + "." + f.Name.Name
				defs[key] = append(defs[key], s.ID)
			}
		}
	}
	edges := []edge{}
	seen := map[string]bool{}
	for _, f := range files {
		for _, d := range f.tree.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				key := ""
				name := ""
				switch v := call.Fun.(type) {
				case *ast.Ident:
					name = v.Name
					key = f.namespace + "." + name
				case *ast.SelectorExpr:
					alias, ok := v.X.(*ast.Ident)
					if !ok {
						return true
					}
					imp := f.imports[alias.Name]
					if len(imp) > 8 && imp[:8] == "fixture/" {
						name = v.Sel.Name
						key = imp[8:] + "." + name
					}
				}
				targets := defs[key]
				if len(targets) == 1 && targets[0] != f.src.ID {
					k := f.src.ID + "\x00" + targets[0] + "\x00" + fn.Name.Name + "\x00" + name
					if !seen[k] {
						seen[k] = true
						edges = append(edges, edge{f.src.ID, targets[0], fn.Name.Name, name})
					}
				}
				return true
			})
		}
	}
	sort.Slice(edges, func(i, j int) bool { return fmt.Sprint(edges[i]) < fmt.Sprint(edges[j]) })
	json.NewEncoder(os.Stdout).Encode(map[string]any{"edges": edges, "parse_unavailable": unknown, "files": len(src), "source_bytes": size, "semantics": "soft_syntactic_call_not_shared_intent"})
}
