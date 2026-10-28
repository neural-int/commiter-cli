package historyattribution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestFetchSpeedLive(t *testing.T) {
	if os.Getenv("COMMITER_FETCH_SPEED") != "1" {
		t.Skip("explicit live-history opt in")
	}
	var plan struct {
		Queries []struct {
			Name, Root   string
			Paths        []string
			Since, Until int64
			Modes        []string
		}
		Cases []struct{ Name, Query, Input string }
	}
	b, e := os.ReadFile(os.Getenv("COMMITER_FETCH_SPEED_PLAN"))
	if e != nil || json.Unmarshal(b, &plan) != nil {
		t.Fatal("fixed plan required")
	}
	dir := os.Getenv("COMMITER_FETCH_SPEED_OUT")
	if dir == "" || os.Mkdir(dir, 0700) != nil {
		t.Fatal("new private directory required")
	}
	rows := []map[string]any{}
	for _, q := range plan.Queries {
		var reference []byte
		for _, mode := range q.Modes {
			targets := map[string]bool{}
			for _, path := range q.Paths {
				if !safePath(path) {
					t.Fatal("unsafe selected path")
				}
				targets[path] = true
			}
			ctx := context.WithValue(context.Background(), unlimitedHistoryContext{}, true)
			ctx = context.WithValue(ctx, historyFetchModeContext{}, mode)
			if os.Getenv("COMMITER_PIPELINE_WAITS") == "1" {
				ctx = context.WithValue(ctx, pipelineWaitContext{}, true)
			}
			if os.Getenv("COMMITER_FETCH_RESPONSE_CAP") == "1" {
				ctx = context.WithValue(ctx, responseLimitContext{}, researchResponseLimit)
			}
			var profile *HistoryTiming
			if os.Getenv("COMMITER_FETCH_TIMING") == "1" {
				profile = &HistoryTiming{}
				ctx = context.WithValue(ctx, historyTimingContext{}, profile)
			}
			h := readHistory(ctx, q.Root, []string{q.Root}, targets, q.Since, q.Until)
			name := q.Name + "-" + mode
			if profile != nil {
				if err := writeNew(filepath.Join(dir, name+"-timing.json"), profile, true); err != nil {
					t.Fatal(err)
				}
			}
			if err := writeNew(filepath.Join(dir, name+"-history.json"), h, true); err != nil {
				t.Fatal(err)
			}
			comparisons := map[string]any{}
			totalCompare := float64(0)
			for _, c := range plan.Cases {
				if c.Query != q.Name {
					continue
				}
				data, e := os.ReadFile(c.Input)
				if e != nil {
					t.Fatal(e)
				}
				var in Input
				if json.Unmarshal(data, &in) != nil {
					t.Fatal("invalid saved Git input")
				}
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
				err := validateInput(in)
				if err != nil {
					comparisons[c.Name] = map[string]any{"error": err}
					continue
				}
				begin := time.Now()
				result := compare(in)
				totalCompare += float64(time.Since(begin).Nanoseconds()) / 1e6
				comparisons[c.Name] = result
			}
			// Exclude measurement fields; retain all attribution-relevant source data.
			evidence := struct {
				Operations  []Operation
				Turns       []TurnContext
				Status      string
				Error       *StageError
				Dropped     int
				Comparisons map[string]any
			}{h.Operations, h.Turns, h.Status, h.Error, h.Dropped, comparisons}
			canonical, _ := json.Marshal(evidence)
			same := true
			if reference == nil {
				reference = canonical
			} else {
				same = reflect.DeepEqual(reference, canonical)
			}
			if err := writeNew(filepath.Join(dir, name+"-results.json"), comparisons, true); err != nil {
				t.Fatal(err)
			}
			row := map[string]any{"query": q.Name, "mode": mode, "acquisition_ms": h.ElapsedMS, "rpc_requests": h.Requests, "received_bytes": h.ReceivedBytes, "threads": h.ThreadsScanned, "turns": h.TurnsScanned, "items": h.ItemsScanned, "operations": len(h.Operations), "retained_turns": len(h.Turns), "status": h.Status, "error": h.Error, "dropped": h.Dropped, "compare_total_ms": totalCompare, "same_source_and_results": same}
			rows = append(rows, row)
			if err := writeNew(filepath.Join(dir, name+"-metrics.json"), row); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "progress.json"), mustJSON(rows), 0600); err != nil {
				t.Fatal(err)
			}
			if !same {
				t.Fatal("source or attribution differs; no retries")
			}
		}
	}
	if err := writeNew(filepath.Join(dir, "summary.json"), rows); err != nil {
		t.Fatal(err)
	}
}
