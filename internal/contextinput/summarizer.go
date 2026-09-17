package contextinput

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

const (
	defaultChunkLines   = canonicalChunkLines
	defaultExcerptLines = 7
	defaultExcerptBytes = 224
)

// HierarchicalSummarizer performs deterministic, local-only diff compression.
// It removes redundant file headers, then unchanged hunk context, and finally
// replaces fixed-size chunks with counts, a digest, and bounded changed-line
// excerpts. Git identities and structural evidence remain outside summaries.
type HierarchicalSummarizer struct {
	ChunkLines   int
	ExcerptLines int
	ExcerptBytes int
}

func NewHierarchicalSummarizer() HierarchicalSummarizer {
	return HierarchicalSummarizer{
		ChunkLines: defaultChunkLines, ExcerptLines: defaultExcerptLines, ExcerptBytes: defaultExcerptBytes,
	}
}

func (s HierarchicalSummarizer) Summarize(ctx context.Context, stage SummaryStage, document Document) (Document, error) {
	if err := s.validate(); err != nil {
		return Document{}, err
	}
	return s.summarize(ctx, stage, document, compressionLimits{
		chunkLines: s.ChunkLines, excerptLines: s.ExcerptLines, excerptBytes: s.ExcerptBytes,
	})
}

func (s HierarchicalSummarizer) SummarizeProfile(ctx context.Context, profile CompressionProfile, document Document) (Document, error) {
	limits, ok := compressionLimitsFor(profile)
	if !ok {
		return Document{}, fmt.Errorf("unsupported compression profile %q", profile)
	}
	return s.summarize(ctx, SummaryChunk, document, limits)
}

func (s HierarchicalSummarizer) summarize(ctx context.Context, stage SummaryStage, document Document, limits compressionLimits) (Document, error) {
	for i := range document.Files {
		if err := ctx.Err(); err != nil {
			return Document{}, err
		}
		file := &document.Files[i]
		if file.Mode != syntax.ModeRawDiff {
			continue
		}
		source := file.RawDiff
		if source == "" {
			source = file.Summary
		}
		if source == "" {
			return Document{}, fmt.Errorf("file %s has no raw diff to summarize", file.ID)
		}
		lines := file.summaryLines
		if len(lines) == 0 {
			lines = indexSummaryLines(source)
		}
		var summary string
		switch stage {
		case SummaryFile:
			lines = summarizeFileLines(lines)
			summary = joinSummary(splitLines(summaryLinesText(lines)), source)
		case SummaryHunk:
			lines = summarizeHunkLines(lines)
			summary = joinSummary(splitLines(summaryLinesText(lines)), source)
		case SummaryChunk:
			compressed := compressSummaryLines(lines, limits)
			summary = joinSummary(splitLines(compressed.summary), source)
			lines = selectedSummaryLines(lines, compressed.selected, limits.chunkLines)
		default:
			return Document{}, fmt.Errorf("unsupported summary stage %q", stage)
		}
		file.RawDiff = ""
		file.Summary = summary
		file.summaryLines = append([]summaryLine(nil), lines...)
	}
	return document, nil
}

func (s HierarchicalSummarizer) validate() error {
	if s.ChunkLines <= 0 || s.ExcerptLines <= 0 || s.ExcerptBytes <= 0 {
		return errors.New("summary limits must be positive")
	}
	return nil
}

func summarizeFile(source string) string {
	return joinSummary(splitLines(summaryLinesText(summarizeFileLines(indexSummaryLines(source)))), source)
}

func summarizeHunks(source string) string {
	return joinSummary(splitLines(summaryLinesText(summarizeHunkLines(indexSummaryLines(source)))), source)
}

func (s HierarchicalSummarizer) summarizeChunks(source string) string {
	summary := compressSummaryLines(indexSummaryLines(source), compressionLimits{
		chunkLines: s.ChunkLines, excerptLines: s.ExcerptLines, excerptBytes: s.ExcerptBytes,
	}).summary
	return joinSummary(splitLines(summary), source)
}

func redundantFileHeader(line string) bool {
	return strings.HasPrefix(line, "diff --git ") || strings.HasPrefix(line, "index ") ||
		strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ")
}

func splitLines(source string) []string {
	return strings.Split(strings.TrimSuffix(source, "\n"), "\n")
}

func joinSummary(lines []string, fallback string) string {
	summary := strings.Join(lines, "\n")
	if summary == "" {
		digest := sha256.Sum256([]byte(fallback))
		return "diff metadata only sha256=" + hex.EncodeToString(digest[:])
	}
	return summary
}
