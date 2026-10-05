package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"time"
)

type baselineCapture struct {
	backend llm.OptionsBackend
	calls   []metric
	groups  [][]string
}

func (c *baselineCapture) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected legacy call")
}
func (c *baselineCapture) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	if o.GenerationProfile == "bounded-category" {
		var p struct {
			Groups []struct {
				FileIDs []string `json:"file_ids"`
			} `json:"groups"`
		}
		if err := json.Unmarshal([]byte(m[1].Content), &p); err != nil {
			return llm.Response{}, err
		}
		for _, g := range p.Groups {
			c.groups = append(c.groups, g.FileIDs)
		}
		return llm.Response{}, errors.New("grouping probe stops before metadata")
	}
	start := time.Now()
	r, e := c.backend.ChatWithOptions(ctx, m, s, o)
	v := metric{Phase: o.GenerationProfile, Wall: time.Since(start).Seconds(), Stop: r.StopReason}
	if r.Availability.PromptEvalCount {
		x := r.PromptEvalCount
		v.Input = &x
	}
	if r.Availability.EvalCount {
		x := r.EvalCount
		v.Output = &x
	}
	c.calls = append(c.calls, v)
	return r, e
}
func baseline(f fixture, b llm.OptionsBackend) observation {
	start := time.Now()
	o := observation{Fixture: f.Name, Architecture: "current-stage1", Files: len(f.Files), Unresolved: true}
	d := contextinput.Document{SchemaVersion: 1, Files: f.Files}
	p, e := planning.Renderer(planning.English)(d)
	if e != nil {
		o.Reason = "render_failure"
		return o
	}
	c := &baselineCapture{backend: b}
	_, e = (planning.ThreePhaseGenerator{Client: c}).Generate(context.Background(), contextinput.Prepared{Document: d, Prompt: p, Budget: contextinput.Budget{ContextTokens: 16384}}, planning.English, planning.SensitiveValues{})
	o.Calls = c.calls
	o.Wall = time.Since(start).Seconds()
	if len(c.groups) == 0 {
		o.Reason = "baseline_stopped"
		return o
	}
	o.Groups = c.groups
	o.Complete = true
	o.Unresolved = false
	x, fm, fs := quality(o.Groups, f.Expected)
	o.Exact = &x
	o.FM = &fm
	o.FS = &fs
	return o
}
