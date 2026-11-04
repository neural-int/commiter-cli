package historyattribution

import (
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func TestPureRenameReachesPhaseProjection(t *testing.T) {
	for _, mode := range []string{"single", "multiple", "wrong_destination", "missing_body", "path_only"} {
		t.Run(mode, func(t *testing.T) {
			request := pipelineRequest()
			old, path := "old.go", "new.go"
			request.Files[0].Status, request.Files[0].OldPath, request.Files[0].NewPath = "renamed", &old, &path
			request.Files[0].RawDiff = "diff --git a/old.go b/new.go\nsimilarity index 100%\nrename from old.go\nrename to new.go\n"
			h := History{Status: "complete", Operations: []Operation{renameOp("move", "")}, Turns: []TurnContext{{Thread: "S", Turn: "T", Status: "completed", Final: "Moved file"}}}
			switch mode {
			case "multiple":
				h.Operations = append(h.Operations, renameOp("other", ""))
			case "wrong_destination":
				h.Operations[0].Path = "other.go"
			case "missing_body":
				h.Operations[0].Diff = nil
			case "path_only":
				h.Operations[0].Kind = "update"
			}
			got, err := attribute(request, h)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Files) != 1 || len(got.Files[0].Blocks) != 1 || got.Files[0].RawDiff != request.Files[0].RawDiff {
				t.Fatal("pure rename was skipped", got)
			}
			b := got.Files[0].Blocks[0]
			if b.Status == "matched" {
				t.Fatal("metadata alone proved a successful content edit")
			}
			if mode == "single" && (len(got.Contexts) != 1 || len(b.Candidates) != 1) {
				t.Fatal("rename context lost", got)
			}
			if (mode == "wrong_destination" || mode == "path_only" || mode == "missing_body") && len(b.Candidates) != 0 {
				t.Fatal("unsupported rename admitted")
			}
			wire, err := planning.ProjectMetadataFile(request.Files[0])
			if err != nil || len(wire.ChangeBlocks) != 1 || wire.RawDiff != got.Files[0].RawDiff || wire.ChangeBlocks[0].DiffRange != b.DiffRange {
				t.Fatal("phase rename projection differs from attribution", err)
			}
		})
	}
}
