package planning

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

//go:embed prompts/harness004.json
var harness004Instructions []byte

// naturalLess keeps F2 before F10 in every model-visible schema dictionary.
func naturalLess(a, b string) bool {
	for len(a) > 0 && len(b) > 0 {
		if a[0] >= '0' && a[0] <= '9' && b[0] >= '0' && b[0] <= '9' {
			i, j := 0, 0
			for i < len(a) && a[i] >= '0' && a[i] <= '9' {
				i++
			}
			for j < len(b) && b[j] >= '0' && b[j] <= '9' {
				j++
			}
			x, y := strings.TrimLeft(a[:i], "0"), strings.TrimLeft(b[:j], "0")
			if len(x) != len(y) {
				return len(x) < len(y)
			}
			if x != y {
				return x < y
			}
			if a[:i] != b[:j] {
				return a[:i] < b[:j]
			}
			a, b = a[i:], b[j:]
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}
func naturalJSON(v any) []byte {
	switch v := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return naturalLess(keys[i], keys[j]) })
		var b bytes.Buffer
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			name, _ := json.Marshal(k)
			b.Write(name)
			b.WriteByte(':')
			b.Write(naturalJSON(v[k]))
		}
		b.WriteByte('}')
		return b.Bytes()
	case []any:
		var b bytes.Buffer
		b.WriteByte('[')
		for i, x := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			b.Write(naturalJSON(x))
		}
		b.WriteByte(']')
		return b.Bytes()
	default:
		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		encoder.SetEscapeHTML(false)
		_ = encoder.Encode(v)
		return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
	}
}
func display004(v any) any {
	switch v := v.(type) {
	case []any:
		r := make([]any, len(v))
		for i, x := range v {
			r[i] = display004(x)
		}
		return r
	case map[string]any:
		r := map[string]any{}
		_, file := v["status"]
		_, group := v["files"]
		for k, x := range v {
			if k == "block_id" || k == "execution_ref" || k == "head_identity" || k == "worktree_identity" || k == "change_hash" || k == "size" || k == "staged" || k == "unstaged" || k == "evidence_pool_limitation" {
				continue
			}
			if k == "user_requests" {
				if a, ok := x.([]any); ok && len(a) == 0 {
					continue
				}
			}
			if k == "id" && file {
				k = "file_id"
			} else if k == "id" && group {
				k = "group_id"
			}
			r[k] = display004(x)
		}
		return r
	default:
		return v
	}
}
func describe004(schema map[string]any, path string, required bool) string {
	state := "optional"
	if required {
		state = "required"
	}
	result := fmt.Sprintf("%s: type=%s; %s\n", path, schema["type"], state)
	if p, ok := schema["properties"].(map[string]any); ok {
		keys := []string{}
		for k := range p {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return naturalLess(keys[i], keys[j]) })
		for _, k := range keys {
			req := false
			for _, r := range schema["required"].([]any) {
				if r == k {
					req = true
				}
			}
			result += describe004(p[k].(map[string]any), path+"."+k, req)
		}
	}
	return result
}
func harness004Messages(profile string, encoded, shape []byte, common *map[string]any) ([]llm.Message, error) {
	var data, schema map[string]any
	if err := json.Unmarshal(encoded, &data); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(shape, &schema); err != nil {
		return nil, err
	}
	data = display004(data).(map[string]any)
	delete(data, "task")
	delete(data, "schema_version")
	var prompts map[string]struct {
		System string `json:"system"`
		Prefix string `json:"instructions_prefix"`
	}
	if err := json.Unmarshal(harness004Instructions, &prompts); err != nil {
		return nil, err
	}
	name := strings.TrimPrefix(profile, "bounded-")
	p, ok := prompts[name]
	if !ok {
		return nil, fmt.Errorf("unknown phase")
	}
	shared := map[string]any{}
	for _, k := range []string{"agent_history_contexts", "codex_context"} {
		if h, ok := data[k]; ok {
			shared[k] = h
			delete(data, k)
		}
	}
	if name == "grouping" {
		shared["files"] = data["files"]
		*common = shared
		refs := []any{}
		for _, f := range data["files"].([]any) {
			refs = append(refs, map[string]any{"file_id": f.(map[string]any)["file_id"]})
		}
		data["files"] = refs
	} else if name == "text" {
		shared = *common
		for _, g := range data["groups"].([]any) {
			group := g.(map[string]any)
			refs := []any{}
			for _, f := range group["files"].([]any) {
				refs = append(refs, map[string]any{"file_id": f.(map[string]any)["file_id"]})
			}
			group["files"] = refs
		}
	}
	if _, ok := shared["codex_context"]; ok {
		p.System += codexTrustInstruction
	}
	instruction := p.Prefix + "Field names, types, and required/optional fields:\n" + describe004(schema, "$", true) + "Exact JSON Schema (including enum values, lengths, patterns, and other format constraints):\n" + string(naturalJSON(schema)) + "\nChoose values independently from the current Git evidence; enum order is not a preference.\n\nPhase input:\n"
	return []llm.Message{{Role: "system", Content: p.System}, {Role: "user", Content: string(naturalJSON(shared))}, {Role: "user", Content: instruction + string(naturalJSON(data))}}, nil
}
