package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	runmetrics "github.com/natsuki0413/commiter-cli/internal/metrics"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

type codexCycleKey struct{}
type codexCycle struct {
	started   time.Time
	collected bool
	result    codexcontext.Result
}

var collectCodexContext = codexcontext.Collect

// This supplements the existing secret-value detector for assignments embedded
// in prose. Suspected secrets are omitted, never logged or returned in errors.
var inlineLogSecret = regexp.MustCompile(`(?i)(?:api[_-]?key|access[_-]?token|password|passwd|client[_-]?secret|authorization)\s*[=:]\s*\S+`)

func optionalCodexContext(ctx context.Context, root string, snapshot gitstate.Snapshot, values config.Values, files []relation.File, sensitive planning.SensitiveValues) []codexcontext.Record {
	cycle, _ := ctx.Value(codexCycleKey{}).(*codexCycle)
	if cycle == nil {
		cycle = &codexCycle{started: time.Now()}
	}
	if cycle.collected {
		return cycle.result.Records
	}
	cycle.collected = true
	cycle.result = codexcontext.Result{Status: "unavailable"}
	defer func() { runmetrics.FromContext(ctx).SetCodexContext(cycle.result.Status, len(cycle.result.Records)) }()
	if values.Planner != config.CandidatePlanner {
		cycle.result.Status = "not_requested"
		return nil
	}
	collectCtx, cancel := context.WithTimeout(ctx, codexcontext.Timeout)
	defer cancel()
	// Use the snapshot's explicit commit, never a potentially advanced HEAD.
	cmd := exec.CommandContext(collectCtx, "git", "-C", root, "show", "-s", "--format=%ct", snapshot.Head)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return nil
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil
	}
	allowPath := func(path string) bool {
		allowed, err := gitstate.ContentReadAllowed(path, values.SensitivePatterns, false)
		return err == nil && allowed
	}
	request := codexcontext.Request{Root: canonical, Start: time.Unix(seconds, 0), End: cycle.started, AllowPath: allowPath, SafeText: func(text string) bool {
		data := []byte(text)
		return !sensitive.Contains(data) && !planning.ExtractSensitiveValues(data).Contains(data) && !inlineLogSecret.Match(data)
	}}
	// A Codex session can start in the main checkout and edit a dedicated
	// worktree. Discover candidates only within this repository's worktree set;
	// the path association still requires edits inside the selected worktree.
	worktrees := exec.CommandContext(collectCtx, "git", "-C", root, "worktree", "list", "--porcelain", "-z")
	worktrees.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	if output, err := worktrees.Output(); err == nil {
		for _, field := range strings.Split(string(output), "\x00") {
			if strings.HasPrefix(field, "worktree ") {
				path := strings.TrimPrefix(field, "worktree ")
				request.SessionRoots = append(request.SessionRoots, path)
				if canonicalPath, err := filepath.EvalSymlinks(path); err == nil {
					request.SessionRoots = append(request.SessionRoots, canonicalPath)
				}
			}
		}
	}
	seenPaths := map[string]bool{}
	for _, file := range files {
		change := file.Change
		if change.Binary || change.Opaque || change.Sensitive {
			continue
		}
		for _, path := range changePaths(change) {
			if !allowPath(path) {
				continue
			}
			if !seenPaths[path] {
				request.Targets = append(request.Targets, codexcontext.Target{Path: path})
				seenPaths[path] = true
			}
		}
	}
	cycle.result = collectCodexContext(collectCtx, request)
	return cycle.result.Records
}
