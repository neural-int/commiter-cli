package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/relation"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestRelationContextSyntheticFixtureMetricsAndGrouping(t *testing.T) {
	changes := []gitstate.Change{
		{ID: "F001", Status: "M", NewPath: stringPointer("src/auth.go")},
		{ID: "F002", Status: "M", NewPath: stringPointer("src/auth_test.go")},
		{ID: "F003", Status: "M", NewPath: stringPointer("src/login.go")},
		{ID: "F004", Status: "M", NewPath: stringPointer("deps/go.mod")},
		{ID: "F005", Status: "M", NewPath: stringPointer("deps/go.sum")},
		{ID: "F006", Status: "M", NewPath: stringPointer("pkg/a.go")},
		{ID: "F007", Status: "M", NewPath: stringPointer("shared/one.go")},
		{ID: "F008", Status: "M", NewPath: stringPointer("shared/two.go")},
	}
	extracted := relation.Result{
		Relations: []relation.Relation{
			{SourceID: "F002", TargetID: "F001", Kind: relation.SourceTest, Class: relation.Soft, Reason: "matching_test_path", Evidence: relation.Evidence{Type: "test_source_paths", Value: "src/auth_test.go", Related: "src/auth.go"}},
			{SourceID: "F003", TargetID: "F001", Kind: relation.DirectImport, Class: relation.Soft, Reason: "observed_import_path", Evidence: relation.Evidence{Type: "import_path", Value: "./auth"}},
			{SourceID: "F004", TargetID: "F005", Kind: relation.ManifestLock, Class: relation.Soft, Reason: "known_manifest_lock_pair", Evidence: relation.Evidence{Type: "manifest_lock_pair", Value: "go.mod", Related: "go.sum"}},
		},
		Observations: []relation.Observation{
			{SourceID: "F007", Kind: relation.DirectImport, Outcome: relation.Ambiguous, Reason: "multiple_changed_targets", Evidence: relation.Evidence{Type: "import_path", Value: "./common"}, CandidateIDs: []string{"F006", "F008"}},
			{SourceID: "F008", Kind: relation.DirectImport, Outcome: relation.Unresolved, Reason: "target_not_in_changed_file_set", Evidence: relation.Evidence{Type: "import_path", Value: "./missing"}},
		},
		Hints: []relation.Hint{{Kind: relation.PathProximity, Evidence: relation.Evidence{Type: "directory", Value: "shared"}, FileIDs: []string{"F007", "F008"}}},
	}
	graph, err := relation.BuildGraph(changes, extracted)
	if err != nil {
		t.Fatal(err)
	}

	trueCandidatePairs := [][2]string{{"F001", "F002"}, {"F001", "F003"}, {"F004", "F005"}}
	recalled := 0
	for _, pair := range trueCandidatePairs {
		if sameComponent(graph.Components, pair[0], pair[1]) {
			recalled++
		}
	}
	if recalled != len(trueCandidatePairs) {
		t.Fatalf("candidate recall=%d/%d", recalled, len(trueCandidatePairs))
	}
	if sameComponent(graph.Components, "F007", "F008") {
		t.Fatal("shared-directory hint connected unrelated fixture files")
	}
	if graph.Statistics.DensePairBaseline != 28 || graph.Statistics.CandidatePairCount != 4 || graph.Statistics.ReducedPairCount != 24 {
		t.Fatalf("candidate reduction statistics=%#v", graph.Statistics)
	}
	if graph.Statistics.ObservationCount != 2 || graph.Statistics.ObservationsByOutcome[relation.Ambiguous] != 1 || graph.Statistics.ObservationsByOutcome[relation.Unresolved] != 1 {
		t.Fatalf("diagnostic counts=%#v", graph.Statistics)
	}

	baseDocument := relationFixtureDocument(changes)
	graphDocument := baseDocument
	graphDocument.RelationContext = contextinput.RelationContextFromGraph(graph)
	language := English
	renderer := Renderer(language)
	baselinePrompt, err := renderer(baseDocument)
	if err != nil {
		t.Fatal(err)
	}
	graphPrompt, err := renderer(graphDocument)
	if err != nil {
		t.Fatal(err)
	}
	config := contextinput.BudgetConfig{Context: "8k", MaxContextTokens: contextinput.Context8K}
	baselineInput, err := contextinput.Prepare(context.Background(), baseDocument, config, renderer, nil)
	if err != nil {
		t.Fatal(err)
	}
	graphInput, err := contextinput.Prepare(context.Background(), graphDocument, config, renderer, nil)
	if err != nil {
		t.Fatal(err)
	}
	if graphInput.RelationContextOmitted {
		t.Fatal("bounded relation context unexpectedly fell back for the small fixture")
	}

	response := `{"commits":[{"type":"fix","scope":"auth","breaking":false,"summary":"update authentication flow","file_ids":["F001","F002","F003"]},{"type":"build","scope":"deps","breaking":false,"summary":"update Go dependency metadata","file_ids":["F004","F005"]},{"type":"refactor","scope":"package","breaking":false,"summary":"reorganize package code","file_ids":["F006"]},{"type":"test","scope":"shared-one","breaking":false,"summary":"update first shared file","file_ids":["F007"]},{"type":"test","scope":"shared-two","breaking":false,"summary":"update second shared file","file_ids":["F008"]}]}`
	baselinePlan := generateFixturePlan(t, baselineInput, response, false)
	graphPlan := generateFixturePlan(t, graphInput, response, false)
	if !reflect.DeepEqual(baselinePlan, graphPlan) {
		t.Fatalf("relation context changed final grouping: %#v / %#v", baselinePlan, graphPlan)
	}

	t.Logf("candidate_recall=%d/%d dense_pairs=%d candidate_pairs=%d reduced_pairs=%d diagnostic_count=%d ambiguous=%d unresolved=%d prompt_bytes_before=%d prompt_bytes_after=%d final_grouping_equal=true", recalled, len(trueCandidatePairs), graph.Statistics.DensePairBaseline, graph.Statistics.CandidatePairCount, graph.Statistics.ReducedPairCount, graph.Statistics.ObservationCount, graph.Statistics.ObservationsByOutcome[relation.Ambiguous], graph.Statistics.ObservationsByOutcome[relation.Unresolved], len(baselinePrompt), len(graphPrompt))
}

func generateFixturePlan(t *testing.T, prepared contextinput.Prepared, response string, repair bool) Plan {
	t.Helper()
	steps := []chatStep{{content: response, stopReason: "completed"}}
	if repair {
		steps = []chatStep{{content: `{"commits":[]}`, stopReason: "completed"}, {content: response, stopReason: "completed"}}
	}
	generated, err := (Generator{Client: &scriptedChat{steps: steps}}).Generate(context.Background(), prepared, English, SensitiveValues{})
	if err != nil {
		t.Fatal(err)
	}
	return generated.Plan
}

func relationFixtureDocument(changes []gitstate.Change) contextinput.Document {
	files := make([]contextinput.File, len(changes))
	for index, change := range changes {
		files[index] = contextinput.File{ID: change.ID, Status: change.Status, OldPath: change.OldPath, NewPath: change.NewPath, ChangeHash: fmt.Sprintf("hash-%s", change.ID), WorktreeKind: "file", Mode: syntax.ModeMetadataOnly}
	}
	return contextinput.Document{SchemaVersion: contextinput.SchemaVersion, Repository: contextinput.Repository{Head: "head", Branch: "main", IndexIdentity: "index"}, Files: files}
}

func sameComponent(components []relation.CandidateComponent, left, right string) bool {
	for _, component := range components {
		leftPresent, rightPresent := false, false
		for _, fileID := range component.FileIDs {
			leftPresent = leftPresent || fileID == left
			rightPresent = rightPresent || fileID == right
		}
		if leftPresent && rightPresent {
			return true
		}
	}
	return false
}

func stringPointer(value string) *string { return &value }

func TestDenseDirectoryFixtureRemainsSparseAndGroupingIndependent(t *testing.T) {
	const fileCount = 100
	changes := make([]gitstate.Change, fileCount)
	files := make([]relation.File, fileCount)
	for index := range changes {
		id := fmt.Sprintf("F%03d", index+1)
		filePath := fmt.Sprintf("shared/file-%03d.go", index+1)
		changes[index] = gitstate.Change{ID: id, Status: "M", NewPath: stringPointer(filePath)}
		files[index] = relation.File{Change: changes[index]}
	}
	extracted, err := relation.Extract(files)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := relation.BuildGraph(changes, extracted)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Edges) != 0 || len(graph.Components) != fileCount || len(graph.Hints) != 1 {
		t.Fatalf("dense directory fixture became connected: edges=%d components=%d hints=%d", len(graph.Edges), len(graph.Components), len(graph.Hints))
	}
	if graph.Statistics.DensePairBaseline != fileCount*(fileCount-1)/2 || graph.Statistics.CandidatePairCount != 0 {
		t.Fatalf("dense directory pair counts=%#v", graph.Statistics)
	}

	baseDocument := relationFixtureDocument(changes)
	graphDocument := baseDocument
	graphDocument.RelationContext = contextinput.RelationContextFromGraph(graph)
	renderer := Renderer(English)
	before, err := renderer(baseDocument)
	if err != nil {
		t.Fatal(err)
	}
	after, err := renderer(graphDocument)
	if err != nil {
		t.Fatal(err)
	}
	config := contextinput.BudgetConfig{Context: "64k", MaxContextTokens: contextinput.Context64K}
	baselinePrepared, err := contextinput.Prepare(context.Background(), baseDocument, config, renderer, nil)
	if err != nil {
		t.Fatal(err)
	}
	graphPrepared, err := contextinput.Prepare(context.Background(), graphDocument, config, renderer, nil)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(changes))
	for index, change := range changes {
		ids[index] = change.ID
	}
	candidate, err := json.Marshal(Plan{SchemaVersion: SchemaVersion, Commits: []Commit{{
		Type: "refactor", Scope: "shared", Summary: "update shared files", FileIDs: ids,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	baselinePlan := generateFixturePlan(t, baselinePrepared, string(candidate), false)
	graphPlan := generateFixturePlan(t, graphPrepared, string(candidate), false)
	if !reflect.DeepEqual(baselinePlan, graphPlan) {
		t.Fatal("directory proximity changed final grouping")
	}
	t.Logf("dense_directory_files=%d edges=%d components=%d auxiliary_hints=%d dense_pairs=%d candidate_pairs=%d prompt_bytes_before=%d prompt_bytes_after=%d final_grouping_equal=true", fileCount, graph.Statistics.EdgeCount, graph.Statistics.ComponentCount, graph.Statistics.AuxiliaryHintCount, graph.Statistics.DensePairBaseline, graph.Statistics.CandidatePairCount, len(before), len(after))
}
