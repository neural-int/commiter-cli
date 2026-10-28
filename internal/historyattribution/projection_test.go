package historyattribution

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProjectionPreservesAttributionAndSourceMessages(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "ok")
	in, err := capture(root, []string{"file.go"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	projected, err := projectionInput(in)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "projected.json")
	if err = writeNew(file, projected); err != nil {
		t.Fatal(err)
	}
	restored, err := loadProjectedInput(file, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(compare(in), compare(restored)) {
		t.Fatal("comparison changed")
	}
	if !reflect.DeepEqual(in.History.Operations, restored.History.Operations) || !reflect.DeepEqual(in.History.Turns, restored.History.Turns) {
		t.Fatal("source data changed")
	}
	if projected.History.Threads[0].Turns[0].UserMessages[0].ItemID != "U" {
		t.Fatal("request identity missing")
	}
	if projected.History.Threads[0].Turns[0].FinalMessages[0].ItemID != "A" {
		t.Fatal("final identity missing")
	}
	out := filepath.Join(t.TempDir(), "result.json")
	if err = execute([]string{"compare-projected", "--input", file, "--out", out}); err != nil {
		t.Fatal(err)
	}
}
func TestProjectionPreservesRenameMissingDiffAndIncomplete(t *testing.T) {
	in := renameInput(update("A", "B"), renameOp("move", update("A", "B")))
	started := int64(1)
	in.History.Turns = []TurnContext{{Thread: "S", Turn: "T", Request: "Request\n", Final: "Done", StartedAt: &started, Status: "completed", UserMessages: []ProjectionMessage{{ItemID: "U", ItemOrder: 1, Text: "Request\n"}}, FinalMessages: []ProjectionMessage{{ItemID: "A", ItemOrder: 3, Text: "Done"}}}}
	for _, missing := range []bool{false, true} {
		if missing {
			in.History.Operations[0].Diff = nil
		}
		in.History.Status = "partial"
		in.History.Dropped = 1
		in.History.Error = failure("history.filter", "sensitive_evidence_omitted").(*StageError)
		p, err := projectionInput(in)
		if err != nil {
			t.Fatal(err)
		}
		restored, err := p.input()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(compare(in), compare(restored)) {
			t.Fatal("partial/rename classification changed")
		}
	}
	in.History.Turns[0].UserMessages = nil
	if _, err := projectionInput(in); err == nil {
		t.Fatal("invented missing source identity")
	}
}
func TestProjectionRejectsExcludedFields(t *testing.T) {
	p := ProjectionInput{Version: 2, History: HistoryProjection{SchemaVersion: 2, Projection: "attribution"}}
	b, _ := json.Marshal(p)
	s := strings.Replace(string(b), `"projection":"attribution"`, `"projection":"attribution","aggregatedOutput":"excluded"`, 1)
	file := filepath.Join(t.TempDir(), "input.json")
	if err := writeNew(file, json.RawMessage(s)); err != nil {
		t.Fatal(err)
	}
	if _, err := loadProjectedInput(file, false); err == nil {
		t.Fatal("excluded field accepted")
	}
}
func TestProjectionWireCountersDoNotSavePayload(t *testing.T) {
	stats := &HistoryWireStats{}
	stats.observe(json.RawMessage(`{"data":[{"item":{"type":"commandExecution","aggregatedOutput":"PRIVATE_CONTENT"}},{"item":{"type":"agentMessage","phase":"commentary","text":"PRIVATE_CONTENT"}},{"item":{"type":"agentMessage","phase":"final_answer","text":"Done"}}]}`))
	b, _ := json.Marshal(stats)
	if strings.Contains(string(b), "PRIVATE_CONTENT") || stats.Items["commandExecution"].Count != 1 || stats.Items["intermediateMessage"].Count != 1 || stats.Items["finalMessage"].Count != 1 {
		t.Fatal("invalid aggregate counters")
	}
	root := gitFixture(t)
	fixtureServer(t, root, "ok")
	ctx := context.WithValue(context.Background(), historyStatsContext{}, stats)
	h := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if h.Status != "complete" || stats.Items["fileChange"].Count != 1 {
		t.Fatal("RPC stats not hooked")
	}
}

func TestProjectionRejectsOrphanOperations(t *testing.T) {
	in := fixed(update("A", "B"), op("1", update("A", "B")))
	if _, err := projectionInput(in); err == nil {
		t.Fatal("operation without source Turn was silently dropped")
	}
}
