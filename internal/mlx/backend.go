package mlx

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Backend adapts the private helper to the planning-facing LLM contract.
// ModelPath must point to an already installed local model.
type Backend struct {
	Client        *Client
	Model         string
	ModelPath     string
	session       *Session
	ModelRevision string
}

var _ llm.OptionsBackend = (*Backend)(nil)
var _ llm.CycleBackend = (*Backend)(nil)

// BeginCycle scopes process reuse to one planner invocation.
func (backend *Backend) BeginCycle(ctx context.Context) (llm.OptionsBackend, func() error, error) {
	session, err := backend.Client.BeginSession(ctx)
	if err != nil {
		return nil, nil, err
	}
	cycle := *backend
	cycle.session = session
	return &cycle, func() error { session.Close(); return nil }, nil
}

func (backend *Backend) Chat(ctx context.Context, messages []llm.Message, schema json.RawMessage) (llm.Response, error) {
	return backend.ChatWithOptions(ctx, messages, schema, llm.Options{})
}

func (backend *Backend) ChatWithOptions(ctx context.Context, messages []llm.Message, schema json.RawMessage, options llm.Options) (llm.Response, error) {
	if backend == nil || backend.Client == nil {
		return llm.Response{}, errors.New("MLX helper client is required")
	}
	model := backend.Model
	if options.GenerationProfile != "" {
		model += "@" + backend.ModelRevision
	}
	if options.LiteralNewlines {
		rendered := append([]llm.Message(nil), messages...)
		if options.HistoryInput != nil {
			if options.InputLayout != "harness004" || len(rendered) != 3 {
				return llm.Response{}, errors.New("invalid history input layout")
			}
			rendered[1].Content = "{}" // Shared data travels once, in history fragments.
			if options.HistoryInput.PhaseTemplate != nil {
				prefix, _, ok := strings.Cut(rendered[2].Content, "\n\nPhase input:\n")
				if !ok {
					return llm.Response{}, errors.New("missing phase input")
				}
				rendered[2].Content = prefix + "\n\nPhase input:\n{}"
			}
		}
		for i, m := range rendered {
			if m.Role == "user" {
				body, prefix := m.Content, ""
				if options.InputLayout == "harness004" && i == 2 {
					before, after, ok := strings.Cut(body, "\n\nPhase input:\n")
					if !ok {
						return llm.Response{}, errors.New("missing phase input")
					}
					prefix, body = before+"\n\nPhase input:\n", after
				}
				content, e := llm.LiteralNewlineDisplay([]byte(body))
				if e != nil {
					return llm.Response{}, e
				}
				rendered[i].Content = prefix + content
			}
		}
		messages = rendered
	}
	if options.HistoryInput != nil && options.HistoryInput.SharedTemplate != nil {
		if options.HistoryInput.PhaseTemplate == nil {
			return llm.Response{}, errors.New("invalid attributed history display")
		}
		history := *options.HistoryInput
		shared, err := history.SharedTemplate.LiteralNewlines()
		if err != nil {
			return llm.Response{}, err
		}
		phase, err := history.PhaseTemplate.LiteralNewlines()
		if err != nil {
			return llm.Response{}, err
		}
		history.SharedTemplate, history.PhaseTemplate = &shared, &phase
		options.HistoryInput = &history
	}
	generate := backend.Client.Generate
	if backend.session != nil {
		generate = backend.session.Generate
	}
	request := Request{
		HistoryInput: options.HistoryInput,
		Schema:       schema, Messages: messages, ContextTokens: options.ContextTokens,
		OutputTokens: options.OutputTokens, Model: model, ModelPath: backend.ModelPath,
		GenerationProfile: options.GenerationProfile, InputLayout: options.InputLayout,
	}
	// The existing IPC byte bound is a resource limit, not a token estimate.
	// If optional attributed history exceeds it, use the host-compiled whole-
	// record omission display. The helper still validates exact mandatory tokens.
	transportOmission := false
	if history := request.HistoryInput; history != nil && history.SharedTemplate != nil {
		limit := backend.Client.MaxRequestBytes
		if limit <= 0 {
			limit = DefaultMaxRequestBytes
		}
		if _, err := marshalRequest(request, limit); errors.Is(err, errMessageTooLarge) {
			request.Messages = append([]llm.Message(nil), request.Messages...)
			request.Messages[0].Content = history.SystemWithoutHistory
			request.Messages[1].Content = history.SharedTemplate.Render([]int{})
			prefix, _, ok := strings.Cut(request.Messages[2].Content, "\n\nPhase input:\n")
			if !ok {
				return llm.Response{}, errors.New("missing phase input")
			}
			request.Messages[2].Content = prefix + "\n\nPhase input:\n" + history.PhaseTemplate.Render([]int{})
			request.HistoryInput = nil
			transportOmission = true
		}
	}
	response, err := generate(ctx, request)
	if err != nil {
		if response.ErrorClass == "input_too_large" {
			return llm.Response{}, llm.ErrInputTooLarge
		}
		var failure *Error
		if errors.As(err, &failure) && failure.Kind == FailureStopState && response.StopReason != StopReasonCompleted {
			return llm.Response{Backend: "mlx", Model: response.Model, StopReason: string(response.StopReason)}, nil
		}
		return llm.Response{}, err
	}
	if options.GenerationProfile != "" && response.GenerationProfile != options.GenerationProfile {
		return llm.Response{}, &Error{Kind: FailureProtocol}
	}
	if (request.HistoryInput != nil) != (response.SelectedHistoryIndices != nil) {
		return llm.Response{}, &Error{Kind: FailureProtocol}
	}
	if transportOmission {
		response.SelectedHistoryIndices = []int{}
	}
	var cache struct {
		InputPreparationCalls int    `json:"inputPreparationCalls"`
		InputTokens           *int   `json:"inputTokens"`
		PrefixReuseStatus     string `json:"prefixReuseStatus"`
		CacheReusedTokens     int    `json:"cacheReusedTokens"`
	}
	_ = json.Unmarshal(response.BenchmarkMetrics, &cache)
	inputCount := 0
	if cache.InputTokens != nil {
		inputCount = *cache.InputTokens
	}
	return llm.Response{
		SelectedHistoryIndices: response.SelectedHistoryIndices,
		InputPreparationCalls:  cache.InputPreparationCalls,
		PromptEvalCount:        inputCount,
		Availability:           llm.TelemetryAvailability{PromptEvalCount: cache.InputTokens != nil},
		PrefixReuseStatus:      cache.PrefixReuseStatus, CacheReusedTokens: cache.CacheReusedTokens,
		Backend: "mlx", Model: response.Model, Content: response.GeneratedJSON,
		StopReason: string(response.StopReason),
	}, nil
}
