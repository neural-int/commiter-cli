package contextinput

import (
	"errors"

	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

// RawDiffProjection supplies complete approved diffs to Three-phase planning.
// Detailed syntax evidence remains in the caller's document. Host-only minimal
// syntax context is retained for the separate Grouping projection.
func RawDiffProjection(document Document) (Document, error) {
	projected := document
	projected.Files = append([]File(nil), document.Files...)
	for i, file := range document.Files {
		if file.Mode == syntax.ModeMetadataOnly || file.Binary || file.Opaque {
			continue
		}
		diff := file.GroupingRawDiff
		if diff == "" {
			diff = file.RawDiff
		}
		if diff == "" && file.Mode == syntax.ModeStructural {
			return Document{}, errors.New("three-phase raw diff is missing for structural input")
		}
		file.Mode = syntax.ModeRawDiff
		file.RawDiff = diff
		file.Evidence = nil
		file.EvidenceReduction = nil
		projected.Files[i] = file
	}
	return projected, nil
}
