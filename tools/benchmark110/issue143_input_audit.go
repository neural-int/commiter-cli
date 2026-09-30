package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"os"
	"reflect"
)

// runIssue143InputAudit rebuilds synthetic inputs without loading a model.
// It records visible file evidence, not full prompts or generated responses.
func runIssue143InputAudit(ctx context.Context) error {
	inputs, err := issue143Prepare(ctx)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(issue143ManifestPath)
	if err != nil {
		return err
	}
	var frozen issue143Manifest
	if err := json.Unmarshal(data, &frozen); err != nil {
		return err
	}
	var current []issue143Contract
	for _, input := range inputs {
		current = append(current, input.contract)
	}
	if !reflect.DeepEqual(current, frozen.Contracts) {
		return fmt.Errorf("frozen contract mismatch")
	}
	type fileObservation struct {
		File            contextinput.File `json:"visible_file"`
		SourceDiffBytes int               `json:"source_diff_bytes"`
		SourceDiffHash  string            `json:"source_diff_sha256"`
		RawDiffExact    bool              `json:"raw_diff_exact"`
	}
	type observation struct {
		Fixture         string                        `json:"fixture"`
		PromptHashes    []string                      `json:"prompt_hashes"`
		SchemaHashes    []string                      `json:"schema_hashes"`
		CandidateIDs    []string                      `json:"candidate_ids"`
		SchemaFixed     bool                          `json:"schema_fixed_between_directions"`
		SystemFixed     bool                          `json:"system_fixed_between_directions"`
		RepositoryFixed bool                          `json:"repository_fixed_between_directions"`
		Files           []fileObservation             `json:"files"`
		RelationContext *contextinput.RelationContext `json:"visible_relation_context,omitempty"`
	}
	result := struct {
		ModelCalls   int           `json:"model_calls"`
		HumanReviews int           `json:"human_reviews"`
		Observations []observation `json:"observations"`
	}{}
	for _, input := range inputs {
		if input.contract.Set != "holdout" {
			continue
		}
		row := observation{Fixture: input.item.name, PromptHashes: input.contract.PromptHashes, SchemaHashes: input.contract.SchemaHashes}
		var systems []string
		var schemas []json.RawMessage
		var documents []contextinput.Document
		for _, reverse := range []bool{false, true} {
			system, prompt, schema, err := issue142GenericForcedInput(input.prepared, input.contract.Candidates, reverse)
			if err != nil {
				return err
			}
			var visible struct {
				Task            string                `json:"task"`
				Candidates      []issue142Candidate   `json:"candidates"`
				RepositoryInput contextinput.Document `json:"repository_input"`
			}
			if err := json.Unmarshal(prompt, &visible); err != nil {
				return err
			}
			systems = append(systems, system)
			schemas = append(schemas, schema)
			documents = append(documents, visible.RepositoryInput)
			if !reverse {
				for _, c := range visible.Candidates {
					row.CandidateIDs = append(row.CandidateIDs, c.ID)
				}
			}
		}
		row.SystemFixed = systems[0] == systems[1]
		row.SchemaFixed = string(schemas[0]) == string(schemas[1])
		row.RepositoryFixed = reflect.DeepEqual(documents[0], documents[1])
		row.RelationContext = documents[0].RelationContext
		if len(documents[0].Files) != len(input.item.files) {
			return fmt.Errorf("file count mismatch")
		}
		for i, f := range documents[0].Files {
			source := input.item.files[i].diff
			row.Files = append(row.Files, fileObservation{File: f, SourceDiffBytes: len(source), SourceDiffHash: issue142Digest([]byte(source)), RawDiffExact: f.RawDiff == source})
		}
		result.Observations = append(result.Observations, row)
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
