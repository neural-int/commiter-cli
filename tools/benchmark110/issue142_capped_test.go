package main

import (
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

func TestIssue142CappedStructuralSupport(t *testing.T) {
	first, second := "src/a.go", "src/a_test.go"
	ids := []string{"F001", "F002"}
	doc := contextinput.Document{
		Files:           []contextinput.File{{ID: ids[0], NewPath: &first}, {ID: ids[1], NewPath: &second}},
		RelationContext: &contextinput.RelationContext{Edges: []relation.Relation{{SourceID: ids[0], TargetID: ids[1], Kind: relation.SourceTest, Class: relation.Soft, Reason: "matching_test_path"}}},
	}
	current, err := issue142PairSupport(doc, ids)
	if err != nil {
		t.Fatal(err)
	}
	capped, err := issue142CappedPairSupport(doc, ids)
	if err != nil {
		t.Fatal(err)
	}
	key := issue142PairKey(ids[0], ids[1])
	if current[key] != 4.1 || capped[key] != 0.4 {
		t.Fatalf("current=%v capped=%v", current[key], capped[key])
	}
	joined := issue142Candidate{ID: "C001", Groups: [][]string{ids}}
	if got := issue142CandidateScore(joined, capped); got >= 0 {
		t.Fatalf("structural-only pair received positive score: %v", got)
	}
}
