package historyattribution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Explicit research opt-in only. A single shared history sweep is reused for
// the frozen per-path cases; these are not independent wire-volume samples.
func TestMeasureUnlimitedLiveHistory(t *testing.T) {
	if os.Getenv("COMMITER_ATTRIBUTION_VOLUME_PROBE") != "1" {
		t.Skip("explicit local-history opt in")
	}
	planFile := os.Getenv("COMMITER_ATTRIBUTION_VOLUME_PLAN")
	dir := os.Getenv("COMMITER_ATTRIBUTION_VOLUME_OUT")
	var plan struct {
		Root  string   `json:"root"`
		Roots []string `json:"roots"`
		Paths []string `json:"paths"`
		Since int64    `json:"since"`
		Until int64    `json:"until"`
	}
	b, err := os.ReadFile(planFile)
	if err != nil || json.Unmarshal(b, &plan) != nil {
		t.Fatal("fixed plan required")
	}
	if dir == "" || os.Mkdir(dir, 0700) != nil {
		t.Fatal("new private output directory required")
	}
	targets := map[string]bool{}
	for _, path := range plan.Paths {
		if !safePath(path) {
			t.Fatal("unsafe selected path")
		}
		targets[path] = true
	}
	ctx := context.WithValue(context.Background(), unlimitedHistoryContext{}, true)
	start := time.Now()
	h := readHistory(ctx, plan.Root, plan.Roots, targets, plan.Since, plan.Until)
	if err := writeNew(filepath.Join(dir, "history.json"), h, true); err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for _, path := range plan.Paths {
		selected := History{Status: h.Status, Error: h.Error}
		turnIDs := map[string]bool{}
		diffBytes := 0
		for _, op := range h.Operations {
			if op.Path == path || op.OldPath == path {
				selected.Operations = append(selected.Operations, op)
				turnIDs[op.Thread+"/"+op.Turn] = true
				if op.Diff != nil {
					diffBytes += len(*op.Diff)
				}
			}
		}
		for _, turn := range h.Turns {
			if turnIDs[turn.Thread+"/"+turn.Turn] {
				selected.Turns = append(selected.Turns, turn)
			}
		}
		data, _ := json.Marshal(selected)
		rows = append(rows, map[string]any{"path": path, "operations": len(selected.Operations), "turns": len(selected.Turns), "diff_bytes": diffBytes, "selected_history_json_bytes": len(data)})
	}
	summary := map[string]any{"shared_sweep": true, "case_count": len(rows), "rows": rows, "history_status": h.Status, "history_error": h.Error, "received_bytes": h.ReceivedBytes, "elapsed_ms": h.ElapsedMS, "wall_ms": float64(time.Since(start).Microseconds()) / 1000, "threads": h.ThreadsScanned, "turns": h.TurnsScanned, "items": h.ItemsScanned, "operations": len(h.Operations), "retained_turns": len(h.Turns), "retained_json_bytes": h.RetainedBytes, "max_message_bytes": h.MaxMessageBytes, "rpc_requests": h.Requests, "dropped": h.Dropped}
	if err := writeNew(filepath.Join(dir, "summary.json"), summary); err != nil {
		t.Fatal(err)
	}
	if h.Error != nil {
		t.Log("history diagnostics retained; not promoting partial to complete")
	}
}
