package main

import (
	"context"
	"testing"
)

func TestIssue142BidirectionalInputFixesSchemaOrder(t *testing.T) {
	item := issue142TournamentFixtures()[0]
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
	_, forward, forwardSchema, err := issue142SelectionInputModeWithSchemaOrder(prepared, candidates[:2], false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	_, reverse, reverseSchema, err := issue142SelectionInputModeWithSchemaOrder(prepared, candidates[:2], true, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if string(forwardSchema) != string(reverseSchema) || string(forward) == string(reverse) {
		t.Fatal("candidate presentation must change while schema enum order stays fixed")
	}
}

func TestIssue142BidirectionalClassificationAndAggregation(t *testing.T) {
	ids := []string{"C001", "C002", "C003"}
	call := func(id string) issue142DiagnosticRow {
		return issue142DiagnosticRow{StopReason: "completed", ValidCandidate: true, SelectedID: id}
	}
	stable := []issue142DiagnosticRow{call("C001"), call("C001"), call("C001"), call("C001")}
	if status, winner := issue142ClassifyBidirectionalPair(stable); status != "stable" || winner != "C001" {
		t.Fatalf("stable: %s %s", status, winner)
	}
	if status, _ := issue142ClassifyBidirectionalPair([]issue142DiagnosticRow{call("C001"), call("C002"), call("C002"), call("C001")}); status != "direction_disagreement" {
		t.Fatalf("direction disagreement: %s", status)
	}
	if status, _ := issue142ClassifyBidirectionalPair([]issue142DiagnosticRow{call("C001"), call("C002"), call("C001"), call("C001")}); status != "repeat_variation" {
		t.Fatalf("repeat variation: %s", status)
	}
	if status, _ := issue142ClassifyBidirectionalPair([]issue142DiagnosticRow{call("C001"), {StopReason: "max_tokens"}, call("C001"), call("C001")}); status != "incomplete" {
		t.Fatalf("incomplete: %s", status)
	}
	pairs := []issue142BidirectionalPair{
		{IDs: []string{"C001", "C002"}, Status: "stable", Winner: "C001"},
		{IDs: []string{"C001", "C003"}, Status: "stable", Winner: "C001"},
		{IDs: []string{"C002", "C003"}, Status: "direction_disagreement"},
	}
	if winner, cycle := issue142AggregateBidirectional(ids, pairs); winner != "C001" || cycle {
		t.Fatalf("Condorcet winner: %s cycle=%t", winner, cycle)
	}
	pairs[1].Winner = "C003"
	pairs[2] = issue142BidirectionalPair{IDs: []string{"C002", "C003"}, Status: "stable", Winner: "C002"}
	if winner, cycle := issue142AggregateBidirectional(ids, pairs); winner != "" || !cycle {
		t.Fatalf("cycle: %s cycle=%t", winner, cycle)
	}
}
