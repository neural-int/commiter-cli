package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestIssue140AblationChangesOnlySelectedFields(t *testing.T) {
	var many fixture
	for _, item := range issue128Fixtures(24) {
		if item.name == "many_files" {
			many = item
			break
		}
	}
	if many.name == "" {
		t.Fatal("many_files fixture missing")
	}
	for _, arm := range append(append([]issue140Arm{}, issue140ManyArms...), issue140GuidanceArms...) {
		prepared, _, err := issue140Prepare(context.Background(), many, arm, 1024)
		if err != nil {
			t.Fatalf("%s: %v", arm.name, err)
		}
		var outer map[string]json.RawMessage
		if err := json.Unmarshal(prepared.Prompt, &outer); err != nil {
			t.Fatal(err)
		}
		_, hasGuidance := outer["relation_context_guidance"]
		if hasGuidance != arm.guidance {
			t.Errorf("%s: guidance=%v", arm.name, hasGuidance)
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(outer["repository_input"], &input); err != nil {
			t.Fatal(err)
		}
		raw, hasRelation := input["relation_context"]
		if hasRelation == (arm.name == "baseline") {
			t.Errorf("%s: relation presence=%v", arm.name, hasRelation)
		}
		if !hasRelation {
			continue
		}
		var relation map[string]json.RawMessage
		if err := json.Unmarshal(raw, &relation); err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]bool{
			"candidate_components": arm.components,
			"auxiliary_hints":      arm.hints,
			"statistics":           arm.statistics,
			"relations":            arm.edges,
		} {
			_, got := relation[key]
			if got != want {
				t.Errorf("%s: %s present=%v want=%v", arm.name, key, got, want)
			}
		}
		if arm.name == "full" {
			full, _, _, err := issue128Prepare(context.Background(), many, true, 1024)
			if err != nil {
				t.Fatal(err)
			}
			if string(prepared.Prompt) != string(full.Prompt) {
				t.Error("full arm differs from Issue #128 prompt")
			}
		}
	}
}

func TestIssue140FullMatchesIssue128Prompt(t *testing.T) {
	for _, item := range issue128Fixtures(24) {
		if item.name == "relation_diagnostics" || item.name == "rename" {
			continue
		}
		prepared, _, err := issue140Prepare(context.Background(), item, issue140Arm{name: "full", components: true, hints: true, guidance: true, statistics: true, edges: true}, 1024)
		if err != nil {
			t.Fatalf("%s: %v", item.name, err)
		}
		original, _, _, err := issue128Prepare(context.Background(), item, true, 1024)
		if err != nil {
			t.Fatalf("%s: %v", item.name, err)
		}
		if string(prepared.Prompt) != string(original.Prompt) {
			t.Errorf("%s: full arm differs from Issue #128", item.name)
		}
	}
}

func TestIssue140GuidanceStatisticsMatchesPriorArm(t *testing.T) {
	var many fixture
	for _, item := range issue128Fixtures(24) {
		if item.name == "many_files" {
			many = item
			break
		}
	}
	if many.name == "" {
		t.Fatal("many_files fixture missing")
	}
	prior, _, err := issue140Prepare(context.Background(), many, issue140ManyArms[3], 1024)
	if err != nil {
		t.Fatal(err)
	}
	current, _, err := issue140Prepare(context.Background(), many, issue140GuidanceArms[3], 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(current.Prompt) != string(prior.Prompt) {
		t.Error("guidance+statistics prompt differs from the prior guidance-stats-only arm")
	}
}

func TestIssue140PairAndAssignmentMetricsSeparateMergeFromSplit(t *testing.T) {
	expected := [][]string{{"F001", "F002"}, {"F003", "F004"}}
	merged := issue140Row{}
	issue140Score(&merged, [][]string{{"F001", "F002", "F003", "F004"}}, expected)
	if merged.FalseMerge != 4 || merged.FalseSplit != 0 || *merged.PairRecall != (fraction{Hit: 2, Total: 2}) || *merged.PerFileAccuracy != (fraction{Hit: 2, Total: 4}) {
		t.Fatalf("merge metrics: %+v", merged)
	}
	split := issue140Row{}
	issue140Score(&split, [][]string{{"F001"}, {"F002"}, {"F003", "F004"}}, expected)
	if split.FalseMerge != 0 || split.FalseSplit != 1 || *split.PairRecall != (fraction{Hit: 1, Total: 2}) || *split.PerFileAccuracy != (fraction{Hit: 3, Total: 4}) {
		t.Fatalf("split metrics: %+v", split)
	}
}

func TestIssue140IntentInputDoesNotReuseFullPlanningTask(t *testing.T) {
	var item fixture
	for _, candidate := range issue128Fixtures(24) {
		if candidate.name == "normal" {
			item = candidate
			break
		}
	}
	if item.name == "" {
		t.Fatal("normal fixture missing")
	}
	prepared, _, _, err := issue128Prepare(context.Background(), item, false, 1024)
	if err != nil {
		t.Fatal(err)
	}
	system, prompt, err := issue140IntentInput(item, prepared)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(system, "provisional change intents") {
		t.Fatalf("intent system prompt missing intent task: %s", system)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(prompt, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, ok := envelope["planning_constraints"]; ok {
		t.Fatal("intent prompt unexpectedly includes full planning constraints")
	}
	var task string
	if err := json.Unmarshal(envelope["task"], &task); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(task, "commit plan") || !strings.Contains(task, "change intents") {
		t.Fatalf("intent task leaked full planning instruction: %q", task)
	}
}
