package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func TestIssue141PartitionValidation(t *testing.T) {
	ids := []string{"F001", "F002"}
	cases := []struct {
		name, candidate string
		want            []string
	}{
		{"valid", `{"groups":[{"group_id":"G2","file_ids":["F002"]},{"group_id":"G1","file_ids":["F001"]}]}`, nil},
		{"missing", `{"groups":[{"group_id":"G1","file_ids":["F001"]}]}`, []string{"missing_file_id"}},
		{"duplicate", `{"groups":[{"group_id":"G1","file_ids":["F001"]},{"group_id":"G2","file_ids":["F001","F002"]}]}`, []string{"duplicate_assignment"}},
		{"unknown", `{"groups":[{"group_id":"G1","file_ids":["F001","F002","F999"]}]}`, []string{"unknown_file_id"}},
		{"empty", `{"groups":[{"group_id":"G1","file_ids":[]},{"group_id":"G2","file_ids":["F001","F002"]}]}`, []string{"empty_group"}},
		{"group ID", `{"groups":[{"group_id":"G1","file_ids":["F001"]},{"group_id":"G1","file_ids":["F002"]}]}`, []string{"invalid_group_id"}},
		{"schema", `{"groups":[{"group_id":"G1","file_ids":["F001","F002"],"summary":"x"}]}`, []string{"invalid_schema"}},
		{"JSON", `{"groups":`, []string{"invalid_json"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, failures := issue141ValidatePartition([]byte(tc.candidate), ids)
			if !reflect.DeepEqual(failures, tc.want) {
				t.Fatalf("failures = %v, want %v", failures, tc.want)
			}
		})
	}
}

func TestIssue141UsesSamePreparedRepositoryInput(t *testing.T) {
	item := issue140AtomicityFixtures()[0]
	prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
	if err != nil {
		t.Fatal(err)
	}
	_, pass1, _, err := issue141Pass1Input(prepared, ids)
	if err != nil {
		t.Fatal(err)
	}
	var original, partition map[string]json.RawMessage
	if err := json.Unmarshal(prepared.Prompt, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(pass1, &partition); err != nil {
		t.Fatal(err)
	}
	var left, right any
	if err := json.Unmarshal(original["repository_input"], &left); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(partition["repository_input"], &right); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatal("Pass 1 changed prepared repository evidence")
	}
	if _, ok := partition["constraints"]; ok {
		t.Fatal("Pass 1 unexpectedly requests commit metadata")
	}
}

func TestIssue141MetadataKeepsPartitionAndUsesPlanningValidate(t *testing.T) {
	groups := []issue141Group{{GroupID: "G2", FileIDs: []string{"F002"}}, {GroupID: "G1", FileIDs: []string{"F001"}}}
	ids := []string{"F001", "F002"}
	valid := `{"metadata":[{"group_id":"G1","type":"fix","scope":"parser","summary":"Fix parser","breaking":false},{"group_id":"G2","type":"test","scope":"parser","summary":"Test parser","breaking":false}]}`
	if got := issue141ValidateMetadata([]byte(valid), groups, ids, planning.English); got != "" {
		t.Fatalf("valid metadata rejected: %s", got)
	}
	missing := `{"metadata":[{"group_id":"G1","type":"fix","scope":"parser","summary":"Fix parser","breaking":false}]}`
	if got := issue141ValidateMetadata([]byte(missing), groups, ids, planning.English); got != "invalid_metadata_groups" {
		t.Fatalf("missing metadata = %s", got)
	}
	noBreaking := `{"metadata":[{"group_id":"G1","type":"fix","scope":"parser","summary":"Fix parser"},{"group_id":"G2","type":"test","scope":"parser","summary":"Test parser","breaking":false}]}`
	if got := issue141ValidateMetadata([]byte(noBreaking), groups, ids, planning.English); got != "invalid_metadata_groups" {
		t.Fatalf("missing breaking = %s", got)
	}
}
