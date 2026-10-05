package main

import (
	"go/ast"
	"go/parser"
	"go/token"
)

type declarationFact struct {
	File    string `json:"file_id"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Value   string `json:"observed_value"`
}

func declarationFacts(f fixture) []declarationFact {
	facts := []declarationFact{}
	for _, after := range []bool{false, true} {
		version := "before"
		if after {
			version = "after"
		}
		for _, file := range f.Files {
			if file.NewPath == nil {
				continue
			}
			tree, e := parser.ParseFile(token.NewFileSet(), *file.NewPath, versionSource(file.RawDiff, after), 0)
			if e != nil {
				continue
			}
			for _, decl := range tree.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Body != nil {
						facts = append(facts, declarationFact{file.ID, version, "function", d.Name.Name, expressionText(&ast.FuncLit{Type: d.Type, Body: d.Body})})
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						if v, ok := spec.(*ast.ValueSpec); ok && len(v.Names) == len(v.Values) {
							for i, name := range v.Names {
								facts = append(facts, declarationFact{file.ID, version, d.Tok.String(), name.Name, expressionText(v.Values[i])})
							}
						}
					}
				}
			}
		}
	}
	return facts
}
