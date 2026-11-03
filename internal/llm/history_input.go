package llm

import (
	"encoding/json"
	"errors"
	"strings"
)

// OptionalHistoryInput carries the shared message as non-overlapping display
// fragments. Large diffs are transmitted once. The helper chooses history and
// reuses the resulting prepared input for generation in the same request.
type OptionalHistoryInput struct {
	Before               string           `json:"before"`
	After                string           `json:"after"`
	Records              []string         `json:"records"`
	SystemWithoutHistory string           `json:"system_without_history"`
	RecordByteCounts     []int            `json:"record_byte_counts,omitempty"`
	SharedTemplate       *DisplayTemplate `json:"shared_template,omitempty"`
	PhaseTemplate        *DisplayTemplate `json:"phase_template,omitempty"`
}

// LiteralNewlines changes only complete JSON leaves, never template syntax.
func (template DisplayTemplate) LiteralNewlines() (DisplayTemplate, error) {
	var err error
	if json.Valid([]byte(template.Literal)) {
		template.Literal, err = LiteralNewlineDisplay([]byte(template.Literal))
		if err != nil {
			return template, err
		}
	}
	template.Parts = append([]DisplayTemplate(nil), template.Parts...)
	template.IfMissing = append([]DisplayTemplate(nil), template.IfMissing...)
	for i := range template.Parts {
		template.Parts[i], err = template.Parts[i].LiteralNewlines()
		if err != nil {
			return template, err
		}
	}
	for i := range template.IfMissing {
		template.IfMissing[i], err = template.IfMissing[i].LiteralNewlines()
		if err != nil {
			return template, err
		}
	}
	return template, nil
}

// DisplayTemplate is a host-authored projection, not code or a helper-side
// history interpreter. Selection only chooses precomputed display fragments.
// Parts omit empty children before joining, which keeps optional JSON arrays valid.
type DisplayTemplate struct {
	Literal   string            `json:"literal,omitempty"`
	Parts     []DisplayTemplate `json:"parts,omitempty"`
	Separator string            `json:"separator,omitempty"`
	Require   []int             `json:"require,omitempty"`
	IfMissing []DisplayTemplate `json:"if_missing,omitempty"`
}

func (template DisplayTemplate) Render(indices []int) string {
	selected := map[int]bool{}
	for _, index := range indices {
		selected[index] = true
	}
	return template.render(selected)
}
func (template DisplayTemplate) render(selected map[int]bool) string {
	for _, index := range template.Require {
		if !selected[index] {
			if len(template.IfMissing) == 1 {
				return template.IfMissing[0].render(selected)
			}
			return ""
		}
	}
	if len(template.Parts) == 0 {
		return template.Literal
	}
	parts := []string{}
	for _, child := range template.Parts {
		if text := child.render(selected); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, template.Separator)
}

var ErrInputTooLarge = errors.New("final model input exceeds context")
