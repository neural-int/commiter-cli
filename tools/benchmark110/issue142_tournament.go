package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

const issue142TournamentCap = 8

// Holdout fixtures are added in a separate commit after the contract is fixed.
func issue142TournamentFixtures() []fixture {
	return append(issue142VerificationFixtures(), issue142TournamentHoldouts()...)
}

func issue142ThresholdPartition(doc contextinput.Document, ids []string) [][]string {
	parent := make(map[string]string, len(ids))
	paths := make(map[string]string, len(ids))
	for _, id := range ids {
		parent[id] = id
	}
	for _, file := range doc.Files {
		paths[file.ID] = issue141FilePath(file)
	}
	var root func(string) string
	root = func(id string) string {
		if parent[id] != id {
			parent[id] = root(parent[id])
		}
		return parent[id]
	}
	join := func(a, b string) {
		if _, ok := parent[a]; !ok {
			return
		}
		if _, ok := parent[b]; !ok {
			return
		}
		ra, rb := root(a), root(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			stem := issue141Stem(paths[a])
			if stem != "" && stem == issue141Stem(paths[b]) {
				join(a, b)
			}
		}
	}
	if doc.RelationContext != nil {
		for _, edge := range doc.RelationContext.Edges {
			if edge.Class != relation.Soft {
				continue
			}
			if edge.Kind == relation.SourceTest && edge.Reason == "matching_test_path" || edge.Kind == relation.DirectImport && edge.Reason == "observed_import_path" {
				join(edge.SourceID, edge.TargetID)
			}
		}
	}
	components := map[string][]string{}
	for _, id := range ids {
		components[root(id)] = append(components[root(id)], id)
	}
	groups := make([][]string, 0, len(components))
	for _, group := range components {
		groups = append(groups, group)
	}
	return groups
}

func issue142ExpandCandidates(doc contextinput.Document, ids []string, baseline []issue142Candidate) ([]issue142Candidate, error) {
	if len(baseline) > issue142TournamentCap {
		return nil, fmt.Errorf("baseline exceeds candidate cap")
	}
	result := make([]issue142Candidate, 0, issue142TournamentCap)
	seen := map[string]bool{}
	add := func(groups [][]string) error {
		normalized, key, err := issue142Canonical(groups, ids)
		if err != nil {
			return err
		}
		if seen[key] || len(result) == issue142TournamentCap {
			return nil
		}
		seen[key] = true
		result = append(result, issue142Candidate{ID: fmt.Sprintf("C%03d", len(result)+1), Groups: normalized})
		return nil
	}
	for _, candidate := range baseline {
		if err := add(candidate.Groups); err != nil {
			return nil, err
		}
	}
	if err := add(issue142ThresholdPartition(doc, ids)); err != nil {
		return nil, err
	}
	if err := add([][]string{append([]string(nil), ids...)}); err != nil {
		return nil, err
	}
	allSplit := make([][]string, len(ids))
	for i, id := range ids {
		allSplit[i] = []string{id}
	}
	if err := add(allSplit); err != nil {
		return nil, err
	}
	if len(result) < 2 {
		return nil, fmt.Errorf("fewer than two candidates")
	}
	return result, nil
}

type issue142TournamentRow struct {
	Fixture         string                  `json:"fixture"`
	Run             int                     `json:"run"`
	BaselineCount   int                     `json:"baseline_count"`
	ExpandedCount   int                     `json:"expanded_count"`
	BaselineRecall  bool                    `json:"baseline_recall"`
	ExpandedRecall  bool                    `json:"expanded_recall"`
	GoldCandidateID string                  `json:"gold_candidate_id,omitempty"`
	BracketIDs      []string                `json:"bracket_ids"`
	PairCalls       []issue142DiagnosticRow `json:"pair_calls,omitempty"`
	SelectedID      string                  `json:"selected_id,omitempty"`
	Completed       bool                    `json:"completed"`
	FinalExact      bool                    `json:"final_exact"`
	FalseMerge      *int                    `json:"false_merge,omitempty"`
	FalseSplit      *int                    `json:"false_split,omitempty"`
	Calls           int                     `json:"calls"`
	WallMS          float64                 `json:"wall_ms"`
}

func issue142HasGold(candidates []issue142Candidate, gold [][]string) (bool, string) {
	for _, candidate := range candidates {
		if sameGroups(candidate.Groups, gold) {
			return true, candidate.ID
		}
	}
	return false, ""
}

func runIssue142Tournament(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" && !options.describe {
		return fmt.Errorf("Issue #142 tournament is fixed to MLX")
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
		baselineRecall, _ := issue142HasGold(baseline, item.reference)
		expandedRecall, goldID := issue142HasGold(candidates, item.reference)
		for run := 1; run <= 2; run++ {
			row := issue142TournamentRow{Fixture: item.name, Run: run, BaselineCount: len(baseline), ExpandedCount: len(candidates), BaselineRecall: baselineRecall, ExpandedRecall: expandedRecall, GoldCandidateID: goldID, BracketIDs: make([]string, len(candidates))}
			bracket := append([]issue142Candidate(nil), candidates...)
			if run == 2 {
				sort.Slice(bracket, func(i, j int) bool { return bracket[i].ID > bracket[j].ID })
			}
			for i, candidate := range bracket {
				row.BracketIDs[i] = candidate.ID
			}
			if !options.describe {
				winner := bracket[0]
				valid := true
				for _, challenger := range bracket[1:] {
					pair := []issue142Candidate{winner, challenger}
					callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
					call := issue142DiagnosticCallMode(callCtx, backend, model, "mlx", "tournament", "pairwise", item, prepared, pair, run, 2048, false, true)
					cancel()
					row.PairCalls = append(row.PairCalls, call)
					row.Calls += call.Calls
					row.WallMS += call.WallMS
					if !call.ValidCandidate {
						valid = false
						break
					}
					if call.SelectedID == challenger.ID {
						winner = challenger
					}
				}
				if valid {
					row.Completed = true
					row.SelectedID = winner.ID
					row.FinalExact = sameGroups(winner.Groups, item.reference)
					score := issue140Row{}
					issue140Score(&score, winner.Groups, item.reference)
					row.FalseMerge = &score.FalseMerge
					row.FalseSplit = &score.FalseSplit
				}
			}
			if err := encoder.Encode(row); err != nil {
				return err
			}
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 tournament fixture %q", options.fixtureName)
	}
	return nil
}
