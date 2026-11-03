package planning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestOptionalFinalResponseCountFailureStopsWithoutSilentOmission(t *testing.T) {
	client := &candidateClient{responses: candidateResponses(), countInput: func([]llm.Message, llm.Options) (int, error) {
		return 0, errors.New("count failed")
	}}
	_, err := (ThreePhaseGenerator{Client: client, CodexContext: []codexcontext.Record{{Paths: []string{"one.go"}, FinalResponse: "Completed work"}}}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
	if err == nil || client.calls != 1 {
		t.Fatal("failed counting was treated as fitting/omittable history")
	}
}

func TestThreePhaseFinalOnlyHistoryHasNoRequestField(t *testing.T) {
	client := &candidateClient{responses: candidateResponses()}
	record := codexcontext.Record{Paths: []string{"one.go"}, FinalResponse: "Corrected the boundary"}
	result, err := (ThreePhaseGenerator{Client: client, CodexContext: []codexcontext.Record{record, record}}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
	if err != nil || result.Calls != 3 {
		t.Fatalf("final-only cycle failed: %v", err)
	}
	for _, messages := range client.messages {
		var payload struct {
			Records []map[string]json.RawMessage `json:"codex_context"`
		}
		if err := json.Unmarshal([]byte(messages[1].Content), &payload); err != nil || len(payload.Records) != 1 {
			t.Fatal("final-only history was lost or duplicated")
		}
		if _, exists := payload.Records[0]["user_request"]; exists {
			t.Fatal("request field remains in model input")
		}
		if string(payload.Records[0]["final_response"]) != `"Corrected the boundary"` || strings.Contains(messages[0].Content, "past user requests") {
			t.Fatal("final-only content or instruction mismatch")
		}
	}
	if client.messages[0][0] != client.messages[2][0] || client.messages[0][1] != client.messages[2][1] {
		t.Fatal("shared prefix changed")
	}
}

func TestHarness004DisplayOmitsNonemptyUserRequests(t *testing.T) {
	source := map[string]any{"agent_history_contexts": []any{map[string]any{"context_ref": "H1", "user_requests": []any{"past request"}, "final_responses": []any{"completed work"}}}}
	display := display004(source).(map[string]any)
	history := display["agent_history_contexts"].([]any)[0].(map[string]any)
	if _, exists := history["user_requests"]; exists {
		t.Fatal("nonempty user requests remain in the 004 display")
	}
	if _, exists := source["agent_history_contexts"].([]any)[0].(map[string]any)["user_requests"]; !exists || history["final_responses"].([]any)[0] != "completed work" {
		t.Fatal("host input or final response changed")
	}
}
