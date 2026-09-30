package main

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestIssue143MembershipPreservesPartitionsAndRepository(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Error(err)
		}
	})
	inputs, _, err := issue143BuildContractManifest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		for _, reverse := range []bool{false, true} {
			_, oldPrompt, oldSchema, err := issue143ContractInput(input, "partition-list", reverse)
			if err != nil {
				t.Fatal(err)
			}
			_, newPrompt, newSchema, err := issue143ContractInput(input, "file-membership", reverse)
			if err != nil {
				t.Fatal(err)
			}
			if string(oldSchema) != string(newSchema) {
				t.Fatal("output schema changed")
			}
			var oldRequest, newRequest map[string]json.RawMessage
			if err = json.Unmarshal(oldPrompt, &oldRequest); err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(newPrompt, &newRequest); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"task", "repository_input"} {
				if string(oldRequest[key]) != string(newRequest[key]) {
					t.Fatalf("%s changed", key)
				}
			}
			if len(newRequest) != 3 || strings.Contains(string(newPrompt), "\"gold\"") || strings.Contains(string(newPrompt), "\"rationale\"") {
				t.Fatal("extra evidence or gold")
			}
			var offered []issue142Candidate
			if err = json.Unmarshal(oldRequest["candidates"], &offered); err != nil {
				t.Fatal(err)
			}
			var table issue143MembershipTable
			if err = json.Unmarshal(newRequest["candidates"], &table); err != nil {
				t.Fatal(err)
			}
			if len(table.CandidateIDs) != len(offered) || len(table.FileGroupRows) != len(input.prepared.Document.Files) {
				t.Fatal("table coverage changed")
			}
			for ci, c := range offered {
				if table.CandidateIDs[ci] != c.ID {
					t.Fatal("candidate order changed")
				}
				groups := map[string][]string{}
				ids := []string{}
				for i, row := range table.FileGroupRows {
					if row.FileID != input.prepared.Document.Files[i].ID || len(row.GroupIDs) != len(offered) {
						t.Fatal("file row changed")
					}
					groups[row.GroupIDs[ci]] = append(groups[row.GroupIDs[ci]], row.FileID)
					ids = append(ids, row.FileID)
				}
				recovered := [][]string{}
				for _, g := range groups {
					recovered = append(recovered, g)
				}
				a, _, err := issue142Canonical(c.Groups, ids)
				if err != nil {
					t.Fatal(err)
				}
				b, _, err := issue142Canonical(recovered, ids)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(a, b) {
					t.Fatal("partition changed by encoding")
				}
			}
		}
	}
}
