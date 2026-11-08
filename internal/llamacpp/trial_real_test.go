//go:build darwin

package llamacpp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestInstalledTrialGrammarFailureAndCancellation(t *testing.T) {
	executable, model := os.Getenv("COMMITER_TEST_LLAMA_SERVER"), os.Getenv("COMMITER_TEST_GGUF_MODEL")
	if executable == "" || model == "" {
		t.Skip("explicit LLGuidance server and verified GGUF required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := verifyTrialModel(ctx, model); err != nil {
		t.Fatal(err)
	}
	t.Run("grammar_failure_stops_server", func(t *testing.T) {
		p, err := startTrialProcess(ctx, executable, model)
		if err != nil {
			t.Fatal(err)
		}
		defer p.close()
		var template struct {
			Prompt string `json:"prompt"`
		}
		if err := p.call(ctx, "/apply-template", map[string]any{"messages": []map[string]string{{"role": "user", "content": "Return only the JSON string ok."}}, "reasoning_effort": "none", "chat_template_kwargs": map[string]bool{"enable_thinking": false}}, &template); err != nil {
			t.Fatal(err)
		}
		var tokenized struct {
			Tokens []int `json:"tokens"`
		}
		if err := p.call(ctx, "/tokenize", map[string]any{"content": template.Prompt, "add_special": false, "parse_special": true}, &tokenized); err != nil {
			t.Fatal(err)
		}
		var response struct {
			Content string `json:"content"`
			Stop    string `json:"stop_type"`
		}
		// Deliberately invalid Lark, isolated from ordinary planning cycles. Do not
		// print generated content. Failure must stop the child, independently of log
		// classification or whether an unconstrained answer would be valid JSON.
		err = p.call(ctx, "/completion", map[string]any{"prompt": tokenized.Tokens, "grammar": "%llguidance {}\nstart: (", "n_predict": 16, "stream": false, "cache_prompt": false, "id_slot": 0, "seed": 144, "temperature": 0}, &response)
		accepted := err == nil
		failed := p.logs.grammarFailed()
		t.Logf("invalid_llguidance_grammar: http_accepted=%t engine_error_observed=%t returned_valid_json=%t", accepted, failed, json.Valid([]byte(response.Content)))
		if accepted || response.Content != "" || response.Stop != "" {
			t.Fatal("LLGuidance failure returned a successful completion")
		}
		select {
		case <-p.done:
		case <-time.After(3 * time.Second):
			t.Fatal("LLGuidance failure did not terminate the owned server")
		}
		exit, ok := p.waitErr.(*exec.ExitError)
		if !ok || exit.ExitCode() != 1 {
			t.Fatal("LLGuidance failure did not take the patched exit path")
		}
		directory := p.directory
		if err := p.close(); err == nil {
			t.Fatal("abnormal child exit was lost during cycle close")
		}
		if _, err := os.Lstat(directory); !os.IsNotExist(err) {
			t.Fatal("failed child left its private socket directory")
		}
		t.Log("patched_exit=1 completion_discarded=true process_reaped=true socket_removed=true")
	})
	t.Run("fresh_server_cancellation", func(t *testing.T) {
		child, stop := context.WithCancel(ctx)
		defer stop()
		p, err := startTrialProcess(child, executable, model)
		if err != nil {
			t.Fatal(err)
		}
		defer p.close()
		stop()
		if err := p.close(); err != nil {
			t.Fatalf("native cancellation cleanup failed: %v", err)
		}
		if _, err := os.Lstat(p.directory); !os.IsNotExist(err) {
			t.Fatal("native cancellation left its private socket directory")
		}
	})
}
