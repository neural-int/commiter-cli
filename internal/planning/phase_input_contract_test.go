package planning

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

func TestFinalPhaseInputContractsAndBreaking(t *testing.T) {
	for _, breaking := range []bool{false, true} {
		h, p := attributedFixture(t)
		c := &candidateClient{responses: candidateResponses()}
		g := ThreePhaseGenerator{Client: c, GroupingHistory: h}
		if breaking {
			g.Evidence = []BreakingEvidence{{ID: "E1", FileIDs: []string{"F001"}, Fact: "observed incompatible contract"}}
			c.responses[1] = `{"G001":{"type":"fix","breaking_evidence_ref":"E1"}}`
		}
		result, err := g.Generate(context.Background(), p, English, SensitiveValues{})
		if err != nil {
			t.Fatal(err)
		}
		if result.Plan.Commits[0].Breaking != breaking {
			t.Fatal("host breaking judgment changed")
		}
		for phase := 1; phase <= 2; phase++ {
			var input map[string]json.RawMessage
			if err := json.Unmarshal([]byte(c.messages[phase][1].Content), &input); err != nil {
				t.Fatal(err)
			}
			required := []string{"schema_version", "task", "agent_history_contexts", "groups"}
			if phase == 1 {
				required = append(required, "breaking_evidence_candidates")
			} else {
				required = append(required, "summary_language")
			}
			if len(input) != len(required) {
				t.Fatalf("phase %d unexpected root keys: %v", phase, input)
			}
			for _, key := range required {
				if input[key] == nil {
					t.Fatalf("missing %s", key)
				}
			}
			var groups []struct {
				ID       string                       `json:"group_id"`
				Files    []map[string]json.RawMessage `json:"files"`
				Type     *string                      `json:"type"`
				Breaking *bool                        `json:"breaking"`
			}
			if err := json.Unmarshal(input["groups"], &groups); err != nil {
				t.Fatal(err)
			}
			if groups[0].ID != "G001" {
				t.Fatal("group identity changed")
			}
			if phase == 1 && (groups[0].Type != nil || groups[0].Breaking != nil) {
				t.Fatal("Category classification predetermined")
			}
			if phase == 2 && (groups[0].Type == nil || *groups[0].Type != "fix" || groups[0].Breaking == nil || *groups[0].Breaking != breaking) {
				t.Fatal("Text lost finalized type/breaking, including false")
			}
			for index, file := range groups[0].Files {
				if string(file["file_id"]) != []string{`"F1"`, `"F2"`}[index] {
					t.Fatal("file IDs changed between phases")
				}
				for _, key := range []string{"file_id", "status", "raw_diff", "change_blocks"} {
					if file[key] == nil {
						t.Fatal("missing " + key)
					}
				}
				for _, key := range []string{"id", "mode", "worktree_kind", "head_identity", "enclosing_declaration_name"} {
					if file[key] != nil {
						t.Fatal("unexpected file key " + key)
					}
				}
			}
			if strings.Contains(string(input["groups"]), "enclosing_declaration_name") {
				t.Fatal("syntax names leaked to metadata")
			}
			var contexts []AgentHistoryContext
			json.Unmarshal(input["agent_history_contexts"], &contexts)
			if len(contexts) != 1 || contexts[0].ContextRef != "H1" {
				t.Fatal("history missing")
			}
		}
	}
}

func TestPhaseHistorySelectionIsUnionNotOmission(t *testing.T) {
	h, p := attributedFixture(t)
	files, err := groupingProjection([]contextinput.File{p.Document.Files[0]})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := groupingPayload(files, map[string]string{files[0].FileID: p.Document.Files[0].ID}, h)
	if err != nil {
		t.Fatal(err)
	}
	source := payload["files"].([]groupingFile)
	other := clonePhaseFiles(source)[0]
	other.FileID = "F3"
	source = append(source, other)
	source[1].ChangeBlocks[0].AgentHistoryCandidates = []AgentHistoryCandidate{{"H2", "exact"}}
	contexts := append(h.Contexts, AgentHistoryContext{"H2", []string{"other request"}, []string{"other response"}})
	contexts = append(contexts, AgentHistoryContext{"H-unused", []string{"unrelated history"}, nil})
	selected := []candidateGroup{{ID: "G001", Files: []contextinput.File{{ID: source[0].FileID}}}}
	input, err := metadataHistoryPayload(selected, source, contexts)
	if err != nil {
		t.Fatal(err)
	}
	data, err := budgetPhaseHistory(input, SensitiveValues{}, "system", 64000, 1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "H2") || strings.Contains(string(data), "H-unused") || strings.Contains(string(data), "agent_history_omission_reasons") {
		t.Fatal("scope selection treated as omission")
	}
	// Distinct groups may reference one shared context; define it only once.
	source[1].ChangeBlocks[0].AgentHistoryCandidates = []AgentHistoryCandidate{{"H1", "exact"}}
	selected = append(selected, candidateGroup{ID: "G002", Files: []contextinput.File{{ID: "F3"}}})
	input, err = metadataHistoryPayload(selected, source, contexts)
	if err != nil {
		t.Fatal(err)
	}
	data, err = budgetPhaseHistory(input, SensitiveValues{}, "system", 64000, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(input["agent_history_contexts"].([]AgentHistoryContext)) != 1 {
		t.Fatal("shared context repeated")
	}
	if strings.Contains(string(data), "agent_history_omission_reasons") {
		t.Fatal("dedup treated as omission")
	}
}

func TestMetadataRestartsFromOriginalHistoryAfterEarlierBudgetOmission(t *testing.T) {
	h, p := attributedFixture(t)
	h.Contexts[0].FinalResponses = []string{strings.Repeat("original history ", 1000)}
	files, err := groupingProjection([]contextinput.File{p.Document.Files[0]})
	if err != nil {
		t.Fatal(err)
	}
	original, err := groupingPayload(files, map[string]string{files[0].FileID: p.Document.Files[0].ID}, h)
	if err != nil {
		t.Fatal(err)
	}
	source := original["files"].([]groupingFile)
	contexts := original["agent_history_contexts"].([]AgentHistoryContext)
	before, _ := json.Marshal(source)
	if _, err := budgetPhaseHistory(original, SensitiveValues{}, "system", 6000, 1); err != nil {
		t.Fatal(err)
	}
	if len(original["agent_history_contexts"].([]AgentHistoryContext)[0].FinalResponses) != 0 {
		t.Fatal("fixture did not omit history")
	}
	input, err := metadataHistoryPayload([]candidateGroup{{ID: "G001", Files: []contextinput.File{{ID: files[0].FileID}}}}, source, contexts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := budgetPhaseHistory(input, SensitiveValues{}, "system", 64000, 1); err != nil {
		t.Fatal(err)
	}
	kept := input["agent_history_contexts"].([]AgentHistoryContext)
	if !reflect.DeepEqual(kept[0].FinalResponses, h.Contexts[0].FinalResponses) {
		t.Fatal("prior phase omission became permanent")
	}
	after, _ := json.Marshal(source)
	if string(before) != string(after) {
		t.Fatal("prior phase mutated original block history")
	}
	if _, exists := input["agent_history_projection"]; exists {
		t.Fatal("global projection returned to metadata input")
	}
}

func TestMetadataKeepsOnlyBlockLevelIncompleteHistory(t *testing.T) {
	h, p := attributedFixture(t)
	h.Acquisition.Status = "partial"
	h.Acquisition.Reasons = []string{"history_incomplete"}
	h.Files[0].Blocks[0].Status = "ambiguous"
	h.Files[0].Blocks[0].Reasons = []string{"history_incomplete"}
	c := &candidateClient{responses: candidateResponses()}
	if _, err := (ThreePhaseGenerator{Client: c, GroupingHistory: h}).Generate(context.Background(), p, English, SensitiveValues{}); err != nil {
		t.Fatal(err)
	}
	for phase := 1; phase <= 2; phase++ {
		wire := c.messages[phase][1].Content
		if strings.Contains(wire, "agent_history_acquisition") || strings.Contains(wire, "agent_history_projection") {
			t.Fatal("global history state leaked")
		}
		if !strings.Contains(wire, `"agent_history_match_reasons":["history_incomplete"]`) {
			t.Fatal("affected block lost incomplete history")
		}
	}
}
