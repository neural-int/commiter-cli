package historyattribution

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func normalMemoryContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, memorySamplerContext{}, func() (memorySample, error) { return memorySample{RSSBytes: 64 * 1024 * 1024, Pressure: 1}, nil })
}
func TestHistoryIdentityDeduplicationAndConflicts(t *testing.T) {
	for _, mode := range []string{"paged100", "parallel100x4", "pipeline100x4"} {
		t.Run(mode, func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, "fetch_modes")
			t.Setenv("POC_EXPECT_FETCH", "paged100")
			t.Setenv("POC_FIXTURE_TURN_TIME", "100")
			ctx := normalMemoryContext(context.WithValue(context.Background(), historyFetchModeContext{}, mode))
			read := func() History {
				return readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, 200)
			}
			base := read()
			if base.Error != nil {
				t.Fatal(base.Error)
			}
			for _, kind := range []string{"turn", "item", "turn_conflict", "item_conflict"} {
				t.Run(kind, func(t *testing.T) {
					t.Setenv("POC_DUPLICATE", kind)
					h := read()
					if kind == "turn" || kind == "item" {
						if h.Error != nil || !reflect.DeepEqual(h.Operations, base.Operations) || !reflect.DeepEqual(h.Turns, base.Turns) {
							t.Fatal("duplicate changed evidence", h.Error)
						}
						if h.DuplicateTurns+h.DuplicateItems == 0 {
							t.Fatal("duplicate not counted")
						}
					} else {
						if h.Error == nil || h.Error.Reason != "history_parse_failed" || h.Status == "complete" {
							t.Fatal("conflict accepted", h.Error)
						}
						for _, op := range h.Operations {
							if kind == "item_conflict" || op.Turn == "T1" {
								t.Fatal("conflicted turn retained")
							}
						}
					}
				})
			}
		})
	}
}
func TestMemoryGuardThresholdsAndRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(normalMemoryContext(context.Background()))
	defer cancel()
	g, stop, err := startMemoryGuard(ctx, cancel, 8)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if err = g.acquire(ctx); err != nil {
		t.Fatal(err)
	}
	g.update(memorySample{RSSBytes: gib, Pressure: 1})
	if g.ceiling != 1 || g.paused {
		t.Fatal("1GiB must freeze growth")
	}
	g.update(memorySample{RSSBytes: gib + gib/2, Pressure: 1})
	if !g.paused {
		t.Fatal("1.5GiB must pause")
	}
	blocked, finish := context.WithTimeout(ctx, 10*time.Millisecond)
	defer finish()
	if g.acquire(blocked) == nil {
		t.Fatal("paused admission succeeded")
	}
	g.release()
	g.update(memorySample{RSSBytes: gib - 1, Pressure: 1})
	g.update(memorySample{RSSBytes: gib - 1, Pressure: 1})
	if g.paused {
		t.Fatal("did not recover")
	}
	if e := g.acquire(ctx); e != nil {
		t.Fatal(e)
	}
	g.release()
	if e := g.update(memorySample{RSSBytes: 2 * gib, Pressure: 1}); e == nil || g.snapshot().StopReason != "memory_limit_exceeded" {
		t.Fatal("hard limit ignored")
	}
}
func TestMemoryGuardCriticalUnavailableAndStalled(t *testing.T) {
	for _, kind := range []string{"critical", "unavailable", "stalled"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(normalMemoryContext(context.Background()))
			defer cancel()
			g, stop, e := startMemoryGuard(ctx, cancel, 4)
			if e != nil {
				t.Fatal(e)
			}
			stop()
			switch kind {
			case "critical":
				e = g.update(memorySample{Pressure: 4})
			case "stalled":
				g.update(memorySample{RSSBytes: gib + gib/2, Pressure: 1})
				g.pausedAt = time.Now().Add(-31 * time.Second)
				e = g.update(memorySample{RSSBytes: gib + gib/2, Pressure: 1})
			case "unavailable":
				ctx = context.WithValue(ctx, memorySamplerContext{}, func() (memorySample, error) {
					return memorySample{}, failure("history.memory", "monitoring_unavailable")
				})
				g, stop, e = startMemoryGuard(ctx, cancel, 4)
				stop()
			}
			if e == nil || g.snapshot().StopReason == "" {
				t.Fatal("unsafe condition accepted")
			}
		})
	}
}

func TestProjectionSourceIdentityGuard(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "fetch_modes")
	t.Setenv("POC_EXPECT_FETCH", "paged100")
	ctx := context.WithValue(context.Background(), historyFetchModeContext{}, "paged100")
	h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	base, e := projectHistory(h)
	if e != nil {
		t.Fatal(e)
	}
	h.Turns = append(h.Turns, h.Turns[0])
	h.Operations = append(h.Operations, h.Operations[0])
	actual, e := projectHistory(h)
	if e != nil || !reflect.DeepEqual(base, actual) {
		t.Fatal("projection duplicated source", e)
	}
	h.Operations[len(h.Operations)-1].Path = "different.go"
	if _, e = projectHistory(h); e == nil {
		t.Fatal("projection accepted conflicting identity")
	}
}
func TestParallelCancelledDoesNotBecomeComplete(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "fetch_modes")
	t.Setenv("POC_EXPECT_FETCH", "paged100")
	ctx, cancel := context.WithCancel(normalMemoryContext(context.WithValue(context.Background(), historyFetchModeContext{}, "parallel100")))
	cancel()
	h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Status == "complete" || h.Error == nil {
		t.Fatal("cancel became complete")
	}
}
func TestParallelMemoryStopBeforeTurnAdmission(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "fetch_modes")
	t.Setenv("POC_EXPECT_FETCH", "paged100")
	ctx := context.WithValue(context.Background(), historyFetchModeContext{}, "parallel100x8")
	calls := 0
	ctx = context.WithValue(ctx, memorySamplerContext{}, func() (memorySample, error) {
		calls++
		if calls == 1 {
			return memorySample{RSSBytes: gib + gib/2, Pressure: 1}, nil
		}
		return memorySample{RSSBytes: 2 * gib, Pressure: 1}, nil
	})
	h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Error == nil || h.Error.Code != "memory_limit_exceeded" || h.Status == "complete" || h.ItemsScanned != 0 || h.Memory == nil || h.Memory.PeakActive != 0 {
		t.Fatal("memory stop failed", h.Error, h.Status, h.ItemsScanned, h.Memory)
	}
}
func TestConflictPreservesOtherCompletedTurns(t *testing.T) {
	for _, mode := range []string{"paged100", "parallel100"} {
		t.Run(mode, func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, "fetch_modes")
			t.Setenv("POC_EXPECT_FETCH", "paged100")
			t.Setenv("POC_DUPLICATE", "item_conflict_last")
			ctx := normalMemoryContext(context.WithValue(context.Background(), historyFetchModeContext{}, mode))
			h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
			if h.Error == nil || h.Status != "partial" || len(h.Operations) == 0 {
				t.Fatal("completed evidence lost", h.Error, h.Status)
			}
			for _, op := range h.Operations {
				if op.Turn == "T6" {
					t.Fatal("conflicting evidence retained")
				}
			}
		})
	}
}

func TestWarningReusesConnectionWhileLimitingParallelism(t *testing.T) {
	for _, sample := range []memorySample{{RSSBytes: 64 * 1024 * 1024, Pressure: 2}, {RSSBytes: gib, Pressure: 1}} {
		t.Run(fmt.Sprintf("rss%d_pressure%d", sample.RSSBytes, sample.Pressure), func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, "fetch_modes")
			t.Setenv("POC_EXPECT_FETCH", "paged100")
			t.Setenv("POC_FIXTURE_TURN_TIME", "100")
			ctx := context.WithValue(context.Background(), historyFetchModeContext{}, "parallel100x4")
			ctx = context.WithValue(ctx, memorySamplerContext{}, func() (memorySample, error) { return sample, nil })
			h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, 200)
			// Enumeration: initialize + two thread lists + one Turn list = 4.
			// Four workers retain their connections: 4 initializations + 12 item calls.
			// Admission remains serial; six per-Turn initializations would total 22.
			if h.Error != nil || h.Status != "complete" || len(h.Operations) != 6 || h.Requests != 20 || h.Memory.PeakActive != 1 {
				t.Fatal("warning caused restart, lost evidence, or excess concurrency", h.Error, h.Status, h.Requests, h.Memory)
			}
		})
	}
}
func TestMemoryPauseStillRetiresConnections(t *testing.T) {
	ctx, cancel := context.WithCancel(normalMemoryContext(context.Background()))
	defer cancel()
	g, stop, e := startMemoryGuard(ctx, cancel, 4)
	if e != nil {
		t.Fatal(e)
	}
	stop()
	g.update(memorySample{RSSBytes: gib, Pressure: 2})
	if g.retire() {
		t.Fatal("warning retired connection")
	}
	g.update(memorySample{RSSBytes: gib + gib/2, Pressure: 1})
	if !g.retire() {
		t.Fatal("pause must release connections")
	}
}
