// This tool interprets a fixed safe Go subset. Repository code is never run.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"sort"
)

type Snapshot struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type Query struct {
	Package   string `json:"package"`
	Function  string `json:"function"`
	Arguments []any  `json:"arguments"`
}
type Request struct {
	Files []Snapshot `json:"files"`
	Query *Query     `json:"query"`
}
type Witness struct {
	ID        string `json:"id"`
	File      string `json:"file"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Span      [2]int `json:"span"`
	Text      string `json:"text"`
	SourceSHA string `json:"source_sha256"`
}
type Fact struct {
	Known      bool     `json:"known"`
	Kind       string   `json:"kind"`
	Value      any      `json:"value"`
	ReturnRefs []string `json:"return_refs"`
}
type Result struct {
	Facts              map[string]Fact `json:"facts"`
	Nodes              []Witness       `json:"nodes"`
	SourceMappingValid bool            `json:"source_mapping_valid"`
}

func hash(raw string) string { sum := sha256.Sum256([]byte(raw)); return hex.EncodeToString(sum[:]) }
func witness(f pureFunction, node ast.Node, kind string) Witness {
	span := [2]int{f.fs.Position(node.Pos()).Offset, f.fs.Position(node.End()).Offset}
	sourceSHA := hash(f.source)
	id := hash(fmt.Sprintf("%s:%s:%s:%s:%v", f.fileID, f.version, sourceSHA, kind, span))[:24]
	return Witness{id, f.fileID, f.version, kind, span, f.source[span[0]:span[1]], sourceSHA}
}
func argument(value any) (pureValue, error) {
	switch v := value.(type) {
	case float64:
		return numeric(v, "int")
	case string:
		if len(v) <= 4096 {
			return pureValue{kind: "string", text: v}, nil
		}
	case bool:
		return pureValue{kind: "bool", boolean: v}, nil
	case []any:
		if len(v) < 1 || len(v) > 64 {
			break
		}
		data := []byte{}
		for _, x := range v {
			n, ok := x.(float64)
			if !ok || n < 0 || n > 255 || n != float64(int(n)) {
				return pureValue{}, errUnknownObservation
			}
			data = append(data, byte(n))
		}
		return pureValue{kind: "byte_array", data: data}, nil
	}
	return pureValue{}, errUnknownObservation
}
func observe(request Request) (Result, error) {
	out := Result{Facts: map[string]Fact{}, Nodes: []Witness{}, SourceMappingValid: true}
	seen := map[string]bool{}
	if len(request.Files) < 1 || len(request.Files) > 16 {
		return out, fmt.Errorf("file_budget")
	}
	for _, file := range request.Files {
		if file.ID == "" || seen[file.ID] {
			return out, fmt.Errorf("duplicate_or_empty_id")
		}
		seen[file.ID] = true
		if len(file.Before) > 65536 || len(file.After) > 65536 {
			return out, fmt.Errorf("source_budget")
		}
	}
	if request.Query != nil && len(request.Query.Arguments) > 4 {
		return out, fmt.Errorf("argument_budget")
	}
	nodeIDs := map[string]bool{}
	for _, version := range []string{"before", "after"} {
		oracle := newPureObserver(request.Files, version)
		for _, defs := range oracle.functions {
			for _, f := range defs {
				ast.Inspect(f.node.Body, func(n ast.Node) bool {
					if _, nested := n.(*ast.FuncLit); nested {
						return false
					}
					kind := ""
					var selected ast.Node
					switch x := n.(type) {
					case *ast.ReturnStmt:
						kind = "return"
						selected = x
					case *ast.IfStmt:
						kind = "condition"
						selected = x.Cond
					}
					if kind != "" {
						item := witness(f, selected, kind)
						if !nodeIDs[item.ID] {
							nodeIDs[item.ID] = true
							out.Nodes = append(out.Nodes, item)
						}
					}
					return true
				})
			}
		}
		state := Fact{Known: false, Kind: "unknown", Value: nil, ReturnRefs: []string{}}
		if request.Query != nil {
			args := []pureValue{}
			valid := true
			for _, a := range request.Query.Arguments {
				v, err := argument(a)
				if err != nil {
					valid = false
					break
				}
				args = append(args, v)
			}
			if valid {
				value, err := oracle.call(request.Query.Package+"/"+request.Query.Function, args, 0)
				if err == nil && (value.kind == "int" || value.kind == "float" || value.kind == "string" || value.kind == "bool") {
					state = Fact{Known: true, Kind: value.kind, Value: value.observed(), ReturnRefs: []string{}}
					for _, step := range oracle.trace {
						if step.Kind == "return" {
							state.ReturnRefs = append(state.ReturnRefs, step.ID)
						}
					}
					sort.Strings(state.ReturnRefs)
				}
			}
		}
		out.Facts[version] = state
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].ID < out.Nodes[j].ID })
	return out, nil
}
func main() {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 4*1024*1024+1))
	if err != nil || len(raw) > 4*1024*1024 {
		fmt.Fprintln(os.Stderr, "input_budget")
		os.Exit(1)
	}
	var req Request
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil {
		fmt.Fprintln(os.Stderr, "invalid_input")
		os.Exit(1)
	}
	if decoder.Decode(new(any)) != io.EOF {
		fmt.Fprintln(os.Stderr, "trailing_input")
		os.Exit(1)
	}
	// Parse-only validation; the bounded observer does not compile or execute files.
	for _, f := range req.Files {
		for _, src := range []string{f.Before, f.After} {
			if _, err := parser.ParseFile(token.NewFileSet(), f.Path, src, 0); err != nil {
				fmt.Fprintln(os.Stderr, "parse_error")
				os.Exit(1)
			}
		}
	}
	result, err := observe(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if json.NewEncoder(os.Stdout).Encode(result) != nil {
		os.Exit(1)
	}
}
