package llamacpp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPhaseSchemaValidation(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"G001":{"type":"object","properties":{"scope":{"type":"string","minLength":1,"maxLength":16},"summary":{"type":"string","minLength":1,"maxLength":48}},"required":["scope","summary"],"additionalProperties":false}},"required":["G001"],"additionalProperties":false}`)
	s, err := compileSchema(schema)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{"G001":{"scope":"cli","summary":"設定を検証する"}}`, true},
		{`{"G001":{"scope":"cli","summary":"` + strings.Repeat("あ", 48) + `"}}`, true},
		{`{"G001":{"scope":"cli","summary":"` + strings.Repeat("あ", 49) + `"}}`, false},
		{`{"G001":{"scope":"","summary":"text"}}`, false},
		{`{"G001":{"scope":"cli"}}`, false},
		{`{"G001":{"scope":"cli","summary":"text","other":"field"}}`, false},
		{`{"G001":{"scope":"cli","summary":"text"},"G999":{}}`, false},
		{`{"G001":{"scope":"cli","summary":"first","summary":"last"}}`, false},
		{`{"G001":{"scope":"cli","summary":"first","\u0073ummary":"last"}}`, false},
		{`{"G001":{"scope":"cli","summary":"text"}} {}`, false},
		{`{"G001":{"scope":null,"summary":"text"}}`, false},
		{`[]`, false}, {`{}`, false}, {`not JSON`, false},
	} {
		if err := s.validate([]byte(tc.body)); (err == nil) != tc.valid {
			t.Errorf("valid=%v err=%v", tc.valid, err)
		}
	}
	if err := s.validate([]byte(strings.Repeat(" ", MaxResponseBytes+1))); err == nil {
		t.Fatal("oversize accepted")
	}
}

func TestSchemaRejectsUnknownVocabularyAndMalformedSchema(t *testing.T) {
	for _, raw := range []string{
		`{"type":"string","pattern":".*"}`, `{"type":"string","enum":["x","x"]}`, `{"type":"string","enum":[]}`,
		`{"type":"string","minLength":-1}`, `{"type":"string","minLength":2,"maxLength":1}`, `{"type":"string","minLength":1.5}`,
		`{"type":"array"}`, `{"type":"string","type":"object"}`, `{"type":"string"}true`,
		`{"type":"object","properties":{},"required":["missing"],"additionalProperties":false}`,
		`{"type":"object","properties":{},"required":[],"additionalProperties":true}`,
		`{"type":"object","properties":{},"required":[]}`, `{"type":"string","enum":[1]}`,
		`{"type":"object","properties":{"x":{"type":"string","format":"email"}},"required":[],"additionalProperties":false}`,
	} {
		if _, err := compileSchema(json.RawMessage(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestGroupingSchemaRequiresKnownCompleteAssignment(t *testing.T) {
	s, err := compileSchema(json.RawMessage(`{"type":"object","properties":{"F1":{"type":"string","enum":["G001","G002"]},"F2":{"type":"string","enum":["G001","G002"]}},"required":["F1","F2"],"additionalProperties":false}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{"F1":"G001"}`, `{"F1":"G001","F2":"G999"}`, `{"F1":"G001","F2":"G002","F3":"G001"}`, `{"F1":"G001","F1":"G002","F2":"G001"}`} {
		if err := s.validate([]byte(body)); err == nil {
			t.Errorf("invalid assignment accepted: %s", body)
		}
	}
	if err := s.validate([]byte(`{"F1":"G001","F2":"G001"}`)); err != nil {
		t.Fatal(err)
	}
}
