package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestSameFileSymbolsAndSelfCall(t *testing.T) {
	input := `[ {"ID":"F1","Path":"calc/value.go","Content":"package calc\nfunc Base(n int) int { if n > 0 { return Base(n-1) }; return n }\nfunc Public(n int) int { return Base(n) }\nfunc Other(n int) int { return Base(n) }\n"} ]`
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("graph failed: %v %s", err, out)
	}
	var graph struct {
		Edges []edge `json:"edges"`
	}
	if err := json.Unmarshal(out, &graph); err != nil {
		t.Fatal(err)
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("want two distinct symbol calls without self-call, got %+v", graph.Edges)
	}
	for _, e := range graph.Edges {
		if e.From != "F1" || e.To != "F1" || e.Callee != "Base" || (e.Caller != "Public" && e.Caller != "Other") {
			t.Fatalf("unexpected edge %+v", e)
		}
	}
}
