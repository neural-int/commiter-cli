package historyattribution

import (
	"context"
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"strings"
	"testing"
)

func TestCommandPathMentions(t *testing.T) {
	for _, tc := range []struct {
		command, cwd string
		want         bool
	}{
		{"cat file.go", "/local", true}, {"cat ./file.go", "/local", true}, {"cat '/local/file.go'", "/else", true},
		{"cat file.go", "/else", false}, {"cat file.go.bak", "/local", false}, {"cat other/file.go", "/local", false},
		{"cat /other/file.go", "/local", false}, {"echo \"file.go\"", "/local", true}, {"echo ./file.go.bak", "/local", false},
		{"cat $FILE", "/local", false}, {"cat fileXgo", "/local", false},
	} {
		t.Run(tc.command+tc.cwd, func(t *testing.T) {
			got := mentionedPaths("/local", tc.cwd, tc.command, map[string]bool{"file.go": true})
			if (len(got) > 0) != tc.want {
				t.Fatalf("mentions=%v", got)
			}
		})
	}
	if got := mentionedPaths("/local", "/local", "cat 'src/a[b].go'", map[string]bool{"src/a[b].go": true}); len(got) != 1 {
		t.Fatal(got)
	}
}

func TestCommandOnlyTurnAndDedup(t *testing.T) {
	req := pipelineRequest()
	old := "old.go"
	req.Files[0].OldPath = &old
	cmd := CommandExecution{ItemID: "C", Command: "sed -i x old.go file.go", Cwd: "/local", Status: "failed"}
	h := History{Status: "complete", Turns: []TurnContext{{Thread: "S", Turn: "T", Status: "completed", Request: "Change retry", Final: "Done", Commands: []CommandExecution{cmd, cmd}}}}
	out, err := attribute(req, h)
	if err != nil {
		t.Fatal(err)
	}
	before := out.Files[0].Blocks[0].Status
	sources := map[string]ContextSource{}
	cs := map[string]CommandSource{}
	appendCommandContexts(req, h, out, sources, cs)
	if len(out.Commands) != 1 || len(out.Contexts) != 1 || len(out.Commands[0].MentionedFileIDs) != 1 || len(cs) != 1 || len(sources) != 1 {
		t.Fatalf("bad command-only projection: %+v", out)
	}
	if out.Files[0].Blocks[0].Status != before || len(out.Files[0].Blocks[0].Candidates) != 0 {
		t.Fatal("commands modified content attribution")
	}
	// Equal command text under a different source identity remains a separate execution.
	h.Turns[0].Commands[1].ItemID = "C2"
	fresh := &planning.GroupingHistory{}
	appendCommandContexts(req, h, fresh, map[string]ContextSource{}, map[string]CommandSource{})
	if len(fresh.Commands) != 2 || len(fresh.Contexts) != 1 {
		t.Fatal("different executions collapsed")
	}
	h.Turns[0].Commands[1].ItemID = "C"
	h.Turns[0].Commands[1].Command = "cat file.go"
	fresh = &planning.GroupingHistory{}
	appendCommandContexts(req, h, fresh, map[string]ContextSource{}, map[string]CommandSource{})
	if len(fresh.Commands) != 0 || fresh.Acquisition.Status != "partial" {
		t.Fatal("conflicting commands accepted")
	}
	req.SafeText = func(s string) bool { return !strings.Contains(s, "sed") }
	h.Turns[0].Commands = []CommandExecution{cmd}
	fresh = &planning.GroupingHistory{}
	appendCommandContexts(req, h, fresh, map[string]ContextSource{}, map[string]CommandSource{})
	if len(fresh.Commands) != 0 {
		t.Fatal("unsafe command accepted")
	}
}

func TestReadCommandOnlyTurnWithoutFileChange(t *testing.T) {
	var items []historyItem
	raw := `[{"type":"userMessage","id":"U","content":[{"type":"text","text":"Inspect retry behavior"}]},{"type":"commandExecution","id":"C","command":"cat file.go","cwd":"/local","status":"failed","exitCode":1},{"type":"agentMessage","id":"A","phase":"final_answer","text":"Inspected"}]`
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatal(err)
	}
	turn := historyTurn{ID: "T", Status: "completed", Items: &items}
	config := fetchConfig{itemsView: "full"}
	h := History{Status: "complete"}
	ctx := context.WithValue(context.Background(), commandHistoryContext{}, true)
	if err := readHistoryTurn(ctx, nil, "/local", historyThread{ID: "S", Cwd: "/local"}, turn, map[string]bool{"file.go": true}, config, &h, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if len(h.Turns) != 1 || len(h.Operations) != 0 || len(h.Turns[0].Commands) != 1 || h.Turns[0].Commands[0].ItemOrder != 2 || h.Turns[0].Commands[0].ExitCode == nil || *h.Turns[0].Commands[0].ExitCode != 1 {
		t.Fatal("command-only turn or source metadata lost")
	}
}
