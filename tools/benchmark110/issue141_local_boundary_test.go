package main

import (
	"context"
	"testing"
)

func TestIssue141BoundaryCandidateCoverage(t *testing.T) {
	items := append(issue140AtomicityFixtures(), issue141GuardrailFixtures()...)
	for _, item := range items {
		t.Run(item.name, func(t *testing.T) {
			prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range prepared.Document.Files {
				if file.RawDiff == "" && file.Summary == "" {
					t.Fatalf("%s has no diff or summary evidence", file.ID)
				}
			}
			pairs, err := issue141BoundaryCandidates(prepared.Document, ids)
			if err != nil {
				t.Fatal(err)
			}
			truePairs, falsePairs, total, connected := issue141BoundaryCoverage(ids, pairs, item.reference)
			if !connected || truePairs+falsePairs != len(pairs) {
				t.Fatalf("candidate coverage: %d true, %d false, %d total; connected=%t", truePairs, falsePairs, total, connected)
			}
			if len(ids) <= 4 && len(pairs) != len(ids)*(len(ids)-1)/2 {
				t.Fatalf("small fixture has %d pairs, want full coverage", len(pairs))
			}
			if item.name == "mixed_24" && (len(pairs) >= 24*23/2 || truePairs == 0 || falsePairs == 0) {
				t.Fatalf("24-file candidates are not bounded and mixed: %d, %d, %d", len(pairs), truePairs, falsePairs)
			}
		})
	}
}

func TestIssue141BoundaryValidationAndTransitiveConflict(t *testing.T) {
	pairs := []issue141Pair{{"F001", "F002"}, {"F001", "F003"}, {"F002", "F003"}}
	for _, tc := range []struct{ body, want string }{
		{`{"same_intent":{"P001":true,"P002":false,"P003":true}}`, ""},
		{`{"same_intent":{"P001":true,"P003":true}}`, "invalid_pair_decisions"},
		{`{"same_intent":{"P001":true,"P002":false,"P004":true}}`, "invalid_pair_decisions"},
		{`{"same_intent":{"P001":true,"P002":false,"P002":true}}`, "duplicate_json_key"},
		{`{"same_intent":{"P001":true,"P002":null,"P003":true}}`, "invalid_pair_decisions"},
	} {
		decisions, failure := issue141ValidateBoundary([]byte(tc.body), pairs, 0)
		if failure != tc.want {
			t.Fatalf("%s: failure %q, want %q", tc.body, failure, tc.want)
		}
		if failure == "" {
			groups, negativeClosure := issue141BoundaryPartition([]string{"F001", "F002", "F003"}, pairs, decisions)
			if len(groups) != 1 || negativeClosure != 1 {
				t.Fatalf("transitive closure: groups=%v, negative closure=%d", groups, negativeClosure)
			}
		}
	}
}
