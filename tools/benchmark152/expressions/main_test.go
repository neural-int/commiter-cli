package main

import (
	"strings"
	"testing"
)

func TestVariableAndSiblingArguments(t *testing.T) {
	s := "package p\nfunc Check(){ got:=Value(1); if got!=4 { fail() }; log(Value(1),\"new\") }"
	rows, e := observe(s)
	if e != nil {
		t.Fatal(e)
	}
	for _, term := range []string{"4", "new"} {
		pos := strings.Index(s, term)
		var best *row
		for i := range rows {
			r := &rows[i]
			if r.Start <= pos && pos < r.End && (best == nil || r.End-r.Start < best.End-best.Start) {
				best = r
			}
		}
		if best == nil {
			t.Fatal("missing expression")
		}
		has := false
		for _, c := range best.Calls {
			has = has || c == "Value"
		}
		if has != (term == "4") {
			t.Fatalf("%s calls %+v", term, best.Calls)
		}
	}
}
