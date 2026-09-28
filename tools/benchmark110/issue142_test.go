package main

import (
	"encoding/json"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

func TestIssue142CanonicalRejectsIncompletePartitions(t *testing.T) {
	ids := []string{"F001", "F002", "F003"}
	for _, groups := range [][][]string{
		{{"F001"}, {"F002"}},
		{{"F001", "F002"}, {"F002", "F003"}},
		{{"F001", "F002"}, {"F003", "F999"}},
		{{"F001", "F002", "F003"}, {}},
	} {
		if _, _, err := issue142Canonical(groups, ids); err == nil {
			t.Fatalf("accepted invalid partition: %v", groups)
		}
	}
	a, keyA, err := issue142Canonical([][]string{{"F003"}, {"F002", "F001"}}, ids)
	if err != nil {
		t.Fatal(err)
	}
	_, keyB, err := issue142Canonical([][]string{{"F001", "F002"}, {"F003"}}, ids)
	if err != nil || keyA != keyB || len(a) != 2 {
		t.Fatalf("canonicalization differs: %q %q %v", keyA, keyB, err)
	}
}

func TestIssue142SelectionRejectsUnknownAndDuplicateKeys(t *testing.T) {
	candidates := []issue142Candidate{{ID: "C001", Groups: [][]string{{"F001"}, {"F002"}}}}
	for input, want := range map[string]string{
		`{"candidate_id":"C999"}`:                       "unknown_candidate_id",
		`{"candidate_id":"C001","candidate_id":"none"}`: "duplicate_json_key",
		`{"candidate_id":["C001","none"]}`:              "invalid_schema",
		`{"candidate_id":"C001"`:                        "invalid_json",
	} {
		_, got := issue142DecodeSelection([]byte(input), candidates)
		if got != want {
			t.Errorf("%s: got %q want %q", input, got, want)
		}
	}
	if selected, failure := issue142DecodeSelection([]byte(`{"candidate_id":"none"}`), candidates); failure != "" || selected != "none" {
		t.Fatalf("none rejected: %q %q", selected, failure)
	}
}

func TestIssue142GenerateIncludesIsolatesAndIsRepeatable(t *testing.T) {
	ids := []string{"F001", "F002", "F003"}
	files := []contextinput.File{}
	for _, id := range ids {
		p := id + ".txt"
		files = append(files, contextinput.File{ID: id, NewPath: &p})
	}
	doc := contextinput.Document{Files: files}
	a, raw, err := issue142Generate(doc, ids)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := issue142Generate(doc, ids)
	if err != nil {
		t.Fatal(err)
	}
	encodedA, _ := json.Marshal(a)
	encodedB, _ := json.Marshal(b)
	if raw != 5 || string(encodedA) != string(encodedB) {
		t.Fatalf("nonrepeatable candidates: %s / %s", encodedA, encodedB)
	}
	for _, candidate := range a {
		if _, _, err := issue142Canonical(candidate.Groups, ids); err != nil {
			t.Fatalf("incomplete candidate: %v", err)
		}
	}
}
