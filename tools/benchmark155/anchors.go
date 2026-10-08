// Package main extracts syntax observations only; it never assigns commit intent.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"sort"
	"strings"
)

type Snapshot struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type Request struct {
	Files []Snapshot `json:"files"`
}
type Evidence struct {
	ID        string   `json:"id"`
	File      string   `json:"file"`
	Version   string   `json:"version"`
	Kind      string   `json:"kind"`
	Function  string   `json:"function"`
	Span      [2]int   `json:"span"`
	Text      string   `json:"text"`
	SourceSHA string   `json:"source_sha256"`
	Syntax    string   `json:"syntax"`
	Changed   bool     `json:"changed"`
	Calls     []string `json:"calls,omitempty"`
	// Parts of one || condition remain a single observed failure condition.
	ConditionSpan [2]int `json:"condition_span"`
}
type Status struct {
	File    string `json:"file"`
	Version string `json:"version"`
	Status  string `json:"status"`
}
type Response struct {
	Evidence []Evidence `json:"evidence"`
	Status   []Status   `json:"status"`
}

func sha(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func syntax(n ast.Node) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, token.NewFileSet(), n)
	return b.String()
}
func observation(fs *token.FileSet, src, fid, version, kind, fn string, n ast.Node, condition ast.Node) Evidence {
	span := [2]int{fs.Position(n.Pos()).Offset, fs.Position(n.End()).Offset}
	cspan := span
	if condition != nil {
		cspan = [2]int{fs.Position(condition.Pos()).Offset, fs.Position(condition.End()).Offset}
	}
	digest := sha(src)
	return Evidence{ID: sha(fmt.Sprintf("%s:%s:%s:%s:%v", fid, version, digest, kind, span))[:24], File: fid, Version: version, Kind: kind, Function: fn, Span: span, Text: src[span[0]:span[1]], SourceSHA: digest, Syntax: syntax(n), ConditionSpan: cspan}
}
func fatal(stmt *ast.IfStmt, receiver string) bool {
	if len(stmt.Body.List) != 1 || stmt.Else != nil {
		return false
	}
	expression, ok := stmt.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expression.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := selector.X.(*ast.Ident)
	return ok && id.Name == receiver && (selector.Sel.Name == "Fatal" || selector.Sel.Name == "Fatalf" || selector.Sel.Name == "Error" || selector.Sel.Name == "Errorf")
}
func leaves(expr ast.Expr) []ast.Expr {
	if x, ok := expr.(*ast.ParenExpr); ok {
		return leaves(x.X)
	}
	if x, ok := expr.(*ast.BinaryExpr); ok && x.Op == token.LOR {
		return append(leaves(x.X), leaves(x.Y)...)
	}
	return []ast.Expr{expr}
}
func extractOne(file Snapshot, version, src string) ([]Evidence, Status) {
	status := Status{file.ID, version, "parsed"}
	out := []Evidence{}
	if !strings.HasSuffix(file.Path, ".go") {
		status.Status = "unsupported_language"
		return out, status
	}
	fs := token.NewFileSet()
	tree, err := parser.ParseFile(fs, file.Path, src, 0)
	if err != nil {
		status.Status = "parse_error"
		return out, status
	}
	for _, decl := range tree.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			if gen, yes := decl.(*ast.GenDecl); yes {
				for _, spec := range gen.Specs {
					kind, name := "", ""
					switch node := spec.(type) {
					case *ast.TypeSpec:
						kind, name = "type_source", node.Name.Name
					case *ast.ValueSpec:
						kind = "binding_source"
						for _, id := range node.Names {
							name += id.Name + ","
						}
					case *ast.ImportSpec:
						kind, name = "import_source", node.Path.Value
					}
					if kind != "" {
						out = append(out, observation(fs, src, file.ID, version, kind, name, spec, nil))
					}
				}
			}
			continue
		}
		if fn.Body == nil {
			continue
		}
		item := observation(fs, src, file.ID, version, "function_source", fn.Name.Name, fn, nil)
		if fn.Recv == nil {
			callSet := map[string]bool{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if id, plain := call.Fun.(*ast.Ident); plain && (id.Obj == nil || id.Obj.Kind == ast.Fun) {
						callSet[id.Name] = true
					}
				}
				return true
			})
			for name := range callSet {
				item.Calls = append(item.Calls, name)
			}
			sort.Strings(item.Calls)
		}
		out = append(out, item)
		if !strings.HasSuffix(file.Path, "_test.go") || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Type.Params == nil {
			continue
		}
		receiver := ""
		for _, param := range fn.Type.Params.List {
			star, ok := param.Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			selector, ok := star.X.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && pkg.Name == "testing" && selector.Sel.Name == "T" && len(param.Names) == 1 {
				receiver = param.Names[0].Name
			}
		}
		if receiver == "" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if _, nested := n.(*ast.FuncLit); nested {
				return false
			} // nested test receivers require separate support
			stmt, ok := n.(*ast.IfStmt)
			if !ok || !fatal(stmt, receiver) {
				return true
			}
			for _, leaf := range leaves(stmt.Cond) {
				out = append(out, observation(fs, src, file.ID, version, "failure_predicate", fn.Name.Name, leaf, stmt.Cond))
			}
			return true
		})
	}
	return out, status
}
func Extract(request Request) (Response, error) {
	response := Response{Evidence: []Evidence{}, Status: []Status{}}
	seen := map[string]bool{}
	if len(request.Files) == 0 || len(request.Files) > 32 {
		return response, fmt.Errorf("file_budget")
	}
	for _, file := range request.Files {
		if file.ID == "" || seen[file.ID] {
			return response, fmt.Errorf("duplicate_or_empty_file_id")
		}
		seen[file.ID] = true
		if len(file.Before) > 65536 || len(file.After) > 65536 {
			return response, fmt.Errorf("source_budget")
		}
		old, os := extractOne(file, "before", file.Before)
		new, ns := extractOne(file, "after", file.After)
		// Multiset cancellation uses syntax only. It asserts no semantic equivalence.
		used := make([]bool, len(new))
		for i := range old {
			old[i].Changed = true
			for j := range new {
				if !used[j] && old[i].Kind == new[j].Kind && old[i].Function == new[j].Function && old[i].Syntax == new[j].Syntax {
					used[j] = true
					old[i].Changed = false
					break
				}
			}
		}
		for j := range new {
			new[j].Changed = !used[j]
		}
		response.Evidence = append(response.Evidence, old...)
		response.Evidence = append(response.Evidence, new...)
		response.Status = append(response.Status, os, ns)
	}
	sort.Slice(response.Evidence, func(i, j int) bool { return response.Evidence[i].ID < response.Evidence[j].ID })
	sort.Slice(response.Status, func(i, j int) bool {
		a, b := response.Status[i], response.Status[j]
		return a.File+":"+a.Version < b.File+":"+b.Version
	})
	return response, nil
}
func main() {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 5*1024*1024+1))
	if err != nil || len(raw) > 5*1024*1024 {
		fmt.Fprintln(os.Stderr, "input_budget")
		os.Exit(1)
	}
	var req Request
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "invalid_input")
		os.Exit(1)
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		fmt.Fprintln(os.Stderr, "trailing_input")
		os.Exit(1)
	}
	response, err := Extract(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = json.NewEncoder(os.Stdout).Encode(response); err != nil {
		os.Exit(1)
	}
}
