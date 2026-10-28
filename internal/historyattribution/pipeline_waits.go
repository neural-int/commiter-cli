package historyattribution

import "sync"

type pipelineWaitContext struct{}
type TurnTiming struct {
	Index         int     `json:"index"`
	DurationMS    float64 `json:"duration_ms"`
	ReceivedBytes int64   `json:"received_bytes"`
	Requests      int     `json:"requests"`
}

// Durations overlap across workers and must not be added to wall time.
type PipelineWaits struct {
	ListingRPC        *HistoryTiming   `json:"listing_rpc,omitempty"`
	WorkerRPC         []*HistoryTiming `json:"worker_rpc,omitempty"`
	mu                sync.Mutex
	WorkerWaitMS      float64      `json:"worker_wait_sum_ms"`
	AdmissionWaitMS   float64      `json:"admission_wait_sum_ms"`
	QueueBlockedMS    float64      `json:"queue_blocked_ms"`
	QueueBlockedCount int          `json:"queue_blocked_count"`
	ListEndMS         float64      `json:"list_end_ms"`
	BodyEndMS         float64      `json:"body_end_ms"`
	Turns             []TurnTiming `json:"turns"`
}
