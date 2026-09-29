package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

const issue142SelectorFixture = "new_stem_doc_diverged"

var issue142EnumerationFixtures = []string{"new_test_pair_diverged", "new_crossdir_collision", "new_paraphrase"}

type issue142SelectorRow struct {
	Fixture    string                  `json:"fixture"`
	Candidates []issue142Candidate     `json:"candidates"`
	GoldID     string                  `json:"gold_id"`
	GoldRank   int                     `json:"gold_rank"`
	Calls      []issue142DiagnosticRow `json:"calls,omitempty"`
}

func runIssue142CappedSelector(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" && !options.describe {
		return fmt.Errorf("capped selector fixed to MLX")
	}
	items, err := issue142LoadHoldoutFixtures(issue142CappedFixturePath)
	if err != nil {
		return err
	}
	var item fixture
	for _, candidate := range items {
		if candidate.name == issue142SelectorFixture {
			item = candidate
			break
		}
	}
	if item.name == "" || options.fixtureName != "all" && options.fixtureName != item.name {
		return fmt.Errorf("selector fixture mismatch")
	}
	input, err := issue142BuildRankInput(ctx, item)
	if err != nil {
		return err
	}
	support, err := issue142CappedPairSupport(input.prepared.Document, input.ids)
	if err != nil {
		return err
	}
	weighted := input.arms[2]
	candidates := make([]issue142Candidate, 0, len(weighted.Candidates))
	for _, ranked := range weighted.Candidates {
		candidates = append(candidates, ranked.issue142Candidate)
	}
	capped := issue142Rank("weighted", candidates, len(input.arms[0].Candidates), item.reference, support)
	pair, err := issue142TopCandidates(capped)
	if err != nil {
		return err
	}
	if pair[0].ID != "C001" || pair[1].ID != "C002" || capped.GoldID != "C002" || capped.GoldRank != 2 {
		return fmt.Errorf("preregistered top-2 or gold changed")
	}
	row := issue142SelectorRow{Fixture: item.name, Candidates: pair, GoldID: capped.GoldID, GoldRank: capped.GoldRank}
	if !options.describe {
		if options.modelSpec.Repo != "mlx-community/Ministral-3-3B-Instruct-2512-4bit" || options.modelSpec.Revision != "a962dcb09eee4169c890e544c9eb938f1113fdee" {
			return fmt.Errorf("selector model pin differs from preregistration")
		}
		backend, model, err := openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
		if err != nil {
			return err
		}
		for run, reverse := range issue142BidirectionalOrder {
			system, prompt, schema, err := issue142GenericForcedInput(input.prepared, pair, reverse)
			if err != nil {
				return err
			}
			callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			call := issue142FollowupCall(callCtx, backend, model, "capped-selector", "weighted", item,
				input.prepared, pair, run+1, system, prompt, schema)
			cancel()
			row.Calls = append(row.Calls, call)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(row)
}

type issue142RelationSummary struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Kind     string `json:"kind"`
	Class    string `json:"class"`
	Reason   string `json:"reason"`
}

type issue142EnumerationRow struct {
	Fixture            string                    `json:"fixture"`
	Relations          []issue142RelationSummary `json:"relations"`
	RawCandidates      []issue142Candidate       `json:"raw_candidates"`
	BaselineCandidates []issue142Candidate       `json:"baseline_candidates"`
	BinaryCandidates   []issue142Candidate       `json:"binary_candidates"`
	InitialCandidates  []issue142Candidate       `json:"initial_candidates"`
	Enumerated         [][][]string              `json:"enumerated"`
	DuplicateCount     int                       `json:"duplicate_count"`
	SkippedCap         int                       `json:"skipped_cap"`
	CapReached         bool                      `json:"cap_reached"`
	Added              []issue142Candidate       `json:"added"`
	CurrentBefore      issue142RankArm           `json:"current_before"`
	CappedBefore       issue142RankArm           `json:"capped_before"`
	CurrentAfter       issue142RankArm           `json:"current_after"`
	CappedAfter        issue142RankArm           `json:"capped_after"`
}

func issue142CandidatesByID(arm issue142RankArm) []issue142Candidate {
	candidates := make([]issue142Candidate, 0, len(arm.Candidates))
	for _, ranked := range arm.Candidates {
		candidates = append(candidates, ranked.issue142Candidate)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	return candidates
}

func issue142SmallPartitions(ids []string) ([][][]string, error) {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	partitions := [][][]string{}
	switch len(sorted) {
	case 3:
		for i, a := range sorted {
			for _, b := range sorted[i+1:] {
				other := ""
				for _, id := range sorted {
					if id != a && id != b {
						other = id
					}
				}
				partitions = append(partitions, [][]string{{a, b}, {other}})
			}
		}
	case 4:
		for _, b := range sorted[1:] {
			rest := []string{}
			for _, id := range sorted[1:] {
				if id != b {
					rest = append(rest, id)
				}
			}
			partitions = append(partitions, [][]string{{sorted[0], b}, rest})
		}
	default:
		return nil, fmt.Errorf("small partitions require 3 or 4 files")
	}
	return partitions, nil
}

func issue142DuplicatePartitions(base []issue142Candidate, ids []string, partitions [][][]string) (int, error) {
	seen := map[string]bool{}
	for _, candidate := range base {
		_, key, err := issue142Canonical(candidate.Groups, ids)
		if err != nil {
			return 0, err
		}
		seen[key] = true
	}
	duplicates := 0
	for _, groups := range partitions {
		_, key, err := issue142Canonical(groups, ids)
		if err != nil {
			return 0, err
		}
		if seen[key] {
			duplicates++
		}
		seen[key] = true
	}
	return duplicates, nil
}

func runIssue142SmallEnumeration(ctx context.Context, options issue140Options) error {
	items, err := issue142LoadHoldoutFixtures(issue142CappedFixturePath)
	if err != nil {
		return err
	}
	byName := map[string]fixture{}
	for _, item := range items {
		byName[item.name] = item
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for _, name := range issue142EnumerationFixtures {
		if options.fixtureName != "all" && options.fixtureName != name {
			continue
		}
		matched = true
		item := byName[name]
		input, err := issue142BuildRankInput(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		current, err := issue142PairSupport(input.prepared.Document, input.ids)
		if err != nil {
			return err
		}
		capped, err := issue142CappedPairSupport(input.prepared.Document, input.ids)
		if err != nil {
			return err
		}
		raw, _, err := issue142Generate(input.prepared.Document, input.ids)
		if err != nil {
			return err
		}
		initial := issue142CandidatesByID(input.arms[2])
		partitions, err := issue142SmallPartitions(input.ids)
		if err != nil {
			return err
		}
		duplicates, err := issue142DuplicatePartitions(initial, input.ids, partitions)
		if err != nil {
			return err
		}
		final, added, capReached, err := issue142AddPartitions(initial, input.ids, partitions)
		if err != nil {
			return err
		}
		row := issue142EnumerationRow{Fixture: name, Relations: []issue142RelationSummary{},
			RawCandidates: raw, BaselineCandidates: issue142CandidatesByID(input.arms[0]),
			BinaryCandidates: issue142CandidatesByID(input.arms[1]), InitialCandidates: initial,
			Enumerated: partitions, DuplicateCount: duplicates, SkippedCap: len(partitions) - duplicates - len(added),
			CapReached: capReached, Added: added,
			CurrentBefore: issue142Rank("weighted", initial, len(initial), item.reference, current),
			CappedBefore:  issue142Rank("weighted", initial, len(initial), item.reference, capped),
			CurrentAfter:  issue142Rank("weighted", final, len(initial), item.reference, current),
			CappedAfter:   issue142Rank("weighted", final, len(initial), item.reference, capped),
		}
		if input.prepared.Document.RelationContext != nil {
			for _, edge := range input.prepared.Document.RelationContext.Edges {
				row.Relations = append(row.Relations, issue142RelationSummary{SourceID: edge.SourceID, TargetID: edge.TargetID,
					Kind: string(edge.Kind), Class: string(edge.Class), Reason: edge.Reason})
			}
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown enumeration fixture %q", options.fixtureName)
	}
	return nil
}
