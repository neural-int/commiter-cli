package llamacpp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestTrialMessagesApplyWholeHistorySelectionBeforeDisplay(t *testing.T) {
	index := []int{0}
	shared := llm.DisplayTemplate{Parts: []llm.DisplayTemplate{{Literal: `{"diff":"a\nb","history":`}, {Require: index, Literal: `["past\nresponse"]`, IfMissing: []llm.DisplayTemplate{{Literal: `[]`}}}, {Literal: `}`}}}
	phase := llm.DisplayTemplate{Literal: `{"group":"G001"}`}
	messages := []llm.Message{{Role: "system", Content: "with history"}, {Role: "user", Content: "unused"}, {Role: "user", Content: "Phase rules\n\nPhase input:\n{}"}}
	options := llm.Options{HistoryInput: &llm.OptionalHistoryInput{SystemWithoutHistory: "without history", SharedTemplate: &shared, PhaseTemplate: &phase}}
	for _, selected := range [][]int{{0}, {}} {
		rendered, err := trialMessages(messages, options, selected)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(rendered[1].Content, "a\nb") {
			t.Fatal("literal newline display was not applied")
		}
		if strings.Contains(rendered[1].Content, "past") != (len(selected) > 0) {
			t.Fatal("history selection was not applied")
		}
		if len(selected) == 0 && rendered[0].Content != "without history" {
			t.Fatal("omission system instruction was not selected")
		}
		if rendered[2].Content != "Phase rules\n\nPhase input:\n"+phase.Literal {
			t.Fatal("phase input was not preserved")
		}
	}
	if messages[1].Content != "unused" {
		t.Fatal("caller messages were mutated")
	}
}

func TestTrialRejectsUnsupportedPhaseBeforeNativeRequest(t *testing.T) {
	cycle := &trialCycle{}
	if _, err := cycle.ChatWithOptions(context.Background(), nil, nil, llm.Options{}); err == nil {
		t.Fatal("unsupported options accepted")
	}
	options := llm.Options{GenerationProfile: "bounded-grouping", InputLayout: "harness004", LiteralNewlines: true, ContextTokens: ContextTokens, OutputTokens: OutputTokens}
	if _, err := cycle.ChatWithOptions(context.Background(), nil, json.RawMessage(`{"type":"string","pattern":".*"}`), options); err == nil {
		t.Fatal("unknown schema constraint reached native generation")
	}
}
