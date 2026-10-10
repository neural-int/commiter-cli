package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

// Numeric observations never retain generated text or unexpected key values.
// DecodeOK is typed JSON decoding only, not authoritative plan validation.
type textDiagnostic struct {
	DecodeOK   bool `json:"typed_decode_ok"`
	Expected   int  `json:"expected_groups"`
	Returned   int  `json:"returned_groups"`
	Missing    int  `json:"missing_groups"`
	Unknown    int  `json:"unknown_groups"`
	MaxScope   int  `json:"max_scope_characters"`
	MaxSummary int  `json:"max_summary_characters"`
}

func diagnoseText(schema json.RawMessage, content string) *textDiagnostic {
	var shape struct {
		Required []string `json:"required"`
	}
	if json.Unmarshal(schema, &shape) != nil {
		return nil
	}
	d := &textDiagnostic{Expected: len(shape.Required)}
	var out map[string]struct {
		Scope   string `json:"scope"`
		Summary string `json:"summary"`
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(content)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil {
		return d
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return d
	}
	d.DecodeOK = true
	d.Returned = len(out)
	for _, id := range shape.Required {
		if _, ok := out[id]; !ok {
			d.Missing++
		}
	}
	for id, text := range out {
		found := false
		for _, expected := range shape.Required {
			found = found || id == expected
		}
		if !found {
			d.Unknown++
		}
		d.MaxScope = max(d.MaxScope, utf8.RuneCountInString(text.Scope))
		d.MaxSummary = max(d.MaxSummary, utf8.RuneCountInString(text.Summary))
	}
	return d
}

func failureCodes(err error) []string {
	if err == nil {
		return nil
	}
	if errors.Is(err, contextinput.ErrTooLarge) {
		return []string{"context_overflow"}
	}
	codes := []string{}
	tokens := strings.FieldsFunc(err.Error(), func(r rune) bool { return !(r >= 'a' && r <= 'z') && r != '_' })
	for _, code := range []string{string(planning.InvalidJSON), string(planning.InvalidSchema), string(planning.InvalidType), string(planning.InvalidScope), string(planning.InvalidSummaryLanguage), string(planning.InvalidSummary), string(planning.InvalidAssignment), string(planning.IncompleteOutput), string(planning.SensitiveOutput), "unresolved_breaking_evidence"} {
		for _, token := range tokens {
			if token == code {
				codes = append(codes, code)
				break
			}
		}
	}
	if len(codes) == 0 {
		codes = append(codes, "unclassified_planner_failure")
	}
	return codes
}
