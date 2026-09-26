package relation

import (
	"reflect"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/gitstate"
)

func TestBuildGraphRetainsAllNodesAndDirectedIndexes(t *testing.T) {
	changes := []gitstate.Change{
		{ID: "F003", Status: "modified"},
		{ID: "F002", Status: "modified"},
		{ID: "F001", Status: "modified"},
	}
	result := Result{Relations: []Relation{{
		SourceID: "F001", TargetID: "F002", Kind: DirectImport, Class: Soft,
		Reason: "observed_import_path", Evidence: Evidence{Type: "import_path", Value: "./b"},
	}}}

	graph, err := BuildGraph(changes, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Nodes) != 3 || len(graph.Components) != 2 {
		t.Fatalf("nodes/components = %d/%d, want 3/2: %#v", len(graph.Nodes), len(graph.Components), graph.Components)
	}
	if !reflect.DeepEqual(graph.Forward["F001"], []string{"F002"}) || len(graph.Forward["F002"]) != 0 {
		t.Fatalf("unexpected forward index: %#v", graph.Forward)
	}
	if !reflect.DeepEqual(graph.Reverse["F002"], []string{"F001"}) || len(graph.Reverse["F001"]) != 0 {
		t.Fatalf("unexpected reverse index: %#v", graph.Reverse)
	}
	if graph.Statistics.DensePairBaseline != 3 || graph.Statistics.CandidatePairCount != 1 || graph.Statistics.ReducedPairCount != 2 || graph.Statistics.PairReductionReason != "no_evidence_path_between_candidate_components" {
		t.Fatalf("unexpected candidate pair statistics: %#v", graph.Statistics)
	}
	if graph.Components[1].FileIDs[0] != "F003" {
		t.Fatalf("isolated node was not retained: %#v", graph.Components)
	}
}

func TestGraphStatisticsCountEdgesByKind(t *testing.T) {
	changes := []gitstate.Change{
		{ID: "F001", Status: "modified"},
		{ID: "F002", Status: "modified"},
		{ID: "F003", Status: "modified"},
		{ID: "F004", Status: "modified"},
	}
	result := Result{Relations: []Relation{
		{SourceID: "F001", TargetID: "F002", Kind: DirectImport, Class: Soft, Reason: "observed_import_path"},
		{SourceID: "F002", TargetID: "F003", Kind: DirectImport, Class: Soft, Reason: "observed_import_path"},
		{SourceID: "F003", TargetID: "F004", Kind: SourceTest, Class: Soft, Reason: "matching_test_path"},
	}}

	graph, err := BuildGraph(changes, result)
	if err != nil {
		t.Fatal(err)
	}
	want := map[Kind]int{DirectImport: 2, SourceTest: 1}
	if !reflect.DeepEqual(graph.Statistics.EdgesByKind, want) {
		t.Fatalf("edge counts by kind = %#v, want %#v", graph.Statistics.EdgesByKind, want)
	}
}

func TestBuildGraphDoesNotConnectDirectoryHintMembers(t *testing.T) {
	changes := make([]gitstate.Change, 80)
	members := make([]string, len(changes))
	for i := range changes {
		id := fileID(i + 1)
		changes[i] = gitstate.Change{ID: id, Status: "modified"}
		members[i] = id
	}
	result := Result{Hints: []Hint{{
		Kind: PathProximity, Evidence: Evidence{Type: "directory", Value: "src"}, FileIDs: members,
	}}}

	graph, err := BuildGraph(changes, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Edges) != 0 || len(graph.Components) != len(changes) {
		t.Fatalf("directory hint created connectivity: edges=%d components=%d", len(graph.Edges), len(graph.Components))
	}
	if graph.Statistics.CandidatePairCount != 0 || graph.Statistics.ReducedPairCount != int64(len(changes)*(len(changes)-1)/2) {
		t.Fatalf("directory hint created candidate pairs: %#v", graph.Statistics)
	}
}

func TestBuildGraphKeepsRenameAsNodeMetadata(t *testing.T) {
	oldPath, newPath := "src/old.go", "src/new.go"
	changes := []gitstate.Change{{ID: "F001", Status: "renamed", OldPath: &oldPath, NewPath: &newPath}}
	result := Result{Relations: []Relation{{
		SourceID: "F001", TargetID: "F001", Kind: GitRename, Class: Hard,
		Reason: "git_reported_rename", Evidence: Evidence{Type: "git_paths", Value: oldPath, Related: newPath},
	}}}

	graph, err := BuildGraph(changes, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Edges) != 0 || len(graph.Forward["F001"]) != 0 || len(graph.Reverse["F001"]) != 0 {
		t.Fatalf("rename became a self edge: %#v", graph)
	}
	if !graph.Nodes[0].Renamed || graph.Nodes[0].OldPath == nil || graph.Nodes[0].NewPath == nil {
		t.Fatalf("rename metadata was lost: %#v", graph.Nodes[0])
	}
	if !reflect.DeepEqual(graph.ReductionReasons, []ReductionReason{{Kind: GitRename, Reason: "rename_retained_as_single_node_metadata", Count: 1}}) {
		t.Fatalf("rename reduction reason was not recorded: %#v", graph.ReductionReasons)
	}
}

func TestBuildGraphOutputIsIndependentOfInputOrder(t *testing.T) {
	changes := []gitstate.Change{{ID: "F002", Status: "modified"}, {ID: "F001", Status: "modified"}, {ID: "F003", Status: "modified"}}
	result := Result{
		Relations: []Relation{
			{SourceID: "F002", TargetID: "F003", Kind: SourceTest, Class: Soft, Reason: "matching_test_path"},
			{SourceID: "F001", TargetID: "F002", Kind: DirectImport, Class: Soft, Reason: "observed_import_path"},
		},
		Observations: []Observation{
			{SourceID: "F002", Kind: DirectImport, Outcome: Unresolved, Reason: "z_missing", Evidence: Evidence{Type: "import_path", Value: "./z"}, CandidateIDs: []string{"F003", "F001"}},
			{SourceID: "F001", Kind: DirectImport, Outcome: Unresolved, Reason: "a_missing", Evidence: Evidence{Type: "import_path", Value: "./a"}},
		},
		Hints: []Hint{
			{Kind: PathProximity, Evidence: Evidence{Type: "directory", Value: "src/z"}, FileIDs: []string{"F003", "F001"}},
			{Kind: PathProximity, Evidence: Evidence{Type: "directory", Value: "src/a"}, FileIDs: []string{"F003", "F001"}},
		},
	}
	first, err := BuildGraph(changes, result)
	if err != nil {
		t.Fatal(err)
	}
	changes[0], changes[2] = changes[2], changes[0]
	result.Relations[0], result.Relations[1] = result.Relations[1], result.Relations[0]
	second, err := BuildGraph(changes, result)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("graph output changed with input order:\nfirst:  %#v\nsecond: %#v", first, second)
	}
}

func TestBuildGraphRejectsInvalidNodeReferences(t *testing.T) {
	tests := []struct {
		name    string
		changes []gitstate.Change
		result  Result
	}{
		{name: "duplicate nodes", changes: []gitstate.Change{{ID: "F001"}, {ID: "F001"}}},
		{name: "missing edge endpoint", changes: []gitstate.Change{{ID: "F001"}}, result: Result{Relations: []Relation{{SourceID: "F001", TargetID: "F002"}}}},
		{name: "missing diagnostic source", changes: []gitstate.Change{{ID: "F001"}}, result: Result{Observations: []Observation{{SourceID: "F002"}}}},
		{name: "missing hint member", changes: []gitstate.Change{{ID: "F001"}}, result: Result{Hints: []Hint{{FileIDs: []string{"F002"}}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := BuildGraph(test.changes, test.result); err == nil {
				t.Fatal("invalid graph input was accepted")
			}
		})
	}
}

func fileID(number int) string {
	if number < 10 {
		return "F00" + string(rune('0'+number))
	}
	return "F0" + string(rune('0'+number/10)) + string(rune('0'+number%10))
}
