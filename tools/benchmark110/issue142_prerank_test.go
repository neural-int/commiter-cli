package main

import (
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

func TestIssue142PrerankScoreAndGoldBlindness(t *testing.T) {
	candidates := []issue142Candidate{
		{ID: "C002", Groups: [][]string{{"F001"}, {"F002"}}},
		{ID: "C001", Groups: [][]string{{"F001", "F002"}}},
	}
	support := map[string]float64{issue142PairKey("F001", "F002"): 1}
	mergedGold := issue142Rank("test", candidates, 1, candidates[1].Groups, support)
	splitGold := issue142Rank("test", candidates, 1, candidates[0].Groups, support)
	if mergedGold.Candidates[0].ID != "C001" || splitGold.Candidates[0].ID != "C001" || mergedGold.GoldRank != 1 || splitGold.GoldRank != 2 {
		t.Fatalf("ranking used gold or score was wrong: merged=%+v split=%+v", mergedGold, splitGold)
	}
	tied := issue142Rank("test", candidates, 1, nil, map[string]float64{issue142PairKey("F001", "F002"): 0.5})
	if tied.Candidates[0].ID != "C001" {
		t.Fatalf("tie should use ID: %+v", tied.Candidates)
	}
}

func TestIssue142PrerankPairSupport(t *testing.T) {
	first, second := "src/parse.go", "src/parse_test.go"
	doc := contextinput.Document{
		Files:           []contextinput.File{{ID: "F001", NewPath: &first}, {ID: "F002", NewPath: &second}},
		RelationContext: &contextinput.RelationContext{Edges: []relation.Relation{{SourceID: "F001", TargetID: "F002", Kind: relation.SourceTest, Class: relation.Soft, Reason: "matching_test_path"}}},
	}
	support, err := issue142PairSupport(doc, []string{"F001", "F002"})
	if err != nil {
		t.Fatal(err)
	}
	if got := support[issue142PairKey("F001", "F002")]; got != 5.1 {
		t.Fatalf("pair support = %v, want 5.1", got)
	}
}
