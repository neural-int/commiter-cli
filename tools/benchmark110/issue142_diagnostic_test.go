package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

type issue142DiagnosticStub struct {
	options  []llm.Options
	response llm.Response
}

func TestIssue142ForcedChoiceExcludesNoneAndScoresCandidate(t *testing.T) {
	item, err := issue142FindFixture("cross_directory")
	if err != nil {
		t.Fatal(err)
	}
	prepared, _, candidates, err := issue142DiagnosticInputs(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	originalSystem, originalPrompt, originalSchema, err := issue142SelectionInput(prepared, candidates, false)
	if err != nil {
		t.Fatal(err)
	}
	if issue142Digest([]byte(originalSystem+string(originalPrompt))) != "bc2f43d3cb86f6b874eb32732a5d7650bf5be0875664e0e407597a36fbcc1544" || issue142Digest(originalSchema) != "7b60d6326c3f18437c9484aa1610835ef73877f89ad4fe77d7d17a4be8b8ad7c" {
		t.Fatal("previous selection contract changed")
	}
	system, prompt, schema, err := issue142ForcedSelectionInput(prepared, candidates, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(system, "none") || strings.Contains(string(schema), "none") || strings.Contains(string(prompt), "or none") {
		t.Fatal("forced-choice input still offers none")
	}
	var decoded struct {
		Properties struct {
			CandidateID struct {
				Enum []string `json:"enum"`
			} `json:"candidate_id"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Properties.CandidateID.Enum) != 2 || decoded.Properties.CandidateID.Enum[0] != "C001" || decoded.Properties.CandidateID.Enum[1] != "C002" {
		t.Fatalf("unexpected forced-choice enum: %v", decoded.Properties.CandidateID.Enum)
	}
	stub := &issue142DiagnosticStub{response: llm.Response{StopReason: "completed", Content: `{"candidate_id":"none"}`}}
	forbidden := issue142DiagnosticCallMode(context.Background(), stub, "stub", "mlx", "forced", "two-choice", item, prepared, candidates, 1, 2048, false, true)
	if forbidden.Failure != "forbidden_none" || forbidden.ValidCandidate || forbidden.None || forbidden.CorrectSelection {
		t.Fatalf("forbidden choice was scored: %+v", forbidden)
	}
	goldID := ""
	for _, candidate := range candidates {
		if sameGroups(candidate.Groups, item.reference) {
			goldID = candidate.ID
		}
	}
	stub.response.Content = `{"candidate_id":"` + goldID + `"}`
	correct := issue142DiagnosticCallMode(context.Background(), stub, "stub", "mlx", "forced", "two-choice", item, prepared, candidates, 1, 2048, false, true)
	if !correct.ValidCandidate || !correct.CorrectSelection || !correct.CompleteAssignment {
		t.Fatalf("gold candidate was not scored: %+v", correct)
	}
}

func (stub *issue142DiagnosticStub) Chat(ctx context.Context, messages []llm.Message, schema json.RawMessage) (llm.Response, error) {
	return stub.ChatWithOptions(ctx, messages, schema, llm.Options{})
}
func (stub *issue142DiagnosticStub) ChatWithOptions(ctx context.Context, messages []llm.Message, schema json.RawMessage, options llm.Options) (llm.Response, error) {
	stub.options = append(stub.options, options)
	return stub.response, nil
}

func TestIssue142BudgetDiagnosticChangesOnlyOutputLimit(t *testing.T) {
	item, err := issue142FindFixture("cross_directory")
	if err != nil {
		t.Fatal(err)
	}
	prepared, _, candidates, err := issue142DiagnosticInputs(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	stub := &issue142DiagnosticStub{response: llm.Response{StopReason: "completed", Content: `{"candidate_id":"none"}`}}
	low := issue142DiagnosticCall(context.Background(), stub, "stub", "mlx", "budget", "louvain", item, prepared, candidates, 1, 1024, false)
	high := issue142DiagnosticCall(context.Background(), stub, "stub", "mlx", "budget", "louvain", item, prepared, candidates, 1, 4096, false)
	if low.PromptSHA256 != high.PromptSHA256 || low.SchemaSHA256 != high.SchemaSHA256 || low.PromptBytes != high.PromptBytes {
		t.Fatal("prompt or schema changed with budget")
	}
	if len(stub.options) != 2 || stub.options[0].OutputTokens != 1024 || stub.options[1].OutputTokens != 4096 || stub.options[0].ContextTokens != stub.options[1].ContextTokens {
		t.Fatalf("unexpected options: %+v", stub.options)
	}
	if !low.None || !high.None || low.CorrectSelection || high.CorrectSelection {
		t.Fatalf("incorrect semantic scoring: %+v %+v", low, high)
	}
}

func TestIssue142IncompleteStopIsNotSemanticSelection(t *testing.T) {
	item, err := issue142FindFixture("cross_directory")
	if err != nil {
		t.Fatal(err)
	}
	prepared, _, candidates, err := issue142DiagnosticInputs(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	stub := &issue142DiagnosticStub{response: llm.Response{StopReason: "max_tokens"}}
	row := issue142DiagnosticCall(context.Background(), stub, "stub", "mlx", "budget", "louvain", item, prepared, candidates, 1, 1024, false)
	if row.Failure != "max_tokens" || row.ValidCandidate || row.None || row.CorrectSelection || row.OutputBytes != 0 {
		t.Fatalf("incomplete response was scored: %+v", row)
	}
}

func TestIssue142GoldSwapChangesIDAndDisplayPosition(t *testing.T) {
	gold := issue142Candidate{ID: "C001", Groups: [][]string{{"F001", "F002"}}}
	wrong := issue142Candidate{ID: "C002", Groups: [][]string{{"F001"}, {"F002"}}}
	swapped := issue142SwapCandidateIDs([]issue142Candidate{gold, wrong})
	if len(swapped) != 2 || swapped[0].ID != "C001" || swapped[1].ID != "C002" || !sameGroups(swapped[1].Groups, gold.Groups) {
		t.Fatalf("gold was not moved to C002 and second position: %+v", swapped)
	}
}
