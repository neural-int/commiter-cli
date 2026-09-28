package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

type issue142DiagnosticStub struct {
	options  []llm.Options
	response llm.Response
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
