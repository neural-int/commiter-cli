package historyattribution

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Research opt-in only; same immutable query scopes and saved Git diffs as the
// previous experiment. Each live response is shared between both format arms.
func TestEvaluateHistoryProjectionLive(t *testing.T) {
	if os.Getenv("COMMITER_ATTRIBUTION_PROJECTION_EVAL") != "1" {
		t.Skip("explicit local-history opt in")
	}
	var plan struct {
		Queries []struct {
			Name, Root   string
			Roots, Paths []string
			Since, Until int64
		}
		Cases []struct{ Name, Query, Input string }
	}
	b, err := os.ReadFile(os.Getenv("COMMITER_ATTRIBUTION_PROJECTION_PLAN"))
	if err != nil || json.Unmarshal(b, &plan) != nil {
		t.Fatal("fixed plan required")
	}
	dir := os.Getenv("COMMITER_ATTRIBUTION_PROJECTION_OUT")
	if dir == "" || os.Mkdir(dir, 0700) != nil {
		t.Fatal("new private output directory required")
	}
	histories := map[string]History{}
	queryRows := []map[string]any{}
	for _, query := range plan.Queries {
		targets := map[string]bool{}
		for _, p := range query.Paths {
			if !safePath(p) {
				t.Fatal("unsafe selected path")
			}
			targets[p] = true
		}
		stats := &HistoryWireStats{}
		ctx := context.WithValue(context.Background(), unlimitedHistoryContext{}, true)
		ctx = context.WithValue(ctx, historyStatsContext{}, stats)
		h := readHistory(ctx, query.Root, query.Roots, targets, query.Since, query.Until)
		histories[query.Name] = h
		start := time.Now()
		projected, err := projectHistory(h)
		projectMS := float64(time.Since(start).Nanoseconds()) / 1e6
		if err != nil {
			t.Fatal(err)
		}
		legacy := legacyProjectionBaseline(h)
		oldJSON, _ := json.Marshal(legacy)
		newJSON, _ := json.Marshal(projected)
		if err = writeNew(filepath.Join(dir, query.Name+"-source-history.json"), h, true); err != nil {
			t.Fatal(err)
		}
		if err = writeNew(filepath.Join(dir, query.Name+"-projected-history.json"), projected, true); err != nil {
			t.Fatal(err)
		}
		row := map[string]any{"query": query.Name, "received_bytes": h.ReceivedBytes, "acquisition_ms": h.ElapsedMS, "rpc_requests": h.Requests, "threads": h.ThreadsScanned, "turns": h.TurnsScanned, "items": h.ItemsScanned, "operations": len(h.Operations), "retained_turns": len(h.Turns), "history_status": h.Status, "history_error": h.Error, "dropped": h.Dropped, "legacy_history_json_bytes": len(oldJSON), "projected_history_json_bytes": len(newJSON), "projection_ms": projectMS, "wire_item_stats": stats, "server_side_projection": false}
		queryRows = append(queryRows, row)
		if err = writeNew(filepath.Join(dir, query.Name+"-metrics.json"), row); err != nil {
			t.Fatal(err)
		}
	}
	rows := []map[string]any{}
	for i, c := range plan.Cases {
		original, err := os.ReadFile(c.Input)
		if err != nil {
			t.Fatal("fixed input unavailable")
		}
		var in Input
		if json.Unmarshal(original, &in) != nil {
			t.Fatal("invalid saved snapshot")
		}
		h := histories[c.Query]
		selected := h
		selected.Operations = nil
		selected.Turns = nil
		paths := map[string]bool{}
		for _, f := range in.Files {
			paths[f.Path] = true
			if f.OldPath != "" {
				paths[f.OldPath] = true
			}
		}
		turns := map[string]bool{}
		for _, op := range h.Operations {
			if paths[op.Path] || paths[op.OldPath] {
				selected.Operations = append(selected.Operations, op)
				turns[op.Thread+"/"+op.Turn] = true
			}
		}
		for _, turn := range h.Turns {
			if turns[turn.Thread+"/"+turn.Turn] {
				selected.Turns = append(selected.Turns, turn)
			}
		}
		in.History = selected
		projected, err := projectionInput(in)
		if err != nil {
			t.Fatal(err)
		}
		old := in
		old.History = legacyProjectionBaseline(selected)
		oldJSON, _ := json.Marshal(old)
		newJSON, _ := json.Marshal(projected)
		if err = writeNew(filepath.Join(dir, c.Name+"-legacy-input.json"), old, true); err != nil {
			t.Fatal(err)
		}
		if err = writeNew(filepath.Join(dir, c.Name+"-projected-input.json"), projected, true); err != nil {
			t.Fatal(err)
		}
		var oldComparison, newComparison Comparison
		var oldErr, newErr error
		var oldDecodeMS, newDecodeMS, oldCompareMS, newCompareMS float64
		runOld := func() {
			start := time.Now()
			var decoded Input
			oldErr = json.Unmarshal(oldJSON, &decoded)
			if oldErr == nil {
				oldErr = validateInput(decoded)
			}
			oldDecodeMS = float64(time.Since(start).Nanoseconds()) / 1e6
			if oldErr == nil {
				start = time.Now()
				oldComparison = compare(decoded)
				oldCompareMS = float64(time.Since(start).Nanoseconds()) / 1e6
			}
		}
		runNew := func() {
			start := time.Now()
			var decoded ProjectionInput
			decoder := json.NewDecoder(bytes.NewReader(newJSON))
			decoder.DisallowUnknownFields()
			newErr = decoder.Decode(&decoded)
			var restored Input
			if newErr == nil {
				restored, newErr = decoded.input()
			}
			newDecodeMS = float64(time.Since(start).Nanoseconds()) / 1e6
			if newErr == nil {
				start = time.Now()
				newComparison = compare(restored)
				newCompareMS = float64(time.Since(start).Nanoseconds()) / 1e6
			}
		}
		order := "legacy_then_projected"
		if i%2 == 0 {
			runOld()
			runNew()
		} else {
			order = "projected_then_legacy"
			runNew()
			runOld()
		}
		equivalent := reflect.DeepEqual(oldErr, newErr) && reflect.DeepEqual(oldComparison, newComparison)
		row := map[string]any{"case": c.Name, "query": c.Query, "order": order, "legacy_input_bytes": len(oldJSON), "projected_input_bytes": len(newJSON), "legacy_decode_validate_ms": oldDecodeMS, "projected_decode_validate_ms": newDecodeMS, "legacy_compare_ms": oldCompareMS, "projected_compare_ms": newCompareMS, "equivalent": equivalent}
		if oldErr != nil {
			row["legacy_error"] = oldErr
			row["projected_error"] = newErr
		} else {
			row["result"] = newComparison
		}
		rows = append(rows, row)
		if err = writeNew(filepath.Join(dir, c.Name+"-result.json"), row); err != nil {
			t.Fatal(err)
		}
		if !equivalent {
			t.Fatal("format arms differ; stop without retry")
		}
	}
	if err = writeNew(filepath.Join(dir, "summary.json"), map[string]any{"queries": queryRows, "cases": rows, "accuracy_claim": false, "same_response_for_both_formats": true, "server_side_projection": false}); err != nil {
		t.Fatal(err)
	}
}
func legacyProjectionBaseline(h History) History {
	// Remove only newly captured source-message metadata to reproduce the prior
	// compact retention format. The prior reader already discarded unused types.
	h.Turns = append([]TurnContext(nil), h.Turns...)
	for i, turn := range h.Turns {
		turn.StartedAt = nil
		turn.Status = ""
		turn.UserMessages = nil
		turn.FinalMessages = nil
		h.Turns[i] = turn
	}
	return h
}
