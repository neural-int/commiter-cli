package main

import (
	"context"
	"math"
	"path/filepath"
	"testing"
)

func TestIssue142ScoreBreakdownMatchesFrozenScores(t *testing.T) {
	items, err := issue142LoadHoldoutFixtures(filepath.Join("..", "..", issue142CappedFixturePath))
	if err != nil {
		t.Fatal(err)
	}
	var item fixture
	for _, candidate := range items {
		if candidate.name == "new_test_pair_diverged" {
			item = candidate
			break
		}
	}
	input, candidates, err := issue142EnumeratedWeighted(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	current, err := issue142PairSupport(input.prepared.Document, input.ids)
	if err != nil {
		t.Fatal(err)
	}
	capped, err := issue142CappedPairSupport(input.prepared.Document, input.ids)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, candidate := range candidates {
		if candidate.ID != "C003" && candidate.ID != "C004" {
			continue
		}
		detail, err := issue142BreakdownCandidate(input.prepared.Document, input.ids, candidate)
		if err != nil {
			t.Fatal(err)
		}
		if len(detail.Pairs) != 1 || math.Abs(detail.CurrentScore-issue142CandidateScore(candidate, current)) > 1e-9 || math.Abs(detail.CappedScore-issue142CandidateScore(candidate, capped)) > 1e-9 {
			t.Fatalf("%s breakdown does not reconstruct score: %+v", candidate.ID, detail)
		}
		seen++
	}
	if seen != 2 {
		t.Fatalf("found %d target candidates, want 2", seen)
	}
}
