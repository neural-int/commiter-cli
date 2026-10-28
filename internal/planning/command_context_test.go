package planning

import (
	"context"
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"reflect"
	"strings"
	"testing"
)

func TestCommandsReachGroupingOnlyWithVisibleFileIDs(t *testing.T) {
	h, p := attributedFixture(t)
	h.Contexts = append(h.Contexts, AgentHistoryContext{"H2", []string{"Check this file"}, []string{"Checked"}})
	h.Commands = []AgentCommandExecutionContext{{"S1", "H2", "cat sample.go", []string{p.Document.Files[0].ID}}}
	before, _ := json.Marshal(h)
	c := &candidateClient{responses: candidateResponses()}
	_, err := (ThreePhaseGenerator{Client: c, GroupingHistory: h}).Generate(context.Background(), p, English, SensitiveValues{})
	if err != nil {
		t.Fatal(err)
	}
	var input struct {
		Commands []AgentCommandExecutionContext `json:"agent_command_execution_contexts"`
		Contexts []AgentHistoryContext          `json:"agent_history_contexts"`
	}
	if json.Unmarshal([]byte(c.messages[0][1].Content), &input) != nil || len(input.Commands) != 1 || len(input.Contexts) != 2 {
		t.Fatal("command-only context dropped")
	}
	if input.Commands[0].MentionedFileIDs[0] == p.Document.Files[0].ID || !strings.HasPrefix(input.Commands[0].MentionedFileIDs[0], "F") {
		t.Fatal("internal ID leaked")
	}
	for _, phase := range []int{1, 2} {
		if strings.Contains(c.messages[phase][1].Content, "agent_command_execution_contexts") || strings.Contains(c.messages[phase][1].Content, `"H2"`) {
			t.Fatal("commands leaked into metadata")
		}
	}
	after, _ := json.Marshal(h)
	if string(before) != string(after) {
		t.Fatal("host input mutated")
	}
}

func TestCommandBudgetRemovesAuxiliaryEvidenceBeforeBlockHistory(t *testing.T) {
	h, p := attributedFixture(t)
	files, err := groupingProjection([]contextinput.File{p.Document.Files[0]})
	if err != nil {
		t.Fatal(err)
	}
	h.Contexts = append(h.Contexts, AgentHistoryContext{"H2", []string{"Check file"}, []string{"Done"}})
	h.Commands = []AgentCommandExecutionContext{{"S1", "H2", strings.Repeat("long-command ", 10000), []string{p.Document.Files[0].ID}}}
	payload, err := groupingPayload(files, map[string]string{files[0].FileID: p.Document.Files[0].ID}, h)
	if err != nil {
		t.Fatal(err)
	}
	before := clonePhaseFiles(payload["files"].([]groupingFile))
	diagnostics := []CommandProjectionOmission{}
	data, err := budgetPhaseHistory(payload, SensitiveValues{}, "system", 16000, 1, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	json.Unmarshal(data, &wire)
	if string(wire["agent_command_execution_contexts"]) != "[]" || strings.Contains(string(wire["agent_history_contexts"]), "H2") || !strings.Contains(string(wire["agent_history_projection"]), "partial") {
		t.Fatal("invalid command omission")
	}
	if len(diagnostics) != 1 || diagnostics[0].Reason != "input_budget_exceeded" {
		t.Fatal(diagnostics)
	}
	var got []groupingFile
	json.Unmarshal(wire["files"], &got)
	if !reflect.DeepEqual(before, got) {
		t.Fatal("command budget modified block attribution")
	}
}

func TestCommandSafetyKeepsCredentialsButAllowsApprovedTestPaths(t *testing.T) {
	path := "src/SiteFooter.test.jsx"
	command := "sed -i x src/File.jsx; npm test -- " + path
	normalized := CommandSafetyText(command, []string{path})
	if ExtractSensitiveValues([]byte(normalized)).Contains([]byte(normalized)) {
		t.Fatal("approved test filename mistaken for token")
	}
	if !ExtractSensitiveValues([]byte(command)).Contains([]byte(command)) {
		t.Fatal("fixture no longer exercises filename false positive")
	}
	for _, secret := range []string{"Bearer example-credential", "abc.def.ghi", "password=confidential"} {
		candidate := command + "\n" + secret
		scanned := CommandSafetyText(candidate, []string{path})
		if !ExtractSensitiveValues([]byte(scanned)).Contains([]byte(scanned)) {
			t.Fatal("credential check bypassed")
		}
	}
	if CommandSafetyText("echo abc.def.ghi", []string{path}) != "echo abc.def.ghi" {
		t.Fatal("unrelated text altered")
	}
}
