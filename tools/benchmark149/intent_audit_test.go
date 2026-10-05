package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

type intentAuditBackend struct{ packets []packet }

func (b *intentAuditBackend) Chat(ctx context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return b.ChatWithOptions(ctx, m, s, llm.Options{})
}
func (b *intentAuditBackend) ChatWithOptions(_ context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	b.packets = append(b.packets, packet{m, s, o})
	var p struct {
		Files []struct {
			ID string `json:"id"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(m[1].Content), &p); err != nil {
		return llm.Response{}, err
	}
	answer := map[string]string{}
	for _, f := range p.Files {
		answer[f.ID] = "unresolved"
	}
	data, _ := json.Marshal(answer)
	return llm.Response{Content: string(data), StopReason: "completed"}, nil
}

// This audits identifiability, not model quality. Both authored intent scenarios
// have exactly the same observed code; neither counterfactual is sent to a model.
func TestIntentAuditSameObservedInputDifferentAuthoredPartition(t *testing.T) {
	independent := fixtures()[6]
	coordinated := independent
	coordinated.Expected = [][]string{{}}
	for _, f := range coordinated.Files {
		coordinated.Expected[0] = append(coordinated.Expected[0], f.ID)
	}
	a, b := &intentAuditBackend{}, &intentAuditBackend{}
	oa := run(independent, "canonical-contracts", a)
	ob := run(coordinated, "canonical-contracts", b)
	if len(a.packets) != 1 || !reflect.DeepEqual(a.packets, b.packets) {
		t.Fatal("counterfactual changed observed request")
	}
	if oa.Complete || ob.Complete || !oa.Unresolved || !ob.Unresolved {
		t.Fatal("audit must not generate a partial plan")
	}
	_, fm, fs := quality(coordinated.Expected, independent.Expected)
	if fm != 120 || fs != 0 {
		t.Fatal("counterfactual partitions are not distinct")
	}
	raw, _ := json.Marshal(a.packets)
	t.Logf("intent_audit input_sha256=%x files=16 independent_groups=16 coordinated_groups=1 incompatible_pairs=%d calls_to_real_model=0", sha256.Sum256(raw), fm)
}
