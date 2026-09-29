package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Each pair is called A-B, B-A, B-A, A-B. Candidate IDs and schema enum order stay fixed.
var issue142BidirectionalOrder = []bool{false, true, true, false}

type issue142BidirectionalPair struct {
	IDs    []string                `json:"ids"`
	Calls  []issue142DiagnosticRow `json:"calls,omitempty"`
	Status string                  `json:"status,omitempty"`
	Winner string                  `json:"winner,omitempty"`
}

type issue142BidirectionalRow struct {
	Fixture        string                      `json:"fixture"`
	CandidateCount int                         `json:"candidate_count"`
	GoldID         string                      `json:"gold_id,omitempty"`
	Pairs          []issue142BidirectionalPair `json:"pairs"`
	Winner         string                      `json:"winner,omitempty"`
	Ambiguous      bool                        `json:"ambiguous"`
	Cycle          bool                        `json:"cycle"`
	FinalExact     bool                        `json:"final_exact"`
	Calls          int                         `json:"calls"`
	WallMS         float64                     `json:"wall_ms"`
}

func issue142ClassifyBidirectionalPair(calls []issue142DiagnosticRow) (string, string) {
	if len(calls) != 4 {
		return "incomplete", ""
	}
	for _, call := range calls {
		if !call.ValidCandidate || call.StopReason != "completed" {
			return "incomplete", ""
		}
	}
	if calls[0].SelectedID != calls[3].SelectedID || calls[1].SelectedID != calls[2].SelectedID {
		return "repeat_variation", ""
	}
	if calls[0].SelectedID != calls[1].SelectedID {
		return "direction_disagreement", ""
	}
	return "stable", calls[0].SelectedID
}

func issue142AggregateBidirectional(ids []string, pairs []issue142BidirectionalPair) (string, bool) {
	stableWins := map[string]map[string]bool{}
	for _, pair := range pairs {
		if pair.Status != "stable" || len(pair.IDs) != 2 {
			continue
		}
		loser := pair.IDs[0]
		if pair.Winner == loser {
			loser = pair.IDs[1]
		}
		if stableWins[pair.Winner] == nil {
			stableWins[pair.Winner] = map[string]bool{}
		}
		stableWins[pair.Winner][loser] = true
	}
	winner := ""
	for _, id := range ids {
		if len(stableWins[id]) == len(ids)-1 {
			winner = id
		}
	}
	// A directed cycle is only counted when every edge in that cycle is stable.
	state := map[string]int{}
	var visit func(string) bool
	visit = func(id string) bool {
		state[id] = 1
		for other := range stableWins[id] {
			if state[other] == 1 || state[other] == 0 && visit(other) {
				return true
			}
		}
		state[id] = 2
		return false
	}
	cycle := false
	for _, id := range ids {
		if state[id] == 0 && visit(id) {
			cycle = true
			break
		}
	}
	return winner, cycle
}

func runIssue142Bidirectional(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" && !options.describe {
		return fmt.Errorf("Issue #142 bidirectional probe is fixed to MLX")
	}
	var backend llm.OptionsBackend
	model := ""
	if !options.describe {
		var err error
		backend, model, err = openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for _, item := range issue142TournamentFixtures() {
		if options.fixtureName != "all" && options.fixtureName != item.name {
			continue
		}
		matched = true
		prepared, ids, err := issue140Prepare(ctx, item, issue140ManyArms[5], 1024)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		baseline, _, err := issue142Generate(prepared.Document, ids)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		candidates, err := issue142ExpandCandidates(prepared.Document, ids, baseline)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		_, goldID := issue142HasGold(candidates, item.reference)
		row := issue142BidirectionalRow{Fixture: item.name, CandidateCount: len(candidates), GoldID: goldID}
		candidateIDs := make([]string, len(candidates))
		for i, candidate := range candidates {
			candidateIDs[i] = candidate.ID
		}
		for i := 0; i < len(candidates); i++ {
			for j := i + 1; j < len(candidates); j++ {
				pair := issue142BidirectionalPair{IDs: []string{candidates[i].ID, candidates[j].ID}}
				if !options.describe {
					for run, reverse := range issue142BidirectionalOrder {
						callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
						call := issue142DiagnosticCallModeWithSchemaOrder(callCtx, backend, model, "mlx", "bidirectional", "pairwise", item, prepared, []issue142Candidate{candidates[i], candidates[j]}, run+1, 2048, reverse, true, true)
						cancel()
						pair.Calls = append(pair.Calls, call)
						row.Calls += call.Calls
						row.WallMS += call.WallMS
					}
					pair.Status, pair.Winner = issue142ClassifyBidirectionalPair(pair.Calls)
				}
				row.Pairs = append(row.Pairs, pair)
			}
		}
		if !options.describe {
			row.Winner, row.Cycle = issue142AggregateBidirectional(candidateIDs, row.Pairs)
			row.Ambiguous = row.Winner == ""
			row.FinalExact = row.Winner != "" && row.Winner == goldID
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 bidirectional fixture %q", options.fixtureName)
	}
	return nil
}
