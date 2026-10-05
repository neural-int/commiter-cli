package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"

	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
)

// measuredBackend is for the instrumented helper's experimental protocol only.
// The source helper's sampling/generation contract is unchanged. Stderr, prompt,
// native thought and response text are never copied into benchmark artifacts.
type measuredBackend struct{ Executable, Model, Revision, Path string }

func (b *measuredBackend) Chat(ctx context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return b.ChatWithOptions(ctx, m, s, llm.Options{})
}

type cappedOutput struct {
	bytes.Buffer
	Limit int
}

func (b *cappedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.Limit {
		return 0, errors.New("experimental helper output limit")
	}
	return b.Buffer.Write(p)
}
func (b *measuredBackend) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	data, err := json.Marshal(mlx.Request{Messages: m, Schema: s, ContextTokens: o.ContextTokens, OutputTokens: o.OutputTokens, Model: b.Model + "@" + b.Revision, ModelPath: b.Path, GenerationProfile: o.GenerationProfile})
	if err != nil || len(data) > mlx.DefaultMaxRequestBytes {
		return llm.Response{}, errors.New("experiment request limit")
	}
	command := exec.CommandContext(ctx, b.Executable)
	command.Stdin = bytes.NewReader(append(data, '\n'))
	stdout := &cappedOutput{Limit: mlx.DefaultMaxResponseBytes}
	command.Stdout = stdout
	if err = command.Run(); err != nil {
		return llm.Response{}, errors.New("experimental helper process failed")
	}
	var response struct {
		mlx.Response
		Input  *int `json:"benchmark_input_tokens"`
		Output *int `json:"benchmark_output_tokens"`
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&response); err != nil {
		return llm.Response{}, errors.New("experimental helper protocol failed")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return llm.Response{}, errors.New("experimental helper returned trailing response")
	}
	r := llm.Response{Backend: "mlx", Model: response.Model, Content: response.GeneratedJSON, StopReason: string(response.StopReason)}
	if response.Input != nil {
		r.PromptEvalCount = *response.Input
		r.Availability.PromptEvalCount = true
	}
	if response.Output != nil {
		r.EvalCount = *response.Output
		r.Availability.EvalCount = true
	}
	if response.GenerationProfile != o.GenerationProfile && response.OK {
		return r, errors.New("experimental helper profile mismatch")
	}
	if response.StopReason == mlx.StopReasonCompleted && (!response.OK || response.Input == nil || response.Output == nil) {
		return r, errors.New("measurement unavailable on completed helper response")
	}
	return r, nil
}
