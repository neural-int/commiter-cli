package cli

import (
	"encoding/json"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

// candidateSourceProjection chooses an existing input representation per file.
// It never truncates the approved diff. Syntax results and relation extraction
// remain host-side; the projection changes only the Three-phase model input.
func candidateSourceProjection(document contextinput.Document) (contextinput.Document, error) {
	projected := document
	projected.Files = append([]contextinput.File(nil), document.Files...)
	for i, file := range document.Files {
		if file.Mode != syntax.ModeStructural || file.Binary || file.Opaque || file.GroupingRawDiff == "" {
			continue
		}
		raw := file
		raw.Mode = syntax.ModeRawDiff
		raw.RawDiff = file.GroupingRawDiff
		raw.Evidence = nil
		raw.EvidenceReduction = nil
		structuralJSON, err := json.Marshal(file)
		if err != nil {
			return contextinput.Document{}, err
		}
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			return contextinput.Document{}, err
		}
		if len(rawJSON) < len(structuralJSON) {
			projected.Files[i] = raw
		}
	}
	return projected, nil
}
