package contextinput

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type Prepared struct {
	Document                  Document      `json:"document"`
	Prompt                    []byte        `json:"-"`
	Budget                    Budget        `json:"budget"`
	SummaryStage              SummaryStage  `json:"summary_stage"`
	SummaryCount              int           `json:"summary_count"`
	SummaryDuration           time.Duration `json:"-"`
	OriginalPromptBytes       int           `json:"original_prompt_bytes"`
	EvidenceReductionCount    int           `json:"evidence_reduction_count"`
	EvidenceReductionDuration time.Duration `json:"-"`
	EvidenceBeforeBytes       int           `json:"evidence_before_bytes"`
	EvidenceAfterBytes        int           `json:"evidence_after_bytes"`
}

// Prepare renders and measures the exact final prompt. Oversized input is
// summarized in the specified file -> hunk -> chunk order. Git identities are
// exact throughout; any later evidence reduction must satisfy its audited
// provenance and structural-coverage contract.
func Prepare(ctx context.Context, document Document, config BudgetConfig, render Renderer, summarizer Summarizer) (Prepared, error) {
	if render == nil {
		return Prepared{}, errors.New("prompt renderer is required")
	}
	original := cloneDocument(document)
	current := cloneDocument(document)
	var summaryDuration time.Duration
	var originalPromptBytes int
	if summarizer == nil {
		standard := NewHierarchicalSummarizer()
		summarizer = standard
	}
	progress := func(stage SummaryStage, count int) Prepared {
		return Prepared{SummaryStage: stage, SummaryCount: count, SummaryDuration: summaryDuration, OriginalPromptBytes: originalPromptBytes}
	}
	for attempt, stage := range []SummaryStage{SummaryNone, SummaryFile, SummaryHunk, SummaryChunk} {
		if stage != SummaryNone {
			started := time.Now()
			next, err := summarizer.Summarize(ctx, stage, cloneDocument(current))
			summaryDuration += time.Since(started)
			if err != nil {
				return progress(stage, attempt), fmt.Errorf("%s summary failed: %w", stage, err)
			}
			if err := ValidatePreserved(original, next); err != nil {
				return progress(stage, attempt), fmt.Errorf("%s summary is incomplete: %w", stage, err)
			}
			current = cloneDocument(next)
		}
		prompt, err := render(cloneDocument(current))
		if err != nil {
			return progress(stage, attempt), fmt.Errorf("cannot render planning input: %w", err)
		}
		if originalPromptBytes == 0 {
			originalPromptBytes = len(prompt) + config.PromptOverheadBytes
		}
		budget, err := SelectContext(prompt, len(original.Files), config)
		if err == nil {
			return Prepared{
				Document: cloneDocument(current), Prompt: append([]byte(nil), prompt...), Budget: budget,
				SummaryStage: stage, SummaryCount: attempt,
				SummaryDuration: summaryDuration, OriginalPromptBytes: originalPromptBytes,
			}, nil
		}
		if !errors.Is(err, ErrTooLarge) {
			return progress(stage, attempt), err
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
		return evidencePrepared(canonical, canonicalPrompt, canonicalBudget, SummaryChunk, 3, summaryDuration, time.Since(started), originalPromptBytes), nil
	}
	if !errors.Is(err, ErrTooLarge) {
		return progress(SummaryChunk, 3), fmt.Errorf("evidence canonicalization failed: %w", err)
	}
	minimum, minimumPrompt, minimumBudget, err := makeCandidate(0)
	if err != nil {
		result := progress(SummaryChunk, 3)
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
			return progress(SummaryChunk, 3), err
		}
		mid := low + (high-low)/2
		candidate, prompt, budget, candidateErr := makeCandidate(mid)
		if candidateErr == nil {
			bestDocument, bestPrompt, bestBudget = candidate, prompt, budget
			low = mid + 1
			continue
		}
		if !errors.Is(candidateErr, ErrTooLarge) {
			return progress(SummaryChunk, 3), fmt.Errorf("evidence reduction failed: %w", candidateErr)
		}
		high = mid - 1
	}
	return evidencePrepared(bestDocument, bestPrompt, bestBudget, SummaryChunk, 3, summaryDuration, time.Since(started), originalPromptBytes), nil
}

func evidencePrepared(document Document, prompt []byte, budget Budget, stage SummaryStage, summaryCount int, summaryDuration, reductionDuration time.Duration, originalPromptBytes int) Prepared {
	prepared := Prepared{
		Document: cloneDocument(document), Prompt: append([]byte(nil), prompt...), Budget: budget,
		SummaryStage: stage, SummaryCount: summaryCount, SummaryDuration: summaryDuration,
		OriginalPromptBytes: originalPromptBytes, EvidenceReductionCount: 1, EvidenceReductionDuration: reductionDuration,
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
		if file.EvidenceReduction != nil {
			reduction := *file.EvidenceReduction
			clone.Files[i].EvidenceReduction = &reduction
		}
	}
	return clone
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
