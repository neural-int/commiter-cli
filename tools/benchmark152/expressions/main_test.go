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

func TestReassignedDefinitionsFailClosed(t *testing.T) {
	for _, body := range []string{"got:=Value(1); got=8; if got!=8 { fail() }", "got:=Value(1); if flag { got=8 }; if got!=8 { fail() }", "got:=Value(1); got++; if got!=8 { fail() }"} {
		src := "package p\nfunc Check(flag bool){" + body + "}"
		rows, err := observe(src)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, r := range rows {
			if src[r.Start:r.End] == "got!=8" {
				found = true
				if r.Status != "partial_unknown" || len(r.Calls) != 0 {
					t.Fatalf("unsafe definition %+v", r)
				}
			}
		}
		if !found {
			t.Fatal("missing comparison")
		}
	}
}
