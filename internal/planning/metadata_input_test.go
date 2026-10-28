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

func metadataString(s string) *string { return &s }
func TestMetadataFileConditionalFields(t *testing.T) {
	base := contextinput.File{ID: "path:src/retry.go", Status: "modified", OldPath: metadataString("src/retry.go"), NewPath: metadataString("src/retry.go"), OldMode: metadataString("100644"), NewMode: metadataString("100644"), WorktreeKind: "file", Language: "Go", Mode: syntax.ModeRawDiff, RawDiff: "complete diff", HeadIdentity: metadataString("host-head"), ChangeHash: "host-change", Staged: true, Size: 999}
	tests := []struct {
		name   string
		modify func(*contextinput.File)
		want   map[string]any
		absent []string
	}{
		{name: "ordinary", want: map[string]any{"id": "path:src/retry.go", "status": "modified", "new_path": "src/retry.go", "mode": "raw_diff", "raw_diff": "complete diff"}},
		{name: "rename", modify: func(f *contextinput.File) { f.Status = "renamed"; f.OldPath = metadataString("src/old.go") }, want: map[string]any{"old_path": "src/old.go", "new_path": "src/retry.go"}},
		{name: "deleted", modify: func(f *contextinput.File) { f.Status = "deleted"; f.WorktreeKind = "deleted"; f.NewMode = nil }, want: map[string]any{"old_path": "src/retry.go"}, absent: []string{"new_path", "worktree_kind"}},
		{name: "permission", modify: func(f *contextinput.File) { f.NewMode = metadataString("100755") }, want: map[string]any{"old_mode": "100644", "new_mode": "100755"}},
		{name: "added_executable", modify: func(f *contextinput.File) {
			f.Status = "added"
			f.OldMode = nil
			f.OldPath = nil
			f.NewMode = metadataString("100755")
		}, want: map[string]any{"new_mode": "100755"}, absent: []string{"old_mode"}},
		{name: "symlink", modify: func(f *contextinput.File) { f.WorktreeKind = "symlink"; f.NewMode = metadataString("120000") }, want: map[string]any{"worktree_kind": "symlink", "new_mode": "120000"}},
		{name: "gitlink", modify: func(f *contextinput.File) {
			f.WorktreeKind = "gitlink"
			f.Mode = syntax.ModeMetadataOnly
			f.NewMode = metadataString("160000")
		}, want: map[string]any{"worktree_kind": "gitlink", "mode": "metadata_only"}, absent: []string{"raw_diff"}},
		{name: "extensionless", modify: func(f *contextinput.File) {
			f.NewPath = metadataString("scripts/run")
			f.OldPath = nil
			f.Language = "Python"
		}, want: map[string]any{"language": "Python"}},
		{name: "ambiguous_extension", modify: func(f *contextinput.File) {
			f.NewPath = metadataString("src/header.h")
			f.OldPath = nil
			f.Language = "C++"
		}, want: map[string]any{"language": "C++"}},
		{name: "binary", modify: func(f *contextinput.File) { f.Binary = true }, want: map[string]any{"binary": true, "mode": "metadata_only"}, absent: []string{"raw_diff"}},
		{name: "opaque", modify: func(f *contextinput.File) { f.Opaque = true }, want: map[string]any{"opaque": true, "mode": "metadata_only"}, absent: []string{"raw_diff"}},
		{name: "vendor", modify: func(f *contextinput.File) { f.Vendor = true }, want: map[string]any{"vendor": true}},
	}
	allowed := map[string]bool{}
	for _, k := range []string{"id", "status", "old_path", "new_path", "old_mode", "new_mode", "worktree_kind", "language", "binary", "vendor", "opaque", "mode", "raw_diff"} {
		allowed[k] = true
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := base
			if tt.modify != nil {
				tt.modify(&f)
			}
			before, _ := json.Marshal(f)
			projected, e := projectFileMetadata(f)
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := json.Marshal(projected)
			var got map[string]any
			json.Unmarshal(raw, &got)
			for k := range got {
				if !allowed[k] {
					t.Fatalf("unexpected field %s", k)
				}
			}
			for k, v := range tt.want {
				if !reflect.DeepEqual(got[k], v) {
					t.Fatalf("%s=%v want %v", k, got[k], v)
				}
			}
			for _, k := range tt.absent {
				if _, ok := got[k]; ok {
					t.Fatalf("unexpected %s", k)
				}
			}
			if tt.name == "ordinary" && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normal update must have exactly five fields: %s", raw)
			}
			after, _ := json.Marshal(f)
			if string(before) != string(after) {
				t.Fatal("host source changed")
			}
		})
	}
}
func TestMetadataRejectsMissingSources(t *testing.T) {
	for _, f := range []contextinput.File{
		{Status: "modified", Mode: syntax.ModeRawDiff},
		{Status: "renamed", NewPath: metadataString("new.go"), Mode: syntax.ModeRawDiff, RawDiff: "diff"},
		{Status: "modified", NewPath: metadataString("new.go"), Mode: syntax.ModeRawDiff},
	} {
		if _, e := projectFileMetadata(f); e == nil {
			t.Fatal("invalid projection accepted")
		}
	}
}
func TestCategoryTextWireOmitsHostFieldsAndEvidenceFact(t *testing.T) {
	p := candidatePrepared(t)
	responses := candidateResponses()
	responses[1] = `{"G001":{"type":"fix","breaking_evidence_ref":"observed-api"}}`
	c := &candidateClient{responses: responses}
	g := ThreePhaseGenerator{Client: c, Evidence: []BreakingEvidence{{ID: "observed-api", FileIDs: []string{"F001"}, Fact: "host-only observed public change"}}}
	if _, e := g.Generate(context.Background(), p, English, SensitiveValues{}); e != nil {
		t.Fatal(e)
	}
	for phase := 1; phase <= 2; phase++ {
		wire := c.messages[phase][1].Content
		for _, key := range []string{"head_identity", "worktree_identity", "change_hash", "size", "staged", "unstaged", "fact", "evidence_pool_limitation"} {
			if strings.Contains(wire, `"`+key+`"`) {
				t.Fatalf("phase %d leaked %s", phase, key)
			}
		}
		var payload struct {
			Groups   []metadataGroup            `json:"groups"`
			Evidence []metadataBreakingEvidence `json:"breaking_evidence_candidates"`
		}
		if e := json.Unmarshal([]byte(wire), &payload); e != nil {
			t.Fatal(e)
		}
		if phase == 1 && (len(payload.Evidence) != 1 || payload.Evidence[0].ID != "observed-api" || !reflect.DeepEqual(payload.Evidence[0].FileIDs, []string{"F2"})) {
			t.Fatalf("evidence identity lost: %+v", payload.Evidence)
		}
		if phase == 1 && payload.Groups[0].Type != "" {
			t.Fatal("category type must be unset")
		}
		if phase == 2 && (len(payload.Evidence) != 0 || payload.Groups[0].Type != "fix" || payload.Groups[0].Breaking == nil || !*payload.Groups[0].Breaking) {
			t.Fatal("Text lost finalized type")
		}
	}
	if g.Evidence[0].Fact == "" || p.Document.Files[0].ChangeHash == "" {
		t.Fatal("host source was removed")
	}
}
