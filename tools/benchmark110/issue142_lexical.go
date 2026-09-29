package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

var issue142LexicalWord = regexp.MustCompile(`[A-Z][a-z]+|[A-Z]+|[a-z]+`)
var issue142LexicalStop = map[string]bool{
	"bool": true, "case": true, "const": true, "docs": true, "document": true,
	"false": true, "for": true, "from": true, "func": true, "go": true,
	"html": true, "how": true, "if": true, "int": true, "md": true,
	"new": true, "nil": true, "return": true, "src": true, "string": true,
	"test": true, "the": true, "true": true, "when": true,
}

type issue142LexicalEdge struct {
	IDs    []string `json:"ids"`
	Tokens []string `json:"tokens"`
}

type issue142LexicalRow struct {
	Fixture       string                `json:"fixture"`
	BaselineCount int                   `json:"baseline_count"`
	BaselineGold  bool                  `json:"baseline_gold"`
	LexicalEdges  []issue142LexicalEdge `json:"lexical_edges"`
	LexicalGroups [][]string            `json:"lexical_groups"`
	LexicalGold   bool                  `json:"lexical_gold"`
	ExpandedCount int                   `json:"expanded_count"`
	ExpandedGold  bool                  `json:"expanded_gold"`
	Added         bool                  `json:"added"`
	AddedID       string                `json:"added_id,omitempty"`
	CapReached    bool                  `json:"cap_reached"`
}

func issue142LexicalTokens(file contextinput.File) map[string]bool {
	tokens := map[string]bool{}
	for _, word := range issue142LexicalWord.FindAllString(issue141FilePath(file)+" "+file.RawDiff, -1) {
		word = strings.ToLower(word)
		if len(word) >= 4 && !issue142LexicalStop[word] {
			tokens[word] = true
		}
	}
	return tokens
}

func issue142LexicalPartition(doc contextinput.Document, ids []string) ([][]string, []issue142LexicalEdge, error) {
	files := map[string]contextinput.File{}
	for _, file := range doc.Files {
		files[file.ID] = file
	}
	parent := map[string]string{}
	tokens := map[string]map[string]bool{}
	for _, id := range ids {
		file, ok := files[id]
		if !ok {
			return nil, nil, fmt.Errorf("missing file %s", id)
		}
		parent[id] = id
		tokens[id] = issue142LexicalTokens(file)
	}
	var root func(string) string
	root = func(id string) string {
		if parent[id] != id {
			parent[id] = root(parent[id])
		}
		return parent[id]
	}
	edges := []issue142LexicalEdge{}
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			shared := []string{}
			for token := range tokens[a] {
				if tokens[b][token] {
					shared = append(shared, token)
				}
			}
			if len(shared) == 0 {
				continue
			}
			sort.Strings(shared)
			edges = append(edges, issue142LexicalEdge{IDs: []string{a, b}, Tokens: shared})
			ra, rb := root(a), root(b)
			if ra != rb {
				parent[rb] = ra
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
	groups, _, err := issue142Canonical(groups, ids)
	return groups, edges, err
}

func runIssue142Lexical(ctx context.Context, options issue140Options) error {
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
		existing, err := issue142ExpandCandidates(prepared.Document, ids, baseline)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		groups, edges, err := issue142LexicalPartition(prepared.Document, ids)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		baseGold, _ := issue142HasGold(existing, item.reference)
		row := issue142LexicalRow{
			Fixture: item.name, BaselineCount: len(existing), BaselineGold: baseGold,
			LexicalEdges: edges, LexicalGroups: groups, LexicalGold: sameGroups(groups, item.reference),
			ExpandedCount: len(existing), ExpandedGold: baseGold, CapReached: len(existing) >= issue142TournamentCap,
		}
		duplicate := false
		for _, candidate := range existing {
			if sameGroups(candidate.Groups, groups) {
				duplicate = true
				break
			}
		}
		if !duplicate && !row.CapReached {
			row.Added = true
			row.AddedID = fmt.Sprintf("C%03d", len(existing)+1)
			existing = append(existing, issue142Candidate{ID: row.AddedID, Groups: groups})
			row.ExpandedCount = len(existing)
			row.ExpandedGold, _ = issue142HasGold(existing, item.reference)
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 lexical fixture %q", options.fixtureName)
	}
	return nil
}
