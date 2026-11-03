package historyattribution

import (
	"context"
	"sync"
	"time"
)

const historyRPCIdleTimeout = 30 * time.Second

type rpcIdleTimeoutContext struct{}

// A call may take arbitrarily long while response bytes keep arriving.
// Closing both pipes also interrupts a blocked request write.
type rpcIdleGuard struct {
	mu            sync.Mutex
	timer         *time.Timer
	last          time.Time
	started       time.Time
	limit         time.Duration
	done, expired bool
	interrupt     func()
}

func newRPCIdleGuard(ctx context.Context, interrupt func()) *rpcIdleGuard {
	limit := historyRPCIdleTimeout
	if override, ok := ctx.Value(rpcIdleTimeoutContext{}).(time.Duration); ok && override > 0 {
		limit = override
	}
	g := &rpcIdleGuard{last: time.Now(), limit: limit, interrupt: interrupt}
	g.started = g.last
	g.timer = time.AfterFunc(limit, g.check)
	return g
}
func (g *rpcIdleGuard) check() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.done {
		return
	}
	if remaining := g.limit - time.Since(g.last); remaining > 0 {
		g.timer.Reset(remaining)
		return
	}
	g.expired, g.done = true, true
	g.interrupt()
}
func (g *rpcIdleGuard) progress() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.done {
		g.last = time.Now()
	}
}

// Unrelated replies and notifications do not make progress on this call.
func (g *rpcIdleGuard) unrelatedMessage() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.done {
		g.last = g.started
		remaining := g.limit - time.Since(g.started)
		if remaining <= 0 {
			remaining = time.Nanosecond
		}
		g.timer.Reset(remaining)
	}
}
func (g *rpcIdleGuard) stop() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.done = true
	g.timer.Stop()
	return g.expired
}
