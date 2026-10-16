package codexcontext

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testDiff = "@@ -1 +1 @@\n-old\n+new\n"

type historyFixture struct {
	root     string
	turns    []map[string]any
	items    map[string][]map[string]any
	failTurn string
	calls    []string
	archived bool
}

func (f *historyFixture) call(method string, params any, result any) error {
	f.calls = append(f.calls, method)
	p := params.(map[string]any)
	var data any
	switch method {
	case "thread/list":
		if p["limit"] != PageSize || p["useStateDbOnly"] != true {
			return errors.New("unbounded listing")
		}
		data = []map[string]any{}
		if p["archived"] == f.archived {
			data = []map[string]any{{"id": "thread", "cwd": f.root, "updatedAt": 150}}
		}
	case "thread/turns/list":
		data = f.turns
	case "thread/items/list":
		id := p["turnId"].(string)
		if id == f.failTurn {
			return errors.New("permission denied")
		}
		data = f.items[id]
		if data == nil {
			data = []map[string]any{}
		}
	default:
		return errors.New("unexpected mutating method")
	}
	raw, _ := json.Marshal(map[string]any{"data": data, "nextCursor": nil})
	return json.Unmarshal(raw, result)
}

func testHistory(t *testing.T) (*historyFixture, Request) {
	t.Helper()
	root := t.TempDir()
	f := &historyFixture{root: root, turns: []map[string]any{{"id": "turn", "startedAt": 125}}, items: map[string][]map[string]any{}}
	f.items["turn"] = []map[string]any{
		logItem("turn", "request", "userMessage", map[string]any{"content": []map[string]string{{"type": "text", "text": "Correct the boundary"}}}),
		editItem("turn", "patch", "one.go", testDiff, "completed"),
		logItem("turn", "progress", "agentMessage", map[string]any{"phase": "commentary", "text": "unverified hypothesis"}),
		logItem("turn", "answer", "agentMessage", map[string]any{"phase": "final_answer", "text": "Corrected the boundary"}),
	}
	r := Request{Root: root, Start: time.Unix(100, 0), End: time.Unix(200, 0), Targets: []Target{{Path: "one.go"}}, SafeText: func(s string) bool { return !strings.Contains(s, "PRIVATE") }, AllowPath: func(s string) bool { return !strings.HasPrefix(s, ".env") }}
	return f, r
}

func logItem(turn, id, kind string, fields map[string]any) map[string]any {
	i := map[string]any{"id": id, "type": kind}
	for k, v := range fields {
		i[k] = v
	}
	return map[string]any{"turnId": turn, "item": i}
}
func editItem(turn, id, path, diff, status string) map[string]any {
	return logItem(turn, id, "fileChange", map[string]any{"status": status, "changes": []map[string]any{{"path": path, "kind": map[string]string{"type": "update"}, "diff": diff}}})
}

func TestCollectUsesDirectEditsAndDeduplicatesSharedRequest(t *testing.T) {
	f, r := testHistory(t)
	r.Targets = append(r.Targets, Target{Path: "two.go"})
	f.items["turn"] = append(f.items["turn"], editItem("turn", "patch2", filepath.Join(r.Root, "two.go"), testDiff, "completed"))
	// Replayed inherited turn/item IDs must not duplicate evidence.
	f.turns = append(f.turns, f.turns[0])
	f.items["turn"] = append(f.items["turn"], f.items["turn"][0])
	result := collect(f, r)
	if result.Status != "available" || len(result.Records) != 1 {
		t.Fatalf("status=%s records=%d", result.Status, len(result.Records))
	}
	got := result.Records[0]
	if strings.Join(got.Paths, ",") != "one.go,two.go" || got.UserRequest != "Correct the boundary" || got.FinalResponse != "Corrected the boundary" {
		t.Fatal("incorrect or duplicated projection")
	}
	raw, _ := json.Marshal(result.Records)
	if strings.Contains(string(raw), "unverified") || strings.Contains(string(raw), "turnId") {
		t.Fatal("internal/progress fields leaked")
	}
}

func TestCollectFallsBackForUnrelatedFailedCancelledAndUnsafeEdits(t *testing.T) {
	for _, name := range []string{"other_path", "other_worktree", "failed", "cancelled", "unknown_time", "old_turn", "future_turn", "private_request", "private_edit"} {
		t.Run(name, func(t *testing.T) {
			f, r := testHistory(t)
			switch name {
			case "other_path":
				f.items["turn"][1] = editItem("turn", "patch", "other.go", testDiff, "completed")
			case "other_worktree":
				f.items["turn"][1] = editItem("turn", "patch", filepath.Join(filepath.Dir(r.Root), "another", "one.go"), testDiff, "completed")
			case "failed", "cancelled":
				f.items["turn"][1] = editItem("turn", "patch", "one.go", testDiff, name)
			case "unknown_time":
				delete(f.turns[0], "startedAt")
			case "old_turn":
				f.turns[0]["startedAt"] = 99
			case "future_turn":
				f.turns[0]["startedAt"] = 201
			case "private_request":
				r.SafeText = func(string) bool { return false }
			case "private_edit":
				f.items["turn"] = append(f.items["turn"], editItem("turn", "private", ".env", testDiff, "completed"))
			}
			result := collect(f, r)
			if result.Status != "no_related_evidence" || len(result.Records) != 0 {
				t.Fatalf("status=%s records=%d", result.Status, len(result.Records))
			}
		})
	}
}

func TestSessionInSiblingWorktreeMustEditSelectedWorktree(t *testing.T) {
	f, r := testHistory(t)
	sessionRoot := f.root
	r.Root = t.TempDir()
	r.SessionRoots = []string{sessionRoot}
	f.items["turn"][1] = editItem("turn", "patch", filepath.Join(r.Root, "one.go"), testDiff, "completed")
	result := collect(f, r)
	if len(result.Records) != 1 {
		t.Fatal("main-checkout session could not supply selected worktree evidence")
	}
	f.items["turn"][1] = editItem("turn", "patch", filepath.Join(sessionRoot, "one.go"), testDiff, "completed")
	if len(collect(f, r).Records) != 0 {
		t.Fatal("sibling worktree edit adopted")
	}
	f.items["turn"][1] = editItem("turn", "patch", "one.go", testDiff, "completed")
	if len(collect(f, r).Records) != 0 {
		t.Fatal("relative path interpreted against wrong worktree")
	}
}

func TestCollectAllowsActiveAndArchivedTurnsWithoutFinalAnswer(t *testing.T) {
	f, r := testHistory(t)
	f.archived = true
	f.items["turn"] = f.items["turn"][:2]
	result := collect(f, r)
	if result.Status != "available" || len(result.Records) != 1 || result.Records[0].FinalResponse != "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestCollectPartialFailureDoesNotAdoptIncompleteTurn(t *testing.T) {
	f, r := testHistory(t)
	f.failTurn = "turn"
	result := collect(f, r)
	if result.Status != "partial" || len(result.Records) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestCollectKeepsMultipleRequestsForSamePath(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		f, r := testHistory(t)
		f.turns = append(f.turns, map[string]any{"id": "another", "startedAt": 130})
		path := "two.go"
		if conflict {
			path = "one.go"
		}
		r.Targets = append(r.Targets, Target{Path: path})
		f.items["another"] = []map[string]any{logItem("another", "request2", "userMessage", map[string]any{"content": []map[string]string{{"type": "text", "text": "Correct the boundary"}}}), editItem("another", "patch2", path, testDiff, "completed")}
		result := collect(f, r)
		want := 2
		if len(result.Records) != want {
			t.Fatalf("conflict=%v records=%d", conflict, len(result.Records))
		}
	}
}

func TestContextBudgetKeepsRequestAndOmitsLargeFinalWithoutTruncating(t *testing.T) {
	f, r := testHistory(t)
	f.items["turn"][3] = logItem("turn", "answer", "agentMessage", map[string]any{"phase": "final_answer", "text": strings.Repeat("あ", MaxContextBytes)})
	result := collect(f, r)
	if len(result.Records) != 1 || result.Records[0].FinalResponse != "" || result.Records[0].UserRequest != "Correct the boundary" {
		t.Fatal("optional text budget changed request")
	}
	encoded, _ := json.Marshal(map[string]any{"codex_context": result.Records})
	if len(encoded) > MaxContextBytes {
		t.Fatal("context exceeds byte cap")
	}
}

func TestCollectUsesCompletedEditedPathRegardlessOfHistoricalContent(t *testing.T) {
	for _, diff := range []string{"@@ -1 +1 @@\n-other\n+intermediate\n", ""} {
		f, r := testHistory(t)
		f.items["turn"][1] = editItem("turn", "patch", "one.go", diff, "completed")
		result := collect(f, r)
		if result.Status != "available" || len(result.Records) != 1 || strings.Join(result.Records[0].Paths, ",") != "one.go" {
			t.Fatalf("path-related history rejected: status=%s records=%d", result.Status, len(result.Records))
		}
	}
}
