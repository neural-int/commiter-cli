package planning

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestThreePhaseUsesRawDiffInAllPhasesAndPreservesHostInput(t *testing.T) {
	p := candidatePrepared(t)
	file := &p.Document.Files[0]
	identity, mode := "host-identity", "100644"
	file.HeadIdentity, file.WorktreeIdentity = &identity, &identity
	file.OldMode, file.NewMode = &mode, &mode
	file.Staged, file.Unstaged, file.Size = true, true, 100
	file.Evidence[0].StartLine, file.Evidence[0].EndLine = 1, 2
	file.Evidence[0].StartByte, file.Evidence[0].EndByte = 2, 20
	file.GroupingSyntaxContext = []syntax.MinimalContext{{HunkIndex: 0, EnclosingDeclaration: "changed"}}
	// Metadata-only files must not expose any accidental content sidecar.
	p.Document.Files[1].GroupingRawDiff = "private binary contents"
	p.Document.Files[1].GroupingSyntaxContext = file.GroupingSyntaxContext
	original, _ := json.Marshal(p.Document)
	originalContext := append([]syntax.MinimalContext(nil), file.GroupingSyntaxContext...)
	c := &candidateClient{responses: candidateResponses()}
	result, err := (ThreePhaseGenerator{Client: c}).Generate(context.Background(), p, English, SensitiveValues{})
	if err != nil || result.Calls != 3 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var payload struct {
		Files []map[string]json.RawMessage `json:"files"`
	}
	if err := json.Unmarshal([]byte(c.messages[0][1].Content), &payload); err != nil {
		t.Fatal(err)
	}
	for _, projected := range payload.Files {
		for _, name := range []string{"old_mode", "new_mode", "head_identity", "worktree_identity", "worktree_kind", "change_hash", "size", "staged", "unstaged", "evidence", "evidence_reduction", "summary"} {
			if _, exists := projected[name]; exists {
				t.Fatalf("grouping exposed %s", name)
			}
		}
	}
	changed, opaque := payload.Files[1], payload.Files[0]
	if len(changed["raw_diff"]) == 0 || string(changed["change_blocks"]) == "[]" || len(changed["file_id"]) == 0 {
		t.Fatalf("raw/context projection missing: %v", changed)
	}
	if string(opaque["raw_diff"]) != `""` || string(opaque["change_blocks"]) != "[]" {
		t.Fatal("opaque content leaked")
	}
	_, body, _ := strings.Cut(c.messages[1][2].Content, "\n\nPhase input:\n")
	var metadata struct {
		Groups []struct {
			Files []struct {
				RawDiff string `json:"raw_diff"`
			} `json:"files"`
		} `json:"groups"`
	}
	if err := json.Unmarshal([]byte(body), &metadata); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Groups) != 1 || metadata.Groups[0].Files[1].RawDiff != file.GroupingRawDiff {
		t.Fatal("Category lost diff")
	}
	if c.messages[0][1] != c.messages[2][1] {
		t.Fatal("Text shared evidence drift")
	}
	after, _ := json.Marshal(p.Document)
	if string(after) != string(original) || !reflect.DeepEqual(file.GroupingSyntaxContext, originalContext) {
		t.Fatal("projection mutated host input")
	}
}

func TestThreePhaseMissingGroupingSourceFailsBeforeCall(t *testing.T) {
	p := candidatePrepared(t)
	p.Document.Files[0].GroupingRawDiff = ""
	c := &candidateClient{responses: candidateResponses()}
	result, err := (ThreePhaseGenerator{Client: c}).Generate(context.Background(), p, English, SensitiveValues{})
	if err == nil || c.calls != 0 || len(result.Plan.Commits) != 0 {
		t.Fatalf("missing diff: calls=%d result=%+v error=%v", c.calls, result, err)
	}
}
