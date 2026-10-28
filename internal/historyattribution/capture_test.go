package historyattribution

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The fake app server runs in a subprocess. No test reads real Codex history.
func TestHistoryFixture(t *testing.T) {
	mode := os.Getenv("POC_FIXTURE_MODE")
	if mode == "" {
		return
	}
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for {
		var req struct {
			ID     any            `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if dec.Decode(&req) != nil {
			os.Exit(0)
		}
		if req.Method == "initialized" {
			continue
		}
		if mode == "timeout" {
			time.Sleep(time.Minute)
		}
		if mode == "limit" {
			_, _ = io.WriteString(os.Stdout, strings.Repeat("x", int(receiveLimit)+10)+"\n")
			os.Exit(0)
		}
		if mode == req.Method || (mode == "partial" && req.Method == "thread/items/list" && req.Params["turnId"] == "T2") {
			enc.Encode(map[string]any{"id": req.ID, "error": map[string]any{"code": -1, "message": "PRIVATE_DIAGNOSTIC_MUST_NOT_ESCAPE"}})
			continue
		}
		if mode == "fetch_modes" || mode == "full_missing" {
			var done func()
			if req.Method == "thread/items/list" && os.Getenv("POC_RPC_EVENTS") != "" {
				done = fixtureRPCEventStart()
			}
			result, valid := fetchFixtureReply(req.Method, req.Params, os.Getenv("POC_EXPECT_FETCH"), mode)
			if done != nil {
				done()
			}
			if !valid {
				enc.Encode(map[string]any{"id": req.ID, "error": map[string]any{"code": -1, "message": "fixture contract violation"}})
			} else {
				enc.Encode(map[string]any{"id": req.ID, "result": result})
			}
			continue
		}
		var result any
		switch req.Method {
		case "initialize":
			if mode == "mutation" {
				os.WriteFile(filepath.Join(os.Getenv("POC_FIXTURE_ROOT"), "file.go"), []byte("C\n"), 0600)
			}
			result = map[string]any{}
			if mode == "large_valid" {
				result = map[string]any{"padding": strings.Repeat("a", int(receiveLimit)+100)}
			}
		case "thread/list":
			result = map[string]any{"data": []any{map[string]any{"id": "S", "cwd": os.Getenv("POC_FIXTURE_ROOT")}}}
		case "thread/turns/list":
			ts := []any{map[string]any{"id": "T1", "startedAt": fixtureTurnTime(), "status": "completed"}}
			if mode == "partial" {
				ts = append(ts, map[string]any{"id": "T2", "startedAt": fixtureTurnTime(), "status": "completed"})
			}
			result = map[string]any{"data": ts}
		case "thread/items/list":
			item := func(id string, v map[string]any) any { v["id"] = id; return map[string]any{"turnId": "T1", "item": v} }
			entries := []any{item("U", map[string]any{"type": "userMessage", "content": []any{map[string]any{"type": "text", "text": "Change A to B"}}}), item("F", map[string]any{"type": "fileChange", "status": "completed", "changes": []any{map[string]any{"path": "file.go", "kind": map[string]any{"type": "update"}, "diff": update("A", "B")}}}), item("A", map[string]any{"type": "agentMessage", "phase": "final_answer", "text": "Changed A to B"})}
			if mode == "out_of_scope" {
				entries = append(entries, item("outside", map[string]any{"type": "fileChange", "status": "completed", "changes": []any{map[string]any{"path": "/unrelated/repository/other.go", "kind": map[string]any{"type": "update"}, "diff": update("C", "D")}}}))
			}
			result = map[string]any{"data": entries}
		}
		if enc.Encode(map[string]any{"id": req.ID, "result": result}) != nil {
			os.Exit(0)
		}
	}
}
func fixtureServer(t *testing.T, root, mode string) {
	t.Helper()
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nexec '" + strings.ReplaceAll(exe, "'", "'\\''") + "' -test.run=TestHistoryFixture -- \"$@\"\n"
	if os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0700) != nil {
		t.Fatal("wrapper")
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("POC_FIXTURE_MODE", mode)
	t.Setenv("POC_FIXTURE_ROOT", root)
	t.Setenv("POC_FIXTURE_TURN_TIME", strconv.FormatInt(time.Now().Unix(), 10))
}
func gitFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fixture Git failed: %s", out)
		}
	}
	run("init", "-q")
	os.WriteFile(filepath.Join(root, "file.go"), []byte("A\n"), 0600)
	run("add", "--", "file.go")
	run("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "fixture")
	os.WriteFile(filepath.Join(root, "file.go"), []byte("B\n"), 0600)
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}
func TestCaptureAndCompareFixtureCycle(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "ok")
	out := filepath.Join(t.TempDir(), "capture.json")
	if err := execute([]string{"capture", "--root", root, "--path", "file.go", "--out", out}); err != nil {
		t.Fatal(err)
	}
	in, err := loadInput(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(in.History.Operations) != 1 || in.History.Status != "complete" {
		t.Fatal(in.History)
	}
	if len(in.History.Turns) != 1 || in.History.Turns[0].Request == "" || in.History.Turns[0].Final == "" {
		t.Fatal("missing turn context")
	}
	result := compare(in)
	if len(result.Results) != 1 || result.Results[0].Status != "matched" {
		t.Fatal(result)
	}
	info, _ := os.Stat(out)
	if info.Mode().Perm() != 0600 {
		t.Fatal("private artifact mode")
	}
	if execute([]string{"capture", "--root", root, "--path", "file.go", "--out", out}) == nil {
		t.Fatal("overwrite allowed")
	}
	report := filepath.Join(t.TempDir(), "comparison.json")
	if err := execute([]string{"compare", "--input", out, "--out", report}); err != nil {
		t.Fatal(err)
	}
}
func TestHistoryErrorsHaveStageAndRemainOptional(t *testing.T) {
	for mode, stage := range map[string]string{"initialize": "history.initialize", "thread/list": "history.threads", "thread/turns/list": "history.turns", "thread/items/list": "history.items", "partial": "history.items"} {
		t.Run(mode, func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, mode)
			in, err := capture(root, []string{"file.go"}, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if in.History.Error == nil || in.History.Error.Stage != stage || in.History.Error.Code != "rpc_error" {
				t.Fatal(in.History)
			}
			if len(in.Files) != 1 || !compare(in).Fallback {
				t.Fatal("Git input lost on optional history failure")
			}
			if mode == "partial" {
				if len(in.History.Operations) != 1 || compare(in).Results[0].Status != "ambiguous" {
					t.Fatal("partial history promoted to matched")
				}
			}
		})
	}
}
func TestHistoryTimeoutAndReceiveLimit(t *testing.T) {
	for _, mode := range []string{"timeout", "limit"} {
		t.Run(mode, func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, mode)
			duration := 150 * time.Millisecond
			if mode == "limit" {
				duration = 2 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), duration)
			defer cancel()
			h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
			want := "deadline_exceeded"
			if mode == "limit" {
				want = "receive_limit"
			}
			if h.Error == nil || h.Error.Code != want {
				t.Fatal(h)
			}
			b, _ := json.Marshal(h)
			if strings.Contains(string(b), "PRIVATE_DIAGNOSTIC") {
				t.Fatal("raw error leaked")
			}
		})
	}
}
func TestGitAndOutputFailuresAreDistinguishable(t *testing.T) {
	root := gitFixture(t)
	if _, err := capture(root, []string{".env"}, time.Now()); err == nil || err.(*StageError).Stage != "git.paths" {
		t.Fatal(err)
	}
	if _, err := outputOutside(root, filepath.Join(root, "result.json")); err == nil || err.(*StageError).Stage != "output.path" {
		t.Fatal(err)
	}
	if _, err := capture(t.TempDir(), []string{"file.go"}, time.Now()); err == nil || err.(*StageError).Stage != "git.root" {
		t.Fatal(err)
	}
	if _, err := capture(root, []string{"missing.go"}, time.Now()); err == nil || err.(*StageError).Code != "no_selected_changes" {
		t.Fatal(err)
	}
	external := t.TempDir()
	os.WriteFile(filepath.Join(external, "file.go"), []byte("secret"), 0600)
	if err := os.Symlink(external, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTarget(root, "linked/file.go"); err == nil {
		t.Fatal("symlink escaped")
	}
	if _, err := loadInput(filepath.Join(t.TempDir(), "missing.json")); err == nil || err.(*StageError).Stage != "input.read" {
		t.Fatal(err)
	}
}
func TestUntrackedAndStagedPlusUnstagedFinalDiff(t *testing.T) {
	root := gitFixture(t)
	ctx := context.Background()
	if err := exec.Command("git", "-C", root, "add", "--", "file.go").Run(); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "file.go"), []byte("C\n"), 0600)
	os.WriteFile(filepath.Join(root, "new.go"), []byte("new\n"), 0600)
	s, err := readGit(ctx, root, []string{"file.go", "new.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.files) != 2 || !strings.Contains(s.files[0].Diff, "-A\n+C") || s.files[1].Kind != "untracked" {
		t.Fatal(s.files)
	}
}

func TestMutationFailsSnapshotAndPreservesStage(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "mutation")
	_, err := capture(root, []string{"file.go"}, time.Now())
	if err == nil || err.(*StageError).Stage != "git.verify" || err.(*StageError).Code != "snapshot_changed" {
		t.Fatal(err)
	}
}
func TestOptionalHistoryFailurePersistsFixedGitInput(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "thread/items/list")
	out := filepath.Join(t.TempDir(), "input.json")
	err := execute([]string{"capture", "--root", root, "--path", "file.go", "--out", out})
	if err == nil || err.(*StageError).Stage != "history.items" {
		t.Fatal(err)
	}
	in, e := loadInput(out)
	if e != nil || len(in.Files) != 1 || in.History.Error == nil {
		t.Fatal(e)
	}
	if !compare(in).Fallback {
		t.Fatal("fallback missing")
	}
}

func fixtureTurnTime() int64 {
	n, _ := strconv.ParseInt(os.Getenv("POC_FIXTURE_TURN_TIME"), 10, 64)
	return n
}

func TestSelectedRenamePairIsNotSilentlyContentMatched(t *testing.T) {
	root := gitFixture(t)
	os.WriteFile(filepath.Join(root, "file.go"), []byte("A\n"), 0600)
	if err := exec.Command("git", "-C", root, "mv", "file.go", "moved.go").Run(); err != nil {
		t.Fatal(err)
	}
	s, err := readGit(context.Background(), root, []string{"file.go", "moved.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.files) != 1 || s.files[0].OldPath != "file.go" || s.files[0].Path != "moved.go" {
		t.Fatal(s.files)
	}
	for _, f := range s.files {
		if f.Kind != "rename" {
			t.Fatal(f)
		}
	}
}
