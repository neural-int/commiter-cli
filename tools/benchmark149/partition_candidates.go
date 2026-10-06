package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type partitionCandidate struct {
	ID     string     `json:"id"`
	Groups [][]string `json:"groups"`
}

// Observed relations propose complete alternatives only, never must-links.
func partitionCandidates(f fixture) ([]partitionCandidate, error) {
	ids := fixtureIDs(f)
	if len(ids) < 1 || len(ids) > 16 {
		return nil, errors.New("file_budget")
	}
	known := map[string]bool{}
	test := map[string]bool{}
	for _, file := range f.Files {
		if known[file.ID] {
			return nil, errors.New("duplicate_file")
		}
		known[file.ID] = true
		test[file.ID] = file.NewPath != nil && strings.HasSuffix(*file.NewPath, "_test.go")
	}
	edges := [][2]string{}
	for _, a := range assertionFacts(f) {
		if a.Callee != "" {
			edges = append(edges, [2]string{a.File, a.Callee})
		}
	}
	calls := [][2]string{}
	for _, c := range callFacts(f) {
		if !test[c.Caller] {
			calls = append(calls, [2]string{c.Caller, c.Callee})
		}
	}
	components := func(relations [][2]string) ([][]string, error) {
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
		for _, e := range relations {
			if !known[e[0]] || !known[e[1]] {
				return nil, errors.New("unknown_relation")
			}
			parent[root(e[0])] = root(e[1])
		}
		membership := map[string]string{}
		labels := map[string]string{}
		for _, id := range ids {
			r := root(id)
			if labels[r] == "" {
				labels[r] = groupID(len(labels))
			}
			membership[id] = labels[r]
		}
		return partition(ids, membership)
	}
	single, err := components(nil)
	if err != nil {
		return nil, err
	}
	pairs, err := components(edges)
	if err != nil {
		return nil, err
	}
	combined := append(append([][2]string{}, edges...), calls...)
	linked, err := components(combined)
	if err != nil {
		return nil, err
	}
	variants := [][][]string{single, pairs, linked, {append([]string{}, ids...)}}
	unique := map[string][][]string{}
	for _, groups := range variants {
		for _, g := range groups {
			sort.Strings(g)
		}
		sort.Slice(groups, func(i, j int) bool { return strings.Join(groups[i], "\x00") < strings.Join(groups[j], "\x00") })
		data, _ := json.Marshal(groups)
		unique[string(data)] = groups
	}
	keys := []string{}
	for k := range unique {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []partitionCandidate{}
	for i, k := range keys {
		out = append(out, partitionCandidate{fmt.Sprintf("C%03d", i+1), unique[k]})
	}
	return out, nil
}
