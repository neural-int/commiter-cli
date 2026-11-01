package main

import (
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"strings"
)

// Adapt only the research consumer's analysis view. Never change backend wires.
func benchmarkPayload(m []llm.Message) []byte {
	if len(m) != 3 {
		return []byte(m[1].Content)
	}
	_, body, ok := strings.Cut(m[2].Content, "\n\nPhase input:\n")
	if !ok {
		return nil
	}
	var payload, shared map[string]any
	if json.Unmarshal([]byte(body), &payload) != nil || json.Unmarshal([]byte(m[1].Content), &shared) != nil {
		return nil
	}
	if files, ok := shared["files"]; ok {
		payload["files"] = files
	}
	byID := map[string]any{}
	if fs, ok := shared["files"].([]any); ok {
		for _, v := range fs {
			f := v.(map[string]any)
			byID[f["file_id"].(string)] = f
		}
	}
	if groups, ok := payload["groups"].([]any); ok {
		for _, v := range groups {
			g := v.(map[string]any)
			if fs, ok := g["files"].([]any); ok {
				for i, v := range fs {
					f := v.(map[string]any)
					if original, ok := byID[f["file_id"].(string)]; ok {
						fs[i] = original
					}
				}
			}
		}
	}
	var restore func(any)
	restore = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if id, ok := v["file_id"]; ok {
				v["id"] = id
			}
			if id, ok := v["group_id"]; ok {
				v["id"] = id
			}
			for _, x := range v {
				restore(x)
			}
		case []any:
			for _, x := range v {
				restore(x)
			}
		}
	}
	restore(payload)
	b, _ := json.Marshal(payload)
	return b
}
