package main

import (
	"context"
	"encoding/json"
	"testing"
)

func TestIssue142DifferenceInputKeepsEvidenceAndOnlyChangedBoundaries(t *testing.T) {
	item := issue142VerificationFixtures()[4]
	prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
	if err != nil {
		t.Fatal(err)
	}
	baseline, _, err := issue142Generate(prepared.Document, ids)
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := issue142ExpandCandidates(prepared.Document, ids, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d", len(candidates))
	}
	var firstSchema []byte
	for _, reverse := range []bool{false, true} {
		_, prompt, schema, err := issue142DifferenceInput(prepared, ids, candidates, reverse)
		if err != nil {
			t.Fatal(err)
		}
		if firstSchema == nil {
			firstSchema = schema
		} else if string(firstSchema) != string(schema) {
			t.Fatal("schema changed with candidate order")
		}
		var input struct {
			CandidateIDs    []string                     `json:"candidate_ids"`
			BoundaryChoices []issue142DifferenceBoundary `json:"boundary_choices"`
			RepositoryInput json.RawMessage              `json:"repository_input"`
		}
		if err := json.Unmarshal(prompt, &input); err != nil {
			t.Fatal(err)
		}
		if len(input.BoundaryChoices) != 1 || len(input.BoundaryChoices[0].FileIDs) != 2 {
			t.Fatalf("unexpected differing boundaries: %+v", input.BoundaryChoices)
		}
		if len(input.RepositoryInput) == 0 || string(input.RepositoryInput) == "null" {
			t.Fatal("repository evidence missing")
		}
		if len(input.CandidateIDs) != 2 || input.BoundaryChoices[0].Candidates[0].CandidateID != input.CandidateIDs[0] {
			t.Fatal("candidate order mismatch")
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(prompt, &raw); err != nil {
			t.Fatal(err)
		}
		if _, exists := raw["candidates"]; exists {
			t.Fatal("complete partition candidates leaked into difference arm")
		}
	}
}
