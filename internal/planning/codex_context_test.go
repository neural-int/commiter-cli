package planning

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

func TestThreePhaseAttributedContextIsSharedOnceAndCategoryFeedsText(t *testing.T) {
	c := &candidateClient{responses: candidateResponses()}
	p := candidatePrepared(t)
	record := codexcontext.Record{Paths: []string{"one.go"}, FinalResponse: "Corrected one behavior"}
	result, err := (ThreePhaseGenerator{Client: c, GroupingHistory: syntheticAttributedHistory(t, p, []codexcontext.Record{record, record})}).Generate(context.Background(), p, English, SensitiveValues{})
	if err != nil || result.Calls != 3 {
		t.Fatalf("cycle failed: %v", err)
	}
	for i, messages := range c.messages {
		if strings.Contains(messages[1].Content, "codex_context") || strings.Contains(messages[1].Content, "user_requests") {
			t.Fatal("legacy input exposed")
		}
		var shared struct {
			Contexts []AgentHistoryContext `json:"agent_history_contexts"`
		}
		if err := json.Unmarshal([]byte(messages[1].Content), &shared); err != nil || len(shared.Contexts) != 1 || shared.Contexts[0].FinalResponses[0] != record.FinalResponse {
			t.Fatalf("phase %d lost context", i)
		}
		if !strings.Contains(messages[0].Content, "untrusted") {
			t.Fatal("trust boundary absent")
		}
		if i > 0 {
			_, body, _ := strings.Cut(messages[2].Content, "\n\nPhase input:\n")
			var payload struct {
				Groups []map[string]json.RawMessage `json:"groups"`
			}
			if err := json.Unmarshal([]byte(body), &payload); err != nil {
				t.Fatal(err)
			}
			if i == 1 && payload.Groups[0]["type"] != nil {
				t.Fatal("unconfirmed type in Category")
			}
			if i == 2 && string(payload.Groups[0]["type"]) != `"fix"` {
				t.Fatal("Text lost type")
			}
		}
	}
	if strings.Join(result.Plan.Commits[0].FileIDs, ",") != "F002,F001" {
		t.Fatal("host membership lost")
	}
}
func TestOptionalAttributedHistoryCannotEnlargeContextOrLeakSecrets(t *testing.T) {
	p := candidatePrepared(t)
	c := &candidateClient{responses: candidateResponses()}
	record := codexcontext.Record{FinalResponse: "Bearer private-test-value"}
	_, err := (ThreePhaseGenerator{Client: c, GroupingHistory: syntheticAttributedHistory(t, p, []codexcontext.Record{record})}).Generate(context.Background(), p, English, SensitiveValues{})
	if err != nil {
		t.Fatal(err)
	}
	for i, messages := range c.messages {
		if strings.Contains(messages[1].Content, "private-test-value") {
			t.Fatal("secret adopted")
		}
		if c.options[i].ContextTokens != contextinput.Context64K {
			t.Fatal("context changed")
		}
	}
}

func TestSyntheticCodexContextFixturesMatchFrozenCasesAndReachPrompt(t *testing.T) {
	goldPath := filepath.Join("..", "..", "docs", "benchmarks", "issue-178", "iteration-03", "gold-frozen.json")
	goldData, err := os.ReadFile(goldPath)
	if err != nil {
		t.Fatal(err)
	}
	var gold struct {
		Cases []struct {
			CaseID   string `json:"case_id"`
			Grouping struct {
				Allowed [][][]string `json:"allowed_partitions"`
			} `json:"grouping"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(goldData, &gold); err != nil {
		t.Fatal(err)
	}
	for _, tc := range gold.Cases {
		t.Run(tc.CaseID, func(t *testing.T) {
			if len(tc.Grouping.Allowed) != 1 {
				t.Fatal("expected one frozen partition")
			}
			allowed := make(map[string]bool)
			var files []contextinput.File
			for _, group := range tc.Grouping.Allowed[0] {
				for _, path := range group {
					allowed[path] = true
					filePath := path
					files = append(files, contextinput.File{NewPath: &filePath})
				}
			}
			fixturePath := filepath.Join("..", "..", "docs", "benchmarks", "issue-175", "semantic-evaluation", strings.ToLower(tc.CaseID)+"-synthetic-context.json")
			fixtureData, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				CaseID  string                `json:"case_id"`
				Context []codexcontext.Record `json:"codex_context"`
			}
			if err := json.Unmarshal(fixtureData, &fixture); err != nil {
				t.Fatal(err)
			}
			if fixture.CaseID != tc.CaseID || len(fixture.Context) == 0 || len(files) == 0 {
				t.Fatalf("invalid synthetic fixture for %s", tc.CaseID)
			}
			for _, record := range fixture.Context {
				if len(record.Paths) == 0 || strings.TrimSpace(record.FinalResponse) == "" {
					t.Fatal("empty synthetic history record")
				}
				for _, path := range record.Paths {
					if !allowed[path] {
						t.Fatalf("synthetic history path %q is absent from the frozen evaluation case", path)
					}
				}
			}
			payload := map[string]any{"task": "synthetic fixture routing check"}
			base, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			encoded, system := addCodexContext(payload, fixture.Context, files, SensitiveValues{}, base, "test system")
			var actual struct {
				Context []codexcontext.Record `json:"codex_context"`
			}
			if err := json.Unmarshal(encoded, &actual); err != nil {
				t.Fatal(err)
			}
			if len(actual.Context) != len(fixture.Context) || !strings.Contains(system, codexTrustInstruction) {
				t.Fatalf("Codex history was dropped from the prompt for %s", tc.CaseID)
			}
		})
	}
}
