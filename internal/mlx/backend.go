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
	ModelRevision string
}

var _ llm.OptionsBackend = (*Backend)(nil)

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
	response, err := backend.Client.Generate(ctx, Request{
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
