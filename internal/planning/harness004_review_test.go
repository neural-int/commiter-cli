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

func TestHarness004HistoryFitsFinalMessagesAtContextCeiling(t *testing.T) {
	for _, ceiling := range []int{contextinput.Context16K, contextinput.Context32K, contextinput.Context64K} {
		for _, delta := range []int{-1, 0, 1} {
			t.Run(fmt.Sprintf("%d/%+d", ceiling, delta), func(t *testing.T) {
				p := candidatePrepared(t)
				p.Budget.ContextTokens = ceiling
				record := codexcontext.Record{Paths: []string{"one.go"}, UserRequest: "Correct the observed behavior", FinalResponse: strings.Repeat("result ", 300)}
				baseline := &candidateClient{responses: candidateResponses()}
				if _, err := (ThreePhaseGenerator{Client: baseline, CodexContext: []codexcontext.Record{record}}).Generate(context.Background(), p, English, SensitiveValues{}); err != nil {
					t.Fatal(err)
				}
				// Keep mandatory evidence within budget, but put the full history just
				// immediately below, at, or above the final Grouping message limit.
				padding := ceiling - requiredHarness004Size(baseline.messages[0]) + delta
				if padding <= 0 {
					t.Fatal("invalid boundary fixture")
				}
				p.Document.Files[0].GroupingRawDiff += strings.Repeat("x", padding)
				p.Prompt, _ = Renderer(English)(p.Document)
				client := &candidateClient{responses: candidateResponses()}
				result, err := (ThreePhaseGenerator{Client: client, CodexContext: []codexcontext.Record{record}}).Generate(context.Background(), p, English, SensitiveValues{})
				if err != nil || result.Calls != 3 {
					t.Fatalf("optional history prevented the three-phase cycle: calls=%d err=%v", result.Calls, err)
				}
				var shared struct {
					History []codexcontext.Record `json:"codex_context"`
				}
				if err := json.Unmarshal([]byte(client.messages[0][1].Content), &shared); err != nil {
					t.Fatal(err)
				}
				wantFinal := record.FinalResponse
				if delta > 0 {
					wantFinal = ""
				}
				if len(shared.History) != 1 || shared.History[0].UserRequest != record.UserRequest || shared.History[0].FinalResponse != wantFinal {
					t.Fatal("history acceptance/final-response fallback differs at the boundary")
				}
				if client.messages[0][0] != client.messages[2][0] || client.messages[0][1] != client.messages[2][1] {
					t.Fatal("Grouping/Text shared history or prefix differs")
				}
				for i, messages := range client.messages {
					if requiredHarness004Size(messages) > client.options[i].ContextTokens || client.options[i].ContextTokens > ceiling {
						t.Fatalf("phase %d exceeded its final message budget", i)
					}
				}
			})
		}
	}
}

func TestHarness004OmitsHistoryWhenRequestAloneCannotFit(t *testing.T) {
	p := candidatePrepared(t)
	p.Budget.ContextTokens = contextinput.Context64K
	baseline := &candidateClient{responses: candidateResponses()}
	if _, err := (ThreePhaseGenerator{Client: baseline}).Generate(context.Background(), p, English, SensitiveValues{}); err != nil {
		t.Fatal(err)
	}
	maxSize := 0
	for _, messages := range baseline.messages {
		if size := requiredHarness004Size(messages); size > maxSize {
			maxSize = size
		}
	}
	p.Document.Files[0].GroupingRawDiff += strings.Repeat("x", contextinput.Context64K-maxSize-32)
	p.Prompt, _ = Renderer(English)(p.Document)
	client := &candidateClient{responses: candidateResponses()}
	record := codexcontext.Record{Paths: []string{"one.go"}, UserRequest: strings.Repeat("request ", 100), FinalResponse: "completed"}
	if _, err := (ThreePhaseGenerator{Client: client, CodexContext: []codexcontext.Record{record}}).Generate(context.Background(), p, English, SensitiveValues{}); err != nil {
		t.Fatal(err)
	}
	for _, messages := range client.messages {
		if strings.Contains(messages[1].Content, "codex_context") {
			t.Fatal("unfittable optional request was retained")
		}
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
				_, body, ok := strings.Cut(client.messages[phase][2].Content, "\n\nPhase input:\n")
				if !ok {
					t.Fatal("missing phase input")
				}
				var payload struct {
					Groups []map[string]json.RawMessage `json:"groups"`
				}
				if err := json.Unmarshal([]byte(body), &payload); err != nil || len(payload.Groups) != 1 {
					t.Fatalf("invalid phase input: %v", err)
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
