package planning

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestPlanningRendererUsesReducedStructuralEvidence(t *testing.T) {
	var html, css, javascript strings.Builder
	for index := 0; index < 100; index++ {
		fmt.Fprintf(&html, "<b data-i=\"%03d\"></b>\n", index)
		fmt.Fprintf(&css, ".i%03d{color:red}\n", index)
	}
	for index := 0; index < 30; index++ {
		fmt.Fprintf(&javascript, "export function f%03d(){x()}\n", index)
	}
	inputs := []struct {
		id, path, language string
		source             []byte
		lines              int
	}{
		{"F001", "index.html", "html", []byte(html.String()), 100},
		{"F002", "styles.css", "css", []byte(css.String()), 100},
		{"F003", "app.js", "javascript", []byte(javascript.String()), 30},
	}
	document := contextinput.Document{SchemaVersion: contextinput.SchemaVersion, Repository: contextinput.Repository{Head: "head", Branch: "main", IndexIdentity: "index"}}
	for _, input := range inputs {
		analysis := syntax.Analyze(syntax.Input{Language: input.language, Content: input.source, Hunks: []syntax.Hunk{{StartLine: 1, EndLine: input.lines}}})
		if analysis.Mode != syntax.ModeStructural {
			t.Fatalf("%s analysis = %#v", input.path, analysis)
		}
		path := input.path
		document.Files = append(document.Files, contextinput.File{ID: input.id, Status: "?", NewPath: &path, Language: input.language, ChangeHash: "hash-" + input.id, Mode: syntax.ModeStructural, Evidence: analysis.Evidence})
	}
	render := Renderer(Japanese)
	original, err := render(document)
	if err != nil {
		t.Fatal(err)
	}
	systemMessage, err := InitialSystemMessage([]string{"F001", "F002", "F003"})
	if err != nil {
		t.Fatal(err)
	}
	config := contextinput.BudgetConfig{Context: "32k", MaxContextTokens: contextinput.Context32K, PromptOverheadBytes: len(systemMessage)}
	if _, err := contextinput.SelectContext(original, len(document.Files), config); err == nil {
		t.Fatalf("fixture no longer exceeds 32K: prompt=%d", len(original))
	}
	prepared, err := contextinput.Prepare(context.Background(), document, config, render, nil)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.EvidenceReductionCount != 1 || prepared.Budget.EstimatedTokens > contextinput.Context32K {
		t.Fatalf("prepared = %#v", prepared)
	}
}
