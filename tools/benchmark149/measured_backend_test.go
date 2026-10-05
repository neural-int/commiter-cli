package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestMeasuredBackendRetainsExpiredContextClassification(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	b := &measuredBackend{Executable: "/usr/bin/true"}
	r, e := b.ChatWithOptions(ctx, nil, nil, llm.Options{})
	if !errors.Is(e, context.DeadlineExceeded) || r.StopReason != "timeout" || r.Availability.EvalCount || r.Availability.PromptEvalCount {
		t.Fatal("deadline classification or unavailable telemetry was lost")
	}
}
