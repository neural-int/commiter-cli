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
	Row, Input, Expected, Call, Condition         Ref
	Function                                      string
	Column                                        int
	Guard, ArgumentDefinition, ArgumentAssignment Ref
	ArgumentType, SelectedArgument                string
	GuardTaken                                    bool
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
func index(e ast.Expr, base string, col string) bool {
	x, ok := e.(*ast.IndexExpr)
	if !ok || id(x.X) != base {
		return false
	}
	n, ok := x.Index.(*ast.BasicLit)
	return ok && n.Kind == token.INT && n.Value == col
}
func assign(s ast.Stmt, expected token.Token) (*ast.AssignStmt, bool) {
	a, ok := s.(*ast.AssignStmt)
	return a, ok && a.Tok == expected && len(a.Lhs) == 1 && len(a.Rhs) == 1
}
func diagnosticExpr(e ast.Expr, row string) bool {
	switch x := e.(type) {
	case *ast.BasicLit:
		return x.Kind == token.STRING
	case *ast.IndexExpr:
		return index(x, row, "2")
	case *ast.BinaryExpr:
		return x.Op == token.ADD && diagnosticExpr(x.X, row) && diagnosticExpr(x.Y, row)
	}
	return false
}
func inspect(tree *ast.File, fn *ast.FuncDecl, ref func(ast.Node) Ref) []Binding {
	out := []Binding{}
	param := parameter(tree, fn)
	if param == "" || fn.Body == nil || len(fn.Body.List) != 2 {
		return out
	}
	d, ok := assign(fn.Body.List[0], token.DEFINE)
	if !ok {
		return out
	}
	table, ok := d.Rhs[0].(*ast.CompositeLit)
	if !ok {
		return out
	}
	cases := id(d.Lhs[0])
	if cases == "" || cases == param {
		return out
	}
	outer, ok := table.Type.(*ast.ArrayType)
	if !ok || outer.Len != nil {
		return out
	}
	inner, ok := outer.Elt.(*ast.ArrayType)
	if !ok || inner.Len != nil || id(inner.Elt) != "string" {
		return out
	}
	loop, ok := fn.Body.List[1].(*ast.RangeStmt)
	if !ok || loop.Tok != token.DEFINE || id(loop.X) != cases || id(loop.Key) != "_" {
		return out
	}
	r := id(loop.Value)
	if r == "" || r == param || r == cases {
		return out
	}
	stmts := loop.Body.List
	if len(stmts) != 6 && len(stmts) != 7 {
		return out
	}
	input, ok := assign(stmts[0], token.DEFINE)
	if !ok || !index(input.Rhs[0], r, "0") {
		return out
	}
	expected, ok := assign(stmts[1], token.DEFINE)
	if !ok || !index(expected.Rhs[0], r, "1") {
		return out
	}
	decl, ok := stmts[2].(*ast.DeclStmt)
	if !ok {
		return out
	}
	gen, ok := decl.Decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.VAR || len(gen.Specs) != 1 {
		return out
	}
	spec, ok := gen.Specs[0].(*ast.ValueSpec)
	if !ok || len(spec.Names) != 1 || len(spec.Values) != 0 {
		return out
	}
	arg := spec.Names[0].Name
	typ := id(spec.Type)
	if typ != "string" && typ != "uint8" {
		return out
	}
	offset := 3
	if typ == "string" {
		if len(stmts) != 7 {
			return out
		}
		init, ok := assign(stmts[offset], token.ASSIGN)
		if !ok || id(init.Lhs[0]) != arg {
			return out
		}
		lit, ok := init.Rhs[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING || lit.Value != "\"\"" {
			return out
		}
		offset++
	} else if len(stmts) != 6 {
		return out
	}
	guard, ok := stmts[offset].(*ast.IfStmt)
	if !ok || guard.Init != nil || guard.Else != nil || len(guard.Body.List) != 1 {
		return out
	}
	cmp, ok := guard.Cond.(*ast.BinaryExpr)
	if !ok || cmp.Op != token.EQL {
		return out
	}
	lenCall, ok := cmp.X.(*ast.CallExpr)
	if !ok || id(lenCall.Fun) != "len" || len(lenCall.Args) != 1 || id(lenCall.Args[0]) != r {
		return out
	}
	lenIdent, ok := lenCall.Fun.(*ast.Ident)
	if !ok || lenIdent.Obj != nil {
		return out
	}
	three, ok := cmp.Y.(*ast.BasicLit)
	if !ok || three.Kind != token.INT || three.Value != "3" {
		return out
	}
	set, ok := assign(guard.Body.List[0], token.ASSIGN)
	if !ok || id(set.Lhs[0]) != arg {
		return out
	}
	if typ == "string" {
		if !index(set.Rhs[0], r, "2") {
			return out
		}
	} else {
		byteIndex, ok := set.Rhs[0].(*ast.IndexExpr)
		if !ok || !index(byteIndex.X, r, "2") {
			return out
		}
		zero, ok := byteIndex.Index.(*ast.BasicLit)
		if !ok || zero.Kind != token.INT || zero.Value != "0" {
			return out
		}
	}
	result, ok := assign(stmts[offset+1], token.DEFINE)
	if !ok {
		return out
	}
	call, ok := result.Rhs[0].(*ast.CallExpr)
	if !ok || id(call.Fun) == "" || len(call.Args) != 2 || id(call.Args[0]) != id(input.Lhs[0]) || id(call.Args[1]) != arg || call.Ellipsis.IsValid() {
		return out
	}
	names := []string{param, cases, r, id(input.Lhs[0]), id(expected.Lhs[0]), arg, id(result.Lhs[0])}
	seen := map[string]bool{}
	for _, name := range names {
		if name == "" || seen[name] {
			return out
		}
		seen[name] = true
	}
	failure, ok := stmts[offset+2].(*ast.IfStmt)
	if !ok || failure.Init != nil || failure.Else != nil {
		return out
	}
	condition, ok := failure.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.NEQ || id(condition.X) != id(result.Lhs[0]) || id(condition.Y) != id(expected.Lhs[0]) {
		return out
	}
	if len(failure.Body.List) != 3 {
		return out
	}
	diagnostic, ok := assign(failure.Body.List[0], token.DEFINE)
	if !ok {
		return out
	}
	diagnosticName := id(diagnostic.Lhs[0])
	if diagnosticName == "" || seen[diagnosticName] {
		return out
	}
	initial, ok := diagnostic.Rhs[0].(*ast.BasicLit)
	if !ok || initial.Kind != token.STRING || initial.Value != "\"\"" {
		return out
	}
	diagGuard, ok := failure.Body.List[1].(*ast.IfStmt)
	if !ok || diagGuard.Init != nil || diagGuard.Else != nil || len(diagGuard.Body.List) != 1 || ref(diagGuard.Cond).Text != ref(guard.Cond).Text {
		return out
	}
	diagSet, ok := assign(diagGuard.Body.List[0], token.ASSIGN)
	if !ok || id(diagSet.Lhs[0]) != diagnosticName || !diagnosticExpr(diagSet.Rhs[0], r) {
		return out
	}
	errorStatement, ok := failure.Body.List[2].(*ast.ExprStmt)
	if !ok {
		return out
	}
	errorCall, ok := errorStatement.X.(*ast.CallExpr)
	if !ok || errorCall.Ellipsis.IsValid() {
		return out
	}
	errorSelector, ok := errorCall.Fun.(*ast.SelectorExpr)
	if !ok || id(errorSelector.X) != param || errorSelector.Sel.Name != "Errorf" {
		return out
	}
	for _, e := range table.Elts {
		row, ok := e.(*ast.CompositeLit)
		if !ok || (len(row.Elts) != 2 && len(row.Elts) != 3) {
			return []Binding{}
		}
		values := []string{}
		for _, e := range row.Elts {
			lit, ok := e.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return []Binding{}
			}
			v, err := strconv.Unquote(lit.Value)
			if err != nil {
				return []Binding{}
			}
			values = append(values, v)
		}
		selected := ""
		if typ == "uint8" {
			selected = "0"
		}
		taken := len(values) == 3
		if taken {
			selected = values[2]
			if typ == "uint8" {
				if len(selected) == 0 {
					return []Binding{}
				}
				selected = strconv.Itoa(int([]byte(selected)[0]))
			}
		}
		out = append(out, Binding{Row: ref(row), Input: ref(row.Elts[0]), Expected: ref(row.Elts[1]), Call: ref(call), Condition: ref(condition), Function: fn.Name.Name, Column: 0, Guard: ref(cmp), ArgumentDefinition: ref(decl), ArgumentAssignment: ref(set), ArgumentType: typ, SelectedArgument: selected, GuardTaken: taken})
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
