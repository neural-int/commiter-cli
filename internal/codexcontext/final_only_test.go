package codexcontext

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCollectKeepsOnlySafeFinalResponse(t *testing.T) {
	for _, withoutRequest := range []bool{false, true} {
		f, request := testHistory(t)
		if withoutRequest {
			f.items["turn"] = f.items["turn"][1:]
		} else {
			f.items["turn"][0] = logItem("turn", "request", "userMessage", map[string]any{"content": []map[string]string{{"type": "text", "text": "PRIVATE request is not input"}}})
		}
		result := collect(f, request)
		if result.Status != "available" || len(result.Records) != 1 || result.Records[0].FinalResponse != "Corrected the boundary" {
			t.Fatalf("safe final response was lost: missing_request=%v status=%s records=%d", withoutRequest, result.Status, len(result.Records))
		}
		encoded, _ := json.Marshal(result.Records)
		if strings.Contains(string(encoded), "user_request") || strings.Contains(string(encoded), "PRIVATE") {
			t.Fatal("user request leaked into final-only history")
		}
	}
}

func TestCollectOmitsMissingUnsafeOrOversizedFinalResponse(t *testing.T) {
	for _, final := range []string{"", "PRIVATE final response", strings.Repeat("x", MaxContextBytes)} {
		f, request := testHistory(t)
		f.items["turn"][3] = logItem("turn", "answer", "agentMessage", map[string]any{"phase": "final_answer", "text": final})
		if result := collect(f, request); len(result.Records) != 0 {
			t.Fatal("empty/unsafe/oversized final response must not fall back to its user request")
		}
	}
}
