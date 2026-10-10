package gitstate

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const FileFirstMaxFiles = 16
const fileFirstMaxBytes = 1 << 20
const fileFirstMaxLines = 20000

// FileFirstGroup owns the complete HEAD-to-worktree change of one selected
// file, including both paths of a rename. It makes no claim about intent.
type FileFirstGroup struct {
	FileID string
	Paths  []string
}

// FileFirstPreview is a structural receipt, not a commit plan or authorization
// to mutate Git. Trees refer to the isolated, already-discarded object store.
type FileFirstPreview struct {
	Groups     []FileFirstGroup
	BaseTree   string
	FinalTree  string
	StageTrees []string
}

type fileFirstEntry struct {
	path, mode, object string
}

type fileFirstBlock struct {
	group         FileFirstGroup
	before, after *fileFirstEntry
}

// PreviewFileFirst connects the existing immutable collection contract to
// file-singleton ownership and binary Git replay. The collection options must
// be the same selection/approval options used to obtain snapshot. It never
// writes the real index, worktree, refs, or object database, invokes clean
// filters, or runs repository code. Unsupported selected changes stop the
// whole preview. Partial staging follows the existing file-level contract:
// a selected file owns all of its HEAD-to-working-tree change.
func PreviewFileFirst(snapshot Snapshot, options Options) (preview FileFirstPreview, err error) {
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return FileFirstPreview{}, err
	}
	if snapshot.Root == "" || len(snapshot.Changes) < 1 || len(snapshot.Changes) > FileFirstMaxFiles {
		return FileFirstPreview{}, safety("file-first preview requires one to sixteen selected files")
	}
	// Custom readers are a collection test seam, not a source of replay bytes.
	options.Files = fileFirstFiles{}
	for _, change := range snapshot.Changes {
		if change.Size > fileFirstMaxBytes {
			return FileFirstPreview{}, safety("file-first selected byte budget exceeded")
		}
	}
	commonDir, err := gitPath(ctx, snapshot.Root, "--git-common-dir")
	if err != nil {
		return FileFirstPreview{}, internal("cannot resolve common Git directory")
	}
	lock, err := acquireLock(commonDir)
	if err != nil {
		return FileFirstPreview{}, err
	}
	defer func() {
		if closeErr := lock.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			preview = FileFirstPreview{}
		}
	}()
	// Even git status may execute a clean filter or fsmonitor command. Reject
	// such configuration before recollection, including unselected paths.
	check := func() error {
		if e := fileFirstCheckExternalGitCommands(ctx, snapshot.Root); e != nil {
			return e
		}
		state, e := inspect(ctx, snapshot.Root)
		if e != nil {
			return e
		}
		if state.head != snapshot.Head || state.branch != snapshot.Branch {
			return safety("file-first repository snapshot changed")
		}
		current, e := collectLocked(snapshot.Root, options, ctx)
		if e != nil {
			return e
		}
		if !sameFileFirstSnapshot(snapshot, current) {
			return safety("file-first snapshot changed")
		}
		return nil
	}
	if err = check(); err != nil {
		return FileFirstPreview{}, err
	}
	indexPath, err := gitPath(ctx, snapshot.Root, "--git-path", "index")
	if err != nil {
		return FileFirstPreview{}, err
	}
	indexBefore, err := os.ReadFile(indexPath)
	if err != nil {
		return FileFirstPreview{}, internal("cannot preserve original Git index")
	}

	// Objects produced by hash-object/write-tree belong only to this directory.
	// Existing repository objects are read through the alternate object store.
	tmp, err := os.MkdirTemp("", "commiter-file-first-")
	if err != nil {
		return FileFirstPreview{}, err
	}
	defer os.RemoveAll(tmp)
	objects := filepath.Join(tmp, "objects")
	if err = os.Mkdir(objects, 0o700); err != nil {
		return FileFirstPreview{}, err
	}
	repositoryObjects, err := gitPath(ctx, snapshot.Root, "--git-path", "objects")
	if err != nil {
		return FileFirstPreview{}, err
	}
	env := fileFirstEnvironment(filepath.Join(tmp, "index"), objects, repositoryObjects)
	run := func(input []byte, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", snapshot.Root, "-c", "core.splitIndex=false", "-c", "core.hooksPath=" + filepath.Join(tmp, "hooks")}, args...)...)
		cmd.Env, cmd.Stdin = env, bytes.NewReader(input)
		out, e := cmd.Output()
		if e != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, safety("file-first temporary Git operation failed: " + args[0])
		}
		return out, nil
	}
	text := func(args ...string) (string, error) {
		out, e := run(nil, args...)
		return strings.TrimSpace(string(out)), e
	}
	preview.BaseTree, err = text("rev-parse", snapshot.Head+"^{tree}")
	if err != nil {
		return FileFirstPreview{}, err
	}

	changes := append([]Change(nil), snapshot.Changes...)
	sort.Slice(changes, func(i, j int) bool { return changes[i].ID < changes[j].ID })
	blocks := make([]fileFirstBlock, 0, len(changes))
	owned := map[string]string{}
	for _, change := range changes {
		block, e := fileFirstReadBlock(ctx, snapshot.Root, change, run)
		if e != nil {
			return FileFirstPreview{}, e
		}
		for _, path := range block.group.Paths {
			for prior, owner := range owned {
				if owner != change.ID && (prior == path || strings.HasPrefix(path, prior+"/") || strings.HasPrefix(prior, path+"/")) {
					return FileFirstPreview{}, safety("file-first ownership contains overlapping paths")
				}
			}
			owned[path] = change.ID
		}
		blocks = append(blocks, block)
		preview.Groups = append(preview.Groups, block.group)
	}
	flags, err := gitBytes(ctx, snapshot.Root, "ls-files", "-v", "-z")
	if err != nil {
		return FileFirstPreview{}, err
	}
	for _, flag := range splitNUL(flags) {
		if len(flag) < 3 {
			return FileFirstPreview{}, safety("file-first index flags are unsupported")
		}
		if owned[flag[2:]] != "" && flag[0] != 'H' {
			return FileFirstPreview{}, safety("file-first selected index flags are unsupported")
		}
	}
	install := func(block fileFirstBlock, forward bool) error {
		for _, path := range block.group.Paths {
			if _, e := run(nil, "update-index", "--force-remove", "--", path); e != nil {
				return e
			}
		}
		entry := block.before
		if forward {
			entry = block.after
		}
		if entry != nil {
			_, e := run(nil, "update-index", "--add", "--cacheinfo", entry.mode, entry.object, entry.path)
			return e
		}
		return nil
	}
	// Independent full target. Replay must match this tree, not a tree computed
	// from the replay's own accumulated patches.
	if _, err = run(nil, "read-tree", preview.BaseTree); err != nil {
		return FileFirstPreview{}, err
	}
	for _, block := range blocks {
		if err = install(block, true); err != nil {
			return FileFirstPreview{}, err
		}
	}
	preview.FinalTree, err = text("write-tree")
	if err != nil {
		return FileFirstPreview{}, err
	}
	for _, reverseOrder := range []bool{false, true} {
		if _, err = run(nil, "read-tree", preview.BaseTree); err != nil {
			return FileFirstPreview{}, err
		}
		for step := range blocks {
			i := step
			if reverseOrder {
				i = len(blocks) - 1 - step
			}
			block := blocks[i]
			beforeTree, e := text("write-tree")
			if e != nil {
				return FileFirstPreview{}, e
			}
			if e = install(block, true); e != nil {
				return FileFirstPreview{}, e
			}
			afterTree, e := text("write-tree")
			if e != nil {
				return FileFirstPreview{}, e
			}
			if beforeTree == afterTree {
				return FileFirstPreview{}, safety("file-first selected change has no tree delta")
			}
			patch, e := run(nil, "diff-tree", "--no-ext-diff", "--no-textconv", "--no-renames", "--binary", "-p", beforeTree, afterTree)
			if e != nil {
				return FileFirstPreview{}, e
			}
			// Revert then replay each complete file block using Git's patch engine.
			for _, reverse := range []bool{true, false} {
				args := []string{"apply", "--cached", "--binary", "--whitespace=nowarn"}
				expected := afterTree
				if reverse {
					args = append(args, "--reverse")
					expected = beforeTree
				}
				if _, e = run(patch, args...); e != nil {
					return FileFirstPreview{}, e
				}
				actual, e := text("write-tree")
				if e != nil {
					return FileFirstPreview{}, e
				}
				if actual != expected {
					return FileFirstPreview{}, safety("file-first intermediate replay tree mismatch")
				}
			}
			if !reverseOrder {
				preview.StageTrees = append(preview.StageTrees, afterTree)
			}
		}
		final, e := text("write-tree")
		if e != nil {
			return FileFirstPreview{}, e
		}
		if final != preview.FinalTree {
			return FileFirstPreview{}, safety("file-first final replay tree mismatch")
		}
		// Undo the full sequence and require the original tree as well.
		for step := range blocks {
			i := len(blocks) - 1 - step
			if reverseOrder {
				i = step
			}
			if e = install(blocks[i], false); e != nil {
				return FileFirstPreview{}, e
			}
		}
		base, e := text("write-tree")
		if e != nil {
			return FileFirstPreview{}, e
		}
		if base != preview.BaseTree {
			return FileFirstPreview{}, safety("file-first inverse reconstruction mismatch")
		}
	}
	if err = check(); err != nil {
		return FileFirstPreview{}, err
	}
	indexAfter, err := os.ReadFile(indexPath)
	if err != nil || !bytes.Equal(indexBefore, indexAfter) {
		return FileFirstPreview{}, safety("file-first original index changed")
	}
	return preview, nil
}

func sameFileFirstSnapshot(before, after Snapshot) bool {
	if before.Root != after.Root || before.Head != after.Head || before.Branch != after.Branch || before.IndexIdentity != after.IndexIdentity ||
		!reflect.DeepEqual(before.Excluded, after.Excluded) || !reflect.DeepEqual(before.Untracked, after.Untracked) || len(before.Changes) != len(after.Changes) {
		return false
	}
	wanted := map[string]Change{}
	for _, change := range before.Changes {
		if change.ID == "" || wanted[change.ID].ID != "" {
			return false
		}
		wanted[change.ID] = change
	}
	for _, change := range after.Changes {
		if !reflect.DeepEqual(wanted[change.ID], change) {
			return false
		}
	}
	return true
}

func fileFirstReadBlock(ctx context.Context, root string, change Change, run func([]byte, ...string) ([]byte, error)) (fileFirstBlock, error) {
	block := fileFirstBlock{group: FileFirstGroup{FileID: change.ID}}
	for _, path := range []*string{change.OldPath, change.NewPath} {
		if path == nil {
			continue
		}
		if _, err := normalizeRepoPath(*path); err != nil {
			return block, err
		}
		if len(block.group.Paths) == 0 || block.group.Paths[0] != *path {
			block.group.Paths = append(block.group.Paths, *path)
		}
	}
	if len(block.group.Paths) == 0 {
		return block, safety("file-first change has no path")
	}
	if change.Status != "added" && change.Status != "modified" && change.Status != "deleted" && change.Status != "renamed" {
		return block, safety("file-first selected status is unsupported")
	}
	if change.OldPath != nil {
		if change.OldMode == nil || !fileFirstRegularMode(*change.OldMode) || change.HeadIdentity == nil {
			return block, safety("file-first supports only regular-file blocks")
		}
		block.before = &fileFirstEntry{*change.OldPath, *change.OldMode, *change.HeadIdentity}
		size, err := gitText(ctx, root, "cat-file", "-s", *change.HeadIdentity)
		n, parseErr := strconv.ParseInt(size, 10, 64)
		if err != nil || parseErr != nil || n < 0 || n > fileFirstMaxBytes {
			return block, safety("file-first before byte budget exceeded")
		}
		content, err := gitBytes(ctx, root, "cat-file", "blob", *change.HeadIdentity)
		if err != nil {
			return block, internal("cannot read file-first source blob")
		}
		if !fileFirstWithinLineBudget(content) {
			return block, safety("file-first before line budget exceeded")
		}
	}
	if change.NewPath != nil {
		if change.NewMode == nil || !fileFirstRegularMode(*change.NewMode) || change.WorktreeKind != "file" || change.WorktreeID == nil {
			return block, safety("file-first supports only regular-file blocks")
		}
		// Refuse normalization/filter contracts rather than claiming raw-byte
		// equality for a future git add that could rewrite the selected content.
		attrs, err := gitBytes(ctx, root, "check-attr", "-z", "filter", "text", "eol", "working-tree-encoding", "ident", "--", *change.NewPath)
		if err != nil {
			return block, internal("cannot inspect file-first Git attributes")
		}
		fields := splitNUL(attrs)
		if len(fields) != 15 {
			return block, safety("file-first Git attributes are unsupported")
		}
		for i := 2; i < len(fields); i += 3 {
			if fields[i] != "unspecified" && fields[i] != "unset" {
				return block, safety("file-first Git content conversion is unsupported")
			}
		}
		autocrlf, err := gitText(ctx, root, "config", "--get", "core.autocrlf")
		if err == nil && autocrlf != "false" {
			return block, safety("file-first core.autocrlf is unsupported")
		}
		filemode, err := gitText(ctx, root, "config", "--get", "core.filemode")
		if err == nil && filemode == "false" {
			return block, safety("file-first core.filemode=false is unsupported")
		}
		absolute := filepath.Join(root, filepath.FromSlash(*change.NewPath))
		info, err := os.Lstat(absolute)
		if err != nil || !info.Mode().IsRegular() || info.Size() > fileFirstMaxBytes {
			return block, safety("file-first after byte budget or mode is unsupported")
		}
		content, err := (fileFirstFiles{}).ReadFile(absolute)
		if err != nil {
			return block, internal("cannot read file-first target bytes")
		}
		if len(content) > fileFirstMaxBytes || !fileFirstWithinLineBudget(content) {
			return block, safety("file-first after content budget exceeded")
		}
		if sha256Hex(content) != *change.WorktreeID {
			return block, safety("file-first target snapshot changed")
		}
		object, err := run(content, "hash-object", "--no-filters", "-w", "--stdin")
		if err != nil {
			return block, err
		}
		objectID := strings.TrimSpace(string(object))
		// Hashing and reading back proves binary/no-final-newline preservation.
		restored, err := run(nil, "cat-file", "blob", objectID)
		if err != nil || !bytes.Equal(restored, content) {
			return block, safety("file-first byte reconstruction mismatch")
		}
		block.after = &fileFirstEntry{*change.NewPath, *change.NewMode, objectID}
	}
	return block, nil
}

func fileFirstRegularMode(mode string) bool { return mode == "100644" || mode == "100755" }

func fileFirstCheckExternalGitCommands(ctx context.Context, root string) error {
	for _, pattern := range []string{`^filter\..*\.(clean|process)$`, `^core\.fsmonitor$`} {
		keys, err := gitBytes(ctx, root, "config", "--name-only", "--get-regexp", pattern)
		if err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) && exit.ExitCode() == 1 {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return internal("cannot inspect file-first Git command configuration")
		}
		if len(keys) > 0 {
			return safety("file-first external Git filters or fsmonitor are unsupported")
		}
	}
	// Status may recurse into submodules with their own filters/monitors.
	// The first version does not support gitlinks, including unselected ones.
	entries, err := gitBytes(ctx, root, "ls-files", "--stage", "-z")
	if err != nil {
		return internal("cannot inspect file-first index modes")
	}
	for _, entry := range splitNUL(entries) {
		if strings.HasPrefix(entry, "160000 ") {
			return safety("file-first repositories with gitlinks are unsupported")
		}
	}
	return nil
}

func fileFirstWithinLineBudget(content []byte) bool {
	lines := bytes.Count(content, []byte{'\n'})
	if len(content) > 0 && content[len(content)-1] != '\n' {
		lines++
	}
	return lines <= fileFirstMaxLines
}

type fileFirstFiles struct{ osFiles }

func (fileFirstFiles) ReadFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > fileFirstMaxBytes {
		return nil, safety("file-first selected byte budget or mode is unsupported")
	}
	content, err := io.ReadAll(io.LimitReader(file, fileFirstMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > fileFirstMaxBytes {
		return nil, safety("file-first selected byte budget exceeded")
	}
	return content, nil
}

func fileFirstEnvironment(index, objects, alternate string) []string {
	env := []string{}
	for _, entry := range os.Environ() {
		name := strings.SplitN(entry, "=", 2)[0]
		switch name {
		case "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_OPTIONAL_LOCKS", "GIT_LITERAL_PATHSPECS", "LC_ALL":
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GIT_INDEX_FILE="+index, "GIT_OBJECT_DIRECTORY="+objects,
		"GIT_ALTERNATE_OBJECT_DIRECTORIES="+strconv.Quote(alternate), "GIT_OPTIONAL_LOCKS=0", "GIT_LITERAL_PATHSPECS=1", "LC_ALL=C")
}
