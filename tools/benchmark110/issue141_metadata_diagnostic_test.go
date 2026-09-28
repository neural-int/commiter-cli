package main

import (
	"reflect"
	"testing"
)

func TestMetadataDiagnosticsKeepOnlyStructuralCounts(t *testing.T) {
	groups := []issue141Group{{GroupID: "G1"}, {GroupID: "G2"}}
	cases := []struct {
		name string
		data string
		want issue141MetadataDiagnostic
	}{
		{"missing", `{"metadata":[{"group_id":"G1","type":"feat","scope":"","summary":"a","breaking":false}]}`, issue141MetadataDiagnostic{ExpectedGroups: 2, ObservedItems: 1, MissingGroups: 1}},
		{"duplicate and missing breaking", `{"metadata":[{"group_id":"G1","type":"feat","scope":"","summary":"a"},{"group_id":"G1","type":"fix","scope":"","summary":"b","breaking":false}]}`, issue141MetadataDiagnostic{ExpectedGroups: 2, ObservedItems: 2, MissingGroups: 1, DuplicateGroups: 1, MissingBreaking: 1}},
		{"unknown", `{"metadata":[{"group_id":"untrusted-name","type":"feat","scope":"","summary":"a","breaking":false}]}`, issue141MetadataDiagnostic{ExpectedGroups: 2, ObservedItems: 1, MissingGroups: 2, UnknownGroups: 1}},
		{"invalid json", `{`, issue141MetadataDiagnostic{ParseFailure: "invalid_json", ExpectedGroups: 2}},
	}
	for _, tc := range cases {
		got := issue141DiagnoseMetadata([]byte(tc.data), groups)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: got %#v, want %#v", tc.name, got, tc.want)
		}
	}
}
