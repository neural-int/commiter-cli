// Strict straight-line table-loop source binding. No source is executed.
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
type Binding struct {
	Row, Input, Expected, Call, Condition Ref
	Function                              string
	Column                                int
	Guard, Continue                       Ref
	Path                                  string
}

func id(e ast.Expr) string {
	if n, ok := e.(*ast.Ident); ok {
		return n.Name
	}
	return ""
}
func field(e ast.Expr, base string) string {
	if n, ok := e.(*ast.SelectorExpr); ok && id(n.X) == base {
		return n.Sel.Name
	}
	return ""
}
func testingParameter(tree *ast.File, fn *ast.FuncDecl) string {
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
	for _, im := range tree.Imports {
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
func errorCall(stmt ast.Stmt, param string) bool {
	e, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	c, ok := e.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	s, ok := c.Fun.(*ast.SelectorExpr)
	return ok && id(s.X) == param && s.Sel.Name == "Errorf"
}
func inspect(tree *ast.File, fn *ast.FuncDecl, ref func(ast.Node) Ref) []Binding {
	out := []Binding{}
	param := testingParameter(tree, fn)
	if param == "" || fn.Body == nil || len(fn.Body.List) != 2 {
		return out
	}
	decl, ok := fn.Body.List[0].(*ast.AssignStmt)
	if !ok || decl.Tok != token.DEFINE || len(decl.Lhs) != 1 || len(decl.Rhs) != 1 {
		return out
	}
	name := id(decl.Lhs[0])
	table, ok := decl.Rhs[0].(*ast.CompositeLit)
	if !ok || name == "" || name == param {
		return out
	}
	slice, ok := table.Type.(*ast.ArrayType)
	if !ok || slice.Len != nil {
		return out
	}
	st, ok := slice.Elt.(*ast.StructType)
	if !ok {
		return out
	}
	fields := []string{}
	types := []string{}
	for _, f := range st.Fields.List {
		typ := id(f.Type)
		if typ != "string" && typ != "uint64" {
			return out
		}
		for _, n := range f.Names {
			fields = append(fields, n.Name)
			types = append(types, typ)
		}
	}
	loop, ok := fn.Body.List[1].(*ast.RangeStmt)
	if !ok || loop.Tok != token.DEFINE || id(loop.X) != name || id(loop.Key) != "_" || id(loop.Value) == "" || id(loop.Value) == param || id(loop.Value) == name || len(loop.Body.List) != 3 {
		return out
	}
	r := id(loop.Value)
	assign, ok := loop.Body.List[0].(*ast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
		return out
	}
	got, errname := id(assign.Lhs[0]), id(assign.Lhs[1])
	if got == "" || errname == "" || got == errname || got == r || errname == r || got == param || errname == param || got == name || errname == name {
		return out
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || id(call.Fun) == "" || len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return out
	}
	input := field(call.Args[0], r)
	if input == "" {
		return out
	}
	guard, ok := loop.Body.List[1].(*ast.IfStmt)
	if !ok || guard.Init != nil || guard.Else != nil || len(guard.Body.List) != 2 {
		return out
	}
	g, ok := guard.Cond.(*ast.BinaryExpr)
	if !ok || g.Op != token.NEQ || id(g.X) != errname || id(g.Y) != "nil" || !errorCall(guard.Body.List[0], param) {
		return out
	}
	nilIdentifier, literalNil := g.Y.(*ast.Ident)
	if !literalNil || nilIdentifier.Obj != nil {
		return out
	}
	branch, ok := guard.Body.List[1].(*ast.BranchStmt)
	if !ok || branch.Tok != token.CONTINUE || branch.Label != nil {
		return out
	}
	compare, ok := loop.Body.List[2].(*ast.IfStmt)
	if !ok || compare.Init != nil || compare.Else != nil || len(compare.Body.List) != 1 || !errorCall(compare.Body.List[0], param) {
		return out
	}
	c, ok := compare.Cond.(*ast.BinaryExpr)
	if !ok || c.Op != token.NEQ || id(c.X) != got {
		return out
	}
	expected := field(c.Y, r)
	if expected == "" {
		return out
	}
	ii, ei := -1, -1
	for i, f := range fields {
		if f == input {
			ii = i
		}
		if f == expected {
			ei = i
		}
	}
	if ii < 0 || ei < 0 || ii == ei || types[ii] != "string" || types[ei] != "uint64" {
		return out
	}
	for _, e := range table.Elts {
		row, ok := e.(*ast.CompositeLit)
		if !ok || len(row.Elts) != len(fields) {
			return []Binding{}
		}
		il, ok := row.Elts[ii].(*ast.BasicLit)
		if !ok || il.Kind != token.STRING {
			return []Binding{}
		}
		el, ok := row.Elts[ei].(*ast.BasicLit)
		if !ok || el.Kind != token.INT {
			return []Binding{}
		}
		for _, v := range row.Elts {
			if _, keyed := v.(*ast.KeyValueExpr); keyed {
				return []Binding{}
			}
		}
		out = append(out, Binding{Row: ref(row), Input: ref(il), Expected: ref(el), Call: ref(call), Condition: ref(c), Function: fn.Name.Name, Column: ii, Guard: ref(g), Continue: ref(branch), Path: "error is nil when value comparison is reached"})
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
