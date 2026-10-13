package planning

import (
	"errors"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

// groupingFile contains only the LLM projection, never Git identities or AST
// coordinates. Original files still supply Category/Text and validation.
type groupingFile struct {
	ID            string                  `json:"id"`
	Status        string                  `json:"status"`
	OldPath       *string                 `json:"old_path"`
	NewPath       *string                 `json:"new_path"`
	Language      string                  `json:"language"`
	Binary        bool                    `json:"binary"`
	Vendor        bool                    `json:"vendor"`
	Opaque        bool                    `json:"opaque"`
	Mode          syntax.Mode             `json:"mode"`
	RawDiff       string                  `json:"raw_diff,omitempty"`
	SyntaxContext []syntax.MinimalContext `json:"syntax_context,omitempty"`
}

func groupingProjection(files []contextinput.File) ([]groupingFile, error) {
	projected := make([]groupingFile, 0, len(files))
	for _, file := range files {
		item := groupingFile{ID: file.ID, Status: file.Status, OldPath: file.OldPath, NewPath: file.NewPath, Language: file.Language, Binary: file.Binary, Vendor: file.Vendor, Opaque: file.Opaque, Mode: syntax.ModeMetadataOnly}
		if file.Mode != syntax.ModeMetadataOnly && !file.Binary && !file.Opaque {
			item.Mode = syntax.ModeRawDiff
			item.RawDiff = file.GroupingRawDiff
			if item.RawDiff == "" {
				item.RawDiff = file.RawDiff
			}
			if file.Mode == syntax.ModeStructural && item.RawDiff == "" {
				return nil, errors.New("grouping raw diff is missing for structural input")
			}
			item.SyntaxContext = append([]syntax.MinimalContext(nil), file.GroupingSyntaxContext...)
		}
		projected = append(projected, item)
	}
	return projected, nil
}
