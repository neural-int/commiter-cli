package relation

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestGitRenameIsOneHardSelfRelation(t *testing.T) {
	file := fixture("F001", "src/new.go", "go", "")
	old := "src/old.go"
	file.Change.OldPath = &old
	file.Change.Status = "renamed"
	got, err := Extract([]File{file})
	if err != nil {
		t.Fatal(err)
	}
	want := Relation{SourceID: "F001", TargetID: "F001", Kind: GitRename, Class: Hard, Reason: "git_reported_rename", Evidence: Evidence{"git_paths", "src/old.go", "src/new.go"}}
	if !reflect.DeepEqual(got.Relations, []Relation{want}) {
		t.Fatalf("relations = %#v", got.Relations)
	}
}

func TestSourceTestNamingIsSoftAndExact(t *testing.T) {
	files := []File{
		fixture("F001", "src/auth.ts", "typescript", ""),
		fixture("F002", "src/auth.test.ts", "typescript", ""),
		fixture("F003", "elsewhere/auth.ts", "typescript", ""),
	}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if !has(got.Relations, "F002", "F001", SourceTest, Soft) {
		t.Fatalf("missing exact source/test relation: %#v", got.Relations)
	}
	if has(got.Relations, "F002", "F003", SourceTest, Soft) {
		t.Fatalf("matched a different directory: %#v", got.Relations)
	}
}

func TestRootTestsDirectoryMatchesSource(t *testing.T) {
	files := []File{
		fixture("F001", "auth.ts", "typescript", ""),
		fixture("F002", "__tests__/auth.ts", "typescript", ""),
	}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if !has(got.Relations, "F002", "F001", SourceTest, Soft) {
		t.Fatalf("missing root __tests__ source relation: %#v", got.Relations)
	}
}

func TestUnchangedSourceForChangedTestIsRecorded(t *testing.T) {
	files := []File{fixture("F001", "src/auth.test.ts", "typescript", "")}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Relations) != 0 || !hasObservation(got.Observations, "F001", SourceTest, Unresolved) {
		t.Fatalf("unchanged source outcome was not preserved: %#v", got)
	}
}

func TestDirectImportUsesObservedSyntaxAndUniqueChangedTarget(t *testing.T) {
	files := []File{
		fixture("F001", "src/view.ts", "typescript", "import { helper } from './helper'\nexport const view = helper()\n"),
		fixture("F002", "src/helper.ts", "typescript", "export function helper() { return 1 }\n"),
	}
	// The import line is unchanged; only the second line belongs to the diff.
	files[0].Evidence = syntax.Analyze(syntax.Input{
		Language: "typescript", Content: files[0].Content,
		Hunks: []syntax.Hunk{{StartLine: 2, EndLine: 2}},
	}).Evidence
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if !has(got.Relations, "F001", "F002", DirectImport, Soft) {
		t.Fatalf("missing import relation: %#v", got.Relations)
	}
	importCount := 0
	for _, relation := range got.Relations {
		if relation.Kind == DirectImport {
			importCount++
			if relation.Evidence != (Evidence{"import_path", "./helper", "import_statement"}) {
				t.Fatalf("unexpected import evidence: %#v", relation)
			}
		}
	}
	if importCount != 1 {
		t.Fatalf("direct import relations = %d, want 1: %#v", importCount, got.Relations)
	}
	for _, observation := range got.Observations {
		if observation.Kind == DirectImport && observation.Outcome == Unsupported {
			t.Fatalf("resolved import also reported unsupported: %#v", got.Observations)
		}
	}
}

func TestChangedIdentifierReferencesUniqueChangedDeclaration(t *testing.T) {
	files := []File{
		fixture("F001", "src/caller.go", "go", "package p\nfunc call() { changed() }\n"),
		fixture("F002", "src/changed.go", "go", "package p\nfunc changed() {}\n"),
	}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if !has(got.Relations, "F001", "F002", ChangedIdentifier, Soft) {
		t.Fatalf("missing identifier relation: %#v", got.Relations)
	}
	files = append(files, fixture("F003", "other/changed.go", "go", "package other\nfunc changed() {}\n"))
	got, err = Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if has(got.Relations, "F001", "F002", ChangedIdentifier, Soft) || has(got.Relations, "F001", "F003", ChangedIdentifier, Soft) {
		t.Fatalf("ambiguous identifier was resolved: %#v", got.Relations)
	}
	if !hasObservation(got.Observations, "F001", ChangedIdentifier, Ambiguous) {
		t.Fatalf("ambiguous identifier attempt was not recorded: %#v", got.Observations)
	}
}

func TestRustChangedFunctionIdentifierReferencesDeclaration(t *testing.T) {
	files := []File{
		fixture("F001", "src/caller.rs", "rust", "fn caller() { changed(); }\n"),
		fixture("F002", "src/changed.rs", "rust", "fn changed() {}\n"),
	}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if !has(got.Relations, "F001", "F002", ChangedIdentifier, Soft) {
		t.Fatalf("missing Rust function identifier relation: %#v", got.Relations)
	}
}

func TestPathAndManifestRelationsRemainSoft(t *testing.T) {
	files := []File{
		fixture("F001", "app/package.json", "", ""),
		fixture("F002", "app/package-lock.json", "", ""),
		fixture("F003", "app/src/a.ts", "", ""),
		fixture("F004", "app/src/b.ts", "", ""),
	}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if !has(got.Relations, "F001", "F002", ManifestLock, Soft) || !hasHint(got.Hints, "app/src", "F003", "F004") {
		t.Fatalf("missing repository or directory relation: %#v", got.Relations)
	}
}

func TestDirectoryProximityStaysGroupedInsteadOfExpandingPairs(t *testing.T) {
	files := make([]File, 100)
	for i := range files {
		files[i] = fixture(fmt.Sprintf("F%03d", i+1), fmt.Sprintf("src/file-%03d.go", i+1), "go", "")
	}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Relations) != 0 || len(got.Hints) != 1 || len(got.Hints[0].FileIDs) != len(files) {
		t.Fatalf("directory proximity expanded into pairwise edges: relations=%d hints=%#v", len(got.Relations), got.Hints)
	}
}

func TestRootDirectoryProximityIsGrouped(t *testing.T) {
	files := []File{
		fixture("F002", "b.go", "go", ""),
		fixture("F001", "a.go", "go", ""),
	}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Relations) != 0 || !reflect.DeepEqual(got.Hints, []Hint{{Kind: PathProximity, Evidence: Evidence{Type: "directory", Value: "."}, FileIDs: []string{"F001", "F002"}}}) {
		t.Fatalf("root directory hint = %#v; relations = %#v", got.Hints, got.Relations)
	}
}

func TestRustUseIsRecordedAsUnsupported(t *testing.T) {
	files := []File{fixture("F001", "src/lib.rs", "rust", "use crate::foo::Bar;\n")}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	want := Observation{
		SourceID: "F001", Kind: DirectImport, Outcome: Unsupported,
		Reason:   "module_resolution_not_available",
		Evidence: Evidence{Type: "import_path", Value: "use crate::foo::Bar;", Related: "use_declaration"},
	}
	if len(got.Relations) != 0 || !reflect.DeepEqual(got.Observations, []Observation{want}) {
		t.Fatalf("Rust use observations = %#v; relations = %#v", got.Observations, got.Relations)
	}
}

func TestUnsupportedAndAmbiguousImportsAreOmitted(t *testing.T) {
	files := []File{
		fixture("F001", "src/view.ts", "typescript", "import './helper'\nimport 'external'\n"),
		fixture("F002", "src/helper.ts", "typescript", "export const x = 1\n"),
		fixture("F003", "src/helper.js", "javascript", "export const x = 1\n"),
	}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, relation := range got.Relations {
		if relation.Kind == DirectImport {
			t.Fatalf("ambiguous or external import was resolved: %#v", relation)
		}
	}
	if !hasObservation(got.Observations, "F001", DirectImport, Ambiguous) || !hasObservation(got.Observations, "F001", DirectImport, Unsupported) {
		t.Fatalf("ambiguous and unsupported imports were not distinguished: %#v", got.Observations)
	}
}

func TestUnresolvedImportIsRecordedWithoutCreatingRelation(t *testing.T) {
	files := []File{fixture("F001", "src/view.ts", "typescript", "import './missing'\n")}
	got, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Relations) != 0 {
		t.Fatalf("unresolved import created a relation: %#v", got.Relations)
	}
	if !hasObservation(got.Observations, "F001", DirectImport, Unresolved) {
		t.Fatalf("unresolved import was not recorded: %#v", got.Observations)
	}
}

func TestOutputDoesNotDependOnFileOrder(t *testing.T) {
	files := []File{
		fixture("F002", "src/auth.test.ts", "", ""),
		fixture("F001", "src/auth.ts", "", ""),
		fixture("F003", "src/other.ts", "", ""),
	}
	first, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	files[0], files[2] = files[2], files[0]
	second, err := Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("unstable output: %#v / %#v", first, second)
	}
	files = append(files, files[0])
	if _, err := Extract(files); err == nil {
		t.Fatal("duplicate ID accepted")
	}
}

func fixture(id, p, language, content string) File {
	change := gitstate.Change{ID: id, Status: "modified", NewPath: &p, Language: language, WorktreeKind: "file"}
	file := File{Change: change, Content: []byte(content)}
	if content != "" {
		analysis := syntax.Analyze(syntax.Input{Language: language, Content: file.Content, Hunks: []syntax.Hunk{{StartLine: 1, EndLine: strings.Count(content, "\n")}}})
		file.Evidence = analysis.Evidence
	}
	return file
}

func has(values []Relation, source, target string, kind Kind, class Class) bool {
	for _, value := range values {
		if value.SourceID == source && value.TargetID == target && value.Kind == kind && value.Class == class {
			return true
		}
	}
	return false
}

func hasObservation(values []Observation, source string, kind Kind, outcome Outcome) bool {
	for _, value := range values {
		if value.SourceID == source && value.Kind == kind && value.Outcome == outcome {
			return true
		}
	}
	return false
}

func hasHint(values []Hint, dir, first, second string) bool {
	for _, value := range values {
		if value.Kind != PathProximity || value.Evidence.Value != dir {
			continue
		}
		hasFirst, hasSecond := false, false
		for _, id := range value.FileIDs {
			hasFirst = hasFirst || id == first
			hasSecond = hasSecond || id == second
		}
		if hasFirst && hasSecond {
			return true
		}
	}
	return false
}
