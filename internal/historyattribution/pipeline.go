package historyattribution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

// Request contains only approved files from the caller's fixed Git snapshot.
// The caller must revalidate that snapshot before and after collection.
type Request struct {
	Root        string
	Since       time.Time
	Until       time.Time
	Files       []contextinput.File
	AllowPath   func(string) bool
	SafeText    func(string) bool
	SafeCommand func(string, []string) bool
}

type CommandSource struct {
	ThreadID, TurnID, ItemID, Cwd, Status string
	ItemOrder                             int
	ExitCode                              *int
}
type ContextSource struct {
	ThreadID, TurnID string
	Edits            []EditSource
}
type EditSource struct {
	ItemID      string
	ChangeIndex int
}

type Result struct {
	ContextSources map[string]ContextSource `json:"-"`
	CommandSources map[string]CommandSource `json:"-"`
	History        *planning.GroupingHistory
	Status         string
	Contexts       int
	Operations     int
	ReceivedBytes  int64
	Requests       int
}

// Collect uses the adopted 100-item, four-worker, current-worktree-only path.
// It never reads Codex rollout files or writes Git. Retrieval failures remain
// explicit optional history failures; malformed fixed Git evidence is an error.
func Collect(parent context.Context, request Request) (Result, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, historyFetchModeContext{}, "pipeline100x4")
	ctx = context.WithValue(ctx, commandHistoryContext{}, true)
	// Disable only the research prototype's legacy 4MB connection-total limit.
	// The parent deadline, 300MB per-response cap and memory guard still apply.
	ctx = context.WithValue(ctx, unlimitedHistoryContext{}, true)
	targets := map[string]bool{}
	for _, file := range request.Files {
		wire, projectionError := planning.ProjectMetadataFile(file)
		if projectionError != nil {
			return Result{}, failure("git.blocks", "unsupported_change_type")
		}
		_ = wire // Commands may mention metadata-only files too.
		for _, path := range []*string{file.OldPath, file.NewPath} {
			if path != nil && safePath(*path) && request.AllowPath != nil && request.AllowPath(*path) {
				targets[*path] = true
			}
		}
	}
	h := History{Status: "complete"}
	if len(targets) > 0 {
		if request.Since.IsZero() || request.Until.Before(request.Since) || request.Root == "" || request.SafeText == nil {
			h.Status = "unavailable"
			h.Error = failure("history.request", "invalid_request").(*StageError)
		} else {
			h = readHistory(ctx, request.Root, []string{request.Root}, targets, request.Since.Unix(), request.Until.Unix())
		}
	}
	sources := map[string]ContextSource{}
	attributed, err := attributeWithSources(request, h, sources)
	if err != nil {
		return Result{}, err
	}
	commandSources := map[string]CommandSource{}
	appendCommandContexts(request, h, attributed, sources, commandSources)
	status := "no_related_evidence"
	if attributed.Acquisition.Status == "unavailable" {
		status = "unavailable"
	} else if attributed.Acquisition.Status == "partial" {
		status = "partial"
	} else if len(attributed.Contexts) > 0 {
		status = "available"
	}
	return Result{ContextSources: sources, CommandSources: commandSources, History: attributed, Status: status, Contexts: len(attributed.Contexts), Operations: len(h.Operations), ReceivedBytes: h.ReceivedBytes, Requests: h.Requests}, nil
}

func sourceTurnKey(thread, turn string) string {
	b, _ := json.Marshal([]string{thread, turn})
	return string(b)
}

// attribute adapts source identities to short references only after matching.
// Distinct edits in the same Turn remain ambiguous even when their context is
// represented once. Source thread/Turn/item IDs never enter the LLM payload.
func attribute(request Request, history History) (*planning.GroupingHistory, error) {
	return attributeWithSources(request, history, nil)
}

func attributeWithSources(request Request, history History, sources map[string]ContextSource) (*planning.GroupingHistory, error) {
	h, err := uniqueHistorySources(history)
	if err != nil {
		h = History{Status: "unavailable", Error: failure("history.identity", "invalid_schema").(*StageError)}
	}
	if h.Status != "complete" && h.Status != "partial" && h.Status != "unavailable" {
		h = History{Status: "unavailable", Error: failure("history.schema", "invalid_schema").(*StageError)}
	}
	out := &planning.GroupingHistory{Contexts: []planning.AgentHistoryContext{}, Files: []planning.AttributedFileHistory{}, Acquisition: planning.AgentHistoryAcquisition{Status: h.Status, Reasons: []string{}}}
	if h.Status != "complete" {
		reason := "history_incomplete"
		if h.Status == "unavailable" {
			reason = "history_unavailable"
		}
		out.Acquisition.Reasons = append(out.Acquisition.Reasons, reason)
	}
	if h.Error != nil && h.Error.Reason != "" && !containsString(out.Acquisition.Reasons, h.Error.Reason) {
		out.Acquisition.Reasons = append(out.Acquisition.Reasons, h.Error.Reason)
	}
	safe := func(text string) bool { return safeText(text) && (request.SafeText == nil || request.SafeText(text)) }
	// Invalid/sensitive retained operations cannot become evidence by projection.
	retained := []Operation{}
	for _, op := range h.Operations {
		if !safePath(op.Path) || (op.OldPath != "" && !safePath(op.OldPath)) || request.AllowPath == nil || !request.AllowPath(op.Path) || (op.OldPath != "" && !request.AllowPath(op.OldPath)) || (op.Diff != nil && !safe(*op.Diff)) {
			out.Acquisition.Status = "partial"
			if !containsString(out.Acquisition.Reasons, "history_incomplete") {
				out.Acquisition.Reasons = append(out.Acquisition.Reasons, "history_incomplete")
			}
			continue
		}
		retained = append(retained, op)
	}
	h.Operations = retained
	h.Status = out.Acquisition.Status
	contexts := map[string]TurnContext{}
	for _, turn := range h.Turns {
		contexts[sourceTurnKey(turn.Thread, turn.Turn)] = turn
	}
	ops := map[string]Operation{}
	for _, op := range h.Operations {
		ops[operationKey(op)] = op
	}
	input := Input{Version: 1, Root: request.Root, Head: "caller-fixed", History: h}
	projected := []planning.MetadataFile{}
	for _, file := range request.Files {
		wire, e := planning.ProjectMetadataFile(file)
		if e != nil {
			return nil, failure("git.blocks", "unsupported_change_type")
		}
		if len(wire.ChangeBlocks) == 0 {
			continue
		}
		path := ""
		if wire.NewPath != nil {
			path = *wire.NewPath
		} else if wire.OldPath != nil {
			path = *wire.OldPath
		}
		kind, oldPath := "update", ""
		if file.Status == "renamed" || strings.HasPrefix(file.Status, "R") {
			kind = "rename"
			if wire.OldPath != nil {
				oldPath = *wire.OldPath
			}
		}
		input.Files = append(input.Files, GitFile{Path: path, OldPath: oldPath, Kind: kind, Diff: wire.RawDiff})
		projected = append(projected, wire)
	}
	compared := compare(input)
	type resultKey struct {
		path  string
		block int
	}
	results := map[resultKey]Attribution{}
	for _, r := range compared.Results {
		results[resultKey{r.Path, r.Block}] = r
	}
	refs := map[string]string{}
	omissions := map[string][]string{}
	for i, file := range input.Files {
		wire := projected[i]
		current, e := comparisonBlocks(file.Kind, file.Diff, file.OldPath)
		if e != nil || len(current) != len(wire.ChangeBlocks) {
			return nil, failure("git.blocks", "unsupported_change_type")
		}
		target := planning.AttributedFileHistory{FileID: wire.FileID, RawDiff: wire.RawDiff, Blocks: []planning.AttributedBlockHistory{}}
		for index, block := range wire.ChangeBlocks {
			r, ok := results[resultKey{file.Path, index}]
			if !ok {
				return nil, failure("history.match", "invalid_schema")
			}
			b := planning.AttributedBlockHistory{BlockID: block.BlockID, DiffRange: block.DiffRange, Status: r.Status, Reasons: append([]string{}, r.Reasons...), Candidates: []planning.AgentHistoryCandidate{}}
			if h.Status == "unavailable" {
				b.Status = "error"
				b.Reasons = append([]string{}, out.Acquisition.Reasons...)
				target.Blocks = append(target.Blocks, b)
				continue
			}
			if !safePath(file.Path) || request.AllowPath == nil || !request.AllowPath(file.Path) {
				b.Status = "error"
				b.Reasons = []string{"history_unavailable"}
				b.Candidates = []planning.AgentHistoryCandidate{}
				target.Blocks = append(target.Blocks, b)
				continue
			}
			if h.Error != nil && h.Error.Reason == "history_parse_failed" {
				b.Status = "error"
				if !containsString(b.Reasons, "history_parse_failed") {
					b.Reasons = append(b.Reasons, "history_parse_failed")
				}
			}
			indexes := map[string]int{}
			for _, candidate := range r.Candidates {
				key := sourceTurnKey(candidate.Thread, candidate.Turn)
				turn, exists := contexts[key]
				if !exists {
					b.Status = "error"
					if !containsString(b.Reasons, "history_parse_failed") {
						b.Reasons = append(b.Reasons, "history_parse_failed")
					}
					continue
				}
				switch turn.Status {
				case "inProgress":
					if b.Status != "error" {
						b.Status = "ambiguous"
					}
					if !containsString(b.Reasons, "history_incomplete") {
						b.Reasons = append(b.Reasons, "history_incomplete")
					}
				case "completed", "interrupted", "failed":
				default:
					b.Status = "error"
					if !containsString(b.Reasons, "history_parse_failed") {
						b.Reasons = append(b.Reasons, "history_parse_failed")
					}
				}
				ref := refs[key]
				if ref == "" {
					ref = fmt.Sprintf("H%d", len(out.Contexts)+1)
					refs[key] = ref
					c := planning.AgentHistoryContext{ContextRef: ref, UserRequests: []string{}, FinalResponses: []string{}}
					requests, finals := turn.UserMessages, turn.FinalMessages
					if len(requests) == 0 && turn.Request != "" {
						requests = []ProjectionMessage{{Text: turn.Request}}
					}
					if len(finals) == 0 && turn.Final != "" {
						finals = []ProjectionMessage{{Text: turn.Final}}
					}
					for _, m := range requests {
						if safe(m.Text) {
							c.UserRequests = append(c.UserRequests, m.Text)
						} else {
							if !containsString(omissions[ref], "user_request_omitted") {
								omissions[ref] = append(omissions[ref], "user_request_omitted")
							}
						}
					}
					for _, m := range finals {
						if safe(m.Text) {
							c.FinalResponses = append(c.FinalResponses, m.Text)
						} else {
							if !containsString(omissions[ref], "final_response_omitted") {
								omissions[ref] = append(omissions[ref], "final_response_omitted")
							}
						}
					}
					out.Contexts = append(out.Contexts, c)
					if sources != nil {
						sources[ref] = ContextSource{ThreadID: candidate.Thread, TurnID: candidate.Turn}
					}
				}
				if sources != nil {
					source := sources[ref]
					edit := EditSource{candidate.Item, candidate.Index}
					seen := false
					for _, previous := range source.Edits {
						if previous == edit {
							seen = true
						}
					}
					if !seen {
						source.Edits = append(source.Edits, edit)
					}
					sources[ref] = source
				}
				match := "partial"
				if candidate.Exact {
					match = "exact"
				}
				op := ops[operationKey(Operation{Thread: candidate.Thread, Turn: candidate.Turn, Item: candidate.Item, Index: candidate.Index})]
				if op.Diff != nil {
					if blocks, e := comparisonBlocks(op.Kind, *op.Diff, op.OldPath); e == nil {
						exact, partial := false, false
						for _, source := range blocks {
							if equalBlock(current[index], source) {
								exact = true
							} else if overlap(current[index], source) {
								partial = true
							}
						}
						if exact && partial {
							match = "exact_and_partial"
						}
					}
				}
				if n, seen := indexes[ref]; seen {
					old := b.Candidates[n].Match
					if old != match {
						b.Candidates[n].Match = "exact_and_partial"
					}
				} else {
					indexes[ref] = len(b.Candidates)
					b.Candidates = append(b.Candidates, planning.AgentHistoryCandidate{ContextRef: ref, Match: match})
				}
				for _, reason := range omissions[ref] {
					if !containsString(b.OmissionReasons, reason) {
						b.OmissionReasons = append(b.OmissionReasons, reason)
					}
				}
			}
			// Projection may combine multiple edits into one context, never uniqueness.
			if b.Status == "matched" && (len(b.Candidates) != 1 || b.Candidates[0].Match != "exact") {
				b.Status = "ambiguous"
				if !containsString(b.Reasons, "partial_match") {
					b.Reasons = append(b.Reasons, "partial_match")
				}
			}
			target.Blocks = append(target.Blocks, b)
		}
		out.Files = append(out.Files, target)
	}
	return out, nil
}
