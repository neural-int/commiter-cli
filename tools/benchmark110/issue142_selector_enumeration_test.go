package main

import "testing"

func TestIssue142SmallPartitions(t *testing.T) {
	cases := []struct {
		ids  []string
		gold [][]string
	}{
		{[]string{"F003", "F001", "F002"}, [][]string{{"F001", "F003"}, {"F002"}}},
		{[]string{"F004", "F002", "F001", "F003"}, [][]string{{"F001", "F002"}, {"F003", "F004"}}},
	}
	for _, tc := range cases {
		partitions, err := issue142SmallPartitions(tc.ids)
		if err != nil {
			t.Fatal(err)
		}
		if len(partitions) != 3 {
			t.Fatalf("got %d partitions, want 3", len(partitions))
		}
		found := false
		for _, groups := range partitions {
			if sameGroups(groups, tc.gold) {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing expected partition: %v", tc.gold)
		}
		duplicates, err := issue142DuplicatePartitions([]issue142Candidate{{ID: "C001", Groups: partitions[0]}}, tc.ids, partitions)
		if err != nil {
			t.Fatal(err)
		}
		if duplicates != 1 {
			t.Fatalf("duplicate count=%d, want 1", duplicates)
		}
	}
}
