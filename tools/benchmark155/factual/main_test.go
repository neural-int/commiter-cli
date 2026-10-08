package main

import "testing"

func TestCountPositiveInputHasStableValueAndCorrectReturn(t *testing.T) {
	files := []Snapshot{{"F1", "count.go", "package sample\nfunc Count(n int) int {return n}\n", "package sample\nfunc Count(n int) int {if n<0 {return 0};return n}\n"}}
	result, err := observe(Request{files, &Query{".", "Count", []any{float64(3)}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"before", "after"} {
		fact := result.Facts[version]
		if !fact.Known || fact.Kind != "int" || fact.Value != float64(3) || len(fact.ReturnRefs) != 1 {
			t.Fatal("incorrect value/path")
		}
		for _, node := range result.Nodes {
			if node.ID == fact.ReturnRefs[0] && node.Text != "return n" {
				t.Fatal("wrong return witness")
			}
		}
	}
}
func TestByteArrayAliasCompareWithoutRunningRepositoryCode(t *testing.T) {
	src := "package sample\nimport \"bytes\"\ntype ID [2]byte\nfunc Compare(a,b ID) int {return bytes.Compare(a[:],b[:])}\n"
	result, err := observe(Request{[]Snapshot{{"F1", "a.go", src, src}}, &Query{".", "Compare", []any{[]any{float64(0), float64(1)}, []any{float64(0), float64(2)}}}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Facts["after"].Known || result.Facts["after"].Value != float64(-1) {
		t.Fatal("wrong array result")
	}
}
func TestUnsupportedAndMissingObservationNeverFabricateValue(t *testing.T) {
	files := []Snapshot{{"F1", "a.go", "package sample\nfunc Opaque(n int) int {return Missing(n)}\n", "package sample\nfunc Opaque(n int) int {return Missing(n)}\n"}}
	for _, query := range []*Query{nil, {".", "Opaque", []any{float64(3)}}} {
		result, err := observe(Request{files, query})
		if err != nil {
			t.Fatal(err)
		}
		for _, fact := range result.Facts {
			if fact.Known || fact.Value != nil || len(fact.ReturnRefs) != 0 {
				t.Fatal("unsupported accepted")
			}
		}
	}
}
