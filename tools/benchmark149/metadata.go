package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type phaseTemplate struct {
	System  string
	Schema  map[string]any
	Payload map[string]any
}
type templateClient struct{ Templates map[string]phaseTemplate }

func (c *templateClient) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected legacy call")
}
func (c *templateClient) ChatWithOptions(_ context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	var shape, payload map[string]any
	if err := json.Unmarshal(s, &shape); err != nil {
		return llm.Response{}, err
	}
	if err := json.Unmarshal([]byte(m[1].Content), &payload); err != nil {
		return llm.Response{}, err
	}
	c.Templates[o.GenerationProfile] = phaseTemplate{System: strings.TrimSuffix(m[0].Content, string(s)), Schema: shape, Payload: payload}
	content := `{"G001":{"scope":"settings","summary":"設定の境界を修正"}}`
	switch o.GenerationProfile {
	case "bounded-grouping":
		files := payload["files"].([]any)
		id := files[0].(map[string]any)["id"].(string)
		encoded, _ := json.Marshal(map[string]string{id: "G001"})
		content = string(encoded)
	case "bounded-category":
		content = `{"G001":{"type":"fix","breaking_evidence_ref":"none"}}`
	}
	return llm.Response{Content: content, StopReason: "completed"}, nil
}
func metadataTemplates() (map[string]phaseTemplate, error) {
	prototype := fixtures()[0].Files[0]
	prototype.RawDiff = ""
	d := contextinput.Document{SchemaVersion: 1, Files: []contextinput.File{prototype}}
	prompt, err := planning.Renderer(planning.Japanese)(d)
	if err != nil {
		return nil, err
	}
	c := &templateClient{Templates: map[string]phaseTemplate{}}
	_, err = (planning.ThreePhaseGenerator{Client: c}).Generate(context.Background(), contextinput.Prepared{Document: d, Prompt: prompt, Budget: contextinput.Budget{ContextTokens: 16384}}, planning.Japanese, planning.SensitiveValues{})
	return c.Templates, err
}
func object(properties map[string]any, ids []string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": ids, "additionalProperties": false}
}

type finalizedGroup struct {
	ID      string              `json:"id"`
	FileIDs []string            `json:"file_ids"`
	Files   []contextinput.File `json:"files"`
}
type category struct {
	Type      string `json:"type"`
	Reference string `json:"breaking_evidence_ref"`
}
type commitText struct {
	Scope   string `json:"scope"`
	Summary string `json:"summary"`
}

// All grouping is immutable before metadata. Metadata batches do not rerun
// membership and cannot merge/split the global partition. Stage 1 alone chooses
// membership. Metadata's complete-group packets are independently 16K-bounded.
func finalMetadata(ctx context.Context, f fixture, membership [][]string, backend llm.OptionsBackend) (planning.Plan, []metric, string) {
	if !completeFixture(f, membership) {
		return planning.Plan{}, nil, "global_grouping_not_final"
	}
	templates, err := metadataTemplates()
	if err != nil {
		return planning.Plan{}, nil, "metadata_template_contract"
	}
	files := append([]contextinput.File(nil), f.Files...)
	sort.Slice(files, func(i, j int) bool {
		pi, pj := *files[i].NewPath, *files[j].NewPath
		ti, tj := strings.HasSuffix(pi, "_test.go"), strings.HasSuffix(pj, "_test.go")
		if ti != tj {
			return !ti
		}
		return pi < pj
	})
	groups := []finalizedGroup{}
	indices := map[int]int{}
	for _, file := range files {
		for membership, ids := range membership {
			if !contains(ids, file.ID) {
				continue
			}
			index, exists := indices[membership]
			if !exists {
				index = len(groups)
				indices[membership] = index
				groups = append(groups, finalizedGroup{ID: groupID(index)})
			}
			groups[index].FileIDs = append(groups[index].FileIDs, file.ID)
			file.ID = "path:" + *file.NewPath
			groups[index].Files = append(groups[index].Files, file)
			break
		}
	}
	recorder := &metadataRecorder{Backend: backend}
	categories := map[string]category{}
	texts := map[string]commitText{}
	// No summary is requested until every category packet has succeeded.
	for _, profile := range []string{"bounded-category", "bounded-text"} {
		template := templates[profile]
		prototype := template.Schema["properties"].(map[string]any)["G001"].(map[string]any)
		for offset := 0; offset < len(groups); offset += 4 {
			batch := groups[offset:min(offset+4, len(groups))]
			properties := map[string]any{}
			names := []string{}

			for _, g := range batch {
				names = append(names, g.ID)

				shape := prototype
				if profile == "bounded-category" {
					encoded, _ := json.Marshal(prototype)
					shape = map[string]any{}
					_ = json.Unmarshal(encoded, &shape)
					testOnly := true
					for _, file := range g.Files {
						testOnly = testOnly && strings.HasSuffix(*file.NewPath, "_test.go")
					}
					if testOnly {
						shape["properties"].(map[string]any)["type"] = map[string]any{"type": "string", "enum": []string{"test"}}
					}
				}
				properties[g.ID] = shape
			}
			shape, _ := json.Marshal(object(properties, names))
			payload := map[string]any{}
			for k, v := range template.Payload {
				payload[k] = v
			}
			payload["groups"] = batch
			encoded, _ := json.Marshal(payload)
			system := template.System + string(shape)
			budget := 768
			if profile == "bounded-category" {
				budget = 512
			}
			if len(system)+len(encoded)+contextinput.TemplateReserve+budget > 16384 {
				return planning.Plan{}, recorder.Calls, "context_overflow"
			}
			r, err := recorder.ChatWithOptions(ctx, []llm.Message{{Role: "system", Content: system}, {Role: "user", Content: string(encoded)}}, shape, llm.Options{ContextTokens: 16384, OutputTokens: budget, GenerationProfile: profile})

			if err != nil || ctx.Err() != nil {
				return planning.Plan{}, recorder.Calls, "metadata_backend_failure"
			}
			if r.StopReason != "completed" {
				return planning.Plan{}, recorder.Calls, "incomplete_metadata_output"
			}
			if profile == "bounded-category" {
				var out map[string]category
				if strictCandidateJSON([]byte(r.Content), &out) != nil || len(out) != len(batch) {
					return planning.Plan{}, recorder.Calls, "invalid_metadata_schema"
				}
				for _, g := range batch {
					c, ok := out[g.ID]
					if !ok {
						return planning.Plan{}, recorder.Calls, "invalid_metadata_group_id"
					}
					allowed := properties[g.ID].(map[string]any)["properties"].(map[string]any)["type"].(map[string]any)["enum"]
					raw, _ := json.Marshal(allowed)
					var types []string
					_ = json.Unmarshal(raw, &types)
					if !contains(types, c.Type) || c.Reference != "none" {
						if c.Reference == "unresolved" {
							return planning.Plan{}, recorder.Calls, "unresolved_breaking_evidence"
						}
						return planning.Plan{}, recorder.Calls, "invalid_metadata_value"
					}
					categories[g.ID] = c
				}
			} else {
				var out map[string]commitText
				if strictCandidateJSON([]byte(r.Content), &out) != nil || len(out) != len(batch) {
					return planning.Plan{}, recorder.Calls, "invalid_metadata_schema"
				}
				for _, g := range batch {
					t, ok := out[g.ID]
					if !ok || utf8.RuneCountInString(t.Scope) > 16 || utf8.RuneCountInString(t.Summary) > 48 {
						return planning.Plan{}, recorder.Calls, "invalid_metadata_value"
					}
					texts[g.ID] = t
				}
			}
		}
	}
	plan := planning.Plan{SchemaVersion: 1}
	for _, g := range groups {
		c, t := categories[g.ID], texts[g.ID]
		plan.Commits = append(plan.Commits, planning.Commit{Type: c.Type, Scope: t.Scope, Summary: t.Summary, FileIDs: g.FileIDs})
	}
	data, _ := json.Marshal(plan)
	validated, violations := planning.Validate(data, fixtureIDs(f), planning.SensitiveValues{}, planning.Japanese)
	if len(violations) > 0 {
		return planning.Plan{}, recorder.Calls, "invalid_final_plan"
	}
	return validated, recorder.Calls, ""
}

func groupID(i int) string { return fmt.Sprintf("G%03d", i+1) }

func contains(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}
func fixtureIDs(f fixture) []string {
	ids := []string{}
	for _, file := range f.Files {
		ids = append(ids, file.ID)
	}
	return ids
}
func completeFixture(f fixture, groups [][]string) bool {
	ids := fixtureIDs(f)
	seen := map[string]bool{}
	for _, g := range groups {
		if len(g) == 0 {
			return false
		}
		for _, id := range g {
			if !contains(ids, id) || seen[id] {
				return false
			}
			seen[id] = true
		}
	}
	return len(seen) == len(ids)
}
