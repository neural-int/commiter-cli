package historyattribution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/planning"
)

const receiveLimit = int64(4 * 1024 * 1024)

var inlineSecret = regexp.MustCompile(`(?i)(api[_-]?key|access[_-]?token|password|passwd|client[_-]?secret|authorization)\s*[=:]\s*\S+`)

func safeText(s string) bool {
	b := []byte(s)
	return !strings.ContainsRune(s, 0) && !planning.ExtractSensitiveValues(b).Contains(b) && !inlineSecret.MatchString(s)
}
func safePath(s string) bool {
	if s == "" || filepath.IsAbs(s) || strings.Contains(s, "\\") || strings.Contains(s, "\x00") || filepath.ToSlash(filepath.Clean(s)) != s || s == "." || strings.HasPrefix(s, "../") {
		return false
	}
	for _, part := range strings.Split(strings.ToLower(s), "/") {
		if part == ".git" || strings.HasPrefix(part, ".env") || strings.Contains(part, "key") || strings.Contains(part, "credential") || strings.Contains(part, "secret") || strings.HasSuffix(part, ".pem") {
			return false
		}
	}
	return true
}
func digest(b []byte) string { x := sha256.Sum256(b); return hex.EncodeToString(x[:]) }

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		b.exceeded = true
		return 0, fmt.Errorf("output_limit")
	}
	return b.Buffer.Write(p)
}
func git(ctx context.Context, root, stage string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root, "--no-optional-locks"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C", "GIT_LITERAL_PATHSPECS=1")
	var out boundedBuffer
	out.limit = int(receiveLimit)
	cmd.Stdout = &out
	if cmd.Run() != nil {
		if ctx.Err() != nil {
			return nil, failure(stage, "deadline_exceeded")
		}
		if out.exceeded {
			return nil, failure(stage, "output_limit")
		}
		return nil, failure(stage, "command_failed")
	}
	return out.Bytes(), nil
}

// Resolve deleted targets through existing parents; reject every symlink component.
func resolveTarget(root, path string) (string, error) {
	if !safePath(path) {
		return "", failure("git.paths", "unsafe_path")
	}
	current := root
	for _, part := range strings.Split(path, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", failure("git.paths", "symlink_not_supported")
			}
		} else if !os.IsNotExist(err) {
			return "", failure("git.paths", "stat_failed")
		}
	}
	return current, nil
}

type gitSnapshot struct {
	head, index, state string
	files              []GitFile
}

func readGit(ctx context.Context, root string, paths []string) (gitSnapshot, error) {
	s := gitSnapshot{}
	head, err := git(ctx, root, "git.head", "rev-parse", "--verify", "HEAD")
	if err != nil {
		return s, err
	}
	s.head = strings.TrimSpace(string(head))
	index, err := git(ctx, root, "git.index", "rev-parse", "--path-format=absolute", "--git-path", "index")
	if err != nil {
		return s, err
	}
	ib, err := readLocalFile(strings.TrimSpace(string(index)))
	if err != nil {
		return s, failure("git.index", "read_failed")
	}
	s.index = digest(ib)
	renamePaths := map[string]string{}
	nameArgs := append([]string{"diff", "HEAD", "--no-ext-diff", "--no-textconv", "--find-renames", "--name-status", "-z", "--"}, paths...)
	names, err := git(ctx, root, "git.status", nameArgs...)
	if err != nil {
		return s, err
	}
	fields := strings.Split(strings.TrimSuffix(string(names), "\x00"), "\x00")
	for i := 0; i < len(fields) && fields[i] != ""; {
		if strings.HasPrefix(fields[i], "R") {
			if i+2 >= len(fields) {
				return s, failure("git.status", "invalid_rename_metadata")
			}
			if !safePath(fields[i+1]) || !safePath(fields[i+2]) {
				return s, failure("git.paths", "unsafe_rename_path")
			}
			renamePaths[fields[i+1]] = fields[i+2]
			renamePaths[fields[i+2]] = fields[i+1]
			i += 3
		} else {
			if i+1 >= len(fields) {
				return s, failure("git.status", "invalid_metadata")
			}
			i += 2
		}
	}
	processedRenames := map[string]bool{}
	for _, path := range paths {
		if processedRenames[path] {
			continue
		}
		full, err := resolveTarget(root, path)
		if err != nil {
			return s, err
		}
		info, statErr := os.Lstat(full)
		if statErr != nil && !os.IsNotExist(statErr) {
			return s, failure("git.worktree", "stat_failed")
		}
		if statErr == nil && (!info.Mode().IsRegular() || info.Size() > receiveLimit) {
			return s, failure("git.worktree", "nonregular_or_too_large")
		}
		kind := "update"
		oldPath := ""
		diffArgs := []string{"diff", "HEAD", "--no-ext-diff", "--no-textconv", "--no-renames", "--unified=3", "--", path}
		if pair, ok := renamePaths[path]; ok {
			// Require both endpoints selected so capture/verification cover both.
			if !containsString(paths, pair) {
				return s, failure("git.paths", "rename_pair_required")
			}
			namePair, err := git(ctx, root, "git.rename", "diff", "HEAD", "--no-ext-diff", "--no-textconv", "--find-renames", "--name-status", "-z", "--", path, pair)
			if err != nil {
				return s, err
			}
			parts := strings.Split(string(namePair), "\x00")
			if len(parts) < 4 || !strings.HasPrefix(parts[0], "R") {
				return s, failure("git.rename", "invalid_rename_metadata")
			}
			oldPath = parts[1]
			path = parts[2]
			kind = "rename"
			for _, endpoint := range []string{oldPath, path} {
				if _, err := resolveTarget(root, endpoint); err != nil {
					return s, err
				}
			}
			processedRenames[oldPath] = true
			processedRenames[path] = true
			diffArgs = []string{"diff", "HEAD", "--no-ext-diff", "--no-textconv", "--find-renames", "--unified=3", "--", oldPath, path}
		}
		diff, err := git(ctx, root, "git.diff", diffArgs...)
		if err != nil {
			return s, err
		}
		if bytes.Contains(diff, []byte("GIT binary patch")) || bytes.Contains(diff, []byte("Binary files ")) {
			return s, failure("git.diff", "unsupported_change_type")
		}

		if len(diff) == 0 {
			untracked, err := git(ctx, root, "git.untracked", "ls-files", "--others", "--exclude-standard", "-z", "--", path)
			if err != nil {
				return s, err
			}
			if string(untracked) != path+"\x00" {
				continue
			}
			diff, err = readLocalFile(full)
			if err != nil {
				return s, failure("git.untracked", "read_failed")
			}
			kind = "untracked"
		}
		if !safeText(string(diff)) {
			return s, failure("git.diff", "sensitive_content")
		}
		if len(diff) > 0 {
			s.files = append(s.files, GitFile{Path: path, Kind: kind, Diff: string(diff), OldPath: oldPath})
		}
	}
	// Include bytes of selected files too: diff output does not express all file modes.
	type fileState struct {
		Path, Hash string
		Mode       uint32
		Missing    bool
	}
	states := []fileState{}
	for _, path := range paths {
		full, _ := resolveTarget(root, path)
		info, err := os.Lstat(full)
		if os.IsNotExist(err) {
			states = append(states, fileState{Path: path, Missing: true})
			continue
		}
		if err != nil {
			return s, failure("git.worktree", "stat_failed")
		}
		b, err := readLocalFile(full)
		if err != nil {
			return s, failure("git.worktree", "read_failed")
		}
		states = append(states, fileState{Path: path, Hash: digest(b), Mode: uint32(info.Mode())})
	}
	b, _ := json.Marshal(struct {
		Files  []GitFile
		States []fileState
	}{s.files, states})
	s.state = digest(b)
	return s, nil
}
func capture(root string, paths []string, clock time.Time, unlimited ...bool) (Input, error) {
	return captureWithMode(root, paths, clock, "paged100", unlimited...)
}
func captureWithMode(root string, paths []string, clock time.Time, mode string, unlimited ...bool) (Input, error) {
	return captureWithTiming(root, paths, clock, mode, nil, unlimited...)
}
func captureWithTiming(root string, paths []string, clock time.Time, mode string, profile *HistoryTiming, unlimited ...bool) (Input, error) {
	if _, err := historyFetchConfig(mode); err != nil {
		return Input{}, err
	}
	begin := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	in := Input{Version: 1}
	abs, err := filepath.Abs(root)
	if err != nil {
		return in, failure("git.root", "invalid_root")
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return in, failure("git.root", "unavailable_root")
	}
	top, err := git(ctx, abs, "git.root", "rev-parse", "--show-toplevel")
	if err != nil {
		return in, err
	}
	topPath, err := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil || topPath != abs {
		return in, failure("git.root", "repository_root_required")
	}
	in.Root = abs
	if len(paths) == 0 {
		return in, failure("git.paths", "explicit_paths_required")
	}
	seen := map[string]bool{}
	for _, p := range paths {
		if !safePath(p) || seen[p] {
			return in, failure("git.paths", "invalid_or_duplicate_path")
		}
		seen[p] = true
	}
	before, err := readGit(ctx, abs, paths)
	if err != nil {
		return in, err
	}
	in.Head = before.head
	in.IndexHash = before.index
	in.SnapshotHash = before.state
	in.Files = before.files
	if len(in.Files) == 0 {
		return in, failure("git.diff", "no_selected_changes")
	}
	timestamp, err := git(ctx, abs, "git.time_window", "show", "-s", "--format=%ct", before.head)
	if err != nil {
		return in, err
	}
	var since int64
	if _, err = fmt.Sscanf(string(timestamp), "%d", &since); err != nil {
		return in, failure("git.time_window", "invalid_timestamp")
	}
	roots := []string{abs}

	targets := map[string]bool{}
	for _, file := range in.Files {
		targets[file.Path] = true
		if file.OldPath != "" {
			targets[file.OldPath] = true
		}
	}
	in.SinceUnix = since
	in.UntilUnix = clock.Unix()
	historyCtx := ctx
	if len(unlimited) > 0 && unlimited[0] {
		historyCtx = context.WithValue(context.Background(), unlimitedHistoryContext{}, true)
	}
	historyCtx = context.WithValue(historyCtx, historyFetchModeContext{}, mode)
	if profile != nil {
		historyCtx = context.WithValue(historyCtx, historyTimingContext{}, profile)
	}
	in.History = readHistory(historyCtx, abs, roots, targets, since, clock.Unix())
	// Verification is bounded separately so a history timeout cannot hide mutation.
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer verifyCancel()
	after, err := readGit(verifyCtx, abs, paths)
	if err != nil {
		return in, &StageError{Stage: "git.verify", Code: "snapshot_unavailable", Cause: err.(*StageError), Category: "error", Reason: "git_snapshot_unavailable"}
	}
	if before.head != after.head || before.index != after.index || before.state != after.state {
		return in, failure("git.verify", "snapshot_changed")
	}
	in.CaptureMS = float64(time.Since(begin).Microseconds()) / 1000
	return in, nil
}
func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
func normalizeHistoryPath(root, cwd, name string) string {
	if !filepath.IsAbs(name) {
		name = filepath.Join(cwd, name)
	}
	rel, err := filepath.Rel(root, filepath.Clean(name))
	if err != nil || !safePath(filepath.ToSlash(rel)) {
		return ""
	}
	if _, err = resolveTarget(root, filepath.ToSlash(rel)); err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}
func writeNew(name string, value any, unlimited ...bool) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return failure("output.encode", "invalid_value")
	}
	if len(b) > maxInputBytes && !(len(unlimited) > 0 && unlimited[0]) {
		return failure("output.encode", "input_limit")
	}
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return failure("output.write", "create_failed")
	}
	defer f.Close()
	if _, err = io.Copy(f, bytes.NewReader(append(b, '\n'))); err != nil {
		return failure("output.write", "write_failed")
	}
	if f.Sync() != nil {
		return failure("output.write", "sync_failed")
	}
	return nil
}

func readLocalFile(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, receiveLimit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > receiveLimit {
		return nil, fmt.Errorf("file_limit")
	}
	return b, nil
}
