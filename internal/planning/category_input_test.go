package planning

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/relation"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestThreePhaseCategoryReceivesDiffWithoutHostIdentity(t *testing.T) {
	p := candidatePrepared(t)
	c := &candidateClient{responses: candidateResponses()}
	if _, err := (ThreePhaseGenerator{Client: c}).Generate(context.Background(), p, English, SensitiveValues{}); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Groups []struct {
			ID       string            `json:"group_id"`
			Contents map[string]string `json:"contents"`
		} `json:"groups"`
	}
	if err := json.Unmarshal([]byte(c.messages[1][1].Content), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Groups) != 1 || payload.Groups[0].ID != "G001" || payload.Groups[0].Contents["one.go"] != p.Document.Files[0].GroupingRawDiff {
		t.Fatalf("Category lost the selected diff: %+v", payload)
	}
	for _, forbidden := range []string{`"file_ids"`, `"head_identity"`, `"worktree_identity"`, "private binary contents"} {
		if strings.Contains(c.messages[1][1].Content, forbidden) {
			t.Fatalf("Category exposed %s", forbidden)
		}
	}
}

func TestCategoryProjectionPreservesSelectedEvidenceAndRoleBoundaries(t *testing.T) {
	path := func(p string) *string { return &p }
	for _, tc := range []struct{ name, language, testPath, status string }{
		{"added test", "Go", "unit_test.go", "added"},
		{"modified test", "Go", "unit_test.go", "modified"},
		{"deleted test", "Go", "unit_test.go", "deleted"},
		{"other language", "TypeScript", "unit.test.ts", "modified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diff := "diff --git a/unit b/unit\nold mode 100644\nnew mode 100755\n@@ -1 +1 @@\n-old\n+new\n@@ -20 +20 @@\n-before\n+after\n"
			files := []contextinput.File{
				{ID: "path:unit", NewPath: path("unit"), Mode: syntax.ModeStructural, GroupingRawDiff: diff},
				{ID: "path:test", NewPath: path(tc.testPath), Status: tc.status, Language: tc.language, Mode: syntax.ModeRawDiff, RawDiff: diff},
			}
			if tc.status == "deleted" {
				files[1].OldPath, files[1].NewPath = files[1].NewPath, nil
			}
			groups := []candidateGroup{{ID: "G001", FileIDs: []string{"F1", "F2"}, Files: files}}
			original, _ := json.Marshal(groups)
			relations := &contextinput.RelationContext{Edges: []relation.Relation{{SourceID: "F2", TargetID: "F1", Kind: relation.SourceTest}}}
			got, err := categoryProjection(groups, relations)
			if err != nil || got[0].Contents["unit"] != diff || got[0].RelatedTests[tc.testPath] != diff || len(got[0].Contents) != 1 {
				t.Fatalf("projection=%+v err=%v", got, err)
			}
			after, _ := json.Marshal(groups)
			if string(original) != string(after) {
				t.Fatal("host membership mutated")
			}
			// Ambiguous/unknown relations and relations across finalized groups
			// must keep the selected test in contents, never fetch another file.
			for _, edges := range [][]relation.Relation{nil, {{SourceID: "F2", TargetID: "outside", Kind: relation.SourceTest}}, {{SourceID: "F2", TargetID: "F1", Kind: relation.ChangedIdentifier}}} {
				got, err = categoryProjection(groups, &contextinput.RelationContext{Edges: edges})
				if err != nil || got[0].RelatedTests != nil || got[0].Contents[tc.testPath] != diff {
					t.Fatalf("unsafe role attribution: %+v %v", got, err)
				}
			}
		})
	}
	// Existing unchanged tests are outside the selected set. A syntactic edge
	// cannot add their contents. Inline tests remain with their implementation.
	f := contextinput.File{NewPath: path("module.rs"), Mode: syntax.ModeRawDiff, RawDiff: "@@ -1 +1 @@\n-old\n+new // inline test\n"}
	groups := []candidateGroup{{ID: "G001", FileIDs: []string{"F1"}, Files: []contextinput.File{f}}}
	got, err := categoryProjection(groups, &contextinput.RelationContext{Edges: []relation.Relation{{SourceID: "unchanged-test", TargetID: "F1", Kind: relation.SourceTest}}})
	if err != nil || len(got[0].Contents) != 1 || got[0].RelatedTests != nil || got[0].Contents["module.rs"] != f.RawDiff {
		t.Fatalf("inline/existing test: %+v %v", got, err)
	}
	// Test-only groups must put the test itself in contents.
	f.NewPath = path("module_test.go")
	groups[0].Files = []contextinput.File{f}
	got, err = categoryProjection(groups, nil)
	if err != nil || got[0].Contents["module_test.go"] != f.RawDiff || got[0].RelatedTests != nil {
		t.Fatal("test-only diff misplaced")
	}
}

func TestCategoryProjectionOpaqueAndRenameState(t *testing.T) {
	old, new, mode, executable := "old.bin", "new.bin", "100644", "100755"
	for _, file := range []contextinput.File{
		{Opaque: true, Mode: syntax.ModeStructural}, {Binary: true, Mode: syntax.ModeRawDiff}, {Mode: syntax.ModeMetadataOnly},
	} {
		file.NewPath, file.OldPath, file.OldMode, file.NewMode = &new, &old, &mode, &executable
		file.RawDiff, file.GroupingRawDiff = "forbidden bytes", "forbidden sidecar"
		file.Status = "renamed"
		got, err := categoryProjection([]candidateGroup{{ID: "G001", FileIDs: []string{"F1"}, Files: []contextinput.File{file}}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(got)
		if strings.Contains(string(encoded), "forbidden") || !reflect.DeepEqual(got[0].Metadata[new].OldPath, &old) || got[0].Metadata[new].NewMode == nil {
			t.Fatalf("opaque/rename state: %s", encoded)
		}
	}
}

func TestThreePhaseFallbackTypesReachFinalValidator(t *testing.T) {
	for _, typ := range []string{"update", "change"} {
		responses := candidateResponses()
		responses[1] = `{"G001":{"type":"` + typ + `","breaking_evidence_ref":"none"}}`
		c := &candidateClient{responses: responses}
		r, err := (ThreePhaseGenerator{Client: c}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
		if err != nil || c.calls != 3 || r.Plan.Commits[0].Type != typ {
			t.Fatalf("fallback %s: %+v %v", typ, r, err)
		}
		plan, _ := json.Marshal(r.Plan)
		if _, v := Validate(plan, []string{"F001", "F002"}, SensitiveValues{}, English); len(v) != 0 {
			t.Fatal(v)
		}
		r.Plan.Commits[0].Scope = typ
		plan, _ = json.Marshal(r.Plan)
		if _, v := Validate(plan, []string{"F001", "F002"}, SensitiveValues{}, English); !containsViolation(v, InvalidScope) {
			t.Fatalf("type scope accepted: %v", v)
		}
	}
}
