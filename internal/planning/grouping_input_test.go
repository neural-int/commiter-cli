package planning

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestThreePhaseProjectsOnlyGroupingAndPreservesOriginalInput(t *testing.T) {
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
	if string(changed["mode"]) != `"raw_diff"` || len(changed["raw_diff"]) == 0 || len(changed["syntax_context"]) == 0 {
		t.Fatalf("raw/context projection missing: %v", changed)
	}
	if string(opaque["mode"]) != `"metadata_only"` || len(opaque["raw_diff"])+len(opaque["syntax_context"]) != 0 {
		t.Fatal("opaque content leaked")
	}
	var metadata struct {
		Groups []struct {
			Files []contextinput.File `json:"files"`
		} `json:"groups"`
	}
	for _, phase := range []int{2} {
		if err := json.Unmarshal([]byte(c.messages[phase][1].Content), &metadata); err != nil {
			t.Fatal(err)
		}
		originalFile := metadata.Groups[0].Files[1]
		if originalFile.Mode != syntax.ModeStructural || originalFile.RawDiff != "" || originalFile.HeadIdentity == nil || *originalFile.HeadIdentity != identity || !reflect.DeepEqual(originalFile.Evidence, file.Evidence) {
			t.Fatalf("phase %d lost original data: %+v", phase, originalFile)
		}
		if strings.Contains(c.messages[phase][1].Content, `"syntax_context"`) {
			t.Fatal("grouping context reached Category/Text")
		}
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
