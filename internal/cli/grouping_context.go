package cli

import (
	"context"
	"fmt"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/relation"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
	"os"
	"os/exec"
	"strings"
)

func attachGroupingSyntaxContext(ctx context.Context, root string, document *contextinput.Document, sources []relation.File) error {
	if len(document.Files) != len(sources) {
		return fmt.Errorf("grouping sources do not cover selected files")
	}
	for i := range document.Files {
		if document.Files[i].ID != sources[i].Change.ID {
			return fmt.Errorf("grouping source identity mismatch")
		}
		file := &document.Files[i]
		if file.Mode == syntax.ModeMetadataOnly || file.Binary || file.Opaque || !strings.EqualFold(file.Language, "Go") {
			continue
		}
		source := sources[i]
		// Only selected, already-approved regular text blobs are read. Symlink and
		// gitlink targets never enter the parser. HeadIdentity pins the Before bytes.
		var before []byte
		if source.Change.HeadIdentity != nil && source.Change.OldMode != nil && (*source.Change.OldMode == "100644" || *source.Change.OldMode == "100755") {
			command := exec.CommandContext(ctx, "git", "-C", root, "cat-file", "blob", *source.Change.HeadIdentity)
			command.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
			var err error
			before, err = command.Output()
			if err != nil {
				return fmt.Errorf("cannot read selected grouping source")
			}
		}
		entries, err := syntax.ExtractMinimalContext(ctx, before, source.Content, file.GroupingRawDiff, file.Language)
		if err != nil {
			return fmt.Errorf("cannot extract selected grouping context")
		}
		file.GroupingSyntaxContext = entries
	}
	return nil
}
