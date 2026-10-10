package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
)

type callMetric struct {
	Profile     string   `json:"profile"`
	WallSeconds float64  `json:"wall_seconds"`
	Stop        string   `json:"stop"`
	Input       *int     `json:"input_tokens"`
	Output      *int     `json:"output_tokens"`
	Load        *float64 `json:"load_seconds"`
	TTFT        *float64 `json:"runtime_ttft_seconds"`
	Peak        *int64   `json:"mlx_peak_bytes"`
	RSS         *int64   `json:"helper_peak_rss_bytes"`
}

type measuredBackend struct {
	Helper, Model, Path string
	Calls               []callMetric
}

type cappedOutput struct {
	bytes.Buffer
	Limit int
}

func (b *cappedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.Limit {
		return 0, errors.New("measurement output limit")
	}
	return b.Buffer.Write(p)
}
func (b *measuredBackend) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("explicit profiles required")
}

var rssLine = regexp.MustCompile(`(?m)^\s*(\d+)\s+maximum resident set size\s*$`)

func (b *measuredBackend) ChatWithOptions(ctx context.Context, messages []llm.Message, schema json.RawMessage, options llm.Options) (llm.Response, error) {
	request, err := json.Marshal(mlx.Request{Messages: messages, Schema: schema, Model: b.Model, ModelPath: b.Path, GenerationProfile: options.GenerationProfile, ContextTokens: options.ContextTokens, OutputTokens: options.OutputTokens})
	if err != nil || len(request) > mlx.DefaultMaxRequestBytes {
		return llm.Response{}, errors.New("measurement request limit")
	}
	metric := callMetric{Profile: options.GenerationProfile, Stop: "process_failure"}
	started := time.Now()
	defer func() { metric.WallSeconds = time.Since(started).Seconds(); b.Calls = append(b.Calls, metric) }()
	cmd := exec.CommandContext(ctx, "/usr/bin/time", "-l", b.Helper)
	cmd.Stdin = bytes.NewReader(append(request, '\n'))
	out := &cappedOutput{Limit: mlx.DefaultMaxResponseBytes}
	diagnostic := &cappedOutput{Limit: 128 * 1024}
	cmd.Stdout = out
	cmd.Stderr = diagnostic
	processErr := cmd.Run()
	if match := rssLine.FindStringSubmatch(diagnostic.String()); len(match) == 2 {
		if rss, e := strconv.ParseInt(match[1], 10, 64); e == nil {
			metric.RSS = &rss
		}
	}
	// Helper stderr is never persisted, printed or passed back to a planner.
	if processErr != nil {
		return llm.Response{}, errors.New("measurement helper process failed")
	}
	var response struct {
		mlx.Response
		Input  *int     `json:"benchmark_input_tokens"`
		Output *int     `json:"benchmark_output_tokens"`
		Load   *float64 `json:"benchmark_load_seconds"`
		TTFT   *float64 `json:"benchmark_ttft_seconds"`
		Peak   *int64   `json:"benchmark_peak_bytes"`
	}
	decoder := json.NewDecoder(bytes.NewReader(out.Bytes()))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&response) != nil {
		return llm.Response{}, errors.New("measurement protocol failed")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return llm.Response{}, errors.New("measurement trailing response")
	}
	metric.Stop = string(response.StopReason)
	metric.Input = response.Input
	metric.Output = response.Output
	metric.Load = response.Load
	metric.TTFT = response.TTFT
	metric.Peak = response.Peak
	if response.GenerationProfile != options.GenerationProfile && response.OK {
		return llm.Response{}, errors.New("measurement profile mismatch")
	}
	r := llm.Response{Content: response.GeneratedJSON, StopReason: string(response.StopReason), Backend: "mlx", Model: response.Model}
	if response.Input != nil {
		r.PromptEvalCount = *response.Input
		r.Availability.PromptEvalCount = true
	}
	if response.Output != nil {
		r.EvalCount = *response.Output
		r.Availability.EvalCount = true
	}
	if response.StopReason == mlx.StopReasonCompleted && (!response.OK || response.Input == nil || response.Output == nil || response.Load == nil || response.TTFT == nil || response.Peak == nil || metric.RSS == nil) {
		return r, errors.New("measurement unavailable")
	}
	return r, nil
}
