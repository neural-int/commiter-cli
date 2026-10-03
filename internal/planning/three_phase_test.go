package planning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

type candidateClient struct {
	responses []string
	stop      string
	err       error
	calls     int
	options   []llm.Options
	messages  [][]llm.Message
	deadline  time.Time
}

func (c *candidateClient) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected legacy call")
}
func (c *candidateClient) ChatWithOptions(ctx context.Context, messages []llm.Message, _ json.RawMessage, options llm.Options) (llm.Response, error) {
	c.calls++
	c.options = append(c.options, options)
	c.messages = append(c.messages, messages)
	deadline, ok := ctx.Deadline()
	if !ok {
		return llm.Response{}, errors.New("missing shared deadline")
	}
	if c.deadline.IsZero() {
		c.deadline = deadline
	} else if c.deadline != deadline {
		return llm.Response{}, errors.New("deadline reset")
	}
	if c.err != nil {
		return llm.Response{}, c.err
	}
	if c.calls > len(c.responses) {
		return llm.Response{}, errors.New("unexpected extra call")
	}
	stop := c.stop
	if stop == "" {
		stop = "completed"
	}
	return llm.Response{Content: c.responses[c.calls-1], StopReason: stop}, nil
}
func candidatePrepared(t *testing.T) contextinput.Prepared {
	t.Helper()
	p := preparedInput(t, English)
	p.Budget.ContextTokens = contextinput.Context16K
	return p
}
func candidateResponses() []string {
	return []string{
		`{"path:one.go":"G002","path:asset.bin":"G002"}`,
		`{"G001":{"type":"fix","breaking_evidence_ref":"none"}}`,
		`{"G001":{"scope":"planner","summary":"correct planning"}}`,
	}
}
func TestThreePhaseRestoresIDsAndUsesSharedBoundedProfiles(t *testing.T) {
	c := &candidateClient{responses: candidateResponses()}
	result, err := (ThreePhaseGenerator{Client: c}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Calls != 3 || result.Repaired || len(result.Plan.Commits) != 1 {
		t.Fatalf("result = %#v", result)
	}
	commit := result.Plan.Commits[0]
	if strings.Join(commit.FileIDs, ",") != "F002,F001" || commit.Breaking {
		t.Fatalf("commit = %#v", commit)
	}
	for i, profile := range []string{"bounded-grouping", "bounded-category", "bounded-text"} {
		if c.options[i].GenerationProfile != profile || c.options[i].ContextTokens != 16384 || c.options[i].OutputTokens != []int{768, 512, 768}[i] {
			t.Fatalf("phase %d options = %#v", i, c.options[i])
		}
	}
	if time.Until(c.deadline) > CandidateCycleTimeout {
		t.Fatal("cycle deadline exceeded")
	}
	if strings.Contains(c.messages[2][1].Content, `"breaking_evidence_ref"`) {
		t.Fatal("generated category fed back into text phase")
	}
}
func TestThreePhaseStopsWithoutRepairOrPartialPlan(t *testing.T) {
	tests := []struct {
		name           string
		phase          int
		response, stop string
	}{
		{"unknown path", 0, `{"path:unknown.go":"G001","path:one.go":"G001"}`, ""},
		{"duplicate path", 0, `{"path:one.go":"G001","path:one.go":"G002","path:asset.bin":"G001"}`, ""},
		{"missing path", 0, `{"path:one.go":"G001"}`, ""},
		{"unknown label", 0, `{"path:one.go":"G999","path:asset.bin":"G999"}`, ""},
		{"incomplete output", 0, `{"path:one.go":"G002","path:asset.bin":"G002"}`, "max_tokens"},
		{"unresolved evidence", 1, `{"G001":{"type":"fix","breaking_evidence_ref":"unresolved"}}`, ""},
		{"invented evidence", 1, `{"G001":{"type":"fix","breaking_evidence_ref":"invented"}}`, ""},
		{"unknown category property", 1, `{"G001":{"type":"fix","breaking_evidence_ref":"none","reasoning":"unsafe"}}`, ""},
		{"empty scope", 2, `{"G001":{"scope":"","summary":"correct planning"}}`, ""},
		{"duplicate nested key", 2, `{"G001":{"scope":"planner","summary":"first","summary":"second"}}`, ""},
		{"trailing prose", 2, `{"G001":{"scope":"planner","summary":"correct planning"}} prose`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := candidateResponses()
			responses[tt.phase] = tt.response
			c := &candidateClient{responses: responses, stop: tt.stop}
			result, err := (ThreePhaseGenerator{Client: c}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
			if err == nil || len(result.Plan.Commits) != 0 || c.calls != tt.phase+1 || result.Repaired {
				t.Fatalf("result=%#v calls=%d error=%v", result, c.calls, err)
			}
		})
	}
}
func TestThreePhaseHostWitnessCannotBeIgnored(t *testing.T) {
	for _, ref := range []string{"none", "observed-api"} {
		t.Run(ref, func(t *testing.T) {
			responses := candidateResponses()
			responses[1] = `{"G001":{"type":"fix","breaking_evidence_ref":"` + ref + `"}}`
			c := &candidateClient{responses: responses}
			g := ThreePhaseGenerator{Client: c, Evidence: []BreakingEvidence{{ID: "observed-api", FileIDs: []string{"F001"}, Fact: "public signature removed"}}}
			result, err := g.Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
			if ref == "none" {
				if err == nil || c.calls != 2 || len(result.Plan.Commits) != 0 {
					t.Fatalf("witness ignored: %#v %v", result, err)
				}
			} else if err != nil || !result.Plan.Commits[0].Breaking {
				t.Fatalf("witness rejected: %#v %v", result, err)
			}
		})
	}
}
func TestThreePhaseRejectsUnsupportedInputBeforeGeneration(t *testing.T) {
	for _, name := range []string{"size", "context", "compression", "prompt", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			p := candidatePrepared(t)
			ctx := context.Background()
			switch name {
			case "size":
				for len(p.Document.Files) < 5 {
					f := p.Document.Files[0]
					f.ID = "F00" + string(rune('1'+len(p.Document.Files)))
					p.Document.Files = append(p.Document.Files, f)
				}
				p.Prompt, _ = Renderer(English)(p.Document)
			case "context":
				p.Budget.ContextTokens = 8192
			case "compression":
				p.SummaryCount = 1
			case "prompt":
				p.Document.Files[0].RawDiff = strings.Repeat("x", 16384)
				p.Prompt, _ = Renderer(English)(p.Document)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			c := &candidateClient{responses: candidateResponses()}
			r, err := (ThreePhaseGenerator{Client: c}).Generate(ctx, p, English, SensitiveValues{})
			if err == nil || c.calls != 0 || len(r.Plan.Commits) != 0 {
				t.Fatalf("calls=%d result=%#v error=%v", c.calls, r, err)
			}
		})
	}
}
func TestThreePhaseDoesNotRetryTransportOrLeakSensitiveOutput(t *testing.T) {
	for _, transport := range []bool{true, false} {
		c := &candidateClient{responses: []string{`{"path:one.go":"fixture-secret","path:asset.bin":"G001"}`}}
		if transport {
			c.err = errors.New("transport failed with fixture-secret")
		}
		r, err := (ThreePhaseGenerator{Client: c}).Generate(context.Background(), candidatePrepared(t), English, ExtractSensitiveValues([]byte(`{"token":"fixture-secret"}`)))
		if err == nil || strings.Contains(err.Error(), "fixture-secret") || c.calls != 1 || len(r.Plan.Commits) != 0 {
			t.Fatalf("result=%#v error=%v", r, err)
		}
	}
}
