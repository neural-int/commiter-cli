package historyattribution

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"
)

type turnCollectContext struct{}
type turnJob struct {
	thread historyThread
	turn   historyTurn
}
type turnCollector struct {
	jobs []turnJob
	emit func(turnJob) error
}

// Bounded independent local App Server connections. Enumeration is performed once,
// then each worker processes whole Turns; cursors inside a Turn remain serial.
func readHistoryParallel(ctx context.Context, root string, targets map[string]bool, since, until int64) History {
	start := time.Now()
	var waits *PipelineWaits
	if enabled, _ := ctx.Value(pipelineWaitContext{}).(bool); enabled {
		waits = &PipelineWaits{}
	}
	mode, _ := ctx.Value(historyFetchModeContext{}).(string)
	workers := parallelWorkers(mode)
	if workers == 0 {
		workers = 2
	}
	if timing(ctx) != nil || ctx.Value(historyStatsContext{}) != nil {
		return History{Status: "unavailable", Error: failure("arguments", "parallel_diagnostics_not_supported").(*StageError)}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = context.WithValue(ctx, historyFetchModeContext{}, "paged100")
	ctx = context.WithValue(ctx, responseLimitContext{}, researchResponseLimit)
	guard, stopGuard, guardError := startMemoryGuard(ctx, cancel, workers)
	defer stopGuard()
	ctx = context.WithValue(ctx, memoryGuardContext{}, guard)
	if guardError != nil {
		return History{FetchMode: mode, Status: "unavailable", Error: guardError.(*StageError), Memory: guard.snapshot()}
	}
	type result struct {
		h   History
		err error
	}
	type work struct {
		index int
		job   turnJob
	}
	jobs := make(chan work, 16)
	results := map[int]result{}
	var resultsMu sync.Mutex
	var out History
	producerDone := make(chan struct{})
	produce := func() {
		defer close(producerDone)
		defer close(jobs)
		collector := &turnCollector{}
		if waits != nil {
			waits.ListingRPC = &HistoryTiming{}
		}
		count := 0
		enqueue := func(job turnJob) error {
			begin := time.Now()
			if waits != nil {
				select {
				case jobs <- work{count, job}:
					count++
					return nil
				default:
				}
			}
			defer func() {
				if waits != nil {
					waits.mu.Lock()
					waits.QueueBlockedMS += elapsedMS(begin)
					waits.QueueBlockedCount++
					waits.mu.Unlock()
				}
			}()
			select {
			case jobs <- work{count, job}:
				count++
				return nil
			case <-ctx.Done():
				return failure("history.turns", "cancelled")
			}
		}
		if mode == "pipeline100x4" || mode == "pipeline100x1" {
			collector.emit = enqueue
		}
		metaCtx := context.WithValue(ctx, turnCollectContext{}, collector)
		if waits != nil {
			metaCtx = context.WithValue(metaCtx, historyTimingContext{}, waits.ListingRPC)
		}
		out = readHistory(metaCtx, root, []string{root}, targets, since, until)
		if waits != nil {
			waits.mu.Lock()
			waits.ListEndMS = elapsedMS(start)
			waits.mu.Unlock()
		}
		if out.Error == nil && collector.emit == nil {
			for _, job := range collector.jobs {
				if enqueue(job) != nil {
					break
				}
			}
		}
		if out.Error != nil {
			cancel()
		}
	}
	// Batch enumeration completes before any worker starts; pipeline emits while listing.
	if mode == "pipeline100x4" || mode == "pipeline100x1" {
		go produce()
	} else {
		collector := &turnCollector{}
		out = readHistory(context.WithValue(ctx, turnCollectContext{}, collector), root, []string{root}, targets, since, until)
		go func() {
			defer close(producerDone)
			defer close(jobs)
			if out.Error != nil {
				cancel()
				return
			}
			for i, job := range collector.jobs {
				select {
				case jobs <- work{i, job}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	workerStats := make([]History, workers)
	if waits != nil {
		waits.WorkerRPC = make([]*HistoryTiming, workers)
		for i := range waits.WorkerRPC {
			waits.WorkerRPC[i] = &HistoryTiming{}
		}
	}
	var firstError error
	var errorOnce sync.Once
	fail := func(err error) { errorOnce.Do(func() { firstError = err; cancel() }) }
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			ctx := ctx
			if waits != nil {
				ctx = context.WithValue(ctx, historyTimingContext{}, waits.WorkerRPC[worker])
			}
			var p *historyRPC
			closeConnection := func() {
				if p != nil {
					workerStats[worker].Requests += p.seq
					workerStats[worker].ReceivedBytes += p.counter.bytes
					if p.maxMessage > workerStats[worker].MaxMessageBytes {
						workerStats[worker].MaxMessageBytes = p.maxMessage
					}
					p.close()
					p = nil
				}
			}
			defer closeConnection()
			config, _ := historyFetchConfig("paged100")
			for {
				waitBegin := time.Now()
				task, ok := <-jobs
				if waits != nil {
					waits.mu.Lock()
					waits.WorkerWaitMS += elapsedMS(waitBegin)
					waits.mu.Unlock()
				}
				if !ok {
					break
				}
				if ctx.Err() != nil {
					return
				}
				if guard.retire() {
					closeConnection()
				}
				admissionBegin := time.Now()
				if err := guard.acquire(ctx); err != nil {
					fail(err)
					return
				}
				if waits != nil {
					waits.mu.Lock()
					waits.AdmissionWaitMS += elapsedMS(admissionBegin)
					waits.mu.Unlock()
				}
				var err error
				if ctx.Err() != nil {
					guard.release()
					return
				}
				if p == nil {
					p, err = openHistory(ctx, root)
					if err == nil {
						var init map[string]json.RawMessage
						err = p.call("history.initialize", "initialize", map[string]any{"clientInfo": map[string]string{"name": "commiter-attribution-poc", "version": "1"}, "capabilities": map[string]bool{"experimentalApi": true}}, &init)
						if err == nil && json.NewEncoder(p.in).Encode(map[string]string{"method": "initialized"}) != nil {
							err = failure("history.initialize", "notification_failed")
						}
					}
				}
				if err != nil {
					guard.release()
					fail(err)
					return
				}
				job := task.job
				h := History{Status: "complete"}
				turnBegin := time.Now()
				beforeBytes, beforeCalls := p.counter.bytes, p.seq
				err = readHistoryTurn(ctx, p, root, job.thread, job.turn, targets, config, &h, map[string]bool{})
				if waits != nil {
					waits.mu.Lock()
					waits.Turns = append(waits.Turns, TurnTiming{task.index, elapsedMS(turnBegin), p.counter.bytes - beforeBytes, p.seq - beforeCalls})
					waits.BodyEndMS = elapsedMS(start)
					waits.mu.Unlock()
				}
				resultsMu.Lock()
				results[task.index] = result{h, err}
				resultsMu.Unlock()
				guard.release()
				if guard.retire() {
					closeConnection()
				}
				if err != nil {
					fail(err)
					return
				}
			}
		}(worker)
	}
	wait.Wait()
	<-producerDone
	out.FetchMode = mode
	out.Waits = waits
	for _, stat := range workerStats {
		out.Requests += stat.Requests
		out.ReceivedBytes += stat.ReceivedBytes
		if stat.MaxMessageBytes > out.MaxMessageBytes {
			out.MaxMessageBytes = stat.MaxMessageBytes
		}
	}
	// An enumeration conflict can concern a previously emitted Turn.
	if out.Error != nil && (mode == "pipeline100x4" || mode == "pipeline100x1") {
		results = map[int]result{}
	}
	// Merge in enumeration order, independently of worker completion order.
	indices := make([]int, 0, len(results))
	for index := range results {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	for _, index := range indices {
		result := results[index]
		out.ItemsScanned += result.h.ItemsScanned
		out.Dropped += result.h.Dropped
		out.DuplicateItems += result.h.DuplicateItems
		if result.err == nil {
			out.Operations = append(out.Operations, result.h.Operations...)
			out.Turns = append(out.Turns, result.h.Turns...)
		}
	}
	stopGuard()
	if out.Error != nil {
		firstError = out.Error
	}
	if e := guard.failure(); e != nil {
		firstError = e
	}
	out.Memory = guard.snapshot()
	if firstError == nil && ctx.Err() != nil {
		firstError = failure("history.items", "cancelled")
	}
	if firstError != nil {
		out.Status = "partial"
		out.Error = firstError.(*StageError)
	}

	if out.Error == nil && out.Dropped > 0 {
		out.Status = "partial"
		out.Error = failure("history.filter", "sensitive_evidence_omitted").(*StageError)
	}
	if out.Error != nil && len(out.Turns) == 0 {
		out.Status = "unavailable"
	}
	retained, _ := json.Marshal(struct {
		Operations []Operation
		Turns      []TurnContext
	}{out.Operations, out.Turns})
	out.RetainedBytes = len(retained)
	out.ElapsedMS = elapsedMS(start)
	return out
}
