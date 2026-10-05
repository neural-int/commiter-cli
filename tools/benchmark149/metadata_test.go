package main

import (
	"context"
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"reflect"
	"testing"
)

type packet struct {
	M []llm.Message
	S json.RawMessage
	O llm.Options
}
type metadataOracle struct {
	f       fixture
	Packets []packet
}

func (c *metadataOracle) Chat(ctx context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return c.ChatWithOptions(ctx, m, s, llm.Options{})
}
func (c *metadataOracle) ChatWithOptions(_ context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	c.Packets = append(c.Packets, packet{m, s, o})
	answers := map[string]any{}
	if o.GenerationProfile == "bounded-grouping" {
		var p struct {
			Files []contextinput.File `json:"files"`
		}
		_ = json.Unmarshal([]byte(m[1].Content), &p)
		for _, file := range p.Files {
			for group, ids := range c.f.Expected {
				for _, id := range ids {
					for _, original := range c.f.Files {
						if original.ID == id && file.ID == "path:"+*original.NewPath {
							answers[file.ID] = groupID(group)
						}
					}
				}
			}
		}
	} else {
		var p struct {
			Groups []struct {
				ID string `json:"id"`
			} `json:"groups"`
		}
		_ = json.Unmarshal([]byte(m[1].Content), &p)
		for _, g := range p.Groups {
			if o.GenerationProfile == "bounded-category" {
				answers[g.ID] = category{"fix", "none"}
			} else {
				answers[g.ID] = commitText{"core", "設定の境界を修正"}
			}
		}
	}
	data, _ := json.Marshal(answers)
	return llm.Response{Content: string(data), StopReason: "completed"}, nil
}
func TestMetadataPacketsMatchCurrentJapaneseFourFileContract(t *testing.T) {
	f := contractFixtures()[0]
	current := &metadataOracle{f: f}
	d := contextinput.Document{SchemaVersion: 1, Files: f.Files}
	prompt, e := planning.Renderer(planning.Japanese)(d)
	if e != nil {
		t.Fatal(e)
	}
	_, e = (planning.ThreePhaseGenerator{Client: current}).Generate(context.Background(), contextinput.Prepared{Document: d, Prompt: prompt, Budget: contextinput.Budget{ContextTokens: 16384}}, planning.Japanese, planning.SensitiveValues{})
	if e != nil {
		t.Fatal(e)
	}
	candidate := &metadataOracle{f: f}
	_, _, reason := finalMetadata(context.Background(), f, f.Expected, candidate)
	if reason != "" {
		t.Fatal(reason)
	}
	if !reflect.DeepEqual(current.Packets[1:], candidate.Packets) {
		t.Fatal("metadata prompt/schema/profile/budget changed")
	}
}
func TestInvalidGlobalGroupingDoesNotInvokeMetadata(t *testing.T) {
	f := contractFixtures()[0]
	b := &metadataOracle{f: f}
	plan, calls, reason := finalMetadata(context.Background(), f, [][]string{{"F001", "F001"}}, b)
	if reason == "" || len(calls) != 0 || len(b.Packets) != 0 || len(plan.Commits) != 0 {
		t.Fatal("partial metadata accepted")
	}
}
