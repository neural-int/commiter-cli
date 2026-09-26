package relation

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/gitstate"
)

type CandidateNode struct {
	FileID  string  `json:"file_id"`
	OldPath *string `json:"old_path,omitempty"`
	NewPath *string `json:"new_path,omitempty"`
	Renamed bool    `json:"renamed,omitempty"`
}

type CandidateComponent struct {
	ID      string   `json:"id"`
	FileIDs []string `json:"file_ids"`
}

type ReductionReason struct {
	Kind   Kind   `json:"kind"`
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type GraphStatistics struct {
	NodeCount             int             `json:"node_count"`
	EdgeCount             int             `json:"edge_count"`
	EdgesByKind           map[Kind]int    `json:"edges_by_kind"`
	ObservationCount      int             `json:"observation_count"`
	ObservationsByKind    map[Kind]int    `json:"observations_by_kind"`
	ObservationsByOutcome map[Outcome]int `json:"observations_by_outcome"`
	ComponentCount        int             `json:"component_count"`
	DensePairBaseline     int64           `json:"dense_pair_baseline"`
	CandidatePairCount    int64           `json:"candidate_pair_count"`
	ReducedPairCount      int64           `json:"reduced_pair_count"`
	PairReductionReason   string          `json:"pair_reduction_reason,omitempty"`
	AuxiliaryHintCount    int             `json:"auxiliary_hint_count"`
}

// CandidateGraph retains every changed file while using only evidence-backed
// relations to connect candidate components. Components are candidate spaces,
// not commit assignments.
type CandidateGraph struct {
	Nodes            []CandidateNode      `json:"nodes"`
	Edges            []Relation           `json:"edges"`
	Forward          map[string][]string  `json:"forward"`
	Reverse          map[string][]string  `json:"reverse"`
	Components       []CandidateComponent `json:"components"`
	Observations     []Observation        `json:"observations,omitempty"`
	Hints            []Hint               `json:"hints,omitempty"`
	ReductionReasons []ReductionReason    `json:"reduction_reasons,omitempty"`
	Statistics       GraphStatistics      `json:"statistics"`
}

// BuildGraph creates a deterministic sparse candidate graph from a complete
// change set and its extraction result. Weak path-proximity hints are retained
// without adding pairwise connectivity.
func BuildGraph(changes []gitstate.Change, extracted Result) (CandidateGraph, error) {
	graph := CandidateGraph{
		Nodes:        make([]CandidateNode, 0, len(changes)),
		Edges:        []Relation{},
		Forward:      make(map[string][]string, len(changes)),
		Reverse:      make(map[string][]string, len(changes)),
		Components:   []CandidateComponent{},
		Observations: cloneObservations(extracted.Observations),
		Hints:        cloneHints(extracted.Hints),
	}
	nodeIndex := make(map[string]int, len(changes))
	for _, change := range changes {
		if change.ID == "" {
			return CandidateGraph{}, errors.New("candidate graph input has an empty file ID")
		}
		if _, exists := nodeIndex[change.ID]; exists {
			return CandidateGraph{}, errors.New("candidate graph input has duplicate file IDs")
		}
		nodeIndex[change.ID] = len(graph.Nodes)
		graph.Nodes = append(graph.Nodes, CandidateNode{
			FileID:  change.ID,
			OldPath: clonePath(change.OldPath),
			NewPath: clonePath(change.NewPath),
			Renamed: change.Status == "renamed",
		})
		graph.Forward[change.ID] = []string{}
		graph.Reverse[change.ID] = []string{}
	}
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].FileID < graph.Nodes[j].FileID })
	for index, node := range graph.Nodes {
		nodeIndex[node.FileID] = index
	}
	sort.Slice(graph.Observations, func(i, j int) bool { return observationLess(graph.Observations[i], graph.Observations[j]) })
	sort.Slice(graph.Hints, func(i, j int) bool { return hintLess(graph.Hints[i], graph.Hints[j]) })
	for _, observation := range graph.Observations {
		if _, exists := nodeIndex[observation.SourceID]; !exists {
			return CandidateGraph{}, fmt.Errorf("candidate graph observation references missing file ID %s", observation.SourceID)
		}
		for _, candidateID := range observation.CandidateIDs {
			if _, exists := nodeIndex[candidateID]; !exists {
				return CandidateGraph{}, fmt.Errorf("candidate graph observation references missing candidate file ID %s", candidateID)
			}
		}
	}
	for _, hint := range graph.Hints {
		for _, fileID := range hint.FileIDs {
			if _, exists := nodeIndex[fileID]; !exists {
				return CandidateGraph{}, fmt.Errorf("candidate graph hint references missing file ID %s", fileID)
			}
		}
	}
	pruned := make(map[string]ReductionReason)
	for _, relation := range extracted.Relations {
		source, sourceExists := nodeIndex[relation.SourceID]
		_, targetExists := nodeIndex[relation.TargetID]
		if !sourceExists || !targetExists {
			return CandidateGraph{}, fmt.Errorf("candidate graph relation references missing node %s -> %s", relation.SourceID, relation.TargetID)
		}
		if relation.Kind == GitRename && relation.SourceID == relation.TargetID {
			graph.Nodes[source].Renamed = true
			addReductionReason(pruned, GitRename, "rename_retained_as_single_node_metadata")
			continue
		}
		if relation.Kind == PathProximity {
			addReductionReason(pruned, PathProximity, "path_proximity_is_auxiliary_evidence")
			continue
		}
		if relation.SourceID == relation.TargetID {
			addReductionReason(pruned, relation.Kind, "self_relation_does_not_connect_component")
			continue
		}
		graph.Edges = append(graph.Edges, relation)
		graph.Forward[relation.SourceID] = appendUnique(graph.Forward[relation.SourceID], relation.TargetID)
		graph.Reverse[relation.TargetID] = appendUnique(graph.Reverse[relation.TargetID], relation.SourceID)
	}
	sort.Slice(graph.Edges, func(i, j int) bool { return relationLess(graph.Edges[i], graph.Edges[j]) })
	for fileID := range graph.Forward {
		sort.Strings(graph.Forward[fileID])
		sort.Strings(graph.Reverse[fileID])
	}
	graph.Components = candidateComponents(graph.Nodes, graph.Edges)
	graph.ReductionReasons = make([]ReductionReason, 0, len(pruned))
	for _, reason := range pruned {
		graph.ReductionReasons = append(graph.ReductionReasons, reason)
	}
	sort.Slice(graph.ReductionReasons, func(i, j int) bool {
		if graph.ReductionReasons[i].Kind != graph.ReductionReasons[j].Kind {
			return graph.ReductionReasons[i].Kind < graph.ReductionReasons[j].Kind
		}
		return graph.ReductionReasons[i].Reason < graph.ReductionReasons[j].Reason
	})
	graph.Statistics = graphStatistics(graph)
	return graph, nil
}

func cloneObservations(observations []Observation) []Observation {
	cloned := make([]Observation, len(observations))
	for i, observation := range observations {
		cloned[i] = observation
		cloned[i].CandidateIDs = append([]string(nil), observation.CandidateIDs...)
		sort.Strings(cloned[i].CandidateIDs)
	}
	return cloned
}

func cloneHints(hints []Hint) []Hint {
	cloned := make([]Hint, len(hints))
	for i, hint := range hints {
		cloned[i] = hint
		cloned[i].FileIDs = append([]string(nil), hint.FileIDs...)
		sort.Strings(cloned[i].FileIDs)
	}
	return cloned
}

func observationLess(a, b Observation) bool {
	if a.SourceID != b.SourceID {
		return a.SourceID < b.SourceID
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Outcome != b.Outcome {
		return a.Outcome < b.Outcome
	}
	if a.Reason != b.Reason {
		return a.Reason < b.Reason
	}
	if a.Evidence.Type != b.Evidence.Type {
		return a.Evidence.Type < b.Evidence.Type
	}
	if a.Evidence.Value != b.Evidence.Value {
		return a.Evidence.Value < b.Evidence.Value
	}
	if a.Evidence.Related != b.Evidence.Related {
		return a.Evidence.Related < b.Evidence.Related
	}
	return strings.Join(a.CandidateIDs, "\x00") < strings.Join(b.CandidateIDs, "\x00")
}

func hintLess(a, b Hint) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Evidence.Type != b.Evidence.Type {
		return a.Evidence.Type < b.Evidence.Type
	}
	if a.Evidence.Value != b.Evidence.Value {
		return a.Evidence.Value < b.Evidence.Value
	}
	if a.Evidence.Related != b.Evidence.Related {
		return a.Evidence.Related < b.Evidence.Related
	}
	return strings.Join(a.FileIDs, "\x00") < strings.Join(b.FileIDs, "\x00")
}

func clonePath(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func addReductionReason(reasons map[string]ReductionReason, kind Kind, reason string) {
	key := string(kind) + "\x00" + reason
	entry := reasons[key]
	entry.Kind = kind
	entry.Reason = reason
	entry.Count++
	reasons[key] = entry
}

func relationLess(a, b Relation) bool {
	if a.SourceID != b.SourceID {
		return a.SourceID < b.SourceID
	}
	if a.TargetID != b.TargetID {
		return a.TargetID < b.TargetID
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Class != b.Class {
		return a.Class < b.Class
	}
	if a.Reason != b.Reason {
		return a.Reason < b.Reason
	}
	if a.Evidence.Type != b.Evidence.Type {
		return a.Evidence.Type < b.Evidence.Type
	}
	if a.Evidence.Value != b.Evidence.Value {
		return a.Evidence.Value < b.Evidence.Value
	}
	return a.Evidence.Related < b.Evidence.Related
}

func candidateComponents(nodes []CandidateNode, edges []Relation) []CandidateComponent {
	neighbors := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		neighbors[node.FileID] = []string{}
	}
	for _, edge := range edges {
		neighbors[edge.SourceID] = appendUnique(neighbors[edge.SourceID], edge.TargetID)
		neighbors[edge.TargetID] = appendUnique(neighbors[edge.TargetID], edge.SourceID)
	}
	for fileID := range neighbors {
		sort.Strings(neighbors[fileID])
	}
	visited := make(map[string]bool, len(nodes))
	components := make([]CandidateComponent, 0, len(nodes))
	for _, node := range nodes {
		if visited[node.FileID] {
			continue
		}
		queue := []string{node.FileID}
		visited[node.FileID] = true
		fileIDs := make([]string, 0, 1)
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			fileIDs = append(fileIDs, current)
			for _, neighbor := range neighbors[current] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		sort.Strings(fileIDs)
		components = append(components, CandidateComponent{ID: fmt.Sprintf("C%03d", len(components)+1), FileIDs: fileIDs})
	}
	return components
}

func graphStatistics(graph CandidateGraph) GraphStatistics {
	stats := GraphStatistics{
		NodeCount:             len(graph.Nodes),
		EdgeCount:             len(graph.Edges),
		EdgesByKind:           make(map[Kind]int),
		ObservationCount:      len(graph.Observations),
		ObservationsByKind:    make(map[Kind]int),
		ObservationsByOutcome: make(map[Outcome]int),
		ComponentCount:        len(graph.Components),
		AuxiliaryHintCount:    len(graph.Hints),
	}
	for _, edge := range graph.Edges {
		stats.EdgesByKind[edge.Kind]++
	}
	for _, observation := range graph.Observations {
		stats.ObservationsByKind[observation.Kind]++
		stats.ObservationsByOutcome[observation.Outcome]++
	}
	n := int64(stats.NodeCount)
	stats.DensePairBaseline = n * (n - 1) / 2
	for _, component := range graph.Components {
		n := int64(len(component.FileIDs))
		stats.CandidatePairCount += n * (n - 1) / 2
	}
	stats.ReducedPairCount = stats.DensePairBaseline - stats.CandidatePairCount
	if stats.ReducedPairCount > 0 {
		stats.PairReductionReason = "no_evidence_path_between_candidate_components"
	}
	return stats
}
