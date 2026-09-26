package contextinput

import (
	"fmt"

	"github.com/natsuki0413/commiter-cli/internal/relation"
)

// RelationContextFromGraph keeps the planning payload to candidate groups,
// evidence-backed edges, auxiliary hints, and aggregate extraction diagnostics.
// Per-file nodes and adjacency maps are omitted because files and edges already
// carry that information.
func RelationContextFromGraph(graph relation.CandidateGraph) *RelationContext {
	return &RelationContext{
		Components:       graph.Components,
		Edges:            graph.Edges,
		Hints:            graph.Hints,
		ReductionReasons: graph.ReductionReasons,
		Statistics:       graph.Statistics,
	}
}

func omittedRelationStatus(context *RelationContext, reason string) *RelationContextStatus {
	status := &RelationContextStatus{Omitted: true, Reason: reason}
	if context == nil {
		return status
	}
	status.ObservationCount = max(0, context.Statistics.ObservationCount)
	for _, outcome := range []relation.Outcome{relation.Ambiguous, relation.Unresolved, relation.Unsupported} {
		if count := context.Statistics.ObservationsByOutcome[outcome]; count > 0 {
			if status.ObservationsByOutcome == nil {
				status.ObservationsByOutcome = make(map[relation.Outcome]int, 3)
			}
			status.ObservationsByOutcome[outcome] = count
		}
	}
	return status
}

func validateRelationContext(document Document) error {
	if document.RelationContext == nil {
		return nil
	}
	fileIDs := make(map[string]bool, len(document.Files))
	for _, file := range document.Files {
		if file.ID == "" || fileIDs[file.ID] {
			return fmt.Errorf("relation context requires unique file IDs")
		}
		fileIDs[file.ID] = true
	}
	componentIDs := make(map[string]bool, len(document.RelationContext.Components))
	componentFiles := make(map[string]bool, len(fileIDs))
	for _, component := range document.RelationContext.Components {
		if component.ID == "" || componentIDs[component.ID] || len(component.FileIDs) == 0 {
			return fmt.Errorf("relation context contains an invalid component")
		}
		componentIDs[component.ID] = true
		for _, fileID := range component.FileIDs {
			if !fileIDs[fileID] || componentFiles[fileID] {
				return fmt.Errorf("relation context components contain an unknown or duplicate file ID")
			}
			componentFiles[fileID] = true
		}
	}
	if len(componentFiles) != len(fileIDs) {
		return fmt.Errorf("relation context components do not cover every file ID")
	}
	for _, edge := range document.RelationContext.Edges {
		if !fileIDs[edge.SourceID] || !fileIDs[edge.TargetID] || edge.SourceID == edge.TargetID || edge.Kind == relation.PathProximity || edge.Reason == "" || edge.Evidence.Type == "" {
			return fmt.Errorf("relation context contains an invalid edge")
		}
	}
	for _, hint := range document.RelationContext.Hints {
		if hint.Kind != relation.PathProximity || len(hint.FileIDs) < 2 {
			return fmt.Errorf("relation context contains an invalid auxiliary hint")
		}
		for _, fileID := range hint.FileIDs {
			if !fileIDs[fileID] {
				return fmt.Errorf("relation context hint references an unknown file ID")
			}
		}
	}
	return nil
}
