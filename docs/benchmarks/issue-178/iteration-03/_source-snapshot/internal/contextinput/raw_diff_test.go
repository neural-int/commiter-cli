package contextinput

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestRawDiffProjectionRetainsCompleteDiffAndHostEvidence(t *testing.T) {
	source := "package example\n" + strings.Repeat("// unchanged comment\n", 40) + "func Added() int { return 1 }\n"
	analysis := syntax.Analyze(syntax.Input{Language: "go", Content: []byte(source), Hunks: []syntax.Hunk{{StartLine: 1, EndLine: 42}}})
	if analysis.Mode != syntax.ModeStructural || len(analysis.Evidence) == 0 {
		t.Fatal("fixture must produce structural evidence")
	}
	// A whole added file includes every source line, not just a purpose hint.
	diff := "@@ -0,0 +1,42 @@\n+" + strings.ReplaceAll(strings.TrimSuffix(source, "\n"), "\n", "\n+") + "\n"
	path := "added.go"
	original := Document{Files: []File{{ID: "F001", NewPath: &path, ChangeHash: "identity", Mode: syntax.ModeStructural, Evidence: analysis.Evidence, GroupingRawDiff: diff, GroupingSyntaxContext: []syntax.MinimalContext{{HunkIndex: 0, EnclosingDeclaration: "Added"}}}}}
	before, _ := json.Marshal(original)
	projected, err := RawDiffProjection(original)
	if err != nil {
		t.Fatal(err)
	}
	file := projected.Files[0]
	if file.Mode != syntax.ModeRawDiff || file.RawDiff != diff || file.GroupingRawDiff != diff || len(file.Evidence) != 0 || file.ID != "F001" || file.ChangeHash != "identity" || !reflect.DeepEqual(file.GroupingSyntaxContext, original.Files[0].GroupingSyntaxContext) {
		t.Fatal("approved source or identity lost")
	}
	after, _ := json.Marshal(original)
	if string(before) != string(after) || len(original.Files[0].Evidence) != len(analysis.Evidence) {
		t.Fatal("host evidence mutated")
	}
}

func TestRawDiffProjectionAlwaysUsesFullDiffAndKeepsMetadataBoundary(t *testing.T) {
	path := "large.go"
	evidence := []syntax.Evidence{{Kind: "function_declaration", Name: "Changed"}}
	diff := strings.Repeat("+changed source\n", 200)
	original := Document{Files: []File{
		{ID: "F001", NewPath: &path, Mode: syntax.ModeStructural, Evidence: evidence, GroupingRawDiff: diff},
		{ID: "F002", Mode: syntax.ModeMetadataOnly, Opaque: true},
		{ID: "F003", Mode: syntax.ModeMetadataOnly, Binary: true},
		{ID: "F004", Mode: syntax.ModeRawDiff, RawDiff: "+fallback source\n"},
	}}
	projected, err := RawDiffProjection(original)
	if err != nil {
		t.Fatal(err)
	}
	if projected.Files[0].Mode != syntax.ModeRawDiff || projected.Files[0].RawDiff != diff || len(projected.Files[0].Evidence) != 0 {
		t.Fatal("compact syntax evidence replaced the complete approved diff")
	}
	if !reflect.DeepEqual(projected.Files[1:], original.Files[1:]) {
		t.Fatal("metadata boundary or existing raw diff changed")
	}
	if !reflect.DeepEqual(original.Files[0].Evidence, evidence) {
		t.Fatal("host evidence mutated")
	}
}

func TestRawDiffProjectionRejectsMissingStructuralSource(t *testing.T) {
	_, err := RawDiffProjection(Document{Files: []File{{ID: "F001", Mode: syntax.ModeStructural, Evidence: []syntax.Evidence{{Kind: "function_declaration", Name: "Changed"}}}}})
	if err == nil {
		t.Fatal("missing approved diff silently fell back to structural evidence")
	}
}
