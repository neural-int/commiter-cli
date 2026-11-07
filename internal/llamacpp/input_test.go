package llamacpp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func TestNativePreparedInputIsReusedWithinBudget(t *testing.T) {
	calls := 0
	tokens := []int{9, 2, 1}
	p, err := prepareInput(context.Background(), []int{100, 200}, func(_ context.Context, indices []int) ([]int, error) { calls++; return tokens, nil })
	if err != nil || calls != 1 || p.PreparationCalls != 1 || !reflect.DeepEqual(p.TokenIDs, tokens) || !reflect.DeepEqual(p.SelectedHistoryIndices, []int{0, 1}) {
		t.Fatalf("p=%+v calls=%d err=%v", p, calls, err)
	}
	tokens[0] = 0
	if p.TokenIDs[0] != 9 {
		t.Fatal("prepared tokens alias mutable tokenizer buffer")
	}
}

func TestHistoryOverflowValidatesMandatoryInputBeforeCandidates(t *testing.T) {
	limit := ContextTokens - OutputTokens
	calls := [][]int{}
	p, err := prepareInput(context.Background(), []int{100, 20, 30}, func(_ context.Context, indices []int) ([]int, error) {
		calls = append(calls, append([]int{}, indices...))
		n := limit + 10
		if len(indices) == 0 {
			n = limit - 5
		}
		if len(indices) == 1 {
			n = limit
		}
		return make([]int, n), nil
	})
	if err != nil || len(calls) < 2 || len(calls) > 5 || len(calls[1]) != 0 || len(p.TokenIDs) > limit || p.PreparationCalls != len(calls) {
		t.Fatalf("calls=%v input=%+v err=%v", calls, p, err)
	}
	_, err = prepareInput(context.Background(), []int{100}, func(context.Context, []int) ([]int, error) { return make([]int, limit+1), nil })
	if !errors.Is(err, llm.ErrInputTooLarge) {
		t.Fatal(err)
	}
}

func TestHistoryFallbackUsesAlreadyValidatedMandatoryTokens(t *testing.T) {
	calls := 0
	p, err := prepareInput(context.Background(), []int{1000, 1, 1, 1}, func(_ context.Context, indices []int) ([]int, error) {
		calls++
		if len(indices) == 0 {
			return []int{42}, nil
		}
		return make([]int, ContextTokens), nil
	})
	if err != nil || calls > 5 || len(p.SelectedHistoryIndices) != 0 || !reflect.DeepEqual(p.TokenIDs, []int{42}) || p.PreparationCalls != calls {
		t.Fatalf("fallback=%+v calls=%d err=%v", p, calls, err)
	}
}

func TestInputCancellationAndInvalidNativeTokens(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := prepareInput(ctx, nil, func(context.Context, []int) ([]int, error) { t.Fatal("tokenized canceled request"); return nil, nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, err = prepareInput(context.Background(), nil, func(context.Context, []int) ([]int, error) { return []int{-1}, nil })
	if err == nil {
		t.Fatal("invalid native token accepted")
	}
}

func TestUnqualifiedBackendCannotStartOrGenerate(t *testing.T) {
	b := &Backend{}
	if _, close, err := b.BeginCycle(context.Background()); !errors.Is(err, ErrUnqualifiedRuntime) || close != nil {
		t.Fatalf("unqualified runtime accepted: %v", err)
	}
	if _, err := b.ChatWithOptions(context.Background(), nil, nil, llm.Options{}); !errors.Is(err, ErrUnqualifiedRuntime) {
		t.Fatal(err)
	}
}
