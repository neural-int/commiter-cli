// benchmark110 measures the same synthetic planning inputs through both local
// backends. It prints aggregate metadata only; prompts and responses stay local.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/ollama"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

const defaultMLXRepo = "mlx-community/Qwen3.5-4B-MLX-4bit"
const defaultMLXRevision = "32f3e8ecf65426fc3306969496342d504bfa13f3"

type observation struct {
	Backend           string          `json:"backend"`
	Fixture           string          `json:"fixture"`
	Run               int             `json:"run"`
	Model             string          `json:"model"`
	ContextTokens     int             `json:"context_tokens"`
	PromptBytes       int             `json:"prompt_bytes"`
	OriginalBytes     int             `json:"original_prompt_bytes"`
	SummaryStage      string          `json:"summary_stage"`
	OutputBudget      int             `json:"output_budget"`
	WallMS            float64         `json:"generator_wall_ms"`
	Calls             int             `json:"calls"`
	Repaired          bool            `json:"repaired"`
	Succeeded         bool            `json:"succeeded"`
	Failure           string          `json:"failure,omitempty"`
	Groups            [][]string      `json:"groups,omitempty"`
	Types             []string        `json:"types,omitempty"`
	MaxScopeBytes     int             `json:"max_scope_bytes,omitempty"`
	MaxSummaryBytes   int             `json:"max_summary_bytes,omitempty"`
	ReferenceGrouping *bool           `json:"reference_grouping,omitempty"`
	Requests          []requestMetric `json:"requests"`
}

type requestMetric struct {
	WallMS        float64              `json:"wall_ms"`
	StopReason    string               `json:"stop_reason"`
	FailureKind   string               `json:"failure_kind,omitempty"`
	ResponseBytes int                  `json:"response_bytes,omitempty"`
	GrammarValid  *bool                `json:"grammar_valid,omitempty"`
	DomainValid   *bool                `json:"domain_valid,omitempty"`
	Violations    []planning.Violation `json:"violations,omitempty"`
	LoadMS        *float64             `json:"load_ms,omitempty"`
	PrefillTokens *int                 `json:"prefill_tokens,omitempty"`
	PrefillMS     *float64             `json:"prefill_ms,omitempty"`
	DecodeTokens  *int                 `json:"decode_tokens,omitempty"`
	DecodeMS      *float64             `json:"decode_ms,omitempty"`
}

type measuredClient struct {
	backend  llm.OptionsBackend
	ids      []string
	language planning.Language
	requests []requestMetric
}

func (client *measuredClient) Chat(ctx context.Context, messages []llm.Message, schema json.RawMessage) (llm.Response, error) {
	return client.ChatWithOptions(ctx, messages, schema, llm.Options{})
}

func (client *measuredClient) ChatWithOptions(ctx context.Context, messages []llm.Message, schema json.RawMessage, options llm.Options) (llm.Response, error) {
	start := time.Now()
	response, err := client.backend.ChatWithOptions(ctx, messages, schema, options)
	metric := requestMetric{WallMS: milliseconds(time.Since(start)), StopReason: response.StopReason}
	if err != nil {
		metric.StopReason = "backend_error"
		var failure *mlx.Error
		if errors.As(err, &failure) {
			metric.FailureKind = string(failure.Kind)
		}
		if ctx.Err() == context.DeadlineExceeded {
			metric.StopReason = "deadline_exceeded"
		} else if ctx.Err() == context.Canceled {
			metric.StopReason = "cancelled"
		}
	} else {
		metric.ResponseBytes = len(response.Content)
		if metric.StopReason == "" {
			metric.StopReason = "completed"
		}
		if metric.StopReason == "completed" {
			grammarViolations := planning.ValidateGrammar([]byte(response.Content))
			grammar := len(grammarViolations) == 0
			metric.GrammarValid = &grammar
			metric.Violations = append(metric.Violations, grammarViolations...)
			if grammar {
				_, violations := planning.Validate([]byte(response.Content), client.ids, planning.SensitiveValues{}, client.language)
				valid := len(violations) == 0
				metric.DomainValid = &valid
				metric.Violations = append(metric.Violations, violations...)
			}
		}
		if response.Availability.LoadDuration {
			value := float64(response.LoadDuration) / 1e6
			metric.LoadMS = &value
		}
		if response.Availability.PromptEvalCount {
			metric.PrefillTokens = &response.PromptEvalCount
		}
		if response.Availability.PromptEvalDuration {
			value := float64(response.PromptEvalDuration) / 1e6
			metric.PrefillMS = &value
		}
		if response.Availability.EvalCount {
			metric.DecodeTokens = &response.EvalCount
		}
		if response.Availability.EvalDuration {
			value := float64(response.EvalDuration) / 1e6
			metric.DecodeMS = &value
		}
	}
	client.requests = append(client.requests, metric)
	return response, err
}

type fixture struct {
	name      string
	language  planning.Language
	files     []fileSpec
	reference [][]string
}

type fileSpec struct{ path, diff string }

func fixtures() []fixture {
	fix := "+func Parse(input string) (string, error) { return normalize(input), nil }\n"
	test := "+func TestParse(t *testing.T) { got, _ := Parse(\"x\"); if got != \"x\" { t.Fatal(got) } }\n"
	large := strings.Repeat("+case format%d: return normalize(value) // handles one input variant\n", 320)
	return []fixture{
		{"normal", planning.English, []fileSpec{{"parser.go", fix}, {"parser_test.go", test}}, [][]string{{"F001", "F002"}}},
		{"multi_commit", planning.English, []fileSpec{{"parser.go", fix}, {"parser_test.go", test}, {"README.md", "+Document the new --format option.\n"}, {"docs/format.md", "+Describe output examples for --format.\n"}}, [][]string{{"F001", "F002"}, {"F003", "F004"}}},
		{"japanese", planning.Japanese, []fileSpec{{"cli.go", "+func PrintHelp() { fmt.Println(\"ヘルプを表示\") }\n"}, {"cli_test.go", "+func TestPrintHelp(t *testing.T) { /* 日本語の表示を確認 */ }\n"}}, [][]string{{"F001", "F002"}}},
		{"large_diff", planning.English, []fileSpec{{"format.go", large}, {"format_test.go", "+Test all format variants.\n"}, {"docs/format.md", "+Document supported formats.\n"}}, [][]string{{"F001", "F002", "F003"}}},
		{"near_64k", planning.English, []fileSpec{{"a.go", ""}, {"b.go", ""}, {"c.go", ""}, {"d.go", ""}, {"e.go", ""}, {"f.go", ""}}, nil},
		{"many_files", planning.English, manyFiles(24), nil},
	}
}

func manyFiles(count int) []fileSpec {
	files := make([]fileSpec, count)
	for i := range files {
		files[i] = fileSpec{path: fmt.Sprintf("feature%02d.go", i+1), diff: fmt.Sprintf("+func EnableFeature%02d() bool { return true }\n", i+1)}
	}
	return files
}

func (fixture fixture) prepare(ctx context.Context, outputBudget int) (contextinput.Prepared, []string, error) {
	doc := contextinput.Document{SchemaVersion: contextinput.SchemaVersion, Repository: contextinput.Repository{Head: "synthetic-head", Branch: "benchmark", IndexIdentity: "synthetic-index"}}
	ids := make([]string, len(fixture.files))
	for i, spec := range fixture.files {
		id := fmt.Sprintf("F%03d", i+1)
		ids[i] = id
		path := spec.path
		diff := "@@ -1,1 +1,2 @@\n" + spec.diff
		hash := sha256.Sum256([]byte(diff))
		file := contextinput.File{ID: id, Status: "M", NewPath: &path, ChangeHash: hex.EncodeToString(hash[:]), WorktreeKind: "file", Language: "go", Size: int64(len(diff)), Unstaged: true, Mode: syntax.ModeRawDiff, RawDiff: diff}
		if fixture.name == "near_64k" {
			file.Mode, file.RawDiff = syntax.ModeStructural, ""
			for j := 0; j < 70; j++ {
				file.Evidence = append(file.Evidence, syntax.Evidence{Kind: "function_declaration", Name: fmt.Sprintf("format_%03d_file_%d", j, i), StartLine: j*4 + 1, EndLine: j*4 + 3})
			}
		}
		doc.Files = append(doc.Files, file)
	}
	system, err := planning.InitialSystemMessage(ids)
	if err != nil {
		return contextinput.Prepared{}, nil, err
	}
	prepared, err := contextinput.Prepare(ctx, doc, contextinput.BudgetConfig{Context: "64k", MaxContextTokens: contextinput.Context64K, PromptOverheadBytes: len(system)}, planning.Renderer(fixture.language), nil)
	if err != nil {
		return contextinput.Prepared{}, nil, err
	}
	if outputBudget > 0 {
		prepared.Budget.ReservedOutputTokens = outputBudget
	}
	return prepared, ids, nil
}

func main() {
	backendName := flag.String("backend", "ollama", "ollama or mlx")
	fixtureName := flag.String("fixture", "normal", "fixture name or all")
	repeats := flag.Int("repeats", 1, "runs per fixture")
	outputBudget := flag.Int("output-tokens", 1024, "maximum generated tokens")
	manyFileCount := flag.Int("many-file-count", 24, "number of independent files in many_files")
	timeout := flag.Duration("timeout", 5*time.Minute, "deadline for each planning run")
	describe := flag.Bool("describe", false, "print fixture sizes without calling a backend")
	compareRelations := flag.Bool("compare-relations", false, "measure Issue #128 fixtures before and after candidate relation context")
	issue140 := flag.Bool("issue140", false, "measure Issue #140 relation input ablations")
	issue141 := flag.Bool("issue141", false, "measure Issue #141 grouping-first two-pass planner")
	issue143Contract := flag.Bool("issue143-contract", false, "compare frozen candidate encodings")
	issue143Audit := flag.Bool("issue143-input-audit", false, "audit frozen synthetic inputs without inference")
	issue143 := flag.Bool("issue143", false, "measure preregistered Issue #143 model exchange")
	issue142 := flag.Bool("issue142", false, "measure Issue #142 candidate partition selection")
	issue142Probe := flag.String("issue142-probe", "comparison", "Issue #142 probe: comparison, budget, gold, forced, verify, tournament, bidirectional, difference, or lexical")
	issue141Probe := flag.String("issue141-probe", "comparison", "Issue #141 probe: comparison, file-centric, hybrid, soft, edge-decision, local-boundary, cluster-boundary, or metadata")
	issue140Probe := flag.String("issue140-probe", "ablation", "Issue #140 probe: ablation, guidance-statistics, grouping, atomicity, intents, or correction")
	prepareModel := flag.Bool("prepare-mlx-model", false, "download the pinned MLX model")
	helpPath := flag.String("helper", "", "MLX helper executable")
	ollamaModel := flag.String("ollama-model", config.Defaults().Values.Model, "Ollama model tag")
	mlxRepo := flag.String("mlx-repo", defaultMLXRepo, "MLX model repository")
	mlxRevision := flag.String("mlx-revision", defaultMLXRevision, "pinned MLX model revision")
	flag.Parse()
	ctx := context.Background()
	spec := mlxmodel.Spec{Repo: *mlxRepo, Revision: *mlxRevision, Quantization: "4bit"}
	if *prepareModel {
		if err := installModel(ctx, spec); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *repeats < 1 || *outputBudget < 0 || *timeout <= 0 || *manyFileCount < 1 || *manyFileCount > 100 {
		fmt.Fprintln(os.Stderr, "invalid repeats, output-tokens, timeout, or many-file-count")
		os.Exit(2)
	}
	if *issue143Contract {
		if err := runIssue143Contract(ctx, *helpPath, *describe); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *issue143Audit {
		if err := runIssue143InputAudit(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *issue143 {
		if err := runIssue143(ctx, *helpPath, *describe); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *compareRelations {
		if err := runIssue128(ctx, issue128Options{backendName: *backendName, fixtureName: *fixtureName, repeats: *repeats, outputBudget: *outputBudget, manyFileCount: *manyFileCount, timeout: *timeout, describe: *describe, helper: *helpPath, ollamaModel: *ollamaModel, modelSpec: spec}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *issue140 {
		options := issue140Options{backendName: *backendName, fixtureName: *fixtureName, repeats: *repeats, outputBudget: *outputBudget, manyFileCount: *manyFileCount, timeout: *timeout, describe: *describe, helper: *helpPath, ollamaModel: *ollamaModel, modelSpec: spec, probe: *issue140Probe}
		var err error
		switch *issue140Probe {
		case "ablation", "guidance-statistics", "grouping", "atomicity":
			err = runIssue140(ctx, options)
		case "intents", "correction":
			err = runIssue140Intents(ctx, options)
		default:
			err = fmt.Errorf("unknown Issue #140 probe %q", *issue140Probe)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *issue142 {
		options := issue140Options{backendName: *backendName, fixtureName: *fixtureName, repeats: *repeats, outputBudget: *outputBudget, timeout: *timeout, describe: *describe, helper: *helpPath, ollamaModel: *ollamaModel, modelSpec: spec}
		var err error
		switch *issue142Probe {
		case "comparison":
			err = runIssue142(ctx, options)
		case "budget":
			err = runIssue142BudgetDiagnostic(ctx, options)
		case "gold":
			err = runIssue142GoldDiagnostic(ctx, options)
		case "forced":
			err = runIssue142ForcedDiagnostic(ctx, options)
		case "verify":
			err = runIssue142Verification(ctx, options)
		case "tournament":
			err = runIssue142Tournament(ctx, options)
		case "bidirectional":
			err = runIssue142Bidirectional(ctx, options)
		case "difference":
			err = runIssue142Difference(ctx, options)
		case "lexical":
			err = runIssue142Lexical(ctx, options)
		case "next-lexical":
			err = runIssue142NextLexical(ctx, options)
		case "next-ranking":
			err = runIssue142NextRanking(ctx, options)
		case "relation-removal":
			err = runIssue142RelationRemoval(ctx, options)
		case "forced-lexical":
			err = runIssue142ForcedLexical(ctx, options)
		case "weighted-lexical":
			err = runIssue142Weighted(ctx, options)
		case "relation-annotation":
			err = runIssue142RelationAnnotation(ctx, options)
		case "prerank", "prerank-top2", "prerank-pairwise":
			err = runIssue142Prerank(ctx, options, *issue142Probe)
		case "capped-holdout":
			err = runIssue142CappedHoldout(ctx, options)
		case "capped-selector":
			err = runIssue142CappedSelector(ctx, options)
		case "small-enumeration":
			err = runIssue142SmallEnumeration(ctx, options)
		case "two-selectors":
			err = runIssue142TwoSelectors(ctx, options)
		case "score-breakdown":
			err = runIssue142ScoreBreakdown(ctx, options)
		case "pair-local":
			err = runIssue142PairLocal(ctx, options)
		case "pair-local-controls":
			err = runIssue142PairLocalControls(ctx, options)
		case "pair-local-stem-doc":
			err = runIssue142PairLocalStemDoc(ctx, options)
		default:
			err = fmt.Errorf("unknown Issue #142 probe %q", *issue142Probe)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *issue141 {
		options := issue140Options{backendName: *backendName, fixtureName: *fixtureName, repeats: *repeats, outputBudget: *outputBudget, timeout: *timeout, describe: *describe, helper: *helpPath, ollamaModel: *ollamaModel, modelSpec: spec}
		var err error
		switch *issue141Probe {
		case "comparison":
			err = runIssue141(ctx, options)
		case "file-centric":
			err = runIssue141FileCentric(ctx, options)
		case "hybrid":
			err = runIssue141Hybrid(ctx, options)
		case "soft":
			err = runIssue141Soft(ctx, options)
		case "edge-decision":
			err = runIssue141EdgeDecision(ctx, options)
		case "local-boundary":
			err = runIssue141LocalBoundary(ctx, options)
		case "cluster-boundary":
			err = runIssue141ClusterBoundary(ctx, options)
		case "metadata":
			err = runIssue141MetadataProbe(ctx, options)
		default:
			err = fmt.Errorf("unknown Issue #141 probe %q", *issue141Probe)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	var backend llm.OptionsBackend
	var model string
	var err error
	if !*describe {
		backend, model, err = openBackend(*backendName, *helpPath, *ollamaModel, spec)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	for _, fixture := range fixtures() {
		if *fixtureName != "all" && *fixtureName != fixture.name {
			continue
		}
		if fixture.name == "many_files" {
			fixture.files = manyFiles(*manyFileCount)
		}
		prepared, ids, err := fixture.prepare(ctx, *outputBudget)
		if err != nil {
			fmt.Fprintln(os.Stderr, fixture.name, err)
			os.Exit(1)
		}
		if *describe {
			_ = encoder.Encode(map[string]any{"fixture": fixture.name, "prompt_bytes": prepared.Budget.PromptBytes, "original_prompt_bytes": prepared.OriginalPromptBytes, "context_tokens": prepared.Budget.ContextTokens, "summary_stage": prepared.SummaryStage})
			continue
		}
		for run := 1; run <= *repeats; run++ {
			measured := &measuredClient{backend: backend, ids: ids, language: fixture.language}
			runContext, cancel := context.WithTimeout(ctx, *timeout)
			start := time.Now()
			result, generateErr := (planning.Generator{Client: measured}).Generate(runContext, prepared, fixture.language, planning.SensitiveValues{})
			cancel()
			row := observation{Backend: *backendName, Fixture: fixture.name, Run: run, Model: model, ContextTokens: prepared.Budget.ContextTokens, PromptBytes: prepared.Budget.PromptBytes, OriginalBytes: prepared.OriginalPromptBytes, SummaryStage: string(prepared.SummaryStage), OutputBudget: prepared.Budget.ReservedOutputTokens, WallMS: milliseconds(time.Since(start)), Calls: result.Calls, Repaired: result.Repaired, Succeeded: generateErr == nil, Requests: measured.requests}
			if generateErr != nil {
				row.Failure = "planning_rejected"
			} else {
				for _, commit := range result.Plan.Commits {
					row.Groups = append(row.Groups, commit.FileIDs)
					row.Types = append(row.Types, commit.Type)
					if len(commit.Scope) > row.MaxScopeBytes {
						row.MaxScopeBytes = len(commit.Scope)
					}
					if len(commit.Summary) > row.MaxSummaryBytes {
						row.MaxSummaryBytes = len(commit.Summary)
					}
				}
				if fixture.reference != nil {
					equal := sameGroups(row.Groups, fixture.reference)
					row.ReferenceGrouping = &equal
				}
			}
			if err := encoder.Encode(row); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
}

func openBackend(name, helper, ollamaModel string, spec mlxmodel.Spec) (llm.OptionsBackend, string, error) {
	if name == "ollama" {
		values := config.Defaults().Values
		values.Model = ollamaModel
		client, err := ollama.New(values)
		return client, values.Model, err
	}
	if name == "mlx" {
		if helper == "" {
			return nil, "", fmt.Errorf("-helper is required for MLX")
		}
		store, err := mlxmodel.DefaultStore()
		if err != nil {
			return nil, "", err
		}
		path, err := store.Ready(spec)
		if err != nil {
			return nil, "", err
		}
		return &mlx.Backend{Client: mlx.NewClient(helper), Model: spec.Repo, ModelPath: path}, spec.Repo + "@" + spec.Revision, nil
	}
	return nil, "", fmt.Errorf("unknown backend %q", name)
}

func installModel(ctx context.Context, spec mlxmodel.Spec) error {
	store, err := mlxmodel.DefaultStore()
	if err != nil {
		return err
	}
	plan, err := store.Plan(ctx, spec)
	if err != nil {
		return err
	}
	destination, err := store.Destination(spec)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "model=%s revision=%s quantization=4bit bytes=%d destination=%s\n", spec.Repo, spec.Revision, plan.Bytes, destination)
	path, err := store.Install(ctx, plan)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func sameGroups(actual, expected [][]string) bool {
	if len(actual) != len(expected) {
		return false
	}
	normalize := func(groups [][]string) []string {
		result := make([]string, len(groups))
		for i, group := range groups {
			ids := append([]string(nil), group...)
			sort.Strings(ids)
			result[i] = strings.Join(ids, ",")
		}
		sort.Strings(result)
		return result
	}
	a, b := normalize(actual), normalize(expected)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
