package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestIssue142VerifierInputAndDecision(t *testing.T) {
	item := issue142VerificationFixtures()[0]
	prepared, _, candidates, err := issue142DiagnosticInputs(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates = %d", len(candidates))
	}
	system, prompt, schema, err := issue142VerifierInput(prepared, candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(system, "gold") || strings.Contains(string(prompt), item.name) {
		t.Fatal("evaluation labels leaked")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(prompt, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, ok := envelope["candidates"]; ok {
		t.Fatal("alternative candidate list leaked")
	}
	if _, ok := envelope["candidate"]; !ok {
		t.Fatal("selected candidate missing")
	}
	var contract struct {
		Properties struct {
			Decision struct {
				Enum []string `json:"enum"`
			} `json:"decision"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &contract); err != nil {
		t.Fatal(err)
	}
	if len(contract.Properties.Decision.Enum) != 2 || contract.Properties.Decision.Enum[0] != "accept" || contract.Properties.Decision.Enum[1] != "reject" {
		t.Fatalf("schema = %s", schema)
	}
	backend := &issue142StubBackend{responses: []string{`{"decision":"none"}`, `{"decision":"reject"}`}}
	first := issue142VerifierCall(context.Background(), backend, prepared, candidates[0], item.reference, "selected")
	if first.Decision != "" || first.Failure != "invalid_decision" {
		t.Fatalf("first = %+v", first)
	}
	second := issue142VerifierCall(context.Background(), backend, prepared, candidates[0], item.reference, "selected")
	if second.Decision != "reject" || second.Failure != "" {
		t.Fatalf("second = %+v", second)
	}
}
