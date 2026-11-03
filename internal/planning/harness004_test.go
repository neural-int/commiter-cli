package planning

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestHarness004ProductionContract(t *testing.T) {
	c := &candidateClient{responses: candidateResponses()}
	p := candidatePrepared(t)
	before, _ := json.Marshal(p.Document)
	r, e := (ThreePhaseGenerator{Client: c}).Generate(context.Background(), p, English, SensitiveValues{})
	if e != nil || r.Calls != 3 {
		t.Fatalf("%v", e)
	}
	if c.messages[0][0] != c.messages[2][0] || c.messages[0][1] != c.messages[2][1] {
		t.Fatal("shared prefix differs")
	}
	for i, msg := range c.messages {
		if len(msg) != 3 || c.options[i].InputLayout != "harness004" || c.options[i].OutputTokens != 4096 {
			t.Fatal("contract mismatch")
		}
		if strings.Count(msg[2].Content, "Exact JSON Schema") != 1 {
			t.Fatal("schema duplicated")
		}
	}
	if !strings.Contains(c.messages[2][2].Content, "Write a concrete, self-contained summary within 48 characters.") {
		t.Fatal("summary instruction missing")
	}
	after, _ := json.Marshal(p.Document)
	if string(before) != string(after) {
		t.Fatal("host data mutated")
	}
	if CandidateMaxFiles != 4 || CandidateCycleTimeout.Seconds() != 120 {
		t.Fatal("preserved limits changed")
	}
}
func TestNaturalSchemaOrder(t *testing.T) {
	v := map[string]any{"F11": true, "F2": true, "F1": true}
	if string(naturalJSON(v)) != `{"F1":true,"F2":true,"F11":true}` {
		t.Fatal(string(naturalJSON(v)))
	}
}

func TestNaturalJSONRetainsCodeCharacters(t *testing.T) {
	value := map[string]any{"raw_diff": "-if a < b && c > d\n+return true\n"}
	encoded := naturalJSON(value)
	if strings.Contains(string(encoded), `\u003c`) || !json.Valid(encoded) {
		t.Fatalf("code characters changed: %s", encoded)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded["raw_diff"] != value["raw_diff"] {
		t.Fatal("code no longer round trips")
	}
}
