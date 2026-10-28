package historyattribution

import (
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
	paths := []string{}
	for path := range targets {
		variants := []string{filepath.Join(root, filepath.FromSlash(path))}
		if cwd != "" && filepath.Clean(cwd) == filepath.Clean(root) {
			variants = append(variants, path, "./"+path)
		}
		found := false
		for _, v := range variants {
			if pathMention(command, v) {
				found = true
				break
			}
		}
		if found {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}
func pathMention(command, path string) bool {
	boundary := func(b byte) bool { return strings.ContainsRune(" \t\r\n\"'`=;|&(){}<>", rune(b)) }
	for offset := 0; offset < len(command); {
		index := strings.Index(command[offset:], path)
		if index < 0 {
			return false
		}
		index += offset
		end := index + len(path)
		if (index == 0 || boundary(command[index-1])) && (end == len(command) || boundary(command[end])) {
			return true
		}
		offset = index + 1
	}
	return false
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
	unique, err := uniqueHistorySources(h)
	if err != nil {
		return
	}
	safe := func(s string) bool { return safeText(s) && (request.SafeText == nil || request.SafeText(s)) }
	refs := map[string]string{}
	for ref, s := range sources {
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
		for _, c := range turn.Commands {
			keyBytes, _ := json.Marshal([]string{turn.Thread, turn.Turn, c.ItemID})
			key := string(keyBytes)
			body, _ := json.Marshal(c)
			if c.ItemID == "" || (fingerprints[key] != "" && fingerprints[key] != string(body)) {
				out.Acquisition.Status = "partial"
				out.Acquisition.Reasons = append(out.Acquisition.Reasons, "history_parse_failed")
				return
			}
			fingerprints[key] = string(body)
		}
	}
	seen := map[string]bool{}
	for _, turn := range unique.Turns {
		for _, command := range turn.Commands {
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
			if !commandOK || turn.Request == "" || !safe(turn.Request) {
				continue
			}
			paths := mentionedPaths(request.Root, command.Cwd, command.Command, targets)
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
				context := planning.AgentHistoryContext{ContextRef: ref, UserRequests: []string{}, FinalResponses: []string{}}
				if len(turn.UserMessages) > 0 {
					for _, m := range turn.UserMessages {
						if safe(m.Text) {
							context.UserRequests = append(context.UserRequests, m.Text)
						}
					}
				} else {
					context.UserRequests = append(context.UserRequests, turn.Request)
				}
				if len(turn.FinalMessages) > 0 {
					for _, m := range turn.FinalMessages {
						if safe(m.Text) {
							context.FinalResponses = append(context.FinalResponses, m.Text)
						}
					}
				} else if turn.Final != "" && safe(turn.Final) {
					context.FinalResponses = append(context.FinalResponses, turn.Final)
				}
				out.Contexts = append(out.Contexts, context)
				sources[ref] = ContextSource{ThreadID: turn.Thread, TurnID: turn.Turn}
			}
			execution := fmt.Sprintf("S%d", len(out.Commands)+1)
			out.Commands = append(out.Commands, planning.AgentCommandExecutionContext{ExecutionRef: execution, ContextRef: ref, Command: command.Command, MentionedFileIDs: ids})
			commandSources[execution] = CommandSource{ThreadID: turn.Thread, TurnID: turn.Turn, ItemID: command.ItemID, Cwd: command.Cwd, Status: command.Status, ItemOrder: command.ItemOrder, ExitCode: command.ExitCode}
		}
	}
}
