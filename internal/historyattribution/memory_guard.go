package historyattribution

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const gib = int64(1024 * 1024 * 1024)

type memoryGuardContext struct{}
type memorySamplerContext struct{}
type memorySample struct {
	RSSBytes int64 `json:"rss_bytes"`
	Pressure int   `json:"pressure"`
}
type memoryEvent struct {
	ElapsedMS float64 `json:"elapsed_ms"`
	RSSBytes  int64   `json:"rss_bytes"`
	Pressure  int     `json:"pressure"`
	Active    int     `json:"active"`
	Ceiling   int     `json:"ceiling"`
	Paused    bool    `json:"paused"`
}
type MemoryProtection struct {
	MaxWorkers int           `json:"max_workers"`
	PeakActive int           `json:"peak_active"`
	Events     []memoryEvent `json:"events"`
	StopReason string        `json:"stop_reason,omitempty"`
}
type memoryGuard struct {
	reduced                      bool
	mu                           sync.Mutex
	max, ceiling, active, normal int
	paused                       bool
	pausedAt                     time.Time
	err                          error
	sample                       memorySample
	pids                         map[int]bool
	changed                      chan struct{}
	report                       MemoryProtection
	start                        time.Time
}

func guardFrom(ctx context.Context) *memoryGuard {
	g, _ := ctx.Value(memoryGuardContext{}).(*memoryGuard)
	return g
}
func (g *memoryGuard) notify()            { close(g.changed); g.changed = make(chan struct{}) }
func (g *memoryGuard) register(pid int)   { g.mu.Lock(); defer g.mu.Unlock(); g.pids[pid] = true }
func (g *memoryGuard) unregister(pid int) { g.mu.Lock(); defer g.mu.Unlock(); delete(g.pids, pid) }
func (g *memoryGuard) update(s memorySample) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sample = s
	if s.RSSBytes >= 2*gib {
		g.err = failure("history.memory", "memory_limit_exceeded")
	} else if s.Pressure >= 4 {
		g.err = failure("history.memory", "critical_memory_pressure")
	}
	if s.RSSBytes >= gib+gib/2 {
		if !g.paused {
			g.pausedAt = time.Now()
		}
		g.paused = true
		g.reduced = true
		if g.ceiling > 1 {
			g.ceiling--
		}
	}
	if s.RSSBytes >= gib || s.Pressure > 1 {
		g.normal = 0
		limit := g.active
		if limit < 1 {
			limit = 1
		}
		if limit < g.ceiling {
			g.ceiling = limit
		}
	} else {
		g.normal++
		if g.normal >= 2 {
			g.paused = false
			g.pausedAt = time.Time{}
			g.ceiling = g.max
			if g.reduced {
				g.ceiling = g.max / 2
				if g.ceiling < 1 {
					g.ceiling = 1
				}
			}
		}
	}
	// A retained baseline can prevent RSS dropping even with all workers idle.
	// Stop rather than wait forever in an unbounded research run.
	if g.paused && time.Since(g.pausedAt) >= 30*time.Second {
		g.err = failure("history.memory", "memory_backpressure_stalled")
	}
	if g.err != nil {
		g.report.StopReason = g.err.(*StageError).Code
	}
	g.report.Events = append(g.report.Events, memoryEvent{elapsedMS(g.start), s.RSSBytes, s.Pressure, g.active, g.ceiling, g.paused})
	g.notify()
	return g.err
}
func (g *memoryGuard) acquire(ctx context.Context) error {
	for {
		g.mu.Lock()
		if g.err != nil {
			e := g.err
			g.mu.Unlock()
			return e
		}
		if !g.paused && g.active < g.ceiling {
			g.active++
			if g.active > g.report.PeakActive {
				g.report.PeakActive = g.active
			}
			g.mu.Unlock()
			return nil
		}
		changed := g.changed
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return failure("history.memory", "cancelled")
		case <-changed:
		}
	}
}
func (g *memoryGuard) release() { g.mu.Lock(); defer g.mu.Unlock(); g.active--; g.notify() }
func (g *memoryGuard) retire() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Warning limits admission; retain connections to avoid per-Turn restarts.
	// Release idle connections only when memory must be reclaimed or stopping.
	return g.paused || g.err != nil
}
func (g *memoryGuard) failure() error { g.mu.Lock(); defer g.mu.Unlock(); return g.err }
func (g *memoryGuard) snapshot() *MemoryProtection {
	g.mu.Lock()
	defer g.mu.Unlock()
	r := g.report
	r.Events = append([]memoryEvent(nil), r.Events...)
	return &r
}
func (g *memoryGuard) nativeSample() (memorySample, error) {
	if runtime.GOOS != "darwin" {
		return memorySample{}, failure("history.memory", "monitoring_unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// comm/args and environment are never requested or logged.
	b, e := exec.CommandContext(ctx, "/bin/ps", "-axo", "pid=,ppid=,rss=").Output()
	if e != nil {
		return memorySample{}, failure("history.memory", "monitoring_unavailable")
	}
	type proc struct {
		pid, parent int
		rss         int64
	}
	var processes []proc
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			continue
		}
		pid, e1 := strconv.Atoi(f[0])
		parent, e2 := strconv.Atoi(f[1])
		rss, e3 := strconv.ParseInt(f[2], 10, 64)
		if e1 == nil && e2 == nil && e3 == nil {
			processes = append(processes, proc{pid, parent, rss * 1024})
		}
	}
	g.mu.Lock()
	owned := map[int]bool{}
	for pid := range g.pids {
		owned[pid] = true
	}
	g.mu.Unlock()
	for pass := 0; pass < len(processes); pass++ {
		changed := false
		for _, p := range processes {
			if owned[p.parent] && !owned[p.pid] {
				owned[p.pid] = true
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	var rss int64
	found := false
	for _, p := range processes {
		if p.pid == os.Getpid() {
			rss += p.rss
			found = true
		} else if owned[p.pid] {
			rss += p.rss
		}
	}
	if !found {
		return memorySample{}, failure("history.memory", "monitoring_unavailable")
	}
	b, e = exec.CommandContext(ctx, "/usr/sbin/sysctl", "-n", "kern.memorystatus_vm_pressure_level").Output()
	if e != nil {
		return memorySample{}, failure("history.memory", "monitoring_unavailable")
	}
	pressure, e := strconv.Atoi(strings.TrimSpace(string(b)))
	if e != nil || (pressure != 1 && pressure != 2 && pressure != 4) {
		return memorySample{}, failure("history.memory", "monitoring_unavailable")
	}
	return memorySample{rss, pressure}, nil
}
func startMemoryGuard(ctx context.Context, cancel context.CancelFunc, max int) (*memoryGuard, func(), error) {
	g := &memoryGuard{max: max, ceiling: max, pids: map[int]bool{}, changed: make(chan struct{}), start: time.Now(), report: MemoryProtection{MaxWorkers: max}}
	sample := g.nativeSample
	if fake, ok := ctx.Value(memorySamplerContext{}).(func() (memorySample, error)); ok {
		sample = fake
	}
	poll := func() error {
		s, e := sample()
		if e != nil {
			g.mu.Lock()
			g.err = e
			g.report.StopReason = "monitoring_unavailable"
			g.notify()
			g.mu.Unlock()
			return e
		}
		return g.update(s)
	}
	if e := poll(); e != nil {
		return g, func() {}, e
	}
	watchCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				if poll() != nil {
					cancel()
					return
				}
			}
		}
	}()
	return g, func() { stop(); <-done }, nil
}
