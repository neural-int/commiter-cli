package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	content := `{"G001":{"scope":"settings","summary":"update retry bounds"}}`
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
	prompt, err := planning.Renderer(planning.English)(d)
	if err != nil {
		return nil, err
	}
	c := &templateClient{Templates: map[string]phaseTemplate{}}
	_, err = (planning.ThreePhaseGenerator{Client: c}).Generate(context.Background(), contextinput.Prepared{Document: d, Prompt: prompt, Budget: contextinput.Budget{ContextTokens: 16384}}, planning.English, planning.SensitiveValues{})
	return c.Templates, err
}
func object(properties map[string]any, ids []string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": ids, "additionalProperties": false}
}
func strictDecode(data []byte, output any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			keys := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := k.(string)
				if !ok || keys[key] {
					return errors.New("duplicate property")
				}
				keys[key] = true
				if err = walk(); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err = walk(); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(output)
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
func finalMetadata(ctx context.Context, f fixture, p partition, backend llm.OptionsBackend, mode string) (planning.Plan, []callMetric, string) {
	if p.Reason != "" || len(p.Unknown) > 0 || !complete(idsFor(f.Graph), p.Groups) {
		return planning.Plan{}, nil, "global_grouping_not_final"
	}
	templates, err := metadataTemplates()
	if err != nil {
		return planning.Plan{}, nil, "metadata_template_contract"
	}
	groups := []finalizedGroup{}
	for i, ids := range p.Groups {
		g := finalizedGroup{ID: groupID(i), FileIDs: ids}
		for _, file := range f.Files {
			if contains(ids, file.ID) {
				visible := file
				visible.ID = "path:" + *file.NewPath
				g.Files = append(g.Files, visible)
			}
		}
		groups = append(groups, g)
	}
	recorder := &recordingClient{Backend: backend}
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
			recorder.IDs = nil
			for _, g := range batch {
				names = append(names, g.ID)
				recorder.IDs = append(recorder.IDs, g.FileIDs...)
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
			var r llm.Response
			if backend != nil {
				r, err = recorder.ChatWithOptions(ctx, []llm.Message{{Role: "system", Content: system}, {Role: "user", Content: string(encoded)}}, shape, llm.Options{ContextTokens: 16384, OutputTokens: budget, GenerationProfile: profile})
			} else {
				answers := map[string]any{}
				for _, g := range batch {
					if profile == "bounded-category" {
						kind := "fix"
						testOnly := true
						for _, file := range g.Files {
							testOnly = testOnly && strings.HasSuffix(*file.NewPath, "_test.go")
						}
						if testOnly {
							kind = "test"
						}
						answers[g.ID] = category{kind, "none"}
					} else {
						answers[g.ID] = commitText{"settings", "update retry bounds"}
					}
				}
				data, _ := json.Marshal(answers)
				r = llm.Response{Content: string(data), StopReason: "completed"}
			}
			if err != nil || ctx.Err() != nil {
				return planning.Plan{}, recorder.Calls, "metadata_backend_failure"
			}
			if r.StopReason != "completed" {
				return planning.Plan{}, recorder.Calls, "incomplete_metadata_output"
			}
			if profile == "bounded-category" {
				var out map[string]category
				if strictDecode([]byte(r.Content), &out) != nil || len(out) != len(batch) {
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
				if strictDecode([]byte(r.Content), &out) != nil || len(out) != len(batch) {
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
	validated, violations := planning.Validate(data, idsFor(f.Graph), planning.SensitiveValues{}, planning.English)
	if len(violations) > 0 {
		return planning.Plan{}, recorder.Calls, "invalid_final_plan"
	}
	return validated, recorder.Calls, ""
}

func groupID(i int) string { return fmt.Sprintf("G%03d", i+1) }
