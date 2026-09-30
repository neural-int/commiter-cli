package main

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestIssue143FrozenInputsAndNoGoldLeak(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Error(err)
		}
	})
	inputs, err := issue143Prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(issue143ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var frozen issue143Manifest
	if err = json.Unmarshal(data, &frozen); err != nil {
		t.Fatal(err)
	}
	current := issue143Manifest{Models: issue143Models, HelperHash: issue143HelperHash, OutputTokens: 2048, TimeoutSeconds: 120, Reverse: issue142BidirectionalOrder}
	for _, input := range inputs {
		current.Contracts = append(current.Contracts, input.contract)
	}
	if !reflect.DeepEqual(current, frozen) {
		t.Fatal("input manifest changed")
	}
	// Check against independently recorded #142 measurements, not newly generated values.
	historical := []string{"docs/benchmarks/issue-142-prerank-top2-2026-09-29.jsonl", "docs/benchmarks/issue-142-capped-selector-2026-09-29.jsonl", "docs/benchmarks/issue-142-two-selectors-2026-09-29.jsonl"}
	observed := map[string]map[string]bool{}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			if v["arm"] == "weighted" && v["prompt_sha256"] != nil {
				name := v["fixture"].(string)
				if observed[name] == nil {
					observed[name] = map[string]bool{}
				}
				observed[name][v["prompt_sha256"].(string)+"/"+v["schema_sha256"].(string)] = true
			}
			for _, child := range v {
				visit(child)
			}
		case []any:
			for _, child := range v {
				visit(child)
			}
		}
	}
	for _, file := range historical {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var row any
			if err = json.Unmarshal([]byte(line), &row); err != nil {
				t.Fatal(err)
			}
			visit(row)
		}
	}
	guards, joins, splits := 0, 0, 0
	for _, input := range inputs {
		for order, reverse := range []bool{false, true} {
			if input.contract.Set == "known" && !observed[input.item.name][input.contract.PromptHashes[order]+"/"+input.contract.SchemaHashes[order]] {
				t.Fatalf("%s differs from #142", input.item.name)
			}
			system, prompt, schema, err := issue142GenericForcedInput(input.prepared, input.contract.Candidates, reverse)
			if err != nil {
				t.Fatal(err)
			}
			var request map[string]json.RawMessage
			if err = json.Unmarshal(prompt, &request); err != nil {
				t.Fatal(err)
			}
			if len(request) != 3 || request["task"] == nil || request["candidates"] == nil || request["repository_input"] == nil {
				t.Fatal("unexpected request field")
			}
			if strings.Contains(string(prompt), "\"gold\"") || strings.Contains(system, "gold") || strings.Contains(string(schema), "none") {
				t.Fatal("evaluation label leaked or none allowed")
			}
		}
		if input.contract.Set == "holdout" {
			if input.contract.Guardrail {
				guards++
			}
			switch input.contract.Category {
			case "join":
				joins++
			case "split":
				splits++
			default:
				t.Fatal("invalid holdout category")
			}
			if input.contract.Guardrail {
				found := false
				if input.prepared.Document.RelationContext != nil {
					for _, edge := range input.prepared.Document.RelationContext.Edges {
						if edge.Kind == "source_test" {
							found = true
						}
					}
				}
				// Stem/docs guardrail has a shared basename, not a source_test edge.
				if !found && input.item.name != "h143_stem_docs_independent" {
					t.Fatal("misleading source/test evidence absent")
				}
			}
		}
	}
	if len(inputs) != 12 || guards != 2 || joins != 4 || splits != 4 {
		t.Fatalf("unexpected coverage %d %d %d %d", len(inputs), guards, joins, splits)
	}
}
