package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func issue142GenericForcedInput(prepared contextinput.Prepared, candidates []issue142Candidate, reverse bool) (string, []byte, json.RawMessage, error) {
	system, prompt, schema, err := issue142SelectionInputModeWithSchemaOrder(prepared, candidates, reverse, false, true)
	if err != nil {
		return "", nil, nil, err
	}
	old := "one of the two complete partition candidates"
	if strings.Count(system, old) != 1 {
		return "", nil, nil, fmt.Errorf("forced system contract changed")
	}
	system = strings.Replace(system, old, "one of the complete partition candidates", 1)
	return system, prompt, schema, nil
}

func issue142AnnotatedRelationInput(prepared contextinput.Prepared, candidates []issue142Candidate, reverse bool) (string, []byte, json.RawMessage, error) {
	system, prompt, schema, err := issue142SelectionInputModeWithSchemaOrder(prepared, candidates, reverse, false, true)
	if err != nil {
		return "", nil, nil, err
	}
	marker := []byte("\"relation_context\":{")
	if bytes.Count(prompt, marker) != 1 {
		return "", nil, nil, fmt.Errorf("relation context object missing or duplicated")
	}
	annotated := bytes.Replace(prompt, marker, []byte("\"relation_context\":{\"relation_interpretation\":\"structural_hint_not_shared_purpose_proof\","), 1)
	if !json.Valid(annotated) {
		return "", nil, nil, fmt.Errorf("invalid annotated prompt")
	}
	return system, annotated, schema, nil
}

func issue142FollowupCall(ctx context.Context, backend llm.OptionsBackend, model, probe, arm string, item fixture,
	prepared contextinput.Prepared, candidates []issue142Candidate, run int, system string, prompt []byte, schema json.RawMessage) issue142DiagnosticRow {
	row := issue142DiagnosticRow{Probe: probe, Fixture: item.name, Run: run, Arm: arm, Backend: "mlx", Model: model,
		OutputBudget: 2048, CandidateCount: len(candidates), OutputTokens: "unavailable",
		PromptBytes: len(system) + len(prompt), PromptSHA256: issue142Digest([]byte(system + string(prompt))), SchemaSHA256: issue142Digest(schema)}
	for _, candidate := range candidates {
		if sameGroups(candidate.Groups, item.reference) {
			row.CandidateRecall = true
			row.GoldCandidateID = candidate.ID
		}
	}
	start := time.Now()
	response, err := backend.ChatWithOptions(ctx, []llm.Message{{Role: "system", Content: system}, {Role: "user", Content: string(prompt)}},
		schema, llm.Options{ContextTokens: prepared.Budget.ContextTokens, OutputTokens: 2048})
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
	if selected == "none" {
		row.Failure = "forbidden_none"
		return row
	}
	row.SelectedID = selected
	row.ValidCandidate = true
	row.CompleteAssignment = true
	for _, candidate := range candidates {
		if candidate.ID != selected {
			continue
		}
		row.CorrectSelection = sameGroups(candidate.Groups, item.reference)
		score := issue140Row{}
		issue140Score(&score, candidate.Groups, item.reference)
		row.FalseMerge = &score.FalseMerge
		row.FalseSplit = &score.FalseSplit
		break
	}
	return row
}

type issue142ForcedLexicalArm struct {
	Name       string                  `json:"name"`
	Candidates []issue142Candidate     `json:"candidates"`
	GoldID     string                  `json:"gold_id,omitempty"`
	Eligible   bool                    `json:"eligible"`
	Calls      []issue142DiagnosticRow `json:"calls,omitempty"`
}
type issue142ForcedLexicalRow struct {
	Fixture string                     `json:"fixture"`
	Arms    []issue142ForcedLexicalArm `json:"arms"`
}

func runIssue142ForcedLexical(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" && !options.describe {
		return fmt.Errorf("forced lexical fixed to MLX")
	}
	var backend llm.OptionsBackend
	model := ""
	if !options.describe {
		var err error
		backend, model, err = openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for index, item := range issue142NextFixtures() {
		if options.fixtureName != "all" && options.fixtureName != item.name {
			continue
		}
		matched = true
		input, err := issue142BuildNextInputs(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		_, baseGoldID := issue142HasGold(input.baseline, item.reference)
		_, lexicalGoldID := issue142HasGold(input.lexical, item.reference)
		row := issue142ForcedLexicalRow{Fixture: item.name, Arms: []issue142ForcedLexicalArm{
			{Name: "baseline", Candidates: input.baseline, GoldID: baseGoldID, Eligible: baseGoldID != ""},
			{Name: "lexical", Candidates: input.lexical, GoldID: lexicalGoldID, Eligible: lexicalGoldID != ""},
		}}
		if !options.describe {
			for run, reverse := range issue142BidirectionalOrder {
				order := []int{0, 1}
				if (index+run)%2 == 1 {
					order[0], order[1] = 1, 0
				}
				for _, arm := range order {
					if !row.Arms[arm].Eligible {
						continue
					}
					system, prompt, schema, err := issue142GenericForcedInput(input.prepared, row.Arms[arm].Candidates, reverse)
					if err != nil {
						return fmt.Errorf("%s: %w", item.name, err)
					}
					callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
					call := issue142FollowupCall(callCtx, backend, model, "forced-lexical", row.Arms[arm].Name, item,
						input.prepared, row.Arms[arm].Candidates, run+1, system, prompt, schema)
					cancel()
					row.Arms[arm].Calls = append(row.Arms[arm].Calls, call)
				}
			}
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown forced lexical fixture %q", options.fixtureName)
	}
	return nil
}

type issue142AnnotationArm struct {
	Name  string                  `json:"name"`
	Calls []issue142DiagnosticRow `json:"calls,omitempty"`
}
type issue142AnnotationRow struct {
	Fixture    string                  `json:"fixture"`
	Candidates []issue142Candidate     `json:"candidates"`
	GoldID     string                  `json:"gold_id"`
	Arms       []issue142AnnotationArm `json:"arms"`
}

func runIssue142RelationAnnotation(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" && !options.describe {
		return fmt.Errorf("relation annotation fixed to MLX")
	}
	var backend llm.OptionsBackend
	model := ""
	if !options.describe {
		var err error
		backend, model, err = openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for index, name := range issue142RelationFixtures {
		if options.fixtureName != "all" && options.fixtureName != name {
			continue
		}
		matched = true
		item := fixture{}
		for _, candidate := range issue142TournamentFixtures() {
			if candidate.name == name {
				item = candidate
				break
			}
		}
		if item.name == "" {
			return fmt.Errorf("fixture %s missing", name)
		}
		prepared, ids, err := issue140Prepare(ctx, item, issue140ManyArms[5], 1024)
		if err != nil {
			return err
		}
		if prepared.Document.RelationContext == nil {
			return fmt.Errorf("%s missing relation context", name)
		}
		raw, _, err := issue142Generate(prepared.Document, ids)
		if err != nil {
			return err
		}
		expanded, err := issue142ExpandCandidates(prepared.Document, ids, raw)
		if err != nil {
			return err
		}
		if len(expanded) < 2 {
			return fmt.Errorf("%s fewer than 2 candidates", name)
		}
		candidates := expanded[:2]
		_, goldID := issue142HasGold(candidates, item.reference)
		row := issue142AnnotationRow{Fixture: name, Candidates: candidates, GoldID: goldID,
			Arms: []issue142AnnotationArm{{Name: "current"}, {Name: "annotated"}}}
		if !options.describe {
			for run, reverse := range issue142BidirectionalOrder {
				order := []int{0, 1}
				if (index+run)%2 == 1 {
					order[0], order[1] = 1, 0
				}
				for _, arm := range order {
					var system string
					var prompt []byte
					var schema json.RawMessage
					if arm == 0 {
						system, prompt, schema, err = issue142SelectionInputModeWithSchemaOrder(prepared, candidates, reverse, false, true)
					} else {
						system, prompt, schema, err = issue142AnnotatedRelationInput(prepared, candidates, reverse)
					}
					if err != nil {
						return fmt.Errorf("%s: %w", name, err)
					}
					callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
					call := issue142FollowupCall(callCtx, backend, model, "relation-annotation", row.Arms[arm].Name, item,
						prepared, candidates, run+1, system, prompt, schema)
					cancel()
					row.Arms[arm].Calls = append(row.Arms[arm].Calls, call)
				}
			}
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown relation annotation fixture %q", options.fixtureName)
	}
	return nil
}
