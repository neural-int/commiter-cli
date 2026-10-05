package main

import "testing"

func TestObservedGoCallsBindSelectedDeclarations(t *testing.T) {
	f := contractFixtures()[2]
	calls := callFacts(f)
	for _, pair := range [][2]string{{"F003", "F001"}, {"F005", "F001"}, {"F009", "F007"}, {"F011", "F007"}} {
		for _, version := range []string{"before", "after"} {
			found := false
			for _, c := range calls {
				if c.Caller == pair[0] && c.Callee == pair[1] && c.Version == version && c.Class == "soft" {
					found = true
				}
			}
			if !found {
				t.Fatalf("observed call missing: %v %s", pair, version)
			}
		}
	}
	t.Logf("original extracted edges=%d; observed Go call facts=%d", len(f.Graph.Edges), len(calls))
	for _, edge := range f.Graph.Edges {
		t.Logf("%s -> %s %s", edge.SourceID, edge.TargetID, edge.Kind)
	}
}
func TestAmbiguousDeclarationsDoNotCreateCallEdge(t *testing.T) {
	f := fixtureFromCases("ambiguous", []changeCase{sourceCase("core/a.go", "core", "", "func Work() int{return 1}", "func Work() int{return 2}"), sourceCase("core/b.go", "core", "", "func Work() int{return 1}", "func Work() int{return 2}"), sourceCase("core/c.go", "core", "", "func Caller() int{return Work()}", "func Caller() int{return Work()+1}")}, [][]int{{0}, {1}, {2}})
	if len(callFacts(f)) != 0 {
		t.Fatal("ambiguous declaration chosen")
	}
}
