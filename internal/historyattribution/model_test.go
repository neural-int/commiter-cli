package historyattribution

import (
	"encoding/json"
	"testing"
)

func ptr(s string) *string      { return &s }
func update(a, b string) string { return "@@ -1 +1 @@\n-" + a + "\n+" + b + "\n" }
func op(id, diff string) Operation {
	return Operation{Thread: "S", Turn: "T", Item: id, Path: "file.go", Kind: "update", Diff: ptr(diff), Completed: true}
}
func fixed(diff string, ops ...Operation) Input {
	return Input{Version: 1, Root: "/local", Head: "H", Files: []GitFile{{Path: "file.go", Kind: "update", Diff: diff}}, History: History{Status: "complete", Operations: ops}}
}
func TestConservativeCorrespondence(t *testing.T) {
	a := update("A", "B")
	partial := "@@ -1,2 +1,2 @@\n-A\n-X\n+B\n+Y\n"
	cases := []struct {
		name   string
		in     Input
		status string
	}{
		{"unique", fixed(a, op("1", a)), "matched"},
		{"path_only", fixed(a, op("1", update("X", "Y"))), "unmatched"},
		{"partial", fixed(a, op("1", partial)), "ambiguous"},
		{"same_turn_two_ops", fixed(a, op("1", a), op("2", a)), "ambiguous"},
		{"reversed_direction", fixed(a, op("1", update("B", "A"))), "unmatched"},
		{"sequential_not_replayed", fixed(update("A", "C"), op("1", a), op("2", update("B", "C"))), "ambiguous"},
		{"duplicate_page_same_op", fixed(a, op("1", a), op("1", a)), "matched"},
		{"whitespace_not_normalized", fixed(a, op("1", update(" A", " B"))), "unmatched"},
		{"malformed_history", fixed(a, op("1", "@@ -1,2 +1 @@\n-A\n+B\n")), "error"},
		{"missing_history", fixed(a), "unmatched"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := compare(c.in)
			if len(got.Results) != 1 || got.Results[0].Status != c.status {
				t.Fatalf("got %#v", got.Results)
			}
		})
	}
}
func TestIncompleteAndUnknownCompetitorsNeverMatch(t *testing.T) {
	a := update("A", "B")
	in := fixed(a, op("1", a))
	in.History.Status = "partial"
	if r := compare(in); !r.Fallback || r.Results[0].Status != "ambiguous" {
		t.Fatal(r)
	}
	in = fixed(a, op("1", a), op("2", a))
	in.History.Operations[1].Diff = nil
	r := compare(in).Results[0]
	if r.Status != "error" || r.Reason != "history_parse_failed" {
		t.Fatal(r)
	}
}
func TestDifferentSessionAndRepeatedPositions(t *testing.T) {
	a := update("A", "B")
	x := op("1", a)
	y := x
	y.Thread = "OTHER"
	in := fixed(a, x, y)
	if r := compare(in).Results[0]; r.Status != "ambiguous" || r.SingleTurn {
		t.Fatal(r)
	}
	repeated := "@@ -1,3 +1,3 @@\n-A\n+B\n keep\n-A\n+B\n"
	in = fixed(repeated, op("1", a))
	for _, r := range compare(in).Results {
		if r.Status != "ambiguous" || r.Reason != "overlapping_candidates" {
			t.Fatal(r)
		}
	}
	in = fixed(a, op("1", repeated))
	if r := compare(in).Results[0]; r.Status != "ambiguous" {
		t.Fatal(r)
	}
}
func TestRawAddedDeletedContentsAndEOF(t *testing.T) {
	for _, kind := range []string{"add", "delete"} {
		raw := "- markdown\n+ literal\n"
		bs, err := parseBlocks(kind, raw)
		if err != nil || len(bs) != 1 || len(bs[0].Lines) != 2 {
			t.Fatal(bs, err)
		}
		direction := "+"
		if kind == "delete" {
			direction = "-"
		}
		if bs[0].Lines[0].Direction != direction {
			t.Fatal(bs)
		}
	}
	withEOF := "@@ -1 +1 @@\n-A\n+B\n\\ No newline at end of file\n"
	if _, err := parseBlocks("update", withEOF); err != nil {
		t.Fatal(err)
	}
	in := fixed(withEOF, op("1", update("A", "B")))
	if r := compare(in).Results[0]; r.Status != "ambiguous" {
		t.Fatal(r)
	}
	in = Input{Version: 1, Root: "/local", Head: "H", Files: []GitFile{{Path: "file.go", Kind: "update", Diff: "--- /dev/null\n+++ b/file.go\n@@ -0,0 +1,2 @@\n+hello\n+world\n"}}, History: History{Status: "complete", Operations: []Operation{{Thread: "S", Turn: "T", Item: "I", Path: "file.go", Kind: "add", Diff: ptr("hello\nworld\n"), Completed: true}}}}
	if r := compare(in).Results[0]; r.Status != "matched" {
		t.Fatal(r)
	}
}
func TestMalformedAndMetadataDiffFailClosed(t *testing.T) {
	for _, s := range []string{"", "@@ -1 +1 @@\n-A\n", "@@ -1 +1 @@\n-A\n+B\n+extra\n", "Binary files a and b differ\n", "@@ -1 +1 @@\n?bad\n"} {
		if _, err := parseBlocks("update", s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
}
func TestSnapshotValidationRejectsUnsafeAndConflictingInput(t *testing.T) {
	in := fixed(update("A", "B"), op("1", update("A", "B")), op("1", update("A", "C")))
	if validateInput(in) == nil {
		t.Fatal("conflicting identity accepted")
	}
	for _, path := range []string{"../escape", "/abs", ".env", "private.pem", ".git/config", "a/../b", "credentials.json", "api_key.go"} {
		if safePath(path) {
			t.Fatal(path)
		}
	}
	if !safePath("src/feature.go") {
		t.Fatal("safe path rejected")
	}
	b, _ := json.Marshal(compare(fixed(update("A", "B"), op("1", update("A", "B")))))
	if !json.Valid(b) {
		t.Fatal("invalid comparison")
	}
}
