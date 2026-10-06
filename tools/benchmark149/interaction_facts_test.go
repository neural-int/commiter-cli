package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestInteractionFactsMatchOracleAndRetainUnknown(t *testing.T) {
	data, err := os.ReadFile("../../docs/benchmarks/issue-149/iteration-17-runtime-interactions.json")
	if err != nil {
		t.Fatal(err)
	}
	var oracle []interactionObservation
	if err := json.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	matched := 0
	for _, f := range []fixture{contractFixtures()[2], sharedCalleeGuardrail()} {
		facts, m, err := interactionFacts(context.Background(), f)
		if err != nil {
			t.Fatal(err)
		}
		if m.Unknown != 0 || m.Samples != len(facts)*4 {
			t.Fatal("unexpected missing probe")
		}
		for _, fact := range facts {
			for _, s := range fact.Snapshots {
				found := false
				for _, r := range oracle {
					if r.Fixture == f.Name && r.Callee == fact.Callee && r.Caller == fact.Caller && r.Function == fact.Function && r.CalleeAfter == s.CalleeAfter && r.CallerAfter == s.CallerAfter {
						v, err := json.Marshal(s.Value)
						if err != nil {
							t.Fatal(err)
						}
						if !s.Known || string(v) != string(r.Value) {
							t.Fatal("runtime value mismatch")
						}
						found = true
						matched++
						break
					}
				}
				if !found {
					t.Fatal("invented observation")
				}
			}
		}
	}
	if matched != 24 {
		t.Fatal("incomplete oracle coverage")
	}
	f := fixtureFromCases("unknown-probe", []changeCase{
		sourceCase("utility/normalize.go", "utility", "", "func Normalize(s string) string {return Missing(s)}", "func Normalize(s string) string {return Missing(s)}"),
		sourceCase("caller/label.go", "caller", "import \"fixture/utility\"", "func Label(s string) string {return utility.Normalize(s)}", "func Label(s string) string {return utility.Normalize(s)}"),
		testCase("caller/label_test.go", "caller", "Label", "if Label(\"A\")!=\"a\" {t.Fatal(\"value\")}", "if Label(\"A\")!=\"b\" {t.Fatal(\"value\")}"),
	}, nil)
	facts, m, err := interactionFacts(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if m.Probes != 1 || m.Unknown != 4 {
		t.Fatal("unknown probes lost")
	}
	for _, s := range facts[0].Snapshots {
		if s.Known || s.Value != nil {
			t.Fatal("unknown converted to a value")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := interactionFacts(ctx, f); err == nil {
		t.Fatal("cancellation ignored")
	}
}
