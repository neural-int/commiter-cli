package historyattribution

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type commandHistoryContext struct{}

// Literal path mentions are neither shell interpretation nor edit evidence.
func mentionedPaths(root, cwd, command string, targets map[string]bool) []string {
	paths, _ := mentionedPathsContext(context.Background(), root, cwd, command, targets)
	return paths
}
func mentionedPathsContext(ctx context.Context, root, cwd, command string, targets map[string]bool) ([]string, error) {
	paths := []string{}
	for path := range targets {
		if err := matchingCheckpoint(ctx); err != nil {
			return nil, err
		}
		variants := []string{filepath.Join(root, filepath.FromSlash(path))}
		if cwd != "" && filepath.Clean(cwd) == filepath.Clean(root) {
			variants = append(variants, path, "./"+path)
		}
		found := false
		for _, v := range variants {
			matched, err := pathMentionContext(ctx, command, v)
			if err != nil {
				return nil, err
			}
			if matched {
				found = true
				break
			}
		}
		if found {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}
func pathMention(command, path string) bool {
	matched, _ := pathMentionContext(context.Background(), command, path)
	return matched
}
func pathMentionContext(ctx context.Context, command, path string) (bool, error) {
	boundary := func(b byte) bool { return strings.ContainsRune(" \t\r\n\"'`=;|&(){}<>", rune(b)) }
	for offset := 0; offset < len(command); {
		if err := matchingCheckpoint(ctx); err != nil {
			return false, err
		}
		index := strings.Index(command[offset:], path)
		if index < 0 {
			return false, nil
		}
		index += offset
		end := index + len(path)
		if (index == 0 || boundary(command[index-1])) && (end == len(command) || boundary(command[end])) {
			return true, nil
		}
		offset = index + 1
	}
	return false, nil
}

func commandSafetyPaths(root string, targets map[string]bool) []string {
	paths := []string{}
	for path := range targets {
		paths = append(paths, path, filepath.Join(root, path))
	}
	return paths
}
func safeCommandText(command, root string, targets map[string]bool) bool {
	return safeText(planning.CommandSafetyText(command, commandSafetyPaths(root, targets)))
}

func appendCommandContexts(request Request, h History, out *planning.GroupingHistory, sources map[string]ContextSource, commandSources map[string]CommandSource) {
	_ = appendCommandContextsContext(context.Background(), request, h, out, sources, commandSources)
}
func appendCommandContextsContext(ctx context.Context, request Request, h History, out *planning.GroupingHistory, sources map[string]ContextSource, commandSources map[string]CommandSource) error {
	if err := matchingCheckpoint(ctx); err != nil {
		return err
	}
	unique, err := uniqueHistorySourcesContext(ctx, h)
	if err != nil {
		return matchingCheckpoint(ctx)
	}
	safe := func(s string) bool { return safeText(s) && (request.SafeText == nil || request.SafeText(s)) }
	refs := map[string]string{}
	for ref, s := range sources {
		if err := matchingCheckpoint(ctx); err != nil {
			return err
		}
		refs[sourceTurnKey(s.ThreadID, s.TurnID)] = ref
	}
	targets := map[string]bool{}
	for _, f := range request.Files {
		for _, path := range []*string{f.OldPath, f.NewPath} {
			if path != nil && safePath(*path) && request.AllowPath != nil && request.AllowPath(*path) {
				targets[*path] = true
			}
		}
	}
	// Validate every command source before emitting any auxiliary evidence.
	fingerprints := map[string]string{}
	for _, turn := range unique.Turns {
		if err := matchingCheckpoint(ctx); err != nil {
			return err
		}
		for _, c := range turn.Commands {
			if err := matchingCheckpoint(ctx); err != nil {
				return err
			}
			keyBytes, _ := json.Marshal([]string{turn.Thread, turn.Turn, c.ItemID})
			key := string(keyBytes)
			body, _ := json.Marshal(c)
			if c.ItemID == "" || (fingerprints[key] != "" && fingerprints[key] != string(body)) {
				out.Acquisition.Status = "partial"
				out.Acquisition.Reasons = append(out.Acquisition.Reasons, "history_parse_failed")
				return nil
			}
			fingerprints[key] = string(body)
		}
	}
	seen := map[string]bool{}
	for _, turn := range unique.Turns {
		if err := matchingCheckpoint(ctx); err != nil {
			return err
		}
		for _, command := range turn.Commands {
			if err := matchingCheckpoint(ctx); err != nil {
				return err
			}
			keyBytes, _ := json.Marshal([]string{turn.Thread, turn.Turn, command.ItemID})
			key := string(keyBytes)
			if seen[key] {
				continue
			}
			seen[key] = true
			commandOK := safe(planning.CommandSafetyText(command.Command, commandSafetyPaths(request.Root, targets)))
			if request.SafeCommand != nil {
				commandOK = commandOK && request.SafeCommand(command.Command, commandSafetyPaths(request.Root, targets))
			}
			if err := matchingCheckpoint(ctx); err != nil {
				return err
			}
			if !commandOK || len(safeFinalResponses(turn, safe)) == 0 {
				continue
			}
			paths, err := mentionedPathsContext(ctx, request.Root, command.Cwd, command.Command, targets)
			if err != nil {
				return err
			}
			ids := []string{}
			for _, f := range request.Files {
				for _, path := range paths {
					if (f.OldPath != nil && *f.OldPath == path) || (f.NewPath != nil && *f.NewPath == path) {
						ids = append(ids, f.ID)
						break
					}
				}
			}
			if len(ids) == 0 {
				continue
			}
			turnKey := sourceTurnKey(turn.Thread, turn.Turn)
			ref := refs[turnKey]
			if ref == "" {
				ref = fmt.Sprintf("H%d", len(out.Contexts)+1)
				refs[turnKey] = ref
				context := planning.AgentHistoryContext{ContextRef: ref, FinalResponses: safeFinalResponses(turn, safe)}
				out.Contexts = append(out.Contexts, context)
				sources[ref] = ContextSource{ThreadID: turn.Thread, TurnID: turn.Turn}
			}
			execution := fmt.Sprintf("S%d", len(out.Commands)+1)
			out.Commands = append(out.Commands, planning.AgentCommandExecutionContext{ExecutionRef: execution, ContextRef: ref, Command: command.Command, MentionedFileIDs: ids})
			commandSources[execution] = CommandSource{ThreadID: turn.Thread, TurnID: turn.Turn, ItemID: command.ItemID, Cwd: command.Cwd, Status: command.Status, ItemOrder: command.ItemOrder, ExitCode: command.ExitCode}
		}
	}
	return matchingCheckpoint(ctx)
}
