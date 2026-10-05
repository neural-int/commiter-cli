package main

import (
	"context"
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"testing"
)

type failingBackend struct{ calls int }

func (b *failingBackend) Chat(c context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return b.ChatWithOptions(c, m, s, llm.Options{})
}
func (b *failingBackend) ChatWithOptions(context.Context, []llm.Message, json.RawMessage, llm.Options) (llm.Response, error) {
	b.calls++
	return llm.Response{StopReason: "max_tokens"}, nil
}
func TestIncompleteExtractionStopsBeforeGlobal(t *testing.T) {
	b := &failingBackend{}
	o := run(contractFixtures()[1], "semantic-ir", b)
	if b.calls != 1 || o.Complete || o.Exact != nil || len(o.Groups) != 0 {
		t.Fatal("partial extraction accepted")
	}
}
func TestInvalidMembershipAndDuplicateJSON(t *testing.T) {
	ids := []string{"A", "B"}
	for _, m := range []map[string]string{{"A": "G001"}, {"A": "G001", "X": "G001"}, {"A": "G001", "B": "unresolved"}, {"A": "G003", "B": "G001"}} {
		if _, e := partition(ids, m); e == nil {
			t.Fatal("invalid assignment accepted")
		}
	}
	for _, s := range []string{`{"A":"G001","A":"G002"}`, `{"A":"G001"} {}`} {
		var m map[string]string
		if strictCandidateJSON([]byte(s), &m) == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
}
