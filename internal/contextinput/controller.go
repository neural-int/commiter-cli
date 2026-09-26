package contextinput

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/relation"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type SummaryStage string

const (
	SummaryNone  SummaryStage = "none"
	SummaryFile  SummaryStage = "file"
	SummaryHunk  SummaryStage = "hunk"
	SummaryChunk SummaryStage = "chunk"
)

type Summarizer interface {
	Summarize(context.Context, SummaryStage, Document) (Document, error)
}

type profiledSummarizer interface {
	SummarizeProfile(context.Context, CompressionProfile, Document) (Document, error)
}

type Prepared struct {
	Document                  Document           `json:"document"`
	Prompt                    []byte             `json:"-"`
	Budget                    Budget             `json:"budget"`
	SummaryStage              SummaryStage       `json:"summary_stage"`
	CompressionProfile        CompressionProfile `json:"compression_profile,omitempty"`
	SummaryCount              int                `json:"summary_count"`
	SummaryDuration           time.Duration      `json:"-"`
	OriginalPromptBytes       int                `json:"original_prompt_bytes"`
	EvidenceReductionCount    int                `json:"evidence_reduction_count"`
	EvidenceReductionDuration time.Duration      `json:"-"`
	EvidenceBeforeBytes       int                `json:"evidence_before_bytes"`
	EvidenceAfterBytes        int                `json:"evidence_after_bytes"`
	RelationContextOmitted    bool               `json:"relation_context_omitted,omitempty"`
}

// Prepare renders and measures the exact final prompt. Oversized input is
// summarized in the specified file -> hunk -> chunk order. Git identities are
// exact throughout; any later evidence reduction must satisfy its audited
// provenance and structural-coverage contract.
func Prepare(ctx context.Context, document Document, config BudgetConfig, render Renderer, summarizer Summarizer) (Prepared, error) {
	if render == nil {
		return Prepared{}, errors.New("prompt renderer is required")
	}
	if err := validateRelationContext(document); err != nil {
		fallbackDocument := cloneDocument(document)
		fallbackDocument.RelationContext = nil
		fallbackDocument.RelationContextStatus = omittedRelationStatus(document.RelationContext, "invalid")
		prepared, fallbackErr := Prepare(ctx, fallbackDocument, config, render, summarizer)
		prepared.RelationContextOmitted = true
		return prepared, fallbackErr
	}
	original := cloneDocument(document)
	current := cloneDocument(document)
	var summaryDuration time.Duration
	var originalPromptBytes int
	if summarizer == nil {
		standard := NewHierarchicalSummarizer()
		summarizer = standard
	}
	baseConfig := normalBudgetConfig(config)
	progress := func(stage SummaryStage, profile CompressionProfile, count int) Prepared {
		return Prepared{SummaryStage: stage, CompressionProfile: profile, SummaryCount: count, SummaryDuration: summaryDuration, OriginalPromptBytes: originalPromptBytes, RelationContextOmitted: current.RelationContextStatus != nil && current.RelationContextStatus.Omitted}
	}
	renderCurrent := func(stage SummaryStage, profile CompressionProfile, count int, budgetConfig BudgetConfig) (Prepared, error) {
		prompt, err := render(cloneDocument(current))
		if err != nil {
			return progress(stage, profile, count), fmt.Errorf("cannot render planning input: %w", err)
		}
		if originalPromptBytes == 0 {
			originalPromptBytes = len(prompt) + config.PromptOverheadBytes
		}
		budget, err := SelectContext(prompt, len(original.Files), budgetConfig)
		if err != nil {
			return progress(stage, profile, count), err
		}
		return Prepared{
			Document: cloneDocument(current), Prompt: append([]byte(nil), prompt...), Budget: budget,
			SummaryStage: stage, CompressionProfile: profile, SummaryCount: count,
			SummaryDuration: summaryDuration, OriginalPromptBytes: originalPromptBytes,
			RelationContextOmitted: current.RelationContextStatus != nil && current.RelationContextStatus.Omitted,
		}, nil
	}
	prepared, err := renderCurrent(SummaryNone, CompressionNone, 0, baseConfig)
	if err == nil {
		return prepared, nil
	}
	if !errors.Is(err, ErrTooLarge) {
		return prepared, err
	}
	if original.RelationContext != nil {
		fallbackDocument := cloneDocument(original)
		fallbackDocument.RelationContext = nil
		fallbackDocument.RelationContextStatus = omittedRelationStatus(original.RelationContext, "over_budget")
		fallback, fallbackErr := Prepare(ctx, fallbackDocument, config, render, summarizer)
		fallback.RelationContextOmitted = true
		return fallback, fallbackErr
	}

	summaryCount := 0
	for _, stage := range []SummaryStage{SummaryFile, SummaryHunk} {
		summaryCount++
		started := time.Now()
		next, summarizeErr := summarizer.Summarize(ctx, stage, cloneDocument(current))
		summaryDuration += time.Since(started)
		if summarizeErr != nil {
			return progress(stage, CompressionNone, summaryCount), fmt.Errorf("%s summary failed: %w", stage, summarizeErr)
		}
		if preserveErr := ValidatePreserved(original, next); preserveErr != nil {
			return progress(stage, CompressionNone, summaryCount), fmt.Errorf("%s summary is incomplete: %w", stage, preserveErr)
		}
		current = cloneDocument(next)
		prepared, err = renderCurrent(stage, CompressionNone, summaryCount, baseConfig)
		if err == nil {
			return prepared, nil
		}
		if !errors.Is(err, ErrTooLarge) {
			return prepared, err
		}
	}

	compressionProfile := CompressionNone
	if profiles, ok := summarizer.(profiledSummarizer); ok {
		hunkDocument := cloneDocument(current)
		for _, candidate := range compressionProfiles {
			summaryCount++
			started := time.Now()
			next, summarizeErr := profiles.SummarizeProfile(ctx, candidate.name, cloneDocument(hunkDocument))
			summaryDuration += time.Since(started)
			if summarizeErr != nil {
				return progress(SummaryChunk, candidate.name, summaryCount), fmt.Errorf("%s summary failed: %w", SummaryChunk, summarizeErr)
			}
			if preserveErr := ValidatePreserved(original, next); preserveErr != nil {
				return progress(SummaryChunk, candidate.name, summaryCount), fmt.Errorf("%s summary is incomplete: %w", SummaryChunk, preserveErr)
			}
			current = cloneDocument(next)
			compressionProfile = candidate.name
			candidateConfig := config
			if candidate.name == CompressionLight {
				candidateConfig = baseConfig
			}
			prepared, err = renderCurrent(SummaryChunk, compressionProfile, summaryCount, candidateConfig)
			if err == nil {
				return prepared, nil
			}
			if !errors.Is(err, ErrTooLarge) {
				return prepared, err
			}
			if candidate.name == CompressionLight && supportsExpandedBudget(config) {
				prepared, err = renderCurrent(SummaryChunk, compressionProfile, summaryCount, config)
				if err == nil {
					return prepared, nil
				}
				if !errors.Is(err, ErrTooLarge) {
					return prepared, err
				}
			}
		}
	} else {
		summaryCount++
		started := time.Now()
		next, summarizeErr := summarizer.Summarize(ctx, SummaryChunk, cloneDocument(current))
		summaryDuration += time.Since(started)
		if summarizeErr != nil {
			return progress(SummaryChunk, CompressionNone, summaryCount), fmt.Errorf("%s summary failed: %w", SummaryChunk, summarizeErr)
		}
		if preserveErr := ValidatePreserved(original, next); preserveErr != nil {
			return progress(SummaryChunk, CompressionNone, summaryCount), fmt.Errorf("%s summary is incomplete: %w", SummaryChunk, preserveErr)
		}
		current = cloneDocument(next)
		prepared, err = renderCurrent(SummaryChunk, CompressionNone, summaryCount, config)
		if err == nil {
			return prepared, nil
		}
		if !errors.Is(err, ErrTooLarge) {
			return prepared, err
		}
	}

	started := time.Now()
	makeCandidate := func(density int) (Document, []byte, Budget, error) {
		if err := ctx.Err(); err != nil {
			return Document{}, nil, Budget{}, err
		}
		candidate := reduceEvidence(current, density)
		if err := ValidatePreserved(original, candidate); err != nil {
			return Document{}, nil, Budget{}, err
		}
		prompt, err := render(cloneDocument(candidate))
		if err != nil {
			return Document{}, nil, Budget{}, fmt.Errorf("cannot render planning input: %w", err)
		}
		budget, err := SelectContext(prompt, len(original.Files), config)
		return candidate, prompt, budget, err
	}

	canonical, canonicalPrompt, canonicalBudget, err := makeCandidate(1000)
	if err == nil {
		return evidencePrepared(canonical, canonicalPrompt, canonicalBudget, SummaryChunk, compressionProfile, summaryCount, summaryDuration, time.Since(started), originalPromptBytes), nil
	}
	if !errors.Is(err, ErrTooLarge) {
		return progress(SummaryChunk, compressionProfile, summaryCount), fmt.Errorf("evidence canonicalization failed: %w", err)
	}
	minimum, minimumPrompt, minimumBudget, err := makeCandidate(0)
	if err != nil {
		result := progress(SummaryChunk, compressionProfile, summaryCount)
		result.EvidenceReductionDuration = time.Since(started)
		if errors.Is(err, ErrTooLarge) {
			return result, ErrTooLarge
		}
		return result, fmt.Errorf("evidence reduction failed: %w", err)
	}

	bestDocument, bestPrompt, bestBudget := minimum, minimumPrompt, minimumBudget
	low, high := 0, 999
	for low <= high {
		if err := ctx.Err(); err != nil {
			return progress(SummaryChunk, compressionProfile, summaryCount), err
		}
		mid := low + (high-low)/2
		candidate, prompt, budget, candidateErr := makeCandidate(mid)
		if candidateErr == nil {
			bestDocument, bestPrompt, bestBudget = candidate, prompt, budget
			low = mid + 1
			continue
		}
		if !errors.Is(candidateErr, ErrTooLarge) {
			return progress(SummaryChunk, compressionProfile, summaryCount), fmt.Errorf("evidence reduction failed: %w", candidateErr)
		}
		high = mid - 1
	}
	return evidencePrepared(bestDocument, bestPrompt, bestBudget, SummaryChunk, compressionProfile, summaryCount, summaryDuration, time.Since(started), originalPromptBytes), nil
}

func normalBudgetConfig(config BudgetConfig) BudgetConfig {
	if !supportsExpandedBudget(config) {
		return config
	}
	config.Context = "auto"
	config.MaxContextTokens = Context32K
	return config
}

func supportsExpandedBudget(config BudgetConfig) bool {
	return config.Context == "64k" || config.Context == "auto" && config.MaxContextTokens == Context64K
}

func evidencePrepared(document Document, prompt []byte, budget Budget, stage SummaryStage, profile CompressionProfile, summaryCount int, summaryDuration, reductionDuration time.Duration, originalPromptBytes int) Prepared {
	prepared := Prepared{
		Document: cloneDocument(document), Prompt: append([]byte(nil), prompt...), Budget: budget,
		SummaryStage: stage, CompressionProfile: profile, SummaryCount: summaryCount, SummaryDuration: summaryDuration,
		OriginalPromptBytes: originalPromptBytes, EvidenceReductionCount: 1, EvidenceReductionDuration: reductionDuration,
		RelationContextOmitted: document.RelationContextStatus != nil && document.RelationContextStatus.Omitted,
	}
	for _, file := range document.Files {
		if file.EvidenceReduction != nil {
			prepared.EvidenceBeforeBytes += file.EvidenceReduction.OriginalBytes
			prepared.EvidenceAfterBytes += file.EvidenceReduction.RetainedBytes
		}
	}
	return prepared
}

func cloneDocument(document Document) Document {
	clone := document
	if document.RelationContextStatus != nil {
		status := *document.RelationContextStatus
		status.ObservationsByOutcome = cloneMap(document.RelationContextStatus.ObservationsByOutcome)
		clone.RelationContextStatus = &status
	}
	if document.RelationContext != nil {
		relations := *document.RelationContext
		relations.Components = make([]relation.CandidateComponent, len(document.RelationContext.Components))
		for i, component := range document.RelationContext.Components {
			relations.Components[i] = component
			relations.Components[i].FileIDs = append([]string(nil), component.FileIDs...)
		}
		relations.Edges = append([]relation.Relation(nil), document.RelationContext.Edges...)
		for i := range relations.Edges {
			if relations.Edges[i].Score != nil {
				score := *relations.Edges[i].Score
				relations.Edges[i].Score = &score
			}
		}
		relations.Hints = make([]relation.Hint, len(document.RelationContext.Hints))
		for i, hint := range document.RelationContext.Hints {
			relations.Hints[i] = hint
			relations.Hints[i].FileIDs = append([]string(nil), hint.FileIDs...)
		}
		relations.ReductionReasons = append([]relation.ReductionReason(nil), document.RelationContext.ReductionReasons...)
		relations.Statistics.EdgesByKind = cloneMap(document.RelationContext.Statistics.EdgesByKind)
		relations.Statistics.ObservationsByKind = cloneMap(document.RelationContext.Statistics.ObservationsByKind)
		relations.Statistics.ObservationsByOutcome = cloneMap(document.RelationContext.Statistics.ObservationsByOutcome)
		clone.RelationContext = &relations
	}
	clone.Files = make([]File, len(document.Files))
	for i, file := range document.Files {
		clone.Files[i] = file
		clone.Files[i].OldPath = cloneString(file.OldPath)
		clone.Files[i].NewPath = cloneString(file.NewPath)
		clone.Files[i].OldMode = cloneString(file.OldMode)
		clone.Files[i].NewMode = cloneString(file.NewMode)
		clone.Files[i].HeadIdentity = cloneString(file.HeadIdentity)
		clone.Files[i].WorktreeIdentity = cloneString(file.WorktreeIdentity)
		clone.Files[i].Evidence = append([]syntax.Evidence(nil), file.Evidence...)
		clone.Files[i].summaryLines = append([]summaryLine(nil), file.summaryLines...)
		if file.EvidenceReduction != nil {
			reduction := *file.EvidenceReduction
			clone.Files[i].EvidenceReduction = &reduction
		}
	}
	return clone
}

func cloneMap[K comparable, V any](value map[K]V) map[K]V {
	if value == nil {
		return nil
	}
	cloned := make(map[K]V, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
