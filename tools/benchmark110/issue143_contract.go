package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"
)

const issue143ContractManifestPath = "docs/benchmarks/issue-143-contract-manifest-2026-09-30.json"

var issue143ContractModelIndices = []int{0, 1, 2, 4}
var issue143ContractArms = []string{"partition-list", "file-membership"}

type issue143MembershipRow struct {
	FileID   string   `json:"file_id"`
	GroupIDs []string `json:"group_ids"`
}
type issue143MembershipTable struct {
	CandidateIDs  []string                `json:"candidate_ids"`
	FileGroupRows []issue143MembershipRow `json:"file_group_rows"`
}
type issue143ContractHash struct {
	Fixture        string `json:"fixture"`
	Arm            string `json:"arm"`
	Reverse        bool   `json:"reverse"`
	PromptHash     string `json:"prompt_sha256"`
	SchemaHash     string `json:"schema_sha256"`
	RepositoryHash string `json:"repository_sha256"`
}
type issue143ContractManifest struct {
	Parent       issue143Manifest       `json:"parent"`
	ModelIndices []int                  `json:"model_indices"`
	Arms         []string               `json:"arms"`
	Hashes       []issue143ContractHash `json:"hashes"`
}

func issue143ContractInput(input issue143Input, arm string, reverse bool) (string, []byte, json.RawMessage, error) {
	system, prompt, schema, err := issue142GenericForcedInput(input.prepared, input.contract.Candidates, reverse)
	if err != nil || arm == "partition-list" {
		return system, prompt, schema, err
	}
	if arm != "file-membership" {
		return "", nil, nil, fmt.Errorf("unknown contract arm")
	}
	var request map[string]json.RawMessage
	if err := json.Unmarshal(prompt, &request); err != nil {
		return "", nil, nil, err
	}
	var offered []issue142Candidate
	if err := json.Unmarshal(request["candidates"], &offered); err != nil {
		return "", nil, nil, err
	}
	ids := []string{}
	for _, f := range input.prepared.Document.Files {
		ids = append(ids, f.ID)
	}
	table := issue143MembershipTable{}
	memberships := make([]map[string]string, len(offered))
	for ci, c := range offered {
		table.CandidateIDs = append(table.CandidateIDs, c.ID)
		canonical, _, err := issue142Canonical(c.Groups, ids)
		if err != nil {
			return "", nil, nil, err
		}
		memberships[ci] = map[string]string{}
		for gi, g := range canonical {
			for _, id := range g {
				memberships[ci][id] = fmt.Sprintf("G%03d", gi+1)
			}
		}
	}
	for _, id := range ids {
		row := issue143MembershipRow{FileID: id}
		for _, m := range memberships {
			row.GroupIDs = append(row.GroupIDs, m[id])
		}
		table.FileGroupRows = append(table.FileGroupRows, row)
	}
	request["candidates"], err = json.Marshal(table)
	if err != nil {
		return "", nil, nil, err
	}
	prompt, err = json.Marshal(struct {
		Task            json.RawMessage `json:"task"`
		Candidates      json.RawMessage `json:"candidates"`
		RepositoryInput json.RawMessage `json:"repository_input"`
	}{request["task"], request["candidates"], request["repository_input"]})
	system += " Candidate encoding is a file membership table. candidate_ids gives the column order. Each file_group_rows row has group_ids in that column order. Within one candidate column, equal group IDs mean the files share one commit; different group IDs mean separate commits. Group IDs are local labels, not rankings. Select a candidate_id."
	return system, prompt, schema, err
}

func issue143BuildContractManifest(ctx context.Context) ([]issue143Input, issue143ContractManifest, error) {
	inputs, err := issue143Prepare(ctx)
	if err != nil {
		return nil, issue143ContractManifest{}, err
	}
	data, err := os.ReadFile(issue143ManifestPath)
	if err != nil {
		return nil, issue143ContractManifest{}, err
	}
	var parent issue143Manifest
	if err = json.Unmarshal(data, &parent); err != nil {
		return nil, issue143ContractManifest{}, err
	}
	actual := []issue143Contract{}
	for _, input := range inputs {
		actual = append(actual, input.contract)
	}
	if !reflect.DeepEqual(actual, parent.Contracts) {
		return nil, issue143ContractManifest{}, fmt.Errorf("parent inputs differ")
	}
	manifest := issue143ContractManifest{Parent: parent, ModelIndices: issue143ContractModelIndices, Arms: issue143ContractArms}
	for _, input := range inputs {
		for _, arm := range issue143ContractArms {
			for _, reverse := range []bool{false, true} {
				system, prompt, schema, err := issue143ContractInput(input, arm, reverse)
				if err != nil {
					return nil, manifest, err
				}
				var request map[string]json.RawMessage
				if err = json.Unmarshal(prompt, &request); err != nil {
					return nil, manifest, err
				}
				manifest.Hashes = append(manifest.Hashes, issue143ContractHash{Fixture: input.item.name, Arm: arm, Reverse: reverse, PromptHash: issue142Digest([]byte(system + string(prompt))), SchemaHash: issue142Digest(schema), RepositoryHash: issue142Digest(request["repository_input"])})
			}
		}
	}
	return inputs, manifest, nil
}

func runIssue143Contract(ctx context.Context, helper string, describe bool) error {
	inputs, manifest, err := issue143BuildContractManifest(ctx)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	if describe {
		return enc.Encode(manifest)
	}
	data, err := os.ReadFile(issue143ContractManifestPath)
	if err != nil {
		return err
	}
	var frozen issue143ContractManifest
	if err = json.Unmarshal(data, &frozen); err != nil {
		return err
	}
	if !reflect.DeepEqual(manifest, frozen) {
		return fmt.Errorf("contract manifest differs")
	}
	helperData, err := os.ReadFile(helper)
	if err != nil {
		return err
	}
	if issue142Digest(helperData) != issue143HelperHash {
		return fmt.Errorf("helper pin mismatch")
	}
	backends := map[int]*issue143MeasuredBackend{}
	models := map[int]string{}
	for _, mi := range issue143ContractModelIndices {
		backend, model, err := openBackend("mlx", helper, "", issue143Models[mi])
		if err != nil {
			return err
		}
		backends[mi] = &issue143MeasuredBackend{OptionsBackend: backend}
		models[mi] = model
	}
	sequence := 0
	for run, reverse := range issue142BidirectionalOrder {
		for fi, input := range inputs {
			for position := range issue143ContractModelIndices {
				mi := issue143ContractModelIndices[(position+fi+run)%len(issue143ContractModelIndices)]
				for offset := range issue143ContractArms {
					arm := issue143ContractArms[(offset+mi+fi+run)%len(issue143ContractArms)]
					system, prompt, schema, err := issue143ContractInput(input, arm, reverse)
					if err != nil {
						return err
					}
					callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
					call := issue142FollowupCall(callCtx, backends[mi], models[mi], "issue143-contract", arm, input.item, input.prepared, input.contract.Candidates, run+1, system, prompt, schema)
					cancel()
					sequence++
					row := issue143Observation{issue142DiagnosticRow: call, FailureKind: backends[mi].FailureKind, Set: input.contract.Set, Category: input.contract.Category, Guardrail: input.contract.Guardrail, ModelIndex: mi, Sequence: sequence, Reverse: reverse, ContextTokens: input.contract.ContextTokens, HelperHash: issue143HelperHash}
					if err := enc.Encode(row); err != nil {
						return err
					}
					fmt.Fprintf(os.Stderr, "call=%d/384 model=%d arm=%s fixture=%s run=%d stop=%s gold=%t wall_ms=%.1f\n", sequence, mi, arm, input.item.name, run+1, call.StopReason, call.CorrectSelection, call.WallMS)
				}
			}
		}
	}
	return nil
}
