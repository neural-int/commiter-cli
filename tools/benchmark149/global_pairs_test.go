package main

import "testing"

func TestGlobalPairsRejectIncompleteAndContradictory(t *testing.T) {
	ids := []string{"F001", "F002", "F003"}
	if _, err := partitionFromGlobalPairs(ids, map[string]string{"P001": "S", "P002": "D", "P003": "S"}); err == nil {
		t.Fatal("transitive contradiction accepted")
	}
	for _, a := range []map[string]string{
		{"P001": "D", "P002": "D"},
		{"P001": "D", "P002": "D", "P999": "D"},
		{"P001": "D", "P002": "U", "P003": "D"},
		{"P001": "D", "P002": "X", "P003": "D"},
	} {
		if g, err := partitionFromGlobalPairs(ids, a); err == nil || g != nil {
			t.Fatal("invalid pairs produced groups")
		}
	}
	g, err := partitionFromGlobalPairs(ids, map[string]string{"P001": "S", "P002": "D", "P003": "D"})
	if err != nil || len(g) != 2 {
		t.Fatal("valid partition rejected")
	}
	if _, err := globalPairs([]string{"F001", "F001"}); err == nil {
		t.Fatal("duplicate accepted")
	}
	pairs, err := globalPairs(fixtureIDs(fixtures()[len(fixtures())-1]))
	if err != nil || len(pairs) != 120 {
		t.Fatal("16 file bound")
	}
}
