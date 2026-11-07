package planning

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/exitcode"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// ThreePhaseGenerator is the provisional Issue #143 candidate. It never repairs
// a failed phase, silently changes a partition, or returns a partial plan.
type ThreePhaseGenerator struct {
	Client ChatClient
	// Evidence is supplied by a trusted host observer, never by the model.
	// No built-in observer claims complete API/CLI/format coverage.
	Evidence        []BreakingEvidence
	GroupingHistory *GroupingHistory
}

type BreakingEvidence struct {
	ID      string   `json:"id"`
	FileIDs []string `json:"file_ids"`
	Fact    string   `json:"fact"`
}

const CandidateCycleTimeout = 120 * time.Second
const CandidateMaxFiles = 4

type candidateGroup struct {
	ID       string              `json:"id"`
	FileIDs  []string            `json:"-"`
	Files    []contextinput.File `json:"files"`
	Type     string              `json:"type,omitempty"`
	Breaking *bool               `json:"breaking,omitempty"`
}

func (g ThreePhaseGenerator) Generate(parent context.Context, prepared contextinput.Prepared, language Language, sensitive SensitiveValues) (result Result, err error) {
	defer func() {
		if err != nil && exitcode.Code(err) == exitcode.Internal {
			err = exitcode.New(exitcode.LLM, err.Error())
		}
	}()
	ids, err := threePhaseFileIDs(prepared, language)
	if err != nil {
		return result, err
	}
	if len(ids) > CandidateMaxFiles {
		return result, exitcode.New(exitcode.LLM, "provisional three-phase planner supports at most four selected files; select at most four files per run")
	}
	client, ok := g.Client.(optionsChatClient)
	if !ok {
		return result, errors.New("three-phase planner requires explicit generation-profile support")
	}
	if prepared.SummaryCount != 0 || prepared.EvidenceReductionCount != 0 || prepared.CompressionProfile != contextinput.CompressionNone {
		return result, exitcode.New(exitcode.LLM, "provisional three-phase planner does not support compressed input")
	}
	ctx, cancel := context.WithTimeout(parent, CandidateCycleTimeout)
	defer cancel()
	// Model preparation is retained for this invocation only. A successful plan
	// is returned only after the backend confirms that its resources are closed.
	if lifecycle, ok := g.Client.(llm.CycleBackend); ok {
		cycle, closeCycle, e := lifecycle.BeginCycle(ctx)
		if e != nil {
			return result, e
		}
		defer func() {
			if closeErr := closeCycle(); closeErr != nil {
				result = Result{}
				err = errors.Join(err, closeErr)
			}
		}()
		client = cycle
	}
	var original map[string]json.RawMessage
	if err := json.Unmarshal(prepared.Prompt, &original); err != nil {
		return result, errors.New("invalid candidate input")
	}
	// All phases use complete diffs; only Grouping adds minimal syntax context.
	// Normalize here as well as before Prepare so alternate callers cannot
	// reintroduce the legacy detailed-evidence representation.
	projected, err := contextinput.RawDiffProjection(prepared.Document)
	if err != nil {
		return result, err
	}
	files := projected.Files
	sort.Slice(files, func(i, j int) bool {
		pi, pj := candidatePath(files[i]), candidatePath(files[j])
		return naturalLess(pi, pj)
	})
	inverse := map[string]string{}
	seenPaths := map[string]bool{}
	visible := make([]string, len(files))
	labels := make([]string, len(files))
	for i := range files {
		id := fmt.Sprintf("F%d", i+1)
		if candidatePath(files[i]) == "" || seenPaths[candidatePath(files[i])] {
			return result, errors.New("candidate requires unique visible file paths")
		}
		seenPaths[candidatePath(files[i])] = true
		inverse[id] = files[i].ID
		files[i].ID = id
		visible[i] = id
		labels[i] = fmt.Sprintf("G%03d", i+1)
	}
	// Evidence must be host-provided, unique, nonempty and bound to selected IDs.
	evidenceByID := map[string]BreakingEvidence{}
	for _, e := range g.Evidence {
		if e.ID == "" || e.ID == "none" || e.ID == "unresolved" || strings.HasPrefix(e.ID, "git:") || e.Fact == "" || len(e.FileIDs) == 0 {
			return result, errors.New("invalid host breaking evidence")
		}
		if _, exists := evidenceByID[e.ID]; exists {
			return result, errors.New("duplicate host breaking evidence")
		}
		seen := map[string]bool{}
		for _, id := range e.FileIDs {
			if !containsString(ids, id) || seen[id] {
				return result, errors.New("host breaking evidence is not bound to selected files")
			}
			seen[id] = true
		}
		evidenceByID[e.ID] = e
	}
	var common004 map[string]any
	invoke := func(payload any, schema any, profile string, budget int, output any) error {
		if ctx.Err() != nil {
			return generationError([]Violation{IncompleteOutput})
		}
		safePhaseHistory(payload.(map[string]any), sensitive)
		encoded, e := json.Marshal(payload)
		if e != nil {
			return errors.New("cannot encode candidate input")
		}
		shape, e := json.Marshal(schema)
		if e != nil {
			return errors.New("cannot encode candidate schema")
		}
		options := llm.Options{ContextTokens: contextinput.Context64K, OutputTokens: budget, GenerationProfile: profile, LiteralNewlines: true, InputLayout: "harness004"}
		messages, e := harness004Messages(profile, encoded, shape, &common004)
		if e != nil {
			return e
		}
		if profile != "bounded-text" {
			options.HistoryInput, e = harness004OptionalHistory(messages)
			if e != nil {
				return e
			}
		}
		contextTokens := contextinput.Context64K

		result.Calls++
		if contextTokens > result.Telemetry.ContextTokens {
			result.Telemetry.ContextTokens = contextTokens
		}
		response, e := client.ChatWithOptions(ctx, messages, shape, options)
		if e != nil {
			if errors.Is(e, llm.ErrInputTooLarge) {
				return contextinput.ErrTooLarge
			}
			return generationError(nil)
		}
		if options.HistoryInput != nil {
			selected, selectErr := selectedAttributed004Shared(options.HistoryInput, response.SelectedHistoryIndices)
			if selectErr != nil {
				return selectErr
			}
			if profile == "bounded-grouping" {
				common004 = selected
			}
		}

		result.Telemetry.add(response, result.Calls == 1)
		if ctx.Err() != nil || response.StopReason != "completed" {
			return generationError([]Violation{IncompleteOutput})
		}
		candidate := []byte(response.Content)
		if sensitive.Contains(candidate) {
			return generationError([]Violation{SensitiveOutput})
		}
		if e := strictCandidateJSON(candidate, output); e != nil {
			return generationError([]Violation{InvalidSchema})
		}
		return nil
	}
	properties := map[string]any{}
	for _, id := range visible {
		properties[id] = map[string]any{"type": "string", "enum": labels}
	}
	groupingFiles, err := groupingProjection(files)
	if err != nil {
		return result, err
	}
	// Observed Git blocks are candidates for Category's semantic decision,
	// not confirmed incompatibilities. The normal CLI needs no injected witness.
	allEvidence := append([]BreakingEvidence(nil), g.Evidence...)
	for fileIndex, file := range groupingFiles {
		for _, block := range file.ChangeBlocks {
			allEvidence = append(allEvidence, BreakingEvidence{
				ID:      "git:" + file.FileID + ":" + block.BlockID,
				FileIDs: []string{inverse[file.FileID]},
				Fact:    fmt.Sprintf("Observed current Git change %s in %s, raw_diff lines %d-%d; incompatibility is not preclassified", block.BlockID, candidatePath(files[fileIndex]), block.DiffRange.StartLine, block.DiffRange.EndLine),
			})
		}
	}
	payload, err := groupingPayload(groupingFiles, inverse, g.GroupingHistory)
	if err != nil {
		return result, err
	}
	originalHistoryContexts := payload["agent_history_contexts"].([]AgentHistoryContext)
	if feedback, ok := original["regeneration_feedback"]; ok {
		payload["regeneration_feedback"] = feedback
	}
	var membership map[string]string
	if err := invoke(payload, candidateObject(properties, visible), "bounded-grouping", 4096, &membership); err != nil {
		return result, err
	}
	if len(membership) != len(files) {
		return result, generationError([]Violation{InvalidAssignment})
	}
	groups := []candidateGroup{}
	groupIndex := map[string]int{}
	for _, f := range files {
		label, ok := membership[f.ID]
		if !ok || !containsString(labels, label) {
			return result, generationError([]Violation{InvalidAssignment})
		}
		index, ok := groupIndex[label]
		if !ok {
			index = len(groups)
			groupIndex[label] = index
			groups = append(groups, candidateGroup{ID: fmt.Sprintf("G%03d", index+1)})
		}
		groups[index].FileIDs = append(groups[index].FileIDs, inverse[f.ID])
		groups[index].Files = append(groups[index].Files, f)
	}
	// Unknown membership keys must not survive restoration.
	for id := range membership {
		if inverse[id] == "" {
			return result, generationError([]Violation{InvalidAssignment})
		}
	}
	names := make([]string, len(groups))
	properties = map[string]any{}
	allowedRefs := map[string][]string{}
	for i, group := range groups {
		names[i] = group.ID
		refs := []string{"none", "unresolved"}
		types := allowedTypes
		confirmedWitness := false
		for _, e := range allEvidence {
			applies := false
			for _, id := range e.FileIDs {
				if containsString(group.FileIDs, id) {
					applies = true
				}
			}
			if applies {
				refs = append(refs, e.ID)
				if _, confirmed := evidenceByID[e.ID]; confirmed {
					confirmedWitness = true
				}
			}
		}
		if confirmedWitness {
			refs = refs[1:]
		} // A positive host witness rules out none.
		allowedRefs[group.ID] = refs
		properties[group.ID] = candidateObject(map[string]any{"type": map[string]any{"type": "string", "enum": types}, "breaking_evidence_ref": map[string]any{"type": "string", "enum": refs}}, []string{"type", "breaking_evidence_ref"})
	}
	metadata, err := metadataHistoryPayload(groups, groupingFiles, originalHistoryContexts)
	if err != nil {
		return result, err
	}
	metadata["summary_language"] = language

	if feedback, ok := original["regeneration_feedback"]; ok {
		metadata["regeneration_feedback"] = feedback
	}
	type category struct {
		Type      string `json:"type"`
		Reference string `json:"breaking_evidence_ref"`
	}
	// Group file_ids stay host-only; bind witnesses to the visible file IDs
	// instead so the model can still associate each witness with its files.
	visibleEvidence := []metadataBreakingEvidence{}
	for _, evidence := range allEvidence {
		projected := metadataBreakingEvidence{ID: evidence.ID, Fact: evidence.Fact}
		for _, originalID := range evidence.FileIDs {
			for visibleID, restoredID := range inverse {
				if restoredID == originalID {
					projected.FileIDs = append(projected.FileIDs, visibleID)
				}
			}
		}
		visibleEvidence = append(visibleEvidence, projected)
	}
	metadata["breaking_evidence_candidates"] = visibleEvidence
	var categories map[string]category
	if err := invoke(metadata, candidateObject(properties, names), "bounded-category", 4096, &categories); err != nil {
		return result, err
	}
	if len(categories) != len(groups) {
		return result, generationError([]Violation{InvalidSchema})
	}
	for i, group := range groups {
		c, ok := categories[group.ID]
		if !ok || !containsString(allowedTypes, c.Type) || !containsString(allowedRefs[group.ID], c.Reference) {
			return result, generationError([]Violation{InvalidSchema})
		}
		if c.Reference == "unresolved" {
			return result, exitcode.New(exitcode.LLM, "three-phase planner stopped: unresolved_breaking_evidence")
		}
		groups[i].Type = c.Type
		breaking := c.Reference != "none"
		groups[i].Breaking = &breaking
	}
	properties = map[string]any{}
	for _, name := range names {
		properties[name] = candidateObject(map[string]any{"scope": map[string]any{"type": "string", "minLength": 1, "maxLength": 16}, "summary": map[string]any{"type": "string", "minLength": 1, "maxLength": 48}}, []string{"scope", "summary"})
	}
	type text struct {
		Scope   string `json:"scope"`
		Summary string `json:"summary"`
	}
	// Text receives the finalized values; evidence selection belongs to Category.
	delete(metadata, "breaking_evidence_candidates")
	textGroups := metadata["groups"].([]metadataGroup)
	for i := range textGroups {
		c := categories[textGroups[i].ID]
		textGroups[i].Type = c.Type
		breaking := c.Reference != "none"
		textGroups[i].Breaking = &breaking
	}
	metadata["groups"] = textGroups
	var texts map[string]text
	if err := invoke(metadata, candidateObject(properties, names), "bounded-text", 4096, &texts); err != nil {
		return result, err
	}
	if len(texts) != len(groups) {
		return result, generationError([]Violation{InvalidSchema})
	}
	plan := Plan{SchemaVersion: SchemaVersion}
	for _, group := range groups {
		t, ok := texts[group.ID]
		if !ok || utf8.RuneCountInString(t.Scope) > 16 || utf8.RuneCountInString(t.Summary) > 48 {
			return result, generationError([]Violation{InvalidSchema})
		}
		c := categories[group.ID]
		plan.Commits = append(plan.Commits, Commit{Type: c.Type, Scope: t.Scope, Breaking: c.Reference != "none", Summary: t.Summary, FileIDs: group.FileIDs})
	}
	// Only the authoritative final validator permits a plan to leave this method.
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

func candidatePath(file contextinput.File) string {
	if file.NewPath != nil {
		return *file.NewPath
	}
	if file.OldPath != nil {
		return *file.OldPath
	}
	return ""
}

func candidateObject(properties map[string]any, required []string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}
func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func strictCandidateJSON(data []byte, target any) error {
	// Recursive duplicate-key detection runs before decoding into typed maps.
	d := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				s, ok := key.(string)
				if !ok || seen[s] {
					return errors.New("duplicate JSON key")
				}
				seen[s] = true
				if err := walk(); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := walk(); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func threePhaseFileIDs(prepared contextinput.Prepared, language Language) ([]string, error) {
	if err := validateLanguage(language); err != nil {
		return nil, err
	}
	ids := []string{}
	for _, file := range prepared.Document.Files {
		ids = append(ids, file.ID)
	}
	return exactIDs(ids)
}
