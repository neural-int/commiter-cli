package historyattribution

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

func pipelineRequest() Request {
	path := "file.go"
	return Request{Root: "/local", Since: time.Unix(1, 0), Until: time.Now(), Files: []contextinput.File{{ID: "host-file", Status: "modified", NewPath: &path, Mode: "raw_diff", RawDiff: update("A", "B")}}, AllowPath: func(string) bool { return true }, SafeText: func(string) bool { return true }}
}
func TestSourceAttributionToNewSchema(t *testing.T) {
	for _, name := range []string{"matched", "ambiguous_same_turn", "partial", "unavailable", "no_match", "duplicate_page", "missing_context", "unsafe_body", "add", "rename", "multi_section", "mode_and_text", "conflicting_source", "active_turn", "missing_turn_status"} {
		t.Run(name, func(t *testing.T) {
			request := pipelineRequest()
			operation := op("edit", update("A", "B"))
			h := History{Status: "complete", Operations: []Operation{operation}, Turns: []TurnContext{{Thread: "S", Turn: "T", Status: "completed", Request: "Change A to B", Final: "Changed A to B"}}}
			want := "matched"
			switch name {
			case "ambiguous_same_turn":
				h.Operations = append(h.Operations, op("other", update("A", "B")))
				want = "ambiguous"
			case "partial":
				h.Status = "partial"
				want = "ambiguous"
			case "unavailable":
				h.Status = "unavailable"
				h.Operations = nil
				want = "error"
			case "no_match":
				h.Operations[0].Diff = ptr(update("X", "Y"))
				want = "unmatched"
			case "duplicate_page":
				h.Operations = append(h.Operations, h.Operations[0])
				h.Turns = append(h.Turns, h.Turns[0])
			case "missing_context":
				h.Turns = nil
				want = "error"
			case "unsafe_body":
				request.SafeText = func(s string) bool { return s != "Changed A to B" }
			case "add":
				request.Files[0].Status = "added"
				request.Files[0].RawDiff = "@@ -0,0 +1 @@\n+B\n"
				h.Operations[0].Kind = "add"
				h.Operations[0].Diff = ptr("B\n")
			case "rename":
				old := "old.go"
				request.Files[0].OldPath = &old
				request.Files[0].Status = "renamed"
				h.Operations[0].Kind = "rename"
				h.Operations[0].OldPath = old
			case "multi_section":
				request.Files[0].RawDiff = "diff --git a/file.go b/file.go\n" + update("A", "B") + "diff --git a/file.go b/file.go\n" + update("C", "D")
				h.Operations[0].Diff = ptr(request.Files[0].RawDiff)
			case "mode_and_text":
				request.Files[0].RawDiff = "diff --git a/file.go b/file.go\nold mode 100644\nnew mode 100755\n" + update("A", "B")
				h.Operations[0].Diff = ptr(request.Files[0].RawDiff)
			case "active_turn":
				h.Turns[0].Status = "inProgress"
				want = "ambiguous"
			case "missing_turn_status":
				h.Turns[0].Status = ""
				want = "error"
			case "conflicting_source":
				other := h.Operations[0]
				other.Diff = ptr(update("A", "C"))
				h.Operations = append(h.Operations, other)
				want = "error"
			}
			sources := map[string]ContextSource{}
			got, err := attributeWithSources(request, h, sources)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Files) != 1 || got.Files[0].FileID != "host-file" || got.Files[0].RawDiff != request.Files[0].RawDiff {
				t.Fatalf("lost fixed input: %+v", got)
			}
			b := got.Files[0].Blocks[0]
			if b.Status != want {
				t.Fatalf("got %s reasons=%v want %s", b.Status, b.Reasons, want)
			}
			if name == "unsafe_body" && (len(got.Contexts[0].FinalResponses) != 0 || len(b.OmissionReasons) != 1 || b.OmissionReasons[0] != "final_response_omitted") {
				t.Fatal("missing body omission")
			}
			if name == "ambiguous_same_turn" && (len(got.Contexts) != 1 || len(b.Candidates) != 1 || len(sources["H1"].Edits) != 2) {
				t.Fatal("context dedup lost ambiguity or sources")
			}
			if len(got.Contexts) > 0 {
				wire, _ := json.Marshal(got.Contexts)
				if strings.Contains(string(wire), "thread") || strings.Contains(string(wire), "item") {
					t.Fatal("source identities leaked")
				}
			}
		})
	}
}

func TestCollectPipelineRPCAndResponseCap(t *testing.T) {
	for _, mode := range []string{"ok", "large_valid", "initialize", "out_of_scope"} {
		t.Run(mode, func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, mode)
			request := pipelineRequest()
			request.Root = root
			request.Since = time.Unix(1, 0)
			request.Until = time.Now()
			ctx := context.WithValue(context.Background(), memorySamplerContext{}, func() (memorySample, error) { return memorySample{RSSBytes: 64 * 1024 * 1024, Pressure: 1}, nil })
			got, err := Collect(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "initialize" {
				if got.Status != "unavailable" || got.History.Files[0].Blocks[0].Status != "error" {
					t.Fatalf("%+v", got)
				}
				return
			}
			if got.Status != "available" || got.History.Files[0].Blocks[0].Status != "matched" || got.Contexts != 1 {
				t.Fatalf("%+v", got)
			}
			if got.History.Contexts[0].ContextRef != "H1" || len(got.ContextSources) != 1 {
				t.Fatal("source mapping missing")
			}
			if mode == "large_valid" && got.ReceivedBytes < receiveLimit {
				t.Fatal("fixture did not cross old 4MB cumulative ceiling")
			}
		})
	}
}

func TestAttributionRespectsDeniedRenameOrigin(t *testing.T) {
	request := pipelineRequest()
	old := "private/old.go"
	request.Files[0].OldPath = &old
	request.Files[0].Status = "renamed"
	request.AllowPath = func(path string) bool { return !strings.HasPrefix(path, "private/") }
	operation := op("edit", update("A", "B"))
	operation.Kind = "rename"
	operation.OldPath = old
	h := History{Status: "complete", Operations: []Operation{operation}, Turns: []TurnContext{{Thread: "S", Turn: "T", Status: "completed", Request: "Rename a file", Final: "Renamed a file"}}}
	got, err := attribute(request, h)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Contexts) != 0 || len(got.Files[0].Blocks[0].Candidates) != 0 || got.Acquisition.Status != "partial" {
		t.Fatal("denied old path supplied history evidence")
	}
}
