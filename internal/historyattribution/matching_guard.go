package historyattribution

import "context"

// CPU matching shares the existing memory thresholds and backpressure policy.
// Checkpoints also run inside candidate loops, not just between pipeline stages.
func matchingCheckpoint(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	g := guardFrom(ctx)
	if g == nil {
		return nil
	}
	for {
		g.mu.Lock()
		err, paused, changed := g.err, g.paused, g.changed
		g.mu.Unlock()
		if err != nil {
			return err
		}
		if !paused {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}
