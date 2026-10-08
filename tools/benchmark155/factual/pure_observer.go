package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"path"
	"strconv"
	"strings"
)

var errUnknownObservation = errors.New("unknown_observation")

// A deliberately small, non-executing observer. Integral values are restricted
// to +/-2^26, so supported arithmetic is exact without overflow. No filesystem,
// process, environment, network, mutation or arbitrary Go function is available.
type pureValue struct {
	kind    string
	number  float64
	text    string
	boolean bool
	data    []byte
}
type pureFunction struct {
	node    *ast.FuncDecl
	pkg     string
	imports map[string]string
	fileID  string
	version string
	source  string
	fs      *token.FileSet
}
type pureObserver struct {
	functions  map[string][]pureFunction
	steps      int
	trace      []Witness
	arrayTypes map[string][]int
}

func newPureObserver(files []Snapshot, version string) *pureObserver {
	o := &pureObserver{functions: map[string][]pureFunction{}, steps: 1024, arrayTypes: map[string][]int{}}
	for _, file := range files {
		if strings.HasSuffix(file.Path, "_test.go") || !strings.HasSuffix(file.Path, ".go") {
			continue
		}
		source := file.Before
		if version == "after" {
			source = file.After
		}
		fs := token.NewFileSet()
		tree, err := parser.ParseFile(fs, file.Path, source, 0)
		if err != nil {
			continue
		}
		imports := map[string]string{}
		for _, i := range tree.Imports {
			p, err := strconv.Unquote(i.Path.Value)
			if err != nil {
				continue
			}
			alias := path.Base(p)
			if i.Name != nil {
				alias = i.Name.Name
			}
			imports[alias] = p
		}
		pkg := path.Dir(file.Path)
		for _, d := range tree.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil {
				key := pkg + "/" + fn.Name.Name
				o.functions[key] = append(o.functions[key], pureFunction{node: fn, pkg: pkg, imports: imports, fileID: file.ID, version: version, source: source, fs: fs})
			}
			if gen, ok := d.(*ast.GenDecl); ok {
				for _, spec := range gen.Specs {
					if typ, ok := spec.(*ast.TypeSpec); ok {
						if arr, ok := typ.Type.(*ast.ArrayType); ok {
							size, ok := arr.Len.(*ast.BasicLit)
							elem, yes := arr.Elt.(*ast.Ident)
							if ok && yes && (elem.Name == "byte" || elem.Name == "uint8") {
								n, err := strconv.Atoi(size.Value)
								if err == nil && n >= 1 && n <= 64 {
									key := pkg + "/" + typ.Name.Name
									o.arrayTypes[key] = append(o.arrayTypes[key], n)
								}
							}
						}
					}
				}
			}
		}
	}
	return o
}
func (o *pureObserver) coerce(v pureValue, typ, pkg string) (pureValue, error) {
	if n := o.arrayTypes[pkg+"/"+typ]; len(n) == 1 && v.kind == "byte_array" && len(v.data) == n[0] {
		return v, nil
	}
	return coercePure(v, typ)
}
func numeric(v float64, kind string) (pureValue, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) || kind == "int" && (math.Abs(v) > 1<<26 || math.Trunc(v) != v) {
		return pureValue{}, errUnknownObservation
	}
	return pureValue{kind: kind, number: v}, nil
}
func (v pureValue) observed() any {
	switch v.kind {
	case "bool":
		return v.boolean
	case "string":
		return v.text
	default:
		return v.number
	}
}
func (o *pureObserver) call(key string, args []pureValue, depth int) (pureValue, error) {
	if depth > 16 || o.steps <= 0 || len(o.functions[key]) != 1 {
		return pureValue{}, errUnknownObservation
	}
	o.steps--
	f := o.functions[key][0]
	n := f.node
	if n.Type.Params == nil || n.Type.Results == nil || len(n.Type.Results.List) != 1 || n.Body == nil {
		return pureValue{}, errUnknownObservation
	}
	env := map[string]pureValue{}
	index := 0
	for _, p := range n.Type.Params.List {
		typ, ok := p.Type.(*ast.Ident)
		if !ok || len(p.Names) == 0 {
			return pureValue{}, errUnknownObservation
		}
		for _, name := range p.Names {
			if index >= len(args) {
				return pureValue{}, errUnknownObservation
			}
			if (typ.Name == "int" || typ.Name == "int64") && args[index].kind != "int" {
				return pureValue{}, errUnknownObservation
			}
			v, err := o.coerce(args[index], typ.Name, f.pkg)
			if err != nil {
				return pureValue{}, err
			}
			env[name.Name] = v
			index++
		}
	}
	if index != len(args) {
		return pureValue{}, errUnknownObservation
	}
	v, returned, err := o.block(n.Body, env, f, depth)
	if err != nil || !returned {
		return pureValue{}, errUnknownObservation
	}
	ret, ok := n.Type.Results.List[0].Type.(*ast.Ident)
	if !ok {
		return pureValue{}, errUnknownObservation
	}
	if (ret.Name == "int" || ret.Name == "int64") && v.kind != "int" {
		return pureValue{}, errUnknownObservation
	}
	return o.coerce(v, ret.Name, f.pkg)
}
func coercePure(v pureValue, typ string) (pureValue, error) {
	switch typ {
	case "int", "int64":
		if v.kind == "int" || v.kind == "float" {
			return numeric(math.Trunc(v.number), "int")
		}
	case "float64":
		if v.kind == "int" || v.kind == "float" {
			return numeric(v.number, "float")
		}
	case "string", "bool":
		if v.kind == typ && (typ != "string" || len(v.text) <= 4096) {
			return v, nil
		}
	}
	return pureValue{}, errUnknownObservation
}
func (o *pureObserver) block(b *ast.BlockStmt, env map[string]pureValue, f pureFunction, depth int) (pureValue, bool, error) {
	for _, stmt := range b.List {
		o.steps--
		if o.steps < 0 {
			return pureValue{}, false, errUnknownObservation
		}
		switch s := stmt.(type) {
		case *ast.ReturnStmt:
			if len(s.Results) != 1 {
				return pureValue{}, false, errUnknownObservation
			}
			v, err := o.expr(s.Results[0], env, f, depth)
			if err == nil {
				o.trace = append(o.trace, witness(f, s, "return"))
			}
			return v, true, err
		case *ast.IfStmt:
			if s.Init != nil || s.Else != nil {
				return pureValue{}, false, errUnknownObservation
			}
			v, err := o.expr(s.Cond, env, f, depth)
			if err != nil || v.kind != "bool" {
				return pureValue{}, false, errUnknownObservation
			}
			o.trace = append(o.trace, witness(f, s.Cond, "condition"))
			if v.boolean {
				v, done, err := o.block(s.Body, env, f, depth)
				if err != nil || done {
					return v, done, err
				}
			}
		default:
			return pureValue{}, false, errUnknownObservation
		}
	}
	return pureValue{}, false, nil
}
func (o *pureObserver) expr(e ast.Expr, env map[string]pureValue, f pureFunction, depth int) (pureValue, error) {
	o.steps--
	if o.steps < 0 {
		return pureValue{}, errUnknownObservation
	}
	switch x := e.(type) {
	case *ast.ParenExpr:
		return o.expr(x.X, env, f, depth)
	case *ast.BasicLit:
		switch x.Kind {
		case token.STRING:
			v, err := strconv.Unquote(x.Value)
			if len(v) > 4096 {
				return pureValue{}, errUnknownObservation
			}
			return pureValue{kind: "string", text: v}, err
		case token.INT:
			v, err := strconv.ParseInt(x.Value, 0, 64)
			if err != nil {
				return pureValue{}, errUnknownObservation
			}
			return numeric(float64(v), "int")
		case token.FLOAT:
			v, err := strconv.ParseFloat(x.Value, 64)
			if err != nil {
				return pureValue{}, errUnknownObservation
			}
			return numeric(v, "float")
		}
	case *ast.Ident:
		if x.Name == "true" || x.Name == "false" {
			return pureValue{kind: "bool", boolean: x.Name == "true"}, nil
		}
		if v, ok := env[x.Name]; ok {
			return v, nil
		}
	case *ast.UnaryExpr:
		v, err := o.expr(x.X, env, f, depth)
		if err != nil {
			return v, err
		}
		if x.Op == token.NOT && v.kind == "bool" {
			v.boolean = !v.boolean
			return v, nil
		}
		if x.Op == token.SUB && (v.kind == "int" || v.kind == "float") {
			return numeric(-v.number, v.kind)
		}
	case *ast.BinaryExpr:
		// Go evaluates untyped floating constant expressions with arbitrary
		// precision. This observer intentionally does not approximate them.
		if constant, floating := pureConstantExpression(x); constant && floating {
			return pureValue{}, errUnknownObservation
		}
		a, err := o.expr(x.X, env, f, depth)
		if err != nil {
			return a, err
		}
		if a.kind == "bool" && (x.Op == token.LAND && !a.boolean || x.Op == token.LOR && a.boolean) {
			return a, nil
		}
		b, err := o.expr(x.Y, env, f, depth)
		if err != nil {
			return b, err
		}
		if a.kind == "bool" && b.kind == "bool" {
			switch x.Op {
			case token.LAND:
				return pureValue{kind: "bool", boolean: a.boolean && b.boolean}, nil
			case token.LOR:
				return pureValue{kind: "bool", boolean: a.boolean || b.boolean}, nil
			case token.EQL:
				return pureValue{kind: "bool", boolean: a.boolean == b.boolean}, nil
			case token.NEQ:
				return pureValue{kind: "bool", boolean: a.boolean != b.boolean}, nil
			}
		}
		if a.kind == "string" && b.kind == "string" {
			switch x.Op {
			case token.ADD:
				if len(a.text)+len(b.text) > 4096 {
					return pureValue{}, errUnknownObservation
				}
				return pureValue{kind: "string", text: a.text + b.text}, nil
			case token.EQL:
				return pureValue{kind: "bool", boolean: a.text == b.text}, nil
			case token.NEQ:
				return pureValue{kind: "bool", boolean: a.text != b.text}, nil
			}
		}
		if (a.kind == "int" || a.kind == "float") && (b.kind == "int" || b.kind == "float") {
			kind := "int"
			if a.kind == "float" || b.kind == "float" {
				kind = "float"
			}
			var result bool
			switch x.Op {
			case token.ADD:
				return numeric(a.number+b.number, kind)
			case token.SUB:
				return numeric(a.number-b.number, kind)
			case token.MUL:
				return numeric(a.number*b.number, kind)
			case token.QUO:
				if b.number == 0 {
					return pureValue{}, errUnknownObservation
				}
				v := a.number / b.number
				if kind == "int" {
					v = math.Trunc(v)
				}
				return numeric(v, kind)
			case token.EQL:
				result = a.number == b.number
			case token.NEQ:
				result = a.number != b.number
			case token.GTR:
				result = a.number > b.number
			case token.GEQ:
				result = a.number >= b.number
			case token.LSS:
				result = a.number < b.number
			case token.LEQ:
				result = a.number <= b.number
			default:
				return pureValue{}, errUnknownObservation
			}
			return pureValue{kind: "bool", boolean: result}, nil
		}
	case *ast.SliceExpr:
		if x.Low != nil || x.High != nil || x.Max != nil {
			return pureValue{}, errUnknownObservation
		}
		v, err := o.expr(x.X, env, f, depth)
		if err != nil || v.kind != "byte_array" {
			return pureValue{}, errUnknownObservation
		}
		v.kind = "byte_slice"
		return v, nil
	case *ast.CallExpr:
		args := []pureValue{}
		for _, arg := range x.Args {
			v, err := o.expr(arg, env, f, depth)
			if err != nil {
				return v, err
			}
			args = append(args, v)
		}
		if id, ok := x.Fun.(*ast.Ident); ok {
			if _, shadow := env[id.Name]; shadow {
				return pureValue{}, errUnknownObservation
			}
			if id.Name == "int" || id.Name == "int64" || id.Name == "float64" {
				if len(args) != 1 {
					return pureValue{}, errUnknownObservation
				}
				return coercePure(args[0], id.Name)
			}
			return o.call(f.pkg+"/"+id.Name, args, depth+1)
		}
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
			if alias, ok := sel.X.(*ast.Ident); ok {
				if _, shadow := env[alias.Name]; shadow {
					return pureValue{}, errUnknownObservation
				}
				pkg := f.imports[alias.Name]
				if strings.HasPrefix(pkg, "fixture/") {
					return o.call(strings.TrimPrefix(pkg, "fixture/")+"/"+sel.Sel.Name, args, depth+1)
				}
				if len(args) == 2 && pkg == "bytes" && sel.Sel.Name == "Compare" && args[0].kind == "byte_slice" && args[1].kind == "byte_slice" {
					return numeric(float64(bytes.Compare(args[0].data, args[1].data)), "int")
				}
				if len(args) == 1 && pkg == "math" && (args[0].kind == "float" || args[0].kind == "int") {
					switch sel.Sel.Name {
					case "Floor":
						return numeric(math.Floor(args[0].number), "float")
					case "Round":
						return numeric(math.Round(args[0].number), "float")
					}
				}
				if len(args) == 1 && pkg == "strings" && args[0].kind == "string" {
					switch sel.Sel.Name {
					case "ToLower":
						return pureValue{kind: "string", text: strings.ToLower(args[0].text)}, nil
					case "TrimSpace":
						return pureValue{kind: "string", text: strings.TrimSpace(args[0].text)}, nil
					}
				}
			}
		}
	}
	return pureValue{}, errUnknownObservation
}

func pureConstantExpression(e ast.Expr) (bool, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		return true, x.Kind == token.FLOAT
	case *ast.ParenExpr:
		return pureConstantExpression(x.X)
	case *ast.UnaryExpr:
		return pureConstantExpression(x.X)
	case *ast.BinaryExpr:
		a, af := pureConstantExpression(x.X)
		b, bf := pureConstantExpression(x.Y)
		return a && b, af || bf
	}
	return false, false
}
