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
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
)

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
	cliWrite(t, repo, "a.txt", "base\n", 0o644)
	cliGit(t, repo, "add", "a.txt")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "a.txt", "changed\n", 0o644)
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
printf '%s\n' '{"ok":true,"stop_reason":"completed","generated_json":"{\"commits\":[{\"type\":\"fix\",\"scope\":\"planner\",\"breaking\":false,\"summary\":\"update planning\",\"file_ids\":[\"F001\"]}]}","model":"owner/model","runtime":"mlx"}'
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
	if len(plan.Commits) != 1 || plan.Commits[0].Summary != "update planning" {
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
