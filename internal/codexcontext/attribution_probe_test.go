package codexcontext

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// Read-only research probe. Never exports history text, paths, or source IDs.
func TestSavedDiffAvailabilityProbe(t *testing.T) {
	if os.Getenv("COMMITER_CODEX_DIFF_PROBE") != "1" {
		t.Skip("opt-in saved history probe")
	}
	root := os.Getenv("COMMITER_CODEX_DIFF_ROOT")
	if root == "" {
		t.Fatal("explicit root required")
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	p, err := start(ctx, root)
	if err != nil {
		t.Fatal("transport unavailable")
	}
	defer p.close()
	stats := map[string]any{"status": "partial", "threads": 0, "turns": 0, "items": 0, "file_changes": 0, "changes": 0, "diff_present": 0, "diff_nonempty": 0, "unified": 0, "other_format": 0, "missing_diff": 0}
	counts := map[string]int{}
	statuses := map[string]int{}
	kinds := map[string]int{}
	finish := func(status string) {
		stats["status"] = status
		for k, v := range counts {
			stats[k] = v
		}
		stats["statuses"] = statuses
		stats["kinds"] = kinds
		stats["received_bytes"] = MaxReceiveBytes + 1 - p.reader.N
		stats["elapsed_ms"] = float64(time.Since(started).Microseconds()) / 1000
		b, _ := json.Marshal(stats)
		t.Log(string(b))
	}
	if p.initialize() != nil {
		finish("initialize_failed")
		return
	}
	for _, archived := range []bool{false, true} {
		cursor := ""
		for {
			args := map[string]any{"cwd": []string{root}, "archived": archived, "limit": PageSize, "useStateDbOnly": true, "sortKey": "updated_at"}
			if cursor != "" {
				args["cursor"] = cursor
			}
			var threads page[thread]
			if p.call("thread/list", args, &threads) != nil {
				finish("thread_list_failed_or_budget")
				return
			}
			for _, th := range threads.Data {
				counts["threads"]++
				tc := ""
				for {
					ta := map[string]any{"threadId": th.ID, "limit": PageSize, "sortDirection": "desc", "itemsView": "notLoaded"}
					if tc != "" {
						ta["cursor"] = tc
					}
					var turns page[turn]
					if p.call("thread/turns/list", ta, &turns) != nil {
						finish("turn_list_failed_or_budget")
						return
					}
					for _, tr := range turns.Data {
						counts["turns"]++
						ic := ""
						for {
							ia := map[string]any{"threadId": th.ID, "turnId": tr.ID, "limit": PageSize, "sortDirection": "asc"}
							if ic != "" {
								ia["cursor"] = ic
							}
							var items page[struct {
								Item struct {
									Type    string                       `json:"type"`
									Status  string                       `json:"status"`
									Changes []map[string]json.RawMessage `json:"changes"`
								} `json:"item"`
							}]
							if p.call("thread/items/list", ia, &items) != nil {
								finish("item_list_failed_or_budget")
								return
							}
							for _, entry := range items.Data {
								counts["items"]++
								it := entry.Item
								if it.Type != "fileChange" {
									continue
								}
								counts["file_changes"]++
								statuses[it.Status]++
								for _, change := range it.Changes {
									counts["changes"]++
									var kind struct {
										Type string `json:"type"`
									}
									_ = json.Unmarshal(change["kind"], &kind)
									kinds[kind.Type]++
									raw, ok := change["diff"]
									if !ok {
										counts["missing_diff"]++
										continue
									}
									counts["diff_present"]++
									var diff string
									if json.Unmarshal(raw, &diff) != nil {
										counts["other_format"]++
										continue
									}
									if diff == "" {
										continue
									}
									counts["diff_nonempty"]++
									if strings.Contains(diff, "@@") {
										counts["unified"]++
									} else {
										counts["other_format"]++
									}
								}
							}
							if items.Next == nil || *items.Next == "" {
								break
							}
							if ic == *items.Next {
								finish("repeated_cursor")
								return
							}
							ic = *items.Next
						}
					}
					if turns.Next == nil || *turns.Next == "" {
						break
					}
					if tc == *turns.Next {
						finish("repeated_cursor")
						return
					}
					tc = *turns.Next
				}
			}
			if threads.Next == nil || *threads.Next == "" {
				break
			}
			if cursor == *threads.Next {
				finish("repeated_cursor")
				return
			}
			cursor = *threads.Next
		}
	}
	finish("complete")
}
