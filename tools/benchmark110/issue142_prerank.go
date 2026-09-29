package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

const issue142PairPenalty = 0.5

type issue142RankedCandidate struct {
	issue142Candidate
	Score float64 `json:"score"`
}

type issue142RankArm struct {
	Name             string                    `json:"name"`
	Candidates       []issue142RankedCandidate `json:"candidates"`
	GoldID           string                    `json:"gold_id,omitempty"`
	GoldRank         int                       `json:"gold_rank"`
	RecallAt1        bool                      `json:"recall_at_1"`
	RecallAt2        bool                      `json:"recall_at_2"`
	RecallAt3        bool                      `json:"recall_at_3"`
	TotalRecall      bool                      `json:"total_recall"`
	AddedNonGold     int                       `json:"added_non_gold"`
	Top2IDs          []string                  `json:"top_2_ids"`
	Top1Top2Margin   float64                   `json:"top_1_top_2_margin"`
	BestNonGoldID    string                    `json:"best_non_gold_id,omitempty"`
	BestNonGoldScore float64                   `json:"best_non_gold_score"`
}

type issue142RankRow struct {
	Fixture string            `json:"fixture"`
	Arms    []issue142RankArm `json:"arms"`
}

type issue142RankInput struct {
	item     fixture
	prepared contextinput.Prepared
	ids      []string
	arms     []issue142RankArm
}

func issue142PairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "\x00" + b
}

func issue142PairSupport(doc contextinput.Document, ids []string) (map[string]float64, error) {
	files := map[string]contextinput.File{}
	for _, file := range doc.Files {
		files[file.ID] = file
	}
	lexical, err := issue142WeightedPairs(doc, ids)
	if err != nil {
		return nil, err
	}
	support := map[string]float64{}
	for _, edge := range lexical {
		support[issue142PairKey(edge.IDs[0], edge.IDs[1])] += edge.Score
	}
	for i, a := range ids {
		pa := issue141FilePath(files[a])
		if pa == "" {
			return nil, fmt.Errorf("missing path for %s", a)
		}
		for _, b := range ids[i+1:] {
			pb := issue141FilePath(files[b])
			if pb == "" {
				return nil, fmt.Errorf("missing path for %s", b)
			}
			key := issue142PairKey(a, b)
			if issue141Stem(pa) != "" && issue141Stem(pa) == issue141Stem(pb) {
				support[key] += 1
			}
			if path.Dir(pa) == path.Dir(pb) {
				support[key] += 0.1
			}
		}
	}
	if doc.RelationContext != nil {
		for _, edge := range doc.RelationContext.Edges {
			if edge.Class != relation.Soft {
				continue
			}
			weight := 0.0
			switch {
			case edge.Kind == relation.SourceTest && edge.Reason == "matching_test_path":
				weight = 4
			case edge.Kind == relation.DirectImport && edge.Reason == "observed_import_path":
				weight = 3
			}
			if weight > 0 {
				support[issue142PairKey(edge.SourceID, edge.TargetID)] += weight
			}
		}
	}
	return support, nil
}

func issue142CandidateScore(candidate issue142Candidate, support map[string]float64) float64 {
	score := 0.0
	for _, group := range candidate.Groups {
		for i, a := range group {
			for _, b := range group[i+1:] {
				score += support[issue142PairKey(a, b)] - issue142PairPenalty
			}
		}
	}
	return score
}

func issue142Rank(name string, candidates []issue142Candidate, baseCount int, gold [][]string, support map[string]float64) issue142RankArm {
	arm := issue142RankArm{Name: name, Candidates: make([]issue142RankedCandidate, len(candidates))}
	for i, candidate := range candidates {
		arm.Candidates[i] = issue142RankedCandidate{issue142Candidate: candidate, Score: issue142CandidateScore(candidate, support)}
		if i >= baseCount && !sameGroups(candidate.Groups, gold) {
			arm.AddedNonGold++
		}
	}
	sort.Slice(arm.Candidates, func(i, j int) bool {
		a, b := arm.Candidates[i], arm.Candidates[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return a.ID < b.ID
	})
	if len(arm.Candidates) >= 2 {
		arm.Top2IDs = []string{arm.Candidates[0].ID, arm.Candidates[1].ID}
		arm.Top1Top2Margin = arm.Candidates[0].Score - arm.Candidates[1].Score
	}
	for i, candidate := range arm.Candidates {
		if sameGroups(candidate.Groups, gold) {
			arm.GoldID, arm.GoldRank = candidate.ID, i+1
		} else if arm.BestNonGoldID == "" {
			arm.BestNonGoldID, arm.BestNonGoldScore = candidate.ID, candidate.Score
		}
	}
	arm.RecallAt1 = arm.GoldRank == 1
	arm.RecallAt2 = arm.GoldRank > 0 && arm.GoldRank <= 2
	arm.RecallAt3 = arm.GoldRank > 0 && arm.GoldRank <= 3
	arm.TotalRecall = arm.GoldRank > 0
	return arm
}

func issue142BuildRankInput(ctx context.Context, item fixture) (issue142RankInput, error) {
	prepared, ids, err := issue140Prepare(ctx, item, issue140ManyArms[5], 1024)
	if err != nil {
		return issue142RankInput{}, err
	}
	raw, _, err := issue142Generate(prepared.Document, ids)
	if err != nil {
		return issue142RankInput{}, err
	}
	base, err := issue142ExpandCandidates(prepared.Document, ids, raw)
	if err != nil {
		return issue142RankInput{}, err
	}
	binaryGroups, _, err := issue142LexicalPartition(prepared.Document, ids)
	if err != nil {
		return issue142RankInput{}, err
	}
	binary, _, _, err := issue142AddPartitions(base, ids, [][][]string{binaryGroups})
	if err != nil {
		return issue142RankInput{}, err
	}
	edges, err := issue142WeightedPairs(prepared.Document, ids)
	if err != nil {
		return issue142RankInput{}, err
	}
	partitions := [][][]string{}
	for _, threshold := range issue142WeightedThresholds {
		groups, _, err := issue142WeightedPartition(ids, edges, threshold)
		if err != nil {
			return issue142RankInput{}, err
		}
		partitions = append(partitions, groups)
	}
	weighted, _, _, err := issue142AddPartitions(base, ids, partitions)
	if err != nil {
		return issue142RankInput{}, err
	}
	support, err := issue142PairSupport(prepared.Document, ids)
	if err != nil {
		return issue142RankInput{}, err
	}
	arms := []issue142RankArm{
		issue142Rank("baseline", base, len(base), item.reference, support),
		issue142Rank("binary", binary, len(base), item.reference, support),
		issue142Rank("weighted", weighted, len(base), item.reference, support),
	}
	return issue142RankInput{item: item, prepared: prepared, ids: ids, arms: arms}, nil
}

func issue142TopCandidates(arm issue142RankArm) ([]issue142Candidate, error) {
	if len(arm.Candidates) < 2 {
		return nil, fmt.Errorf("%s has fewer than two candidates", arm.Name)
	}
	return []issue142Candidate{arm.Candidates[0].issue142Candidate, arm.Candidates[1].issue142Candidate}, nil
}

var issue142Top2Targets = []struct{ fixture, arm string }{
	{"holdout_crossdir_semantic", "binary"},
	{"lexical_crossdir_pairs", "binary"},
	{"lexical_crossdir_pairs", "weighted"},
	{"lexical_bridge", "weighted"},
	{"lexical_collision", "weighted"},
	{"verify_misleading_relation", "weighted"},
}

type issue142Top2Row struct {
	Fixture    string                  `json:"fixture"`
	Arm        string                  `json:"arm"`
	Candidates []issue142Candidate     `json:"candidates"`
	GoldID     string                  `json:"gold_id,omitempty"`
	GoldRank   int                     `json:"gold_rank"`
	Eligible   bool                    `json:"eligible"`
	SkipReason string                  `json:"skip_reason,omitempty"`
	Calls      []issue142DiagnosticRow `json:"calls,omitempty"`
}

func issue142OpenRankBackend(options issue140Options) (llm.OptionsBackend, string, error) {
	if options.describe {
		return nil, "", nil
	}
	if options.backendName != "mlx" {
		return nil, "", fmt.Errorf("prerank selector fixed to MLX")
	}
	return openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
}

func runIssue142Prerank(ctx context.Context, options issue140Options, probe string) error {
	var backend llm.OptionsBackend
	model := ""
	if probe != "prerank" {
		var err error
		backend, model, err = issue142OpenRankBackend(options)
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	fallbackFixtures := 0
	for _, item := range issue142AllLexicalFixtures() {
		if options.fixtureName != "all" && options.fixtureName != item.name {
			continue
		}
		matched = true
		input, err := issue142BuildRankInput(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		if probe == "prerank" {
			if err := encoder.Encode(issue142RankRow{Fixture: item.name, Arms: input.arms}); err != nil {
				return err
			}
			continue
		}
		if probe == "prerank-top2" {
			for _, target := range issue142Top2Targets {
				if target.fixture != item.name {
					continue
				}
				arm := input.arms[1]
				if target.arm == "weighted" {
					arm = input.arms[2]
				}
				candidates, err := issue142TopCandidates(arm)
				if err != nil {
					return err
				}
				row := issue142Top2Row{Fixture: item.name, Arm: arm.Name, Candidates: candidates, GoldID: arm.GoldID,
					GoldRank: arm.GoldRank, Eligible: arm.RecallAt2}
				if !row.Eligible {
					row.SkipReason = "gold_not_in_top_2"
				}
				if row.Eligible && !options.describe {
					for run, reverse := range issue142BidirectionalOrder {
						system, prompt, schema, err := issue142GenericForcedInput(input.prepared, candidates, reverse)
						if err != nil {
							return err
						}
						callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
						call := issue142FollowupCall(callCtx, backend, model, probe, arm.Name, item,
							input.prepared, candidates, run+1, system, prompt, schema)
						cancel()
						row.Calls = append(row.Calls, call)
					}
				}
				if err := encoder.Encode(row); err != nil {
					return err
				}
			}
			continue
		}
		if probe == "prerank-pairwise" {
			arm := input.arms[2]
			if !arm.TotalRecall || arm.RecallAt2 || fallbackFixtures >= 2 {
				continue
			}
			fallbackFixtures++
			pairs := []issue142PairwiseRow{}
			candidates := make([]issue142Candidate, len(arm.Candidates))
			for i, candidate := range arm.Candidates {
				candidates[i] = candidate.issue142Candidate
			}
			sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
			wins := map[string]int{}
			allComplete := true
			for i, a := range candidates {
				for _, b := range candidates[i+1:] {
					pair := []issue142Candidate{a, b}
					result := issue142PairwiseRow{IDs: []string{a.ID, b.ID}}
					if !options.describe {
						for run, reverse := range []bool{false, true} {
							system, prompt, schema, err := issue142GenericForcedInput(input.prepared, pair, reverse)
							if err != nil {
								return err
							}
							callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
							call := issue142FollowupCall(callCtx, backend, model, probe, "weighted", item,
								input.prepared, pair, run+1, system, prompt, schema)
							cancel()
							result.Calls = append(result.Calls, call)
						}
						if len(result.Calls) == 2 && result.Calls[0].ValidCandidate && result.Calls[1].ValidCandidate &&
							result.Calls[0].SelectedID == result.Calls[1].SelectedID {
							result.StableWinner = result.Calls[0].SelectedID
							wins[result.StableWinner]++
						} else {
							allComplete = false
						}
					}
					pairs = append(pairs, result)
				}
			}
			strict := ""
			if allComplete {
				for id, count := range wins {
					if count == len(candidates)-1 {
						strict = id
					}
				}
			}
			row := issue142PairwiseFixtureRow{Fixture: item.name, GoldID: arm.GoldID, GoldRank: arm.GoldRank,
				CandidateCount: len(candidates), Candidates: candidates, Pairs: pairs, StrictWinner: strict}
			if err := encoder.Encode(row); err != nil {
				return err
			}
		}
	}
	if !matched {
		return fmt.Errorf("unknown fixture %q", options.fixtureName)
	}
	if probe == "prerank-pairwise" && fallbackFixtures == 0 {
		return encoder.Encode(issue142PairwiseFixtureRow{Fixture: "none", CandidateCount: 0, Pairs: []issue142PairwiseRow{}})
	}
	return nil
}

type issue142PairwiseRow struct {
	IDs          []string                `json:"ids"`
	Calls        []issue142DiagnosticRow `json:"calls,omitempty"`
	StableWinner string                  `json:"stable_winner,omitempty"`
}

type issue142PairwiseFixtureRow struct {
	Fixture        string                `json:"fixture"`
	GoldID         string                `json:"gold_id,omitempty"`
	GoldRank       int                   `json:"gold_rank"`
	CandidateCount int                   `json:"candidate_count"`
	Candidates     []issue142Candidate   `json:"candidates,omitempty"`
	Pairs          []issue142PairwiseRow `json:"pairs"`
	StrictWinner   string                `json:"strict_winner,omitempty"`
}
