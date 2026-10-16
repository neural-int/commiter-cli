package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestCandidateSourceProjectionRetainsCompleteDiffAndHostEvidence(t *testing.T) {
	source := "package example\n" + strings.Repeat("// unchanged comment\n", 40) + "func Added() int { return 1 }\n"
	analysis := syntax.Analyze(syntax.Input{Language: "go", Content: []byte(source), Hunks: []syntax.Hunk{{StartLine: 1, EndLine: 42}}})
	if analysis.Mode != syntax.ModeStructural || len(analysis.Evidence) == 0 {
		t.Fatal("fixture must produce structural evidence")
	}
	// A whole added file includes every source line, not just a purpose hint.
	diff := "@@ -0,0 +1,42 @@\n+" + strings.ReplaceAll(strings.TrimSuffix(source, "\n"), "\n", "\n+") + "\n"
	path := "added.go"
	original := contextinput.Document{Files: []contextinput.File{{ID: "F001", NewPath: &path, ChangeHash: "identity", Mode: syntax.ModeStructural, Evidence: analysis.Evidence, GroupingRawDiff: diff, GroupingSyntaxContext: []syntax.MinimalContext{{HunkIndex: 0, EnclosingDeclaration: "Added"}}}}}
	before, _ := json.Marshal(original)
	projected, err := candidateSourceProjection(original)
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

func TestCandidateSourceProjectionKeepsSmallerEvidenceAndOpaqueBoundary(t *testing.T) {
	path := "large.go"
	evidence := []syntax.Evidence{{Kind: "function_declaration", Name: "Changed"}}
	original := contextinput.Document{Files: []contextinput.File{
		{ID: "F001", NewPath: &path, Mode: syntax.ModeStructural, Evidence: evidence, GroupingRawDiff: strings.Repeat("+changed source\n", 200)},
		{ID: "F002", Mode: syntax.ModeMetadataOnly, Opaque: true},
		{ID: "F003", Mode: syntax.ModeMetadataOnly, Binary: true},
		{ID: "F004", Mode: syntax.ModeStructural, Evidence: evidence},
	}}
	projected, err := candidateSourceProjection(original)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projected, original) {
		t.Fatal("existing compact/metadata-only representation changed")
	}
}
