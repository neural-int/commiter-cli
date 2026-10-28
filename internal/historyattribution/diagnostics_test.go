package historyattribution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiagnosticReasons(t *testing.T) {
	a := update("A", "B")
	for _, tc := range []struct {
		name   string
		input  Input
		reason string
	}{
		{"multiple", fixed(a, op("1", a), op("2", a)), "multiple_history_candidates"},
		{"partial", fixed(a, op("1", update("A", "C"))), "partial_match"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := compare(tc.input).Results[0]
			if r.Status != "ambiguous" || !containsString(r.Reasons, tc.reason) {
				t.Fatal(r)
			}
		})
	}
	in := fixed(a)
	in.History.Status = "partial"
	r := compare(in).Results[0]
	if r.Status != "ambiguous" || !containsString(r.Reasons, "history_incomplete") {
		t.Fatal(r)
	}
	in = fixed(a, op("1", a), op("2", a))
	in.History.Status = "partial"
	r = compare(in).Results[0]
	if !containsString(r.Reasons, "multiple_history_candidates") || !containsString(r.Reasons, "history_incomplete") {
		t.Fatal(r)
	}
	for _, tc := range []struct{ stage, code, want string }{
		{"git.diff", "command_failed", "git_snapshot_unavailable"},
		{"git.verify", "snapshot_changed", "snapshot_changed"},
		{"history.items", "rpc_error", "history_unavailable"},
		{"history.items", "invalid_json", "history_parse_failed"},
		{"git.diff", "unsupported_change_type", "unsupported_change_type"},
	} {
		e := failure(tc.stage, tc.code).(*StageError)
		if e.Category != "error" || e.Reason != tc.want {
			t.Fatal(e)
		}
	}
}
func renameInput(diff string, ops ...Operation) Input {
	return Input{Version: 1, Root: "/local", Head: "H", Files: []GitFile{{Path: "new.go", OldPath: "old.go", Kind: "rename", Diff: diff}}, History: History{Status: "complete", Operations: ops}}
}
func renameOp(id, diff string) Operation {
	o := op(id, diff)
	o.OldPath = "old.go"
	o.Path = "new.go"
	o.Kind = "rename"
	return o
}
func TestRenameContentAndEndpoints(t *testing.T) {
	pure := "diff --git a/old.go b/new.go\nsimilarity index 100%\nrename from old.go\nrename to new.go\n"
	for _, tc := range []struct {
		name           string
		in             Input
		status, reason string
	}{
		{"pure", renameInput(pure, renameOp("1", "")), "matched", "unique_exact_rename"},
		{"edit", renameInput(update("A", "B"), renameOp("1", update("A", "B"))), "matched", "unique_exact_rename"},
		{"multiple", renameInput(pure, renameOp("1", ""), renameOp("2", "")), "ambiguous", "rename_multiple_candidates"},
		{"malformed", renameInput(pure, renameOp("1", "@@ broken")), "ambiguous", "rename_unresolved"},
		{"wrong_path", renameInput(update("A", "B"), op("1", update("A", "B"))), "ambiguous", "rename_unresolved"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := compare(tc.in).Results[0]
			if r.Status != tc.status || r.Reason != tc.reason {
				t.Fatal(r)
			}
		})
	}
	in := renameInput(pure, renameOp("1", ""))
	in.History.Operations[0].Path = "other.go"
	if r := compare(in).Results[0]; r.Status != "ambiguous" || r.Reason != "rename_unresolved" {
		t.Fatal(r)
	}
	// The language source can be syntactically invalid: content matching is independent.
	if r := compare(renameInput(update("func {", "func {{"), renameOp("1", update("func {", "func {{")))).Results[0]; r.Status != "matched" {
		t.Fatal(r)
	}
}
func TestOrderedRewriteEvidence(t *testing.T) {
	a, z := op("1", update("A", "B")), op("2", update("B", "C"))
	a.ItemOrder = 1
	z.ItemOrder = 2
	if r := compare(fixed(update("A", "C"), a, z)).Results[0]; r.Status != "ambiguous" || !containsString(r.Reasons, "history_superseded_or_rewritten") {
		t.Fatal(r)
	}
	a.ItemOrder = 0
	if r := compare(fixed(update("A", "C"), a, z)).Results[0]; containsString(r.Reasons, "history_superseded_or_rewritten") {
		t.Fatal("unproven chronology", r)
	}
}
func TestBinaryCaptureIsExplicitError(t *testing.T) {
	root := gitFixture(t)
	if err := os.WriteFile(filepath.Join(root, "file.go"), []byte{'A', 0, 'B'}, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readGit(context.Background(), root, []string{"file.go"})
	if err == nil || err.(*StageError).Reason != "unsupported_change_type" {
		t.Fatal(err)
	}
}
func TestRenameCaptureThroughComparison(t *testing.T) {
	root := gitFixture(t)
	os.WriteFile(filepath.Join(root, "file.go"), []byte("A\n"), 0600)
	if err := exec.Command("git", "-C", root, "mv", "file.go", "moved.go").Run(); err != nil {
		t.Fatal(err)
	}
	s, err := readGit(context.Background(), root, []string{"file.go", "moved.go"})
	if err != nil {
		t.Fatal(err)
	}
	o := renameOp("move", "")
	o.OldPath = "file.go"
	o.Path = "moved.go"
	in := Input{Version: 1, Root: root, Head: s.head, Files: s.files, History: History{Status: "complete", Operations: []Operation{o}}}
	if err := validateInput(in); err != nil {
		t.Fatal(err)
	}
	if r := compare(in).Results[0]; r.Status != "matched" {
		t.Fatal(r)
	}
}
func TestUnrestrictedHistoryReadsLargeValidMessage(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "large_valid")
	ctx := context.WithValue(context.Background(), unlimitedHistoryContext{}, true)
	h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Status != "complete" || h.Error != nil || h.ReceivedBytes <= receiveLimit || h.MaxMessageBytes <= int(receiveLimit) || len(h.Operations) != 1 {
		t.Fatal(h.Status, h.Error, h.ReceivedBytes, h.MaxMessageBytes)
	}
	if h.RetainedBytes >= int(h.ReceivedBytes) || h.ElapsedMS <= 0 || !h.Unlimited {
		t.Fatal("metrics")
	}
	// Default mode still rejects the same valid large reply.
	h = readHistory(context.Background(), root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Error == nil || h.Error.Code != "receive_limit" {
		t.Fatal("default ceiling lost")
	}
}
func TestResearchArtifactHasNoHiddenSizeCeiling(t *testing.T) {
	in := fixed(update("A", "B"), op("1", update("A", "B")))
	in.History.Turns = []TurnContext{{Thread: "S", Turn: "T", Request: strings.Repeat("a", maxInputBytes+1)}}
	path := filepath.Join(t.TempDir(), "input.json")
	if writeNew(path, in) == nil {
		t.Fatal("default ceiling lost")
	}
	if err := writeNew(path, in, true); err != nil {
		t.Fatal(err)
	}
	if _, err := loadInput(path); err == nil {
		t.Fatal("default input ceiling lost")
	}
	if _, err := loadInput(path, true); err != nil {
		t.Fatal(err)
	}
}

func TestOutOfScopeHistoryIsNotIncomplete(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "out_of_scope")
	h := readHistory(context.Background(), root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Status != "complete" || h.Dropped != 0 || len(h.Operations) != 1 {
		t.Fatal(h.Status, h.Dropped)
	}
}

func TestSplitMergeRequiresCrossFileDisjointContent(t *testing.T) {
	first := op("1", "A\nB\nC\nD\n")
	first.Kind = "add"
	first.Path = "source.go"
	in := Input{Version: 1, Root: "/local", Head: "H", Files: []GitFile{{Path: "one.go", Kind: "untracked", Diff: "A\nB\n"}, {Path: "two.go", Kind: "untracked", Diff: "C\nD\n"}}, History: History{Status: "complete", Operations: []Operation{first}}}
	for _, r := range compare(in).Results {
		if r.Status != "ambiguous" || !containsString(r.Reasons, "split_merge_mapping_uncertain") {
			t.Fatal(r)
		}
	}
	in.Files[1].Diff = "A\nB\n"
	for _, r := range compare(in).Results {
		if containsString(r.Reasons, "split_merge_mapping_uncertain") {
			t.Fatal("same fragment is not split evidence", r)
		}
	}
	first.Diff = ptr("A\nB\n")
	last := op("2", "C\nD\n")
	last.Kind = "add"
	last.Path = "other.go"
	in.Files = []GitFile{{Path: "merged.go", Kind: "untracked", Diff: "A\nB\nC\nD\n"}}
	in.History.Operations = []Operation{first, last}
	r := compare(in).Results[0]
	if r.Status != "ambiguous" || !containsString(r.Reasons, "split_merge_mapping_uncertain") {
		t.Fatal(r)
	}
}
