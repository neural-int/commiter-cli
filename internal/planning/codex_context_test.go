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

func TestThreePhaseCodexContextIsSharedOnceAndCategoryFeedsText(t *testing.T) {
	c := &candidateClient{responses: candidateResponses()}
	record := codexcontext.Record{Paths: []string{"one.go", "asset.bin"}, UserRequest: "Correct one behavior", FinalResponse: "Corrected one behavior"}
	result, err := (ThreePhaseGenerator{Client: c, CodexContext: []codexcontext.Record{record, record}}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Calls != 3 {
		t.Fatal("extra LLM stage")
	}
	for i, messages := range c.messages {
		if strings.Contains(messages[1].Content, `"codex_context"`) {
			t.Fatalf("phase %d received legacy path-only history", i)
		}
		if !strings.Contains(messages[1].Content, `"agent_history_contexts":[]`) {
			t.Fatal("missing mandatory history array")
		}
	}
	if strings.Join(result.Plan.Commits[0].FileIDs, ",") != "F002,F001" {
		t.Fatal("host lost original membership")
	}
}

func TestOptionalCodexContextCannotEnlargeContextOrCauseBudgetFailure(t *testing.T) {
	for _, text := range []string{strings.Repeat("long context ", 1000), "Bearer private-test-value"} {
		p := candidatePrepared(t)
		c := &candidateClient{responses: candidateResponses()}
		_, err := (ThreePhaseGenerator{Client: c, CodexContext: []codexcontext.Record{{Paths: []string{"one.go"}, UserRequest: text}}}).Generate(context.Background(), p, English, SensitiveValues{})
		if err != nil {
			t.Fatal(err)
		}
		for _, messages := range c.messages {
			if strings.Contains(messages[1].Content, "codex_context") {
				t.Fatal("oversize/sensitive evidence adopted")
			}
		}
		for _, option := range c.options {
			if option.ContextTokens != contextinput.Context16K {
				t.Fatal("optional log increased context tier")
			}
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
				if len(record.Paths) == 0 || strings.TrimSpace(record.UserRequest) == "" {
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
			encoded, system := addCodexContext(payload, fixture.Context, files, SensitiveValues{}, base, "test system", contextinput.Context16K, 768)
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
