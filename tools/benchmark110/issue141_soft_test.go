package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestSoftCandidatesAuditMainAndGuardrailFixtures(t *testing.T) {
	items := append(issue140AtomicityFixtures(), issue141GuardrailFixtures()...)
	want := map[string][2]int{
		"multi_commit":                  {1, 1},
		"cross_directory":               {0, 1},
		"same_directory_independent":    {0, 0},
		"mixed_24":                      {0, 0},
		"holdout_split":                 {2, 2},
		"holdout_join":                  {1, 1},
		"source_test_separate_purposes": {1, 1},
		"import_separate_purposes":      {0, 1},
	}
	for _, item := range items {
		prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
		if err != nil {
			t.Fatal(err)
		}
		for policy, index := range map[string]int{"soft-source-test": 0, "soft-source-test-import": 1} {
			candidates, err := issue141SoftCandidates(prepared.Document.RelationContext, ids, policy)
			if err != nil {
				t.Fatal(err)
			}
			if len(candidates) != want[item.name][index] {
				t.Fatalf("%s %s candidates = %d, want %d", item.name, policy, len(candidates), want[item.name][index])
			}
			_, falsePairs := issue141CandidatePairCounts(candidates, item.reference)
			if item.name == "source_test_separate_purposes" && falsePairs != 1 || item.name == "import_separate_purposes" && index == 1 && falsePairs != 1 {
				t.Fatalf("%s %s false candidate pairs = %d, want 1", item.name, policy, falsePairs)
			}
			if item.name != "source_test_separate_purposes" && item.name != "import_separate_purposes" && falsePairs != 0 {
				t.Fatalf("%s %s unexpected false candidate pairs = %d", item.name, policy, falsePairs)
			}
		}
	}
}

func TestSoftCandidateRemainsSplittableAndKeepsFileSchema(t *testing.T) {
	item := issue141GuardrailFixtures()[0]
	prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
	if err != nil {
		t.Fatal(err)
	}
	_, baseline, baselineSchema, err := issue141FileCentricInput(prepared, ids)
	if err != nil {
		t.Fatal(err)
	}
	_, soft, softSchema, err := issue141SoftInput(prepared, ids, "soft-source-test")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(baselineSchema, softSchema) {
		t.Fatal("soft candidate changed the assignment schema")
	}
	var before, after map[string]json.RawMessage
	if err := json.Unmarshal(baseline, &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(soft, &after); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before["repository_input"], after["repository_input"]) || !reflect.DeepEqual(before["relation_context_guidance"], after["relation_context_guidance"]) {
		t.Fatal("soft candidate changed repository evidence or relation guidance")
	}
	var candidates []issue141Candidate
	if err := json.Unmarshal(after["positive_relation_candidates"], &candidates); err != nil || len(candidates) != 1 {
		t.Fatalf("soft candidates = %#v, err = %v", candidates, err)
	}
	partition, failures := issue141ValidateAssignments([]byte(`{"assignments":{"F001":"g1","F002":"g2"}}`), ids)
	if len(failures) != 0 || !sameGroups([][]string{partition.Groups[0].FileIDs, partition.Groups[1].FileIDs}, item.reference) {
		t.Fatalf("separating a candidate relation failed: %#v, %#v", partition, failures)
	}
}
