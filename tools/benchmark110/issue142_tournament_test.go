package main

import (
	"context"
	"testing"
)

func TestIssue142ExpandedCandidatesAreBoundedCompleteAndRepeatable(t *testing.T) {
	for _, item := range issue142VerificationFixtures() {
		prepared, ids, baseline, err := issue142DiagnosticInputs(context.Background(), item)
		if err != nil {
			t.Fatal(err)
		}
		first, err := issue142ExpandCandidates(prepared.Document, ids, baseline)
		if err != nil {
			t.Fatal(err)
		}
		second, err := issue142ExpandCandidates(prepared.Document, ids, baseline)
		if err != nil {
			t.Fatal(err)
		}
		if len(first) < 2 || len(first) > issue142TournamentCap || len(first) != len(second) {
			t.Fatalf("%s: unexpected candidate count %d", item.name, len(first))
		}
		seen := map[string]bool{}
		for i, candidate := range first {
			_, key, err := issue142Canonical(candidate.Groups, ids)
			if err != nil || seen[key] || candidate.ID != second[i].ID || !sameGroups(candidate.Groups, second[i].Groups) {
				t.Fatalf("%s: invalid or unstable candidate %d: %v", item.name, i, err)
			}
			seen[key] = true
		}
	}
}
