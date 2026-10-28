package historyattribution

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os/exec"
	"strconv"
	"time"
)

type historyRPC struct {
	cmd           *exec.Cmd
	in            io.WriteCloser
	out           io.ReadCloser
	reader        *io.LimitedReader
	scanner       *bufio.Reader
	counter       *countedReader
	unlimited     bool
	maxMessage    int
	responseLimit int
	seq           int
	ctx           context.Context
	stop          func() bool
}

type unlimitedHistoryContext struct{}
type countedReader struct {
	source io.Reader
	bytes  int64
}

func (r *countedReader) Read(b []byte) (int, error) {
	n, e := r.source.Read(b)
	r.bytes += int64(n)
	return n, e
}
func unlimitedHistory(ctx context.Context) bool {
	b, _ := ctx.Value(unlimitedHistoryContext{}).(bool)
	return b
}
func openHistory(ctx context.Context, root string) (*historyRPC, error) {
	begin := time.Now()
	defer func() {
		if p := timing(ctx); p != nil {
			p.SpawnMS += elapsedMS(begin)
		}
	}()
	cmd := exec.CommandContext(ctx, "codex", "app-server", "--stdio", "-c", "analytics.enabled=false")
	cmd.Dir = root
	cmd.WaitDelay = 100 * time.Millisecond
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, failure("history.start", "stdin_failed")
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return nil, failure("history.start", "stdout_failed")
	}
	if cmd.Start() != nil {
		in.Close()
		out.Close()
		return nil, failure("history.start", "command_failed")
	}
	if g := guardFrom(ctx); g != nil {
		g.register(cmd.Process.Pid)
	}
	counter := &countedReader{source: out}
	reader := &io.LimitedReader{R: counter, N: receiveLimit + 1}
	var source io.Reader = reader
	if unlimitedHistory(ctx) {
		source = counter
	}
	scanner := bufio.NewReader(source)
	p := &historyRPC{cmd: cmd, in: in, out: out, reader: reader, scanner: scanner, counter: counter, unlimited: unlimitedHistory(ctx), ctx: ctx}
	if mode, _ := ctx.Value(historyFetchModeContext{}).(string); mode == "paged500" {
		p.responseLimit = researchResponseLimit
	}
	if limit, ok := ctx.Value(responseLimitContext{}).(int); ok {
		p.responseLimit = limit
	}
	p.stop = context.AfterFunc(ctx, func() { in.Close(); out.Close() })
	return p, nil
}
func (p *historyRPC) close() {
	begin := time.Now()
	defer func() {
		if profile := timing(p.ctx); profile != nil {
			profile.CleanupMS += elapsedMS(begin)
		}
	}()
	if p.stop != nil {
		p.stop()
	}
	p.in.Close()
	p.out.Close()
	_ = p.cmd.Process.Kill()
	_ = p.cmd.Wait()
	if g := guardFrom(p.ctx); g != nil {
		g.unregister(p.cmd.Process.Pid)
	}
}
func (p *historyRPC) call(stage, method string, params, result any) (callError error) {
	begin := time.Now()
	row := RPCTiming{Method: method, CallIndex: p.seq + 1}
	profile := timing(p.ctx)
	if profile != nil {
		row.StartNS = begin.UnixNano()
		raw, _ := json.Marshal(params)
		digest := sha256.Sum256(raw)
		row.RequestKey = hex.EncodeToString(digest[:])
	}
	defer func() {
		if profile != nil {
			row.TotalMS = elapsedMS(begin)
			row.EndNS = time.Now().UnixNano()
			row.Failed = callError != nil
			profile.Calls = append(profile.Calls, row)
		}
	}()
	sendBegin := time.Now()
	p.seq++
	if json.NewEncoder(p.in).Encode(map[string]any{"id": p.seq, "method": method, "params": params}) != nil {
		return failure(stage, "send_failed")
	}
	row.SendMS = elapsedMS(sendBegin)
	for {
		readBegin := time.Now()
		line, readErr := readLimitedResponse(p.scanner, p.responseLimit)
		row.WaitReadMS += elapsedMS(readBegin)
		row.ResponseBytes += len(line)
		if len(line) > p.maxMessage {
			p.maxMessage = len(line)
		}
		if !p.unlimited && p.reader.N <= 1 {
			return failure(stage, "receive_limit")
		}
		if readErr == errResponseLimit {
			return failure(stage, "response_size_limit")
		}
		if readErr != nil {
			break
		}
		var reply struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
			Method string          `json:"method"`
		}
		decodeBegin := time.Now()
		decodeError := json.Unmarshal(line, &reply)
		row.EnvelopeDecodeMS += elapsedMS(decodeBegin)
		if decodeError != nil {
			return failure(stage, "invalid_json")
		}
		if reply.Method != "" && len(reply.ID) > 0 {
			return failure(stage, "unexpected_server_request")
		}
		if string(reply.ID) != strconv.Itoa(p.seq) {
			continue
		}
		if len(reply.Error) > 0 && string(reply.Error) != "null" {
			return failure(stage, "rpc_error")
		}
		decodeBegin = time.Now()
		decodeError = json.Unmarshal(reply.Result, result)
		row.ResultDecodeMS += elapsedMS(decodeBegin)
		if len(reply.Result) == 0 || decodeError != nil {
			return failure(stage, "invalid_schema")
		}
		if stats, ok := p.ctx.Value(historyStatsContext{}).(*HistoryWireStats); ok && method == "thread/items/list" {
			stats.observe(reply.Result)
		}
		return nil
	}
	if p.ctx.Err() != nil {
		return failure(stage, "deadline_exceeded")
	}
	if !p.unlimited && p.reader.N <= 1 {
		return failure(stage, "receive_limit")
	}
	return failure(stage, "transport_closed_or_oversized_message")
}

type historyPage[T any] struct {
	Data []T     `json:"data"`
	Next *string `json:"nextCursor"`
}
type historyThread struct {
	ID  string `json:"id"`
	Cwd string `json:"cwd"`
}
type historyTurn struct {
	ID      string         `json:"id"`
	Started *int64         `json:"startedAt"`
	Status  string         `json:"status"`
	Items   *[]historyItem `json:"items"`
}
type historyItem struct {
	Command  string `json:"command"`
	Cwd      string `json:"cwd"`
	ExitCode *int   `json:"exitCode"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Phase    string `json:"phase"`
	Text     string `json:"text"`
	Content  []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Changes []struct {
		Path string  `json:"path"`
		Diff *string `json:"diff"`
		Kind struct {
			Type     string  `json:"type"`
			MovePath *string `json:"move_path"`
		} `json:"kind"`
	} `json:"changes"`
}
type historyEntry struct {
	TurnID string      `json:"turnId"`
	Item   historyItem `json:"item"`
}

func readHistory(ctx context.Context, root string, roots []string, targets map[string]bool, since, until int64) History {
	mode, ok := ctx.Value(historyFetchModeContext{}).(string)
	if !ok {
		mode = "paged20"
	}
	if parallelWorkers(mode) > 0 {
		return readHistoryParallel(ctx, root, targets, since, until)
	}
	config, err := historyFetchConfig(mode)
	if err != nil {
		return History{Status: "unavailable", Error: err.(*StageError)}
	}
	// Deliberately scoped to threads recorded in this exact worktree.
	roots = []string{root}
	started := time.Now()
	defer func() {
		if p := timing(ctx); p != nil {
			p.TotalMS = elapsedMS(started)
		}
	}()
	h := History{Status: "complete", Unlimited: unlimitedHistory(ctx), FetchMode: mode}
	p, err := openHistory(ctx, root)
	if err != nil {
		h.Status = "unavailable"
		h.Error = err.(*StageError)
		return h
	}
	defer p.close()
	finish := func(err error) History {
		h.ReceivedBytes = p.counter.bytes
		h.Requests = p.seq
		h.MaxMessageBytes = p.maxMessage
		h.ElapsedMS = float64(time.Since(started).Microseconds()) / 1000
		retained, _ := json.Marshal(struct {
			Operations []Operation
			Turns      []TurnContext
		}{h.Operations, h.Turns})
		h.RetainedBytes = len(retained)
		if err != nil {
			h.Status = "partial"
			if len(h.Turns) == 0 {
				h.Status = "unavailable"
			}
			h.Error = err.(*StageError)
		}
		return h
	}
	var init map[string]json.RawMessage
	if err = p.call("history.initialize", "initialize", map[string]any{"clientInfo": map[string]string{"name": "commiter-attribution-poc", "version": "1"}, "capabilities": map[string]bool{"experimentalApi": true}}, &init); err != nil {
		return finish(err)
	}
	if json.NewEncoder(p.in).Encode(map[string]string{"method": "initialized"}) != nil {
		return finish(failure("history.initialize", "notification_failed"))
	}
	seenThreads := map[string]bool{}
	seenTurns := map[string]string{}
	seenOps := map[string]bool{}
	for _, archived := range []bool{false, true} {
		cursor := ""
		cursors := map[string]bool{}
		for {
			args := map[string]any{"cwd": roots, "archived": archived, "limit": config.threadPage, "sortKey": "updated_at", "sortDirection": "desc", "useStateDbOnly": true}
			if cursor != "" {
				args["cursor"] = cursor
			}
			var threads historyPage[historyThread]
			if err = p.call("history.threads", "thread/list", args, &threads); err != nil {
				return finish(err)
			}
			if threads.Data == nil {
				return finish(failure("history.threads", "invalid_schema"))
			}
			for _, th := range threads.Data {
				if th.ID == "" || th.Cwd == "" {
					return finish(failure("history.threads", "invalid_schema"))
				}
				if seenThreads[th.ID] || !containsString(roots, th.Cwd) {
					continue
				}
				seenThreads[th.ID] = true
				h.ThreadsScanned++
				tc := ""
				turnCursors := map[string]bool{}
				for {
					ta := map[string]any{"threadId": th.ID, "limit": config.turnPage, "sortDirection": "desc", "itemsView": config.itemsView}
					if tc != "" {
						ta["cursor"] = tc
					}
					var turns historyPage[historyTurn]
					if err = p.call("history.turns", "thread/turns/list", ta, &turns); err != nil {
						return finish(err)
					}
					if turns.Data == nil {
						return finish(failure("history.turns", "invalid_schema"))
					}
					for _, turn := range turns.Data {
						if turn.ID == "" || turn.Started == nil {
							return finish(failure("history.turns", "invalid_schema"))
						}
						if *turn.Started < since || *turn.Started > until {
							continue
						}
						turnKey, _ := json.Marshal([]string{th.ID, turn.ID})
						fingerprint, _ := json.Marshal(turn)
						if previous, exists := seenTurns[string(turnKey)]; exists {
							if previous != string(fingerprint) {
								// Invalidate evidence already collected for this identity.
								keptOps := h.Operations[:0]
								for _, op := range h.Operations {
									if op.Thread != th.ID || op.Turn != turn.ID {
										keptOps = append(keptOps, op)
									}
								}
								h.Operations = keptOps
								keptTurns := h.Turns[:0]
								for _, record := range h.Turns {
									if record.Thread != th.ID || record.Turn != turn.ID {
										keptTurns = append(keptTurns, record)
									}
								}
								h.Turns = keptTurns
								return finish(failure("history.turns", "conflicting_duplicate_turn"))
							}
							h.DuplicateTurns++
							continue
						}
						seenTurns[string(turnKey)] = string(fingerprint)
						if collector, ok := ctx.Value(turnCollectContext{}).(*turnCollector); ok {
							job := turnJob{thread: th, turn: turn}
							if collector.emit != nil {
								if err := collector.emit(job); err != nil {
									return finish(err)
								}
							} else {
								collector.jobs = append(collector.jobs, job)
							}
							h.TurnsScanned++
							continue
						}
						if err := readHistoryTurn(ctx, p, root, th, turn, targets, config, &h, seenOps); err != nil {
							return finish(err)
						}
					}
					if turns.Next == nil || *turns.Next == "" {
						break
					}
					if turnCursors[*turns.Next] {
						return finish(failure("history.turns", "repeated_cursor"))
					}
					tc = *turns.Next
					turnCursors[tc] = true
				}
			}
			if threads.Next == nil || *threads.Next == "" {
				break
			}
			if cursors[*threads.Next] {
				return finish(failure("history.threads", "repeated_cursor"))
			}
			cursor = *threads.Next
			cursors[cursor] = true
		}
	}
	if h.Dropped > 0 {
		h.Status = "partial"
		h.Error = &StageError{Stage: "history.filter", Code: "sensitive_evidence_omitted", Category: "ambiguous", Reason: "history_incomplete"}
	}
	return finish(nil)
}

func readHistoryTurn(ctx context.Context, p *historyRPC, root string, th historyThread, turn historyTurn, targets map[string]bool, config fetchConfig, h *History, seenOps map[string]bool) error {
	itemOrder := 0
	var err error
	h.TurnsScanned++
	record := TurnContext{Thread: th.ID, Turn: turn.ID, StartedAt: turn.Started, Status: turn.Status}
	turnOps := []Operation{}
	ic := ""
	itemCursors := map[string]bool{}
	seenItems := map[string]string{}
	for {
		ia := map[string]any{"threadId": th.ID, "turnId": turn.ID, "limit": config.itemPage, "sortDirection": "asc"}
		if ic != "" {
			ia["cursor"] = ic
		}
		var items historyPage[historyEntry]
		if config.itemsView == "full" {
			if turn.Items == nil {
				return failure("history.turns", "invalid_schema")
			}
			items.Data = []historyEntry{}
			for _, item := range *turn.Items {
				items.Data = append(items.Data, historyEntry{TurnID: turn.ID, Item: item})
			}
		} else if err = p.call("history.items", "thread/items/list", ia, &items); err != nil {
			return err
		}
		if items.Data == nil {
			return failure("history.items", "invalid_schema")
		}
		selectionDone := selectionTimer(ctx, "item")
		for _, entry := range items.Data {
			h.ItemsScanned++
			it := entry.Item
			if entry.TurnID != turn.ID || it.ID == "" {
				selectionDone()
				return failure("history.items", "invalid_item_identity")
			}
			fingerprint, _ := json.Marshal(it)
			if previous, exists := seenItems[it.ID]; exists {
				if previous != string(fingerprint) {
					selectionDone()
					return failure("history.items", "conflicting_duplicate_item")
				}
				h.DuplicateItems++
				continue
			}
			seenItems[it.ID] = string(fingerprint)
			itemOrder++
			switch it.Type {
			case "userMessage":
				messageText := ""
				for _, part := range it.Content {
					if part.Type == "text" {
						record.Request += part.Text + "\n"
						messageText += part.Text + "\n"
					}
				}
				if messageText != "" {
					record.UserMessages = append(record.UserMessages, ProjectionMessage{ItemID: it.ID, ItemOrder: itemOrder, Text: messageText})
				}
			case "agentMessage":
				if it.Phase == "final_answer" {
					record.Final = it.Text
					record.FinalMessages = []ProjectionMessage{{ItemID: it.ID, ItemOrder: itemOrder, Text: it.Text}}
				}
			case "commandExecution":
				if enabled, _ := ctx.Value(commandHistoryContext{}).(bool); enabled && safeCommandText(it.Command, root, targets) {
					paths := mentionedPaths(root, it.Cwd, it.Command, targets)
					if len(paths) > 0 {
						record.Commands = append(record.Commands, CommandExecution{ItemID: it.ID, ItemOrder: itemOrder, Command: it.Command, Cwd: it.Cwd, Status: it.Status, ExitCode: it.ExitCode, MentionedPaths: paths})
					}
				}
			case "fileChange":
				if it.Status != "completed" {
					continue
				}
				for index, ch := range it.Changes {
					pathDone := selectionTimer(ctx, "path")
					path := normalizeHistoryPath(root, th.Cwd, ch.Path)
					oldPath := ""
					kind := ch.Kind.Type
					if ch.Kind.MovePath != nil {
						oldPath = path
						path = normalizeHistoryPath(root, th.Cwd, *ch.Kind.MovePath)
						kind = "rename"
					}
					pathDone()
					if path == "" || (kind == "rename" && oldPath == "") {
						// Outside the selected worktree is out of scope,
						// not missing evidence for a selected safe path.
						continue
					}
					if !targets[path] && !targets[oldPath] {
						continue
					}
					if ch.Diff != nil && !timedSafeText(ctx, *ch.Diff) {
						h.Dropped++
						continue
					}
					id := operationKey(Operation{Thread: th.ID, Turn: turn.ID, Item: it.ID, Index: index})
					if seenOps[id] {
						continue
					}
					seenOps[id] = true
					turnOps = append(turnOps, Operation{Thread: th.ID, Turn: turn.ID, Item: it.ID, Index: index, Path: path, Kind: kind, Diff: ch.Diff, Completed: true, OldPath: oldPath, ItemOrder: itemOrder})
				}
			}
		}
		selectionDone()
		if items.Next == nil || *items.Next == "" {
			break
		}
		if itemCursors[*items.Next] {
			return failure("history.items", "repeated_cursor")
		}
		ic = *items.Next
		itemCursors[ic] = true
	}
	contextDone := selectionTimer(ctx, "context")
	if record.Request == "" || !timedSafeText(ctx, record.Request) {
		h.Dropped += len(turnOps)
		contextDone()
		return nil
	}
	if (len(turnOps) > 0 || len(record.Commands) > 0) && !timedSafeText(ctx, record.Final) {
		record.Final = ""
		record.FinalMessages = nil
		h.Dropped++
	}
	if len(turnOps) > 0 || len(record.Commands) > 0 {
		h.Operations = append(h.Operations, turnOps...)
		h.Turns = append(h.Turns, record)
	}
	contextDone()
	return nil
}
