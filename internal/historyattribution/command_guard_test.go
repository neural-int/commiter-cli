package historyattribution

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func TestCommandProjectionStopsInsideWork(t *testing.T) {
	for _, phase := range []string{"validation", "emission"} {
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
				trigger := func() {
					if stop == "cancel" {
						cancel()
					} else {
						g.update(memorySample{RSSBytes: 2 * gib, Pressure: 1})
					}
				}
				request := pipelineRequest()
				count := 0
				if phase == "validation" {
					ctx = &checkpointContext{Context: ctx, trigger: trigger}
				} else {
					request.SafeCommand = func(string, []string) bool {
						count++
						if count == 5 {
							trigger()
						}
						return true
					}
				}
				h := History{Status: "complete", Turns: []TurnContext{{Thread: "S", Turn: "T", Status: "completed", Final: "Inspected files"}}}
				for i := 0; i < 500; i++ {
					h.Turns[0].Commands = append(h.Turns[0].Commands, CommandExecution{ItemID: fmt.Sprint(i), Command: "cat file.go", Cwd: "/local", Status: "completed"})
				}
				out := &planning.GroupingHistory{}
				err = appendCommandContextsContext(ctx, request, h, out, map[string]ContextSource{}, map[string]CommandSource{})
				if err == nil || len(out.Commands) >= 500 {
					t.Fatal("command projection ignored stop", err)
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
				if phase == "emission" && count != 5 {
					t.Fatal("stop did not happen during command processing", count)
				}
			})
		}
	}
}
