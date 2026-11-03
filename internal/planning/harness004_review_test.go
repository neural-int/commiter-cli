package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func requiredHarness004Size(messages []llm.Message) int {
	size := contextinput.TemplateReserve + 4096
	for _, message := range messages {
		size += len(message.Content)
	}
	return size
}

func TestHarness004HistoryFitsExactTokenBoundary(t *testing.T) {
	for _, delta := range []int{-1, 0, 1} {
		t.Run(fmt.Sprint(delta), func(t *testing.T) {
			record := codexcontext.Record{Paths: []string{"one.go"}, FinalResponse: "Corrected the boundary"}
			client := &candidateClient{responses: candidateResponses(), countInput: func(messages []llm.Message, options llm.Options) (int, error) {
				if options.ContextTokens != contextinput.Context64K {
					t.Fatal("count did not use fixed 64K")
				}
				if strings.Contains(messages[1].Content, "Corrected the boundary") {
					return 61440 + delta, nil
				}
				return 1000, nil
			}}
			result, err := (ThreePhaseGenerator{Client: client, GroupingHistory: syntheticAttributedHistory(t, candidatePrepared(t), []codexcontext.Record{record})}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
			if err != nil || result.Calls != 3 {
				t.Fatalf("calls=%d err=%v", result.Calls, err)
			}
			for i, messages := range client.messages {
				retained := strings.Contains(messages[1].Content, "Corrected the boundary")
				if retained != (delta <= 0) {
					t.Fatalf("phase %d optional history retention=%v", i, retained)
				}
				if client.options[i].ContextTokens != contextinput.Context64K {
					t.Fatal("not fixed 64K")
				}
			}
			if client.messages[0][0] != client.messages[2][0] || client.messages[0][1] != client.messages[2][1] {
				t.Fatal("Grouping/Text shared history differs")
			}
		})
	}
}

func TestHarness004CategoryHasNoUnconfirmedBreaking(t *testing.T) {
	for _, breaking := range []bool{false, true} {
		t.Run(map[bool]string{false: "nonbreaking", true: "breaking"}[breaking], func(t *testing.T) {
			responses := candidateResponses()
			client := &candidateClient{responses: responses}
			generator := ThreePhaseGenerator{Client: client}
			if breaking {
				generator.Evidence = []BreakingEvidence{{ID: "observed-api", FileIDs: []string{"F001"}, Fact: "public signature removed"}}
				client.responses[1] = `{"G001":{"type":"fix","breaking_evidence_ref":"observed-api"}}`
			}
			result, err := generator.Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
			if err != nil {
				t.Fatal(err)
			}
			for _, phase := range []int{1, 2} {
				visible := strings.Join([]string{client.messages[phase][0].Content, client.messages[phase][1].Content, client.messages[phase][2].Content}, "\n")
				if phase == 1 && breaking && !strings.Contains(visible, `"fact":"public signature removed"`) {
					t.Fatal("Category lost the meaning of the breaking evidence")
				}
				if phase == 2 && (strings.Contains(visible, "public signature removed") || strings.Contains(visible, "breaking_evidence_candidates")) {
					t.Fatal("Text received evidence instead of finalized values")
				}
				_, body, ok := strings.Cut(client.messages[phase][2].Content, "\n\nPhase input:\n")
				if !ok {
					t.Fatal("missing phase input")
				}
				var payload struct {
					Groups   []map[string]json.RawMessage `json:"groups"`
					Evidence []metadataBreakingEvidence   `json:"breaking_evidence_candidates"`
				}
				if err := json.Unmarshal([]byte(body), &payload); err != nil || len(payload.Groups) != 1 {
					t.Fatalf("invalid phase input: %v", err)
				}
				// one.go (host F001) sorts after asset.bin and is displayed as F2.
				if phase == 1 && breaking && (len(payload.Evidence) != 1 || payload.Evidence[0].ID != "observed-api" || payload.Evidence[0].Fact != "public signature removed" || len(payload.Evidence[0].FileIDs) != 1 || payload.Evidence[0].FileIDs[0] != "F2") {
					t.Fatal("Category evidence ID, file binding or fact changed")
				}
				value, exists := payload.Groups[0]["breaking"]
				if phase == 1 && exists {
					t.Fatalf("Category received unconfirmed breaking: %s", value)
				}
				if phase == 2 && (!exists || string(value) != map[bool]string{false: "false", true: "true"}[breaking] || string(payload.Groups[0]["type"]) != `"fix"`) {
					t.Fatal("Text did not receive finalized type/breaking")
				}
			}
			if result.Plan.Commits[0].Breaking != breaking {
				t.Fatal("final plan breaking changed")
			}
		})
	}
}
