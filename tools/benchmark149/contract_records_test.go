package main

import (
	"encoding/json"
	"testing"
)

func TestContractRecordsRetainEveryObservation(t *testing.T) {
	for _, f := range append(fixtures(), contractFixtures()...) {
		c, _, err := canonicalFixture(f)
		if err != nil {
			t.Fatal(err)
		}
		facts := anchorEvidence(c)
		records, looseTests, looseCalls := contractRecords(c, facts)
		if len(records) != len(facts) {
			t.Fatalf("%s: evidence lost", f.Name)
		}
		count := func(values any) map[string]int {
			data, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			var entries []json.RawMessage
			if err := json.Unmarshal(data, &entries); err != nil {
				t.Fatal(err)
			}
			out := map[string]int{}
			for _, e := range entries {
				out[string(e)]++
			}
			return out
		}
		tests, calls := append([]assertionFact{}, looseTests...), append([]callFact{}, looseCalls...)
		for _, r := range records {
			tests = append(tests, r.Tests...)
			calls = append(calls, r.Calls...)
		}
		for _, pair := range [][2]any{{tests, assertionFacts(c)}, {calls, callFacts(c)}} {
			got, want := count(pair[0]), count(pair[1])
			if len(got) != len(want) {
				t.Fatalf("%s: observation lost or invented", f.Name)
			}
			for key, n := range want {
				if got[key] != n {
					t.Fatalf("%s: observation duplicated or lost", f.Name)
				}
			}
		}
	}
}
