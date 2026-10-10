package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type structureObservation struct {
	Workload      string  `json:"workload"`
	Files         int     `json:"files"`
	Complete      bool    `json:"complete_assignment"`
	Replay        bool    `json:"byte_tree_replay"`
	GitUnchanged  bool    `json:"git_unchanged"`
	Groups        int     `json:"provisional_groups"`
	StaticQuality quality `json:"static_singleton_reference_quality"`
	BaseTree      string  `json:"base_tree"`
	FinalTree     string  `json:"final_tree"`
	LLMCalls      int     `json:"llm_calls"`
}

// Structural receipts and static singleton/gold comparison need no inference.
// They are never reported as completed metadata plans or model accuracy.
func structuralObservation(f fixture) (structureObservation, error) {
	o := structureObservation{Workload: f.Name, Files: len(f.Files)}
	root, e := makeRepository(f)
	if e != nil {
		return o, e
	}
	defer os.RemoveAll(root)
	snapshot, e := gitstate.Collect(root, gitstate.Options{})
	if e != nil {
		return o, e
	}
	before, e := repositoryDigest(root)
	if e != nil {
		return o, e
	}
	preview, e := gitstate.PreviewFileFirst(snapshot, gitstate.Options{})
	if e != nil {
		return o, e
	}
	after, e := repositoryDigest(root)
	if e != nil {
		return o, e
	}
	o.GitUnchanged = before == after
	o.Groups = len(preview.Groups)
	seen := map[string]int{}
	provisional := planning.Plan{}
	for _, g := range preview.Groups {
		seen[g.FileID]++
		provisional.Commits = append(provisional.Commits, planning.Commit{FileIDs: []string{g.FileID}})
	}
	o.Complete = len(seen) == len(snapshot.Changes) && len(snapshot.Changes) == len(f.Files)
	for _, c := range snapshot.Changes {
		o.Complete = o.Complete && seen[c.ID] == 1
	}
	o.Replay = preview.BaseTree != "" && preview.FinalTree != "" && len(preview.StageTrees) == len(f.Files)
	o.BaseTree = preview.BaseTree
	o.FinalTree = preview.FinalTree
	o.StaticQuality = partitionQuality(f, snapshot, provisional)
	if !o.GitUnchanged || !o.Complete || !o.Replay {
		return o, errors.New("structural observation failed")
	}
	return o, nil
}
func runStructures(selected []fixture, filter string) error {
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	_ = os.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	encoder := json.NewEncoder(os.Stdout)
	found := false
	for _, f := range selected {
		if filter != "" && f.Name != filter {
			continue
		}
		found = true
		o, e := structuralObservation(f)
		if e != nil {
			return e
		}
		if e = encoder.Encode(o); e != nil {
			return e
		}
	}
	if !found {
		return errors.New("unknown structural fixture")
	}
	return nil
}
