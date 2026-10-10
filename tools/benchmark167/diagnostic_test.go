package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestTextDiagnosticPreservesNumbersWithoutGeneratedValues(t *testing.T) {
	schema := json.RawMessage(`{"required":["G001"]}`)
	content := `{"G001":{"scope":"settings","summary":"` + strings.Repeat("あ", 49) + `"},"unexpected":{"scope":"private","summary":"fixture-secret"}}`
	d := diagnoseText(schema, content)
	if d == nil || !d.DecodeOK || d.Expected != 1 || d.Returned != 2 || d.Unknown != 1 || d.MaxSummary != 49 {
		t.Fatalf("diagnostic=%#v", d)
	}
	data, _ := json.Marshal(d)
	if strings.Contains(string(data), "private") || strings.Contains(string(data), "fixture-secret") || strings.Contains(string(data), "unexpected") {
		t.Fatal("generated values leaked")
	}
	if codes := failureCodes(errors.New("unsafe fixture-secret (violations: invalid_schema)")); len(codes) != 1 || codes[0] != "invalid_schema" {
		t.Fatalf("codes=%v", codes)
	}
}
