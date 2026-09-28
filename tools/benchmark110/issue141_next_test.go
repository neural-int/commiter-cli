package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func TestIssue141KeyedMetadataRejectsDuplicateAndMissingKeys(t *testing.T) {
	groups := []issue141Group{{GroupID: "G1", FileIDs: []string{"F001"}}, {GroupID: "G2", FileIDs: []string{"F002"}}}
	ids := []string{"F001", "F002"}
	item := `{"type":"fix","scope":"parser","summary":"Fix parser","breaking":false}`
	for _, tc := range []struct{ name, candidate, want string }{
		{"valid", `{"G1":` + item + `,"G2":` + item + `}`, ""},
		{"duplicate raw key", `{"G1":` + item + `,"G1":` + item + `}`, "duplicate_json_key"},
		{"missing", `{"G1":` + item + `}`, "invalid_metadata_groups"},
		{"unknown", `{"G1":` + item + `,"G9":` + item + `}`, "invalid_metadata_groups"},
		{"missing breaking", `{"G1":{"type":"fix","scope":"parser","summary":"Fix parser"},"G2":` + item + `}`, "invalid_metadata_groups"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := issue141ValidateKeyedMetadata([]byte(tc.candidate), groups, ids, planning.English); got != tc.want {
				t.Fatalf("failure = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIssue141DecisionRequiresEveryEdgeOnce(t *testing.T) {
	for _, tc := range []struct{ candidate, want string }{
		{`{"decisions":{"E001":true,"E002":false}}`, ""},
		{`{"decisions":{"E001":true}}`, "invalid_edge_decisions"},
		{`{"decisions":{"E001":true,"E003":false}}`, "invalid_edge_decisions"},
		{`{"decisions":{"E001":true,"E001":false}}`, "duplicate_json_key"},
	} {
		_, got := issue141ValidateDecisions([]byte(tc.candidate), 2)
		if got != tc.want {
			t.Fatalf("%s: failure = %q, want %q", tc.candidate, got, tc.want)
		}
	}
}

func TestIssue141DecisionRejectsWrongGuardrailEdge(t *testing.T) {
	item := issue141GuardrailFixtures()[0]
	prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := issue141SoftCandidates(prepared.Document.RelationContext, ids, "soft-source-test-import")
	if err != nil || len(candidates) != 1 {
		t.Fatalf("candidates = %v, err = %v", candidates, err)
	}
	selected := issue141AcceptedContext(prepared.Document.RelationContext, []bool{false}, "soft-source-test-import")
	units, edges, err := issue141HybridUnits(selected, ids, "hybrid-source-test-import")
	if err != nil || edges != 0 || len(units) != 2 {
		t.Fatalf("rejected edge: units = %v, edges = %d, err = %v", units, edges, err)
	}
	_, _, schema, err := issue141DecisionInput(prepared, ids, candidates)
	if err != nil {
		t.Fatal(err)
	}
	var shape map[string]any
	if err := json.Unmarshal(schema, &shape); err != nil {
		t.Fatal(err)
	}
}
