package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/historyattribution"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

// This subprocess receives the real helper wire request but never loads a model.
// Fixed replies exist only to let the host construct the subsequent phase input.
func TestPreInferenceReceiverProcess(t *testing.T) {
	if os.Getenv("COMMITER_LIVE_ETL_RECEIVER") != "1" {
		return
	}
	var request mlx.Request
	if json.NewDecoder(os.Stdin).Decode(&request) != nil || len(request.Messages) != 2 {
		os.Exit(30)
	}
	var payload struct {
		Files []struct {
			ID string `json:"file_id"`
		} `json:"files"`
	}
	if json.Unmarshal([]byte(request.Messages[1].Content), &payload) != nil {
		os.Exit(31)
	}
	data, err := json.Marshal(request)
	if err != nil || os.WriteFile(filepath.Join(os.Getenv("COMMITER_LIVE_ETL_OUT"), request.GenerationProfile+".json"), data, 0600) != nil {
		os.Exit(32)
	}
	var reply any
	switch request.GenerationProfile {
	case "bounded-grouping":
		membership := map[string]string{}
		for _, file := range payload.Files {
			membership[file.ID] = "G001"
		}
		reply = membership
	case "bounded-category":
		reply = map[string]any{"G001": map[string]any{"type": "fix", "breaking_evidence_ref": "none"}}
	case "bounded-text":
		reply = map[string]any{"G001": map[string]any{"scope": "etl", "summary": "verify history transport"}}
	default:
		os.Exit(33)
	}
	generated, _ := json.Marshal(reply)
	json.NewEncoder(os.Stdout).Encode(mlx.Response{OK: true, StopReason: mlx.StopReasonCompleted, GeneratedJSON: string(generated), GenerationProfile: request.GenerationProfile})
	os.Exit(0)
}

// Opt-in only: reads the user's approved live App Server history; all raw
// requests remain in a private directory outside the repository.
func TestLiveHistoryPreInferenceETL(t *testing.T) {
	root := os.Getenv("COMMITER_LIVE_ETL_ROOT")
	if root == "" {
		t.Skip("explicit live history root required")
	}
	out := os.Getenv("COMMITER_LIVE_ETL_OUT")
	if out == "" {
		t.Fatal("private output required")
	}
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	outAbsolute, err := filepath.Abs(out)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(rootAbsolute, outAbsolute)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))) {
		t.Fatal("capture directory must be outside the target repository")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	paths := strings.Split(os.Getenv("COMMITER_LIVE_ETL_PATHS"), ",")
	if len(paths) == 0 || len(paths) > 4 || paths[0] == "" {
		t.Fatal("one to four explicit paths required")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := gitstate.Collect(root, gitstate.Options{Pathspecs: paths})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Changes) == 0 || len(snapshot.Changes) > 4 {
		t.Fatal("expected one to four real changes")
	}
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
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(t.TempDir(), "capture-helper")
	quoted := "'" + strings.ReplaceAll(binary, "'", "'\\''") + "'"
	if err = os.WriteFile(helper, []byte("#!/bin/sh\nexec "+quoted+" -test.run=^TestPreInferenceReceiverProcess$\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMMITER_LIVE_ETL_RECEIVER", "1")
	oldStore, oldHelper, oldCollect := newMLXModelStore, mlxHelperPath, collectAttributedHistory
	t.Cleanup(func() { newMLXModelStore, mlxHelperPath, collectAttributedHistory = oldStore, oldHelper, oldCollect })
	newMLXModelStore = func() (mlxmodel.Store, error) { return store, nil }
	mlxHelperPath = func() (string, error) { return helper, nil }
	var acquisitionSeconds float64
	var acquisitionDeadlineExceeded bool
	collectAttributedHistory = func(ctx context.Context, request historyattribution.Request) (historyattribution.Result, error) {
		start := time.Now()
		result, err := historyattribution.Collect(ctx, request)
		acquisitionSeconds = time.Since(start).Seconds()
		acquisitionDeadlineExceeded = errors.Is(ctx.Err(), context.DeadlineExceeded)
		return result, err
	}
	cycle := &codexCycle{started: time.Now()}
	ctx := context.WithValue(context.Background(), codexCycleKey{}, cycle)
	start := time.Now()
	_, generationErr := generateCommitPlan(ctx, root, snapshot, values, "")
	report := map[string]any{"acquisition_seconds": acquisitionSeconds, "acquisition_deadline_exceeded": acquisitionDeadlineExceeded, "elapsed_seconds": time.Since(start).Seconds(), "inference_calls": 0, "receiver_replies": "fixed fixtures for phase traversal only", "status": cycle.attribution.Status, "contexts": cycle.attribution.Contexts, "operations": cycle.attribution.Operations, "command_sources": len(cycle.attribution.CommandSources), "received_bytes": cycle.attribution.ReceivedBytes, "rpc_requests": cycle.attribution.Requests, "files": len(snapshot.Changes)}
	phases := []map[string]any{}
	defer func() {
		report["phases"] = phases
		data, _ := json.MarshalIndent(report, "", "  ")
		os.WriteFile(filepath.Join(out, "summary.json"), data, 0600)
	}()
	if cycle.attribution.History != nil {
		report["acquisition"] = cycle.attribution.History.Acquisition
	}
	if generationErr != nil {
		t.Fatal(generationErr)
	}
	final, err := gitstate.Collect(root, gitstate.Options{Pathspecs: paths})
	unchanged := err == nil && final.Head == snapshot.Head && final.Branch == snapshot.Branch && final.IndexIdentity == snapshot.IndexIdentity && len(changedSnapshotPaths(snapshot, final)) == 0
	report["git_unchanged"] = unchanged
	if !unchanged {
		t.Fatal("Git snapshot changed")
	}
	source := map[string]planning.AgentHistoryContext{}
	if cycle.attribution.History != nil {
		for _, c := range cycle.attribution.History.Contexts {
			source[c.ContextRef] = c
		}
	}
	fileHashes := map[string]string{}
	missingHistory := false
	for _, phase := range []string{"bounded-grouping", "bounded-category", "bounded-text"} {
		data, err := os.ReadFile(filepath.Join(out, phase+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var request mlx.Request
		if json.Unmarshal(data, &request) != nil {
			t.Fatal("bad wire request")
		}
		var p struct {
			Contexts []planning.AgentHistoryContext `json:"agent_history_contexts"`
			Files    []json.RawMessage              `json:"files"`
			Groups   []struct {
				Files    []json.RawMessage `json:"files"`
				Type     string            `json:"type"`
				Breaking *bool             `json:"breaking"`
			} `json:"groups"`
			Acquisition json.RawMessage `json:"agent_history_acquisition"`
			Projection  json.RawMessage `json:"agent_history_projection"`
		}
		if json.Unmarshal([]byte(request.Messages[1].Content), &p) != nil {
			t.Fatal("bad input")
		}
		var commandPayload struct {
			Commands []planning.AgentCommandExecutionContext `json:"agent_command_execution_contexts"`
		}
		if json.Unmarshal([]byte(request.Messages[1].Content), &commandPayload) != nil {
			t.Fatal("invalid command payload")
		}
		if phase != "bounded-grouping" && len(commandPayload.Commands) > 0 {
			t.Fatal("commands leaked to metadata")
		}
		if phase == "bounded-grouping" && os.Getenv("COMMITER_LIVE_EXPECT_COMMANDS") == "1" && len(commandPayload.Commands) == 0 {
			t.Fatal("real shell history missing")
		}
		seen := map[string]bool{}
		bodyBytes := 0
		for _, c := range p.Contexts {
			expected, ok := source[c.ContextRef]
			a, _ := json.Marshal(c)
			b, _ := json.Marshal(expected)
			if !ok || seen[c.ContextRef] || string(a) != string(b) {
				t.Fatal("context changed or duplicated")
			}
			seen[c.ContextRef] = true
			for _, s := range append(append([]string{}, c.UserRequests...), c.FinalResponses...) {
				bodyBytes += len(s)
			}
		}
		executionRefs := map[string]bool{}
		for _, c := range commandPayload.Commands {
			if !seen[c.ContextRef] || executionRefs[c.ExecutionRef] || c.Command == "" {
				t.Fatal("invalid command context reference")
			}
			executionRefs[c.ExecutionRef] = true
			originalCommand := ""
			for _, host := range cycle.attribution.History.Commands {
				if host.ExecutionRef == c.ExecutionRef {
					originalCommand = host.Command
				}
			}
			if originalCommand != c.Command {
				t.Fatal("command body changed during projection")
			}
			source, ok := cycle.attribution.CommandSources[c.ExecutionRef]
			if !ok || source.ItemID == "" {
				t.Fatal("command source lost")
			}
			for _, id := range c.MentionedFileIDs {
				if !strings.HasPrefix(id, "F") {
					t.Fatal("internal command file ID leaked")
				}
			}
		}
		files := p.Files
		if phase != "bounded-grouping" {
			if len(p.Acquisition) > 0 || len(p.Projection) > 0 {
				t.Fatal("global history status leaked")
			}
			for _, g := range p.Groups {
				files = append(files, g.Files...)
				if phase == "bounded-text" && (g.Type != "fix" || g.Breaking == nil || *g.Breaking) {
					t.Fatal("category decision lost")
				}
			}
		}
		candidates, blocks := 0, 0
		statuses := map[string]int{}
		for _, raw := range files {
			var f struct {
				ID     string `json:"file_id"`
				Diff   string `json:"raw_diff"`
				Blocks []struct {
					Status     string                           `json:"agent_history_match_status"`
					Candidates []planning.AgentHistoryCandidate `json:"agent_history_candidates"`
				} `json:"change_blocks"`
			}
			if json.Unmarshal(raw, &f) != nil || f.ID == "" {
				t.Fatal("invalid file")
			}
			hash := sha256.Sum256([]byte(f.Diff))
			h := hex.EncodeToString(hash[:])
			if phase == "bounded-grouping" {
				fileHashes[f.ID] = h
			} else if fileHashes[f.ID] != h {
				t.Fatal("diff or file identity changed")
			}
			for _, b := range f.Blocks {
				blocks++
				statuses[b.Status]++
				for _, c := range b.Candidates {
					candidates++
					if !seen[c.ContextRef] {
						t.Fatal("dangling history ref")
					}
				}
			}
		}
		phases = append(phases, map[string]any{"phase": phase, "command_executions": len(commandPayload.Commands), "wire_bytes": len(data), "contexts": len(p.Contexts), "history_body_bytes": bodyBytes, "files": len(files), "blocks": blocks, "candidate_refs": candidates, "match_statuses": statuses, "raw_diff_sha256": fileHashes})
		if len(p.Contexts) == 0 || candidates == 0 || bodyBytes == 0 {
			missingHistory = true
		}
	}
	report["verified"] = !missingHistory
	if missingHistory {
		t.Fatal("no live history delivered; empty fallback is not success")
	}
	t.Logf("real App Server to three capture-only helper inputs verified: contexts=%d operations=%d RPCs=%d bytes=%d", cycle.attribution.Contexts, cycle.attribution.Operations, cycle.attribution.Requests, cycle.attribution.ReceivedBytes)
}
