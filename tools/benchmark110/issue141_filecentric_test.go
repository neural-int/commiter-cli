package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestIssue141FileCentricAssignmentValidation(t *testing.T) {
	ids := []string{"F001", "F002", "F003"}
	cases := []struct {
		name, candidate string
		want            []string
		groups          [][]string
	}{
		{"valid", `{"assignments":{"F003":"G2","F002":"G1","F001":"G1"}}`, nil, [][]string{{"F001", "F002"}, {"F003"}}},
		{"missing", `{"assignments":{"F001":"G1","F002":"G1"}}`, []string{"missing_file_id"}, nil},
		{"unknown", `{"assignments":{"F001":"G1","F002":"G1","F003":"G2","F999":"G2"}}`, []string{"unknown_file_id"}, nil},
		{"duplicate", `{"assignments":{"F001":"G1","F001":"G2","F002":"G1","F003":"G2"}}`, []string{"duplicate_json_key"}, nil},
		{"escaped duplicate", `{"assignments":{"F001":"G1","\u0046001":"G2","F002":"G1","F003":"G2"}}`, []string{"duplicate_json_key"}, nil},
		{"duplicate envelope", `{"assignments":{"F001":"G1","F002":"G1","F003":"G2"},"assignments":{"F001":"G1","F002":"G1","F003":"G2"}}`, []string{"duplicate_json_key"}, nil},
		{"blank label", `{"assignments":{"F001":"","F002":"G1","F003":"G2"}}`, []string{"invalid_group_id"}, nil},
		{"null label", `{"assignments":{"F001":null,"F002":"G1","F003":"G2"}}`, []string{"invalid_group_id"}, nil},
		{"unknown envelope field", `{"assignments":{"F001":"G1","F002":"G1","F003":"G2"},"summary":"x"}`, []string{"invalid_schema"}, nil},
		{"invalid JSON", `{"assignments":`, []string{"invalid_json"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			partition, failures := issue141ValidateAssignments([]byte(tc.candidate), ids)
			if !reflect.DeepEqual(failures, tc.want) {
				t.Fatalf("failures = %v, want %v", failures, tc.want)
			}
			if tc.groups != nil {
				var actual [][]string
				for _, group := range partition.Groups {
					actual = append(actual, group.FileIDs)
				}
				if !reflect.DeepEqual(actual, tc.groups) {
					t.Fatalf("groups = %v, want %v", actual, tc.groups)
				}
			}
		})
	}
}

func TestIssue141FileCentricUsesSameRepositoryEvidence(t *testing.T) {
	item := issue140AtomicityFixtures()[0]
	prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
	if err != nil {
		t.Fatal(err)
	}
	_, prompt, _, err := issue141FileCentricInput(prepared, ids)
	if err != nil {
		t.Fatal(err)
	}
	var original, newInput map[string]json.RawMessage
	if err := json.Unmarshal(prepared.Prompt, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(prompt, &newInput); err != nil {
		t.Fatal(err)
	}
	var left, right any
	if err := json.Unmarshal(original["repository_input"], &left); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(newInput["repository_input"], &right); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatal("file-centric input changed prepared repository evidence")
	}
}
