package main

import (
	"context"
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Routing changes only global membership. Metadata retains pinned Gemma and
// the unchanged production profiles. There is no implicit model fallback.
type groupingBackend struct {
	Group, Base llm.OptionsBackend
	Profile     string
	Output      int
}

func (b *groupingBackend) Chat(ctx context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return b.ChatWithOptions(ctx, m, s, llm.Options{})
}
func (b *groupingBackend) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	if o.GenerationProfile == "bounded-grouping" {
		o.GenerationProfile = b.Profile
		o.OutputTokens = b.Output
		return b.Group.ChatWithOptions(ctx, m, s, o)
	}
	return b.Base.ChatWithOptions(ctx, m, s, o)
}
