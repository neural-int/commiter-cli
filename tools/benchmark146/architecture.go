// Package main is an experimental Issue #146 harness, not a production planner.
package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/natsuki0413/commiter-cli/internal/relation"
)

type windowResult struct {
	IDs    []string
	Groups [][]string
}
type partition struct {
	Groups  [][]string
	Unknown [][2]string
	Reason  string
}
type pair [2]string

func key(a, b string) pair {
	if a > b {
		a, b = b, a
	}
	return pair{a, b}
}

// Only validated local equality/inequality becomes a constraint. Graph edges
// choose presentations, never union files. Missing edges are not inequalities.
func reconcile(ids []string, results []windowResult) partition {
	parent := map[string]string{}
	for _, id := range ids {
		if id == "" || parent[id] != "" {
			return partition{Reason: "invalid_selected_ids"}
		}
		parent[id] = id
	}
	var find func(string) string
	find = func(id string) string {
		if parent[id] != id {
			parent[id] = find(parent[id])
		}
		return parent[id]
	}
	observations := map[pair]bool{}
	observed := map[string]bool{}
	for _, r := range results {
		members := map[string]int{}
		supplied := map[string]bool{}
		if len(r.IDs) == 0 || len(r.IDs) > 4 {
			return partition{Reason: "invalid_window_size"}
		}
		for _, id := range r.IDs {
			if parent[id] == "" || supplied[id] {
				return partition{Reason: "unknown_or_duplicate_window_id"}
			}
			supplied[id] = true
			observed[id] = true
		}
		for g, group := range r.Groups {
			if len(group) == 0 {
				return partition{Reason: "empty_local_group"}
			}
			for _, id := range group {
				if !supplied[id] {
					return partition{Reason: "unknown_local_id"}
				}
				if _, ok := members[id]; ok {
					return partition{Reason: "duplicate_local_id"}
				}
				members[id] = g
			}
		}
		if len(members) != len(supplied) {
			return partition{Reason: "missing_local_id"}
		}
		for i, a := range r.IDs {
			for _, b := range r.IDs[i+1:] {
				p := key(a, b)
				same := members[a] == members[b]
				if old, ok := observations[p]; ok && old != same {
					return partition{Reason: "contradictory_pair"}
				}
				observations[p] = same
			}
		}
	}
	if len(observed) != len(ids) {
		return partition{Reason: "missing_global_id"}
	}
	for p, same := range observations {
		if same {
			a, b := find(p[0]), find(p[1])
			if a > b {
				a, b = b, a
			}
			parent[b] = a
		}
	}
	negative := map[pair]bool{}
	for p, same := range observations {
		if !same {
			a, b := find(p[0]), find(p[1])
			if a == b {
				return partition{Reason: "transitive_contradiction"}
			}
			negative[key(a, b)] = true
		}
	}
	buckets := map[string][]string{}
	for _, id := range ids {
		root := find(id)
		buckets[root] = append(buckets[root], id)
	}
	roots := []string{}
	for root := range buckets {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	out := partition{}
	for _, root := range roots {
		sort.Strings(buckets[root])
		out.Groups = append(out.Groups, buckets[root])
	}
	for i, a := range roots {
		for _, b := range roots[i+1:] {
			if !negative[key(a, b)] {
				out.Unknown = append(out.Unknown, [2]string{a, b})
			}
		}
	}
	if len(out.Unknown) > 0 {
		out.Reason = "unobserved_group_relation"
	}
	return out
}

// Cover candidate edges with <=4-file windows, then cover all remaining files.
// Sorting and lexicographic tie breaks make input/map ordering irrelevant.
func initialWindows(graph relation.CandidateGraph) ([][]string, error) {
	ids := []string{}
	known := map[string]bool{}
	for _, node := range graph.Nodes {
		if node.FileID == "" || known[node.FileID] {
			return nil, errors.New("invalid graph nodes")
		}
		known[node.FileID] = true
		ids = append(ids, node.FileID)
	}
	sort.Strings(ids)
	if len(ids) <= 4 {
		if len(ids) == 0 {
			return nil, errors.New("empty graph")
		}
		return [][]string{ids}, nil
	}
	edges := map[pair]bool{}
	for _, e := range graph.Edges {
		if !known[e.SourceID] || !known[e.TargetID] {
			return nil, errors.New("unknown graph endpoint")
		}
		if e.SourceID != e.TargetID && e.Kind != relation.PathProximity {
			edges[key(e.SourceID, e.TargetID)] = true
		}
	}
	windows := [][]string{}
	covered := map[string]bool{}
	for len(edges) > 0 {
		pairs := []pair{}
		for p := range edges {
			pairs = append(pairs, p)
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i][0] != pairs[j][0] {
				return pairs[i][0] < pairs[j][0]
			}
			return pairs[i][1] < pairs[j][1]
		})
		w := []string{pairs[0][0], pairs[0][1]}
		for len(w) < 4 {
			best := ""
			score := 0
			for _, id := range ids {
				if contains(w, id) {
					continue
				}
				s := 0
				for _, other := range w {
					if edges[key(id, other)] {
						s++
					}
				}
				if s > score {
					best, score = id, s
				}
			}
			if best == "" {
				break
			}
			w = append(w, best)
		}
		sort.Strings(w)
		windows = append(windows, w)
		for _, id := range w {
			covered[id] = true
		}
		for i, a := range w {
			for _, b := range w[i+1:] {
				delete(edges, key(a, b))
			}
		}
	}
	spare := []string{}
	for _, id := range ids {
		if !covered[id] {
			spare = append(spare, id)
		}
	}
	for len(spare) > 0 {
		n := min(4, len(spare))
		windows = append(windows, append([]string(nil), spare[:n]...))
		spare = spare[n:]
	}
	return windows, nil
}
func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// Query representatives of globally unresolved components. A negative witness
// separates two positive equivalence classes; no negative witness is invented.
func bridgeWindow(p partition) []string {
	if len(p.Unknown) == 0 {
		return nil
	}
	w := []string{p.Unknown[0][0], p.Unknown[0][1]}
	for len(w) < 4 {
		best := ""
		score := 0
		for _, group := range p.Groups {
			id := group[0]
			if contains(w, id) {
				continue
			}
			s := 0
			for _, u := range p.Unknown {
				if u[0] == id && contains(w, u[1]) || u[1] == id && contains(w, u[0]) {
					s++
				}
			}
			if s > score {
				best, score = id, s
			}
		}
		if best == "" {
			break
		}
		w = append(w, best)
	}
	sort.Strings(w)
	return w
}
func idsFor(graph relation.CandidateGraph) []string {
	ids := []string{}
	for _, n := range graph.Nodes {
		ids = append(ids, n.FileID)
	}
	sort.Strings(ids)
	return ids
}
func signature(groups [][]string) string {
	parts := []string{}
	for _, g := range groups {
		copy := append([]string(nil), g...)
		sort.Strings(copy)
		parts = append(parts, fmt.Sprint(copy))
	}
	sort.Strings(parts)
	return fmt.Sprint(parts)
}

// auditWindow covers file pairs not directly presented together. It probes
// context dependence without treating inferred equivalence as another vote.
func auditWindow(ids []string, results []windowResult) []string {
	ids = append([]string(nil), ids...)
	sort.Strings(ids)
	observed := map[pair]bool{}
	for _, r := range results {
		for i, a := range r.IDs {
			for _, b := range r.IDs[i+1:] {
				observed[key(a, b)] = true
			}
		}
	}
	var w []string
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			if !observed[key(a, b)] {
				w = []string{a, b}
				break
			}
		}
		if len(w) > 0 {
			break
		}
	}
	if len(w) == 0 {
		return nil
	}
	for len(w) < 4 {
		best, score := "", 0
		for _, id := range ids {
			if contains(w, id) {
				continue
			}
			n := 0
			for _, other := range w {
				if !observed[key(id, other)] {
					n++
				}
			}
			if n > score {
				best, score = id, n
			}
		}
		if best == "" {
			break
		}
		w = append(w, best)
	}
	sort.Strings(w)
	return w
}
