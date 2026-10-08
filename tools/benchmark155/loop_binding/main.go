// Strict straight-line table-loop source binding. No source is executed.
package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
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
type Binding struct {
	Row, Input, Expected, Call, Condition Ref
	Function                              string
	Column                                int
}
type Origin struct {
	column int
	call   *ast.CallExpr
}

func id(e ast.Expr) string {
	if x, ok := e.(*ast.Ident); ok {
		return x.Name
	}
	return ""
}
func parameter(tree *ast.File, fn *ast.FuncDecl) string {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return ""
	}
	p := fn.Type.Params.List[0]
	if len(p.Names) != 1 {
		return ""
	}
	typ := p.Type
	if x, ok := typ.(*ast.StarExpr); ok {
		typ = x.X
	}
	sel, ok := typ.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "T" && sel.Sel.Name != "TB") {
		return ""
	}
	for _, im := range tree.Imports {
		alias := "testing"
		if im.Name != nil {
			alias = im.Name.Name
		}
		if im.Path.Value == "\"testing\"" && id(sel.X) == alias {
			return p.Names[0].Name
		}
	}
	return ""
}
func inspect(tree *ast.File, fn *ast.FuncDecl, ref func(ast.Node) Ref) []Binding {
	out := []Binding{}
	p := parameter(tree, fn)
	if p == "" || fn.Body == nil || len(fn.Body.List) != 2 {
		return out
	}
	decl, ok := fn.Body.List[0].(*ast.AssignStmt)
	if !ok || decl.Tok != token.DEFINE || len(decl.Lhs) != 1 || len(decl.Rhs) != 1 || id(decl.Lhs[0]) == "" || id(decl.Lhs[0]) == p {
		return out
	}
	table, ok := decl.Rhs[0].(*ast.CompositeLit)
	if !ok {
		return out
	}
	a, ok := table.Type.(*ast.ArrayType)
	if !ok || a.Len != nil {
		return out
	}
	b, ok := a.Elt.(*ast.ArrayType)
	if !ok || b.Len != nil || id(b.Elt) != "string" {
		return out
	}
	loop, ok := fn.Body.List[1].(*ast.RangeStmt)
	if !ok || loop.Tok != token.DEFINE || id(loop.X) != id(decl.Lhs[0]) || id(loop.Key) != "_" || id(loop.Value) == "" || id(loop.Value) == p || len(loop.Body.List) != 4 {
		return out
	}
	rowname := id(loop.Value)
	env := map[string]Origin{}
	for _, s := range loop.Body.List[:3] {
		assign, ok := s.(*ast.AssignStmt)
		if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return out
		}
		name := id(assign.Lhs[0])
		if name == "" || name == p || name == rowname || name == id(decl.Lhs[0]) {
			return out
		}
		if _, exists := env[name]; exists {
			return out
		}
		switch x := assign.Rhs[0].(type) {
		case *ast.IndexExpr:
			if id(x.X) != rowname {
				return out
			}
			lit, ok := x.Index.(*ast.BasicLit)
			if !ok || lit.Kind != token.INT {
				return out
			}
			col, err := strconv.Atoi(lit.Value)
			if err != nil || col < 0 {
				return out
			}
			env[name] = Origin{col, nil}
		case *ast.CallExpr:
			if id(x.Fun) == "" || len(x.Args) != 1 || x.Ellipsis.IsValid() {
				return out
			}
			arg, ok := env[id(x.Args[0])]
			if !ok || arg.call != nil {
				return out
			}
			env[name] = Origin{arg.column, x}
		default:
			return out
		}
	}
	branch, ok := loop.Body.List[3].(*ast.IfStmt)
	if !ok || branch.Init != nil || branch.Else != nil || len(branch.Body.List) != 1 {
		return out
	}
	cmp, ok := branch.Cond.(*ast.BinaryExpr)
	if !ok || cmp.Op != token.NEQ {
		return out
	}
	actual, aok := env[id(cmp.X)]
	expected, eok := env[id(cmp.Y)]
	if !aok || !eok || actual.call == nil || expected.call != nil {
		return out
	}
	es, ok := branch.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return out
	}
	errcall, ok := es.X.(*ast.CallExpr)
	if !ok {
		return out
	}
	sel, ok := errcall.Fun.(*ast.SelectorExpr)
	if !ok || id(sel.X) != p || sel.Sel.Name != "Errorf" {
		return out
	}
	for _, e := range table.Elts {
		row, ok := e.(*ast.CompositeLit)
		if !ok || actual.column >= len(row.Elts) || expected.column >= len(row.Elts) {
			return []Binding{}
		}
		for _, v := range row.Elts {
			lit, ok := v.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return []Binding{}
			}
		}
		out = append(out, Binding{ref(row), ref(row.Elts[actual.column]), ref(row.Elts[expected.column]), ref(actual.call), ref(cmp), fn.Name.Name, actual.column})
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
	out := []Binding{}
	for _, f := range req.Files {
		if !strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		for _, v := range []string{"before", "after"} {
			s := f.Before
			if v == "after" {
				s = f.After
			}
			fs := token.NewFileSet()
			tree, err := parser.ParseFile(fs, f.Path, s, 0)
			if err != nil {
				os.Exit(2)
			}
			ref := func(n ast.Node) Ref {
				a, b := fs.Position(n.Pos()).Offset, fs.Position(n.End()).Offset
				return Ref{f.ID, v, [2]int{a, b}, s[a:b]}
			}
			for _, d := range tree.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok {
					out = append(out, inspect(tree, fn, ref)...)
				}
			}
		}
	}
	if json.NewEncoder(os.Stdout).Encode(out) != nil {
		os.Exit(3)
	}
}
