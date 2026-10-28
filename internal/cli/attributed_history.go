package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/exitcode"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/historyattribution"
	runmetrics "github.com/natsuki0413/commiter-cli/internal/metrics"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

var collectAttributedHistory = historyattribution.Collect

// attributedHistoryForPlan is the normal CLI's Extract/Transform boundary.
// A regeneration may reuse extraction only for the same root and fixed input;
// phase-specific projection always starts again from the original host result.
func attributedHistoryForPlan(ctx context.Context, root string, snapshot gitstate.Snapshot, values config.Values, document contextinput.Document, sensitive planning.SensitiveValues) (*planning.GroupingHistory, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, exitcode.New(exitcode.Safety, "git_snapshot_unavailable")
	}
	if err = verifyAttributionSnapshot(ctx, canonical, snapshot, values); err != nil {
		return nil, err
	}
	cycle, _ := ctx.Value(codexCycleKey{}).(*codexCycle)
	if cycle == nil {
		cycle = &codexCycle{started: time.Now()}
	}
	raw, _ := json.Marshal(struct {
		Root, Head, Index string
		SensitivePatterns []string
		Files             []contextinput.File
	}{canonical, snapshot.Head, snapshot.IndexIdentity, values.SensitivePatterns, document.Files})
	key := sha256.Sum256(raw)
	if cycle.attributionCollected && cycle.attributionKey == key {
		runmetrics.FromContext(ctx).SetCodexContext(cycle.attribution.Status, cycle.attribution.Contexts)
		return cycle.attribution.History, nil
	}
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(queryCtx, "git", "-C", canonical, "show", "-s", "--format=%ct", snapshot.Head)
	command.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	timestamp, err := command.Output()
	if err != nil {
		return nil, exitcode.New(exitcode.Safety, "git_snapshot_unavailable")
	}
	since, err := strconv.ParseInt(strings.TrimSpace(string(timestamp)), 10, 64)
	if err != nil {
		return nil, exitcode.New(exitcode.Safety, "git_snapshot_unavailable")
	}
	request := historyattribution.Request{Root: canonical, Since: time.Unix(since, 0), Until: cycle.started, Files: document.Files,
		AllowPath: func(path string) bool {
			allowed, e := gitstate.ContentReadAllowed(path, values.SensitivePatterns, false)
			return e == nil && allowed
		},
		SafeCommand: func(text string, paths []string) bool {
			scanned := planning.CommandSafetyText(text, paths)
			return !sensitive.Contains([]byte(text)) && !planning.ExtractSensitiveValues([]byte(scanned)).Contains([]byte(scanned)) && !inlineLogSecret.Match([]byte(text))
		},
		SafeText: func(text string) bool {
			data := []byte(text)
			return !sensitive.Contains(data) && !planning.ExtractSensitiveValues(data).Contains(data) && !inlineLogSecret.Match(data)
		},
	}
	result, err := collectAttributedHistory(queryCtx, request)
	// Verification has its own deadline, so a history timeout cannot hide mutation.
	if verifyErr := verifyAttributionSnapshot(ctx, canonical, snapshot, values); verifyErr != nil {
		return nil, verifyErr
	}
	if err != nil {
		var stage *historyattribution.StageError
		if errors.As(err, &stage) {
			return nil, exitcode.New(exitcode.LLM, "history_attribution_failed: "+stage.Stage+":"+stage.Reason)
		}
		return nil, exitcode.New(exitcode.LLM, "history_attribution_failed")
	}
	cycle.attributionCollected, cycle.attributionKey, cycle.attribution = true, key, result
	runmetrics.FromContext(ctx).SetCodexContext(result.Status, result.Contexts)
	return result.History, nil
}

func verifyAttributionSnapshot(parent context.Context, root string, original gitstate.Snapshot, values config.Values) error {
	if parent.Err() != nil {
		return exitcode.New(exitcode.Canceled, "history_acquisition_canceled")
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	paths := []string{}
	seen := map[string]bool{}
	for _, change := range original.Changes {
		for _, path := range changePaths(change) {
			if !seen[path] {
				seen[path] = true
				paths = append(paths, path)
			}
		}
	}
	current, err := revalidateSnapshot(ctx, root, values, paths, original)
	if err != nil {
		return exitcode.New(exitcode.Safety, "git_snapshot_unavailable")
	}
	if current.Branch != original.Branch || len(changedSnapshotPaths(original, current)) > 0 {
		return exitcode.New(exitcode.Safety, "snapshot_changed")
	}
	return nil
}
