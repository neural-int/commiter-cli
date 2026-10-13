package cli

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type groupingFlowClient struct{ messages [][]llm.Message }

func (c *groupingFlowClient) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected legacy call")
}
func (c *groupingFlowClient) ChatWithOptions(_ context.Context, m []llm.Message, _ json.RawMessage, o llm.Options) (llm.Response, error) {
	c.messages = append(c.messages, m)
	output := `{"G001":{"scope":"sample","summary":"update value"}}`
	switch o.GenerationProfile {
	case "bounded-grouping":
		var input struct {
			Files []struct {
				ID string `json:"id"`
			} `json:"files"`
		}
		if err := json.Unmarshal([]byte(m[1].Content), &input); err != nil {
			return llm.Response{}, err
		}
		membership := map[string]string{}
		for _, f := range input.Files {
			membership[f.ID] = "G001"
		}
		data, _ := json.Marshal(membership)
		output = string(data)
	case "bounded-category":
		output = `{"G001":{"type":"fix","breaking_evidence_ref":"none"}}`
	}
	return llm.Response{Content: output, StopReason: "completed"}, nil
}

func TestGroupingContextConnectsGitChangesToThreePhase(t *testing.T) {
	for _, kind := range []string{"modified", "added", "deleted", "renamed", "split rename", "unsupported"} {
		t.Run(kind, func(t *testing.T) {
			before := "package sample\nfunc Value() int {\n return 1\n}\n"
			after := strings.Replace(before, "return 1", "return 2", 1)
			root := cliRepository(t)
			path := "sample.go"
			if kind == "unsupported" {
				path = "sample.txt"
				before = "first line\n"
				after = "second line\n"
			}
			cliWrite(t, root, path, before, 0o644)
			cliGit(t, root, "add", path)
			cliGit(t, root, "commit", "-m", "base")
			switch kind {
			case "deleted":
				cliGit(t, root, "rm", path)
			case "added":
				path = "added.go"
				cliWrite(t, root, path, after, 0o644)
			case "renamed", "split rename":
				cliGit(t, root, "mv", path, "renamed.go")
				path = "renamed.go"
				if kind == "split rename" {
					after = "package replacement\nfunc Other() string { return \"changed\" }\n"
				}
				cliWrite(t, root, path, after, 0o644)
			default:
				cliWrite(t, root, path, after, 0o644)
			}
			snapshot, err := gitstate.Collect(root, gitstate.Options{})
			if err != nil {
				t.Fatal(err)
			}
			results, sensitive, _, sources, err := analyzeForPlanningWithStatsAndRelationsContext(context.Background(), root, snapshot)
			if err != nil {
				t.Fatal(err)
			}
			document, err := contextinput.Build(snapshot, results)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "split rename" && strings.Count(document.Files[0].GroupingRawDiff, "diff --git ") != 2 {
				t.Fatalf("split rename not reproduced: %s", document.Files[0].GroupingRawDiff)
			}
			if err := attachGroupingSyntaxContext(context.Background(), root, &document, sources); err != nil {
				t.Fatal(err)
			}
			prepared, err := contextinput.Prepare(context.Background(), document, contextinput.BudgetConfig{Context: "16k"}, planning.Renderer(planning.English), nil)
			if err != nil {
				t.Fatal(err)
			}
			prepared.Budget.ContextTokens = contextinput.Context16K
			client := &groupingFlowClient{}
			result, err := (planning.ThreePhaseGenerator{Client: client}).Generate(context.Background(), prepared, planning.English, sensitive)
			if err != nil {
				t.Fatal(err)
			}
			if result.Calls != 3 || len(result.Plan.Commits) != 1 || len(result.Plan.Commits[0].FileIDs) != 1 || result.Plan.Commits[0].FileIDs[0] != snapshot.Changes[0].ID {
				t.Fatalf("bad plan: %+v", result)
			}
			var input struct {
				Files []struct {
					ID            string `json:"id"`
					Mode          string `json:"mode"`
					RawDiff       string `json:"raw_diff"`
					SyntaxContext []struct {
						HunkIndex            int    `json:"hunk_index"`
						EnclosingDeclaration string `json:"enclosing_declaration"`
					} `json:"syntax_context"`
				} `json:"files"`
			}
			if err := json.Unmarshal([]byte(client.messages[0][1].Content), &input); err != nil {
				t.Fatal(err)
			}
			if input.Files[0].RawDiff != document.Files[0].GroupingRawDiff {
				t.Fatal("grouping changed the complete raw diff")
			}
			if input.Files[0].RawDiff == "" || input.Files[0].Mode != "raw_diff" || input.Files[0].ID != "path:"+path {
				t.Fatalf("bad grouping input: %s", client.messages[0][1].Content)
			}
			if kind == "unsupported" || kind == "split rename" {
				if len(input.Files[0].SyntaxContext) != 0 {
					t.Fatal("unsupported context guessed")
				}
			} else if len(input.Files[0].SyntaxContext) != 1 || input.Files[0].SyntaxContext[0].EnclosingDeclaration != "Value" {
				t.Fatalf("context lost: %+v", input.Files[0])
			}
			final, err := gitstate.Collect(root, gitstate.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(snapshot, final) {
				t.Fatal("planning changed HEAD/index/worktree")
			}
		})
	}
}

func TestGroupingContextKeepsGitSourceErrors(t *testing.T) {
	root := cliRepository(t)
	before := "package sample\nfunc Value() int { return 1 }\n"
	cliWrite(t, root, "sample.go", before, 0o644)
	cliGit(t, root, "add", "sample.go")
	cliGit(t, root, "commit", "-m", "base")
	cliGit(t, root, "mv", "sample.go", "renamed.go")
	cliWrite(t, root, "renamed.go", "package replacement\nfunc Other() string { return \"changed\" }\n", 0o644)
	snapshot, err := gitstate.Collect(root, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}
	results, _, _, sources, err := analyzeForPlanningWithStatsAndRelationsContext(context.Background(), root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	document, err := contextinput.Build(snapshot, results)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(document.Files[0].GroupingRawDiff, "diff --git ") != 2 {
		t.Fatal("multiple diff sections not reproduced")
	}
	missingBlob := strings.Repeat("0", 40)
	sources[0].Change.HeadIdentity = &missingBlob
	if err := attachGroupingSyntaxContext(context.Background(), root, &document, sources); err == nil {
		t.Fatal("Git blob failure was ignored")
	}
}
