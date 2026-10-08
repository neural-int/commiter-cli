// This is a strict source-binding proof, not a Go evaluator or intent classifier.
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
type Ref struct {
	File    string `json:"file"`
	Version string `json:"version"`
	Span    [2]int `json:"span"`
	Text    string `json:"text"`
}
type Parsed struct {
	file            File
	version, source string
	tree            *ast.File
	fs              *token.FileSet
}

func (p Parsed) ref(n ast.Node) Ref {
	a, b := p.fs.Position(n.Pos()).Offset, p.fs.Position(n.End()).Offset
	return Ref{p.file.ID, p.version, [2]int{a, b}, p.source[a:b]}
}

type Rule struct {
	name, method, left, right string
	li, ri, fields            int
	definition, condition     Ref
}
type Binding struct {
	Row        Ref `json:"row"`
	Actual     Ref `json:"actual_expression"`
	Expected   Ref `json:"expected_literal"`
	Definition Ref `json:"type_definition"`
	Condition  Ref `json:"failure_condition"`
	Invocation Ref `json:"validator_invocation"`
}

func id(e ast.Expr) string {
	if x, ok := e.(*ast.Ident); ok {
		return x.Name
	}
	return ""
}
func field(e ast.Expr, base string) string {
	if x, ok := e.(*ast.SelectorExpr); ok && id(x.X) == base {
		return x.Sel.Name
	}
	return ""
}
func testingParam(p Parsed, fn *ast.FuncDecl) string {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return ""
	}
	a := fn.Type.Params.List[0]
	if len(a.Names) != 1 {
		return ""
	}
	ptr, ok := a.Type.(*ast.StarExpr)
	if !ok {
		return ""
	}
	sel, ok := ptr.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "T" {
		return ""
	}
	for _, im := range p.tree.Imports {
		alias := "testing"
		if im.Name != nil {
			alias = im.Name.Name
		}
		if im.Path.Value == "\"testing\"" && id(sel.X) == alias {
			return a.Names[0].Name
		}
	}
	return ""
}
func rules(p Parsed) []Rule {
	out := []Rule{}
	for _, d := range p.tree.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 || fn.Body == nil || len(fn.Body.List) != 1 {
			continue
		}
		recv := fn.Recv.List[0]
		if len(recv.Names) != 1 {
			continue
		}
		name := id(recv.Type)
		if name == "" {
			continue
		}
		param := testingParam(p, fn)
		if param == "" {
			continue
		}
		loop, ok := fn.Body.List[0].(*ast.RangeStmt)
		if !ok || id(loop.X) != recv.Names[0].Name || loop.Tok != token.DEFINE || id(loop.Key) != "_" || id(loop.Value) == "" || id(loop.Value) == param || len(loop.Body.List) != 1 {
			continue
		}
		cond, ok := loop.Body.List[0].(*ast.IfStmt)
		if !ok || cond.Init != nil || cond.Else != nil || len(cond.Body.List) != 1 {
			continue
		}
		cmp, ok := cond.Cond.(*ast.BinaryExpr)
		if !ok || cmp.Op != token.NEQ {
			continue
		}
		left, right := field(cmp.X, id(loop.Value)), field(cmp.Y, id(loop.Value))
		if left == "" || right == "" {
			continue
		}
		stmt, ok := cond.Body.List[0].(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := stmt.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || id(sel.X) != param || sel.Sel.Name != "Errorf" {
			continue
		}
		for _, decl := range p.tree.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gen.Specs {
				typ, ok := spec.(*ast.TypeSpec)
				if !ok || typ.Name.Name != name {
					continue
				}
				arr, ok := typ.Type.(*ast.ArrayType)
				if !ok || arr.Len != nil {
					continue
				}
				st, ok := arr.Elt.(*ast.StructType)
				if !ok {
					continue
				}
				names := []string{}
				valid := true
				for _, f := range st.Fields.List {
					if id(f.Type) != "string" {
						valid = false
					}
					for _, n := range f.Names {
						names = append(names, n.Name)
					}
				}
				li, ri := -1, -1
				for i, n := range names {
					if n == left {
						li = i
					}
					if n == right {
						ri = i
					}
				}
				if valid && li >= 0 && ri >= 0 && li != ri {
					out = append(out, Rule{name, fn.Name.Name, left, right, li, ri, len(names), p.ref(typ), p.ref(cmp)})
				}
			}
		}
	}
	return out
}
func main() {
	var req struct {
		Files []File `json:"files"`
	}
	if json.NewDecoder(os.Stdin).Decode(&req) != nil {
		os.Exit(1)
	}
	bindings := []Binding{}
	for _, version := range []string{"before", "after"} {
		parsed := []Parsed{}
		all := map[string][]Rule{}
		for _, f := range req.Files {
			if !strings.HasSuffix(f.Path, "_test.go") {
				continue
			}
			source := f.Before
			if version == "after" {
				source = f.After
			}
			fs := token.NewFileSet()
			tree, err := parser.ParseFile(fs, f.Path, source, 0)
			if err != nil {
				os.Exit(2)
			}
			p := Parsed{f, version, source, tree, fs}
			parsed = append(parsed, p)
			for _, r := range rules(p) {
				key := tree.Name.Name + "/" + r.name + "/" + r.method
				all[key] = append(all[key], r)
			}
		}
		for _, p := range parsed {
			for _, d := range p.tree.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil || len(fn.Body.List) != 1 {
					continue
				}
				only, direct := fn.Body.List[0].(*ast.ExprStmt)
				if !direct {
					continue
				}
				param := testingParam(p, fn)
				if param == "" {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok || only.X != call || len(call.Args) != 1 || id(call.Args[0]) != param {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					literal, ok := sel.X.(*ast.CompositeLit)
					if !ok {
						return true
					}
					key := p.tree.Name.Name + "/" + id(literal.Type) + "/" + sel.Sel.Name
					if len(all[key]) != 1 {
						return true
					}
					r := all[key][0]
					for _, e := range literal.Elts {
						row, ok := e.(*ast.CompositeLit)
						if !ok || len(row.Elts) != r.fields {
							continue
						}
						keyed := false
						for _, x := range row.Elts {
							if _, ok := x.(*ast.KeyValueExpr); ok {
								keyed = true
							}
						}
						expected, ok := row.Elts[r.ri].(*ast.BasicLit)
						if keyed || !ok || expected.Kind != token.STRING {
							continue
						}
						bindings = append(bindings, Binding{p.ref(row), p.ref(row.Elts[r.li]), p.ref(expected), r.definition, r.condition, p.ref(call)})
					}
					return true
				})
			}
		}
	}
	if json.NewEncoder(os.Stdout).Encode(bindings) != nil {
		os.Exit(3)
	}
}
