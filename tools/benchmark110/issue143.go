package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

const issue143ManifestPath = "docs/benchmarks/issue-143-manifest-2026-09-30.json"
const issue143HoldoutPath = "docs/benchmarks/issue-143-holdout-2026-09-30.json"
const issue143HelperHash = "da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48"

var issue143Models = []mlxmodel.Spec{
	{Repo: "mlx-community/Ministral-3-3B-Instruct-2512-4bit", Revision: "a962dcb09eee4169c890e544c9eb938f1113fdee", Quantization: "4bit"},
	{Repo: "ibm-granite/granite-4.2-3b-q4-mlx", Revision: "0c6f39b1827afd5eb2c1c3b13751929857434953", Quantization: "4bit"},
	{Repo: "mlx-community/Phi-4-mini-instruct-4bit", Revision: "ac1c269cb4222a4e136a3d09edad301056c1f36a", Quantization: "4bit"},
	{Repo: "mlx-community/NVIDIA-Nemotron-3-Nano-4B-4bit", Revision: "c4d79ba1901d99806ef757642a552acebb851a35", Quantization: "4bit"},
	{Repo: "mlx-community/gemma-4-E4B-it-4bit", Revision: "475b9088d29754a3379866cf5aeb6b41acd313c2", Quantization: "4bit"},
}

type issue143Contract struct {
	Fixture       string              `json:"fixture"`
	Set           string              `json:"set"`
	Category      string              `json:"category"`
	Guardrail     bool                `json:"guardrail"`
	Gold          [][]string          `json:"gold"`
	Candidates    []issue142Candidate `json:"candidates"`
	ContextTokens int                 `json:"context_tokens"`
	PromptHashes  []string            `json:"prompt_hashes"`
	SchemaHashes  []string            `json:"schema_hashes"`
}
type issue143Input struct {
	item     fixture
	prepared contextinput.Prepared
	contract issue143Contract
}
type issue143Holdout struct {
	Name       string                `json:"name"`
	Category   string                `json:"category"`
	Guardrail  bool                  `json:"guardrail"`
	Files      []issue142HoldoutFile `json:"files"`
	Gold       [][]string            `json:"gold"`
	Candidates []issue142Candidate   `json:"candidates"`
	Rationale  string                `json:"rationale"`
}
type issue143Manifest struct {
	Models         []mlxmodel.Spec    `json:"models"`
	HelperHash     string             `json:"helper_sha256"`
	OutputTokens   int                `json:"output_tokens"`
	TimeoutSeconds int                `json:"timeout_seconds"`
	Reverse        []bool             `json:"reverse"`
	Contracts      []issue143Contract `json:"contracts"`
}
type issue143MeasuredBackend struct {
	llm.OptionsBackend
	FailureKind string
}

func (backend *issue143MeasuredBackend) ChatWithOptions(ctx context.Context, messages []llm.Message, schema json.RawMessage, options llm.Options) (llm.Response, error) {
	backend.FailureKind = ""
	response, err := backend.OptionsBackend.ChatWithOptions(ctx, messages, schema, options)
	var failure *mlx.Error
	if errors.As(err, &failure) {
		backend.FailureKind = string(failure.Kind)
	}
	return response, err
}

type issue143Observation struct {
	FailureKind string `json:"failure_kind,omitempty"`
	issue142DiagnosticRow
	Set           string `json:"set"`
	Category      string `json:"category"`
	Guardrail     bool   `json:"guardrail"`
	ModelIndex    int    `json:"model_index"`
	Sequence      int    `json:"sequence"`
	Reverse       bool   `json:"reverse"`
	ContextTokens int    `json:"context_tokens"`
	HelperHash    string `json:"helper_sha256"`
}

func issue143Prepare(ctx context.Context) ([]issue143Input, error) {
	inputs := []issue143Input{}
	// Exactly reuse the four historical selector inputs; do not tune ranking.
	items := append(issue142TournamentFixtures(), issue142NextFixtures()...)
	capped, err := issue142LoadHoldoutFixtures(issue142CappedFixturePath)
	if err != nil {
		return nil, err
	}
	items = append(items, capped...)
	for _, name := range []string{"verify_misleading_relation", "lexical_crossdir_pairs", "new_stem_doc_diverged", "new_paraphrase"} {
		var item fixture
		for _, f := range items {
			if f.name == name {
				item = f
				break
			}
		}
		if item.name == "" {
			return nil, fmt.Errorf("missing known fixture %s", name)
		}
		input, err := issue142BuildRankInput(ctx, item)
		if err != nil {
			return nil, err
		}
		ranked := input.arms[2]
		if name == "new_stem_doc_diverged" || name == "new_paraphrase" {
			candidates := issue142CandidatesByID(ranked)
			if name == "new_paraphrase" {
				partitions, err := issue142SmallPartitions(input.ids)
				if err != nil {
					return nil, err
				}
				candidates, _, _, err = issue142AddPartitions(candidates, input.ids, partitions)
				if err != nil {
					return nil, err
				}
			}
			support, err := issue142CappedPairSupport(input.prepared.Document, input.ids)
			if err != nil {
				return nil, err
			}
			ranked = issue142Rank("weighted", candidates, len(input.arms[2].Candidates), item.reference, support)
		}
		pair, err := issue142TopCandidates(ranked)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, issue143Input{item: item, prepared: input.prepared, contract: issue143Contract{Fixture: name, Set: "known", Category: "diagnostic", Gold: item.reference, Candidates: pair}})
	}
	data, err := os.ReadFile(issue143HoldoutPath)
	if err != nil {
		return nil, err
	}
	var holdouts []issue143Holdout
	if err = json.Unmarshal(data, &holdouts); err != nil {
		return nil, err
	}
	if len(holdouts) != 8 {
		return nil, fmt.Errorf("expected eight holdouts")
	}
	for _, h := range holdouts {
		specs := []fileSpec{}
		for _, f := range h.Files {
			specs = append(specs, fileSpec{path: f.Path, diff: f.Diff})
		}
		item := fixture{name: h.Name, language: planning.English, files: specs, reference: h.Gold}
		prepared, _, err := issue140Prepare(ctx, item, issue140ManyArms[5], 1024)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, issue143Input{item: item, prepared: prepared, contract: issue143Contract{Fixture: h.Name, Set: "holdout", Category: h.Category, Guardrail: h.Guardrail, Gold: h.Gold, Candidates: h.Candidates}})
	}
	seen := map[string]bool{}
	for i := range inputs {
		input := &inputs[i]
		if seen[input.item.name] {
			return nil, fmt.Errorf("duplicate fixture")
		}
		seen[input.item.name] = true
		ids := []string{}
		for _, f := range input.prepared.Document.Files {
			ids = append(ids, f.ID)
		}
		if len(input.contract.Candidates) != 2 {
			return nil, fmt.Errorf("two candidates required")
		}
		goldCount := 0
		partitions := map[string]bool{}
		candidateIDs := map[string]bool{}
		for _, c := range input.contract.Candidates {
			if c.ID == "" || c.ID == "none" || candidateIDs[c.ID] {
				return nil, fmt.Errorf("invalid candidate ID")
			}
			candidateIDs[c.ID] = true
			_, key, err := issue142Canonical(c.Groups, ids)
			if err != nil {
				return nil, err
			}
			if partitions[key] {
				return nil, fmt.Errorf("duplicate partition")
			}
			partitions[key] = true
			if sameGroups(c.Groups, input.item.reference) {
				goldCount++
			}
		}
		if goldCount != 1 {
			return nil, fmt.Errorf("expected exactly one gold candidate")
		}
		input.contract.ContextTokens = input.prepared.Budget.ContextTokens
		for _, reverse := range []bool{false, true} {
			system, prompt, schema, err := issue142GenericForcedInput(input.prepared, input.contract.Candidates, reverse)
			if err != nil {
				return nil, err
			}
			input.contract.PromptHashes = append(input.contract.PromptHashes, issue142Digest([]byte(system+string(prompt))))
			input.contract.SchemaHashes = append(input.contract.SchemaHashes, issue142Digest(schema))
		}
	}
	return inputs, nil
}

func runIssue143(ctx context.Context, helper string, describe bool) error {
	inputs, err := issue143Prepare(ctx)
	if err != nil {
		return err
	}
	manifest := issue143Manifest{Models: issue143Models, HelperHash: issue143HelperHash, OutputTokens: 2048, TimeoutSeconds: 120, Reverse: issue142BidirectionalOrder}
	for _, input := range inputs {
		manifest.Contracts = append(manifest.Contracts, input.contract)
	}
	encoder := json.NewEncoder(os.Stdout)
	if describe {
		return encoder.Encode(manifest)
	}
	data, err := os.ReadFile(issue143ManifestPath)
	if err != nil {
		return err
	}
	var frozen issue143Manifest
	if err = json.Unmarshal(data, &frozen); err != nil {
		return err
	}
	if !reflect.DeepEqual(manifest, frozen) {
		return fmt.Errorf("preregistered manifest differs")
	}
	helperBytes, err := os.ReadFile(helper)
	if err != nil {
		return err
	}
	if issue142Digest(helperBytes) != issue143HelperHash {
		return fmt.Errorf("helper pin mismatch")
	}
	backends := make([]*issue143MeasuredBackend, len(issue143Models))
	models := make([]string, len(issue143Models))
	for i, spec := range issue143Models {
		var backend llm.OptionsBackend
		backend, models[i], err = openBackend("mlx", helper, "", spec)
		backends[i] = &issue143MeasuredBackend{OptionsBackend: backend}
		if err != nil {
			return fmt.Errorf("model %d not ready: %w", i, err)
		}
	}
	sequence := 0
	for run, reverse := range issue142BidirectionalOrder {
		for fi, input := range inputs {
			system, prompt, schema, err := issue142GenericForcedInput(input.prepared, input.contract.Candidates, reverse)
			if err != nil {
				return err
			}
			// Fixed rotation balances model execution position over fixtures/runs.
			for position := range issue143Models {
				mi := (position + fi + run) % len(issue143Models)
				sequence++
				callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
				call := issue142FollowupCall(callCtx, backends[mi], models[mi], "issue143", "fixed-two", input.item, input.prepared, input.contract.Candidates, run+1, system, prompt, schema)
				cancel()
				row := issue143Observation{issue142DiagnosticRow: call, FailureKind: backends[mi].FailureKind, Set: input.contract.Set, Category: input.contract.Category, Guardrail: input.contract.Guardrail, ModelIndex: mi, Sequence: sequence, Reverse: reverse, ContextTokens: input.contract.ContextTokens, HelperHash: issue143HelperHash}
				if err = encoder.Encode(row); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "call=%d/240 model=%d fixture=%s run=%d stop=%s gold=%t wall_ms=%.1f\n", sequence, mi, input.item.name, run+1, call.StopReason, call.CorrectSelection, call.WallMS)
			}
		}
	}
	return nil
}
