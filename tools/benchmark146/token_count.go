package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Oracle/template paths do not tokenize or generate. Real backends must expose
// actual counting; the old byte budget is never substituted for real tokens.
func countBackendInput(ctx context.Context, backend llm.OptionsBackend, m []llm.Message, s json.RawMessage, o llm.Options) (int, error) {
	if backend == nil {
		return 1000, nil
	}
	if counter, ok := backend.(llm.InputTokenCounter); ok {
		return counter.CountInputTokens(ctx, m, s, o)
	}
	return 0, errors.New("benchmark backend requires actual input token counting")
}

func (c *capture) CountInputTokens(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (int, error) {
	return countBackendInput(ctx, c.Backend, m, s, o)
}

func (c *recordingClient) CountInputTokens(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (int, error) {
	return countBackendInput(ctx, c.Backend, m, s, o)
}

func (c *templateClient) CountInputTokens(context.Context, []llm.Message, json.RawMessage, llm.Options) (int, error) {
	return 1000, nil // Synthetic schema-template capture, never measured inference.
}
