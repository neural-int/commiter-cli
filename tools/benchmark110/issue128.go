package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"github.com/natsuki0413/commiter-cli/internal/relation"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type issue128Options struct {
	backendName, fixtureName, helper, ollamaModel string
	repeats, outputBudget, manyFileCount          int
	timeout                                       time.Duration
	describe                                      bool
	modelSpec                                     mlxmodel.Spec
}

type issue128Metrics struct {
	PreprocessMS        float64               `json:"preprocess_ms"`
	CandidateInputBytes int                   `json:"candidate_input_bytes"`
	NodeCount           int                   `json:"node_count"`
	EdgeCount           int                   `json:"edge_count"`
	DensePairs          int64                 `json:"dense_pairs"`
	CandidatePairs      int64                 `json:"candidate_pairs"`
	ReducedPairs        int64                 `json:"reduced_pairs"`
	IsolatedFiles       int                   `json:"isolated_files"`
	EdgeRecall          *fraction             `json:"edge_recall,omitempty"`
	HintRecall          *fraction             `json:"hint_recall,omitempty"`
	HardHit             int                   `json:"hard_hit"`
	HardMiss            int                   `json:"hard_miss"`
	SoftHit             int                   `json:"soft_hit"`
	SoftMiss            int                   `json:"soft_miss"`
	HintHit             int                   `json:"hint_hit"`
	HintMiss            int                   `json:"hint_miss"`
	Ambiguous           int                   `json:"ambiguous"`
	Unresolved          int                   `json:"unresolved"`
	Unsupported         int                   `json:"unsupported"`
	EdgesByKind         map[relation.Kind]int `json:"edges_by_kind"`
	Omitted             bool                  `json:"relation_context_omitted"`
	OmissionReason      string                `json:"omission_reason,omitempty"`
}

type fraction struct {
	Hit   int `json:"hit"`
	Total int `json:"total"`
}

type issue128Row struct {
	observation
	Arm string `json:"arm"`
	issue128Metrics
}

func issue128Fixtures(manyFileCount int) []fixture {
	all := fixtures()
	selected := make([]fixture, 0, 9)
	for _, item := range all {
		if item.name == "near_64k" {
			continue
		}
		if item.name == "many_files" {
			item.files = manyFiles(manyFileCount)
		}
		selected = append(selected, item)
	}
	selected = append(selected,
		fixture{"cross_directory", planning.English, []fileSpec{
			{"src/auth.js", "+export function login() { return true; }\n"},
			{"tests/auth.test.js", "+import { login } from '../src/auth.js';\n+if (!login()) throw new Error('login');\n"},
		}, [][]string{{"F001", "F002"}}},
		fixture{"same_directory_independent", planning.English, []fileSpec{
			{"shared/login.go", "+func Login() bool { return true }\n"},
			{"shared/report.go", "+func Report() bool { return true }\n"},
		}, [][]string{{"F001"}, {"F002"}}},
		fixture{"relation_diagnostics", planning.English, []fileSpec{
			{"src/main.js", "+import './common';\n+import './missing';\n"},
			{"src/common.js", "+export const common = 1;\n"},
			{"src/common.jsx", "+export const common = 1;\n"},
			{"src/legacy.rs", "+use crate::common;\n"},
		}, nil},
		fixture{"rename", planning.English, []fileSpec{
			{"src/auth_new.go", "+func Authenticate() bool { return true }\n"},
		}, [][]string{{"F001"}}},
	)
	return selected
}

func issue128Prepare(ctx context.Context, item fixture, withRelations bool, outputBudget int) (contextinput.Prepared, []string, issue128Metrics, error) {
	start := time.Now()
	doc := contextinput.Document{SchemaVersion: contextinput.SchemaVersion, Repository: contextinput.Repository{Head: "synthetic-head", Branch: "benchmark", IndexIdentity: "synthetic-index"}}
	ids := make([]string, len(item.files))
	changes := make([]gitstate.Change, len(item.files))
	relationFiles := make([]relation.File, len(item.files))
	for i, spec := range item.files {
		id := fmt.Sprintf("F%03d", i+1)
		ids[i] = id
		path := spec.path
		status := "M"
		var oldPath *string
		if item.name == "rename" && i == 0 {
			status = "renamed"
			old := "src/auth_old.go"
			oldPath = &old
		}
		diff := "@@ -1,1 +1,2 @@\n" + spec.diff
		hash := sha256.Sum256([]byte(diff))
		language := "go"
		switch filepath.Ext(path) {
		case ".js", ".jsx":
			language = "javascript"
		case ".ts":
			language = "typescript"
		case ".rs":
			language = "rust"
		}
		doc.Files = append(doc.Files, contextinput.File{ID: id, Status: status, OldPath: oldPath, NewPath: &path, ChangeHash: hex.EncodeToString(hash[:]), WorktreeKind: "file", Language: language, Size: int64(len(diff)), Unstaged: true, Mode: syntax.ModeRawDiff, RawDiff: diff})
		changes[i] = gitstate.Change{ID: id, Status: status, OldPath: oldPath, NewPath: &path, Language: language, WorktreeKind: "file"}
		relationFiles[i] = relation.File{Change: changes[i]}
		if language == "javascript" || language == "typescript" || language == "rust" {
			relationFiles[i].Content = []byte(strings.ReplaceAll(spec.diff, "\n+", "\n"))
			relationFiles[i].Content = []byte(strings.TrimPrefix(string(relationFiles[i].Content), "+"))
		}
	}
	metrics := issue128Metrics{}
	if withRelations {
		extracted, err := relation.Extract(relationFiles)
		if err != nil {
			return contextinput.Prepared{}, nil, issue128Metrics{}, err
		}
		graph, err := relation.BuildGraph(changes, extracted)
		if err != nil {
			return contextinput.Prepared{}, nil, issue128Metrics{}, err
		}
		metrics = measureIssue128Graph(item, extracted, graph)
		candidateContext := contextinput.RelationContextFromGraph(graph)
		serialized, err := json.Marshal(candidateContext)
		if err != nil {
			return contextinput.Prepared{}, nil, issue128Metrics{}, err
		}
		metrics.CandidateInputBytes = len(serialized)
		if len(graph.Edges) == 0 && len(graph.Hints) == 0 && len(graph.Observations) == 0 {
			doc.RelationContextStatus = &contextinput.RelationContextStatus{Omitted: true, Reason: "ineffective"}
		} else {
			doc.RelationContext = candidateContext
		}
	}
	system, err := planning.InitialSystemMessage(ids)
	if err != nil {
		return contextinput.Prepared{}, nil, issue128Metrics{}, err
	}
	prepared, err := contextinput.Prepare(ctx, doc, contextinput.BudgetConfig{Context: "64k", MaxContextTokens: contextinput.Context64K, PromptOverheadBytes: len(system)}, planning.Renderer(item.language), nil)
	if err != nil {
		return contextinput.Prepared{}, nil, issue128Metrics{}, err
	}
	if outputBudget > 0 {
		prepared.Budget.ReservedOutputTokens = outputBudget
	}
	metrics.PreprocessMS = milliseconds(time.Since(start))
	metrics.Omitted = prepared.RelationContextOmitted
	if prepared.Document.RelationContextStatus != nil {
		metrics.OmissionReason = prepared.Document.RelationContextStatus.Reason
	}
	return prepared, ids, metrics, nil
}

func measureIssue128Graph(item fixture, extracted relation.Result, graph relation.CandidateGraph) issue128Metrics {
	metrics := issue128Metrics{NodeCount: graph.Statistics.NodeCount, EdgeCount: graph.Statistics.EdgeCount, DensePairs: graph.Statistics.DensePairBaseline, CandidatePairs: graph.Statistics.CandidatePairCount, ReducedPairs: graph.Statistics.ReducedPairCount, Ambiguous: graph.Statistics.ObservationsByOutcome[relation.Ambiguous], Unresolved: graph.Statistics.ObservationsByOutcome[relation.Unresolved], Unsupported: graph.Statistics.ObservationsByOutcome[relation.Unsupported], EdgesByKind: graph.Statistics.EdgesByKind}
	for _, component := range graph.Components {
		if len(component.FileIDs) == 1 {
			metrics.IsolatedFiles++
		}
	}
	if item.name == "rename" {
		for _, edge := range extracted.Relations {
			if edge.Kind == relation.GitRename && edge.Class == relation.Hard && edge.SourceID == "F001" && edge.TargetID == "F001" && len(graph.Nodes) == 1 && graph.Nodes[0].Renamed {
				metrics.HardHit++
			}
		}
		if metrics.HardHit != 1 {
			metrics.HardMiss = 1
		}
	}
	reference := item.reference
	if reference == nil {
		return metrics
	}
	groupByID := make(map[string]int)
	for group, ids := range reference {
		for _, id := range ids {
			groupByID[id] = group
		}
	}
	for _, edge := range graph.Edges {
		hit := groupByID[edge.SourceID] == groupByID[edge.TargetID]
		if edge.Class == relation.Hard {
			if hit {
				metrics.HardHit++
			} else {
				metrics.HardMiss++
			}
		} else if hit {
			metrics.SoftHit++
		} else {
			metrics.SoftMiss++
		}
	}
	seenHintPairs := make(map[[2]string]bool)
	for _, hint := range graph.Hints {
		for i := range hint.FileIDs {
			for j := i + 1; j < len(hint.FileIDs); j++ {
				pair := [2]string{hint.FileIDs[i], hint.FileIDs[j]}
				if pair[0] > pair[1] {
					pair[0], pair[1] = pair[1], pair[0]
				}
				if seenHintPairs[pair] {
					continue
				}
				seenHintPairs[pair] = true
				if groupByID[pair[0]] == groupByID[pair[1]] {
					metrics.HintHit++
				} else {
					metrics.HintMiss++
				}
			}
		}
	}
	edgeRecall, hintRecall := fraction{}, fraction{}
	for _, group := range reference {
		for i := range group {
			for j := i + 1; j < len(group); j++ {
				edgeRecall.Total++
				hintRecall.Total++
				if issue128SameComponent(graph.Components, group[i], group[j]) {
					edgeRecall.Hit++
				}
				for _, hint := range graph.Hints {
					if contains(hint.FileIDs, group[i]) && contains(hint.FileIDs, group[j]) {
						hintRecall.Hit++
						break
					}
				}
			}
		}
	}
	metrics.EdgeRecall, metrics.HintRecall = &edgeRecall, &hintRecall
	return metrics
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func issue128SameComponent(components []relation.CandidateComponent, left, right string) bool {
	for _, component := range components {
		if contains(component.FileIDs, left) && contains(component.FileIDs, right) {
			return true
		}
	}
	return false
}

func runIssue128(ctx context.Context, options issue128Options) error {
	var backend llm.OptionsBackend
	var model string
	var err error
	if !options.describe {
		backend, model, err = openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	matched := false
	for _, item := range issue128Fixtures(options.manyFileCount) {
		if options.fixtureName == "all" && item.name == "rename" && !options.describe {
			continue // The hard self-relation is preprocessing-only; the planner omits its empty graph.
		}
		if options.fixtureName != "all" && options.fixtureName != item.name {
			continue
		}
		matched = true
		for run := 1; run <= options.repeats; run++ {
			arms := []bool{false, true}
			if run%2 == 0 {
				arms[0], arms[1] = arms[1], arms[0]
			}
			for _, withRelations := range arms {
				arm := "before"
				if withRelations {
					arm = "after"
				}
				prepared, ids, metrics, err := issue128Prepare(ctx, item, withRelations, options.outputBudget)
				if err != nil {
					return fmt.Errorf("%s %s preparation: %w", item.name, arm, err)
				}
				row := issue128Row{observation: observation{Backend: options.backendName, Fixture: item.name, Run: run, Model: model, ContextTokens: prepared.Budget.ContextTokens, PromptBytes: prepared.Budget.PromptBytes, OriginalBytes: prepared.OriginalPromptBytes, SummaryStage: string(prepared.SummaryStage), OutputBudget: prepared.Budget.ReservedOutputTokens}, Arm: arm, issue128Metrics: metrics}
				if !options.describe {
					measured := &measuredClient{backend: backend, ids: ids, language: item.language}
					runContext, cancel := context.WithTimeout(ctx, options.timeout)
					start := time.Now()
					result, generateErr := (planning.Generator{Client: measured}).Generate(runContext, prepared, item.language, planning.SensitiveValues{})
					cancel()
					row.WallMS = milliseconds(time.Since(start))
					row.Calls, row.Repaired, row.Succeeded, row.Requests = result.Calls, result.Repaired, generateErr == nil, measured.requests
					if generateErr != nil {
						row.Failure = classifyIssue128Failure(measured.requests)
					} else {
						for _, commit := range result.Plan.Commits {
							row.Groups = append(row.Groups, commit.FileIDs)
							row.Types = append(row.Types, commit.Type)
						}
						if item.reference != nil {
							match := sameGroups(row.Groups, item.reference)
							row.ReferenceGrouping = &match
						}
					}
				}
				if err := encoder.Encode(row); err != nil {
					return err
				}
			}
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #128 fixture %q", options.fixtureName)
	}
	return nil
}

func classifyIssue128Failure(requests []requestMetric) string {
	if len(requests) == 0 {
		return "no_backend_response"
	}
	last := requests[len(requests)-1]
	if last.StopReason != "completed" {
		return last.StopReason
	}
	if len(last.Violations) > 0 {
		return string(last.Violations[0])
	}
	return "planning_rejected"
}
