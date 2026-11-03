package planning

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

// Explicit synthetic attribution for migrated contract tests. Production never
// converts path-only records into matched evidence.
func syntheticAttributedHistory(t *testing.T, prepared contextinput.Prepared, records []codexcontext.Record) *GroupingHistory {
	t.Helper()
	history := &GroupingHistory{Acquisition: AgentHistoryAcquisition{Status: "complete", Reasons: []string{}}}
	seen := map[string]bool{}
	for _, record := range records {
		data, _ := json.Marshal(record)
		if seen[string(data)] {
			continue
		}
		seen[string(data)] = true
		history.Contexts = append(history.Contexts, AgentHistoryContext{ContextRef: fmt.Sprintf("H%d", len(history.Contexts)+1), UserRequests: []string{"Nonempty request must never reach the model"}, FinalResponses: []string{record.FinalResponse}})
	}
	for _, file := range prepared.Document.Files {
		projected, err := ProjectMetadataFile(file)
		if err != nil {
			t.Fatal(err)
		}
		attributed := AttributedFileHistory{FileID: file.ID, RawDiff: projected.RawDiff}
		for _, block := range projected.ChangeBlocks {
			candidates := []AgentHistoryCandidate{}
			for _, record := range history.Contexts {
				candidates = append(candidates, AgentHistoryCandidate{record.ContextRef, "exact"})
			}
			status := "matched"
			reasons := []string{}
			if len(candidates) > 1 {
				status = "ambiguous"
				reasons = []string{"multiple_history_candidates"}
			}
			attributed.Blocks = append(attributed.Blocks, AttributedBlockHistory{BlockID: block.BlockID, DiffRange: block.DiffRange, Status: status, Reasons: reasons, Candidates: candidates})
		}
		history.Files = append(history.Files, attributed)
	}
	return history
}
