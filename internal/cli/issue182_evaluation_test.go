package cli

// Opt-in cross-model study: production collection, prompts and validators;
// model-default chat templates, retained model/grammar, no prefix KV reuse.
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func TestIssue182Evaluation(t *testing.T) {
	if os.Getenv("COMMITER_182_RUN") != "1" {
		t.Skip("opt-in Issue182")
	}
	out := os.Getenv("COMMITER_182_OUT")
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("COMMITER_182_REPO")
	snapshot, err := gitstate.Collect(root, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}
	results, sensitive, _, relations, err := analyzeForPlanningWithStatsAndRelationsContext(context.Background(), root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	document, err := contextinput.Build(snapshot, results)
	if err != nil {
		t.Fatal(err)
	}
	attachRelationContext(&document, snapshot.Changes, relations)
	if err = attachGroupingSyntaxContext(context.Background(), root, &document, relations); err != nil {
		t.Fatal(err)
	}
	document, err = contextinput.RawDiffProjection(document)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	paths := map[string]string{}
	for _, file := range document.Files {
		ids = append(ids, file.ID)
		if file.NewPath != nil {
			paths[file.ID] = *file.NewPath
		}
	}
	system, err := planning.InitialSystemMessage(ids)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := contextinput.Prepare(context.Background(), document, contextinput.BudgetConfig{RequireUncompressed: true, Context: "auto", MaxContextTokens: 65536, PromptOverheadBytes: len(system)}, planning.Renderer(planning.Japanese), nil)
	if err != nil {
		t.Fatal(err)
	}
	prepared.Budget.ContextTokens = 65536
	var fixture struct {
		Context []codexcontext.Record `json:"codex_context"`
	}
	read180(t, os.Getenv("COMMITER_182_FIXTURE"), &fixture)
	client := &evaluation182Client{evaluation180Client: &evaluation180Client{t: t, out: out}, model: os.Getenv("COMMITER_182_MODEL"), path: os.Getenv("COMMITER_182_MODEL_PATH")}
	defer client.close()
	write175(t, filepath.Join(out, "input.json"), map[string]any{"prepared": prepared, "paths": paths})
	start := time.Now()
	result, genErr := (planning.ThreePhaseGenerator{Client: client, CodexContext: fixture.Context}).Generate(context.Background(), prepared, planning.Japanese, sensitive)
	client.close()
	reason := ""
	if genErr != nil {
		reason = genErr.Error()
	}
	write175(t, filepath.Join(out, "result.json"), map[string]any{"result": result, "error": reason, "wall_seconds": time.Since(start).Seconds(), "paths": paths, "calls": client.count, "grouping_injected": false})
	t.Logf("calls=%d accepted=%v", client.count, genErr == nil)
}

type evaluation182Client struct {
	*evaluation180Client
	model, path string
}

func (c *evaluation182Client) ChatWithOptions(ctx context.Context, messages []llm.Message, schema json.RawMessage, options llm.Options) (llm.Response, error) {
	c.count++
	if c.count > 3 {
		return llm.Response{}, errors.New("Issue182 call cap")
	}
	request := mlx.Request{Messages: messages, Schema: schema, ContextTokens: options.ContextTokens, OutputTokens: options.OutputTokens, Model: c.model, ModelPath: c.path, GenerationProfile: options.GenerationProfile}
	wire, err := json.Marshal(request)
	if err != nil {
		return llm.Response{}, err
	}
	write175(c.t, filepath.Join(c.out, fmt.Sprintf("%02d-request.json", c.count)), request)
	if c.process == nil {
		if err = c.start(ctx, "--benchmark-model-defaults"); err != nil {
			return llm.Response{}, err
		}
	}
	start := time.Now()
	if _, err = c.input.Write(append(wire, '\n')); err != nil {
		return llm.Response{}, err
	}
	if !c.output.Scan() {
		return llm.Response{}, errors.New("helper exited before response; possible shared cycle timeout")
	}
	raw := append(json.RawMessage(nil), c.output.Bytes()...)
	var response mlx.Response
	err = json.Unmarshal(raw, &response)
	write175(c.t, filepath.Join(c.out, fmt.Sprintf("%02d-response.json", c.count)), map[string]any{"native": raw, "wall_seconds": time.Since(start).Seconds()})
	if err != nil {
		return llm.Response{}, err
	}
	if !response.OK || response.StopReason != mlx.StopReasonCompleted || response.GenerationProfile != options.GenerationProfile {
		return llm.Response{}, fmt.Errorf("helper failed: %s/%s", response.StopReason, response.ErrorClass)
	}
	return llm.Response{Content: response.GeneratedJSON, StopReason: string(response.StopReason), Model: response.Model, Backend: response.Runtime}, nil
}
