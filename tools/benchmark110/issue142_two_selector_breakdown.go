package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

type issue142TwoSelectorRow struct {
	Fixture    string                  `json:"fixture"`
	Candidates []issue142Candidate     `json:"candidates"`
	GoldID     string                  `json:"gold_id"`
	GoldRank   int                     `json:"gold_rank"`
	Calls      []issue142DiagnosticRow `json:"calls,omitempty"`
}

type issue142TwoSelectorInput struct {
	item     fixture
	prepared contextinput.Prepared
	row      issue142TwoSelectorRow
}

func issue142EnumeratedWeighted(ctx context.Context, item fixture) (issue142RankInput, []issue142Candidate, error) {
	input, err := issue142BuildRankInput(ctx, item)
	if err != nil {
		return issue142RankInput{}, nil, err
	}
	initial := issue142CandidatesByID(input.arms[2])
	partitions, err := issue142SmallPartitions(input.ids)
	if err != nil {
		return issue142RankInput{}, nil, err
	}
	final, _, _, err := issue142AddPartitions(initial, input.ids, partitions)
	return input, final, err
}

func runIssue142TwoSelectors(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" && !options.describe {
		return fmt.Errorf("two selectors fixed to MLX")
	}
	if options.fixtureName != "all" {
		return fmt.Errorf("two selector probe requires all fixed fixtures")
	}
	items, err := issue142LoadHoldoutFixtures(issue142CappedFixturePath)
	if err != nil {
		return err
	}
	byName := map[string]fixture{}
	for _, item := range items {
		byName[item.name] = item
	}
	targets := []struct{ name, first, second, gold string }{
		{"new_crossdir_collision", "C003", "C004", "C004"},
		{"new_paraphrase", "C001", "C003", "C003"},
	}
	inputs := make([]issue142TwoSelectorInput, 0, len(targets))
	for _, target := range targets {
		item := byName[target.name]
		input, final, err := issue142EnumeratedWeighted(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", target.name, err)
		}
		support, err := issue142CappedPairSupport(input.prepared.Document, input.ids)
		if err != nil {
			return err
		}
		ranked := issue142Rank("weighted", final, len(input.arms[2].Candidates), item.reference, support)
		pair, err := issue142TopCandidates(ranked)
		if err != nil {
			return err
		}
		if pair[0].ID != target.first || pair[1].ID != target.second || ranked.GoldID != target.gold || ranked.GoldRank != 2 {
			return fmt.Errorf("%s: preregistered candidate IDs or gold changed", target.name)
		}
		inputs = append(inputs, issue142TwoSelectorInput{item: item, prepared: input.prepared,
			row: issue142TwoSelectorRow{Fixture: item.name, Candidates: pair, GoldID: ranked.GoldID, GoldRank: ranked.GoldRank}})
	}
	var backend llm.OptionsBackend
	model := ""
	if !options.describe {
		if options.modelSpec.Repo != "mlx-community/Ministral-3-3B-Instruct-2512-4bit" || options.modelSpec.Revision != "a962dcb09eee4169c890e544c9eb938f1113fdee" {
			return fmt.Errorf("selector model pin differs from preregistration")
		}
		backend, model, err = openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	for _, input := range inputs {
		if !options.describe {
			for run, reverse := range issue142BidirectionalOrder {
				system, prompt, schema, err := issue142GenericForcedInput(input.prepared, input.row.Candidates, reverse)
				if err != nil {
					return err
				}
				callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				call := issue142FollowupCall(callCtx, backend, model, "two-selectors", "weighted", input.item,
					input.prepared, input.row.Candidates, run+1, system, prompt, schema)
				cancel()
				input.row.Calls = append(input.row.Calls, call)
			}
		}
		if err := encoder.Encode(input.row); err != nil {
			return err
		}
	}
	return nil
}

type issue142LexicalShare struct {
	DiffDiff         float64 `json:"diff_diff"`
	PathPath         float64 `json:"path_path"`
	Mixed            float64 `json:"mixed"`
	SharedTokenCount int     `json:"shared_token_count"`
}

type issue142PairBreakdown struct {
	IDs              []string             `json:"ids"`
	SourceTest       float64              `json:"source_test"`
	DirectImport     float64              `json:"direct_import"`
	Stem             float64              `json:"stem"`
	SameDirectory    float64              `json:"same_directory"`
	RawStructural    float64              `json:"raw_structural"`
	CappedStructural float64              `json:"capped_structural"`
	LexicalTotal     float64              `json:"lexical_total"`
	Lexical          issue142LexicalShare `json:"lexical"`
	Penalty          float64              `json:"penalty"`
	CurrentPairScore float64              `json:"current_pair_score"`
	CappedPairScore  float64              `json:"capped_pair_score"`
}

type issue142CandidateBreakdown struct {
	Candidate    issue142Candidate       `json:"candidate"`
	Pairs        []issue142PairBreakdown `json:"pairs"`
	CurrentScore float64                 `json:"current_score"`
	CappedScore  float64                 `json:"capped_score"`
}

type issue142BreakdownRow struct {
	Fixture    string                       `json:"fixture"`
	Candidates []issue142CandidateBreakdown `json:"candidates"`
}

func issue142TokensFromText(value string) map[string]bool {
	tokens := map[string]bool{}
	for _, word := range issue142LexicalWord.FindAllString(value, -1) {
		word = strings.ToLower(word)
		if len(word) >= 4 && !issue142LexicalStop[word] {
			tokens[word] = true
		}
	}
	return tokens
}

func issue142BreakdownPair(doc contextinput.Document, ids []string, a, b string) (issue142PairBreakdown, error) {
	files := map[string]contextinput.File{}
	pathTokens, diffTokens, allTokens := map[string]map[string]bool{}, map[string]map[string]bool{}, map[string]map[string]bool{}
	frequency := map[string]int{}
	for _, file := range doc.Files {
		files[file.ID] = file
	}
	for _, id := range ids {
		file, ok := files[id]
		if !ok {
			return issue142PairBreakdown{}, fmt.Errorf("missing file %s", id)
		}
		pathTokens[id] = issue142TokensFromText(issue141FilePath(file))
		diffTokens[id] = issue142TokensFromText(file.RawDiff)
		allTokens[id] = map[string]bool{}
		for token := range pathTokens[id] {
			allTokens[id][token] = true
		}
		for token := range diffTokens[id] {
			allTokens[id][token] = true
		}
		for token := range allTokens[id] {
			frequency[token]++
		}
	}
	result := issue142PairBreakdown{IDs: []string{a, b}, Penalty: issue142PairPenalty}
	pa, pb := issue141FilePath(files[a]), issue141FilePath(files[b])
	if pa == "" || pb == "" {
		return result, fmt.Errorf("missing path")
	}
	if issue141Stem(pa) != "" && issue141Stem(pa) == issue141Stem(pb) {
		result.Stem = 1
	}
	if path.Dir(pa) == path.Dir(pb) {
		result.SameDirectory = 0.1
	}
	if doc.RelationContext != nil {
		for _, edge := range doc.RelationContext.Edges {
			if edge.Class != relation.Soft || issue142PairKey(edge.SourceID, edge.TargetID) != issue142PairKey(a, b) {
				continue
			}
			switch {
			case edge.Kind == relation.SourceTest && edge.Reason == "matching_test_path":
				result.SourceTest += 4
			case edge.Kind == relation.DirectImport && edge.Reason == "observed_import_path":
				result.DirectImport += 3
			}
		}
	}
	result.RawStructural = result.SourceTest + result.DirectImport + result.Stem + result.SameDirectory
	result.CappedStructural = math.Min(issue142CappedStructural, result.RawStructural)
	for token := range allTokens[a] {
		if !allTokens[b][token] {
			continue
		}
		weight := 1 / float64(frequency[token]-1)
		result.LexicalTotal += weight
		result.Lexical.SharedTokenCount++
		switch {
		case diffTokens[a][token] && diffTokens[b][token]:
			result.Lexical.DiffDiff += weight
		case pathTokens[a][token] && pathTokens[b][token]:
			result.Lexical.PathPath += weight
		default:
			result.Lexical.Mixed += weight
		}
	}
	if math.Abs(result.LexicalTotal-(result.Lexical.DiffDiff+result.Lexical.PathPath+result.Lexical.Mixed)) > 1e-9 {
		return result, fmt.Errorf("lexical attribution mismatch")
	}
	result.CurrentPairScore = result.RawStructural + result.LexicalTotal - result.Penalty
	result.CappedPairScore = result.CappedStructural + result.LexicalTotal - result.Penalty
	return result, nil
}

func issue142BreakdownCandidate(doc contextinput.Document, ids []string, candidate issue142Candidate) (issue142CandidateBreakdown, error) {
	result := issue142CandidateBreakdown{Candidate: candidate, Pairs: []issue142PairBreakdown{}}
	for _, group := range candidate.Groups {
		for i, a := range group {
			for _, b := range group[i+1:] {
				pair, err := issue142BreakdownPair(doc, ids, a, b)
				if err != nil {
					return result, err
				}
				result.Pairs = append(result.Pairs, pair)
				result.CurrentScore += pair.CurrentPairScore
				result.CappedScore += pair.CappedPairScore
			}
		}
	}
	return result, nil
}

func runIssue142ScoreBreakdown(ctx context.Context, options issue140Options) error {
	if options.fixtureName != "all" && options.fixtureName != "new_test_pair_diverged" {
		return fmt.Errorf("score breakdown fixture mismatch")
	}
	items, err := issue142LoadHoldoutFixtures(issue142CappedFixturePath)
	if err != nil {
		return err
	}
	var item fixture
	for _, candidate := range items {
		if candidate.name == "new_test_pair_diverged" {
			item = candidate
			break
		}
	}
	input, final, err := issue142EnumeratedWeighted(ctx, item)
	if err != nil {
		return err
	}
	current, err := issue142PairSupport(input.prepared.Document, input.ids)
	if err != nil {
		return err
	}
	capped, err := issue142CappedPairSupport(input.prepared.Document, input.ids)
	if err != nil {
		return err
	}
	result := issue142BreakdownRow{Fixture: item.name, Candidates: []issue142CandidateBreakdown{}}
	for _, id := range []string{"C003", "C004"} {
		found := false
		for _, candidate := range final {
			if candidate.ID != id {
				continue
			}
			found = true
			detail, err := issue142BreakdownCandidate(input.prepared.Document, input.ids, candidate)
			if err != nil {
				return err
			}
			if math.Abs(detail.CurrentScore-issue142CandidateScore(candidate, current)) > 1e-9 || math.Abs(detail.CappedScore-issue142CandidateScore(candidate, capped)) > 1e-9 {
				return fmt.Errorf("%s score decomposition mismatch", id)
			}
			result.Candidates = append(result.Candidates, detail)
		}
		if !found {
			return fmt.Errorf("missing preregistered candidate %s", id)
		}
	}
	if math.Abs(result.Candidates[0].CurrentScore-4.6) > 1e-9 || math.Abs(result.Candidates[1].CurrentScore-1.5) > 1e-9 || math.Abs(result.Candidates[0].CappedScore-0.9) > 1e-9 || math.Abs(result.Candidates[1].CappedScore-0.9) > 1e-9 {
		return fmt.Errorf("preregistered score changed")
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
