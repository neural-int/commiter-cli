package contextinput

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

func TestCanonicalEvidenceCoalescesEquivalentFacts(t *testing.T) {
	input := []syntax.Evidence{
		{Kind: "identifier", EnclosingDeclaration: "save", StartLine: 1, EndLine: 1, StartByte: 0, EndByte: 4},
		{Kind: "identifier", EnclosingDeclaration: "save", StartLine: 8, EndLine: 8, StartByte: 70, EndByte: 74},
		{Kind: "call_expression", EnclosingDeclaration: "save", Role: "call", StartLine: 9, EndLine: 9, StartByte: 80, EndByte: 90},
	}

	got := canonicalEvidence(input)
	if len(got) != 2 {
		t.Fatalf("canonical evidence count = %d, want 2: %#v", len(got), got)
	}
	if got[0].StartByte != 70 || got[0].EndByte != 74 || got[0].StartLine != 8 || got[0].EndLine != 8 {
		t.Fatalf("representative fact = %#v", got[0])
	}
}

func TestEvidenceSelectionPreservesCoverageAndSpansFile(t *testing.T) {
	values := []syntax.Evidence{
		{Kind: "function_declaration", Name: "first", StartLine: 1, EndLine: 10, StartByte: 0, EndByte: 100},
		{Kind: "identifier", EnclosingDeclaration: "first", StartLine: 2, EndLine: 2, StartByte: 10, EndByte: 20},
		{Kind: "call_expression", EnclosingDeclaration: "first", Role: "call", StartLine: 8, EndLine: 8, StartByte: 80, EndByte: 90},
		{Kind: "identifier", StartLine: 50, EndLine: 50, StartByte: 500, EndByte: 510},
		{Kind: "function_declaration", Name: "last", StartLine: 90, EndLine: 100, StartByte: 900, EndByte: 1000},
		{Kind: "call_expression", EnclosingDeclaration: "last", Role: "call", StartLine: 98, EndLine: 98, StartByte: 980, EndByte: 990},
	}
	retained := selectEvidence(values, requiredEvidenceCount(values))
	reduction := newEvidenceReduction("strong", values, retained)
	if err := validateEvidencePreserved(values, retained, reduction); err != nil {
		t.Fatal(err)
	}
	if !containsEvidence(retained, func(value syntax.Evidence) bool { return value.Name == "first" }) ||
		!containsEvidence(retained, func(value syntax.Evidence) bool { return value.Name == "last" }) {
		t.Fatalf("declaration coverage was lost: %#v", retained)
	}
}

func TestEvidenceReductionWorksForGoWithoutLanguageRules(t *testing.T) {
	source := []byte("package p\nimport \"fmt\"\nfunc first() { fmt.Println(1) }\nfunc second() { fmt.Println(2) }\n")
	analysis := syntax.Analyze(syntax.Input{Language: "go", Content: source, Hunks: []syntax.Hunk{{StartLine: 1, EndLine: 4}}})
	if analysis.Mode != syntax.ModeStructural {
		t.Fatalf("analysis = %#v", analysis)
	}
	document := Document{Files: []File{{ID: "F001", Mode: syntax.ModeStructural, Evidence: analysis.Evidence}}}
	reduced := reduceEvidence(document, 0)
	if err := ValidatePreserved(document, reduced); err != nil {
		t.Fatal(err)
	}
	if reduced.Files[0].EvidenceReduction == nil || reduced.Files[0].EvidenceReduction.OriginalCoverageDigest != reduced.Files[0].EvidenceReduction.RetainedCoverageDigest {
		t.Fatalf("reduction metadata = %#v", reduced.Files[0].EvidenceReduction)
	}
}

func TestValidatePreservedRejectsEvidenceReductionTampering(t *testing.T) {
	original := Document{Files: []File{{ID: "F001", Mode: syntax.ModeStructural, Evidence: []syntax.Evidence{
		{Kind: "function_declaration", Name: "first", StartLine: 1, EndLine: 3, StartByte: 0, EndByte: 30},
		{Kind: "call_expression", EnclosingDeclaration: "first", Role: "call", StartLine: 2, EndLine: 2, StartByte: 10, EndByte: 20},
	}}}}
	valid := reduceEvidence(original, 0)
	if err := ValidatePreserved(original, valid); err != nil {
		t.Fatal(err)
	}

	for name, mutate := range map[string]func(Document){
		"metadata": func(document Document) { document.Files[0].EvidenceReduction.OriginalCount++ },
		"coverage": func(document Document) { document.Files[0].Evidence = document.Files[0].Evidence[:1] },
		"range":    func(document Document) { document.Files[0].Evidence[0].EndByte++ },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := cloneDocument(valid)
			mutate(candidate)
			if err := ValidatePreserved(original, candidate); err == nil {
				t.Fatal("tampered reduction was accepted")
			}
		})
	}
}

func TestPrepareReducesHTMLCSSAndJavaScriptEvidenceWithin32K(t *testing.T) {
	document := inflatedWebFixture(t)
	originalPrompt, err := JSONRenderer(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(originalPrompt)+TemplateReserve+MinimumOutputSpace <= Context32K {
		t.Fatalf("fixture no longer reproduces overflow: prompt bytes = %d", len(originalPrompt))
	}

	prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "32k", MaxContextTokens: Context32K}, JSONRenderer, nil)
	if err != nil {
		minimum := reduceEvidence(document, 0)
		minimumPrompt, _ := JSONRenderer(minimum)
		counts := make([]string, 0, 3)
		for _, file := range minimum.Files[:3] {
			counts = append(counts, fmt.Sprintf("%s:%d->%d", file.ID, file.EvidenceReduction.OriginalCount, file.EvidenceReduction.RetainedCount))
		}
		t.Fatalf("%v; minimum prompt=%d counts=%v", err, len(minimumPrompt), counts)
	}
	if prepared.Budget.EstimatedTokens > Context32K || prepared.EvidenceReductionCount != 1 {
		t.Fatalf("prepared = %#v", prepared)
	}
	if prepared.SummaryStage != SummaryChunk || prepared.SummaryCount != 3 {
		t.Fatalf("raw diff stages were not exhausted before evidence reduction: %#v", prepared)
	}
	if prepared.EvidenceBeforeBytes <= prepared.EvidenceAfterBytes || prepared.OriginalPromptBytes != len(originalPrompt) {
		t.Fatalf("before/after bytes = %d/%d, original prompt = %d/%d", prepared.EvidenceBeforeBytes, prepared.EvidenceAfterBytes, prepared.OriginalPromptBytes, len(originalPrompt))
	}

	for _, file := range prepared.Document.Files[:3] {
		if file.EvidenceReduction == nil {
			t.Fatalf("file %s was not audited", file.ID)
		}
		if file.EvidenceReduction.OriginalCoverageDigest != file.EvidenceReduction.RetainedCoverageDigest {
			t.Fatalf("file %s lost coverage", file.ID)
		}
	}
	css := prepared.Document.Files[1].Evidence
	if !containsEvidence(css, func(value syntax.Evidence) bool { return value.Kind == "class_selector" && value.StartLine > 60 }) {
		t.Fatalf("CSS selection is biased to the beginning: %#v", css)
	}
	javascript := prepared.Document.Files[2].Evidence
	if !containsEvidence(javascript, func(value syntax.Evidence) bool { return value.Role == "export" }) ||
		!containsEvidence(javascript, func(value syntax.Evidence) bool { return value.Role == "call" }) {
		t.Fatalf("JavaScript roles were lost: %#v", javascript)
	}

	repeated, err := Prepare(context.Background(), document, BudgetConfig{Context: "32k", MaxContextTokens: Context32K}, JSONRenderer, nil)
	if err != nil || !bytes.Equal(prepared.Prompt, repeated.Prompt) || !reflect.DeepEqual(prepared.Document, repeated.Document) {
		t.Fatalf("reduction is not deterministic: error=%v", err)
	}
	tighter, err := Prepare(context.Background(), document, BudgetConfig{Context: "32k", MaxContextTokens: Context32K, PromptOverheadBytes: 1024}, JSONRenderer, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tighter.EvidenceAfterBytes >= prepared.EvidenceAfterBytes {
		t.Fatalf("tighter budget did not reduce evidence further: reserved=%d base=%d", tighter.EvidenceAfterBytes, prepared.EvidenceAfterBytes)
	}
	t.Logf("planning baseline: prompt=%d reduced_prompt=%d evidence_bytes=%d->%d (reserved=%d)", len(originalPrompt), prepared.Budget.PromptBytes, prepared.EvidenceBeforeBytes, prepared.EvidenceAfterBytes, tighter.EvidenceAfterBytes)
}

func TestPrepareFailsClosedWhenRequiredDeclarationCoverageCannotFit(t *testing.T) {
	evidence := make([]syntax.Evidence, 0, 500)
	for index := 0; index < 500; index++ {
		evidence = append(evidence, syntax.Evidence{
			Kind: "function_declaration", Name: fmt.Sprintf("function_with_a_deliberately_long_unique_name_%04d", index),
			StartLine: index + 1, EndLine: index + 1, StartByte: uint(index * 100), EndByte: uint(index*100 + 90),
		})
	}
	path := "many.go"
	document := Document{SchemaVersion: SchemaVersion, Repository: Repository{Head: "head", Branch: "main", IndexIdentity: "index"}, Files: []File{{
		ID: "F001", Status: "?", NewPath: &path, Language: "go", ChangeHash: "hash", Mode: syntax.ModeStructural, Evidence: evidence,
	}}}
	prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "32k", MaxContextTokens: Context32K}, JSONRenderer, nil)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("prepared=%#v error=%v", prepared, err)
	}
	if len(document.Files[0].Evidence) != 500 || document.Files[0].EvidenceReduction != nil {
		t.Fatal("failed reduction mutated its input")
	}
}

func inflatedWebFixture(t *testing.T) Document {
	t.Helper()
	var html, css, javascript strings.Builder
	for index := 0; index < 112; index++ {
		fmt.Fprintf(&html, "<b data-i=\"%03d\"></b>\n", index)
	}
	for index := 0; index < 120; index++ {
		fmt.Fprintf(&css, ".i%03d{color:red}\n", index)
	}
	for index := 0; index < 40; index++ {
		fmt.Fprintf(&javascript, "export function f%03d(){x()}\n", index)
	}
	inputs := []struct {
		id, path, language string
		source             []byte
		lines              int
	}{
		{"F001", "index.html", "html", []byte(html.String()), 112},
		{"F002", "css/styles.css", "css", []byte(css.String()), 120},
		{"F003", "js/app.js", "javascript", []byte(javascript.String()), 40},
	}
	rawSource := "fixture-line-x\n" + strings.Repeat("fixture-line\n", 7)
	totalLines, totalBytes := 16, len(rawSource)*2
	document := Document{SchemaVersion: SchemaVersion, Repository: Repository{Head: "head", Branch: "main", IndexIdentity: "index"}}
	for _, input := range inputs {
		totalLines += input.lines
		totalBytes += len(input.source)
		analysis := syntax.Analyze(syntax.Input{Language: input.language, Content: input.source, Hunks: []syntax.Hunk{{StartLine: 1, EndLine: input.lines}}})
		if analysis.Mode != syntax.ModeStructural || len(analysis.Evidence) == 0 {
			t.Fatalf("%s analysis = %#v", input.path, analysis)
		}
		path := input.path
		document.Files = append(document.Files, File{ID: input.id, Status: "?", NewPath: &path, Language: input.language, ChangeHash: "hash-" + input.id, Mode: syntax.ModeStructural, Evidence: analysis.Evidence})
	}
	if totalLines != 288 {
		t.Fatalf("fixture lines = %d, want 288", totalLines)
	}
	if totalBytes != 5724 {
		t.Fatalf("fixture source bytes = %d, want 5724", totalBytes)
	}
	t.Logf("web fixture baseline: lines=%d source_bytes=%d evidence=%d/%d/%d", totalLines, totalBytes, len(document.Files[0].Evidence), len(document.Files[1].Evidence), len(document.Files[2].Evidence))
	for index, item := range []struct{ id, path string }{{"F004", "README.md"}, {"F005", ".gitignore"}} {
		path := item.path
		rawDiff := "@@ -0,0 +1,8 @@\n+" + strings.ReplaceAll(strings.TrimSuffix(rawSource, "\n"), "\n", "\n+") + "\n"
		document.Files = append(document.Files, File{ID: item.id, Status: "?", NewPath: &path, ChangeHash: fmt.Sprintf("raw-%d", index), Mode: syntax.ModeRawDiff, RawDiff: rawDiff})
	}
	return document
}

func containsEvidence(values []syntax.Evidence, match func(syntax.Evidence) bool) bool {
	for _, value := range values {
		if match(value) {
			return true
		}
	}
	return false
}
