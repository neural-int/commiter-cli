package contextinput

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/relation"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type summarizeFunc func(context.Context, SummaryStage, Document) (Document, error)

func (f summarizeFunc) Summarize(ctx context.Context, stage SummaryStage, document Document) (Document, error) {
	return f(ctx, stage, document)
}

func TestPrepareUsesSmallestContextWithoutSummary(t *testing.T) {
	document := testDocument()
	prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "auto", MaxContextTokens: Context32K}, JSONRenderer, nil)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Budget.ContextTokens != Context8K || prepared.SummaryStage != SummaryNone || prepared.SummaryCount != 0 {
		t.Fatalf("prepared=%#v", prepared)
	}
	if len(prepared.Prompt) != prepared.Budget.PromptBytes {
		t.Fatalf("prompt=%d budget=%#v", len(prepared.Prompt), prepared.Budget)
	}
}

func TestPrepareSelectsContextUsingPromptOverhead(t *testing.T) {
	for _, test := range []struct {
		name       string
		promptSize int
		want       int
	}{
		{name: "8k plus overhead", promptSize: Context8K - TemplateReserve - MinimumOutputSpace, want: Context16K},
		{name: "16k plus overhead", promptSize: Context16K - TemplateReserve - MinimumOutputSpace, want: Context32K},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepared, err := Prepare(context.Background(), testDocument(), BudgetConfig{
				Context: "auto", MaxContextTokens: Context32K, PromptOverheadBytes: 1,
			}, func(Document) ([]byte, error) {
				return make([]byte, test.promptSize), nil
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if prepared.Budget.ContextTokens != test.want || prepared.SummaryStage != SummaryNone {
				t.Fatalf("prepared=%#v", prepared)
			}
		})
	}
}

func TestPrepareSummarizesFixedContextWhenPromptOverheadDoesNotFit(t *testing.T) {
	stages := []SummaryStage{}
	prepared, err := Prepare(context.Background(), testDocument(), BudgetConfig{
		Context: "8k", MaxContextTokens: Context32K, PromptOverheadBytes: 1,
	}, func(document Document) ([]byte, error) {
		if document.Files[0].Summary == "file" {
			return []byte("fits"), nil
		}
		return make([]byte, Context8K-TemplateReserve-MinimumOutputSpace), nil
	}, summarizeFunc(func(_ context.Context, stage SummaryStage, document Document) (Document, error) {
		stages = append(stages, stage)
		document.Files[0].RawDiff = ""
		document.Files[0].Summary = string(stage)
		return document, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stages, []SummaryStage{SummaryFile}) || prepared.SummaryStage != SummaryFile || prepared.SummaryCount != 1 {
		t.Fatalf("stages=%v prepared=%#v", stages, prepared)
	}
}

func TestPrepareSummarizesInFileHunkChunkOrder(t *testing.T) {
	document := testDocument()
	stages := []SummaryStage{}
	render := func(document Document) ([]byte, error) {
		switch document.Files[0].Summary {
		case "file":
			return make([]byte, Context16K), nil
		case "hunk":
			return make([]byte, Context8K), nil
		case "chunk":
			return []byte("fits"), nil
		default:
			return make([]byte, Context32K), nil
		}
	}
	summarizer := summarizeFunc(func(_ context.Context, stage SummaryStage, document Document) (Document, error) {
		stages = append(stages, stage)
		document.Files[0].RawDiff = ""
		document.Files[0].Summary = string(stage)
		return document, nil
	})
	prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "auto", MaxContextTokens: Context8K}, render, summarizer)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stages, []SummaryStage{SummaryFile, SummaryHunk, SummaryChunk}) || prepared.SummaryStage != SummaryChunk || prepared.SummaryCount != 3 {
		t.Fatalf("stages=%v prepared=%#v", stages, prepared)
	}
}

func TestPrepareRejectsIncompleteSummaryBeforeRenderingOrPlanning(t *testing.T) {
	document := testDocument()
	renders := 0
	render := func(Document) ([]byte, error) {
		renders++
		return make([]byte, Context32K), nil
	}
	summarizer := summarizeFunc(func(_ context.Context, _ SummaryStage, document Document) (Document, error) {
		document.Files[0].ChangeHash = "rewritten"
		return document, nil
	})
	prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "auto", MaxContextTokens: Context8K}, render, summarizer)
	if err == nil || !strings.Contains(err.Error(), "summary is incomplete") || renders != 1 || prepared.SummaryCount != 1 {
		t.Fatalf("renders=%d count=%d error=%v", renders, prepared.SummaryCount, err)
	}
}

func TestPrepareRejectsMissingOrDuplicateFileAndEvidenceChanges(t *testing.T) {
	for name, mutate := range map[string]func(Document) Document{
		"missing": func(document Document) Document {
			document.Files = document.Files[:0]
			return document
		},
		"duplicate": func(document Document) Document {
			document.Files = append(document.Files, document.Files[0])
			return document
		},
		"evidence": func(document Document) Document {
			document.Files[0].Evidence[0].Kind = "rewritten"
			return document
		},
		"path": func(document Document) Document {
			*document.Files[0].NewPath = "rewritten.go"
			return document
		},
	} {
		t.Run(name, func(t *testing.T) {
			document := testDocument()
			summarizer := summarizeFunc(func(_ context.Context, _ SummaryStage, value Document) (Document, error) { return mutate(value), nil })
			_, err := Prepare(context.Background(), document, BudgetConfig{Context: "8k", MaxContextTokens: Context32K}, func(Document) ([]byte, error) {
				return make([]byte, Context8K), nil
			}, summarizer)
			if err == nil {
				t.Fatal("incomplete summary was accepted")
			}
		})
	}
}

func TestPrepareStopsAfterSummaryFailureOrFinalOverflow(t *testing.T) {
	document := testDocument()
	failure := errors.New("summary unavailable")
	failed, err := Prepare(context.Background(), document, BudgetConfig{Context: "8k", MaxContextTokens: Context32K}, func(Document) ([]byte, error) {
		return make([]byte, Context8K), nil
	}, summarizeFunc(func(context.Context, SummaryStage, Document) (Document, error) {
		time.Sleep(time.Millisecond)
		return Document{}, failure
	}))
	if !errors.Is(err, failure) || failed.SummaryCount != 1 || failed.SummaryDuration <= 0 {
		t.Fatalf("failed=%#v error=%v", failed, err)
	}

	stages := 0
	overflow, err := Prepare(context.Background(), document, BudgetConfig{Context: "8k", MaxContextTokens: Context32K}, func(Document) ([]byte, error) {
		return make([]byte, Context8K), nil
	}, summarizeFunc(func(_ context.Context, _ SummaryStage, document Document) (Document, error) {
		stages++
		time.Sleep(time.Millisecond)
		return document, nil
	}))
	if !errors.Is(err, ErrTooLarge) || stages != 3 || overflow.SummaryCount != 3 || overflow.SummaryDuration <= 0 {
		t.Fatalf("stages=%d overflow=%#v error=%v", stages, overflow, err)
	}
}

func testDocument() Document {
	path := "main.go"
	return Document{
		SchemaVersion: SchemaVersion,
		Repository:    Repository{Head: "head", Branch: "main", IndexIdentity: "index"},
		Files: []File{{
			ID: "F001", Status: "M", NewPath: &path, Language: "Go", ChangeHash: "hash",
			Mode: syntax.ModeStructural, Evidence: []syntax.Evidence{{Kind: "function_declaration", Name: "changed"}},
		}},
	}
}

func TestPrepareFallsBackWithoutOversizedRelationContext(t *testing.T) {
	document := testDocument()
	document.RelationContext = &RelationContext{
		Components: []relation.CandidateComponent{{ID: "C001", FileIDs: []string{"F001"}}},
		Statistics: relation.GraphStatistics{NodeCount: 1},
	}
	relationRenders, fallbackRenders := 0, 0
	render := func(value Document) ([]byte, error) {
		if value.RelationContext != nil {
			relationRenders++
			return make([]byte, Context8K), nil
		}
		fallbackRenders++
		return []byte("{}"), nil
	}
	prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "8k", MaxContextTokens: Context32K}, render, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.RelationContextOmitted || prepared.Document.RelationContext != nil || len(prepared.Document.Files) != 1 || prepared.Document.Files[0].ID != "F001" {
		t.Fatalf("relation fallback changed required input or was not reported: %#v", prepared)
	}
	if relationRenders == 0 || fallbackRenders != 1 || string(prepared.Prompt) != "{}" {
		t.Fatalf("relation/fallback renders=%d/%d prompt=%q", relationRenders, fallbackRenders, prepared.Prompt)
	}
}

func TestValidatePreservedRejectsRelationContextMutation(t *testing.T) {
	original := testDocument()
	original.RelationContext = &RelationContext{Components: []relation.CandidateComponent{{ID: "C001", FileIDs: []string{"F001"}}}}
	summarized := cloneDocument(original)
	summarized.RelationContext.Components[0].FileIDs[0] = "F002"
	if err := ValidatePreserved(original, summarized); err == nil {
		t.Fatal("summary rewrote relation context")
	}
	if original.RelationContext.Components[0].FileIDs[0] != "F001" {
		t.Fatal("cloned relation context aliases the original document")
	}
}

func TestPrepareFallsBackWhenRelationContextIsInvalid(t *testing.T) {
	document := testDocument()
	document.RelationContext = &RelationContext{Components: []relation.CandidateComponent{{ID: "C001", FileIDs: []string{"F999"}}}}
	prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "8k", MaxContextTokens: Context32K}, JSONRenderer, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.RelationContextOmitted || prepared.Document.RelationContext != nil || len(prepared.Document.Files) != 1 || prepared.Document.Files[0].ID != "F001" {
		t.Fatalf("invalid graph fallback changed required file IDs: %#v", prepared)
	}
}
