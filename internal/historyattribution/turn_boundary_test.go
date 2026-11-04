package historyattribution

import (
	"context"
	"testing"
	"time"
)

func TestCollectMatchesContentAcrossHeadBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, started, diff, status string
		contexts                    int
	}{
		{"started_before_head_matching_edit", "100", update("A", "B"), "matched", 1},
		{"started_before_head_unrelated_edit", "100", update("X", "Y"), "unmatched", 0},
		{"started_after_head_matching_edit", "250", update("A", "B"), "matched", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, "ok")
			t.Setenv("POC_FIXTURE_TURN_TIME", tc.started)
			request := pipelineRequest()
			request.Root, request.Since, request.Until = root, time.Unix(200, 0), time.Unix(300, 0)
			request.Files[0].RawDiff = tc.diff
			got, err := Collect(normalMemoryContext(context.Background()), request)
			if err != nil {
				t.Fatal(err)
			}
			if got.Operations != 1 || got.Contexts != tc.contexts || got.History.Files[0].Blocks[0].Status != tc.status {
				t.Fatalf("old Turn content was lost or unrelated history admitted: %+v", got)
			}
		})
	}
}

func TestManyOldTurnsRetainResourceGuards(t *testing.T) {
	for _, mode := range []string{"warning_throttles", "cancel", "memory_limit"} {
		t.Run(mode, func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, "fetch_modes")
			t.Setenv("POC_EXPECT_FETCH", "paged100")
			t.Setenv("POC_FIXTURE_TURN_TIME", "100")
			t.Setenv("POC_FIXTURE_TURN_COUNT", "150")
			ctx := context.WithValue(context.Background(), historyFetchModeContext{}, "pipeline100x4")
			ctx = context.WithValue(ctx, memorySamplerContext{}, func() (memorySample, error) {
				if mode == "memory_limit" {
					return memorySample{RSSBytes: 3 * gib, Pressure: 1}, nil
				}
				return memorySample{RSSBytes: 64 * 1024 * 1024, Pressure: 2}, nil
			})
			deadline := 10 * time.Second
			if mode == "cancel" {
				deadline = 150 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(ctx, deadline)
			defer cancel()
			h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 200, 300)
			if mode == "warning_throttles" {
				if h.Error != nil || len(h.Operations) != 150 || h.Memory.PeakActive != 1 {
					t.Fatalf("old Turn pagination or warning throttle failed: operations=%d memory=%+v error=%v", len(h.Operations), h.Memory, h.Error)
				}
			} else if h.Status == "complete" || h.Error == nil {
				t.Fatal("resource stop promoted to complete")
			}
			if mode == "memory_limit" && h.Memory.StopReason != "memory_limit_exceeded" {
				t.Fatalf("memory limit lost: %+v", h.Memory)
			}
			if mode == "cancel" && ctx.Err() == nil {
				t.Fatal("parent cancellation not exercised")
			}
		})
	}
}
