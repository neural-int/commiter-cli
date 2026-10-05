package main

import (
	"context"
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"time"
)

type metadataRecorder struct {
	Backend llm.OptionsBackend
	Calls   []metric
}

func (c *metadataRecorder) Chat(ctx context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return c.ChatWithOptions(ctx, m, s, llm.Options{})
}
func (c *metadataRecorder) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	start := time.Now()
	r, e := c.Backend.ChatWithOptions(callCtx, m, s, o)
	v := metric{Phase: o.GenerationProfile, Wall: time.Since(start).Seconds(), Stop: safeStop(r.StopReason)}
	if r.Availability.PromptEvalCount {
		x := r.PromptEvalCount
		v.Input = &x
	}
	if r.Availability.EvalCount {
		x := r.EvalCount
		v.Output = &x
	}
	c.Calls = append(c.Calls, v)
	return r, e
}
