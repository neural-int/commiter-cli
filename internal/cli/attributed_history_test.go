package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/historyattribution"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
)

// Subprocess fixtures exchange the actual App Server JSON-RPC and MLX helper
// protocols. They never contact a real server or load a model.
func TestETLAppServerProcess(t *testing.T) {
	mode := os.Getenv("COMMITER_ETL_FIXTURE")
	if mode == "" {
		return
	}
	dec, enc := json.NewDecoder(os.Stdin), json.NewEncoder(os.Stdout)
	event := func() {
		f, _ := os.OpenFile(os.Getenv("COMMITER_ETL_SERVER_EVENTS"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if f != nil {
			f.WriteString("initialize\n")
			f.Close()
		}
	}
	for {
		var request struct {
			ID     any            `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if dec.Decode(&request) != nil {
			os.Exit(0)
		}
		if request.Method == "initialized" {
			continue
		}
		var result any
		switch request.Method {
		case "initialize":
			event()
			if mode == "unavailable" {
				enc.Encode(map[string]any{"id": request.ID, "error": map[string]any{"code": -1, "message": "PRIVATE_MUST_NOT_LEAK"}})
				continue
			}
			if mode == "mutation" {
				os.WriteFile(filepath.Join(os.Getenv("COMMITER_ETL_ROOT"), "sample.go"), []byte("package sample\nfunc value() int { return 3 }\n"), 0600)
			}
			result = map[string]any{}
		case "thread/list":
			roots, ok := request.Params["cwd"].([]any)
			if !ok || len(roots) != 1 || roots[0] != os.Getenv("COMMITER_ETL_ROOT") || request.Params["limit"] != float64(100) {
				os.Exit(4)
			}
			rows := []any{}
			if request.Params["archived"] != true {
				rows = append(rows, map[string]any{"id": "source-thread-ETL", "cwd": os.Getenv("COMMITER_ETL_ROOT")})
			}
			result = map[string]any{"data": rows}
		case "thread/turns/list":
			if request.Params["limit"] != float64(100) || request.Params["itemsView"] != "notLoaded" {
				os.Exit(5)
			}
			unix, _ := strconv.ParseInt(os.Getenv("COMMITER_ETL_TIME"), 10, 64)
			result = map[string]any{"data": []any{map[string]any{"id": "source-turn-ETL", "startedAt": unix, "status": "completed"}}}
		case "thread/items/list":
			if request.Params["limit"] != float64(100) {
				os.Exit(6)
			}
			wrap := func(item map[string]any) any { return map[string]any{"turnId": "source-turn-ETL", "item": item} }
			change := func(id string) map[string]any {
				return map[string]any{"id": id, "type": "fileChange", "status": "completed", "changes": []any{map[string]any{"path": "sample.go", "kind": map[string]any{"type": "update"}, "diff": "@@ -2 +2 @@\n-func value() int { return 1 }\n+func value() int { return 2 }\n"}}}
			}
			edit := change("source-edit-ETL")
			rows := []any{wrap(map[string]any{"id": "source-user-ETL", "type": "userMessage", "content": []any{map[string]any{"type": "text", "text": "Change the return value to two"}}}), wrap(edit)}
			if mode == "duplicate" {
				rows = append(rows, wrap(edit))
			}
			if mode == "ambiguous" {
				rows = append(rows, wrap(change("source-other-edit-ETL")))
			}
			rows = append(rows, wrap(map[string]any{"id": "source-final-ETL", "type": "agentMessage", "phase": "final_answer", "text": "Changed the return value to two"}))
			result = map[string]any{"data": rows}
		default:
			os.Exit(7)
		}
		if enc.Encode(map[string]any{"id": request.ID, "result": result}) != nil {
			os.Exit(8)
		}
	}
}

func TestETLHelperProcess(t *testing.T) {
	mode := os.Getenv("COMMITER_ETL_FIXTURE")
	if mode == "" {
		return
	}
	var request mlx.Request
	if json.NewDecoder(os.Stdin).Decode(&request) != nil {
		os.Exit(10)
	}
	if len(request.Messages) != 2 {
		os.Exit(11)
	}
	var payload map[string]json.RawMessage
	if json.Unmarshal([]byte(request.Messages[1].Content), &payload) != nil {
		os.Exit(12)
	}
	phase := request.GenerationProfile
	var files []struct {
		ID     string `json:"file_id"`
		Blocks []struct {
			Status     string `json:"agent_history_match_status"`
			Candidates []struct {
				Ref string `json:"context_ref"`
			} `json:"agent_history_candidates"`
		} `json:"change_blocks"`
	}
	if phase == "bounded-grouping" {
		json.Unmarshal(payload["files"], &files)
	} else {
		var groups []struct {
			ID       string          `json:"group_id"`
			Files    json.RawMessage `json:"files"`
			Type     *string         `json:"type"`
			Breaking *bool           `json:"breaking"`
		}
		if json.Unmarshal(payload["groups"], &groups) != nil || len(groups) != 1 || groups[0].ID != "G001" {
			os.Exit(13)
		}
		if json.Unmarshal(groups[0].Files, &files) != nil {
			os.Exit(14)
		}
		if payload["agent_history_acquisition"] != nil || payload["agent_history_projection"] != nil || payload["codex_context"] != nil {
			os.Exit(15)
		}
		if phase == "bounded-text" && (groups[0].Type == nil || *groups[0].Type != "fix" || groups[0].Breaking == nil || *groups[0].Breaking) {
			os.Exit(16)
		}
	}
	if len(files) != 1 || files[0].ID != "F1" || len(files[0].Blocks) != 1 {
		os.Exit(17)
	}
	want := "matched"
	if mode == "ambiguous" {
		want = "ambiguous"
	}
	if mode == "unavailable" {
		want = "error"
	}
	if files[0].Blocks[0].Status != want {
		os.Exit(18)
	}
	var contexts []struct {
		Ref      string   `json:"context_ref"`
		Requests []string `json:"user_requests"`
		Finals   []string `json:"final_responses"`
	}
	if json.Unmarshal(payload["agent_history_contexts"], &contexts) != nil {
		os.Exit(19)
	}
	if mode == "unavailable" {
		if len(contexts) != 0 {
			os.Exit(20)
		}
	} else {
		if len(contexts) != 1 || contexts[0].Ref != "H1" || len(contexts[0].Requests) != 1 || len(contexts[0].Finals) != 1 || len(files[0].Blocks[0].Candidates) != 1 || files[0].Blocks[0].Candidates[0].Ref != "H1" {
			os.Exit(21)
		}
	}
	for _, private := range []string{"source-thread-ETL", "source-turn-ETL", "source-edit-ETL", "source-user-ETL", "PRIVATE_MUST_NOT_LEAK"} {
		if strings.Contains(request.Messages[1].Content, private) {
			os.Exit(22)
		}
	}
	data, _ := json.Marshal(request)
	if os.WriteFile(filepath.Join(os.Getenv("COMMITER_ETL_HELPER_EVENTS"), phase+".json"), data, 0600) != nil {
		os.Exit(23)
	}
	generated := `{"G001":{"scope":"sample","summary":"correct return value"}}`
	if phase == "bounded-grouping" {
		generated = `{"F1":"G001"}`
	} else if phase == "bounded-category" {
		generated = `{"G001":{"type":"fix","breaking_evidence_ref":"none"}}`
	}
	json.NewEncoder(os.Stdout).Encode(mlx.Response{OK: true, StopReason: mlx.StopReasonCompleted, GeneratedJSON: generated, GenerationProfile: phase})
	os.Exit(0)
}

func TestCLIHistoryETLToHelper(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("production memory sampler is macOS-specific; portable RPC pipeline tests use an injected sampler")
	}
	for _, mode := range []string{"matched", "duplicate", "ambiguous", "unavailable", "mutation"} {
		t.Run(mode, func(t *testing.T) {
			root := cliRepository(t)
			cliWrite(t, root, "sample.go", "package sample\nfunc value() int { return 1 }\n", 0600)
			cliGit(t, root, "add", "sample.go")
			cliGit(t, root, "commit", "-m", "base")
			cliWrite(t, root, "sample.go", "package sample\nfunc value() int { return 2 }\n", 0600)
			canonical, err := filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatal(err)
			}
			root = canonical
			snapshot, err := gitstate.Collect(root, gitstate.Options{})
			if err != nil {
				t.Fatal(err)
			}
			initialHead, initialIndex := snapshot.Head, snapshot.IndexIdentity
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			quoted := "'" + strings.ReplaceAll(binary, "'", "'\\''") + "'"
			dir := t.TempDir()
			helper := filepath.Join(dir, "helper")
			for name, entry := range map[string]string{"codex": "TestETLAppServerProcess", "helper": "TestETLHelperProcess"} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexec "+quoted+" -test.run=^"+entry+"$ -- \"$@\"\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			helperEvents := t.TempDir()
			serverEvents := filepath.Join(t.TempDir(), "server-events")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("COMMITER_ETL_FIXTURE", mode)
			t.Setenv("COMMITER_ETL_ROOT", root)
			t.Setenv("COMMITER_ETL_TIME", strconv.FormatInt(time.Now().Unix(), 10))
			t.Setenv("COMMITER_ETL_HELPER_EVENTS", helperEvents)
			t.Setenv("COMMITER_ETL_SERVER_EVENTS", serverEvents)
			values := config.Defaults().Values
			store := mlxmodel.Store{Root: t.TempDir()}
			spec := mlxmodel.Spec{Repo: values.Model, Revision: values.ModelRevision, Quantization: values.ModelQuantization}
			modelPath, err := store.Destination(spec)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(modelPath, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(modelPath, "config.json"), []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			metadata, _ := json.Marshal(mlxmodel.Plan{Spec: spec, Files: []mlxmodel.File{{Path: "config.json", Size: 2}}})
			if err = os.WriteFile(filepath.Join(modelPath, "commiter-model.json"), metadata, 0600); err != nil {
				t.Fatal(err)
			}
			oldStore, oldHelper, oldCollect := newMLXModelStore, mlxHelperPath, collectAttributedHistory
			t.Cleanup(func() { newMLXModelStore, mlxHelperPath, collectAttributedHistory = oldStore, oldHelper, oldCollect })
			newMLXModelStore = func() (mlxmodel.Store, error) { return store, nil }
			mlxHelperPath = func() (string, error) { return helper, nil }
			collectAttributedHistory = historyattribution.Collect
			cycle := &codexCycle{started: time.Now()}
			ctx := context.WithValue(context.Background(), codexCycleKey{}, cycle)
			plan, err := generateCommitPlan(ctx, root, snapshot, values, "")
			events, _ := os.ReadDir(helperEvents)
			if mode == "mutation" {
				if err == nil || !strings.Contains(err.Error(), "snapshot_changed") || len(events) != 0 {
					t.Fatalf("mutation escaped: err=%v calls=%d", err, len(events))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 3 || len(plan.Commits) != 1 || len(plan.Commits[0].FileIDs) != 1 || plan.Commits[0].FileIDs[0] != snapshot.Changes[0].ID {
				t.Fatal("ETL did not complete all three helper phases")
			}
			if mode == "matched" && len(cycle.attribution.ContextSources["H1"].Edits) != 1 {
				t.Fatal("host source mapping lost")
			}
			if mode == "ambiguous" && len(cycle.attribution.ContextSources["H1"].Edits) != 2 {
				t.Fatal("different edits in one Turn collapsed")
			}
			before, _ := os.ReadFile(serverEvents)
			if _, err = generateCommitPlan(ctx, root, snapshot, values, ""); err != nil {
				t.Fatal(err)
			}
			after, _ := os.ReadFile(serverEvents)
			if string(before) != string(after) {
				t.Fatal("unchanged regeneration refetched history")
			}
			final, e := gitstate.Collect(root, gitstate.Options{})
			if e != nil || final.Head != initialHead || final.IndexIdentity != initialIndex || len(changedSnapshotPaths(snapshot, final)) != 0 {
				t.Fatal("ETL changed Git")
			}
		})
	}
}

func TestAttributedHistoryCacheAndSnapshotGuard(t *testing.T) {
	root := cliRepository(t)
	cliWrite(t, root, "one.txt", "old\n", 0600)
	cliGit(t, root, "add", "one.txt")
	cliGit(t, root, "commit", "-m", "base")
	cliWrite(t, root, "one.txt", "new\n", 0600)
	snapshot, err := gitstate.Collect(root, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}
	values := config.Defaults().Values
	results, sensitive, _, _, err := analyzeForPlanningWithStatsAndRelationsContext(context.Background(), root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	document, err := contextinput.Build(snapshot, results)
	if err != nil {
		t.Fatal(err)
	}
	document, err = contextinput.RawDiffProjection(document)
	if err != nil {
		t.Fatal(err)
	}
	old := collectAttributedHistory
	t.Cleanup(func() { collectAttributedHistory = old })
	calls := 0
	collectAttributedHistory = func(ctx context.Context, request historyattribution.Request) (historyattribution.Result, error) {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 5*time.Second {
			t.Fatal("unbounded history acquisition")
		}
		if request.AllowPath(".env") || request.SafeText("password=private-value") || request.SafeText("Bearer private-value") {
			t.Fatal("history safety policy bypass")
		}
		if len(request.Files) != 1 || request.Files[0].ID != snapshot.Changes[0].ID {
			t.Fatal("internal snapshot IDs lost before attribution")
		}
		return historyattribution.Result{Status: "unavailable"}, nil
	}
	ctx := context.WithValue(context.Background(), codexCycleKey{}, &codexCycle{started: time.Now()})
	for i := 0; i < 2; i++ {
		if _, err := attributedHistoryForPlan(ctx, root, snapshot, values, document, sensitive); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatal("same fixed input was re-extracted")
	}
	values.SensitivePatterns = []string{"private/*"}
	if _, err := attributedHistoryForPlan(ctx, root, snapshot, values, document, sensitive); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("different safety policy reused cache")
	}
	cliGit(t, root, "checkout", "-b", "changed-branch")
	if _, err := attributedHistoryForPlan(ctx, root, snapshot, values, document, sensitive); err == nil || !strings.Contains(err.Error(), "snapshot_changed") {
		t.Fatal("branch change was not rejected")
	}
	if calls != 2 {
		t.Fatal("changed snapshot reached extraction")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := attributedHistoryForPlan(canceled, root, snapshot, values, document, sensitive); err == nil {
		t.Fatal("canceled request reached extraction")
	}
}
