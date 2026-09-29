package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestIssue142RelationRemovalChangesOnlyRepositoryRelationContext(t *testing.T) {
	for _, name := range issue142RelationFixtures {
		t.Run(name, func(t *testing.T) {
			var item fixture
			for _, candidate := range issue142TournamentFixtures() {
				if candidate.name == name {
					item = candidate
					break
				}
			}
			if item.name == "" {
				t.Fatal("fixture missing")
			}
			prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
			if err != nil {
				t.Fatal(err)
			}
			if prepared.Document.RelationContext == nil {
				t.Fatal("expected relation context")
			}
			raw, _, err := issue142Generate(prepared.Document, ids)
			if err != nil {
				t.Fatal(err)
			}
			expanded, err := issue142ExpandCandidates(prepared.Document, ids, raw)
			if err != nil {
				t.Fatal(err)
			}
			candidates := expanded[:2]
			without := prepared
			without.Document.RelationContext = nil
			for _, reverse := range []bool{false, true} {
				systemA, promptA, schemaA, err := issue142SelectionInputModeWithSchemaOrder(prepared, candidates, reverse, false, true)
				if err != nil {
					t.Fatal(err)
				}
				systemB, promptB, schemaB, err := issue142SelectionInputModeWithSchemaOrder(without, candidates, reverse, false, true)
				if err != nil {
					t.Fatal(err)
				}
				if systemA != systemB || !bytes.Equal(schemaA, schemaB) {
					t.Fatal("system or schema changed")
				}
				var a, b map[string]any
				if err := json.Unmarshal(promptA, &a); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(promptB, &b); err != nil {
					t.Fatal(err)
				}
				repoA := a["repository_input"].(map[string]any)
				repoB := b["repository_input"].(map[string]any)
				if _, ok := repoA["relation_context"]; !ok {
					t.Fatal("with arm missing relation context")
				}
				if _, ok := repoB["relation_context"]; ok {
					t.Fatal("without arm includes relation context")
				}
				delete(repoA, "relation_context")
				normalizedA, _ := json.Marshal(a)
				normalizedB, _ := json.Marshal(b)
				if !bytes.Equal(normalizedA, normalizedB) {
					t.Fatal("another prompt field changed")
				}
			}
		})
	}
}

func TestIssue142NextFixtureGoldIsComplete(t *testing.T) {
	for _, item := range issue142NextFixtures() {
		t.Run(item.name, func(t *testing.T) {
			ids := make([]string, len(item.files))
			for i := range ids {
				ids[i] = "F00" + string(rune('1'+i))
			}
			if _, _, err := issue142Canonical(item.reference, ids); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestIssue142NextRankingFixesSchemaOrderAcrossPresentation(t *testing.T) {
	item := issue142NextFixtures()[0]
	input, err := issue142BuildNextInputs(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	stub := &issue142DiagnosticStub{response: llm.Response{StopReason: "completed", Content: `{"candidate_id":"none"}`}}
	forward := issue142DiagnosticCallModeWithSchemaOrder(context.Background(), stub, "stub", "mlx",
		"next-ranking", "lexical", item, input.prepared, input.lexical, 1, 2048, false, false, true)
	reverse := issue142DiagnosticCallModeWithSchemaOrder(context.Background(), stub, "stub", "mlx",
		"next-ranking", "lexical", item, input.prepared, input.lexical, 2, 2048, true, false, true)
	if forward.SchemaSHA256 != reverse.SchemaSHA256 {
		t.Fatal("schema order changed")
	}
	if forward.PromptSHA256 == reverse.PromptSHA256 {
		t.Fatal("presentation order did not change")
	}
}
