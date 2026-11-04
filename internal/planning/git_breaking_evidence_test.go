package planning

import (
	"context"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
	"strings"
	"testing"
)

func TestGitChangesCanGroundBreakingWithoutInjectedWitness(t *testing.T) {
	for _, ref := range []string{"git:F2:B1", "none", "unresolved", "git:F99:B1"} {
		t.Run(ref, func(t *testing.T) {
			client := &candidateClient{responses: candidateResponses()}
			client.responses[1] = `{"G001":{"type":"fix","breaking_evidence_ref":"` + ref + `"}}`
			result, err := (ThreePhaseGenerator{Client: client}).Generate(context.Background(), candidatePrepared(t), English, SensitiveValues{})
			if ref == "unresolved" || ref == "git:F99:B1" {
				if err == nil || client.calls != 2 || len(result.Plan.Commits) != 0 {
					t.Fatal("unresolved or invented witness accepted", result, err)
				}
				return
			}
			if err != nil || len(result.Plan.Commits) != 1 || result.Plan.Commits[0].Breaking != (ref != "none") {
				t.Fatal("normal Git input cannot select breaking", result, err)
			}
			if !strings.Contains(client.messages[1][2].Content, "git:F2:B1") || strings.Contains(client.messages[2][2].Content, "breaking_evidence_candidates") {
				t.Fatal("evidence phase contract lost")
			}
		})
	}
}

func TestGitBreakingReferenceMustBelongToSelectedGroup(t *testing.T) {
	p := candidatePrepared(t)
	path := "two.go"
	p.Document.Files[1].NewPath = &path
	p.Document.Files[1].Opaque = false
	p.Document.Files[1].Mode = syntax.ModeRawDiff
	p.Document.Files[1].RawDiff = p.Document.Files[0].GroupingRawDiff
	client := &candidateClient{responses: []string{
		`{"F1":"G001","F2":"G002"}`,
		`{"G001":{"type":"fix","breaking_evidence_ref":"git:F2:B1"},"G002":{"type":"fix","breaking_evidence_ref":"none"}}`,
	}}
	result, err := (ThreePhaseGenerator{Client: client}).Generate(context.Background(), p, English, SensitiveValues{})
	if err == nil || client.calls != 2 || len(result.Plan.Commits) != 0 {
		t.Fatal("foreign group's evidence accepted", result, err)
	}
}
