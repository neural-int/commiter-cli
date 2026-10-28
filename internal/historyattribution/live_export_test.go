package historyattribution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Research-only harvesting. Every selected Turn is separately bounded at 5s/4MiB.
// Normal tests skip it; private exports contain no command output.
func TestExportLiveHistoryCases(t *testing.T) {
	if os.Getenv("COMMITER_ATTRIBUTION_LIVE_EXPORT") != "1" {
		t.Skip("explicit local-history opt in")
	}
	root, err := filepath.EvalSymlinks(os.Getenv("COMMITER_ATTRIBUTION_ROOT"))
	if err != nil {
		t.Fatal("root")
	}
	dir := os.Getenv("COMMITER_ATTRIBUTION_EXPORT")
	if dir == "" {
		t.Fatal("private directory required")
	}
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("directory must be new")
	}
	roots := []string{root}
	if extraRoot := os.Getenv("COMMITER_ATTRIBUTION_ADDITIONAL_ROOT"); extraRoot != "" {
		roots = append(roots, extraRoot)
	}
	type turnRef struct {
		Thread string
		Cwd    string
		Turn   historyTurn
	}
	refs := []turnRef{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	p, e := openHistory(ctx, root)
	if e != nil {
		cancel()
		t.Fatal(e)
	}
	initialize := func(p *historyRPC) error {
		var v map[string]json.RawMessage
		if e := p.call("history.initialize", "initialize", map[string]any{"clientInfo": map[string]string{"name": "attribution-research", "version": "1"}, "capabilities": map[string]bool{"experimentalApi": true}}, &v); e != nil {
			return e
		}
		return json.NewEncoder(p.in).Encode(map[string]string{"method": "initialized"})
	}
	if e = initialize(p); e != nil {
		p.close()
		cancel()
		t.Fatal(e)
	}
	var ts historyPage[historyThread]
	e = p.call("history.threads", "thread/list", map[string]any{"cwd": roots, "limit": 20, "useStateDbOnly": true, "sortKey": "updated_at"}, &ts)
	if e == nil {
		for _, th := range ts.Data {
			cursor := ""
			for page := 0; page < 4; page++ {
				args := map[string]any{"threadId": th.ID, "limit": 20, "sortDirection": "desc", "itemsView": "notLoaded"}
				if cursor != "" {
					args["cursor"] = cursor
				}
				var turns historyPage[historyTurn]
				e = p.call("history.turns", "thread/turns/list", args, &turns)
				if e != nil {
					break
				}
				for _, turn := range turns.Data {
					refs = append(refs, turnRef{th.ID, th.Cwd, turn})
				}
				if turns.Next == nil || *turns.Next == "" {
					break
				}
				cursor = *turns.Next
			}
			if len(refs) >= 80 || e != nil {
				break
			}
		}
	}
	p.close()
	cancel()
	if e != nil {
		t.Fatal(e)
	}
	if refFile := os.Getenv("COMMITER_ATTRIBUTION_REF_FILE"); refFile != "" {
		b, err := os.ReadFile(refFile)
		if err != nil || json.Unmarshal(b, &refs) != nil {
			t.Fatal("fixed refs")
		}
	}
	if err := writeNew(filepath.Join(dir, "selection.json"), refs); err != nil {
		t.Fatal(err)
	}
	distinct := map[string]bool{}
	exports := 0
	attempts := []map[string]any{}
	for n, ref := range refs {
		started := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		p, e := openHistory(ctx, root)
		if e != nil {
			cancel()
			t.Fatal(e)
		}
		e = initialize(p)
		h := History{Status: "complete"}
		record := TurnContext{Thread: ref.Thread, Turn: ref.Turn.ID}
		cursor := ""
		seen := map[string]bool{}
		if e == nil {
			for {
				args := map[string]any{"threadId": ref.Thread, "turnId": ref.Turn.ID, "limit": 20, "sortDirection": "asc"}
				if cursor != "" {
					args["cursor"] = cursor
				}
				var items historyPage[historyEntry]
				e = p.call("history.items", "thread/items/list", args, &items)
				if e != nil {
					break
				}
				if items.Data == nil {
					e = failure("history.items", "invalid_schema")
					break
				}
				for _, en := range items.Data {
					it := en.Item
					if en.TurnID != ref.Turn.ID || it.ID == "" {
						e = failure("history.items", "invalid_identity")
						break
					}
					if seen[it.ID] {
						continue
					}
					seen[it.ID] = true
					h.ItemsScanned++
					switch it.Type {
					case "userMessage":
						for _, part := range it.Content {
							if part.Type == "text" {
								record.Request += part.Text + "\n"
							}
						}
					case "agentMessage":
						if it.Phase == "final_answer" {
							record.Final = it.Text
						}
					case "fileChange":
						if it.Status != "completed" {
							continue
						}
						for i, ch := range it.Changes {
							path := normalizeHistoryPath(root, ref.Cwd, ch.Path)
							if filter := os.Getenv("COMMITER_ATTRIBUTION_PATHS"); filter != "" && !containsString(strings.Split(filter, ","), path) {
								continue
							}
							if path == "" || ch.Diff == nil || !safeText(*ch.Diff) {
								h.Dropped++
								continue
							}
							kind := ch.Kind.Type
							if ch.Kind.MovePath != nil {
								kind = "rename"
							}
							h.Operations = append(h.Operations, Operation{Thread: ref.Thread, Turn: ref.Turn.ID, Item: it.ID, Index: i, Path: path, Kind: kind, Diff: ch.Diff, Completed: true})
						}
					}
				}
				if e != nil {
					break
				}
				if items.Next == nil || *items.Next == "" {
					break
				}
				if cursor == *items.Next {
					e = failure("history.items", "cursor_repeat")
					break
				}
				cursor = *items.Next
			}
		}
		h.ReceivedBytes = receiveLimit + 1 - p.reader.N
		p.close()
		cancel()
		status := "complete"
		if e != nil {
			status = e.Error()
		} else if !safeText(record.Request) || !safeText(record.Final) || strings.TrimSpace(record.Request) == "" {
			status = "unsafe_or_missing_context"
		} else if h.Dropped > 0 {
			status = "filtered_operations"
		}
		attempts = append(attempts, map[string]any{"selection_index": n, "status": status, "operations": len(h.Operations), "received_bytes": h.ReceivedBytes, "elapsed_ms": float64(time.Since(started).Microseconds()) / 1000})
		if (status == "complete" || status == "filtered_operations") && len(h.Operations) > 0 {
			if status != "complete" {
				h.Status = "partial"
				h.Error = &StageError{Stage: "history.filter", Code: "sensitive_or_out_of_scope_operations_omitted"}
			}
			h.Turns = []TurnContext{record}
			h.TurnsScanned = 1
			h.ThreadsScanned = 1
			in := Input{Version: 1, Root: root, Head: "history-only-not-a-git-capture", History: h}
			name := filepath.Join(dir, "turn-"+time.Now().Format("150405.000000000")+".json")
			if err := writeNew(name, in); err != nil {
				t.Fatal(err)
			}
			exports++
			for _, o := range h.Operations {
				distinct[o.Path] = true
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "attempts.json"), mustJSON(attempts), 0600); err != nil {
			t.Fatal(err)
		}
		if len(distinct) >= 12 {
			break
		}
	}
	t.Logf("complete_turn_exports=%d distinct_safe_paths=%d metadata_refs=%d; private artifacts only", exports, len(distinct), len(refs))
}
func mustJSON(v any) []byte { b, _ := json.MarshalIndent(v, "", "  "); return append(b, '\n') }
