package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/historyattribution"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	runmetrics "github.com/natsuki0413/commiter-cli/internal/metrics"
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
	values.Model = config.CandidateModel
	values.ModelRevision = config.CandidateRevision
	values.ModelQuantization = "4bit"
	_, err := generateCommitPlan(context.Background(), "", gitstate.Snapshot{}, values, "")
	if err == nil || !strings.Contains(err.Error(), "run commiter setup") {
		t.Fatalf("missing model error = %v", err)
	}
}

func TestRemovedPlannerCannotStartInstalledHelper(t *testing.T) {
	oldHelper := mlxHelperPath
	oldStore := newMLXModelStore
	t.Cleanup(func() { mlxHelperPath = oldHelper; newMLXModelStore = oldStore })
	mlxHelperPath = func() (string, error) { t.Fatal("removed planner started helper"); return "", nil }
	newMLXModelStore = func() (mlxmodel.Store, error) {
		t.Fatal("removed planner opened model store")
		return mlxmodel.Store{}, nil
	}
	values := config.Defaults().Values
	values.Planner = "single-pass"
	if _, err := generateCommitPlan(context.Background(), "", gitstate.Snapshot{}, values, ""); err == nil || !strings.Contains(err.Error(), "migrate") {
		t.Fatalf("removed planner accepted: %v", err)
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

// The optional smoke path reuses an installed pin; it never downloads a model.
func TestGemmaThreePhaseDefaultPlanning(t *testing.T) {
	// Normal tests never read the user's Codex history. Exercise the real CLI
	// wiring with a synthetic record; preserve the opt-in model smoke baseline.
	oldCollect := collectAttributedHistory
	t.Cleanup(func() { collectAttributedHistory = oldCollect })
	collectAttributedHistory = func(ctx context.Context, request historyattribution.Request) (historyattribution.Result, error) {
		h := &planning.GroupingHistory{Contexts: []planning.AgentHistoryContext{{ContextRef: "H1", FinalResponses: []string{"Corrected the value"}}}, Acquisition: planning.AgentHistoryAcquisition{Status: "complete"}}
		for _, file := range request.Files {
			projected, err := planning.ProjectMetadataFile(file)
			if err != nil {
				return historyattribution.Result{}, err
			}
			attributed := planning.AttributedFileHistory{FileID: file.ID, RawDiff: projected.RawDiff}
			for _, block := range projected.ChangeBlocks {
				attributed.Blocks = append(attributed.Blocks, planning.AttributedBlockHistory{BlockID: block.BlockID, DiffRange: block.DiffRange, Status: "matched", Reasons: []string{}, Candidates: []planning.AgentHistoryCandidate{{ContextRef: "H1", Match: "exact"}}})
			}
			h.Files = append(h.Files, attributed)
		}
		return historyattribution.Result{History: h, Status: "available", Contexts: 1}, nil
	}
	repo := cliRepository(t)
	cliWrite(t, repo, "sample.go", "package sample\nfunc value() int { return 1 }\n", 0o644)
	cliGit(t, repo, "add", "sample.go")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "sample.go", "package sample\nfunc value() int { return 2 }\n", 0o644)
	snapshot, err := gitstate.Collect(repo, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}
	values := config.Defaults().Values
	store := mlxmodel.Store{Root: t.TempDir()}
	realHelper := os.Getenv("COMMITER_CANDIDATE_SMOKE_HELPER")
	if realHelper != "" {
		store.Root = os.Getenv("COMMITER_CANDIDATE_SMOKE_CACHE")
	} else {
		spec := mlxmodel.Spec{Repo: values.Model, Revision: values.ModelRevision, Quantization: values.ModelQuantization}
		path, err := store.Destination(spec)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "config.json"), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
		metadata, _ := json.Marshal(mlxmodel.Plan{Spec: spec, Files: []mlxmodel.File{{Path: "config.json", Size: 2}}})
		if err := os.WriteFile(filepath.Join(path, "commiter-model.json"), metadata, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	oldStore, oldHelper := newMLXModelStore, mlxHelperPath
	t.Cleanup(func() { newMLXModelStore = oldStore; mlxHelperPath = oldHelper })
	newMLXModelStore = func() (mlxmodel.Store, error) { return store, nil }
	if realHelper != "" {
		mlxHelperPath = func() (string, error) { return realHelper, nil }
	} else {
		helper := filepath.Join(t.TempDir(), "helper")
		script := `#!/bin/sh
set -eu
while IFS= read -r request; do
case "$request" in
 *'"context_tokens":65536'*) ;;
 *) exit 1 ;;
esac
case "$request" in
 *'"generation_profile":"bounded-grouping"'*) profile=bounded-grouping; result='{"F1":"G001"}' ;;
 *'"generation_profile":"bounded-category"'*) profile=bounded-category; result='{"G001":{"type":"fix","breaking_evidence_ref":"none"}}' ;;
 *'"generation_profile":"bounded-text"'*) profile=bounded-text; result='{"G001":{"scope":"sample","summary":"correct value"}}' ;;
 *) exit 1 ;;
esac
selection=''
case "$request" in *'"history_input":'*) selection=',"selected_history_indices":[0]' ;; esac
escaped=$(printf '%s' "$result" | sed 's/"/\\"/g')
printf '{"ok":true,"stop_reason":"completed","generation_profile":"%s","generated_json":"%s"%s}\n' "$profile" "$escaped" "$selection"
done
`
		if err := os.WriteFile(helper, []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
		mlxHelperPath = func() (string, error) { return helper, nil }
	}
	started := time.Now()
	recorder := runmetrics.New()
	plan, err := generateCommitPlan(runmetrics.WithRecorder(context.Background(), recorder), repo, snapshot, values, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Commits) != 1 || strings.Join(plan.Commits[0].FileIDs, ",") != "F001" {
		t.Fatalf("invalid assignment: %#v", plan)
	}
	if record := recorder.Finish("success"); record.Context != "64k" || record.ContextCeiling != "64k" {
		t.Fatalf("context metrics=%s ceiling=%s", record.Context, record.ContextCeiling)
	}
	if realHelper == "" {
		record := recorder.Finish("success")
		if record.CodexContextStatus != "available" || record.CodexContextRecords != 1 {
			t.Fatal("CLI did not acquire optional context")
		}
	}
	t.Logf("three-phase installed helper=%t elapsed=%s commits=%d", realHelper != "", time.Since(started).Round(time.Millisecond), len(plan.Commits))
	status := cliGitOutput(t, repo, "status", "--porcelain")
	if strings.TrimSpace(status) != "M sample.go" {
		t.Fatalf("planning changed Git state: %q", status)
	}
	helper, err := mlxHelperPath()
	if err != nil {
		t.Fatal(err)
	}
	modelPath, err := store.Ready(mlxmodel.Spec{Repo: values.Model, Revision: values.ModelRevision, Quantization: values.ModelQuantization})
	if err != nil {
		t.Fatal(err)
	}
	started = time.Now()
	if ok, err := mlxPlannerCapabilityProbe(context.Background(), helper, values, modelPath); err != nil || !ok {
		t.Fatalf("candidate setup/doctor probe: ok=%t error=%v", ok, err)
	}
	t.Logf("candidate setup/doctor probe elapsed=%s", time.Since(started).Round(time.Millisecond))
}

func TestRecordGeneratedContextWithoutBackendTiming(t *testing.T) {
	recorder := runmetrics.New()
	recorder.SetContextCeiling("64k")
	recordGeneratedTelemetry(recorder, planning.Result{Telemetry: planning.Telemetry{ContextTokens: contextinput.Context32K}})
	record := recorder.Finish("llm_failure")
	if record.Context != "32k" || record.ContextCeiling != "64k" || record.Durations.Generation != nil {
		t.Fatalf("context metrics=%+v", record)
	}
}
