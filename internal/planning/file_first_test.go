package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

type fileFirstMetadataClient struct {
	calls     int
	failAt    int
	bad       string
	stop      string
	transport bool
	deadline  time.Time
	packets   [][]string
}

func (c *fileFirstMetadataClient) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("legacy call forbidden")
}

func (c *fileFirstMetadataClient) ChatWithOptions(ctx context.Context, messages []llm.Message, schema json.RawMessage, options llm.Options) (llm.Response, error) {
	c.calls++
	deadline, ok := ctx.Deadline()
	if !ok || (!c.deadline.IsZero() && deadline != c.deadline) {
		return llm.Response{}, errors.New("shared deadline changed")
	}
	c.deadline = deadline
	var payload struct {
		Groups []candidateGroup `json:"groups"`
	}
	if err := json.Unmarshal([]byte(messages[1].Content), &payload); err != nil {
		return llm.Response{}, err
	}
	if len(payload.Groups) < 1 || len(payload.Groups) > 4 || strings.Contains(messages[1].Content, `"breaking_evidence_ref"`) {
		return llm.Response{}, errors.New("invalid metadata packet")
	}
	budget := 512
	if options.GenerationProfile == "bounded-text" {
		budget = 768
	} else if options.GenerationProfile != "bounded-category" {
		return llm.Response{}, errors.New("grouping invoked")
	}
	if options.ContextTokens != 16384 || options.OutputTokens != budget {
		return llm.Response{}, errors.New("profile budgets changed")
	}
	var shape struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schema, &shape); err != nil {
		return llm.Response{}, err
	}
	answers := map[string]any{}
	ids := []string{}
	for _, group := range payload.Groups {
		ids = append(ids, group.ID)
		if len(group.FileIDs) != 1 || len(group.Files) != 1 || !strings.HasPrefix(group.Files[0].ID, "path:") {
			return llm.Response{}, errors.New("membership changed")
		}
		if options.GenerationProfile == "bounded-category" {
			kind, ref := "fix", "none"
			if candidateTestOnly(group) {
				kind = "test"
			}
			if group.FileIDs[0] == "F005" {
				ref = "observed-api"
			}
			answers[group.ID] = map[string]string{"type": kind, "breaking_evidence_ref": ref}
		} else {
			answers[group.ID] = map[string]string{"scope": "settings", "summary": "correct retry bounds"}
		}
	}
	if !reflect.DeepEqual(ids, shape.Required) {
		return llm.Response{}, errors.New("schema packet mismatch")
	}
	c.packets = append(c.packets, ids)
	data, _ := json.Marshal(answers)
	response := llm.Response{Content: string(data), StopReason: "completed"}
	if c.calls == c.failAt {
		if c.transport {
			return llm.Response{}, errors.New("transport fixture-secret")
		}
		response.Content = c.bad
		if c.stop != "" {
			response.StopReason = c.stop
		}
	}
	return response, nil
}

func fileFirstPrepared(t *testing.T, count int) contextinput.Prepared {
	t.Helper()
	p := candidatePrepared(t)
	prototype := p.Document.Files[0]
	p.Document.Files = nil
	for i := 1; i <= count; i++ {
		file := prototype
		file.ID = fmt.Sprintf("F%03d", i)
		path := fmt.Sprintf("module%d/setting.go", i)
		if i%2 == 0 {
			path = fmt.Sprintf("module%d/setting_test.go", i)
		}
		file.NewPath = &path
		p.Document.Files = append(p.Document.Files, file)
	}
	p.Prompt, _ = Renderer(English)(p.Document)
	return p
}

func fileFirstGenerator(c ChatClient, count int) FileFirstGenerator {
	g := FileFirstGenerator{Client: c}
	if count >= 5 {
		g.Evidence = []BreakingEvidence{{ID: "observed-api", FileIDs: []string{"F005"}, Fact: "public signature removed"}}
	}
	return g
}

func TestFileFirstMetadataOneThroughSixteenKeepsBoundariesAndBudget(t *testing.T) {
	for n := 1; n <= 16; n++ {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			p := fileFirstPrepared(t, n)
			// Input order does not change fixed stage order or packet membership.
			for i, j := 0, len(p.Document.Files)-1; i < j; i, j = i+1, j-1 {
				p.Document.Files[i], p.Document.Files[j] = p.Document.Files[j], p.Document.Files[i]
			}
			p.Prompt, _ = Renderer(English)(p.Document)
			before, _ := json.Marshal(p.Document)
			c := &fileFirstMetadataClient{}
			r, err := fileFirstGenerator(c, n).Generate(context.Background(), p, English, SensitiveValues{})
			if err != nil {
				t.Fatal(err)
			}
			if c.calls != 2*((n+3)/4) || r.Calls != c.calls || r.Repaired || len(r.Plan.Commits) != n {
				t.Fatalf("result=%#v calls=%d", r, c.calls)
			}
			for i, commit := range r.Plan.Commits {
				if !reflect.DeepEqual(commit.FileIDs, []string{fmt.Sprintf("F%03d", i+1)}) || commit.Breaking != (i == 4) {
					t.Fatalf("commit=%#v", commit)
				}
				kind := "fix"
				if (i+1)%2 == 0 {
					kind = "test"
				}
				if commit.Type != kind {
					t.Fatalf("type=%s", commit.Type)
				}
			}
			after, _ := json.Marshal(p.Document)
			if string(before) != string(after) || time.Until(c.deadline) > CandidateCycleTimeout {
				t.Fatal("input or cycle bound changed")
			}
		})
	}
}

func TestFileFirstMetadataLateFailureNeverReturnsPartialPlan(t *testing.T) {
	for _, tt := range []struct {
		name, bad, stop string
		transport       bool
	}{
		{"missing groups", `{}`, "", false},
		{"join attempt", `{"G005":{"scope":"settings","summary":"correct retry bounds","file_ids":["F001","F005"]}}`, "", false},
		{"duplicate key", `{"G005":{"scope":"settings","summary":"first","summary":"second"}}`, "", false},
		{"invalid scope", `{"G005":{"scope":"fix","summary":"correct retry bounds"}}`, "", false},
		{"invalid summary", `{"G005":{"scope":"settings","summary":""}}`, "", false},
		{"unresolved", `{"G005":{"type":"fix","breaking_evidence_ref":"unresolved"}}`, "", false},
		{"ignored witness", `{"G005":{"type":"fix","breaking_evidence_ref":"none"}}`, "", false},
		{"sensitive", `{"G005":{"scope":"settings","summary":"fixture-secret"}}`, "", false},
		{"incomplete", `{}`, "max_tokens", false},
		{"transport", "", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			at := 4
			if tt.name == "unresolved" || tt.name == "ignored witness" {
				at = 3
			}
			c := &fileFirstMetadataClient{failAt: at, bad: tt.bad, stop: tt.stop, transport: tt.transport}
			r, err := fileFirstGenerator(c, 5).Generate(context.Background(), fileFirstPrepared(t, 5), English, ExtractSensitiveValues([]byte(`{"token":"fixture-secret"}`)))
			if err == nil || len(r.Plan.Commits) != 0 || c.calls != at || r.Repaired || strings.Contains(err.Error(), "fixture-secret") {
				t.Fatalf("result=%#v calls=%d error=%v", r, c.calls, err)
			}
		})
	}
}

func TestFileFirstMetadataRejectsUnsupportedBeforeGeneration(t *testing.T) {
	for _, name := range []string{"seventeen", "duplicate ID", "duplicate path", "missing path", "compressed", "context", "oversized packet", "invalid evidence", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			p := fileFirstPrepared(t, 2)
			g := FileFirstGenerator{}
			ctx := context.Background()
			switch name {
			case "seventeen":
				p = fileFirstPrepared(t, 17)
			case "duplicate ID":
				p.Document.Files[1].ID = p.Document.Files[0].ID
			case "duplicate path":
				p.Document.Files[1].NewPath = p.Document.Files[0].NewPath
			case "missing path":
				p.Document.Files[0].NewPath = nil
				p.Document.Files[0].OldPath = nil
			case "compressed":
				p.SummaryCount = 1
			case "context":
				p.Budget.ContextTokens = 8192
			case "oversized packet":
				p.Document.Files[0].RawDiff = strings.Repeat("x", 16384)
			case "invalid evidence":
				g.Evidence = []BreakingEvidence{{ID: "witness", FileIDs: []string{"unknown"}, Fact: "fact"}}
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			p.Prompt, _ = Renderer(English)(p.Document)
			c := &fileFirstMetadataClient{}
			g.Client = c
			r, err := g.Generate(ctx, p, English, SensitiveValues{})
			if err == nil || c.calls != 0 || len(r.Plan.Commits) != 0 {
				t.Fatalf("result=%#v calls=%d error=%v", r, c.calls, err)
			}
		})
	}
}
