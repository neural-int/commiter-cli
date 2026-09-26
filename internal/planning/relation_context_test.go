package planning

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

func TestGeneratorKeepsCompleteFileAssignmentWithRelationContextOnGenerationAndRepair(t *testing.T) {
	prepared := preparedInputWithRelationContext(t)
	candidate := validPlan()
	for _, repair := range []bool{false, true} {
		name := "generation"
		steps := []chatStep{{content: candidate, stopReason: "completed"}}
		if repair {
			name = "repair"
			steps = []chatStep{{content: `{"commits":[]}`, stopReason: "completed"}, {content: candidate, stopReason: "completed"}}
		}
		t.Run(name, func(t *testing.T) {
			client := &scriptedChat{steps: steps}
			result, err := (Generator{Client: client}).Generate(context.Background(), prepared, English, SensitiveValues{})
			if err != nil || len(result.Plan.Commits) != 1 || !reflect.DeepEqual(result.Plan.Commits[0].FileIDs, []string{"F001", "F002"}) || result.Repaired != repair {
				t.Fatalf("result=%#v error=%v", result, err)
			}
			if !strings.Contains(client.messages[0][1].Content, "relation_context") {
				t.Fatal("initial request omitted relation context")
			}
			if repair && !strings.Contains(client.messages[1][1].Content, "relation_context") {
				t.Fatal("repair request omitted the original relation input")
			}
		})
	}
}

func preparedInputWithRelationContext(t *testing.T) contextinput.Prepared {
	t.Helper()
	document := planningDocument()
	document.RelationContext = &contextinput.RelationContext{
		Components: []relation.CandidateComponent{{ID: "C001", FileIDs: []string{"F001", "F002"}}},
		Edges:      []relation.Relation{{SourceID: "F001", TargetID: "F002", Kind: relation.DirectImport, Class: relation.Soft, Reason: "observed_import_path", Evidence: relation.Evidence{Type: "import_path", Value: "./asset"}}},
		Statistics: relation.GraphStatistics{NodeCount: 2, EdgeCount: 1, ComponentCount: 1},
	}
	prepared, err := contextinput.Prepare(context.Background(), document, contextinput.BudgetConfig{Context: "8k", MaxContextTokens: contextinput.Context32K}, Renderer(English), nil)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}
