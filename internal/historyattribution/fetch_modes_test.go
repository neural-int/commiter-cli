package historyattribution

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func fetchFixtureReply(method string, args map[string]any, mode, failureMode string) (any, bool) {
	config, err := historyFetchConfig(mode)
	if err != nil {
		return nil, false
	}
	root := os.Getenv("POC_FIXTURE_ROOT")
	items := []any{
		map[string]any{"id": "U", "type": "userMessage", "content": []any{map[string]any{"type": "text", "text": "Change A to B"}}},
		map[string]any{"id": "F", "type": "fileChange", "status": "completed", "changes": []any{map[string]any{"path": "file.go", "kind": map[string]any{"type": "update"}, "diff": update("A", "B")}}},
		map[string]any{"id": "A", "type": "agentMessage", "phase": "final_answer", "text": "Changed A to B"},
	}
	for i := 0; i < 101; i++ {
		items = append(items, map[string]any{"id": fmt.Sprintf("progress%d", i), "type": "agentMessage", "phase": "commentary", "text": "progress"})
	}
	duplicate := os.Getenv("POC_DUPLICATE")
	switch method {
	case "initialize":
		return map[string]any{}, true
	case "thread/list":
		cwd, ok := args["cwd"].([]any)
		if !ok || len(cwd) != 1 || cwd[0] != root || args["limit"] != float64(config.threadPage) {
			return nil, false
		}
		return map[string]any{"data": []any{map[string]any{"id": "OTHER", "cwd": root + "/another-worktree"}, map[string]any{"id": "S", "cwd": root}}}, true
	case "thread/turns/list":
		if args["threadId"] != "S" || args["limit"] != float64(config.turnPage) || args["itemsView"] != config.itemsView {
			return nil, false
		}
		start := 0
		if cursor, ok := args["cursor"].(string); ok {
			start, _ = strconv.Atoi(cursor)
		}
		end := start + config.turnPage
		if end > 6 {
			end = 6
		}
		turns := []any{}
		for i := start; i < end; i++ {
			turn := map[string]any{"id": fmt.Sprintf("T%d", i+1), "startedAt": fixtureTurnTime(), "status": "completed"}
			if config.itemsView == "full" && failureMode != "full_missing" {
				turn["items"] = items
			}
			turns = append(turns, turn)
		}
		if duplicate == "turn" || duplicate == "turn_conflict" {
			extra := map[string]any{"id": "T1", "startedAt": fixtureTurnTime(), "status": "completed"}
			if duplicate == "turn_conflict" {
				extra["status"] = "inProgress"
			}
			turns = append(turns, extra)
		}
		page := map[string]any{"data": turns}
		if end < 6 {
			page["nextCursor"] = strconv.Itoa(end)
		}
		return page, true
	case "thread/items/list":
		if config.itemsView == "full" || args["limit"] != float64(config.itemPage) || args["threadId"] != "S" || !containsString([]string{"T1", "T2", "T3", "T4", "T5", "T6"}, fmt.Sprint(args["turnId"])) {
			return nil, false
		}
		start := 0
		if cursor, ok := args["cursor"].(string); ok {
			start, _ = strconv.Atoi(cursor)
		}
		end := start + config.itemPage
		if end > len(items) {
			end = len(items)
		}
		entries := []any{}
		for _, item := range items[start:end] {
			entries = append(entries, map[string]any{"turnId": args["turnId"], "item": item})
		}
		if start > 0 && (duplicate == "item" || duplicate == "item_conflict" || (duplicate == "item_conflict_last" && args["turnId"] == "T6")) {
			item := items[1]
			if duplicate == "item_conflict" || duplicate == "item_conflict_last" {
				item = map[string]any{"id": "F", "type": "fileChange", "status": "failed"}
			}
			entries = append(entries, map[string]any{"turnId": args["turnId"], "item": item})
		}
		page := map[string]any{"data": entries}
		if end < len(items) {
			page["nextCursor"] = strconv.Itoa(end)
		}
		return page, true
	}
	return nil, false
}
func TestFetchModesPreserveResultsAndReduceRequests(t *testing.T) {
	root := gitFixture(t)
	fixedTurnTime := strconv.FormatInt(time.Now().Unix(), 10)
	var baseline History
	requests := map[string]int{}
	for _, mode := range []string{"paged20", "paged100", "paged500", "full5"} {
		t.Run(mode, func(t *testing.T) {
			fixtureServer(t, root, "fetch_modes")
			t.Setenv("POC_FIXTURE_TURN_TIME", fixedTurnTime)
			t.Setenv("POC_EXPECT_FETCH", mode)
			ctx := context.WithValue(context.Background(), historyFetchModeContext{}, mode)
			h := readHistory(ctx, root, []string{root, root + "/another-worktree"}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
			if h.Status != "complete" || h.Error != nil || h.ThreadsScanned != 1 || len(h.Operations) != 6 || h.ItemsScanned != 624 || h.FetchMode != mode {
				t.Fatal(h.Status, h.Error, h.ThreadsScanned, h.ItemsScanned)
			}
			requests[mode] = h.Requests
			if mode == "paged20" {
				baseline = h
			} else if !reflect.DeepEqual(baseline.Operations, h.Operations) || !reflect.DeepEqual(baseline.Turns, h.Turns) {
				t.Fatal("source records changed")
			}
			old, new := fixed(update("A", "B")), fixed(update("A", "B"))
			old.History = baseline
			new.History = h
			if !reflect.DeepEqual(compare(old), compare(new)) {
				t.Fatal("attribution changed")
			}
		})
	}
	if !(requests["paged20"] > requests["paged100"] && requests["paged100"] > requests["paged500"] && requests["paged500"] > requests["full5"]) {
		t.Fatal(requests)
	}
}
func TestFullModeMissingItemsFailsClosed(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "full_missing")
	t.Setenv("POC_EXPECT_FETCH", "full5")
	ctx := context.WithValue(context.Background(), historyFetchModeContext{}, "full5")
	h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Error == nil || h.Error.Code != "invalid_schema" || h.Status == "complete" {
		t.Fatal("missing full items accepted")
	}
}
func TestCaptureFetchModeAndInvalidMode(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "fetch_modes")
	t.Setenv("POC_EXPECT_FETCH", "full5")
	in, err := captureWithMode(root, []string{"file.go"}, time.Now(), "full5")
	if err != nil || in.History.Status != "complete" || in.History.FetchMode != "full5" {
		t.Fatal(err, in.History.Status)
	}
	if _, err = captureWithMode(root, []string{"file.go"}, time.Now(), "invalid"); err == nil {
		t.Fatal("invalid mode accepted")
	}
}
