package main

import (
	"strings"
	"testing"
)

func TestAssertionAndDiagnosticScopes(t *testing.T) {
	s := "package p\nfunc Check(){ if Value(1)!=4 { fail() }; log(\"changed\") }"
	rows, e := observe(s)
	if e != nil {
		t.Fatal(e)
	}
	pos := strings.Index(s, "changed")
	found := false
	for _, r := range rows {
		if r.Start <= pos && pos < r.End {
			found = true
			for _, c := range r.Calls {
				if c == "Value" {
					t.Fatal("diagnostic inherited assertion call")
				}
			}
		}
	}
	if !found {
		t.Fatal("missing diagnostic statement")
	}
}
