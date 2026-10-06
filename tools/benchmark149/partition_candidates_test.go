package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestH19CandidateCoverageDiagnostic(t *testing.T) {
	weak := fixtures()[len(fixtures())-1]
	for _, f := range []fixture{weak, contractFixtures()[2], sharedCalleeGuardrail()} {
		candidates, err := partitionCandidates(f)
		if err != nil {
			t.Fatal(err)
		}
		contained := false
		for _, c := range candidates {
			exact, _, _ := quality(c.Groups, f.Expected)
			contained = contained || exact
		}
		data, _ := json.Marshal(map[string]any{"fixture": f.Name, "candidates": candidates, "gold_contained": contained, "backend_calls": 0})
		fmt.Println(string(data))
	}
}
func TestPartitionCandidatesGoldAndOrderIndependent(t *testing.T) {
	f := contractFixtures()[2]
	a, err := partitionCandidates(f)
	if err != nil {
		t.Fatal(err)
	}
	f.Expected = nil
	f.Name = "unrelated name"
	for i, j := 0, len(f.Files)-1; i < j; i, j = i+1, j-1 {
		f.Files[i], f.Files[j] = f.Files[j], f.Files[i]
	}
	b, err := partitionCandidates(f)
	if err != nil {
		t.Fatal(err)
	}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	if string(x) != string(y) {
		t.Fatal("candidates depend on gold/name/order")
	}
	f.Files = append(f.Files, f.Files[0])
	if _, err = partitionCandidates(f); err == nil {
		t.Fatal("duplicate accepted")
	}
}

func TestSelectionRejectsUnknownAndUnresolved(t *testing.T) {
	candidates := []partitionCandidate{{ID: "C001", Groups: [][]string{{"F001"}}}}
	for _, id := range []string{"", "unresolved", "C999"} {
		if groups, err := selectedPartition(candidates, id); err == nil || groups != nil {
			t.Fatal("invalid selection returned groups")
		}
	}
	if groups, err := selectedPartition(candidates, "C001"); err != nil || len(groups) != 1 {
		t.Fatal("valid selection rejected")
	}
}
