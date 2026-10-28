package planning

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type groupingFile struct {
	FileID       string                `json:"file_id"`
	Status       string                `json:"status"`
	RawDiff      string                `json:"raw_diff"`
	OldPath      *string               `json:"old_path,omitempty"`
	NewPath      *string               `json:"new_path,omitempty"`
	Language     string                `json:"language,omitempty"`
	Binary       bool                  `json:"binary,omitempty"`
	Vendor       bool                  `json:"vendor,omitempty"`
	Opaque       bool                  `json:"opaque,omitempty"`
	OldMode      *string               `json:"old_mode,omitempty"`
	NewMode      *string               `json:"new_mode,omitempty"`
	ChangeBlocks []groupingChangeBlock `json:"change_blocks"`
}

// DiffRange indexes the exact transmitted raw_diff, one-based and inclusive.
type DiffRange struct {
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
}
type groupingChangeBlock struct {
	BlockID                     string                  `json:"block_id"`
	DiffRange                   DiffRange               `json:"diff_range"`
	EnclosingDeclarationName    string                  `json:"enclosing_declaration_name,omitempty"`
	AgentHistoryMatchStatus     string                  `json:"agent_history_match_status"`
	AgentHistoryMatchReasons    []string                `json:"agent_history_match_reasons"`
	AgentHistoryCandidates      []AgentHistoryCandidate `json:"agent_history_candidates"`
	AgentHistoryOmissionReasons []string                `json:"agent_history_omission_reasons,omitempty"`
}

func groupingProjection(files []contextinput.File) ([]groupingFile, error) {
	out := make([]groupingFile, 0, len(files))
	for _, file := range files {
		if file.GroupingRawDiff != "" && file.Mode != syntax.ModeMetadataOnly && !file.Binary && !file.Opaque {
			file.RawDiff = file.GroupingRawDiff
		}
		metadata, err := projectFileMetadata(file)
		if err != nil {
			return nil, err
		}
		item := groupingFile{FileID: file.ID, Status: file.Status, OldPath: metadata.OldPath, NewPath: metadata.NewPath, Language: metadata.Language, Binary: file.Binary, Vendor: file.Vendor, Opaque: file.Opaque, OldMode: metadata.OldMode, NewMode: metadata.NewMode, ChangeBlocks: []groupingChangeBlock{}}
		// This schema has no worktree_kind. Retain special modes even when
		// unchanged so a symlink/gitlink cannot appear to be a regular file.
		if file.OldMode != nil && (*file.OldMode == "120000" || *file.OldMode == "160000") {
			item.OldMode = file.OldMode
		}
		if file.NewMode != nil && (*file.NewMode == "120000" || *file.NewMode == "160000") {
			item.NewMode = file.NewMode
		}
		if file.Mode != syntax.ModeMetadataOnly && !file.Binary && !file.Opaque {
			item.RawDiff = file.GroupingRawDiff
			if item.RawDiff == "" {
				item.RawDiff = file.RawDiff
			}
			if item.RawDiff == "" {
				return nil, errors.New("grouping raw diff is missing")
			}
			blocks, err := parseGroupingBlocks(item.RawDiff)
			if err != nil {
				return nil, fmt.Errorf("grouping block parsing failed: %w", err)
			}
			item.ChangeBlocks = blocks
			if len(blocks) == 0 {
				item.RawDiff = ""
			}
		}
		out = append(out, item)
	}
	return out, nil
}

var groupingHunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(?:.*)$`)

// Context lines separate runs of changed lines. EOF markers belong to the
// preceding changed run. Malformed or truncated input is never an empty success.
func parseGroupingBlocks(diff string) ([]groupingChangeBlock, error) {
	out := []groupingChangeBlock{}
	oldLeft, newLeft := 0, 0
	inHunk, sawHunk, metadata := false, false, false
	activeStart, activeEnd := 0, 0
	lastKind := byte(0)
	flush := func() {
		if activeStart > 0 {
			out = append(out, groupingChangeBlock{BlockID: fmt.Sprintf("B%d", len(out)+1), DiffRange: DiffRange{activeStart, activeEnd}, AgentHistoryMatchStatus: "error", AgentHistoryMatchReasons: []string{"history_unavailable"}, AgentHistoryCandidates: []AgentHistoryCandidate{}})
			activeStart, activeEnd = 0, 0
		}
	}
	lines := strings.Split(strings.TrimSuffix(diff, "\n"), "\n")
	for n, line := range lines {
		if inHunk && strings.HasPrefix(line, "diff --git ") {
			if oldLeft != 0 || newLeft != 0 {
				return nil, errors.New("incomplete_hunk")
			}
			flush()
			inHunk = false
		}
		if m := groupingHunkHeader.FindStringSubmatch(line); m != nil {
			if inHunk && (oldLeft != 0 || newLeft != 0) {
				return nil, errors.New("incomplete_hunk")
			}
			flush()
			inHunk, sawHunk = true, true
			lastKind = 0
			oldLeft, newLeft = 1, 1
			var err error
			if m[2] != "" {
				oldLeft, err = strconv.Atoi(m[2])
				if err != nil {
					return nil, errors.New("invalid_hunk_count")
				}
			}
			if m[4] != "" {
				newLeft, err = strconv.Atoi(m[4])
				if err != nil {
					return nil, errors.New("invalid_hunk_count")
				}
			}
			continue
		}
		if !inHunk {
			switch {
			case strings.HasPrefix(line, "old mode "), strings.HasPrefix(line, "new mode "), strings.HasPrefix(line, "new file mode "), strings.HasPrefix(line, "deleted file mode "), strings.HasPrefix(line, "rename from "), strings.HasPrefix(line, "rename to "):
				metadata = true
			case strings.HasPrefix(line, "diff --git "), strings.HasPrefix(line, "index "), strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "similarity index "):
			default:
				return nil, errors.New("unsupported_diff")
			}
			continue
		}
		if line == `\ No newline at end of file` {
			if lastKind == 0 || lastKind == '\\' {
				return nil, errors.New("invalid_eof_marker")
			}
			if activeStart > 0 && lastKind != ' ' {
				activeEnd = n + 1
			}
			lastKind = '\\'
			continue
		}
		if line == "" {
			return nil, errors.New("malformed_line")
		}
		switch line[0] {
		case ' ':
			flush()
			oldLeft--
			newLeft--
		case '-':
			if activeStart == 0 {
				activeStart = n + 1
			}
			activeEnd = n + 1
			oldLeft--
		case '+':
			if activeStart == 0 {
				activeStart = n + 1
			}
			activeEnd = n + 1
			newLeft--
		default:
			return nil, errors.New("malformed_line")
		}
		lastKind = line[0]
		if oldLeft < 0 || newLeft < 0 {
			return nil, errors.New("hunk_count_mismatch")
		}
	}
	if inHunk && (oldLeft != 0 || newLeft != 0) {
		return nil, errors.New("incomplete_hunk")
	}
	if !sawHunk && !metadata {
		return nil, errors.New("missing_hunk")
	}
	flush()
	return out, nil
}
