package historyattribution

import (
	"context"
	"testing"
	"time"
)

func TestCollectUnresponsiveServerFallsBack(t *testing.T) {
	for _, mode := range []string{"timeout", "stalled_response", "progress_response", "fetch_stall", "notifications"} {
		t.Run(mode, func(t *testing.T) {
			root := gitFixture(t)
			fixtureServer(t, root, mode)
			t.Setenv("POC_EXPECT_FETCH", "paged100")
			request := pipelineRequest()
			request.Root = root
			ctx, cancel := context.WithTimeout(normalMemoryContext(context.Background()), 15*time.Second)
			defer cancel()
			idleLimit := 200 * time.Millisecond
			if mode == "progress_response" {
				idleLimit = time.Second
			}
			ctx = context.WithValue(ctx, rpcIdleTimeoutContext{}, idleLimit)
			got, err := Collect(ctx, request)
			if mode == "progress_response" {
				if err != nil || got.Status != "available" || ctx.Err() != nil {
					t.Fatalf("progressing RPC was stopped: %+v %v", got, err)
				}
				return
			}
			if err != nil || ctx.Err() != nil || got.Status != "unavailable" || got.History.Acquisition.Status != "unavailable" || len(got.History.Files) != 1 {
				t.Fatalf("idle failure lost Git fallback: %+v %v parent=%v", got, err, ctx.Err())
			}
			// Retain the diagnostic code rather than misclassifying idle as a transport failure.
			readCtx := ctx
			if mode == "fetch_stall" {
				readCtx = context.WithValue(ctx, historyFetchModeContext{}, "pipeline100x4")
			}
			h := readHistory(readCtx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
			if h.Error == nil || h.Error.Code != "rpc_idle_timeout" {
				t.Fatalf("wrong idle failure: %+v", h.Error)
			}
		})
	}
}

func TestRPCIdleProgressAllowsLongCollection(t *testing.T) {
	ctx := context.WithValue(context.Background(), rpcIdleTimeoutContext{}, 100*time.Millisecond)
	interrupted := make(chan struct{})
	g := newRPCIdleGuard(ctx, func() { close(interrupted) })
	defer g.stop()
	for i := 0; i < 8; i++ {
		time.Sleep(25 * time.Millisecond)
		g.progress()
		select {
		case <-interrupted:
			t.Fatal("active collection timed out")
		default:
		}
	}
	select {
	case <-interrupted:
	case <-time.After(time.Second):
		t.Fatal("idle collection never stopped")
	}
}
