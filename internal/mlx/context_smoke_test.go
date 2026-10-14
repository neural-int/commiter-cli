package mlx

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Opt-in capacity smoke: prepared synthetic fixtures and an already installed
// model only. No download, retry, metadata quality scoring, or Git mutation.
func TestInstalledHelperExpandedContext(t *testing.T) {
	helper, fixtures := os.Getenv("COMMITER_CONTEXT_SMOKE_HELPER"), os.Getenv("COMMITER_CONTEXT_SMOKE_FIXTURES")
	if helper == "" || fixtures == "" {
		t.Skip("requires an installed helper and frozen synthetic fixtures")
	}
	for _, tier := range []string{"32k", "64k"} {
		t.Run(tier, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(fixtures, "request-"+tier+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var request Request
			if err := json.Unmarshal(data, &request); err != nil {
				t.Fatal(err)
			}
			expected := 32768
			if tier == "64k" {
				expected = 65536
			}
			if request.ContextTokens != expected || request.OutputTokens != 768 || request.GenerationProfile != "bounded-text" {
				t.Fatal("unexpected smoke request contract")
			}
			backend := &Backend{Client: NewClient(helper), Model: "mlx-community/gemma-4-E4B-it-4bit", ModelRevision: "475b9088d29754a3379866cf5aeb6b41acd313c2", ModelPath: request.ModelPath}
			// This isolates helper capacity; the product's shared 120s deadline remains
			// unchanged and must not be inferred from this individual-call smoke.
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer cancel()
			started := time.Now()
			response, err := backend.ChatWithOptions(ctx, request.Messages, request.Schema, llm.Options{ContextTokens: request.ContextTokens, OutputTokens: request.OutputTokens, GenerationProfile: request.GenerationProfile})
			if err != nil || response.StopReason != "completed" {
				t.Fatalf("helper context=%d stop=%s error=%v", request.ContextTokens, response.StopReason, err)
			}
			var output map[string]map[string]string
			if err := json.Unmarshal([]byte(response.Content), &output); err != nil {
				t.Fatal("helper returned invalid JSON")
			}
			if len(output) != 1 || len(output["G001"]) != 2 || output["G001"]["scope"] != "smoke" || output["G001"]["summary"] != "check context" {
				t.Fatal("helper output did not match the fixed smoke schema")
			}
			t.Logf("context=%d profile=%s wall=%s completed", request.ContextTokens, request.GenerationProfile, time.Since(started).Round(time.Millisecond))
		})
	}
}
