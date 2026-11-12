//go:build darwin || linux

package llamacpp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestTrialShrinksHistoryAfterRequestByteOverflow(t *testing.T) {
	for _, stage := range []string{"apply-template", "tokenize"} {
		t.Run(stage, func(t *testing.T) {
			largeRecord := strings.Repeat("x", MaxRequestBytes)
			if stage == "tokenize" {
				largeRecord = "wide"
			}
			selectedTokens := []int{19, 7, 11}
			baseTokenizations, selectedTokenizations, completions := 0, 0, 0
			_, transport := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) > MaxRequestBytes {
					t.Error("oversized request reached server")
					w.WriteHeader(500)
					return
				}
				switch r.URL.Path {
				case "/apply-template":
					var request struct {
						Messages []llm.Message `json:"messages"`
					}
					if err := json.Unmarshal(body, &request); err != nil || len(request.Messages) != 3 || !strings.Contains(request.Messages[1].Content, "MANDATORY_DIFF") {
						t.Error("mandatory input changed")
						w.WriteHeader(500)
						return
					}
					prompt := "base"
					if strings.Contains(request.Messages[1].Content, "small") {
						prompt = "selected"
					}
					if strings.Contains(request.Messages[1].Content, "wide") {
						// A valid template response whose escaped tokenize request
						// exceeds 1 MiB, despite the small apply-template request.
						prompt = strings.Repeat("\\", MaxRequestBytes/2)
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"prompt": prompt + "<think>\n\n</think>\n\n"})
				case "/tokenize":
					var request struct {
						Content string `json:"content"`
					}
					_ = json.Unmarshal(body, &request)
					tokens := []int{42}
					if strings.HasPrefix(request.Content, "selected") {
						selectedTokenizations++
						tokens = selectedTokens
					} else {
						baseTokenizations++
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"tokens": tokens})
				case "/completion":
					completions++
					var request struct {
						Prompt []int `json:"prompt"`
					}
					_ = json.Unmarshal(body, &request)
					if !reflect.DeepEqual(request.Prompt, selectedTokens) {
						t.Errorf("completion tokens=%v", request.Prompt)
					}
					_, _ = w.Write([]byte(`{"content":"{\"ok\":\"yes\"}","stop_type":"eos","truncated":false,"tokens_evaluated":3,"timings":{"prompt_n":3,"prompt_ms":1,"predicted_n":1,"predicted_ms":1,"cache_n":0}}`))
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					w.WriteHeader(500)
				}
			})
			largeJSON, _ := json.Marshal(largeRecord)
			shared := llm.DisplayTemplate{Parts: []llm.DisplayTemplate{
				{Literal: `{"diff":"MANDATORY_DIFF","history":[`},
				{Separator: ",", Parts: []llm.DisplayTemplate{{Require: []int{0}, Literal: string(largeJSON)}, {Require: []int{1}, Literal: `"small"`}}},
				{Literal: `]}`},
			}}
			phase := llm.DisplayTemplate{Literal: `{}`}
			messages := []llm.Message{{Role: "system", Content: "rules"}, {Role: "user"}, {Role: "user", Content: "Phase rules\n\nPhase input:\n{}"}}
			options := llm.Options{GenerationProfile: "bounded-grouping", InputLayout: "harness004", LiteralNewlines: true, ContextTokens: ContextTokens, OutputTokens: OutputTokens,
				HistoryInput: &llm.OptionalHistoryInput{RecordByteCounts: []int{len(largeRecord), 5}, SystemWithoutHistory: "rules", SharedTemplate: &shared, PhaseTemplate: &phase}}
			cycle := &trialCycle{process: &trialProcess{ctx: context.Background(), done: make(chan struct{}), transport: transport}}
			response, err := cycle.ChatWithOptions(context.Background(), messages, json.RawMessage(`{"type":"object","properties":{"ok":{"type":"string","enum":["yes"]}},"required":["ok"],"additionalProperties":false}`), options)
			if err != nil || !reflect.DeepEqual(response.SelectedHistoryIndices, []int{1}) || response.InputPreparationCalls > 5 || baseTokenizations != 1 || selectedTokenizations != 1 || completions != 1 {
				t.Fatalf("selected=%v preparations=%d base=%d selected=%d completions=%d err=%v", response.SelectedHistoryIndices, response.InputPreparationCalls, baseTokenizations, selectedTokenizations, completions, err)
			}
		})
	}
}
