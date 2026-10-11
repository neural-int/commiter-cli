package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Fixture collection and syntax preparation reused from benchmark167 at c30dc2d8.
func digest(value []byte) string  { d := sha256.Sum256(value); return hex.EncodeToString(d[:]) }
func jsonDigest(value any) string { data, _ := json.Marshal(value); return digest(data) }

// Hash every fixture worktree and Git file, including index/objects/refs.
func repositoryDigest(root string) (string, error) {
	entries := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if entry.IsDir() {
			return nil
		}
		info, e := entry.Info()
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, path)
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("unexpected fixture symlink")
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		entries[rel] = fmt.Sprintf("%o:%s", info.Mode(), digest(data))
		return nil
	})
	return jsonDigest(entries), err
}
func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_OPTIONAL_LOCKS=0", "GIT_LITERAL_PATHSPECS=1", "GIT_AUTHOR_DATE=2026-10-10T00:00:00Z", "GIT_COMMITTER_DATE=2026-10-10T00:00:00Z")
	data, e := cmd.Output()
	if e != nil {
		return "", errors.New("fixture Git command failed")
	}
	return string(data), nil
}
func makeRepository(f fixture) (string, error) {
	root, e := os.MkdirTemp("", "commiter-168-fixture-")
	if e != nil {
		return "", e
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.email", "fixture@example.invalid"}, {"config", "user.name", "Issue 168 fixture"}, {"config", "core.autocrlf", "false"}} {
		if _, e = git(root, args...); e != nil {
			return "", e
		}
	}
	for _, file := range f.Files {
		path := filepath.Join(root, file.Path)
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return "", e
		}
		if e = os.WriteFile(path, []byte(file.Before), 0600); e != nil {
			return "", e
		}
	}
	if _, e = git(root, "add", "--all"); e != nil {
		return "", e
	}
	if _, e = git(root, "commit", "-qm", "fixture baseline"); e != nil {
		return "", e
	}
	for _, file := range f.Files {
		if e = os.WriteFile(filepath.Join(root, file.Path), []byte(file.After), 0600); e != nil {
			return "", e
		}
	}
	return root, nil
}

var hunkHeader = regexp.MustCompile(`(?m)^@@ -[0-9]+(?:,[0-9]+)? \+([0-9]+)(?:,([0-9]+))? @@`)

func prepare(root string, snapshot gitstate.Snapshot) (contextinput.Prepared, int, int, error) {
	results := []syntax.ChangeResult{}
	diffBytes, diffLines := 0, 0
	for _, change := range snapshot.Changes {
		path := *change.NewPath
		raw, e := git(root, "diff", "--no-ext-diff", "--no-textconv", "--unified=0", "HEAD", "--", path)
		if e != nil {
			return contextinput.Prepared{}, 0, 0, e
		}
		diffBytes += len(raw)
		diffLines += strings.Count(raw, "\n")
		hunks := []syntax.Hunk{}
		for _, m := range hunkHeader.FindAllStringSubmatch(raw, -1) {
			start, _ := strconv.Atoi(m[1])
			count := 1
			if m[2] != "" {
				count, _ = strconv.Atoi(m[2])
			}
			hunks = append(hunks, syntax.Hunk{StartLine: start, EndLine: start + count - 1})
		}
		content, e := os.ReadFile(filepath.Join(root, path))
		if e != nil {
			return contextinput.Prepared{}, 0, 0, e
		}
		result, e := syntax.AnalyzeChange(syntax.ChangeInput{Change: change, Content: content, RawDiff: raw, Hunks: hunks})
		if e != nil {
			return contextinput.Prepared{}, 0, 0, e
		}
		results = append(results, result)
	}
	document, e := contextinput.Build(snapshot, results)
	if e != nil {
		return contextinput.Prepared{}, 0, 0, e
	}
	prompt, e := planning.Renderer(planning.English)(document)
	if e != nil {
		return contextinput.Prepared{}, 0, 0, e
	}
	// The actual per-profile prompts are independently bounded by both planners.
	return contextinput.Prepared{Document: document, Prompt: prompt, Budget: contextinput.Budget{ContextTokens: 16384}}, diffBytes, diffLines, nil
}
