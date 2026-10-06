package main

import "testing"

func TestDeltaCoverageRejectsPartialOrUnresolvedIR(t *testing.T) {
	before, after, resolved, unresolved := "named entity returns 2", "named entity returns 3", false, true
	valid := contractDelta{&before, &after, &resolved}
	records := []contractRecord{{contractEvidence: contractEvidence{ID: "E001"}}, {contractEvidence: contractEvidence{ID: "E002"}}}
	for _, tc := range []struct {
		name     string
		answer   map[string]contractDelta
		accepted bool
	}{
		{"complete", map[string]contractDelta{"E001": valid, "E002": valid}, true},
		{"missing", map[string]contractDelta{"E001": valid}, false},
		{"unknown instead of required", map[string]contractDelta{"E001": valid, "E999": valid}, false},
		{"unresolved", map[string]contractDelta{"E001": valid, "E002": {&before, &after, &unresolved}}, false},
		{"missing required status", map[string]contractDelta{"E001": valid, "E002": {&before, &after, nil}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if (validateContractDeltas(records, tc.answer) == nil) != tc.accepted {
				t.Fatal("unsafe delta acceptance")
			}
		})
	}
}
