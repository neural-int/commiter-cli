package mlx

import (
	"context"
	"encoding/json"
	"errors"
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
