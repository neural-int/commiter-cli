package planning

import (
	"errors"
	"strings"

	"github.com/go-enry/go-enry/v2"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

// fileMetadata is an intermediate host projection, never a phase payload.
type fileMetadata struct {
	ID           string      `json:"id"`
	Status       string      `json:"status"`
	OldPath      *string     `json:"old_path,omitempty"`
	NewPath      *string     `json:"new_path,omitempty"`
	OldMode      *string     `json:"old_mode,omitempty"`
	NewMode      *string     `json:"new_mode,omitempty"`
	WorktreeKind string      `json:"worktree_kind,omitempty"`
	Language     string      `json:"language,omitempty"`
	Binary       bool        `json:"binary,omitempty"`
	Vendor       bool        `json:"vendor,omitempty"`
	Opaque       bool        `json:"opaque,omitempty"`
	Mode         syntax.Mode `json:"mode"`
	RawDiff      string      `json:"raw_diff,omitempty"`
}
type metadataGroup struct {
	ID       string         `json:"group_id"`
	Files    []MetadataFile `json:"files"`
	Type     string         `json:"type,omitempty"`
	Breaking *bool          `json:"breaking,omitempty"`
}
type metadataBreakingEvidence struct {
	ID      string   `json:"id"`
	FileIDs []string `json:"file_ids"`
}

// projectFileMetadata selects safe metadata without mutating host input.
func projectFileMetadata(file contextinput.File) (fileMetadata, error) {
	item := fileMetadata{ID: file.ID, Status: file.Status, Binary: file.Binary, Vendor: file.Vendor, Opaque: file.Opaque, Mode: file.Mode}
	present := func(p *string) bool { return p != nil && *p != "" }
	if present(file.OldPath) {
		item.OldPath = file.OldPath
	}
	if present(file.NewPath) {
		item.NewPath = file.NewPath
	}
	deleted := file.Status == "deleted" || file.Status == "D"
	renamed := file.Status == "renamed" || strings.HasPrefix(file.Status, "R")
	if deleted {
		item.NewPath = nil
	}
	if item.OldPath == nil && item.NewPath == nil {
		return item, errors.New("metadata file requires a path")
	}
	if renamed && (item.OldPath == nil || item.NewPath == nil) {
		return item, errors.New("metadata rename requires both paths")
	}
	if !renamed && item.OldPath != nil && item.NewPath != nil && *item.OldPath == *item.NewPath {
		item.OldPath = nil
	}
	if present(file.OldMode) && present(file.NewMode) {
		if *file.OldMode != *file.NewMode {
			item.OldMode, item.NewMode = file.OldMode, file.NewMode
		}
	} else {
		if present(file.OldMode) && *file.OldMode != "100644" {
			item.OldMode = file.OldMode
		}
		if present(file.NewMode) && *file.NewMode != "100644" {
			item.NewMode = file.NewMode
		}
	}
	if file.WorktreeKind != "file" && file.WorktreeKind != "deleted" {
		item.WorktreeKind = file.WorktreeKind
	}
	path := ""
	if item.NewPath != nil {
		path = *item.NewPath
	} else if item.OldPath != nil {
		path = *item.OldPath
	}
	languages := enry.GetLanguagesByExtension(path, nil, nil)
	if len(languages) != 1 || !strings.EqualFold(languages[0], file.Language) {
		item.Language = file.Language
	}
	if file.Binary || file.Opaque || file.Mode == syntax.ModeMetadataOnly {
		item.Mode = syntax.ModeMetadataOnly
	} else {
		item.Mode = syntax.ModeRawDiff
		item.RawDiff = file.RawDiff
		if item.RawDiff == "" {
			return item, errors.New("metadata raw diff is missing")
		}
	}
	return item, nil
}

// MetadataFile uses the same file/block names and representation as Grouping.
type MetadataFile = groupingFile

// ProjectMetadataFile projects a file with all changed blocks. Attribution is
// supplied separately by the host, never inferred from path-only records.
func ProjectMetadataFile(file contextinput.File) (MetadataFile, error) {
	files, err := groupingProjection([]contextinput.File{file})
	if err != nil {
		return MetadataFile{}, err
	}
	return files[0], nil
}

// metadataHistoryPayload selects only this request's files and the union of
// referenced contexts from the original host result, not a prior phase prompt.
func metadataHistoryPayload(groups []candidateGroup, original []groupingFile, contexts []AgentHistoryContext) (map[string]any, error) {
	byID := map[string]groupingFile{}
	for _, f := range original {
		byID[f.FileID] = f
	}
	projected := make([]metadataGroup, 0, len(groups))
	referenced := map[string]bool{}
	for _, group := range groups {
		item := metadataGroup{ID: group.ID, Files: []MetadataFile{}}
		for _, f := range group.Files {
			source, ok := byID[f.ID]
			if !ok {
				return nil, errors.New("metadata file is absent from original grouping input")
			}
			source = clonePhaseFiles([]groupingFile{source})[0]
			for i := range source.ChangeBlocks {
				source.ChangeBlocks[i].EnclosingDeclarationName = ""
			}
			item.Files = append(item.Files, source)
			for _, b := range source.ChangeBlocks {
				for _, c := range b.AgentHistoryCandidates {
					referenced[c.ContextRef] = true
				}
			}
		}
		projected = append(projected, item)
	}
	selected := []AgentHistoryContext{}
	for _, c := range contexts {
		if referenced[c.ContextRef] {
			selected = append(selected, c)
		}
	}
	return map[string]any{"schema_version": 1, "task": "Generate metadata without changing finalized groups.", "agent_history_contexts": selected, "groups": projected}, nil
}
