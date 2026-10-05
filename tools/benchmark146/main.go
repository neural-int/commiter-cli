// benchmark146 measures window construction and fail-closed reconciliation.
// It reuses production Stage 1 unchanged and never changes Git or CLI defaults.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type callMetric struct {
	Window            []string         `json:"window"`
	Phase             string           `json:"phase"`
	WallMS            float64          `json:"wall_ms"`
	InputBytes        int              `json:"input_bytes"`
	OutputBytes       int              `json:"output_bytes"`
	InputTokens       *int             `json:"input_tokens"`
	OutputTokens      *int             `json:"output_tokens"`
	OutputReservation int              `json:"output_reservation"`
	Stop              string           `json:"stop"`
	TextDiagnostics   *textDiagnostics `json:"text_diagnostics,omitempty"`
}
type observation struct {
	Fixture          string         `json:"fixture"`
	Files            int            `json:"files"`
	Mode             string         `json:"mode"`
	Strategy         string         `json:"strategy"`
	Exact            *bool          `json:"exact"`
	FM               *int           `json:"false_merge_pairs"`
	FS               *int           `json:"false_split_pairs"`
	Complete         bool           `json:"complete_assignment"`
	Unresolved       bool           `json:"unresolved"`
	Reason           string         `json:"reason,omitempty"`
	Groups           [][]string     `json:"groups,omitempty"`
	Windows          [][]string     `json:"windows"`
	Calls            []callMetric   `json:"calls"`
	BackendCalls     int            `json:"backend_calls"`
	WallMS           float64        `json:"wall_ms"`
	MaxWindows       int            `json:"max_windows"`
	TimeoutSeconds   float64        `json:"timeout_seconds"`
	LocalResults     []windowResult `json:"local_results"`
	PlanSucceeded    bool           `json:"plan_succeeded"`
	PlanFailure      string         `json:"plan_failure,omitempty"`
	MetadataExecuted bool           `json:"metadata_executed"`
}

// capture receives finalized groups only after the existing generator has
// validated the entire membership object. Category is deliberately intercepted:
// window-level type/text would incorrectly make windows commit boundaries.
type capture struct {
	Backend llm.OptionsBackend
	Truth   map[string]string
	Metrics []callMetric
	Groups  [][]string
	Window  []string
	Reached bool
}

func (c *capture) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected legacy call")
}
func (c *capture) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	if o.GenerationProfile == "bounded-category" {
		var payload struct {
			Groups []struct {
				FileIDs []string `json:"file_ids"`
			} `json:"groups"`
		}
		if err := json.Unmarshal([]byte(m[1].Content), &payload); err != nil {
			return llm.Response{}, err
		}
		for _, g := range payload.Groups {
			c.Groups = append(c.Groups, g.FileIDs)
		}
		c.Reached = true
		return llm.Response{}, errors.New("experiment stops before local metadata")
	}
	if o.GenerationProfile != "bounded-grouping" {
		return llm.Response{}, errors.New("unexpected phase")
	}
	start := time.Now()
	r := llm.Response{}
	var err error
	if c.Backend != nil {
		r, err = c.Backend.ChatWithOptions(ctx, m, s, o)
	} else {
		var payload struct {
			Files []contextinput.File `json:"files"`
		}
		if err = json.Unmarshal([]byte(m[1].Content), &payload); err == nil {
			labels := map[string]string{}
			membership := map[string]string{}
			for _, f := range payload.Files {
				truth, ok := c.Truth[f.ID]
				if !ok {
					return r, errors.New("oracle missing file")
				}
				if labels[truth] == "" {
					labels[truth] = fmt.Sprintf("G%03d", len(labels)+1)
				}
				membership[f.ID] = labels[truth]
			}
			encoded, _ := json.Marshal(membership)
			r = llm.Response{Content: string(encoded), StopReason: "completed"}
		}
	}
	metric := callMetric{Window: c.Window, Phase: o.GenerationProfile, WallMS: float64(time.Since(start)) / float64(time.Millisecond), OutputBytes: len(r.Content), OutputReservation: o.OutputTokens, Stop: r.StopReason}
	for _, message := range m {
		metric.InputBytes += len(message.Content)
	}
	if r.Availability.PromptEvalCount {
		metric.InputTokens = &r.PromptEvalCount
	}
	if r.Availability.EvalCount {
		metric.OutputTokens = &r.EvalCount
	}
	if err != nil {
		metric.Stop = "backend_error"
	}
	c.Metrics = append(c.Metrics, metric)
	return r, err
}
func local(ctx context.Context, f fixture, w []string, backend llm.OptionsBackend) (windowResult, []callMetric, error) {
	d := contextinput.Document{SchemaVersion: 1}
	for _, file := range f.Files {
		if contains(w, file.ID) {
			d.Files = append(d.Files, file)
		}
	}
	prompt, err := planning.Renderer(planning.English)(d)
	if err != nil {
		return windowResult{}, nil, err
	}
	p := contextinput.Prepared{Document: d, Prompt: prompt, Budget: contextinput.Budget{ContextTokens: 16384}}
	truth := map[string]string{}
	for g, group := range f.Expected {
		for _, id := range group {
			for _, file := range f.Files {
				if file.ID == id {
					truth["path:"+*file.NewPath] = fmt.Sprint(g)
				}
			}
		}
	}
	c := &capture{Backend: backend, Truth: truth, Window: w}
	_, err = (planning.ThreePhaseGenerator{Client: c}).Generate(ctx, p, planning.English, planning.SensitiveValues{})
	if !c.Reached {
		return windowResult{}, c.Metrics, err
	}
	return windowResult{IDs: w, Groups: c.Groups}, c.Metrics, nil
}
func evaluate(expected, actual [][]string) (bool, int, int) {
	truth, pred := map[string]int{}, map[string]int{}
	ids := []string{}
	for g, group := range expected {
		for _, id := range group {
			truth[id] = g
			ids = append(ids, id)
		}
	}
	for g, group := range actual {
		for _, id := range group {
			pred[id] = g
		}
	}
	fm, fs := 0, 0
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			same := pred[a] == pred[b]
			want := truth[a] == truth[b]
			if same && !want {
				fm++
			}
			if !same && want {
				fs++
			}
		}
	}
	return signature(expected) == signature(actual), fm, fs
}
func run(parent context.Context, f fixture, strategy, mode string, backend llm.OptionsBackend, maxWindows int, timeout time.Duration, metadata ...bool) observation {
	start := time.Now()
	o := observation{Fixture: f.Name, Files: len(f.Files), Mode: mode, Strategy: strategy, Calls: []callMetric{}, MaxWindows: maxWindows, TimeoutSeconds: timeout.Seconds()}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	selected := make([]string, len(f.Files))
	for i, file := range f.Files {
		selected[i] = file.ID
	}
	if !complete(selected, [][]string{idsFor(f.Graph)}) {
		o.Reason = "graph_selected_id_mismatch"
		o.Unresolved = true
		return o
	}
	windows, err := initialWindows(f.Graph)
	if err != nil {
		o.Reason = err.Error()
		o.Unresolved = true
		return o
	}
	results := []windowResult{}
	p := partition{}
	for {
		for _, w := range windows {
			if len(o.Windows) >= maxWindows {
				o.Reason = "window_budget"
				break
			}
			r, metrics, err := local(ctx, f, w, backend)
			o.Windows = append(o.Windows, w)
			o.Calls = append(o.Calls, metrics...)
			if mode == "mlx" {
				o.BackendCalls += len(metrics)
			}
			if err != nil {
				if errors.Is(err, contextinput.ErrTooLarge) {
					o.Reason = "context_overflow"
				} else if ctx.Err() != nil {
					o.Reason = "cycle_timeout"
				} else {
					o.Reason = "invalid_or_incomplete_local_output"
				}
				break
			}
			results = append(results, r)
			partial := reconcile(idsFor(f.Graph), results)
			if partial.Reason == "contradictory_pair" || partial.Reason == "transitive_contradiction" {
				o.Reason = partial.Reason
				break
			}
		}
		if o.Reason != "" {
			break
		}
		p = reconcile(idsFor(f.Graph), results)
		if strategy == "audited" && p.Reason == "" {
			if w := auditWindow(idsFor(f.Graph), results); len(w) > 0 {
				windows = [][]string{w}
				continue
			}
		}
		if strategy == "graph-only" || p.Reason != "unobserved_group_relation" {
			o.Reason = p.Reason
			break
		}
		windows = [][]string{bridgeWindow(p)}
	}
	o.LocalResults = results
	o.WallMS = float64(time.Since(start)) / float64(time.Millisecond)
	o.Unresolved = o.Reason != ""
	if !o.Unresolved {
		o.Groups = p.Groups
		o.Complete = complete(idsFor(f.Graph), p.Groups)
		if !o.Complete {
			o.Reason = "invalid_global_assignment"
			o.Unresolved = true
			o.Groups = nil
		} else {
			exact, fm, fs := evaluate(f.Expected, p.Groups)
			o.Exact = &exact
			o.FM = &fm
			o.FS = &fs
			if len(metadata) > 0 && metadata[0] {
				plan, calls, failure := finalMetadata(ctx, f, p, backend, mode)
				o.Calls = append(o.Calls, calls...)
				if mode == "mlx" {
					o.BackendCalls += len(calls)
				}
				o.MetadataExecuted = len(calls) > 0 || (backend == nil && failure == "")
				o.PlanFailure = failure
				o.PlanSucceeded = failure == "" && len(plan.Commits) > 0
			}
		}
	}
	o.WallMS = float64(time.Since(start)) / float64(time.Millisecond)
	return o
}
func complete(ids []string, groups [][]string) bool {
	seen := map[string]bool{}
	for _, g := range groups {
		for _, id := range g {
			if seen[id] || !contains(ids, id) {
				return false
			}
			seen[id] = true
		}
	}
	return len(seen) == len(ids)
}
func main() {
	suite := flag.String("suite", "controlled", "controlled or contract fixed fixture suite")
	metadata := flag.Bool("metadata", false, "run Stage 2/3 only after global membership is finalized")
	mode := flag.String("mode", "oracle", "oracle checks only the contract; mlx performs actual inference")
	helper := flag.String("helper", "", "already built production helper")
	cache := flag.String("cache", "", "existing model cache root")
	filter := flag.String("fixture", "", "one fixed fixture, empty runs all")
	strategy := flag.String("strategy", "both", "graph-only, bridge, audited, both, or baseline")
	maxWindows := flag.Int("max-windows", 48, "experimental call guard, not an adopted production limit")
	timeout := flag.Duration("timeout", 10*time.Minute, "experimental per-fixture wall-time guard")
	flag.Parse()
	if *maxWindows < 1 || *timeout <= 0 || (*strategy != "both" && *strategy != "bridge" && *strategy != "graph-only" && *strategy != "baseline" && *strategy != "audited") {
		fmt.Fprintln(os.Stderr, "invalid experiment bounds")
		os.Exit(2)
	}
	var backend llm.OptionsBackend
	if *mode == "mlx" {
		v := config.Defaults().Values
		store := mlxmodel.Store{Root: *cache}
		path, err := store.Ready(mlxmodel.Spec{Repo: v.Model, Revision: v.ModelRevision, Quantization: v.ModelQuantization})
		if err != nil || *helper == "" {
			fmt.Fprintln(os.Stderr, "explicit cached model and helper required")
			os.Exit(2)
		}
		backend = &measuredBackend{Executable: *helper, Model: v.Model, Revision: v.ModelRevision, Path: path}
	} else if *mode != "oracle" {
		fmt.Fprintln(os.Stderr, "invalid mode")
		os.Exit(2)
	}
	encoder := json.NewEncoder(os.Stdout)
	found := false
	selected := fixtures()
	if *suite == "contract" {
		selected = contractFixtures()
	} else if *suite != "controlled" {
		fmt.Fprintln(os.Stderr, "unknown suite")
		os.Exit(2)
	}
	for _, f := range selected {
		if *filter != "" && f.Name != *filter {
			continue
		}
		found = true
		if *strategy == "baseline" {
			if err := encoder.Encode(runBaseline(context.Background(), f, *mode, backend)); err != nil {
				panic(err)
			}
			continue
		}
		for _, s := range []string{"graph-only", "bridge", "audited"} {
			if (*strategy == "both" && s == "audited") || (*strategy != "both" && *strategy != s) {
				continue
			}
			if err := encoder.Encode(run(context.Background(), f, s, *mode, backend, *maxWindows, *timeout, *metadata)); err != nil {
				panic(err)
			}
		}
	}
	if !found {
		fmt.Fprintln(os.Stderr, "unknown fixture")
		os.Exit(2)
	}
}
