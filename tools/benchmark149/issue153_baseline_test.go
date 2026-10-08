package main

import (
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"os"
	"testing"
)

func TestIssue153PinnedProjection(t *testing.T) {
	if os.Getenv("ISSUE153_BASELINE") != "1" {
		t.Skip("explicit local inference only")
	}
	var input struct {
		Files []struct {
			ID     string `json:"id"`
			Path   string `json:"path"`
			Before string `json:"before"`
			After  string `json:"after"`
		}
	}
	data, e := os.ReadFile("../benchmark153/multifile-fixture.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(data, &input); e != nil {
		t.Fatal(e)
	}
	var diffs map[string]string
	data, e = os.ReadFile("../../docs/benchmarks/issue-153/iteration-18-projection-diffs.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(data, &diffs); e != nil {
		t.Fatal(e)
	}
	f := fixture{Name: "issue153-four-file-projection", Expected: [][]string{{"F001", "F002"}, {"F003", "F004"}}}
	for _, x := range input.Files[:4] {
		path := x.Path
		f.Files = append(f.Files, contextinput.File{ID: x.ID, NewPath: &path, Status: "modified", Language: "go", RawDiff: diffs[x.ID]})
	}
	v := config.Defaults().Values
	p, e := (mlxmodel.Store{Root: "/Users/Natsuki/Library/Caches/commiter/mlx-models"}).Ready(mlxmodel.Spec{Repo: v.Model, Revision: v.ModelRevision, Quantization: v.ModelQuantization})
	if e != nil {
		t.Fatal("pinned cached baseline unavailable")
	}
	b := &measuredBackend{Executable: "/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper", Model: v.Model, Revision: v.ModelRevision, Path: p}
	result := baselineMode(f, b, true)
	data, e = json.MarshalIndent(result, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile("../../docs/benchmarks/issue-153/iteration-18-baseline-results.json", append(data, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
