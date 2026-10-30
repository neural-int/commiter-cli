package mlx

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestBackendPassesCompletedCandidateToPlanningContract(t *testing.T) {
	helper := shellHelper(t, `IFS= read -r _
printf '%s\n' '{"ok":true,"stop_reason":"completed","generated_json":"{\"commits\":[]}","model":"local-model","runtime":"mlx"}'
`)
	backend := &Backend{Client: NewClient(helper), Model: "local-model", ModelPath: "/local/model"}
	response, err := backend.ChatWithOptions(context.Background(), []llm.Message{{Role: "user", Content: "plan"}}, json.RawMessage(`{"type":"object"}`), llm.Options{ContextTokens: 8192, OutputTokens: 2048})
	if err != nil {
		t.Fatal(err)
	}
	if response.Backend != "mlx" || response.Model != "local-model" || response.StopReason != "completed" || response.Content != `{"commits":[]}` {
		t.Fatalf("response=%#v", response)
	}
}

func TestBackendExposesIncompleteStopReasonWithoutCandidate(t *testing.T) {
	helper := shellHelper(t, `IFS= read -r _
printf '%s\n' '{"ok":false,"stop_reason":"max_tokens","generated_json":"{\"commits\":[]}"}'
`)
	backend := &Backend{Client: NewClient(helper)}
	response, err := backend.Chat(context.Background(), []llm.Message{{Role: "user", Content: "plan"}}, json.RawMessage(`{"type":"object"}`))
	if err != nil || response.StopReason != string(StopReasonMaxTokens) || response.Content != "" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestBackendPreservesFailedCompletedResponseAsError(t *testing.T) {
	helper := shellHelper(t, `IFS= read -r _
printf '%s\n' '{"ok":false,"stop_reason":"completed","generated_json":"{\"commits\":[]}"}'
`)
	backend := &Backend{Client: NewClient(helper)}
	response, err := backend.Chat(context.Background(), []llm.Message{{Role: "user", Content: "plan"}}, json.RawMessage(`{"type":"object"}`))
	var failure *Error
	if !errors.As(err, &failure) || failure.Kind != FailureStopState || response != (llm.Response{}) {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestBackendRejectsHelperThatIgnoresGenerationProfile(t *testing.T) {
	for _, echo := range []string{"", "bounded-category", "bounded-grouping"} {
		t.Run(echo, func(t *testing.T) {
			helper := shellHelper(t, `IFS= read -r request
case "$request" in
 *'"model":"candidate@pinned"'*'"generation_profile":"bounded-grouping"'*) ;;
 *) exit 1 ;;
esac
printf '%s\n' '{"ok":true,"stop_reason":"completed","generated_json":"{}","generation_profile":"`+echo+`"}'
`)
			b := &Backend{Client: NewClient(helper), Model: "candidate", ModelRevision: "pinned", ModelPath: "/local/model"}
			response, err := b.ChatWithOptions(context.Background(), []llm.Message{{Role: "user", Content: `{"task":"plan"}`}}, json.RawMessage(`{"type":"object"}`), llm.Options{GenerationProfile: "bounded-grouping"})
			if echo == "bounded-grouping" {
				if err != nil || response.Content != "{}" {
					t.Fatalf("response=%#v error=%v", response, err)
				}
			} else if err == nil || response.Content != "" {
				t.Fatalf("ignored profile accepted: %#v %v", response, err)
			}
		})
	}
}

func TestBoundedBackendRendersOnlyModelBodyAndKeepsHostJSON(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "request.json")
	helper := shellHelper(t, "IFS= read -r request\nprintf '%s' \"$request\" > "+shellQuote(marker)+"\nprintf '%s\\n' '{\"ok\":true,\"stop_reason\":\"completed\",\"generated_json\":\"{}\",\"generation_profile\":\"bounded-grouping\"}'")
	original := `{"raw_diff":"@@ -1 +1 @@\n-old\n+new\n"}`
	messages := []llm.Message{{Role: "user", Content: original}}
	b := &Backend{Client: NewClient(helper), Model: "candidate", ModelRevision: "pinned"}
	_, err := b.ChatWithOptions(context.Background(), messages, json.RawMessage(`{"type":"object"}`), llm.Options{GenerationProfile: "bounded-grouping", LiteralNewlines: true})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatal("invalid outer IPC JSON")
	}
	var request Request
	if err = json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(request.Messages[0].Content, "@@ -1 +1 @@\n-old\n+new\n") || messages[0].Content != original {
		t.Fatal("model display or original host changed")
	}
}

func TestBoundedBackendPreservesPlainTextInput(t *testing.T) {
	helper := shellHelper(t, `IFS= read -r request
case "$request" in *'"content":"plain text"'*) ;; *) exit 1 ;; esac
printf '%s\n' '{"ok":true,"stop_reason":"completed","generated_json":"{}","generation_profile":"bounded-grouping"}'`)
	b := &Backend{Client: NewClient(helper), Model: "candidate", ModelRevision: "pinned"}
	_, err := b.ChatWithOptions(context.Background(), []llm.Message{{Role: "user", Content: "plain text"}}, json.RawMessage(`{"type":"object"}`), llm.Options{GenerationProfile: "bounded-grouping"})
	if err != nil {
		t.Fatal(err)
	}
}
