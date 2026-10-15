package planning

import (
	"errors"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/relation"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type categoryGroup struct {
	ID           string                       `json:"group_id"`
	Contents     map[string]string            `json:"contents"`
	RelatedTests map[string]string            `json:"related_tests"`
	Metadata     map[string]categoryFileState `json:"file_metadata"`
}

// Only semantic file state is exposed; identities and membership stay in Go.
type categoryFileState struct {
	Status   string  `json:"status"`
	OldPath  *string `json:"old_path,omitempty"`
	OldMode  *string `json:"old_mode,omitempty"`
	NewMode  *string `json:"new_mode,omitempty"`
	Language string  `json:"language"`
	Opaque   bool    `json:"opaque,omitempty"`
	Binary   bool    `json:"binary,omitempty"`
	Vendor   bool    `json:"vendor,omitempty"`
}

func categoryProjection(groups []candidateGroup, relations *contextinput.RelationContext) ([]categoryGroup, error) {
	projected := make([]categoryGroup, 0, len(groups))
	for _, group := range groups {
		item := categoryGroup{ID: group.ID, Contents: map[string]string{}, Metadata: map[string]categoryFileState{}}
		for i, file := range group.Files {
			p := candidatePath(file)
			if p == "" {
				return nil, errors.New("category requires a visible file path")
			}
			if _, exists := item.Metadata[p]; exists {
				return nil, errors.New("category requires unique paths within each group")
			}
			state := categoryFileState{Status: file.Status, OldMode: file.OldMode, NewMode: file.NewMode, Language: file.Language, Opaque: file.Opaque || file.Mode == syntax.ModeMetadataOnly, Binary: file.Binary, Vendor: file.Vendor}
			if file.OldPath != nil && *file.OldPath != p {
				state.OldPath = file.OldPath
			}
			item.Metadata[p] = state
			diff := ""
			if file.Mode != syntax.ModeMetadataOnly && !file.Binary && !file.Opaque {
				diff = file.GroupingRawDiff
				if diff == "" {
					diff = file.RawDiff
				}
				if diff == "" {
					return nil, errors.New("category raw diff is missing for selected text input")
				}
			}
			// A matched source/test path is a syntactic role observation, not
			// reference resolution or proof of purpose. Grouping is already final.
			related := false
			if relations != nil && !candidateTestOnly(group) {
				for _, edge := range relations.Edges {
					if edge.Kind == relation.SourceTest && edge.SourceID == group.FileIDs[i] && edge.TargetID != edge.SourceID && containsString(group.FileIDs, edge.TargetID) {
						related = true
					}
				}
			}
			if related && diff != "" {
				if item.RelatedTests == nil {
					item.RelatedTests = map[string]string{}
				}
				item.RelatedTests[p] = diff
			} else {
				item.Contents[p] = diff
			}
		}
		projected = append(projected, item)
	}
	return projected, nil
}
