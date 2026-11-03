package planning

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestHistoryFragmentsReproduceTheOriginalDisplayWithoutDuplicatingDiff(t *testing.T) {
	shared := map[string]any{
		"alpha": []any{},

		"codex_context": []any{map[string]any{"paths": []string{"one.go"}, "final_response": "修正\n完了\\n\"引用\""}},
		"files":         []any{map[string]any{"file_id": "F10", "raw_diff": "@@ -1 +1 @@\n-old\n+new\n"}},
	}
	messages := []llm.Message{{Role: "system", Content: "base" + codexTrustInstruction}, {Role: "user", Content: string(naturalJSON(shared))}}
	history, err := harness004OptionalHistory(messages)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := llm.LiteralNewlineDisplay(naturalJSON(shared))
	got := "{" + history.Before + ",\"codex_context\":[" + strings.Join(history.Records, ",") + "]," + history.After + "}"
	if got != want || history.SystemWithoutHistory != "base" {
		t.Fatal("history fragments changed the original display/system")
	}
	data, _ := json.Marshal(history)
	if strings.Count(string(data), "raw_diff") != 1 {
		t.Fatal("diff repeated in history descriptor")
	}
	if messages[1].Content != string(naturalJSON(shared)) {
		t.Fatal("host canonical input mutated")
	}
	for _, indices := range [][]int{nil, {-1}, {1}, {0, 0}} {
		if _, err := selected004History(messages, indices); err == nil {
			t.Fatalf("accepted invalid indices %v", indices)
		}
	}
}

func TestHelperSelectedHistoryIsFrozenForTextPrefix(t *testing.T) {
	client := &candidateClient{responses: candidateResponses(), countInput: func(messages []llm.Message, _ llm.Options) (int, error) {
		if strings.Contains(messages[1].Content, "too large") {
			return 61441, nil
		}
		return 1000, nil
	}}
	records := []codexcontext.Record{{Paths: []string{"one.go"}, FinalResponse: "too large"}, {Paths: []string{"one.go"}, FinalResponse: "fits"}}
	result, err := (ThreePhaseGenerator{Client: client, GroupingHistory: syntheticAttributedHistory(t, candidatePrepared(t), records)}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
	if err != nil || result.Calls != 3 {
		t.Fatalf("calls=%d err=%v", result.Calls, err)
	}
	for _, messages := range client.messages {
		if strings.Contains(messages[1].Content, "too large") || !strings.Contains(messages[1].Content, "fits") {
			t.Fatal("omitted history was reintroduced")
		}
	}
	if client.messages[0][0] != client.messages[2][0] || client.messages[0][1] != client.messages[2][1] {
		t.Fatal("Grouping/Text prefix differs after selection")
	}
}
