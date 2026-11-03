package mlx

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestThreePhasesSendExactlyThreeRequestsIncludingHistoryAdmission(t *testing.T) {
	captured := filepath.Join(t.TempDir(), "requests.jsonl")
	helper := shellHelper(t, `for profile in bounded-grouping bounded-category bounded-text; do
 IFS= read -r request || exit 1
 case "$request" in *'"generation_profile":"'"$profile"'"'*) ;; *) exit 2 ;; esac
 case "$request" in *'"count_only"'*) exit 3 ;; esac
 printf '%s\n' "$request" >> '`+captured+`'
 selection=''
 if [ "$profile" != bounded-text ]; then selection=',"selected_history_indices":[0]'; fi
 printf '{"ok":true,"stop_reason":"completed","generation_profile":"%s","generated_json":"{}"%s}\n' "$profile" "$selection"
 done`)
	backend := &Backend{Client: NewClient(helper), Model: "test", ModelRevision: "test"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cycle, closeCycle, err := backend.BeginCycle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer closeCycle()
	for _, profile := range []string{"bounded-grouping", "bounded-category", "bounded-text"} {
		opts := llm.Options{ContextTokens: 65536, OutputTokens: 4096, GenerationProfile: profile, InputLayout: "harness004", LiteralNewlines: true}
		messages := []llm.Message{{Role: "system", Content: "rules"}, {Role: "user", Content: `{"raw_diff":"a\nb"}`}, {Role: "user", Content: "instructions\n\nPhase input:\n{}"}}
		if profile != "bounded-text" {
			opts.HistoryInput = &llm.OptionalHistoryInput{After: "\"raw_diff\":\"a\nb\"", Records: []string{`{"final_response":"done","paths":["a.go"]}`}, SystemWithoutHistory: "rules"}
		}
		response, err := cycle.ChatWithOptions(ctx, messages, json.RawMessage(`{}`), opts)
		if err != nil || response.Content != "{}" {
			t.Fatalf("generation=%+v err=%v", response, err)
		}
		if messages[1].Content != `{"raw_diff":"a\nb"}` {
			t.Fatal("host input mutated")
		}
	}
	data, err := os.ReadFile(captured)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("IPC requests=%d, want exactly 3", len(lines))
	}
	for i, line := range lines {
		var request Request
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			t.Fatal(err)
		}
		if i < 2 {
			if request.Messages[1].Content != "{}" || request.HistoryInput == nil || request.HistoryInput.After != "\"raw_diff\":\"a\nb\"" {
				t.Fatal("shared diff was duplicated or display changed")
			}
		} else if request.Messages[1].Content != "{\"raw_diff\":\"a\nb\"}" {
			t.Fatal("normal display changed")
		}
	}
}

func TestRuntimeInputOverflowIsReturnedWithoutRetry(t *testing.T) {
	helper := shellHelper(t, `IFS= read -r request
 printf '%s\n' '{"ok":false,"stop_reason":"internal_error","error_class":"input_too_large","generation_profile":"bounded-grouping"}'`)
	backend := &Backend{Client: NewClient(helper)}
	_, err := backend.ChatWithOptions(context.Background(), []llm.Message{{Role: "user", Content: "{}"}}, json.RawMessage(`{}`), llm.Options{GenerationProfile: "bounded-grouping"})
	if !errors.Is(err, llm.ErrInputTooLarge) {
		t.Fatalf("overflow classification lost: %v", err)
	}
}
