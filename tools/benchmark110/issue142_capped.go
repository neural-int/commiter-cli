package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

const issue142CappedStructural = 0.4
const issue142CappedFixturePath = "docs/benchmarks/issue-142-capped-structural-fixtures-2026-09-29.json"

type issue142HoldoutFile struct {
	Path string `json:"path"`
	Diff string `json:"diff"`
}

type issue142HoldoutFixture struct {
	Name  string                `json:"name"`
	Files []issue142HoldoutFile `json:"files"`
	Gold  [][]string            `json:"gold"`
}

type issue142HoldoutFileSet struct {
	Fixtures []issue142HoldoutFixture `json:"fixtures"`
}

var issue142HoldoutContract = []struct {
	name  string
	count int
	gold  [][]string
}{
	{"new_test_pair_shared", 3, [][]string{{"F001", "F002"}, {"F003"}}},
	{"new_test_pair_diverged", 3, [][]string{{"F001", "F003"}, {"F002"}}},
	{"new_stem_doc_shared", 3, [][]string{{"F001", "F002"}, {"F003"}}},
	{"new_stem_doc_diverged", 3, [][]string{{"F001"}, {"F002"}, {"F003"}}},
	{"new_crossdir_shared", 4, [][]string{{"F001", "F002"}, {"F003", "F004"}}},
	{"new_crossdir_collision", 4, [][]string{{"F001", "F002"}, {"F003", "F004"}}},
	{"new_lexical_bridge", 4, [][]string{{"F001", "F002"}, {"F003", "F004"}}},
	{"new_paraphrase", 3, [][]string{{"F001", "F002"}, {"F003"}}},
}

func issue142LoadHoldoutFixtures(filePath string) ([]fixture, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var source issue142HoldoutFileSet
	if err := decoder.Decode(&source); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("unexpected trailing JSON: %v", err)
	}
	if len(source.Fixtures) != len(issue142HoldoutContract) {
		return nil, fmt.Errorf("expected %d fixtures, got %d", len(issue142HoldoutContract), len(source.Fixtures))
	}
	result := make([]fixture, 0, len(source.Fixtures))
	for i, entry := range source.Fixtures {
		contract := issue142HoldoutContract[i]
		if entry.Name != contract.name || len(entry.Files) != contract.count || !sameGroups(entry.Gold, contract.gold) {
			return nil, fmt.Errorf("fixture %d violates preregistered name, count, or gold", i+1)
		}
		specs := make([]fileSpec, 0, len(entry.Files))
		seen := map[string]bool{}
		for _, file := range entry.Files {
			if file.Path == "" || file.Diff == "" || seen[file.Path] {
				return nil, fmt.Errorf("%s: invalid file", entry.Name)
			}
			seen[file.Path] = true
			specs = append(specs, fileSpec{path: file.Path, diff: file.Diff})
		}
		result = append(result, fixture{name: entry.Name, language: planning.English, files: specs, reference: entry.Gold})
	}
	return result, nil
}

func issue142CappedPairSupport(doc contextinput.Document, ids []string) (map[string]float64, error) {
	current, err := issue142PairSupport(doc, ids)
	if err != nil {
		return nil, err
	}
	lexicalEdges, err := issue142WeightedPairs(doc, ids)
	if err != nil {
		return nil, err
	}
	lexical := make(map[string]float64, len(lexicalEdges))
	for _, edge := range lexicalEdges {
		lexical[issue142PairKey(edge.IDs[0], edge.IDs[1])] += edge.Score
	}
	capped := make(map[string]float64, len(current))
	for key, total := range current {
		structural := total - lexical[key]
		if structural < 0 && structural > -1e-9 {
			structural = 0
		}
		if structural < 0 {
			return nil, fmt.Errorf("negative structural support")
		}
		if structural > issue142CappedStructural {
			structural = issue142CappedStructural
		}
		capped[key] = lexical[key] + structural
	}
	return capped, nil
}

type issue142CappedRow struct {
	Fixture string            `json:"fixture"`
	Current []issue142RankArm `json:"current"`
	Capped  []issue142RankArm `json:"capped"`
}

func runIssue142CappedHoldout(ctx context.Context, options issue140Options) error {
	items, err := issue142LoadHoldoutFixtures(issue142CappedFixturePath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for _, item := range items {
		if options.fixtureName != "all" && options.fixtureName != item.name {
			continue
		}
		matched = true
		input, err := issue142BuildRankInput(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		support, err := issue142CappedPairSupport(input.prepared.Document, input.ids)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		capped := make([]issue142RankArm, 0, len(input.arms))
		baseCount := len(input.arms[0].Candidates)
		for _, arm := range input.arms {
			candidates := make([]issue142Candidate, 0, len(arm.Candidates))
			for _, ranked := range arm.Candidates {
				candidates = append(candidates, ranked.issue142Candidate)
			}
			capped = append(capped, issue142Rank(arm.Name, candidates, baseCount, item.reference, support))
		}
		if err := encoder.Encode(issue142CappedRow{Fixture: item.name, Current: input.arms, Capped: capped}); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown holdout fixture %q", options.fixtureName)
	}
	return nil
}
