package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

// Explicit, opt-in research runner. Default tests never load history or a model.
func TestIssue175Evaluation(t *testing.T) {
	mode := os.Getenv("COMMITER_175_MODE")
	if mode == "" {
		t.Skip("opt-in evaluation")
	}
	if mode != "capture" && mode != "run" {
		t.Fatal("invalid mode")
	}
	repoFile := os.Getenv("COMMITER_175_REPOS")
	out := os.Getenv("COMMITER_175_OUT")
	if out == "" {
		t.Fatal("output required")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	var repos []struct {
		CaseID string `json:"case_id"`
		Root   string `json:"root"`
	}
	raw, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &repos); err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatal("exactly two cases required")
	}
	// Read the actual current shared instruction, avoiding a second maintained copy.
	source, err := parser.ParseFile(token.NewFileSet(), "../planning/codex_context.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	instruction := ""
	ast.Inspect(source, func(n ast.Node) bool {
		v, ok := n.(*ast.ValueSpec)
		if ok && len(v.Names) == 1 && v.Names[0].Name == "codexTrustInstruction" {
			instruction, _ = strconv.Unquote(v.Values[0].(*ast.BasicLit).Value)
		}
		return true
	})
	if instruction == "" {
		t.Fatal("instruction missing")
	}
	values := config.Defaults().Values
	store, err := mlxmodel.DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	modelPath, err := store.Ready(mlxmodel.Spec{Repo: values.Model, Revision: values.ModelRevision, Quantization: values.ModelQuantization})
	if err != nil {
		t.Fatal(err)
	}
	helper := os.Getenv("COMMITER_175_HELPER")
	if helper == "" {
		t.Fatal("helper required")
	}
	for _, repo := range repos {
		snapshot, err := gitstate.Collect(repo.Root, gitstate.Options{})
		if err != nil {
			t.Fatal(err)
		}
		results, sensitive, _, relationFiles, err := analyzeForPlanningWithStatsAndRelationsContext(context.Background(), repo.Root, snapshot)
		if err != nil {
			t.Fatal(err)
		}
		document, err := contextinput.Build(snapshot, results)
		if err != nil {
			t.Fatal(err)
		}
		attachRelationContext(&document, snapshot.Changes, relationFiles)
		if err = attachGroupingSyntaxContext(context.Background(), repo.Root, &document, relationFiles); err != nil {
			t.Fatal(err)
		}
		hostDocument := document
		document, err = contextinput.RawDiffProjection(document)
		if err != nil {
			t.Fatal(err)
		}
		projectionAudit := []map[string]any{}
		for i, before := range hostDocument.Files {
			after := document.Files[i]
			beforeJSON, _ := json.Marshal(before)
			afterJSON, _ := json.Marshal(after)
			preserved := after.RawDiff == before.GroupingRawDiff
			if after.Mode == before.Mode {
				preserved = len(after.Evidence) == len(before.Evidence)
			}
			if !preserved {
				t.Fatal("approved source lost")
			}
			projectionAudit = append(projectionAudit, map[string]any{
				"path": before.NewPath, "before_mode": before.Mode, "after_mode": after.Mode,
				"host_evidence_count": len(before.Evidence), "before_bytes": len(beforeJSON), "after_bytes": len(afterJSON),
				"approved_diff_sha256":      fmt.Sprintf("%x", sha256.Sum256([]byte(before.GroupingRawDiff))),
				"selected_raw_diff_sha256":  fmt.Sprintf("%x", sha256.Sum256([]byte(after.RawDiff))),
				"approved_source_preserved": preserved,
			})
		}
		write175(t, filepath.Join(out, repo.CaseID+"-projection-audit.json"), projectionAudit)

		ids := []string{}
		for _, f := range document.Files {
			ids = append(ids, f.ID)
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
		write175(t, filepath.Join(out, repo.CaseID+"-preparation.json"), prepared)
		write175(t, filepath.Join(out, repo.CaseID+"-original-document.json"), document)
		var fixture struct {
			Context []codexcontext.Record `json:"codex_context"`
		}
		fixturePath := evaluationFixturePath("COMMITER_175_FIXTURE_DIR", "../../docs/benchmarks/issue-175/semantic-evaluation", repo.CaseID)
		raw, err = os.ReadFile(fixturePath)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, &fixture); err != nil {
			t.Fatal(err)
		}
		write175(t, filepath.Join(out, repo.CaseID+"-prepared.json"), prepared.Document)
		for _, arm := range []string{"A", "B"} {
			dir := filepath.Join(out, repo.CaseID+"-"+arm)
			if err = os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			backend := &evaluation175Client{t: t, dir: dir, mode: mode, arm: arm, instruction: instruction, backend: &mlx.Backend{Client: mlx.NewClient(helper), Model: values.Model, ModelPath: modelPath, ModelRevision: values.ModelRevision}}
			records := []codexcontext.Record(nil)
			if arm == "B" {
				records = fixture.Context
			}
			started := time.Now()
			result, genErr := (planning.ThreePhaseGenerator{Client: backend, CodexContext: records}).Generate(context.Background(), prepared, planning.Japanese, sensitive)
			reason := ""
			if genErr != nil {
				reason = genErr.Error()
			}
			write175(t, filepath.Join(dir, "result.json"), map[string]any{"result": result, "error": reason, "wall_seconds": time.Since(started).Seconds(), "mode": mode, "file_ids": ids, "file_paths": document.Files})
			t.Logf("%s %s mode=%s calls=%d accepted=%v error=%s", repo.CaseID, arm, mode, result.Calls, genErr == nil, reason)
		}
	}
}

type evaluation175Client struct {
	t                           *testing.T
	dir, mode, arm, instruction string
	count                       int
	backend                     llm.OptionsBackend
}

func (c *evaluation175Client) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, fmt.Errorf("legacy call forbidden")
}
func (c *evaluation175Client) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	c.count++
	if c.count > 3 {
		return llm.Response{}, fmt.Errorf("call limit")
	}
	m = append([]llm.Message(nil), m...)
	if !strings.HasSuffix(m[0].Content, c.instruction) {
		m[0].Content += c.instruction
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(m[1].Content), &payload); err != nil {
		return llm.Response{}, err
	}
	_, hasContext := payload["codex_context"]
	if hasContext != (c.arm == "B") {
		return llm.Response{}, fmt.Errorf("context presence mismatch")
	}
	required := len(m[0].Content) + len(m[1].Content) + contextinput.TemplateReserve + o.OutputTokens
	if required > o.ContextTokens {
		return llm.Response{}, fmt.Errorf("actual prompt byte bound exceeded")
	}
	request := mlx.Request{Messages: m, Schema: s, ContextTokens: o.ContextTokens, OutputTokens: o.OutputTokens, Model: config.CandidateModel + "@" + config.CandidateRevision, GenerationProfile: o.GenerationProfile}
	write175(c.t, filepath.Join(c.dir, fmt.Sprintf("%02d-request.json", c.count)), request)
	start := time.Now()
	response := llm.Response{}
	var err error
	if c.mode == "capture" {
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err = json.Unmarshal(s, &schema); err != nil {
			return response, err
		}
		output := map[string]any{}
		for key := range schema.Properties {
			switch o.GenerationProfile {
			case "bounded-grouping":
				output[key] = "G001"
			case "bounded-category":
				output[key] = map[string]string{"type": "fix", "breaking_evidence_ref": "none"}
			case "bounded-text":
				output[key] = map[string]string{"scope": "fixture", "summary": "変更内容を確認"}
			default:
				return response, fmt.Errorf("unknown phase")
			}
		}
		raw, _ := json.Marshal(output)
		response = llm.Response{Content: string(raw), StopReason: "completed"}
	} else {
		response, err = c.backend.ChatWithOptions(ctx, m, s, o)
	}
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	write175(c.t, filepath.Join(c.dir, fmt.Sprintf("%02d-response.json", c.count)), map[string]any{"response": response, "error": reason, "wall_seconds": time.Since(start).Seconds(), "mode": c.mode})
	return response, err
}
func write175(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err = f.Write(append(raw, '\n')); err != nil {
		t.Fatal(err)
	}
}
