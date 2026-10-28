// Conservative content matching shared by the CLI ETL and research command.
package historyattribution

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const maxInputBytes = 8 * 1024 * 1024

type StageError struct {
	Cause    *StageError `json:"cause,omitempty"`
	Stage    string      `json:"stage"`
	Code     string      `json:"code"`
	Category string      `json:"category"`
	Reason   string      `json:"reason"`
}

func (e *StageError) Error() string { return e.Stage + ":" + e.Code }
func failure(stage, code string) error {
	reason := "git_snapshot_unavailable"
	category := "error"
	switch {
	case code == "snapshot_changed":
		reason = "snapshot_changed"
	case stage == "history.filter":
		reason = "history_incomplete"
		category = "ambiguous"
	case strings.HasPrefix(stage, "history"):
		reason = "history_unavailable"
		if code == "invalid_json" || code == "invalid_schema" || code == "invalid_item_identity" || code == "repeated_cursor" || code == "conflicting_duplicate_turn" || code == "conflicting_duplicate_item" {
			reason = "history_parse_failed"
		}
	case code == "unsupported_change_type":
		reason = code
	case strings.HasPrefix(stage, "input"):
		reason = "input_invalid"
	case strings.HasPrefix(stage, "output"):
		reason = "artifact_unavailable"
	case stage == "arguments":
		reason = "invalid_arguments"
	}
	return &StageError{Stage: stage, Code: code, Category: category, Reason: reason}
}

type GitFile struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Diff    string `json:"diff"`
	OldPath string `json:"old_path,omitempty"`
}
type Operation struct {
	Thread    string  `json:"thread"`
	Turn      string  `json:"turn"`
	Item      string  `json:"item"`
	Index     int     `json:"index"`
	Path      string  `json:"path"`
	Kind      string  `json:"kind"`
	Diff      *string `json:"diff"`
	Completed bool    `json:"completed"`
	OldPath   string  `json:"old_path,omitempty"`
	ItemOrder int     `json:"item_order,omitempty"`
}
type CommandExecution struct {
	ItemID         string   `json:"item_id"`
	ItemOrder      int      `json:"item_order"`
	Command        string   `json:"command"`
	Cwd            string   `json:"cwd"`
	Status         string   `json:"status"`
	ExitCode       *int     `json:"exit_code,omitempty"`
	MentionedPaths []string `json:"mentioned_paths"`
}
type TurnContext struct {
	Commands      []CommandExecution  `json:"commands,omitempty"`
	Thread        string              `json:"thread"`
	Turn          string              `json:"turn"`
	Request       string              `json:"user_request"`
	Final         string              `json:"final_response,omitempty"`
	StartedAt     *int64              `json:"started_at,omitempty"`
	Status        string              `json:"status,omitempty"`
	UserMessages  []ProjectionMessage `json:"user_messages,omitempty"`
	FinalMessages []ProjectionMessage `json:"final_messages,omitempty"`
}
type History struct {
	Waits           *PipelineWaits    `json:"pipeline_waits,omitempty"`
	Memory          *MemoryProtection `json:"memory_protection,omitempty"`
	DuplicateTurns  int               `json:"duplicate_turns,omitempty"`
	DuplicateItems  int               `json:"duplicate_items,omitempty"`
	FetchMode       string            `json:"fetch_mode,omitempty"`
	ThreadsScanned  int               `json:"threads_scanned"`
	TurnsScanned    int               `json:"turns_scanned"`
	ItemsScanned    int               `json:"items_scanned"`
	Status          string            `json:"status"`
	Error           *StageError       `json:"error,omitempty"`
	ReceivedBytes   int64             `json:"received_bytes"`
	Operations      []Operation       `json:"operations"`
	Turns           []TurnContext     `json:"turns"`
	Dropped         int               `json:"dropped"`
	Unlimited       bool              `json:"unlimited_acquisition"`
	ElapsedMS       float64           `json:"elapsed_ms"`
	Requests        int               `json:"rpc_requests"`
	MaxMessageBytes int               `json:"max_message_bytes"`
	RetainedBytes   int               `json:"retained_json_bytes"`
}
type Input struct {
	SinceUnix    int64     `json:"since_unix"`
	UntilUnix    int64     `json:"until_unix"`
	Version      int       `json:"version"`
	Root         string    `json:"root"`
	Head         string    `json:"head"`
	IndexHash    string    `json:"index_hash"`
	SnapshotHash string    `json:"snapshot_hash"`
	Files        []GitFile `json:"files"`
	History      History   `json:"history"`
	CaptureMS    float64   `json:"capture_ms"`
}
type EditLine struct {
	Direction string `json:"direction"`
	Text      string `json:"text"`
}
type Block struct {
	Lines []EditLine `json:"lines"`
}
type Candidate struct {
	Thread string `json:"thread"`
	Turn   string `json:"turn"`
	Item   string `json:"item"`
	Index  int    `json:"index"`
	Exact  bool   `json:"exact"`
}
type Attribution struct {
	Path         string      `json:"path"`
	Block        int         `json:"block"`
	Status       string      `json:"status"`
	Reason       string      `json:"reason"`
	Reasons      []string    `json:"reasons"`
	Candidates   []Candidate `json:"candidates"`
	PathBaseline []Candidate `json:"path_baseline"`
	SingleTurn   bool        `json:"single_turn"`
}
type Comparison struct {
	Results       []Attribution `json:"results"`
	HistoryStatus string        `json:"history_status"`
	Fallback      bool          `json:"git_diff_only_fallback"`
	Errors        []*StageError `json:"errors,omitempty"`
	Limits        string        `json:"limits"`
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(?:.*)$`)

// Parse validates hunk lengths. Context separates blocks; line numbers do not
// establish correspondence. Whitespace, ordering, duplicate lines and EOF marks remain.
func parseBlocks(kind, diff string) ([]Block, error) {
	if kind == "add" || kind == "delete" {
		if diff == "" {
			return nil, nil
		}
		direction := "+"
		if kind == "delete" {
			direction = "-"
		}
		var lines []EditLine
		parts := strings.Split(diff, "\n")
		end := len(parts)
		if strings.HasSuffix(diff, "\n") {
			end--
		}
		for _, s := range parts[:end] {
			lines = append(lines, EditLine{direction, s})
		}
		if !strings.HasSuffix(diff, "\n") {
			lines = append(lines, EditLine{"eof", direction})
		}
		return []Block{{lines}}, nil
	}
	if kind != "update" {
		return nil, fmt.Errorf("unsupported_kind")
	}
	var blocks []Block
	var active []EditLine
	oldLeft, newLeft := 0, 0
	inHunk := false
	lastDirection := ""
	flush := func() {
		if len(active) > 0 {
			blocks = append(blocks, Block{active})
			active = nil
		}
	}
	for _, line := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
		if inHunk && strings.HasPrefix(line, "diff --git ") {
			if oldLeft != 0 || newLeft != 0 {
				return nil, fmt.Errorf("incomplete_hunk")
			}
			flush()
			inHunk = false
		}
		if m := hunkHeader.FindStringSubmatch(line); m != nil {
			if inHunk && (oldLeft != 0 || newLeft != 0) {
				return nil, fmt.Errorf("incomplete_hunk")
			}
			flush()
			inHunk = true
			oldLeft, newLeft = 1, 1
			if m[2] != "" {
				oldLeft, _ = strconv.Atoi(m[2])
			}
			if m[4] != "" {
				newLeft, _ = strconv.Atoi(m[4])
			}
			lastDirection = ""
			continue
		}
		if !inHunk {
			if strings.HasPrefix(line, "diff --git ") || strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "new file mode ") || strings.HasPrefix(line, "deleted file mode ") || strings.HasPrefix(line, "similarity index ") || strings.HasPrefix(line, "rename from ") || strings.HasPrefix(line, "rename to ") || strings.HasPrefix(line, "old mode ") || strings.HasPrefix(line, "new mode ") {
				continue
			}
			if line == "" {
				continue
			}
			return nil, fmt.Errorf("unsupported_diff")
		}
		if line == `\ No newline at end of file` {
			if lastDirection == "" {
				return nil, fmt.Errorf("invalid_eof")
			}
			if lastDirection != " " {
				active = append(active, EditLine{"eof", lastDirection})
			}
			continue
		}
		if line == "" {
			return nil, fmt.Errorf("malformed_line")
		}
		switch line[0] {
		case ' ':
			flush()
			oldLeft--
			newLeft--
			lastDirection = " "
		case '-':
			active = append(active, EditLine{"-", line[1:]})
			oldLeft--
			lastDirection = "-"
		case '+':
			active = append(active, EditLine{"+", line[1:]})
			newLeft--
			lastDirection = "+"
		default:
			return nil, fmt.Errorf("malformed_line")
		}
		if oldLeft < 0 || newLeft < 0 {
			return nil, fmt.Errorf("hunk_count_mismatch")
		}
	}
	if !inHunk || oldLeft != 0 || newLeft != 0 {
		return nil, fmt.Errorf("incomplete_hunk")
	}
	flush()
	return blocks, nil
}
func equalBlock(a, b Block) bool {
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(aa) == string(bb)
}
func overlap(a, b Block) bool {
	for _, x := range a.Lines {
		if x.Direction == "eof" {
			continue
		}
		for _, y := range b.Lines {
			if x == y {
				return true
			}
		}
	}
	return false
}

// Rename uses explicit old/new path evidence. Empty diff is a pure move only
// when both endpoints are known; no syntax parser or fuzzy text normalization.
func comparisonBlocks(kind, diff, oldPath string) ([]Block, error) {
	if kind == "rename" {
		if oldPath == "" {
			return nil, fmt.Errorf("rename_endpoints_missing")
		}
		if strings.TrimSpace(diff) == "" {
			return []Block{{}}, nil
		}
		bs, err := parseBlocks("update", diff)
		if err == nil {
			return bs, nil
		}
		// Git's 100% rename metadata provides a content-preserving move without hunks.
		if strings.Contains(diff, "similarity index 100%\n") && !strings.Contains(diff, "@@") && !strings.Contains(diff, "Binary files") {
			for _, line := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
				if line != "" && !strings.HasPrefix(line, "diff --git ") && !strings.HasPrefix(line, "similarity index 100%") && !strings.HasPrefix(line, "rename from ") && !strings.HasPrefix(line, "rename to ") {
					return nil, err
				}
			}
			return []Block{{}}, nil
		}
		return nil, err
	}
	return parseBlocks(kind, diff)
}
func addReason(r *Attribution, reason string) {
	if !containsString(r.Reasons, reason) {
		r.Reasons = append(r.Reasons, reason)
	}
}
func blockSide(b Block, direction string) string {
	var lines []string
	for _, line := range b.Lines {
		if line.Direction == direction {
			lines = append(lines, line.Text)
		}
		if line.Direction == "eof" && line.Text == direction {
			lines = append(lines, "\x00EOF")
		}
	}
	if len(lines) == 0 {
		return ""
	}
	v, _ := json.Marshal(lines)
	return string(v)
}

// A -> B followed by B -> C, with explicit item order inside one Turn.
// This labels the observed rewriting chain; it does not attribute A -> C.
func rewrittenByOrderedPair(current, first, last Block) bool {
	old, new := blockSide(current, "-"), blockSide(current, "+")
	middle := blockSide(first, "+")
	return old != "" && new != "" && middle != "" && old == blockSide(first, "-") && middle == blockSide(last, "-") && new == blockSide(last, "+")
}
func compare(in Input) Comparison {
	out := Comparison{HistoryStatus: in.History.Status, Fallback: in.History.Status != "complete", Limits: "unique content correspondence, not causal provenance; no patch replay; superseded/split-merge causes are not inferred from mismatch"}
	if e := in.History.Error; e != nil {
		if e.Reason == "" {
			e = failure(e.Stage, e.Code).(*StageError)
		}
		if e.Category == "error" {
			out.Errors = append(out.Errors, e)
		}
	}

	type parsed struct {
		op     Operation
		blocks []Block
		err    error
	}
	var ops []parsed
	seen := map[string]bool{}
	for _, op := range in.History.Operations {
		if !op.Completed || seen[operationKey(op)] {
			continue
		}
		seen[operationKey(op)] = true
		var bs []Block
		var err error
		if op.Diff == nil {
			err = fmt.Errorf("missing_diff")
		} else {
			bs, err = comparisonBlocks(op.Kind, *op.Diff, op.OldPath)
		}
		ops = append(ops, parsed{op, bs, err})
	}
	for _, file := range in.Files {
		kind := "update"
		if file.Kind == "untracked" {
			kind = "add"
		}
		if file.Kind == "rename" {
			kind = "rename"
		}
		bs, err := comparisonBlocks(kind, file.Diff, file.OldPath)
		if err != nil || len(bs) == 0 {
			r := Attribution{Path: file.Path, Block: -1, Status: "error", Reason: "unsupported_change_type", Reasons: []string{"unsupported_change_type"}}
			if file.Kind == "rename" && !strings.Contains(file.Diff, "Binary files ") && !strings.Contains(file.Diff, "GIT binary patch") {
				r.Status = "ambiguous"
				r.Reason = "rename_unresolved"
				r.Reasons = []string{r.Reason}
			}
			out.Results = append(out.Results, r)
			out.Fallback = true
			continue
		}
		for n, b := range bs {
			r := Attribution{Path: file.Path, Block: n, Status: "unmatched", Reason: "no_content_evidence", Reasons: []string{}}
			exactCount := 0
			unknown := false
			for _, entry := range ops {
				op := entry.op
				relevant := op.Path == file.Path || (file.OldPath != "" && (op.Path == file.OldPath || op.OldPath == file.OldPath)) || (op.OldPath != "" && op.OldPath == file.Path)
				if !relevant {
					continue
				}
				c := Candidate{op.Thread, op.Turn, op.Item, op.Index, false}
				r.PathBaseline = append(r.PathBaseline, c)
				if entry.err != nil {
					unknown = true
					if file.Kind == "rename" {
						addReason(&r, "rename_unresolved")
					} else {
						addReason(&r, "history_parse_failed")
					}
					continue
				}
				if file.Kind == "rename" && (op.Kind != "rename" || op.Path != file.Path || op.OldPath != file.OldPath) {
					reason := "rename_unresolved"
					if op.Kind == "rename" && op.OldPath == file.OldPath && op.Path != file.Path {
						mappings := map[string]bool{}
						for _, candidate := range ops {
							if candidate.err == nil && candidate.op.Kind == "rename" && candidate.op.OldPath == file.OldPath {
								mappings[candidate.op.Path] = true
							}
						}
						if len(mappings) > 1 {
							reason = "rename_multiple_candidates"
						}
					}
					addReason(&r, reason)
					continue
				}
				if file.Kind != "rename" && op.Kind == "rename" {
					addReason(&r, "rename_unresolved")
					continue
				}
				exact, partial, positions := false, false, 0
				for _, eb := range entry.blocks {
					if equalBlock(b, eb) {
						exact = true
						positions++
					} else if overlap(b, eb) {
						partial = true
					}
				}
				if exact || partial {
					c.Exact = exact
					r.Candidates = append(r.Candidates, c)
				}
				if exact {
					exactCount++
				}
				if partial && !exact {
					addReason(&r, "partial_match")
				}
				if positions > 1 {
					addReason(&r, "overlapping_candidates")
				}
			}
			if exactCount == 0 {
				for _, first := range ops {
					for _, last := range ops {
						if first.op.Path != file.Path || last.op.Path != file.Path || first.op.Thread != last.op.Thread || first.op.Turn != last.op.Turn || first.op.ItemOrder <= 0 || first.op.ItemOrder >= last.op.ItemOrder {
							continue
						}
						for _, a := range first.blocks {
							for _, z := range last.blocks {
								if rewrittenByOrderedPair(b, a, z) {
									addReason(&r, "history_superseded_or_rewritten")
								}
							}
						}
					}
				}
			}
			if exactCount > 1 {
				reason := "multiple_history_candidates"
				if file.Kind == "rename" {
					reason = "rename_multiple_candidates"
				}
				addReason(&r, reason)
			}
			repeats := 0
			for _, other := range bs {
				if equalBlock(b, other) {
					repeats++
				}
			}
			if repeats > 1 {
				addReason(&r, "overlapping_candidates")
			}
			if file.Kind == "rename" && exactCount == 0 && len(r.Reasons) == 0 {
				addReason(&r, "rename_unresolved")
			}
			turns := map[string]bool{}
			for _, c := range r.Candidates {
				turns[c.Thread+"/"+c.Turn] = true
			}
			r.SingleTurn = len(turns) == 1
			if in.History.Status != "complete" {
				addReason(&r, "history_incomplete")
			}
			if len(r.Reasons) > 0 {
				r.Status = "ambiguous"
				r.Reason = r.Reasons[0]
				if unknown && file.Kind != "rename" {
					r.Status = "error"
					r.Reason = "history_parse_failed"
				}
			} else if len(r.Candidates) == 1 && exactCount == 1 {
				r.Status = "matched"
				r.Reason = "unique_exact_block"
				if file.Kind == "rename" {
					r.Reason = "unique_exact_rename"
				}
			}
			out.Results = append(out.Results, r)
		}
	}
	markSplitMergeEvidence(in, &out)

	for _, r := range out.Results {
		if r.Status == "error" {
			out.Fallback = true
		}
	}
	return out
}

func operationKey(op Operation) string {
	b, _ := json.Marshal([]any{op.Thread, op.Turn, op.Item, op.Index})
	return string(b)
}
