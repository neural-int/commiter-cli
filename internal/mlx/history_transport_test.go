package mlx

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestOptionalAttributedHistoryCannotOverflowTransport(t *testing.T) {
	captured := filepath.Join(t.TempDir(), "request.json")
	t.Setenv("COMMITER_TRANSPORT_CAPTURE", captured)
	helper := shellHelper(t, `IFS= read -r request
printf '%s' "$request" > "$COMMITER_TRANSPORT_CAPTURE"
printf '%s\n' '{"ok":true,"stop_reason":"completed","generated_json":"{}","generation_profile":"bounded-grouping"}'`)
	shared := llm.DisplayTemplate{Parts: []llm.DisplayTemplate{
		{Literal: `{"agent_history_contexts":[`},
		{Literal: `{"context_ref":"H1","final_responses":["` + strings.Repeat("h", 20000) + `"]}`, Require: []int{0}},
		{Literal: `],"files":[]}`},
	}}
	phase := llm.DisplayTemplate{Literal: `{}`}
	history := &llm.OptionalHistoryInput{Records: []string{}, RecordByteCounts: []int{20000}, SharedTemplate: &shared, PhaseTemplate: &phase, SystemWithoutHistory: "rules"}
	backend := &Backend{Client: NewClient(helper)}
	backend.Client.MaxRequestBytes = 4096
	messages := []llm.Message{{Role: "system", Content: "rules"}, {Role: "user", Content: shared.Render([]int{0})}, {Role: "user", Content: "contract\n\nPhase input:\n{}"}}
	response, err := backend.ChatWithOptions(context.Background(), messages, json.RawMessage(`{}`), llm.Options{ContextTokens: 65536, OutputTokens: 4096, GenerationProfile: "bounded-grouping", InputLayout: "harness004", LiteralNewlines: true, HistoryInput: history})
	if err != nil {
		t.Fatalf("optional history stopped mandatory Git input: %v", err)
	}
	if response.SelectedHistoryIndices == nil || len(response.SelectedHistoryIndices) != 0 {
		t.Fatal("transport omission was not reflected to the planner")
	}
	wire, err := os.ReadFile(captured)
	if err != nil {
		t.Fatal(err)
	}
	var request Request
	if err := json.Unmarshal(wire, &request); err != nil {
		t.Fatal(err)
	}
	if len(wire) > 4096 || request.HistoryInput != nil || request.Messages[1].Content != `{"agent_history_contexts":[],"files":[]}` {
		t.Fatal("transport fallback lost mandatory display or kept optional data")
	}
	if history.SharedTemplate.Render([]int{0}) != messages[1].Content {
		t.Fatal("host history was mutated")
	}
}
