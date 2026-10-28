package historyattribution

import (
	"context"
	"time"
)

type historyTimingContext struct{}
type RPCTiming struct {
	StartNS          int64   `json:"start_ns"`
	EndNS            int64   `json:"end_ns"`
	RequestKey       string  `json:"request_key,omitempty"`
	Method           string  `json:"method"`
	CallIndex        int     `json:"call_index"`
	TotalMS          float64 `json:"total_ms"`
	SendMS           float64 `json:"send_ms"`
	WaitReadMS       float64 `json:"wait_read_ms"`
	EnvelopeDecodeMS float64 `json:"envelope_decode_ms"`
	ResultDecodeMS   float64 `json:"result_decode_ms"`
	ResponseBytes    int     `json:"response_bytes"`
	Failed           bool    `json:"failed"`
}
type HistoryTiming struct {
	CleanupMS           float64     `json:"cleanup_ms"`
	SpawnMS             float64     `json:"process_spawn_ms"`
	TotalMS             float64     `json:"history_total_ms"`
	ItemSelectionMS     float64     `json:"item_selection_ms"`
	ContextValidationMS float64     `json:"turn_context_validation_ms"`
	PathMS              float64     `json:"path_validation_ms"`
	SafetyMS            float64     `json:"safety_validation_ms"`
	Calls               []RPCTiming `json:"calls"`
}

func timing(ctx context.Context) *HistoryTiming {
	p, _ := ctx.Value(historyTimingContext{}).(*HistoryTiming)
	return p
}
func elapsedMS(begin time.Time) float64 { return float64(time.Since(begin).Nanoseconds()) / 1e6 }
func selectionTimer(ctx context.Context, kind string) func() {
	p := timing(ctx)
	if p == nil {
		return func() {}
	}
	begin := time.Now()
	return func() {
		v := elapsedMS(begin)
		switch kind {
		case "item":
			p.ItemSelectionMS += v
		case "context":
			p.ContextValidationMS += v
		case "path":
			p.PathMS += v
		}
	}
}
func timedSafeText(ctx context.Context, text string) bool {
	p := timing(ctx)
	if p == nil {
		return safeText(text)
	}
	begin := time.Now()
	ok := safeText(text)
	p.SafetyMS += elapsedMS(begin)
	return ok
}
