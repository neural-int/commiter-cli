package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"os"
	"strings"
	"testing"
)

type observed004 struct {
	llm.OptionsBackend
	rows   *[]llm.Response
	counts map[string]int
}

func (o observed004) CountInputTokens(c context.Context, m []llm.Message, s json.RawMessage, p llm.Options) (int, error) {
	n, err := o.OptionsBackend.(llm.InputTokenCounter).CountInputTokens(c, m, s, p)
	if err == nil {
		o.counts[p.GenerationProfile] = n
	}
	return n, err
}

func (o observed004) ChatWithOptions(c context.Context, m []llm.Message, s json.RawMessage, p llm.Options) (llm.Response, error) {
	r, e := o.OptionsBackend.ChatWithOptions(c, m, s, p)
	if e == nil && (!r.Availability.PromptEvalCount || r.PromptEvalCount != o.counts[p.GenerationProfile]) {
		return r, fmt.Errorf("counted input %d differs from generated input %d", o.counts[p.GenerationProfile], r.PromptEvalCount)
	}
	*o.rows = append(*o.rows, r)
	return r, e
}

type observed004Cycle struct {
	backend *mlx.Backend
	rows    *[]llm.Response
}

func (o observed004Cycle) Chat(c context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return o.backend.Chat(c, m, s)
}
func (o observed004Cycle) ChatWithOptions(c context.Context, m []llm.Message, s json.RawMessage, p llm.Options) (llm.Response, error) {
	return o.backend.ChatWithOptions(c, m, s, p)
}
func (o observed004Cycle) BeginCycle(c context.Context) (llm.OptionsBackend, func(), error) {
	b, close, e := o.backend.BeginCycle(c)
	if e != nil {
		return nil, nil, e
	}
	return observed004{b, o.rows, map[string]int{}}, close, nil
}

// Explicit installed assets only: never downloads weights or mutates a Git repository.
func TestHarness004InstalledModelCycle(t *testing.T) {
	helper, path, id := os.Getenv("COMMITER_TEST_REAL_HELPER"), os.Getenv("COMMITER_TEST_REAL_MODEL"), os.Getenv("COMMITER_TEST_REAL_MODEL_ID")
	if helper == "" || path == "" || id == "" {
		t.Skip("explicit installed helper/model required")
	}
	name, revision, ok := strings.Cut(id, "@")
	if !ok {
		t.Fatal("pinned model revision required")
	}
	backend := &mlx.Backend{Client: mlx.NewClient(helper), Model: name, ModelRevision: revision, ModelPath: path}
	rows := []llm.Response{}
	p := candidatePrepared(t)
	p.Document.Files = p.Document.Files[:1]
	var e error
	p.Prompt, e = Renderer(English)(p.Document)
	if e != nil {
		t.Fatal(e)
	}
	r, e := (ThreePhaseGenerator{Client: observed004Cycle{backend, &rows}}).Generate(context.Background(), p, English, SensitiveValues{})
	if e != nil {
		t.Fatal(e)
	}
	if r.Calls != 3 || len(rows) != 3 {
		t.Fatalf("expected three phase responses: calls=%d responses=%d", r.Calls, len(rows))
	}
	if rows[0].PrefixReuseStatus != "created" || rows[1].CacheReusedTokens != 0 || rows[2].PrefixReuseStatus != "reused" || rows[2].CacheReusedTokens <= 0 {
		t.Fatalf("cycle/prefix contract: calls=%d statuses=%v/%v/%v reused=%d", r.Calls, rows[0].PrefixReuseStatus, rows[1].PrefixReuseStatus, rows[2].PrefixReuseStatus, rows[2].CacheReusedTokens)
	}
	t.Logf("three phase final validator accepted; Text reused %d tokens", rows[2].CacheReusedTokens)
	for i, row := range rows {
		t.Logf("phase %d actual input tokens=%d", i+1, row.PromptEvalCount)
	}
}
