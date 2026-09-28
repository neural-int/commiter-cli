package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func TestHybridSeedCensusUsesOnlyObservedEdges(t *testing.T) {
	want := map[string][2]int{
		"multi_commit":               {1, 1},
		"cross_directory":            {0, 1},
		"same_directory_independent": {0, 0},
		"mixed_24":                   {0, 0},
		"holdout_split":              {2, 2},
		"holdout_join":               {1, 1},
	}
	for _, item := range issue140AtomicityFixtures() {
		prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
		if err != nil {
			t.Fatal(err)
		}
		for policy, index := range map[string]int{"hybrid-source-test": 0, "hybrid-source-test-import": 1} {
			units, edges, err := issue141HybridUnits(prepared.Document.RelationContext, ids, policy)
			if err != nil {
				t.Fatal(err)
			}
			if edges != want[item.name][index] {
				t.Fatalf("%s %s seed edges = %d, want %d", item.name, policy, edges, want[item.name][index])
			}
			_, falsePairs := issue141SeedPairCounts(units, item.reference)
			if falsePairs != 0 {
				t.Fatalf("%s %s forced %d false pairs", item.name, policy, falsePairs)
			}
		}
	}
}

func TestHybridSeedCounterexamplesAreNotSafeMustLinks(t *testing.T) {
	guardrails := []fixture{
		{name: "source_test_separate_purposes", language: planning.English, files: []fileSpec{
			{path: "src/cache.go", diff: "+func Cache() bool { return true }\n"},
			{path: "src/cache_test.go", diff: "+func TestCache(t *testing.T) { /* independent test cleanup */ }\n"},
		}, reference: [][]string{{"F001"}, {"F002"}}},
		{name: "import_separate_purposes", language: planning.English, files: []fileSpec{
			{path: "src/shared.js", diff: "+export function helper() { return true; }\n"},
			{path: "src/featureB.js", diff: "+import { helper } from './shared.js';\n+export function featureB() { return helper(); }\n"},
		}, reference: [][]string{{"F001"}, {"F002"}}},
	}
	for _, guardrail := range guardrails {
		prepared, ids, _, err := issue128Prepare(context.Background(), guardrail, true, 1024)
		if err != nil {
			t.Fatal(err)
		}
		units, edges, err := issue141HybridUnits(prepared.Document.RelationContext, ids, "hybrid-source-test-import")
		if err != nil {
			t.Fatal(err)
		}
		_, falsePairs := issue141SeedPairCounts(units, guardrail.reference)
		if edges != 1 || falsePairs != 1 {
			t.Fatalf("%s seed edges / false pairs = %d / %d, want 1 / 1", guardrail.name, edges, falsePairs)
		}
	}
}

func TestHybridAssignmentExpandsAllFilesWithoutOracle(t *testing.T) {
	units := []issue141Unit{
		{ID: "U001", FileIDs: []string{"F001", "F002"}},
		{ID: "U002", FileIDs: []string{"F003"}},
	}
	assignment, failures := issue141ValidateAssignments([]byte(`{"assignments":{"U001":"g1","U002":"g2"}}`), issue141UnitIDs(units))
	if len(failures) != 0 {
		t.Fatal(failures)
	}
	expanded, err := issue141ExpandUnits(assignment, units)
	if err != nil {
		t.Fatal(err)
	}
	if !sameGroups([][]string{expanded.Groups[0].FileIDs, expanded.Groups[1].FileIDs}, [][]string{{"F001", "F002"}, {"F003"}}) {
		t.Fatalf("expanded partition = %#v", expanded)
	}
	_, missing := issue141ValidateAssignments([]byte(`{"assignments":{"U001":"g1"}}`), issue141UnitIDs(units))
	if !reflect.DeepEqual(missing, []string{"missing_file_id"}) {
		t.Fatalf("missing unit validation = %#v", missing)
	}
}

func TestHybridInputPreservesRepositoryEvidence(t *testing.T) {
	item := issue140AtomicityFixtures()[0]
	prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
	if err != nil {
		t.Fatal(err)
	}
	units, _, err := issue141HybridUnits(prepared.Document.RelationContext, ids, "hybrid-source-test")
	if err != nil {
		t.Fatal(err)
	}
	_, prompt, _, err := issue141HybridInput(prepared, units, issue141UnitIDs(units))
	if err != nil {
		t.Fatal(err)
	}
	var original, hybrid map[string]json.RawMessage
	if err := json.Unmarshal(prepared.Prompt, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(prompt, &hybrid); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original["repository_input"], hybrid["repository_input"]) || !reflect.DeepEqual(original["relation_context_guidance"], hybrid["relation_context_guidance"]) {
		t.Fatal("hybrid input changed repository evidence or relation guidance")
	}
}
