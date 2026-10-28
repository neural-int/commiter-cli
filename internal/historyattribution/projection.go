package historyattribution

import (
	"encoding/json"
	"strings"
)

// Research-only local projection, not an existing App Server request option.
// Git snapshot fields and transport measurements remain in the outer artifact.
type ProjectionMessage struct {
	ItemID    string `json:"item_id"`
	ItemOrder int    `json:"item_order"`
	Text      string `json:"text"`
}
type ProjectionChange struct {
	ChangeIndex int     `json:"change_index"`
	Path        string  `json:"path"`
	Kind        string  `json:"kind"`
	MovePath    *string `json:"move_path"`
	Diff        *string `json:"diff"`
}
type ProjectionEdit struct {
	ItemID    string             `json:"item_id"`
	ItemOrder int                `json:"item_order"`
	Status    string             `json:"status"`
	Changes   []ProjectionChange `json:"changes"`
}
type ProjectionTurn struct {
	TurnID        string              `json:"turn_id"`
	StartedAt     *int64              `json:"started_at"`
	Status        string              `json:"status"`
	UserMessages  []ProjectionMessage `json:"user_messages"`
	FileChanges   []ProjectionEdit    `json:"file_changes"`
	FinalMessages []ProjectionMessage `json:"final_messages"`
}
type ProjectionThread struct {
	ThreadID string           `json:"thread_id"`
	Turns    []ProjectionTurn `json:"turns"`
}
type ProjectionCoverage struct {
	Status  string        `json:"status"`
	Dropped int           `json:"dropped"`
	Reasons []string      `json:"reasons"`
	Errors  []*StageError `json:"errors"`
}
type HistoryProjection struct {
	SchemaVersion int                `json:"schema_version"`
	Projection    string             `json:"projection"`
	Threads       []ProjectionThread `json:"threads"`
	NextCursor    *string            `json:"next_cursor"`
	Coverage      ProjectionCoverage `json:"coverage"`
}

func projectHistory(h History) (HistoryProjection, error) {
	var identityError error
	h, identityError = uniqueHistorySources(h)
	if identityError != nil {
		return HistoryProjection{}, identityError
	}
	p := HistoryProjection{SchemaVersion: 2, Projection: "attribution", Threads: []ProjectionThread{}, Coverage: ProjectionCoverage{Status: h.Status, Dropped: h.Dropped, Reasons: []string{}, Errors: []*StageError{}}}
	if h.Status != "complete" {
		p.Coverage.Reasons = append(p.Coverage.Reasons, "history_incomplete")
	}
	if h.Error != nil {
		p.Coverage.Errors = append(p.Coverage.Errors, h.Error)
	}
	// Preserve encounter order; never manufacture source message identifiers for
	// old captures that did not save them.
	for _, context := range h.Turns {
		if context.StartedAt == nil || context.Status == "" || len(context.UserMessages) == 0 {
			return p, failure("input.schema", "projection_source_metadata_missing")
		}
		requestText := ""
		for _, message := range context.UserMessages {
			requestText += message.Text
		}
		finalText := ""
		for _, message := range context.FinalMessages {
			finalText = message.Text
		}
		if requestText != context.Request || finalText != context.Final {
			return p, failure("input.schema", "projection_context_mismatch")
		}
		var thread *ProjectionThread
		for i := range p.Threads {
			if p.Threads[i].ThreadID == context.Thread {
				thread = &p.Threads[i]
				break
			}
		}
		if thread == nil {
			p.Threads = append(p.Threads, ProjectionThread{ThreadID: context.Thread, Turns: []ProjectionTurn{}})
			thread = &p.Threads[len(p.Threads)-1]
		}
		turn := ProjectionTurn{TurnID: context.Turn, StartedAt: context.StartedAt, Status: context.Status, UserMessages: context.UserMessages, FinalMessages: context.FinalMessages, FileChanges: []ProjectionEdit{}}
		if turn.FinalMessages == nil {
			turn.FinalMessages = []ProjectionMessage{}
		}
		for _, op := range h.Operations {
			if op.Thread != context.Thread || op.Turn != context.Turn || !op.Completed {
				continue
			}
			var edit *ProjectionEdit
			for i := range turn.FileChanges {
				if turn.FileChanges[i].ItemID == op.Item {
					edit = &turn.FileChanges[i]
					break
				}
			}
			if edit == nil {
				turn.FileChanges = append(turn.FileChanges, ProjectionEdit{ItemID: op.Item, ItemOrder: op.ItemOrder, Status: "completed", Changes: []ProjectionChange{}})
				edit = &turn.FileChanges[len(turn.FileChanges)-1]
			}
			change := ProjectionChange{ChangeIndex: op.Index, Path: op.Path, Kind: op.Kind, Diff: op.Diff}
			if op.Kind == "rename" {
				change.Path = op.OldPath
				change.Kind = "update"
				move := op.Path
				change.MovePath = &move
			}
			edit.Changes = append(edit.Changes, change)
		}
		thread.Turns = append(thread.Turns, turn)
	}
	originalCount := 0
	for _, op := range h.Operations {
		if op.Completed {
			originalCount++
		}
	}
	projectedCount := 0
	for _, thread := range p.Threads {
		for _, turn := range thread.Turns {
			for _, edit := range turn.FileChanges {
				projectedCount += len(edit.Changes)
			}
		}
	}
	if originalCount != projectedCount {
		return p, failure("input.schema", "projection_orphan_operation")
	}
	return p, nil
}
func (p HistoryProjection) history() (History, error) {
	h := History{Status: p.Coverage.Status, Dropped: p.Coverage.Dropped}
	if p.SchemaVersion != 2 || p.Projection != "attribution" || p.NextCursor != nil {
		return h, failure("input.schema", "projection_schema_invalid")
	}
	if p.Coverage.Status != "complete" && p.Coverage.Status != "partial" && p.Coverage.Status != "unavailable" {
		return h, failure("input.schema", "invalid_history_status")
	}
	if len(p.Coverage.Errors) > 1 || (p.Coverage.Status == "complete" && (len(p.Coverage.Errors) > 0 || p.Coverage.Dropped > 0)) {
		return h, failure("input.schema", "inconsistent_history_status")
	}
	if len(p.Coverage.Errors) > 0 {
		h.Error = p.Coverage.Errors[0]
	}
	identities := map[string]bool{}
	for _, thread := range p.Threads {
		if thread.ThreadID == "" {
			return h, failure("input.schema", "missing_thread_identity")
		}
		for _, turn := range thread.Turns {
			id := thread.ThreadID + "/" + turn.TurnID
			if turn.TurnID == "" || turn.StartedAt == nil || identities[id] || !containsString([]string{"completed", "interrupted", "failed", "inProgress"}, turn.Status) {
				return h, failure("input.schema", "invalid_turn_metadata")
			}
			identities[id] = true
			for _, message := range append(append([]ProjectionMessage(nil), turn.UserMessages...), turn.FinalMessages...) {
				if strings.TrimSpace(message.ItemID) == "" || message.ItemOrder < 0 {
					return h, failure("input.schema", "invalid_message_identity")
				}
			}
			context := TurnContext{Thread: thread.ThreadID, Turn: turn.TurnID, StartedAt: turn.StartedAt, Status: turn.Status, UserMessages: turn.UserMessages, FinalMessages: turn.FinalMessages}
			for _, message := range turn.UserMessages {
				context.Request += message.Text
			}
			for _, message := range turn.FinalMessages {
				context.Final = message.Text
			}
			h.Turns = append(h.Turns, context)
			for _, edit := range turn.FileChanges {
				if edit.Status != "completed" || edit.ItemID == "" || edit.ItemOrder < 0 {
					return h, failure("input.schema", "invalid_edit_metadata")
				}
				for _, change := range edit.Changes {
					op := Operation{Thread: thread.ThreadID, Turn: turn.TurnID, Item: edit.ItemID, Index: change.ChangeIndex, ItemOrder: edit.ItemOrder, Completed: edit.Status == "completed", Path: change.Path, Kind: change.Kind, Diff: change.Diff}
					if change.MovePath != nil {
						op.Kind = "rename"
						op.OldPath = change.Path
						op.Path = *change.MovePath
					}
					h.Operations = append(h.Operations, op)
				}
			}
		}
	}
	return h, nil
}

type historyStatsContext struct{}
type ItemWireCount struct {
	Count     int   `json:"count"`
	JSONBytes int64 `json:"item_json_bytes"`
}
type HistoryWireStats struct {
	Items map[string]ItemWireCount `json:"items"`
}

func (s *HistoryWireStats) observe(result json.RawMessage) {
	var page struct {
		Data []struct {
			Item json.RawMessage `json:"item"`
		} `json:"data"`
	}
	if json.Unmarshal(result, &page) != nil {
		return
	}
	if s.Items == nil {
		s.Items = map[string]ItemWireCount{}
	}
	for _, entry := range page.Data {
		var tag struct {
			Type  string `json:"type"`
			Phase string `json:"phase"`
		}
		if json.Unmarshal(entry.Item, &tag) != nil {
			continue
		}
		category := "other"
		switch tag.Type {
		case "userMessage", "fileChange", "commandExecution", "mcpToolCall", "dynamicToolCall", "functionCallOutput", "reasoning", "plan", "webSearch", "imageView", "imageGeneration":
			category = tag.Type
		case "agentMessage":
			category = "intermediateMessage"
			if tag.Phase == "final_answer" {
				category = "finalMessage"
			}
		}
		count := s.Items[category]
		count.Count++
		count.JSONBytes += int64(len(entry.Item))
		s.Items[category] = count
	}
}

// The outer Git snapshot is unchanged. Acquisition diagnostics live in a
// separate metrics artifact, not inside the minimal history payload.
type ProjectionInput struct {
	Version      int               `json:"version"`
	Root         string            `json:"root"`
	Head         string            `json:"head"`
	IndexHash    string            `json:"index_hash"`
	SnapshotHash string            `json:"snapshot_hash"`
	SinceUnix    int64             `json:"since_unix"`
	UntilUnix    int64             `json:"until_unix"`
	Files        []GitFile         `json:"files"`
	History      HistoryProjection `json:"history"`
}

func projectionInput(in Input) (ProjectionInput, error) {
	h, err := projectHistory(in.History)
	return ProjectionInput{Version: 2, Root: in.Root, Head: in.Head, IndexHash: in.IndexHash, SnapshotHash: in.SnapshotHash, SinceUnix: in.SinceUnix, UntilUnix: in.UntilUnix, Files: in.Files, History: h}, err
}
func (p ProjectionInput) input() (Input, error) {
	h, err := p.History.history()
	if err != nil {
		return Input{}, err
	}
	if p.Version != 2 {
		return Input{}, failure("input.schema", "invalid_projection_version")
	}
	in := Input{Version: 1, Root: p.Root, Head: p.Head, IndexHash: p.IndexHash, SnapshotHash: p.SnapshotHash, SinceUnix: p.SinceUnix, UntilUnix: p.UntilUnix, Files: p.Files, History: h}
	if err = validateInput(in); err != nil {
		return Input{}, err
	}
	return in, nil
}
