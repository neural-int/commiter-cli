package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type recordingClient struct {
	Backend         llm.OptionsBackend
	IDs             []string
	Calls           []callMetric
	ValidatedGroups [][]string
}

func (c *recordingClient) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected legacy call")
}
func (c *recordingClient) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	if o.GenerationProfile == "bounded-category" {
		var payload struct {
			Groups []struct {
				FileIDs []string `json:"file_ids"`
			} `json:"groups"`
		}
		if json.Unmarshal([]byte(m[1].Content), &payload) == nil {
			c.ValidatedGroups = nil
			for _, g := range payload.Groups {
				c.ValidatedGroups = append(c.ValidatedGroups, g.FileIDs)
			}
		}
	}
	start := time.Now()
	r, err := c.Backend.ChatWithOptions(ctx, m, s, o)
	metric := callMetric{Window: c.IDs, Phase: o.GenerationProfile, WallMS: float64(time.Since(start)) / float64(time.Millisecond), OutputBytes: len(r.Content), OutputReservation: o.OutputTokens, Stop: r.StopReason}
	for _, message := range m {
		metric.InputBytes += len(message.Content)
	}
	if r.Availability.PromptEvalCount {
		metric.InputTokens = &r.PromptEvalCount
	}
	if r.Availability.EvalCount {
		metric.OutputTokens = &r.EvalCount
	}
	if err != nil {
		metric.Stop = "backend_error"
	}
	c.Calls = append(c.Calls, metric)
	return r, err
}
func runBaseline(parent context.Context, f fixture, mode string, backend llm.OptionsBackend) observation {
	start := time.Now()
	o := observation{Fixture: f.Name, Files: len(f.Files), Mode: mode, Strategy: "current-three-phase", TimeoutSeconds: planning.CandidateCycleTimeout.Seconds(), MaxWindows: 1, Calls: []callMetric{}}
	if len(f.Files) > 4 || backend == nil {
		o.Reason = "baseline_requires_real_backend_and_at_most_four_files"
		o.Unresolved = true
		return o
	}
	d := contextinput.Document{SchemaVersion: 1, Files: f.Files}
	prompt, err := planning.Renderer(planning.English)(d)
	if err != nil {
		o.Reason = "baseline_render"
		o.Unresolved = true
		return o
	}
	c := &recordingClient{Backend: backend, IDs: idsFor(f.Graph)}
	r, err := (planning.ThreePhaseGenerator{Client: c}).Generate(parent, contextinput.Prepared{Document: d, Prompt: prompt, Budget: contextinput.Budget{ContextTokens: 16384}}, planning.English, planning.SensitiveValues{})
	o.Calls = c.Calls
	o.BackendCalls = len(c.Calls)
	o.WallMS = float64(time.Since(start)) / float64(time.Millisecond)
	for _, call := range c.Calls {
		if call.Phase != "bounded-grouping" {
			o.MetadataExecuted = true
		}
	}
	if len(c.ValidatedGroups) > 0 {
		o.Groups = c.ValidatedGroups
		o.Complete = complete(idsFor(f.Graph), o.Groups)
		exact, fm, fs := evaluate(f.Expected, o.Groups)
		o.Exact = &exact
		o.FM = &fm
		o.FS = &fs
	}
	if err != nil {
		o.Unresolved = len(c.ValidatedGroups) == 0
		o.PlanFailure = "baseline_failed"
		for _, code := range []string{"invalid_json", "invalid_schema", "invalid_type", "invalid_scope", "invalid_summary", "invalid_assignment", "incomplete_output", "unresolved_breaking_evidence"} {
			if strings.Contains(err.Error(), code) {
				o.PlanFailure = code
				break
			}
		}
		return o
	}
	o.PlanSucceeded = true
	o.Groups = nil
	for _, commit := range r.Plan.Commits {
		o.Groups = append(o.Groups, commit.FileIDs)
	}
	o.Complete = complete(idsFor(f.Graph), o.Groups)
	exact, fm, fs := evaluate(f.Expected, o.Groups)
	o.Exact = &exact
	o.FM = &fm
	o.FS = &fs
	return o
}
