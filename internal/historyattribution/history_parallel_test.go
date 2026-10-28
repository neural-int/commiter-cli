package historyattribution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fixtureRPCEventStart() func() {
	write := func(kind string) {
		f, e := os.OpenFile(os.Getenv("POC_RPC_EVENTS"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			os.Exit(3)
		}
		b, _ := json.Marshal(struct {
			At   int64
			Kind string
		}{time.Now().UnixNano(), kind})
		f.Write(append(b, '\n'))
		f.Close()
	}
	write("start")
	time.Sleep(50 * time.Millisecond)
	return func() { write("end") }
}
func TestParallelTurnsPreserveSourceAndActuallyOverlap(t *testing.T) {
	root := gitFixture(t)
	fixed := strconv.FormatInt(time.Now().Unix(), 10)
	var baseline History
	for _, mode := range []string{"paged100", "parallel100", "parallel100x4", "parallel100x8", "pipeline100x4", "pipeline100x1"} {
		t.Run(mode, func(t *testing.T) {
			fixtureServer(t, root, "fetch_modes")
			t.Setenv("POC_EXPECT_FETCH", "paged100")
			t.Setenv("POC_FIXTURE_TURN_TIME", fixed)
			events := filepath.Join(t.TempDir(), "events.jsonl")
			t.Setenv("POC_RPC_EVENTS", events)
			ctx := context.WithValue(context.Background(), historyFetchModeContext{}, mode)
			ctx = context.WithValue(ctx, memorySamplerContext{}, func() (memorySample, error) { return memorySample{RSSBytes: 64 * 1024 * 1024, Pressure: 1}, nil })
			if mode == "pipeline100x4" {
				ctx = context.WithValue(ctx, pipelineWaitContext{}, true)
			}
			h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
			if h.Error != nil || h.Status != "complete" || len(h.Operations) != 6 || h.ItemsScanned != 624 {
				t.Fatal(h.Error, h.Status, h.ItemsScanned)
			}
			if mode == "pipeline100x4" && (h.Waits == nil || len(h.Waits.Turns) != 6 || h.Waits.ListingRPC == nil || len(h.Waits.ListingRPC.Calls) == 0 || len(h.Waits.WorkerRPC) != 4 || h.Waits.BodyEndMS <= 0 || h.Waits.ListEndMS <= 0) {
				t.Fatal("missing wait metrics")
			}
			if mode == "paged100" {
				baseline = h
			} else {
				if !reflect.DeepEqual(baseline.Operations, h.Operations) || !reflect.DeepEqual(baseline.Turns, h.Turns) {
					t.Fatal("parallel changed source/order")
				}
				a, b := fixedInputForParallel(baseline), fixedInputForParallel(h)
				if !reflect.DeepEqual(compare(a), compare(b)) {
					t.Fatal("parallel changed attribution")
				}
			}
			data, e := os.ReadFile(events)
			if e != nil {
				t.Fatal(e)
			}
			var rows []struct {
				At   int64
				Kind string
			}
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var row struct {
					At   int64
					Kind string
				}
				if json.Unmarshal([]byte(line), &row) != nil {
					t.Fatal("invalid events")
				}
				rows = append(rows, row)
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i].At < rows[j].At })
			active, max := 0, 0
			for _, row := range rows {
				if row.Kind == "start" {
					active++
				} else {
					active--
				}
				if active > max {
					max = active
				}
			}
			expected := 1
			if count := parallelWorkers(mode); count > 0 {
				expected = count
				if expected > 6 {
					expected = 6
				}
			}
			if max != expected || active != 0 {
				t.Fatal("wrong concurrent request count", max, active)
			}
		})
	}
}
func fixedInputForParallel(h History) Input { in := fixed(update("A", "B")); in.History = h; return in }
func TestParallelFailureDoesNotBecomeComplete(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "partial")
	ctx := context.WithValue(context.Background(), historyFetchModeContext{}, "parallel100")
	ctx = context.WithValue(ctx, memorySamplerContext{}, func() (memorySample, error) { return memorySample{RSSBytes: 64 * 1024 * 1024, Pressure: 1}, nil })
	h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Error == nil || h.Status == "complete" || !compare(fixedInputForParallel(h)).Fallback {
		t.Fatal("parallel failure became complete")
	}
}
