package planning

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

func attributedFixture(t *testing.T) (*GroupingHistory, contextinput.Prepared) {
	t.Helper()
	p := candidatePrepared(t)
	h := &GroupingHistory{Contexts: []AgentHistoryContext{{"H1", []string{"Change retry behavior"}, []string{"Changed retry behavior"}}}, Acquisition: AgentHistoryAcquisition{Status: "complete"}, Files: []AttributedFileHistory{{FileID: p.Document.Files[0].ID, RawDiff: p.Document.Files[0].GroupingRawDiff, Blocks: []AttributedBlockHistory{{BlockID: "B1", DiffRange: DiffRange{2, 3}, Status: "matched", Candidates: []AgentHistoryCandidate{{"H1", "exact"}}}}}}}
	return h, p
}

func TestAttributedGroupingReachesActualRequestAndPreservesHost(t *testing.T) {
	h, p := attributedFixture(t)
	before, _ := json.Marshal(h)
	c := &candidateClient{responses: candidateResponses()}
	result, err := (ThreePhaseGenerator{Client: c, GroupingHistory: h}).Generate(context.Background(), p, English, SensitiveValues{})
	if err != nil || result.Calls != 3 {
		t.Fatalf("%+v %v", result, err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(c.messages[0][1].Content), &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "task", "agent_history_contexts", "files", "agent_history_acquisition", "agent_history_projection"} {
		if payload[key] == nil {
			t.Fatal("missing " + key)
		}
	}
	if payload["codex_context"] != nil {
		t.Fatal("old history duplicated")
	}
	var files []groupingFile
	json.Unmarshal(payload["files"], &files)
	block := files[1].ChangeBlocks[0]
	if block.AgentHistoryMatchStatus != "matched" || block.AgentHistoryCandidates[0].ContextRef != "H1" || block.DiffRange != (DiffRange{2, 3}) {
		t.Fatalf("bad block: %+v", block)
	}
	if strings.Contains(string(payload["files"]), `"id"`) {
		t.Fatal("legacy file ID key")
	}
	if len(block.AgentHistoryOmissionReasons) != 0 || strings.Contains(string(payload["files"]), "agent_history_omission_reasons") {
		t.Fatal("empty optional omissions sent")
	}
	after, _ := json.Marshal(h)
	if string(before) != string(after) {
		t.Fatal("host attribution mutated")
	}
	for i := 1; i < 3; i++ {
		if !strings.Contains(c.messages[i][1].Content, `"context_ref":"H1"`) {
			t.Fatal("attributed history missing from metadata phase")
		}
	}
	if !reflect.DeepEqual(result.Plan.Commits[0].FileIDs, []string{"F002", "F001"}) {
		t.Fatal("file mapping lost")
	}
}

func TestAttributedGroupingRejectsStaleOrInvalidEvidenceBeforeCall(t *testing.T) {
	for _, name := range []string{"diff", "range", "reference", "duplicate context", "duplicate block", "unknown file", "invalid match", "false unique", "missing coverage"} {
		t.Run(name, func(t *testing.T) {
			h, p := attributedFixture(t)
			switch name {
			case "diff":
				h.Files[0].RawDiff += " "
			case "range":
				h.Files[0].Blocks[0].DiffRange.StartLine++
			case "reference":
				h.Files[0].Blocks[0].Candidates[0].ContextRef = "missing"
			case "duplicate context":
				h.Contexts = append(h.Contexts, h.Contexts[0])
			case "duplicate block":
				h.Files[0].Blocks = append(h.Files[0].Blocks, h.Files[0].Blocks[0])
			case "unknown file":
				h.Files[0].FileID = "unknown"
			case "invalid match":
				h.Files[0].Blocks[0].Candidates[0].Match = "similar"
			case "false unique":
				h.Acquisition.Status = "partial"
			case "missing coverage":
				h.Files = nil
			}
			c := &candidateClient{responses: candidateResponses()}
			_, err := (ThreePhaseGenerator{Client: c, GroupingHistory: h}).Generate(context.Background(), p, English, SensitiveValues{})
			if err == nil || c.calls != 0 {
				t.Fatalf("calls=%d err=%v", c.calls, err)
			}
		})
	}
}

func TestGroupingBlockRangesAndParseFailures(t *testing.T) {
	raw := "diff --git a/a b/a\n--- a/a\n+++ b/a\n@@ -1,3 +1,3 @@\n-a\n+b\n same\n-c\n+d\n\\ No newline at end of file\n"
	blocks, err := parseGroupingBlocks(raw)
	if err != nil || len(blocks) != 2 {
		t.Fatalf("%+v %v", blocks, err)
	}
	if blocks[0].DiffRange != (DiffRange{5, 6}) || blocks[1].DiffRange != (DiffRange{8, 10}) {
		t.Fatalf("%+v", blocks)
	}
	for _, invalid := range []string{"", "+line\n", "@@ -1 +1 @@\n-a\n", "@@ -1 +1 @@\n-a\n+b\n+c\n", "@@ -1 +1 @@\n\\ No newline at end of file\n-a\n+b\n"} {
		if _, err := parseGroupingBlocks(invalid); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
	if _, err := parseGroupingBlocks(raw + "diff --git a/b b/b\n@@ -1 +1 @@\n-x\n+y\n"); err != nil {
		t.Fatal(err)
	}
	if blocks, err := parseGroupingBlocks("diff --git a/a b/a\nold mode 100644\nnew mode 100755\n"); err != nil || len(blocks) != 0 {
		t.Fatalf("mode-only %+v %v", blocks, err)
	}
}

func TestGroupingOmissionReasonsDescribeOnlyFinalRetainedContexts(t *testing.T) {
	for _, name := range []string{"none", "request", "final", "both", "budget final", "budget request", "whole"} {
		t.Run(name, func(t *testing.T) {
			h, p := attributedFixture(t)
			if name == "request" || name == "both" {
				h.Contexts[0].UserRequests = []string{"Bearer fixture-private-value"}
			}
			if name == "final" || name == "both" {
				h.Contexts[0].FinalResponses = []string{"Bearer fixture-private-value"}
			}
			if name == "budget final" {
				h.Contexts[0].FinalResponses = []string{strings.Repeat("result ", 2000)}
			}
			if name == "budget request" {
				h.Contexts[0].FinalResponses = nil
				h.Contexts[0].UserRequests = []string{strings.Repeat("request ", 2000)}
			}
			files, err := groupingProjection([]contextinput.File{p.Document.Files[0]})
			if err != nil {
				t.Fatal(err)
			}
			input, err := groupingPayload(files, map[string]string{files[0].FileID: p.Document.Files[0].ID}, h)
			if err != nil {
				t.Fatal(err)
			}
			limit := 6000
			if name == "whole" {
				// Context alone is too large; omit it without reporting nested omissions.
				ref := strings.Repeat("H", 12000)
				h.Contexts[0].ContextRef = ref
				h.Files[0].Blocks[0].Candidates[0].ContextRef = ref
				input, err = groupingPayload(files, map[string]string{files[0].FileID: p.Document.Files[0].ID}, h)
				if err != nil {
					t.Fatal(err)
				}
			}
			data, err := budgetPhaseHistory(input, SensitiveValues{}, "system", limit, 1)
			if err != nil {
				t.Fatal(err)
			}
			var payload struct {
				Files      []groupingFile        `json:"files"`
				Contexts   []AgentHistoryContext `json:"agent_history_contexts"`
				Projection struct {
					Status string `json:"status"`
				} `json:"agent_history_projection"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				t.Fatal(err)
			}
			b := payload.Files[0].ChangeBlocks[0]
			expected := map[string][]string{"none": nil, "request": {"user_request_omitted"}, "final": {"final_response_omitted"}, "both": {"user_request_omitted", "final_response_omitted"}, "budget final": {"final_response_omitted", "input_budget_exceeded"}, "budget request": {"user_request_omitted", "input_budget_exceeded"}, "whole": {"candidate_context_omitted", "input_budget_exceeded"}}[name]
			if !reflect.DeepEqual(b.AgentHistoryOmissionReasons, expected) {
				t.Fatalf("got %v want %v", b.AgentHistoryOmissionReasons, expected)
			}
			if name == "none" {
				if payload.Projection.Status != "complete" {
					t.Fatal("false partial")
				}
			} else if payload.Projection.Status != "partial" {
				t.Fatal("lost partial")
			}
			if name == "whole" && (len(payload.Contexts) != 0 || len(b.AgentHistoryCandidates) != 0 || b.AgentHistoryMatchStatus != "ambiguous") {
				t.Fatal("dangling reference or false unique")
			}
			if strings.Contains(string(data), "fixture-private-value") {
				t.Fatal("secret reached prompt")
			}
		})
	}
}

func TestGroupingAmbiguitySurvivesSharedContext(t *testing.T) {
	h, p := attributedFixture(t)
	h.Files[0].Blocks[0].Status = "ambiguous"
	h.Files[0].Blocks[0].Reasons = []string{"multiple_history_candidates"}
	c := &candidateClient{responses: candidateResponses()}
	_, err := (ThreePhaseGenerator{Client: c, GroupingHistory: h}).Generate(context.Background(), p, English, SensitiveValues{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.messages[0][1].Content, `"agent_history_match_status":"ambiguous"`) {
		t.Fatal("ambiguous history became unique")
	}
}

func TestGroupingRetainsAllFourteenHostOmissionCombinations(t *testing.T) {
	for mask := 1; mask <= 7; mask++ {
		for _, budget := range []bool{false, true} {
			h, p := attributedFixture(t)
			expected := []string{}
			for i, reason := range []string{"candidate_context_omitted", "user_request_omitted", "final_response_omitted"} {
				if mask&(1<<i) != 0 {
					expected = append(expected, reason)
				}
			}
			if budget {
				expected = append(expected, "input_budget_exceeded")
			}
			h.Files[0].Blocks[0].OmissionReasons = expected
			// A candidate omitted before this boundary is no longer a wire reference.
			if mask&1 != 0 {
				h.Files[0].Blocks[0].Status = "ambiguous"
				h.Files[0].Blocks[0].Candidates = nil
			}
			c := &candidateClient{responses: candidateResponses()}
			_, err := (ThreePhaseGenerator{Client: c, GroupingHistory: h}).Generate(context.Background(), p, English, SensitiveValues{})
			if err != nil {
				t.Fatal(err)
			}
			var payload struct {
				Files      []groupingFile `json:"files"`
				Projection struct {
					Status string `json:"status"`
				} `json:"agent_history_projection"`
			}
			if err := json.Unmarshal([]byte(c.messages[0][1].Content), &payload); err != nil {
				t.Fatal(err)
			}
			got := payload.Files[1].ChangeBlocks[0].AgentHistoryOmissionReasons
			if !reflect.DeepEqual(got, expected) || payload.Projection.Status != "partial" {
				t.Fatalf("mask=%d budget=%v got=%v want=%v projection=%s", mask, budget, got, expected, payload.Projection.Status)
			}
		}
	}
}

func TestAttributedGroupingBudgetDoesNotEnlargeTierOrDiscardGitDiff(t *testing.T) {
	h, p := attributedFixture(t)
	h.Contexts[0].FinalResponses = []string{strings.Repeat("large history result ", 10000)}
	c := &candidateClient{responses: candidateResponses()}
	_, err := (ThreePhaseGenerator{Client: c, GroupingHistory: h}).Generate(context.Background(), p, English, SensitiveValues{})
	if err != nil {
		t.Fatal(err)
	}
	if c.options[0].ContextTokens != contextinput.Context16K {
		t.Fatal("history enlarged context tier")
	}
	var payload struct {
		Files []groupingFile `json:"files"`
	}
	if err := json.Unmarshal([]byte(c.messages[0][1].Content), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Files[1].RawDiff != p.Document.Files[0].GroupingRawDiff {
		t.Fatal("history evicted current diff")
	}
	if !reflect.DeepEqual(payload.Files[1].ChangeBlocks[0].AgentHistoryOmissionReasons, []string{"final_response_omitted", "input_budget_exceeded"}) {
		t.Fatal("lost budget omission")
	}
}

func TestGroupingMetadataOnlyKeepsSpecialFileModesAndEmptyArrays(t *testing.T) {
	path, mode := "link", "120000"
	files, err := groupingProjection([]contextinput.File{{ID: "host", Status: "modified", NewPath: &path, OldMode: &mode, NewMode: &mode, WorktreeKind: "symlink", Mode: "metadata_only"}})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(files)
	if files[0].RawDiff != "" || files[0].ChangeBlocks == nil || files[0].NewMode == nil || *files[0].NewMode != "120000" {
		t.Fatalf("lost metadata: %s", data)
	}
	if !strings.Contains(string(data), `"raw_diff":""`) || !strings.Contains(string(data), `"change_blocks":[]`) {
		t.Fatal("mandatory empty fields omitted")
	}
}
