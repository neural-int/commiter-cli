package llamacpp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Trial revisions and the approved local patch are fixed for the native trial.
// Their identity does not certify the remaining release qualification gates.
const TrialLlamaCommit = "80d2c0c2c56e4a1bbf74be98f64b99927bf230bc"
const TrialLLGuidanceCommit = "d795912fedc7d393de740177ea9ea761e7905774"
const TrialLlamaBuild = "b1-" + TrialLlamaCommit + "+commiter-llg-fail-closed-v1"
const TrialModelID = "unsloth/Qwen3.5-4B-GGUF"
const TrialModelRevision = "e87f176479d0855a907a41277aca2f8ee7a09523"
const TrialModelSHA256 = "00fe7986ff5f6b463e62455821146049db6f9313603938a70800d1fb69ef11a4"
const TrialModelBytes int64 = 2740937888

var errTrial = errors.New("llamacpp: experimental native cycle failed")

// TrialBackend is an explicit internal integration-test entry point. Normal CLI
// factories never construct it: release qualification remains incomplete.
type TrialBackend struct {
	Executable, ModelPath string
	Observe               func(TrialObservation)
}

// TrialObservation contains host-side diagnostic evidence, never model input.
type TrialObservation struct {
	PID                          int
	Directory                    string
	ModelLoads                   int
	MetalDevice                  bool
	OffloadedLayers, TotalLayers int
	ConstraintFailure            string
}

var _ llm.CycleBackend = (*TrialBackend)(nil)

func (b *TrialBackend) Chat(ctx context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return b.ChatWithOptions(ctx, m, s, llm.Options{})
}
func (*TrialBackend) ChatWithOptions(context.Context, []llm.Message, json.RawMessage, llm.Options) (llm.Response, error) {
	return llm.Response{}, errors.New("llamacpp: begin an explicit trial cycle before generation")
}

func (b *TrialBackend) BeginCycle(ctx context.Context) (llm.OptionsBackend, func() error, error) {
	if err := verifyTrialModel(ctx, b.ModelPath); err != nil {
		return nil, nil, err
	}
	process, err := startTrialProcess(ctx, b.Executable, b.ModelPath)
	if err != nil {
		return nil, nil, err
	}
	// In this revision n_ctx is exposed by /props, not /completion's
	// generation_settings. Verify the loaded slot, model and compiled revision.
	data, err := process.transport.request(ctx, http.MethodGet, "/props", nil)
	var props struct {
		Settings struct {
			Context int `json:"n_ctx"`
		} `json:"default_generation_settings"`
		Slots     int    `json:"total_slots"`
		ModelPath string `json:"model_path"`
		Build     string `json:"build_info"`
	}
	if err == nil {
		_, err = strictJSON(data)
	}
	if err == nil {
		err = json.Unmarshal(data, &props)
	}
	if err != nil || props.Settings.Context != ContextTokens || props.Slots != 1 || props.ModelPath != b.ModelPath || props.Build != TrialLlamaBuild {
		return nil, nil, errors.Join(errors.New("llamacpp: loaded model, context or build identity mismatch"), process.close())
	}
	if err := process.waitForStartupObservation(ctx); err != nil {
		return nil, nil, errors.Join(err, process.close())
	}
	cycle := &trialCycle{process: process}
	observe := func() {
		if b.Observe != nil {
			observation := process.logs.observation()
			observation.PID = process.command.Process.Pid
			observation.Directory = process.directory
			b.Observe(observation)
		}
	}
	observe()
	return cycle, func() error { err := process.close(); observe(); return err }, nil
}

func verifyTrialModel(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(path) {
		return errors.New("llamacpp: trial model path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != TrialModelBytes {
		return errors.New("llamacpp: trial requires the pinned Qwen3.5-4B Q4_K_M GGUF artifact")
	}
	f, err := os.Open(path)
	if err != nil {
		return errTrial
	}
	defer f.Close()
	h := sha256.New()
	buffer := make([]byte, 1<<20)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := f.Read(buffer)
		if n > 0 {
			_, _ = h.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return errTrial
		}
	}
	if hex.EncodeToString(h.Sum(nil)) != TrialModelSHA256 {
		return errors.New("llamacpp: trial GGUF digest does not match the fixed artifact")
	}
	return nil
}

type trialCycle struct {
	process *trialProcess
	mu      sync.Mutex
	phase   int
}

func (c *trialCycle) Chat(ctx context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return c.ChatWithOptions(ctx, m, s, llm.Options{})
}

func (c *trialCycle) ChatWithOptions(ctx context.Context, messages []llm.Message, rawSchema json.RawMessage, options llm.Options) (llm.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	profiles := []string{"bounded-grouping", "bounded-category", "bounded-text"}
	if c.phase >= len(profiles) || options.GenerationProfile != profiles[c.phase] || options.InputLayout != "harness004" || !options.LiteralNewlines || options.ContextTokens != ContextTokens || options.OutputTokens != OutputTokens {
		return llm.Response{}, errTrial
	}
	shape, err := compileSchema(rawSchema)
	if err != nil {
		return llm.Response{}, err
	}
	started := time.Now()
	counts := []int{}
	if h := options.HistoryInput; h != nil {
		counts = h.RecordByteCounts
		if counts == nil {
			counts = make([]int, len(h.Records))
			for i, record := range h.Records {
				counts[i] = len(record)
			}
		}
	}
	prepared, err := prepareInput(ctx, counts, func(ctx context.Context, indices []int) ([]int, error) {
		rendered, err := trialMessages(messages, options, indices)
		if err != nil {
			return nil, err
		}
		var template struct {
			Prompt string `json:"prompt"`
		}
		if err := c.process.call(ctx, "/apply-template", map[string]any{"messages": rendered, "chat_template_kwargs": map[string]bool{"enable_thinking": false}, "reasoning_effort": "none"}, &template); err != nil {
			return nil, err
		}
		if template.Prompt == "" || !strings.HasSuffix(template.Prompt, "<think>\n\n</think>\n\n") {
			return nil, errors.New("llamacpp: Qwen thinking-off template suffix was not observed")
		}
		var tokenized struct {
			Tokens []int `json:"tokens"`
		}
		if err := c.process.call(ctx, "/tokenize", map[string]any{"content": template.Prompt, "add_special": false, "parse_special": true}, &tokenized); err != nil {
			return nil, err
		}
		if len(tokenized.Tokens) == 0 {
			return nil, errTrial
		}
		return tokenized.Tokens, nil
	})
	if err != nil {
		return llm.Response{}, err
	}
	var native struct {
		Content   string `json:"content"`
		StopType  string `json:"stop_type"`
		Truncated *bool  `json:"truncated"`
		Evaluated int    `json:"tokens_evaluated"`
		Timings   struct {
			PromptN     int      `json:"prompt_n"`
			PromptMS    *float64 `json:"prompt_ms"`
			PredictedN  int      `json:"predicted_n"`
			PredictedMS *float64 `json:"predicted_ms"`
			CachedN     *int     `json:"cache_n"`
		} `json:"timings"`
	}
	// A %llguidance grammar is rejected by builds without LLGuidance; it cannot
	// silently become standard GBNF. The fixed patch stops the owned process if
	// LLGuidance fails, and the non-streaming response is then discarded.
	grammar := "%llguidance {}\nstart: %json " + string(rawSchema)
	request := map[string]any{"prompt": prepared.TokenIDs, "grammar": grammar, "n_predict": OutputTokens, "stream": false, "cache_prompt": false, "id_slot": 0, "seed": 144, "temperature": 0.6, "top_k": 20, "top_p": 0.95, "min_p": 0, "repeat_penalty": 1.0, "n_keep": -1, "return_tokens": true}
	if err := c.process.call(ctx, "/completion", request, &native); err != nil {
		return llm.Response{}, err
	}
	observed := native.Truncated != nil && native.Timings.PromptMS != nil && native.Timings.PredictedMS != nil && native.Timings.CachedN != nil
	if !observed {
		return llm.Response{}, errors.New("llamacpp: native completion omitted required measurements")
	}
	if native.StopType != "eos" || *native.Truncated || native.Evaluated != len(prepared.TokenIDs) || native.Timings.PromptN != len(prepared.TokenIDs) || *native.Timings.CachedN != 0 || native.Timings.PredictedN <= 0 || *native.Timings.PromptMS < 0 || *native.Timings.PredictedMS < 0 || c.process.logs.grammarFailed() {
		return llm.Response{}, fmt.Errorf("%w: eos=%t truncated=%t prepared=%d evaluated=%d prompt=%d cached=%d grammar_failure=%t", errTrial, native.StopType == "eos", *native.Truncated, len(prepared.TokenIDs), native.Evaluated, native.Timings.PromptN, *native.Timings.CachedN, c.process.logs.grammarFailed())
	}
	if err := shape.validate([]byte(native.Content)); err != nil {
		return llm.Response{}, err
	}
	c.phase++
	return llm.Response{Backend: "llamacpp", Model: TrialModelID + "@" + TrialModelRevision, Content: native.Content, StopReason: "completed", TotalDuration: int64(time.Since(started)), PromptEvalCount: len(prepared.TokenIDs), PromptEvalDuration: int64(*native.Timings.PromptMS * float64(time.Millisecond)), EvalCount: native.Timings.PredictedN, EvalDuration: int64(*native.Timings.PredictedMS * float64(time.Millisecond)), SelectedHistoryIndices: prepared.SelectedHistoryIndices, InputPreparationCalls: prepared.PreparationCalls, PrefixReuseStatus: "unsupported", Availability: llm.TelemetryAvailability{TotalDuration: true, PromptEvalCount: true, PromptEvalDuration: true, EvalCount: true, EvalDuration: true}}, nil
}

func trialMessages(messages []llm.Message, options llm.Options, indices []int) ([]llm.Message, error) {
	if len(messages) != 3 {
		return nil, errTrial
	}
	rendered := append([]llm.Message(nil), messages...)
	if history := options.HistoryInput; history != nil {
		if len(indices) == 0 {
			rendered[0].Content = history.SystemWithoutHistory
		}
		if history.SharedTemplate == nil || history.PhaseTemplate == nil {
			return nil, errTrial
		}
		rendered[1].Content = history.SharedTemplate.Render(indices)
		prefix, _, ok := strings.Cut(rendered[2].Content, "\n\nPhase input:\n")
		if !ok {
			return nil, errTrial
		}
		rendered[2].Content = prefix + "\n\nPhase input:\n" + history.PhaseTemplate.Render(indices)
	}
	for i, message := range rendered {
		if message.Role != "user" {
			continue
		}
		body, prefix := message.Content, ""
		if i == 2 {
			before, after, ok := strings.Cut(body, "\n\nPhase input:\n")
			if !ok {
				return nil, errTrial
			}
			prefix, body = before+"\n\nPhase input:\n", after
		}
		display, err := llm.LiteralNewlineDisplay([]byte(body))
		if err != nil {
			return nil, errTrial
		}
		rendered[i].Content = prefix + display
	}
	return rendered, nil
}
