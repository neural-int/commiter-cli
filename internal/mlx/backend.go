package mlx

import (
	"context"
	"encoding/json"
	"errors"

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

// BeginCycle scopes process reuse to one planner invocation.
func (backend *Backend) BeginCycle(ctx context.Context) (llm.OptionsBackend, func(), error) {
	session, err := backend.Client.BeginSession(ctx)
	if err != nil {
		return nil, nil, err
	}
	cycle := *backend
	cycle.session = session
	return &cycle, session.Close, nil
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
	if options.GenerationProfile != "" {
		rendered := append([]llm.Message(nil), messages...)
		for i, m := range rendered {
			if m.Role == "user" {
				content, e := llm.LiteralNewlineDisplay([]byte(m.Content))
				if e != nil {
					return llm.Response{}, e
				}
				rendered[i].Content = content
			}
		}
		messages = rendered
	}
	generate := backend.Client.Generate
	if backend.session != nil {
		generate = backend.session.Generate
	}
	response, err := generate(ctx, Request{
		Schema: schema, Messages: messages, ContextTokens: options.ContextTokens,
		OutputTokens: options.OutputTokens, Model: model, ModelPath: backend.ModelPath,
		GenerationProfile: options.GenerationProfile,
	})
	if err != nil {
		var failure *Error
		if errors.As(err, &failure) && failure.Kind == FailureStopState && response.StopReason != StopReasonCompleted {
			return llm.Response{Backend: "mlx", Model: response.Model, StopReason: string(response.StopReason)}, nil
		}
		return llm.Response{}, err
	}
	if options.GenerationProfile != "" && response.GenerationProfile != options.GenerationProfile {
		return llm.Response{}, &Error{Kind: FailureProtocol}
	}
	return llm.Response{
		Backend: "mlx", Model: response.Model, Content: response.GeneratedJSON,
		StopReason: string(response.StopReason),
	}, nil
}
