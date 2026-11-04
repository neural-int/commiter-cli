package historyattribution

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// Trigger from an actual checkpoint, independently of machine speed.
type checkpointContext struct {
	context.Context
	calls   int
	trigger func()
}

func (c *checkpointContext) Err() error {
	c.calls++
	if c.calls == 100 {
		c.trigger()
	}
	return c.Context.Err()
}

func TestMatchingStopsDuringCPUWork(t *testing.T) {
	for _, phase := range []string{"attribution", "comparison", "split_merge"} {
		for _, stop := range []string{"cancel", "memory"} {
			t.Run(phase+"/"+stop, func(t *testing.T) {
				parent, cancel := context.WithCancel(context.Background())
				defer cancel()
				ctx := normalMemoryContext(parent)
				g, stopGuard, err := startMemoryGuard(ctx, cancel, 1)
				if err != nil {
					t.Fatal(err)
				}
				stopGuard()
				ctx = context.WithValue(ctx, memoryGuardContext{}, g)
				counted := &checkpointContext{Context: ctx, trigger: func() {
					if stop == "cancel" {
						cancel()
					} else {
						g.update(memorySample{RSSBytes: 2 * gib, Pressure: 1})
					}
				}}
				in := fixed("@@ -0,0 +1 @@\n+target\n")
				for i := 0; i < 1000; i++ {
					in.History.Operations = append(in.History.Operations, op(fmt.Sprint(i), "@@ -0,0 +1 @@\n+unrelated\n"))
				}
				switch phase {
				case "attribution":
					value, e := attributeWithSourcesContext(counted, pipelineRequest(), in.History, map[string]ContextSource{})
					err = e
					if value != nil {
						t.Fatal("partial attribution escaped")
					}
				case "comparison":
					value, e := compareContext(counted, in)
					err = e
					if len(value.Results) > 0 {
						t.Fatal("partial comparison escaped")
					}
				case "split_merge":
					value := Comparison{Results: []Attribution{{Path: "file.go", Block: 0, Status: "unmatched"}}}
					err = markSplitMergeEvidenceContext(counted, in, &value)
				}
				if err == nil || counted.calls < 100 {
					t.Fatal("matching ignored stop", err, counted.calls)
				}
				if stop == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if stop == "memory" {
					var stage *StageError
					if !errors.As(err, &stage) || stage.Code != "memory_limit_exceeded" {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestMatchingBackpressureResumesAndCancels(t *testing.T) {
	for _, resume := range []bool{false, true} {
		ctx, cancel := context.WithCancel(normalMemoryContext(context.Background()))
		g, stop, err := startMemoryGuard(ctx, cancel, 1)
		if err != nil {
			t.Fatal(err)
		}
		stop()
		g.update(memorySample{RSSBytes: gib + gib/2, Pressure: 1})
		ctx = context.WithValue(ctx, memoryGuardContext{}, g)
		done := make(chan error, 1)
		go func() { done <- matchingCheckpoint(ctx) }()
		select {
		case <-done:
			t.Fatal("CPU matching ignored memory backpressure")
		case <-time.After(20 * time.Millisecond):
		}
		if resume {
			g.update(memorySample{RSSBytes: 1, Pressure: 1})
			g.update(memorySample{RSSBytes: 1, Pressure: 1})
		} else {
			cancel()
		}
		select {
		case e := <-done:
			if resume && e != nil || !resume && !errors.Is(e, context.Canceled) {
				t.Fatal(e)
			}
		case <-time.After(time.Second):
			t.Fatal("matching stuck")
		}
		cancel()
	}
}

func TestContentContainmentCancelsInsideSearch(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &checkpointContext{Context: parent, trigger: cancel}
	whole, part := make([]EditLine, 5000), make([]EditLine, 1000)
	for i := range whole {
		whole[i] = EditLine{"+", "same"}
	}
	for i := range part {
		part[i] = EditLine{"+", "same"}
	}
	part[len(part)-1] = EditLine{"+", "different"}
	_, _, ok, err := containedLinesContext(ctx, whole, part)
	if ok || !errors.Is(err, context.Canceled) || ctx.calls < 100 {
		t.Fatal("inner containment search ignored cancellation", err)
	}
}

func TestCollectRequiresMatchingMemoryProtection(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "ok")
	var samples atomic.Int32
	ctx := context.WithValue(context.Background(), memorySamplerContext{}, func() (memorySample, error) {
		if samples.Add(1) > 1 {
			return memorySample{RSSBytes: 2 * gib, Pressure: 1}, nil
		}
		return memorySample{RSSBytes: 1, Pressure: 1}, nil
	})
	request := pipelineRequest()
	request.Root = root
	result, err := Collect(ctx, request)
	var stage *StageError
	if result.History != nil || !errors.As(err, &stage) || stage.Code != "memory_limit_exceeded" {
		t.Fatal("Collect ran matching without memory admission", result, err)
	}
}
