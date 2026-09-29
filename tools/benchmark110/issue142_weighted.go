package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

var issue142WeightedThresholds = []float64{0.75, 1.25}

type issue142WeightedEdge struct {
	IDs    []string `json:"ids"`
	Tokens []string `json:"tokens"`
	Score  float64  `json:"score"`
}

type issue142WeightedThresholdRow struct {
	Threshold         float64                `json:"threshold"`
	Groups            [][]string             `json:"groups"`
	Exact             bool                   `json:"exact"`
	Edges             []issue142WeightedEdge `json:"edges"`
	GoldInternalEdges int                    `json:"gold_internal_edges"`
	GoldCrossEdges    int                    `json:"gold_cross_edges"`
}

type issue142WeightedRow struct {
	Fixture              string                         `json:"fixture"`
	BaselineCount        int                            `json:"baseline_count"`
	BinaryCount          int                            `json:"binary_count"`
	WeightedCount        int                            `json:"weighted_count"`
	BaselineGold         bool                           `json:"baseline_gold"`
	BinaryGold           bool                           `json:"binary_gold"`
	WeightedGold         bool                           `json:"weighted_gold"`
	BinaryAdded          []issue142Candidate            `json:"binary_added"`
	WeightedAdded        []issue142Candidate            `json:"weighted_added"`
	BinaryAddedNonGold   int                            `json:"binary_added_non_gold"`
	WeightedAddedNonGold int                            `json:"weighted_added_non_gold"`
	BinaryCap            bool                           `json:"binary_cap"`
	WeightedCap          bool                           `json:"weighted_cap"`
	Thresholds           []issue142WeightedThresholdRow `json:"thresholds"`
}

func issue142AllLexicalFixtures() []fixture {
	return append(issue142TournamentFixtures(), issue142NextFixtures()...)
}

func issue142WeightedPairs(doc contextinput.Document, ids []string) ([]issue142WeightedEdge, error) {
	files := map[string]contextinput.File{}
	for _, file := range doc.Files {
		files[file.ID] = file
	}
	tokens := map[string]map[string]bool{}
	frequency := map[string]int{}
	for _, id := range ids {
		file, ok := files[id]
		if !ok {
			return nil, fmt.Errorf("missing file %s", id)
		}
		tokens[id] = issue142LexicalTokens(file)
		for token := range tokens[id] {
			frequency[token]++
		}
	}
	edges := []issue142WeightedEdge{}
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			shared := []string{}
			score := 0.0
			for token := range tokens[a] {
				if !tokens[b][token] {
					continue
				}
				shared = append(shared, token)
				score += 1 / float64(frequency[token]-1)
			}
			if len(shared) == 0 {
				continue
			}
			sort.Strings(shared)
			edges = append(edges, issue142WeightedEdge{IDs: []string{a, b}, Tokens: shared, Score: score})
		}
	}
	return edges, nil
}

func issue142WeightedPartition(ids []string, edges []issue142WeightedEdge, threshold float64) ([][]string, []issue142WeightedEdge, error) {
	parent := map[string]string{}
	for _, id := range ids {
		parent[id] = id
	}
	var root func(string) string
	root = func(id string) string {
		if parent[id] != id {
			parent[id] = root(parent[id])
		}
		return parent[id]
	}
	accepted := []issue142WeightedEdge{}
	for _, edge := range edges {
		if edge.Score < threshold {
			continue
		}
		accepted = append(accepted, edge)
		a, b := root(edge.IDs[0]), root(edge.IDs[1])
		if a != b {
			parent[b] = a
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
	normalized, _, err := issue142Canonical(groups, ids)
	return normalized, accepted, err
}

func issue142AddPartitions(base []issue142Candidate, ids []string, partitions [][][]string) ([]issue142Candidate, []issue142Candidate, bool, error) {
	result := append([]issue142Candidate(nil), base...)
	added := []issue142Candidate{}
	capReached := len(result) >= issue142TournamentCap
	for _, groups := range partitions {
		normalized, _, err := issue142Canonical(groups, ids)
		if err != nil {
			return nil, nil, false, err
		}
		duplicate := false
		for _, candidate := range result {
			if sameGroups(candidate.Groups, normalized) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		if len(result) >= issue142TournamentCap {
			capReached = true
			continue
		}
		candidate := issue142Candidate{ID: fmt.Sprintf("C%03d", len(result)+1), Groups: normalized}
		result = append(result, candidate)
		added = append(added, candidate)
		if len(result) >= issue142TournamentCap {
			capReached = true
		}
	}
	return result, added, capReached, nil
}

func runIssue142Weighted(ctx context.Context, options issue140Options) error {
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for _, item := range issue142AllLexicalFixtures() {
		if options.fixtureName != "all" && options.fixtureName != item.name {
			continue
		}
		matched = true
		prepared, ids, err := issue140Prepare(ctx, item, issue140ManyArms[5], 1024)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		raw, _, err := issue142Generate(prepared.Document, ids)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		base, err := issue142ExpandCandidates(prepared.Document, ids, raw)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		binaryGroups, _, err := issue142LexicalPartition(prepared.Document, ids)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		binary, binaryAdded, binaryCap, err := issue142AddPartitions(base, ids, [][][]string{binaryGroups})
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		edges, err := issue142WeightedPairs(prepared.Document, ids)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		thresholds := []issue142WeightedThresholdRow{}
		partitions := [][][]string{}
		for _, threshold := range issue142WeightedThresholds {
			groups, accepted, err := issue142WeightedPartition(ids, edges, threshold)
			if err != nil {
				return fmt.Errorf("%s: %w", item.name, err)
			}
			part := issue142WeightedThresholdRow{Threshold: threshold, Groups: groups, Exact: sameGroups(groups, item.reference), Edges: accepted}
			for _, edge := range accepted {
				same := false
				for _, goldGroup := range item.reference {
					a, b := false, false
					for _, id := range goldGroup {
						a = a || id == edge.IDs[0]
						b = b || id == edge.IDs[1]
					}
					if a && b {
						same = true
						break
					}
				}
				if same {
					part.GoldInternalEdges++
				} else {
					part.GoldCrossEdges++
				}
			}
			thresholds = append(thresholds, part)
			partitions = append(partitions, groups)
		}
		weighted, weightedAdded, weightedCap, err := issue142AddPartitions(base, ids, partitions)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		baseGold, _ := issue142HasGold(base, item.reference)
		binaryGold, _ := issue142HasGold(binary, item.reference)
		weightedGold, _ := issue142HasGold(weighted, item.reference)
		row := issue142WeightedRow{Fixture: item.name, BaselineCount: len(base), BinaryCount: len(binary), WeightedCount: len(weighted),
			BaselineGold: baseGold, BinaryGold: binaryGold, WeightedGold: weightedGold,
			BinaryAdded: binaryAdded, WeightedAdded: weightedAdded, BinaryCap: binaryCap, WeightedCap: weightedCap, Thresholds: thresholds}
		for _, candidate := range binaryAdded {
			if !sameGroups(candidate.Groups, item.reference) {
				row.BinaryAddedNonGold++
			}
		}
		for _, candidate := range weightedAdded {
			if !sameGroups(candidate.Groups, item.reference) {
				row.WeightedAddedNonGold++
			}
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown weighted fixture %q", options.fixtureName)
	}
	return nil
}
