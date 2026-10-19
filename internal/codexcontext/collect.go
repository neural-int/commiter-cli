package codexcontext

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Request struct {
	Root string
	// Candidate sessions may start in another worktree of this Git repository.
	// Their edits must still resolve inside Root.
	SessionRoots []string
	Start        time.Time
	End          time.Time
	Targets      []Target
	SafeText     func(string) bool
	AllowPath    func(string) bool
}

type page[T any] struct {
	Data []T     `json:"data"`
	Next *string `json:"nextCursor"`
}
type thread struct {
	ID      string `json:"id"`
	Cwd     string `json:"cwd"`
	Updated int64  `json:"updatedAt"`
}
type turn struct {
	ID      string `json:"id"`
	Started *int64 `json:"startedAt"`
}
type item struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Text    string `json:"text"`
	Phase   string `json:"phase"`
	Status  string `json:"status"`
	Changes []struct {
		Path string `json:"path"`
		Kind struct {
			Type     string  `json:"type"`
			MovePath *string `json:"move_path"`
		} `json:"kind"`
	} `json:"changes"`
}
type entry struct {
	TurnID string `json:"turnId"`
	Item   item   `json:"item"`
}
type observation struct {
	started int64
	record  Record
}

// Collect is optional evidence: all failures return a content-free status and
// never prevent the caller's ordinary Git-diff planning path.
func Collect(parent context.Context, request Request) Result {
	ctx, cancel := context.WithTimeout(parent, Timeout)
	defer cancel()
	if len(request.Targets) == 0 || request.Start.IsZero() || request.Start.After(request.End) {
		return Result{Status: "no_related_evidence"}
	}
	p, err := start(ctx, request.Root)
	if err != nil {
		return Result{Status: "unavailable"}
	}
	defer p.close()
	if p.initialize() != nil {
		return Result{Status: "unavailable"}
	}
	return collect(p, request)
}

func collect(client caller, request Request) Result {
	root, err := filepath.Abs(request.Root)
	if err != nil || request.SafeText == nil || request.AllowPath == nil {
		return Result{Status: "unavailable"}
	}
	root = filepath.Clean(root)
	roots := map[string]bool{root: true}
	for _, candidate := range request.SessionRoots {
		if filepath.IsAbs(candidate) {
			roots[filepath.Clean(candidate)] = true
		}
	}
	rootFilter := make([]string, 0, len(roots))
	for candidate := range roots {
		rootFilter = append(rootFilter, candidate)
	}
	sort.Strings(rootFilter)
	targets := map[string]bool{}
	for _, target := range request.Targets {
		path := localPath(root, target.Path)
		if path != "" && request.AllowPath(path) {
			targets[path] = true
		}
	}
	var observations []observation
	seenThreads := map[string]bool{}
	seenTurns := map[string]bool{}
	readThreads := false
	finish := func(partial bool) Result {
		result := selectRecords(observations)
		switch {
		case partial:
			result.Status = "partial"
		case len(result.Records) > 0:
			result.Status = "available"
		case !readThreads:
			result.Status = "absent"
		default:
			result.Status = "no_related_evidence"
		}
		if partial && !readThreads {
			result.Status = "unavailable"
		}
		return result
	}
	for _, archived := range []bool{false, true} {
		cursor := ""
		seenCursors := map[string]bool{}
		for {
			params := map[string]any{"cwd": rootFilter, "archived": archived, "limit": PageSize, "sortKey": "updated_at", "sortDirection": "desc", "useStateDbOnly": true, "sourceKinds": []string{"cli", "vscode", "exec", "appServer", "subAgentThreadSpawn"}}
			if cursor != "" {
				params["cursor"] = cursor
			}
			var threads page[thread]
			if client.call("thread/list", params, &threads) != nil || threads.Data == nil {
				return finish(true)
			}
			for _, th := range threads.Data {
				if th.ID == "" || seenThreads[th.ID] || !roots[filepath.Clean(th.Cwd)] {
					continue
				}
				seenThreads[th.ID] = true
				readThreads = true
				if th.Updated > 0 && th.Updated < request.Start.Unix() {
					continue
				}
				turnCursor := ""
				seenTurnCursors := map[string]bool{}
				for {
					args := map[string]any{"threadId": th.ID, "limit": PageSize, "sortDirection": "desc", "itemsView": "notLoaded"}
					if turnCursor != "" {
						args["cursor"] = turnCursor
					}
					var turns page[turn]
					if client.call("thread/turns/list", args, &turns) != nil || turns.Data == nil {
						return finish(true)
					}
					for _, tr := range turns.Data {
						if tr.ID == "" || seenTurns[tr.ID] || tr.Started == nil || *tr.Started < request.Start.Unix() || *tr.Started > request.End.Unix() {
							continue
						}
						obs, ok := readTurn(client, root, th.Cwd, th.ID, tr, targets, request)
						if !ok {
							return finish(true)
						}
						seenTurns[tr.ID] = true
						if obs != nil {
							observations = append(observations, *obs)
						}
					}
					if turns.Next == nil || *turns.Next == "" {
						break
					}
					turnCursor = *turns.Next
					if seenTurnCursors[turnCursor] {
						return finish(true)
					}
					seenTurnCursors[turnCursor] = true
				}
			}
			if threads.Next == nil || *threads.Next == "" {
				break
			}
			cursor = *threads.Next
			if seenCursors[cursor] {
				return finish(true)
			}
			seenCursors[cursor] = true
		}
	}
	return finish(false)
}

func readTurn(client caller, root, sessionCwd, threadID string, tr turn, targets map[string]bool, request Request) (*observation, bool) {
	obs := observation{started: *tr.Started}
	paths := map[string]bool{}
	seenItems := map[string]bool{}
	cursor := ""
	seenCursors := map[string]bool{}
	unsafe := false
	resolve := func(name string) string {
		if name == "" {
			return ""
		}
		if !filepath.IsAbs(name) {
			name = filepath.Join(sessionCwd, filepath.FromSlash(name))
		}
		return localPath(root, name)
	}
	for {
		params := map[string]any{"threadId": threadID, "turnId": tr.ID, "limit": PageSize, "sortDirection": "asc"}
		if cursor != "" {
			params["cursor"] = cursor
		}
		var items page[entry]
		if client.call("thread/items/list", params, &items) != nil || items.Data == nil {
			return nil, false
		}
		for _, en := range items.Data {
			it := en.Item
			if en.TurnID != tr.ID || it.ID == "" || seenItems[it.ID] {
				continue
			}
			seenItems[it.ID] = true
			switch it.Type {
			case "userMessage":
				for _, part := range it.Content {
					if part.Type == "text" {
						if obs.record.UserRequest != "" {
							obs.record.UserRequest += "\n"
						}
						obs.record.UserRequest += part.Text
					}
				}
			case "agentMessage":
				if it.Phase == "final_answer" {
					obs.record.FinalResponse = it.Text
				}
			case "fileChange":
				if it.Status != "completed" {
					continue
				}
				for _, change := range it.Changes {
					path := resolve(change.Path)
					if path == "" || !request.AllowPath(path) {
						unsafe = true
						continue
					}
					if change.Kind.MovePath != nil { // cross-path attribution is deliberately conservative
						move := resolve(*change.Kind.MovePath)
						if move == "" || !request.AllowPath(move) {
							unsafe = true
						}
						continue
					}
					if targets[path] {
						paths[path] = true
					}
				}
			}
		}
		if items.Next == nil || *items.Next == "" {
			break
		}
		cursor = *items.Next
		if seenCursors[cursor] {
			return nil, false
		}
		seenCursors[cursor] = true
	}
	if unsafe || len(paths) == 0 || strings.TrimSpace(obs.record.UserRequest) == "" || !request.SafeText(obs.record.UserRequest) {
		return nil, true
	}
	if !request.SafeText(obs.record.FinalResponse) {
		obs.record.FinalResponse = ""
	}
	for path := range paths {
		obs.record.Paths = append(obs.record.Paths, path)
	}
	sort.Strings(obs.record.Paths)
	return &obs, true
}

func selectRecords(observations []observation) Result {
	sort.SliceStable(observations, func(i, j int) bool { return observations[i].started > observations[j].started })
	var records []Record
	for _, obs := range observations {
		record := obs.record
		candidate := append(append([]Record(nil), records...), record)
		encoded, _ := json.Marshal(map[string]any{"codex_context": candidate})
		if len(encoded) > MaxContextBytes {
			record.FinalResponse = ""
			candidate = append(append([]Record(nil), records...), record)
			encoded, _ = json.Marshal(map[string]any{"codex_context": candidate})
		}
		if len(encoded) <= MaxContextBytes {
			records = candidate
		}
	}
	return Result{Records: records}
}
