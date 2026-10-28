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

func TestHistoryTimingPreservesSourceAndSeparatesPhases(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "fetch_modes")
	t.Setenv("POC_EXPECT_FETCH", "paged100")
	ctx := context.WithValue(context.Background(), historyFetchModeContext{}, "paged100")
	baseline := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	profile := &HistoryTiming{}
	ctx = context.WithValue(ctx, historyTimingContext{}, profile)
	measured := readHistory(ctx, root, []string{root}, map[string]bool{"file.go": true}, 0, time.Now().Unix())
	if !reflect.DeepEqual(baseline.Operations, measured.Operations) || !reflect.DeepEqual(baseline.Turns, measured.Turns) {
		t.Fatal("timing changed selected source")
	}
	if len(profile.Calls) != measured.Requests || profile.TotalMS <= 0 || profile.SpawnMS <= 0 || profile.ItemSelectionMS <= 0 || profile.ContextValidationMS <= 0 || profile.PathMS <= 0 || profile.SafetyMS <= 0 || profile.CleanupMS <= 0 {
		t.Fatal("missing phases")
	}
	var sum float64
	for _, call := range profile.Calls {
		phases := call.SendMS + call.WaitReadMS + call.EnvelopeDecodeMS + call.ResultDecodeMS
		if call.TotalMS < phases || call.Failed || call.ResponseBytes <= 0 {
			t.Fatal("overlapping/missing RPC phases")
		}
		sum += call.TotalMS
	}
	if profile.TotalMS < sum+profile.SpawnMS+profile.ItemSelectionMS+profile.ContextValidationMS+profile.CleanupMS {
		t.Fatal("top level phases overlap")
	}
	b, _ := json.Marshal(profile)
	for _, value := range []string{root, "Change A to B", "Changed A to B", "file.go", "progress100"} {
		if strings.Contains(string(b), value) {
			t.Fatal("source data leaked into timing")
		}
	}
}
func TestTimingCLIPrivateOutput(t *testing.T) {
	root := gitFixture(t)
	fixtureServer(t, root, "ok")
	dir := t.TempDir()
	capture := filepath.Join(dir, "input.json")
	log := filepath.Join(dir, "timing.json")
	if err := execute([]string{"capture", "--root", root, "--path", "file.go", "--out", capture, "--history-fetch", "paged100", "--history-timing-log", log}); err != nil {
		t.Fatal(err)
	}
	if err := execute([]string{"capture", "--root", root, "--path", "file.go", "--out", filepath.Join(dir, "another.json"), "--history-fetch", "paged100", "--history-timing-log", log}); err == nil {
		t.Fatal("timing overwrite allowed")
	}
}
