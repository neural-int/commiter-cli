package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestIssue142GenericForcedInputSupportsThreeCandidates(t *testing.T) {
	item := issue142NextFixtures()[0]
	input, err := issue142BuildNextInputs(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if len(input.lexical) != 3 {
		t.Fatal("expected three candidates")
	}
	forward, promptA, schemaA, err := issue142GenericForcedInput(input.prepared, input.lexical, false)
	if err != nil {
		t.Fatal(err)
	}
	reverse, promptB, schemaB, err := issue142GenericForcedInput(input.prepared, input.lexical, true)
	if err != nil {
		t.Fatal(err)
	}
	if forward != reverse || strings.Contains(forward, "two complete partition") || strings.Contains(forward, "Return none") {
		t.Fatal("forced system mentions two or none")
	}
	if !bytes.Equal(schemaA, schemaB) || bytes.Equal(promptA, promptB) {
		t.Fatal("schema or presentation order changed unexpectedly")
	}
	var schema struct {
		Properties struct {
			CandidateID struct {
				Enum []string `json:"enum"`
			} `json:"candidate_id"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schemaA, &schema); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(schema.Properties.CandidateID.Enum, ","); got != "C001,C002,C003" {
		t.Fatalf("enum=%s", got)
	}
}

func TestIssue142RelationAnnotationAddsOnlyOneField(t *testing.T) {
	marker := []byte("\"relation_context\":{")
	annotation := []byte("\"relation_context\":{\"relation_interpretation\":\"structural_hint_not_shared_purpose_proof\",")
	for _, name := range issue142RelationFixtures {
		t.Run(name, func(t *testing.T) {
			var item fixture
			for _, candidate := range issue142TournamentFixtures() {
				if candidate.name == name {
					item = candidate
					break
				}
			}
			prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
			if err != nil {
				t.Fatal(err)
			}
			raw, _, err := issue142Generate(prepared.Document, ids)
			if err != nil {
				t.Fatal(err)
			}
			expanded, err := issue142ExpandCandidates(prepared.Document, ids, raw)
			if err != nil {
				t.Fatal(err)
			}
			for _, reverse := range []bool{false, true} {
				systemA, promptA, schemaA, err := issue142SelectionInputModeWithSchemaOrder(prepared, expanded[:2], reverse, false, true)
				if err != nil {
					t.Fatal(err)
				}
				systemB, promptB, schemaB, err := issue142AnnotatedRelationInput(prepared, expanded[:2], reverse)
				if err != nil {
					t.Fatal(err)
				}
				if systemA != systemB || !bytes.Equal(schemaA, schemaB) {
					t.Fatal("system or schema changed")
				}
				if bytes.Count(promptB, annotation) != 1 || !bytes.Equal(promptA, bytes.Replace(promptB, annotation, marker, 1)) {
					t.Fatal("prompt differs beyond annotation")
				}
			}
		})
	}
}

func TestIssue142WeightedBridgeThresholdKeepsGoldGroups(t *testing.T) {
	item := issue142NextFixtures()[3]
	input, err := issue142BuildNextInputs(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	edges, err := issue142WeightedPairs(input.prepared.Document, input.ids)
	if err != nil {
		t.Fatal(err)
	}
	groups, accepted, err := issue142WeightedPartition(input.ids, edges, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	if !sameGroups(groups, item.reference) || len(accepted) != 2 {
		t.Fatalf("groups=%v accepted=%v", groups, accepted)
	}
}
