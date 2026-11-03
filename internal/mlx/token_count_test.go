package mlx

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestTokenCountsShareCycleWithoutAdvancingOrGenerating(t *testing.T) {
	helper := shellHelper(t, `for profile in bounded-grouping bounded-category bounded-text; do
 for attempt in 1 2 3; do
  IFS= read -r request || exit 1
  case "$request" in *'"generation_profile":"'"$profile"'"'*) ;; *) exit 2 ;; esac
  case "$request" in *'"context_tokens":65536'*) ;; *) exit 3 ;; esac
  case "$request" in *'"raw_diff":"a\nb"'*) exit 4 ;; esac
  if [ "$attempt" -lt 3 ]; then
   case "$request" in *'"count_only":true'*) ;; *) exit 5 ;; esac
   printf '{"ok":true,"stop_reason":"completed","generation_profile":"%s","input_tokens":12001}\n' "$profile"
  else
   case "$request" in *'"count_only":true'*) exit 6 ;; esac
   printf '{"ok":true,"stop_reason":"completed","generation_profile":"%s","generated_json":"{}"}\n' "$profile"
  fi
 done
done`)
	backend := &Backend{Client: NewClient(helper), Model: "test", ModelRevision: "test"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cycle, closeCycle, err := backend.BeginCycle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer closeCycle()
	counter := cycle.(llm.InputTokenCounter)
	for _, profile := range []string{"bounded-grouping", "bounded-category", "bounded-text"} {
		opts := llm.Options{ContextTokens: 65536, OutputTokens: 4096, GenerationProfile: profile, InputLayout: "harness004", LiteralNewlines: true}
		messages := []llm.Message{{Role: "system", Content: "rules"}, {Role: "user", Content: `{"raw_diff":"a\nb"}`}, {Role: "user", Content: "instructions\n\nPhase input:\n{}"}}
		for i := 0; i < 2; i++ {
			count, err := counter.CountInputTokens(ctx, messages, json.RawMessage(`{}`), opts)
			if err != nil || count != 12001 {
				t.Fatalf("count=%d err=%v", count, err)
			}
		}
		response, err := cycle.ChatWithOptions(ctx, messages, json.RawMessage(`{}`), opts)
		if err != nil || response.Content != "{}" {
			t.Fatalf("generation=%+v err=%v", response, err)
		}
		if messages[1].Content != `{"raw_diff":"a\nb"}` {
			t.Fatal("host input mutated")
		}
	}
}

func TestCountFailureIsNeverTreatedAsZeroTokens(t *testing.T) {
	for _, body := range []string{
		`{"ok":false,"stop_reason":"internal_error","generation_profile":"bounded-grouping"}`,
		`{"ok":true,"stop_reason":"completed","generation_profile":"bounded-grouping","generated_json":"{}"}`,
		`{"ok":true,"stop_reason":"completed","generation_profile":"bounded-grouping","input_tokens":-1}`,
	} {
		helper := shellHelper(t, "IFS= read -r request\nprintf '%s\\n' '"+body+"'")
		backend := &Backend{Client: NewClient(helper)}
		_, err := backend.CountInputTokens(context.Background(), []llm.Message{{Role: "user", Content: "{}"}}, json.RawMessage(`{}`), llm.Options{GenerationProfile: "bounded-grouping", InputLayout: "harness004"})
		var failure *Error
		if !errors.As(err, &failure) {
			t.Fatalf("missing explicit failure: %v", err)
		}
	}
}
