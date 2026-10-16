package codexcontext

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// These are projections of local historical events, not captured RPC responses
// or semantic gold. No app server, source log, network, or model is needed.
func TestHistoricalCandidateReplay(t *testing.T) {
	// These expectations cover path-based association. The original block-match
	// fallback remains documented in the fixture README; semantic gold is unchanged.
	expectedPaths := map[string][]string{
		"image-static-import-fix":     {"src/components/common/progressiveimage/ProgressiveImage.jsx", "src/components/common/svg/AISummaryBadge.jsx", "src/lib/staticAsset.js"},
		"hook-dependency-maintenance": {"src/components/ui/dia-text-reveal.jsx"},
		"file-first-feature":          {"docs/decisions/issue-167-file-first.md", "internal/gitstate/collect.go", "internal/gitstate/file_first.go", "internal/gitstate/file_first_test.go"},
		"motion-feature":              {"assets/css/style.css", "assets/js/script.js", "index.html"},
		"remove-form-note":            {"index.html"},
	}
	dir := filepath.Join("testdata", "history_candidates")
	var manifest struct {
		Cases []struct {
			ID     string `json:"id"`
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
		} `json:"cases"`
	}
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range manifest.Cases {
		t.Run(candidate.ID, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, candidate.File))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			if hex.EncodeToString(sum[:]) != candidate.SHA256 {
				t.Fatal("candidate changed without updating its manifest")
			}
			var fixture struct {
				Turns []struct {
					ID        string           `json:"id"`
					StartedAt int64            `json:"started_at"`
					Items     []map[string]any `json:"items"`
				} `json:"turns"`
				Window struct {
					Start int64 `json:"start"`
					End   int64 `json:"end"`
				} `json:"window"`
				Targets []struct {
					Path      string `json:"path"`
					Diff      string `json:"diff"`
					FinalDiff string `json:"final_diff"`
				} `json:"targets"`
			}
			if err := json.Unmarshal(raw, &fixture); err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			client := historicalCandidateCaller{
				historyFixture: historyFixture{root: root, items: map[string][]map[string]any{}},
				updated:        fixture.Window.End,
			}
			for _, turn := range fixture.Turns {
				client.turns = append(client.turns, map[string]any{"id": turn.ID, "startedAt": turn.StartedAt})
				client.items[turn.ID] = turn.Items
			}
			request := Request{
				Root: root, Start: time.Unix(fixture.Window.Start, 0), End: time.Unix(fixture.Window.End, 0),
				SafeText:  func(string) bool { return true },
				AllowPath: func(path string) bool { return !strings.HasPrefix(path, ".env") },
			}
			for _, target := range fixture.Targets {
				request.Targets = append(request.Targets, Target{Path: target.Path})
			}
			result := collect(&client, request)
			var paths []string
			for _, record := range result.Records {
				paths = append(paths, record.Paths...)
			}
			sort.Strings(paths)
			t.Logf("status=%s records=%d adopted_paths=%q", result.Status, len(result.Records), paths)
			want, ok := expectedPaths[candidate.ID]
			if !ok {
				t.Fatal("candidate has no reviewed collection expectation")
			}
			wantStatus, wantRecords := "available", 1
			if len(want) == 0 {
				wantStatus, wantRecords = "no_related_evidence", 0
			}
			if result.Status != wantStatus || len(result.Records) != wantRecords || !reflect.DeepEqual(paths, want) {
				t.Fatalf("collection changed: got status=%s records=%d paths=%q; want status=%s records=%d paths=%q", result.Status, len(result.Records), paths, wantStatus, wantRecords, want)
			}
		})
	}
}

type historicalCandidateCaller struct {
	historyFixture
	updated int64
}

func (f *historicalCandidateCaller) call(method string, params any, result any) error {
	if method != "thread/list" {
		return f.historyFixture.call(method, params, result)
	}
	f.calls = append(f.calls, method)
	threads := []map[string]any{}
	if params.(map[string]any)["archived"] == false {
		threads = append(threads, map[string]any{"id": "historical-thread", "cwd": f.root, "updatedAt": f.updated})
	}
	raw, err := json.Marshal(map[string]any{"data": threads, "nextCursor": nil})
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}
