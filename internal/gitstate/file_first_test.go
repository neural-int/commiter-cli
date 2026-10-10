package gitstate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFileFirstReplayOneThroughSixteenFilesPreservesRepository(t *testing.T) {
	fileFirstIsolateGitConfig(t)
	for count := 1; count <= FileFirstMaxFiles; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			repo := newRepository(t)
			write(t, repo, "outside.txt", "base\n", 0o644)
			for i := 0; i < count; i++ {
				write(t, repo, fmt.Sprintf("f%02d.txt", i), "before\n", 0o644)
			}
			git(t, repo, "add", "--all")
			git(t, repo, "commit", "-m", "base")
			for i := 0; i < count; i++ {
				write(t, repo, fmt.Sprintf("f%02d.txt", i), fmt.Sprintf("after %d\r\nno final newline", i), 0o644)
			}
			write(t, repo, "outside.txt", "staged outside\n", 0o644)
			git(t, repo, "add", "outside.txt")
			write(t, repo, "outside.txt", "working outside remains\n", 0o644)
			opts := Options{Pathspecs: []string{"f*.txt"}}
			snapshot, err := Collect(repo, opts)
			if err != nil {
				t.Fatal(err)
			}
			expected := fileFirstExpectedTree(t, repo, snapshot)
			before := fileFirstRepositoryState(t, repo)
			preview, err := PreviewFileFirst(snapshot, opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(preview.Groups) != count || len(preview.StageTrees) != count || preview.FinalTree != expected || preview.StageTrees[count-1] != expected {
				t.Fatalf("incomplete replay: %+v, expected %s", preview, expected)
			}
			for i, group := range preview.Groups {
				if group.FileID != snapshot.Changes[i].ID || len(group.Paths) != 1 {
					t.Fatalf("ownership: %+v", group)
				}
			}
			if count == FileFirstMaxFiles {
				for i, j := 0, len(snapshot.Changes)-1; i < j; i, j = i+1, j-1 {
					snapshot.Changes[i], snapshot.Changes[j] = snapshot.Changes[j], snapshot.Changes[i]
				}
				reversed, err := PreviewFileFirst(snapshot, opts)
				if err != nil || !reflect.DeepEqual(preview, reversed) {
					t.Fatalf("input-order dependence: %+v %v", reversed, err)
				}
			}
			if after := fileFirstRepositoryState(t, repo); !reflect.DeepEqual(before, after) {
				t.Fatal("preview changed HEAD/index/worktree/object database")
			}
		})
	}
}

func TestFileFirstMixedGitBoundariesAndBinaryReplay(t *testing.T) {
	fileFirstIsolateGitConfig(t)
	repo := newRepository(t)
	for name, data := range map[string]string{"partial.txt": "base\n", "deleted.txt": "delete\n", "old name.txt": "rename\n", "mode.txt": "mode\n", "staged.txt": "base\n", "binary.dat": "\x00before\xff", "empty.txt": ""} {
		write(t, repo, name, data, 0o644)
	}
	git(t, repo, "add", "--all")
	git(t, repo, "commit", "-m", "base")
	write(t, repo, "partial.txt", "staged\n", 0o644)
	git(t, repo, "add", "partial.txt")
	write(t, repo, "partial.txt", "working final longer\n", 0o644)
	write(t, repo, "staged.txt", "staged only\n", 0o644)
	git(t, repo, "add", "staged.txt")
	git(t, repo, "mv", "old name.txt", "日本語\n name.txt")
	write(t, repo, "日本語\n name.txt", "renamed and changed\n", 0o644)
	if err := os.Remove(filepath.Join(repo, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(repo, "mode.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, repo, "binary.dat", "\x00after\xff\r\n", 0o644)
	write(t, repo, "new binary.dat", "\x00new\xfe", 0o644)
	write(t, repo, "empty.txt", "not empty anymore", 0o644)
	write(t, repo, "empty added.txt", "", 0o644)
	snapshot, err := Collect(repo, Options{})
	if err != nil {
		t.Fatal(err)
	}
	expected := fileFirstExpectedTree(t, repo, snapshot)
	// Temporary-index operations must never execute post-index-change hooks.
	write(t, repo, ".git/hooks/post-index-change", "#!/bin/sh\nprintf 'ran' > hook-was-run\n", 0o755)
	before := fileFirstRepositoryState(t, repo)
	preview, err := PreviewFileFirst(snapshot, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Groups) != len(snapshot.Changes) || preview.FinalTree != expected {
		t.Fatalf("incomplete: %+v", preview)
	}
	if !reflect.DeepEqual(before, fileFirstRepositoryState(t, repo)) {
		t.Fatal("mixed preview mutated repository")
	}
}

func TestFileFirstStopsOnDriftAndInvalidOwnershipWithoutPartialResult(t *testing.T) {
	fileFirstIsolateGitConfig(t)
	for _, scenario := range []string{"missing", "duplicate", "unknown-id", "path-tamper", "hash-tamper", "worktree-drift", "index-drift", "head-drift", "extra-selected", "excluded-drift", "no-tree-delta", "locked", "merge", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			repo := newRepository(t)
			write(t, repo, "one.txt", "base\n", 0o644)
			write(t, repo, "two.txt", "base\n", 0o644)
			git(t, repo, "add", "--all")
			git(t, repo, "commit", "-m", "base")
			write(t, repo, "one.txt", "changed\n", 0o644)
			write(t, repo, "two.txt", "changed\n", 0o644)
			snapshot, err := Collect(repo, Options{})
			if err != nil {
				t.Fatal(err)
			}
			opts := Options{}
			switch scenario {
			case "missing":
				snapshot.Changes = snapshot.Changes[:1]
			case "duplicate":
				snapshot.Changes[1] = snapshot.Changes[0]
			case "unknown-id":
				snapshot.Changes[0].ID = "unknown"
			case "path-tamper":
				snapshot.Changes[0].NewPath = stringPointer("../escape.txt")
			case "hash-tamper":
				snapshot.Changes[0].ChangeHash = strings.Repeat("0", 64)
			case "worktree-drift":
				write(t, repo, "one.txt", "later changed\n", 0o644)
			case "index-drift":
				git(t, repo, "add", "one.txt")
			case "head-drift":
				git(t, repo, "commit", "--allow-empty", "-m", "another")
			case "extra-selected":
				write(t, repo, "three.txt", "new\n", 0o644)
			case "excluded-drift":
				snapshot.Excluded = append(snapshot.Excluded, Excluded{Path: "fabricated", Reason: "excluded"})
			case "no-tree-delta":
				git(t, repo, "add", "one.txt")
				write(t, repo, "one.txt", "base\n", 0o644)
			case "locked":
				write(t, repo, ".git/index.lock", "held", 0o600)
			case "merge":
				write(t, repo, ".git/MERGE_HEAD", snapshot.Head, 0o600)
			case "cancel":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				opts.Context = ctx
			}
			before := fileFirstRepositoryState(t, repo)
			preview, err := PreviewFileFirst(snapshot, opts)
			if err == nil || !reflect.DeepEqual(preview, FileFirstPreview{}) {
				t.Fatalf("partial/accepted unsafe preview: %+v %v", preview, err)
			}
			if !reflect.DeepEqual(before, fileFirstRepositoryState(t, repo)) {
				t.Fatal("failure mutated repository")
			}
		})
	}
}

func TestFileFirstRejectsUnsupportedStatesAndBudgets(t *testing.T) {
	fileFirstIsolateGitConfig(t)
	for _, scenario := range []string{"seventeen", "symlink", "gitlink", "large-before", "large-after", "lines", "filter", "text", "autocrlf", "filemode", "fsmonitor", "skip-worktree", "path-overlap"} {
		t.Run(scenario, func(t *testing.T) {
			repo := newRepository(t)
			base := "base\n"
			if scenario == "large-before" {
				base = strings.Repeat("x", fileFirstMaxBytes+1)
			}
			write(t, repo, "one.txt", base, 0o644)
			git(t, repo, "add", "--all")
			git(t, repo, "commit", "-m", "base")
			write(t, repo, "one.txt", "changed\n", 0o644)
			switch scenario {
			case "seventeen":
				for i := 0; i < 16; i++ {
					write(t, repo, fmt.Sprintf("new%02d.txt", i), "new", 0o644)
				}
			case "symlink":
				if err := os.Symlink("one.txt", filepath.Join(repo, "link.txt")); err != nil {
					t.Fatal(err)
				}
			case "gitlink":
				git(t, repo, "update-index", "--add", "--cacheinfo", "160000", strings.TrimSpace(git(t, repo, "rev-parse", "HEAD")), "submodule")
				if err := os.Mkdir(filepath.Join(repo, "submodule"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "large-after":
				write(t, repo, "one.txt", strings.Repeat("x", fileFirstMaxBytes+1), 0o644)
			case "lines":
				write(t, repo, "one.txt", strings.Repeat("x\n", fileFirstMaxLines)+"x", 0o644)
			case "filter":
				// The filter would leave a marker if any selected content were sent
				// through Git add. Preview must reject it without executing it.
				write(t, repo, ".gitattributes", "one.txt filter=marker\n", 0o644)
				git(t, repo, "config", "filter.marker.clean", "touch filter-was-run; cat")
			case "text":
				write(t, repo, ".gitattributes", "one.txt text\n", 0o644)
			case "autocrlf":
				git(t, repo, "config", "core.autocrlf", "true")
			case "filemode":
				git(t, repo, "config", "core.filemode", "false")
			case "path-overlap":
				if err := os.Remove(filepath.Join(repo, "one.txt")); err != nil {
					t.Fatal(err)
				}
				write(t, repo, "one.txt/new.txt", "new", 0o644)
			}
			snapshot, err := Collect(repo, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "gitlink" && len(snapshot.Changes) != 2 {
				t.Fatal("fixture did not select its gitlink")
			}
			if scenario == "filter" {
				if err := os.Remove(filepath.Join(repo, "filter-was-run")); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
			}
			if scenario == "fsmonitor" {
				git(t, repo, "config", "core.fsmonitor", "touch fsmonitor-was-run")
			}
			if scenario == "skip-worktree" {
				git(t, repo, "update-index", "--skip-worktree", "one.txt")
				snapshot.IndexIdentity, err = indexIdentity(context.Background(), repo)
				if err != nil {
					t.Fatal(err)
				}
			}
			before := fileFirstRepositoryState(t, repo)
			preview, err := PreviewFileFirst(snapshot, Options{})
			if err == nil || !reflect.DeepEqual(preview, FileFirstPreview{}) {
				t.Fatalf("unsupported preview: %+v %v", preview, err)
			}
			if !reflect.DeepEqual(before, fileFirstRepositoryState(t, repo)) {
				t.Fatal("unsupported preview changed repository or ran filter")
			}
		})
	}
}

// Construct the target independently with actual git add in a private index.
// This runs before preservation measurements and only in authored test repos.
func fileFirstExpectedTree(t *testing.T, repo string, snapshot Snapshot) string {
	t.Helper()
	env := append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(t.TempDir(), "index"), "GIT_LITERAL_PATHSPECS=1")
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
		return strings.TrimSpace(string(out))
	}
	run("read-tree", "HEAD")
	for _, c := range snapshot.Changes {
		paths := []string{}
		if c.OldPath != nil {
			paths = append(paths, *c.OldPath)
		}
		if c.NewPath != nil && (c.OldPath == nil || *c.NewPath != *c.OldPath) {
			paths = append(paths, *c.NewPath)
		}
		run(append([]string{"add", "--"}, paths...)...)
	}
	return run("write-tree")
}

// Include raw index bytes, all worktree bytes/modes, refs and stored objects.
// Git status alone could miss accidental index refreshes or orphan objects.
func fileFirstRepositoryState(t *testing.T, repo string) map[string]string {
	t.Helper()
	state := map[string]string{}
	err := filepath.WalkDir(repo, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		var data []byte
		if info.Mode()&os.ModeSymlink != 0 {
			value, e := os.Readlink(path)
			err = e
			data = []byte(value)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			return err
		}
		state[rel] = fmt.Sprintf("%s:%s", info.Mode(), data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestFileFirstPreservesLiteralPathsWithAlternateDirectorySeparator(t *testing.T) {
	fileFirstIsolateGitConfig(t)
	// Environment isolation is exercised through replay, including an object
	// path containing a colon (Git alternate-directory separator on Unix).
	repo := filepath.Join(t.TempDir(), "repo:with spaces")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "--initial-branch=main")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Test")
	write(t, repo, "literal*.txt", "base\n", 0o644)
	git(t, repo, "add", "--all")
	git(t, repo, "commit", "-m", "base")
	write(t, repo, "literal*.txt", "changed\n", 0o644)
	snapshot, err := Collect(repo, Options{})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewFileFirst(snapshot, Options{})
	if err != nil || len(preview.Groups) != 1 {
		t.Fatalf("%+v %v", preview, err)
	}
	if !bytes.Equal([]byte(preview.Groups[0].Paths[0]), []byte("literal*.txt")) {
		t.Fatal("literal path changed")
	}
}

func fileFirstIsolateGitConfig(t *testing.T) {
	t.Helper()
	// Authored repositories have an explicit Git contract; personal/system
	// clean filters and fsmonitor must not change that contract between hosts.
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
}

func TestFileFirstDetectsConcurrentWorktreeChangeDuringReplay(t *testing.T) {
	fileFirstIsolateGitConfig(t)
	repo := newRepository(t)
	write(t, repo, "one.txt", "base\n", 0o644)
	git(t, repo, "add", "--all")
	git(t, repo, "commit", "-m", "base")
	write(t, repo, "one.txt", "planned\n", 0o644)
	snapshot, err := Collect(repo, Options{})
	if err != nil {
		t.Fatal(err)
	}
	before := fileFirstRepositoryState(t, repo)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	shim := t.TempDir()
	write(t, shim, "git", `#!/bin/sh
for arg in "$@"; do
  if [ "$arg" = write-tree ] && [ ! -e "$FILEFIRST_DRIFT_MARKER" ]; then
    printf 'concurrent writer\n' > "$FILEFIRST_DRIFT_PATH"
    printf 'done' > "$FILEFIRST_DRIFT_MARKER"
  fi
done
exec "$FILEFIRST_REAL_GIT" "$@"
`, 0o755)
	t.Setenv("FILEFIRST_REAL_GIT", realGit)
	t.Setenv("FILEFIRST_DRIFT_PATH", filepath.Join(repo, "one.txt"))
	t.Setenv("FILEFIRST_DRIFT_MARKER", filepath.Join(shim, "marker"))
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	preview, err := PreviewFileFirst(snapshot, Options{})
	if !IsKind(err, ErrorSafety) || !strings.Contains(err.Error(), "snapshot changed") || !reflect.DeepEqual(preview, FileFirstPreview{}) {
		t.Fatalf("concurrent drift returned plan: %+v %v", preview, err)
	}
	info, err := os.Stat(filepath.Join(repo, "one.txt"))
	if err != nil {
		t.Fatal(err)
	}
	before["one.txt"] = fmt.Sprintf("%s:%s", info.Mode(), "concurrent writer\n")
	if !reflect.DeepEqual(before, fileFirstRepositoryState(t, repo)) {
		t.Fatal("preview mutated Git or overwrote concurrent writer")
	}
}
