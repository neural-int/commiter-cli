package planning

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
)

func TestAttributedDisplaySelectionPreservesDiffAndUncertainty(t *testing.T) {
	p := candidatePrepared(t)
	history := syntheticAttributedHistory(t, p, []codexcontext.Record{{FinalResponse: "First final"}, {FinalResponse: "Second final"}})
	history.Commands = []AgentCommandExecutionContext{{"S1", "H1", "cat one.go", []string{p.Document.Files[0].ID}}, {"S2", "H1", "cat one.go", []string{p.Document.Files[0].ID}}}
	files, err := groupingProjection(p.Document.Files)
	if err != nil {
		t.Fatal(err)
	}
	inverse := map[string]string{}
	for i := range files {
		inverse[files[i].FileID] = files[i].FileID
	}
	payload, err := groupingPayload(files, inverse, history)
	if err != nil {
		t.Fatal(err)
	}
	safePhaseHistory(payload, SensitiveValues{})
	encoded, _ := json.Marshal(payload)
	schema, _ := json.Marshal(candidateObject(map[string]any{}, []string{}))
	var common map[string]any
	messages, err := harness004Messages("bounded-grouping", encoded, schema, &common)
	if err != nil {
		t.Fatal(err)
	}
	input, err := attributed004History(messages)
	if err != nil {
		t.Fatal(err)
	}
	if len(input.RecordByteCounts) != 4 {
		t.Fatal("identical separate executions collapsed")
	}
	all := []int{0, 1, 2, 3}
	if input.SharedTemplate.Render(all) != messages[1].Content {
		t.Fatal("all-fit shared display changed")
	}
	_, phase, _ := strings.Cut(messages[2].Content, "\n\nPhase input:\n")
	if input.PhaseTemplate.Render(all) != phase {
		t.Fatal("all-fit phase display changed")
	}
	wire, _ := json.Marshal(input)
	if strings.Count(string(wire), "raw_diff") != len(p.Document.Files) {
		t.Fatal("Git diff appears more than once on IPC")
	}
	selected, err := selectedAttributed004Shared(input, []int{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	contexts := selected["agent_history_contexts"].([]any)
	if len(contexts) != 1 || contexts[0].(map[string]any)["context_ref"] != "H2" {
		t.Fatal("whole-record selection failed")
	}
	for _, file := range selected["files"].([]any) {
		for _, value := range file.(map[string]any)["change_blocks"].([]any) {
			block := value.(map[string]any)
			if block["agent_history_match_status"] != "ambiguous" {
				t.Fatal("omission manufactured unique match")
			}
			candidates := block["agent_history_candidates"].([]any)
			if len(candidates) != 1 || candidates[0].(map[string]any)["context_ref"] != "H2" {
				t.Fatal("dangling candidate")
			}
			if !strings.Contains(string(naturalJSON(block["agent_history_omission_reasons"])), "input_budget_exceeded") {
				t.Fatal("missing budget omission")
			}
		}
	}
	var selectedPhase map[string]any
	if err := json.Unmarshal([]byte(input.PhaseTemplate.Render([]int{1, 2, 3})), &selectedPhase); err != nil {
		t.Fatal(err)
	}
	if len(selectedPhase["agent_command_execution_contexts"].([]any)) != 0 {
		t.Fatal("command survived missing context")
	}
	if selectedPhase["agent_history_projection"].(map[string]any)["status"] != "partial" {
		t.Fatal("missing projection uncertainty")
	}
	if input.SharedTemplate.Render([]int{1, 2, 3}) != string(naturalJSON(selected)) {
		t.Fatal("selected prefix loses natural key order")
	}
	for _, invalid := range [][]int{nil, {-1}, {4}, {1, 1}, {1, 0}} {
		if _, err := selectedAttributed004Shared(input, invalid); err == nil {
			t.Fatal("invalid selection accepted")
		}
	}
}
