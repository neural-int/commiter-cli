package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/exitcode"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
)

// FileFirstGenerator is the gated metadata-only path for Issue #167. It does
// not collect or mutate Git. Callers must verify the immutable Git snapshot.
// Every selected file remains one complete group, including opaque changes.
type FileFirstGenerator struct {
	Client   ChatClient
	Evidence []BreakingEvidence
}

func (g FileFirstGenerator) Generate(parent context.Context, prepared contextinput.Prepared, language Language, sensitive SensitiveValues) (result Result, err error) {
	defer func() {
		if err != nil {
			result.Plan = Plan{}
			if exitcode.Code(err) == exitcode.Internal {
				err = exitcode.New(exitcode.LLM, err.Error())
			}
		}
	}()
	ids, err := validatePrepared(prepared, language)
	if err != nil {
		return result, err
	}
	if len(ids) > gitstate.FileFirstMaxFiles {
		return result, errors.New("file-first planner supports at most sixteen selected files")
	}
	client, ok := g.Client.(optionsChatClient)
	if !ok || prepared.Budget.ContextTokens != contextinput.Context16K {
		return result, errors.New("file-first planner requires explicit generation profiles and fixed 16k context")
	}
	if prepared.SummaryCount != 0 || prepared.EvidenceReductionCount != 0 || prepared.CompressionProfile != contextinput.CompressionNone {
		return result, errors.New("file-first planner does not support compressed input")
	}
	if err := validateCandidateEvidence(g.Evidence, ids); err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(parent, CandidateCycleTimeout)
	defer cancel()
	var original map[string]json.RawMessage
	if err := json.Unmarshal(prepared.Prompt, &original); err != nil {
		return result, errors.New("invalid file-first input")
	}
	files := append([]contextinput.File(nil), prepared.Document.Files...)
	// Stable IDs determine stage order. No path/dependency must-link inference.
	sort.Slice(files, func(i, j int) bool { return files[i].ID < files[j].ID })
	groups := make([]candidateGroup, len(files))
	paths := map[string]bool{}
	for i, file := range files {
		path := candidatePath(file)
		if path == "" || paths[path] {
			return result, errors.New("file-first requires unique visible file paths")
		}
		paths[path] = true
		id := file.ID
		file.ID = "path:" + path
		groups[i] = candidateGroup{ID: fmt.Sprintf("G%03d", i+1), FileIDs: []string{id}, Files: []contextinput.File{file}}
	}
	invoke := candidateInvocation(ctx, client, prepared.Budget.ContextTokens, sensitive, &result)
	plan := Plan{SchemaVersion: SchemaVersion}
	for offset := 0; offset < len(groups); offset += CandidateMaxFiles {
		packet := groups[offset:min(offset+CandidateMaxFiles, len(groups))]
		packetIDs := make([]string, len(packet))
		for i, group := range packet {
			packetIDs[i] = group.FileIDs[0]
		}
		part, e := candidateMetadata(ctx, packet, packetIDs, g.Evidence, original, language, sensitive, invoke)
		if e != nil {
			return result, e
		}
		plan.Commits = append(plan.Commits, part.Commits...)
	}
	encoded, _ := json.Marshal(plan)
	final, violations := Validate(encoded, ids, sensitive, language)
	if ctx.Err() != nil {
		return result, generationError([]Violation{IncompleteOutput})
	}
	if len(violations) != 0 {
		return result, generationError(violations)
	}
	result.Plan = final
	return result, nil
}
