package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

func TestAttachRelationContextFallbackStatusReachesPlanningPrompt(t *testing.T) {
	sourcePath, testPath, readmePath := "src/auth.ts", "src/auth.test.ts", "README.md"
	source := gitstate.Change{ID: "F001", Status: "modified", NewPath: &sourcePath, Language: "typescript"}
	testFile := gitstate.Change{ID: "F002", Status: "modified", NewPath: &testPath, Language: "typescript"}
	readme := gitstate.Change{ID: "F001", Status: "modified", NewPath: &readmePath, Language: "markdown"}
	for _, test := range []struct {
		name              string
		changes           []gitstate.Change
		files             []relation.File
		wantReason        string
		wantObservations  int
		wantRelationEdges int
	}{
		{"extract unavailable", []gitstate.Change{source}, []relation.File{{Change: source}, {Change: source}}, "unavailable", 0, 0},
		{"graph unavailable", []gitstate.Change{source}, []relation.File{{Change: testFile}}, "unavailable", 1, 0},
		{"no graph evidence", []gitstate.Change{readme}, []relation.File{{Change: readme}}, "ineffective", 0, 0},
		{"diagnostic only graph", []gitstate.Change{testFile}, []relation.File{{Change: testFile}}, "", 0, 0},
		{"usable graph", []gitstate.Change{source, testFile}, []relation.File{{Change: source}, {Change: testFile}}, "", 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := contextinput.Document{SchemaVersion: contextinput.SchemaVersion}
			for _, change := range test.changes {
				document.Files = append(document.Files, contextinput.File{ID: change.ID})
			}
			attachRelationContext(&document, test.changes, test.files)
			prepared, err := contextinput.Prepare(context.Background(), document, contextinput.BudgetConfig{Context: "8k"}, planning.Renderer(planning.English), nil)
			if err != nil {
				t.Fatal(err)
			}
			var envelope struct {
				RepositoryInput contextinput.Document `json:"repository_input"`
			}
			if err := json.Unmarshal(prepared.Prompt, &envelope); err != nil {
				t.Fatal(err)
			}
			input := envelope.RepositoryInput
			if test.wantReason != "" {
				if !prepared.RelationContextOmitted || input.RelationContext != nil || input.RelationContextStatus == nil || !input.RelationContextStatus.Omitted || input.RelationContextStatus.Reason != test.wantReason || input.RelationContextStatus.ObservationCount != test.wantObservations {
					t.Fatalf("fallback prompt=%s", prepared.Prompt)
				}
				if test.wantObservations > 0 && input.RelationContextStatus.ObservationsByOutcome[relation.Unresolved] != test.wantObservations {
					t.Fatalf("diagnostic counts missing: %s", prepared.Prompt)
				}
				if test.name == "extract unavailable" && strings.Contains(string(prepared.Prompt), "\"observation_count\"") {
					t.Fatalf("unknown observation count was reported as zero: %s", prepared.Prompt)
				}
				return
			}
			if prepared.RelationContextOmitted || input.RelationContextStatus != nil || input.RelationContext == nil || len(input.RelationContext.Edges) != test.wantRelationEdges {
				t.Fatalf("usable graph missing: %s", prepared.Prompt)
			}
		})
	}
}

func TestMLXPlanningFailsClosedWithoutModelOrOllamaFallback(t *testing.T) {
	oldStore := newMLXModelStore
	t.Cleanup(func() { newMLXModelStore = oldStore })
	newMLXModelStore = func() (mlxmodel.Store, error) {
		return mlxmodel.Store{Root: t.TempDir()}, nil
	}
	values := config.Defaults().Values
	values.Backend = "mlx"
	values.Model = "owner/model"
	values.ModelRevision = strings.Repeat("a", 40)
	values.ModelQuantization = "4bit"
	_, err := generateCommitPlan(context.Background(), "", gitstate.Snapshot{}, values, "")
	if err == nil || !strings.Contains(err.Error(), "run commiter setup") {
		t.Fatalf("missing model error = %v", err)
	}
}

func TestMLXPlanningUsesInstalledModelAndHelper(t *testing.T) {
	repo := cliRepository(t)
	cliWrite(t, repo, "src/main.ts", "import { helper } from './helper'\nexport const value = helper()\n", 0o644)
	cliWrite(t, repo, "src/helper.ts", "export function helper() { return 1 }\n", 0o644)
	cliGit(t, repo, "add", "src/main.ts", "src/helper.ts")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "src/main.ts", "import { helper } from './helper'\nexport const value = helper() + 1\n", 0o644)
	cliWrite(t, repo, "src/helper.ts", "export function helper() { return 2 }\n", 0o644)
	snapshot, err := gitstate.Collect(repo, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}

	values := config.Defaults().Values
	values.Backend = "mlx"
	values.Model = "owner/model"
	values.ModelRevision = strings.Repeat("a", 40)
	values.ModelQuantization = "4bit"
	spec := mlxmodel.Spec{Repo: values.Model, Revision: values.ModelRevision, Quantization: values.ModelQuantization}
	store := mlxmodel.Store{Root: t.TempDir()}
	modelPath, err := store.Destination(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(modelPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelPath, "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(mlxmodel.Plan{Spec: spec, Files: []mlxmodel.File{{Path: "config.json", Size: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelPath, "commiter-model.json"), metadata, 0o600); err != nil {
		t.Fatal(err)
	}
	oldStore := newMLXModelStore
	t.Cleanup(func() { newMLXModelStore = oldStore })
	newMLXModelStore = func() (mlxmodel.Store, error) { return store, nil }
	oldHelperPath := mlxHelperPath
	t.Cleanup(func() { mlxHelperPath = oldHelperPath })

	helperDir := t.TempDir()
	helper := filepath.Join(helperDir, "commiter-mlx-helper")
	script := `#!/bin/sh
set -eu
cat > "$MLX_TEST_REQUEST"
printf '%s\n' '{"ok":true,"stop_reason":"completed","generated_json":"{\"commits\":[{\"type\":\"fix\",\"scope\":\"planner\",\"breaking\":false,\"summary\":\"update planning\",\"file_ids\":[\"F001\",\"F002\"]}]}","model":"owner/model","runtime":"mlx"}'
`
	if err := os.WriteFile(helper, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	mlxHelperPath = func() (string, error) { return helper, nil }
	requestPath := filepath.Join(t.TempDir(), "request.json")
	t.Setenv("MLX_TEST_REQUEST", requestPath)

	plan, err := generateCommitPlan(context.Background(), repo, snapshot, values, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Commits) != 1 || plan.Commits[0].Summary != "update planning" || len(plan.Commits[0].FileIDs) != 2 {
		t.Fatalf("plan = %#v", plan)
	}
	requestData, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var request mlx.Request
	if err := json.Unmarshal(requestData, &request); err != nil {
		t.Fatal(err)
	}
	if request.Model != values.Model || request.ModelPath != modelPath || len(request.Schema) == 0 || len(request.Messages) != 2 {
		t.Fatalf("helper request model=%q path=%q schema_bytes=%d messages=%d", request.Model, request.ModelPath, len(request.Schema), len(request.Messages))
	}
	if !strings.Contains(request.Messages[1].Content, "relation_context") || !strings.Contains(request.Messages[1].Content, "candidate_components") || !strings.Contains(request.Messages[1].Content, "observed_import_path") {
		t.Fatalf("planning request did not include extracted relation context: %s", request.Messages[1].Content)
	}
}

func TestAnalyzeForPlanningStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, _, err := analyzeForPlanningWithStatsContext(ctx, t.TempDir(), gitstate.Snapshot{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestAnalyzeForPlanningExtractsValuesOnlyFromApprovedSensitiveCandidates(t *testing.T) {
	repo := cliRepository(t)
	cliWrite(t, repo, "normal.txt", "base\n", 0o644)
	cliWrite(t, repo, "auth.json", "{}\n", 0o600)
	cliGit(t, repo, "add", "normal.txt", "auth.json")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "normal.txt", "token = false\n", 0o644)
	cliWrite(t, repo, "auth.json", `{"token":"approved-fixture-value"}`+"\n", 0o600)
	snapshot, err := gitstate.Collect(repo, gitstate.Options{
		ApproveSensitiveCandidates: func([]gitstate.Candidate) (bool, error) {
			return true, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, sensitive, err := analyzeForPlanning(repo, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if sensitive.Contains([]byte("false")) {
		t.Fatal("normal file value was treated as an approved sensitive value")
	}
	if !sensitive.Contains([]byte("approved-fixture-value")) {
		t.Fatal("approved sensitive candidate value was not extracted")
	}
}

func TestPlanningStatsDoNotInventSyntaxFallbackForMetadataOnlyFiles(t *testing.T) {
	repo := cliRepository(t)
	cliWrite(t, repo, "main.go", "package main\nfunc main() {}\n", 0o644)
	cliWrite(t, repo, "asset.bin", "\x00base", 0o644)
	cliGit(t, repo, "add", "main.go", "asset.bin")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "main.go", "package main\nfunc main() { println(1) }\n", 0o644)
	cliWrite(t, repo, "asset.bin", "\x00changed", 0o644)
	snapshot, err := gitstate.Collect(repo, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, _, stats, err := analyzeForPlanningWithStats(repo, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if stats.syntaxSuccess != 1 || stats.syntaxFallback != 0 || stats.lines != 2 || stats.bytes == 0 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestWorktreeDiffTreatsCollectedPathsLiterally(t *testing.T) {
	repo := cliRepository(t)
	cliWrite(t, repo, "*", "literal base\n", 0o644)
	cliWrite(t, repo, "outside.txt", "outside base\n", 0o644)
	cliGit(t, repo, "add", "*", "outside.txt")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "*", "literal change\n", 0o644)
	cliWrite(t, repo, "outside.txt", "outside change\n", 0o644)

	diff, err := worktreeDiff(repo, []string{"*"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "literal change") || strings.Contains(diff, "outside change") {
		t.Fatalf("worktree diff expanded collected path as pathspec: %q", diff)
	}
}
