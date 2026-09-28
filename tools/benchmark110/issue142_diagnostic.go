package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

var issue142DiagnosticBudgets = []int{1024, 2048, 4096}
var issue142DiagnosticFixtures = []string{"cross_directory", "same_directory_independent", "mixed_24", "source_test_separate_purposes"}
var issue142InjectedFixtures = []string{"multi_commit", "new_same_directory_split"}

type issue142DiagnosticRow struct {
	Probe              string  `json:"probe"`
	Fixture            string  `json:"fixture"`
	Run                int     `json:"run"`
	Arm                string  `json:"arm"`
	Backend            string  `json:"backend"`
	Model              string  `json:"model"`
	OutputBudget       int     `json:"output_budget"`
	CandidateCount     int     `json:"candidate_count"`
	CandidateRecall    bool    `json:"candidate_recall"`
	GoldCandidateID    string  `json:"gold_candidate_id,omitempty"`
	PromptSHA256       string  `json:"prompt_sha256"`
	SchemaSHA256       string  `json:"schema_sha256"`
	PromptBytes        int     `json:"prompt_bytes"`
	StopReason         string  `json:"stop_reason"`
	Failure            string  `json:"failure,omitempty"`
	SelectedID         string  `json:"selected_id,omitempty"`
	None               bool    `json:"none"`
	ValidCandidate     bool    `json:"valid_candidate"`
	CorrectSelection   bool    `json:"correct_selection"`
	CompleteAssignment bool    `json:"complete_assignment"`
	FalseMerge         *int    `json:"false_merge,omitempty"`
	FalseSplit         *int    `json:"false_split,omitempty"`
	Calls              int     `json:"calls"`
	WallMS             float64 `json:"wall_ms"`
	OutputBytes        int     `json:"output_bytes"`
	OutputTokens       any     `json:"output_tokens"`
}

func issue142Digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func issue142DiagnosticCall(ctx context.Context, backend llm.OptionsBackend, model, backendName, probe, arm string, item fixture, prepared contextinput.Prepared, candidates []issue142Candidate, run, budget int, reverse bool) issue142DiagnosticRow {
	return issue142DiagnosticCallMode(ctx, backend, model, backendName, probe, arm, item, prepared, candidates, run, budget, reverse, false)
}

func issue142DiagnosticCallMode(ctx context.Context, backend llm.OptionsBackend, model, backendName, probe, arm string, item fixture, prepared contextinput.Prepared, candidates []issue142Candidate, run, budget int, reverse, forced bool) issue142DiagnosticRow {
	row := issue142DiagnosticRow{Probe: probe, Fixture: item.name, Run: run, Arm: arm, Backend: backendName, Model: model, OutputBudget: budget, CandidateCount: len(candidates), OutputTokens: "unavailable"}
	for _, candidate := range candidates {
		if sameGroups(candidate.Groups, item.reference) {
			row.CandidateRecall = true
			row.GoldCandidateID = candidate.ID
		}
	}
	var system string
	var prompt []byte
	var schema json.RawMessage
	var err error
	if forced {
		system, prompt, schema, err = issue142ForcedSelectionInput(prepared, candidates, reverse)
	} else {
		system, prompt, schema, err = issue142SelectionInput(prepared, candidates, reverse)
	}
	if err != nil {
		row.Failure = "input_error"
		return row
	}
	row.PromptBytes = len(system) + len(prompt)
	row.PromptSHA256 = issue142Digest([]byte(system + string(prompt)))
	row.SchemaSHA256 = issue142Digest(schema)
	prepared.Budget.ReservedOutputTokens = budget
	start := time.Now()
	response, err := backend.ChatWithOptions(ctx, []llm.Message{{Role: "system", Content: system}, {Role: "user", Content: string(prompt)}}, schema, llm.Options{ContextTokens: prepared.Budget.ContextTokens, OutputTokens: prepared.Budget.ReservedOutputTokens})
	row.Calls = 1
	row.WallMS = milliseconds(time.Since(start))
	row.StopReason = response.StopReason
	if err != nil {
		row.StopReason = "backend_error"
		if ctx.Err() == context.DeadlineExceeded {
			row.StopReason = "deadline_exceeded"
		} else if ctx.Err() == context.Canceled {
			row.StopReason = "cancelled"
		}
		row.Failure = row.StopReason
		return row
	}
	if row.StopReason == "" {
		row.StopReason = "completed"
	}
	if row.StopReason != "completed" {
		row.Failure = row.StopReason
		return row
	}
	row.OutputBytes = len(response.Content)
	if response.Availability.EvalCount {
		row.OutputTokens = response.EvalCount
	}
	selected, failure := issue142DecodeSelection([]byte(response.Content), candidates)
	if failure != "" {
		row.Failure = failure
		return row
	}
	row.SelectedID = selected
	if selected == "none" {
		if forced {
			row.Failure = "forbidden_none"
			return row
		}
		row.None = true
		row.CorrectSelection = !row.CandidateRecall
		return row
	}
	row.ValidCandidate = true
	for _, candidate := range candidates {
		if candidate.ID != selected {
			continue
		}
		row.CompleteAssignment = true // candidate generation already validated all IDs exactly once.
		row.CorrectSelection = sameGroups(candidate.Groups, item.reference)
		score := issue140Row{}
		issue140Score(&score, candidate.Groups, item.reference)
		row.FalseMerge = &score.FalseMerge
		row.FalseSplit = &score.FalseSplit
		return row
	}
	row.Failure = "unknown_candidate_id"
	return row
}

func runIssue142ForcedDiagnostic(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" {
		return fmt.Errorf("Issue #142 forced-choice diagnostic is fixed to MLX")
	}
	backend, model, err := openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for _, name := range issue142DiagnosticFixtures {
		if options.fixtureName != "all" && options.fixtureName != name {
			continue
		}
		matched = true
		item, err := issue142FindFixture(name)
		if err != nil {
			return err
		}
		prepared, _, candidates, err := issue142DiagnosticInputs(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if len(candidates) != 2 || !sameGroups(candidates[0].Groups, item.reference) && !sameGroups(candidates[1].Groups, item.reference) {
			return fmt.Errorf("%s: preregistered candidate assumption failed", name)
		}
		for run := 1; run <= 2; run++ {
			runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			row := issue142DiagnosticCallMode(runCtx, backend, model, options.backendName, "forced", "two-choice", item, prepared, candidates, run, 2048, run == 2, true)
			cancel()
			if err := encoder.Encode(row); err != nil {
				return err
			}
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 forced-choice fixture %q", options.fixtureName)
	}
	return nil
}

func issue142FindFixture(name string) (fixture, error) {
	for _, item := range issue142Fixtures() {
		if item.name == name {
			return item, nil
		}
	}
	return fixture{}, fmt.Errorf("unknown Issue #142 fixture %q", name)
}

func issue142DiagnosticInputs(ctx context.Context, item fixture) (contextinput.Prepared, []string, []issue142Candidate, error) {
	prepared, ids, err := issue140Prepare(ctx, item, issue140ManyArms[5], 1024)
	if err != nil {
		return contextinput.Prepared{}, nil, nil, err
	}
	candidates, _, err := issue142Generate(prepared.Document, ids)
	return prepared, ids, candidates, err
}

func runIssue142BudgetDiagnostic(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" {
		return fmt.Errorf("Issue #142 diagnostic is fixed to MLX")
	}
	backend, model, err := openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for _, name := range issue142DiagnosticFixtures {
		if options.fixtureName != "all" && options.fixtureName != name {
			continue
		}
		matched = true
		item, err := issue142FindFixture(name)
		if err != nil {
			return err
		}
		prepared, _, candidates, err := issue142DiagnosticInputs(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if len(candidates) != 2 || !sameGroups(candidates[0].Groups, item.reference) && !sameGroups(candidates[1].Groups, item.reference) {
			return fmt.Errorf("%s: preregistered candidate assumption failed", name)
		}
		for run := 1; run <= 2; run++ {
			budgets := append([]int(nil), issue142DiagnosticBudgets...)
			if run == 2 {
				budgets[0], budgets[2] = budgets[2], budgets[0]
			}
			for _, budget := range budgets {
				runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				row := issue142DiagnosticCall(runCtx, backend, model, options.backendName, "budget", "louvain", item, prepared, candidates, run, budget, run == 2)
				cancel()
				if err := encoder.Encode(row); err != nil {
					return err
				}
			}
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 diagnostic fixture %q", options.fixtureName)
	}
	return nil
}

func issue142SwapCandidateIDs(candidates []issue142Candidate) []issue142Candidate {
	result := make([]issue142Candidate, len(candidates))
	for i := range candidates {
		candidate := candidates[len(candidates)-1-i]
		candidate.ID = fmt.Sprintf("C%03d", i+1)
		result[i] = candidate
	}
	return result
}

func runIssue142GoldDiagnostic(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" {
		return fmt.Errorf("Issue #142 diagnostic is fixed to MLX")
	}
	backend, model, err := openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for _, name := range issue142InjectedFixtures {
		if options.fixtureName != "all" && options.fixtureName != name {
			continue
		}
		matched = true
		item, err := issue142FindFixture(name)
		if err != nil {
			return err
		}
		prepared, ids, original, err := issue142DiagnosticInputs(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if len(original) != 2 || sameGroups(original[0].Groups, item.reference) || sameGroups(original[1].Groups, item.reference) {
			return fmt.Errorf("%s: preregistered missing-gold assumption failed", name)
		}
		gold, _, err := issue142Canonical(item.reference, ids)
		if err != nil {
			return err
		}
		injected := []issue142Candidate{{ID: "C001", Groups: gold}, {ID: "C002", Groups: original[0].Groups}}
		for run := 1; run <= 2; run++ {
			arms := []string{"original", "gold-injected"}
			if run == 2 {
				arms[0], arms[1] = arms[1], arms[0]
			}
			for _, arm := range arms {
				candidates := original
				if arm == "gold-injected" {
					candidates = injected
				}
				if run == 2 {
					candidates = issue142SwapCandidateIDs(candidates)
				}
				runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				row := issue142DiagnosticCall(runCtx, backend, model, options.backendName, "gold", arm, item, prepared, candidates, run, 4096, false)
				cancel()
				if err := encoder.Encode(row); err != nil {
					return err
				}
			}
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 gold fixture %q", options.fixtureName)
	}
	return nil
}
