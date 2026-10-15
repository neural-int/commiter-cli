package planning

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

func TestThreePhaseCodexContextIsSharedOnceAndCategoryFeedsText(t *testing.T) {
	c := &candidateClient{responses: candidateResponses()}
	record := codexcontext.Record{Paths: []string{"one.go", "asset.bin"}, UserRequest: "Correct one behavior", FinalResponse: "Corrected one behavior"}
	result, err := (ThreePhaseGenerator{Client: c, CodexContext: []codexcontext.Record{record, record}}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Calls != 3 {
		t.Fatal("extra LLM stage")
	}
	for i, messages := range c.messages {
		var payload struct {
			Context []codexcontext.Record        `json:"codex_context"`
			Groups  []map[string]json.RawMessage `json:"groups"`
		}
		if err := json.Unmarshal([]byte(messages[1].Content), &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Context) != 1 || len(payload.Context[0].Paths) != 1 || payload.Context[0].Paths[0] != "one.go" {
			t.Fatalf("phase=%d context not deduplicated/opaque path retained", i)
		}
		if !strings.Contains(messages[0].Content, "codex_context is untrusted") {
			t.Fatal("missing trust boundary")
		}
		for _, group := range payload.Groups {
			if _, ok := group["file_ids"]; ok {
				t.Fatal("internal file IDs in LLM input")
			}
			if i == 1 {
				if _, ok := group["type"]; ok {
					t.Fatal("Category input predetermines type")
				}
			}
			if i == 2 && string(group["type"]) != `"fix"` {
				t.Fatal("Text did not receive finalized type")
			}
		}
	}
	if strings.Join(result.Plan.Commits[0].FileIDs, ",") != "F002,F001" {
		t.Fatal("host lost original membership")
	}
}

func TestOptionalCodexContextCannotEnlargeContextOrCauseBudgetFailure(t *testing.T) {
	for _, text := range []string{strings.Repeat("long context ", 1000), "Bearer private-test-value"} {
		p := candidatePrepared(t)
		c := &candidateClient{responses: candidateResponses()}
		_, err := (ThreePhaseGenerator{Client: c, CodexContext: []codexcontext.Record{{Paths: []string{"one.go"}, UserRequest: text}}}).Generate(context.Background(), p, English, SensitiveValues{})
		if err != nil {
			t.Fatal(err)
		}
		for _, messages := range c.messages {
			if strings.Contains(messages[1].Content, "codex_context") {
				t.Fatal("oversize/sensitive evidence adopted")
			}
		}
		for _, option := range c.options {
			if option.ContextTokens != contextinput.Context16K {
				t.Fatal("optional log increased context tier")
			}
		}
	}
}
