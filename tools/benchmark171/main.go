// Research-only: capture the real ThreePhaseGenerator first request and stop.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type fixtureFile struct {
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type fixture struct {
	Name  string        `json:"name"`
	Files []fixtureFile `json:"files"`
}

var captureStop = errors.New("first request captured; no backend invocation")

type captureClient struct {
	Calls   int
	Request mlx.Request
}

func (c *captureClient) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("explicit profile required")
}
func (c *captureClient) ChatWithOptions(_ context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	c.Calls++
	c.Request = mlx.Request{Messages: m, Schema: s, Model: "mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2", ModelPath: "<local-model>", ContextTokens: o.ContextTokens, OutputTokens: o.OutputTokens, GenerationProfile: o.GenerationProfile}
	return llm.Response{}, captureStop
}
func must(e error) {
	if e != nil {
		panic(e)
	}
}
func main() {
	manifest := flag.String("fixtures", "tools/benchmark171/fixtures.json", "fixtures")
	out := flag.String("out", "", "new output directory")
	runtime := flag.String("runtime", "", "local roots receipt")
	flag.Parse()
	if *out == "" || *runtime == "" {
		panic("explicit output/runtime required")
	}
	if _, e := os.Stat(*out); !os.IsNotExist(e) {
		panic("never overwrite captures")
	}
	must(os.MkdirAll(*out, 0700))
	data, e := os.ReadFile(*manifest)
	must(e)
	var fx []fixture
	must(json.Unmarshal(data, &fx))
	roots := map[string]string{}
	mapping := map[string]any{}
	rows := []map[string]any{}
	for _, f := range fx {
		root, e := makeRepository(f)
		must(e)
		roots[f.Name] = root
		state, e := gitstate.Collect(root, gitstate.Options{})
		must(e)
		p, _, _, e := prepare(root, state)
		must(e)
		initial, e := repositoryDigest(root)
		must(e)
		preparedHash := jsonDigest(p)
		snapshotHash := jsonDigest(state)
		c := &captureClient{}
		_, e = (planning.ThreePhaseGenerator{Client: c}).Generate(context.Background(), p, planning.English, planning.SensitiveValues{})
		if e == nil || c.Calls != 1 {
			b, _ := json.Marshal(p)
			_ = os.WriteFile(filepath.Join(*out, f.Name+"-rejected-prepared.json"), b, 0600)
			panic(fmt.Sprintf("capture %s calls=%d stopped before grouping: %v", f.Name, c.Calls, e))
		}
		raw, e := json.Marshal(c.Request)
		must(e)
		must(os.WriteFile(filepath.Join(*out, f.Name+"-baseline.json"), raw, 0600))
		var payload struct {
			Files []json.RawMessage `json:"files"`
		}
		must(json.Unmarshal([]byte(c.Request.Messages[1].Content), &payload))
		changed := 0
		for _, ff := range f.Files {
			if ff.Before != ff.After {
				changed++
			}
		}
		if len(payload.Files) != changed || len(state.Excluded) > 0 {
			panic("selected file mismatch")
		}
		files := map[string]any{}
		for _, raw := range payload.Files {
			var v map[string]any
			must(json.Unmarshal(raw, &v))
			id := v["id"].(string)
			path := v["new_path"].(string)
			before, e := git(root, "show", "HEAD:"+path)
			must(e)
			after, e := os.ReadFile(filepath.Join(root, path))
			must(e)
			diff, e := git(root, "diff", "--no-ext-diff", "--no-textconv", "--unified=3", "HEAD", "--", path)
			must(e)
			if len(diff) > 65536 {
				panic("diff cap; never truncate")
			}
			files[id] = map[string]any{"path": path, "before_sha256": digest([]byte(before)), "after_sha256": digest(after), "raw_diff": diff, "raw_diff_sha256": digest([]byte(diff)), "baseline_file": json.RawMessage(raw)}
		}
		final, e := repositoryDigest(root)
		must(e)
		if final != initial || jsonDigest(p) != preparedHash || jsonDigest(state) != snapshotHash {
			panic("Git/prepared mutation")
		}
		public := state
		public.Root = "<local-fixture:" + f.Name + ">"
		mapping[f.Name] = map[string]any{"files": files, "snapshot": public, "git_unchanged": true}
		rows = append(rows, map[string]any{"fixture": f.Name, "repository_sha256": initial, "snapshot_sha256": snapshotHash, "prepared_sha256": preparedHash, "selected_files": changed, "capture_calls": c.Calls, "model_calls": 0})
	}
	for name, value := range map[string]any{"manifest.json": rows, "source-mapping.json": mapping, "runtime.json": map[string]any{"fixture_roots": roots}} {
		b, e := json.MarshalIndent(value, "", "  ")
		must(e)
		must(os.WriteFile(filepath.Join(*out, name), append(b, '\n'), 0600))
	}
	fmt.Printf("captured %d real grouping requests; model calls 0\n", len(rows))
}
