package codexcontext

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

type sequenceItem struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Phase   string `json:"phase"`
	Text    string `json:"text"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Changes []struct {
		Path string `json:"path"`
		Diff string `json:"diff"`
		Kind struct {
			Type string `json:"type"`
		} `json:"kind"`
	} `json:"changes"`
}
type savedEdit struct {
	turn, path, kind, added, removed string
	order                            int
	hasDiff                          bool
}
type savedTurn struct {
	user, final            bool
	complete               bool
	requestText, finalText string
}

func TestSavedHistorySequenceProbe(t *testing.T) {
	if os.Getenv("COMMITER_CODEX_SEQUENCE_PROBE") != "1" {
		t.Skip("opt-in history sequence inspection")
	}
	root := os.Getenv("COMMITER_CODEX_DIFF_ROOT")
	if root == "" {
		t.Fatal("explicit root required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	begin := time.Now()
	p, err := start(ctx, root)
	if err != nil {
		t.Fatal("transport unavailable")
	}
	defer p.close()
	turnsSeen := map[string]*savedTurn{}
	edits := []savedEdit{}
	counts := map[string]int{}
	finish := func(status string) {
		byPath := map[string][]savedEdit{}
		for _, e := range edits {
			byPath[e.path] = append(byPath[e.path], e)
		}
		candidates := []map[string]any{}
		for _, events := range byPath {
			if len(events) < 2 {
				continue
			}
			sort.SliceStable(events, func(i, j int) bool { return events[i].order < events[j].order })
			ids := map[string]bool{}
			allUser, allFinal := true, true
			links, reversals := 0, 0
			for i, e := range events {
				ids[e.turn] = true
				tr := turnsSeen[e.turn]
				allUser = allUser && tr.user
				allFinal = allFinal && tr.final
				if i > 0 {
					prev := events[i-1]
					if prev.added != "" && prev.added == e.removed {
						links++
					}
					if prev.added != "" && prev.removed != "" && prev.added == e.removed && prev.removed == e.added {
						reversals++
					}
				}
			}
			candidates = append(candidates, map[string]any{"edit_events": len(events), "turns": len(ids), "all_turns_have_user_request": allUser, "all_turns_have_final_response": allFinal, "adjacent_added_equals_next_removed": links, "exact_reverse_edit_pairs": reversals})
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i]["edit_events"].(int) > candidates[j]["edit_events"].(int) })
		result := map[string]any{"status": status, "counts": counts, "turns_observed": len(turnsSeen), "completed_edit_events": len(edits), "repeated_path_candidates": candidates, "received_bytes": MaxReceiveBytes + 1 - p.reader.N, "elapsed_ms": float64(time.Since(begin).Microseconds()) / 1000, "limit": "item order retained within each turn; turn order uses startedAt; no Git comparison; changed-line equality is not complete file-state proof"}

		if os.Getenv("COMMITER_CODEX_SEQUENCE_DETAIL") == "1" {
			clean := func(text string, limit int) string {
				for _, pattern := range []string{`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`, `(?i)(?:gh[pousr]_|sk-)[A-Za-z0-9_-]{16,}`, `/Users/[^\s"']+`} {
					text = regexp.MustCompile(pattern).ReplaceAllString(text, "[REDACTED]")
				}
				if len(text) > limit {
					text = text[:limit]
					for !utf8.ValidString(text) {
						text = text[:len(text)-1]
					}
				}
				return text
			}
			selected := []map[string]any{}
			pickedSingle, pickedMulti := false, false
			// Stable selection by path; keep one single-turn and one multi-turn case.
			names := []string{}
			for name, es := range byPath {
				if len(es) > 1 {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			for _, name := range names {
				es := byPath[name]
				sort.SliceStable(es, func(i, j int) bool { return es[i].order < es[j].order })
				ids := map[string]bool{}
				for _, e := range es {
					ids[e.turn] = true
				}
				multi := len(ids) > 1
				if (multi && pickedMulti) || (!multi && pickedSingle) {
					continue
				}
				if multi {
					pickedMulti = true
				} else {
					pickedSingle = true
				}
				timeline := []map[string]any{}
				turnLabels := map[string]string{}
				turnDetails := []map[string]any{}
				for _, e := range es {
					label, exists := turnLabels[e.turn]
					if !exists {
						label = string(rune('A' + len(turnLabels)))
						turnLabels[e.turn] = label
						tr := turnsSeen[e.turn]
						turnDetails = append(turnDetails, map[string]any{"turn": label, "all_items_read": tr.complete, "request": clean(tr.requestText, 650), "final_response": clean(tr.finalText, 850)})
					}
					timeline = append(timeline, map[string]any{"turn": label, "kind": e.kind, "removed_excerpt": clean(e.removed, 200), "added_excerpt": clean(e.added, 200), "added_bytes": len(e.added), "removed_bytes": len(e.removed)})
				}
				selected = append(selected, map[string]any{"path": clean(name, 180), "turns": turnDetails, "edits_in_order": timeline})
			}
			detail := map[string]any{"selected_examples": selected, "privacy": "only bounded excerpts; no commands/output; private local inspection"}
			b, _ := json.Marshal(detail)
			if len(b) > 8192 {
				t.Log("detail exceeds 8KiB; withheld")
			} else {
				t.Log(string(b))
			}
		}
		b, _ := json.Marshal(result)
		t.Log(string(b))
	}
	if p.initialize() != nil {
		finish("initialize_failed")
		return
	}
	var threads page[thread]
	if p.call("thread/list", map[string]any{"cwd": []string{root}, "limit": PageSize, "useStateDbOnly": true, "sortKey": "updated_at"}, &threads) != nil {
		finish("thread_list_failed")
		return
	}
	if len(threads.Data) == 0 {
		finish("no_threads")
		return
	}
	th := threads.Data[0]
	cursor := ""
	for {
		args := map[string]any{"threadId": th.ID, "limit": PageSize, "sortDirection": "desc", "itemsView": "notLoaded"}
		if cursor != "" {
			args["cursor"] = cursor
		}
		var turns page[turn]
		if p.call("thread/turns/list", args, &turns) != nil {
			finish("turn_budget_or_failure")
			return
		}
		for _, tr := range turns.Data {
			meta := &savedTurn{}
			turnsSeen[tr.ID] = meta
			ic := ""
			order := 0
			for {
				ia := map[string]any{"threadId": th.ID, "turnId": tr.ID, "limit": PageSize, "sortDirection": "asc"}
				if ic != "" {
					ia["cursor"] = ic
				}
				var items page[struct {
					Item sequenceItem `json:"item"`
				}]
				if p.call("thread/items/list", ia, &items) != nil {
					finish("item_budget_or_failure")
					return
				}
				for _, entry := range items.Data {
					it := entry.Item
					counts[it.Type]++
					order++
					if it.Type == "userMessage" {
						for _, c := range it.Content {
							if c.Type == "text" && c.Text != "" {
								meta.user = true
								meta.requestText += c.Text + "\n"
							}
						}
					}
					if it.Type == "agentMessage" && it.Phase == "final_answer" && it.Text != "" {
						meta.final = true
						meta.finalText += it.Text + "\n"
					}
					if it.Type != "fileChange" || it.Status != "completed" {
						continue
					}
					for _, ch := range it.Changes {
						lower := strings.ToLower(ch.Path)
						if strings.Contains(lower, ".env") || strings.Contains(lower, "key") || strings.Contains(lower, "credential") || strings.Contains(lower, "secret") {
							continue
						}
						add, del := []string{}, []string{}
						for _, line := range strings.Split(ch.Diff, "\n") {
							if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
								add = append(add, line[1:])
							}
							if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
								del = append(del, line[1:])
							}
						}
						if ch.Kind.Type == "add" {
							add = []string{ch.Diff}
							del = nil // Added files contain raw text; Markdown bullets are not deletion records.
						}
						stamp := 0
						if tr.Started != nil {
							stamp = int(*tr.Started)
						}
						edits = append(edits, savedEdit{turn: tr.ID, path: ch.Path, kind: ch.Kind.Type, added: strings.Join(add, "\n"), removed: strings.Join(del, "\n"), order: stamp*100000 + order, hasDiff: ch.Diff != ""})
					}
				}
				if items.Next == nil || *items.Next == "" {
					meta.complete = true
					break
				}
				if ic == *items.Next {
					finish("cursor_repeat")
					return
				}
				ic = *items.Next
			}
		}
		if turns.Next == nil || *turns.Next == "" {
			break
		}
		if cursor == *turns.Next {
			finish("cursor_repeat")
			return
		}
		cursor = *turns.Next
	}
	finish("complete_first_thread")
}
