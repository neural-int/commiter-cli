package contextinput

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type crossLanguageFixture struct {
	language string
	path     string
	build    func(int) string
	wants    []evidenceExpectation
}

type evidenceExpectation struct {
	name  string
	match func(syntax.Evidence) bool
}

func TestPrepareReducesStructuralEvidenceAcrossAllSupportedLanguages(t *testing.T) {
	fixtures := crossLanguageFixtures()
	document := Document{
		SchemaVersion: SchemaVersion,
		Repository:    Repository{Head: "head", Branch: "main", IndexIdentity: "index"},
	}
	lineCounts := make(map[string]int, len(fixtures))

	for index, fixture := range fixtures {
		source := fixture.build(4)
		lines := strings.Count(source, "\n")
		analysis := syntax.Analyze(syntax.Input{
			Language: fixture.language,
			Content:  []byte(source),
			Hunks:    []syntax.Hunk{{StartLine: 1, EndLine: lines}},
		})
		if analysis.Mode != syntax.ModeStructural || len(analysis.Evidence) == 0 {
			t.Fatalf("%s analysis = %#v", fixture.language, analysis)
		}
		for _, want := range fixture.wants {
			if !containsEvidence(analysis.Evidence, want.match) {
				t.Fatalf("%s lost %s before reduction: %#v", fixture.language, want.name, analysis.Evidence)
			}
		}
		path := fixture.path
		document.Files = append(document.Files, File{
			ID: fmt.Sprintf("F%03d", index+1), Status: "A", NewPath: &path,
			Language: fixture.language, ChangeHash: fmt.Sprintf("hash-%03d", index+1),
			Mode: syntax.ModeStructural, Evidence: analysis.Evidence,
		})
		lineCounts[fixture.language] = lines
	}

	originalPrompt, err := JSONRenderer(document)
	if err != nil {
		t.Fatal(err)
	}
	originalEstimated := len(originalPrompt) + TemplateReserve + maxInt(MinimumOutputSpace, OutputPerFile*len(document.Files))
	if originalEstimated <= Context32K {
		t.Fatalf("cross-language fixture no longer exceeds 32K: prompt=%d estimated=%d", len(originalPrompt), originalEstimated)
	}

	config := BudgetConfig{Context: "32k", MaxContextTokens: Context32K}
	prepared, err := Prepare(context.Background(), document, config, JSONRenderer, nil)
	if err != nil {
		t.Fatalf("cross-language reduction failed: %v", err)
	}
	if prepared.SummaryStage != SummaryChunk || prepared.SummaryCount != 3 {
		t.Fatalf("raw-diff summary stages were not exhausted before reduction: %#v", prepared)
	}
	if prepared.EvidenceReductionCount != 1 || prepared.Budget.EstimatedTokens > Context32K {
		t.Fatalf("budget result = %#v", prepared)
	}
	if prepared.OriginalPromptBytes != len(originalPrompt) || prepared.EvidenceBeforeBytes <= prepared.EvidenceAfterBytes {
		t.Fatalf("prompt/evidence measurements = original_prompt=%d/%d evidence=%d->%d", prepared.OriginalPromptBytes, len(originalPrompt), prepared.EvidenceBeforeBytes, prepared.EvidenceAfterBytes)
	}
	if err := ValidatePreserved(document, prepared.Document); err != nil {
		t.Fatalf("reduced document was not preserved: %v", err)
	}

	for index, file := range prepared.Document.Files {
		fixture := fixtures[index]
		reduction := file.EvidenceReduction
		if reduction == nil {
			t.Fatalf("%s has no reduction audit", fixture.language)
		}
		if reduction.OriginalCount < reduction.RetainedCount || reduction.OriginalBytes < reduction.RetainedBytes {
			t.Fatalf("%s reduction grew evidence: %#v", fixture.language, reduction)
		}
		if reduction.OriginalCount == reduction.RetainedCount && reduction.OriginalBytes == reduction.RetainedBytes {
			t.Fatalf("%s evidence was not reduced: %#v", fixture.language, reduction)
		}
		if reduction.OriginalCoverageDigest != reduction.RetainedCoverageDigest {
			t.Fatalf("%s lost required coverage: %#v", fixture.language, reduction)
		}
		for _, want := range fixture.wants {
			if !containsEvidence(file.Evidence, want.match) {
				t.Fatalf("%s lost %s after reduction: %#v", fixture.language, want.name, file.Evidence)
			}
		}
		maxEndLine := 0
		for _, evidence := range file.Evidence {
			if evidence.EndLine > maxEndLine {
				maxEndLine = evidence.EndLine
			}
		}
		if maxEndLine < lineCounts[fixture.language]-5 {
			t.Fatalf("%s evidence is biased to the beginning: max_end_line=%d lines=%d", fixture.language, maxEndLine, lineCounts[fixture.language])
		}
		t.Logf("%s: source_bytes=%d lines=%d evidence=%d->%d bytes=%d->%d level=%s coverage=%d", fixture.language, len(fixture.build(4)), lineCounts[fixture.language], reduction.OriginalCount, reduction.RetainedCount, reduction.OriginalBytes, reduction.RetainedBytes, reduction.Level, reduction.CoverageCount)
	}
	t.Logf("planning input: prompt_bytes=%d estimated_tokens=%d selected_context=%d summary_stage=%s reduction_count=%d validation=success", prepared.Budget.PromptBytes, prepared.Budget.EstimatedTokens, prepared.Budget.ContextTokens, prepared.SummaryStage, prepared.EvidenceReductionCount)

	repeated, err := Prepare(context.Background(), document, config, JSONRenderer, nil)
	if err != nil || !bytes.Equal(prepared.Prompt, repeated.Prompt) || !reflect.DeepEqual(prepared.Document, repeated.Document) {
		t.Fatalf("cross-language reduction is not deterministic: error=%v", err)
	}
}

func TestPreparePreservesEvidenceAcrossSeparatedHunks(t *testing.T) {
	source, hunks := separatedGoFixture()
	analysis := syntax.Analyze(syntax.Input{Language: "go", Content: []byte(source), Hunks: hunks})
	if len(hunks) != 3 || analysis.Mode != syntax.ModeStructural || len(analysis.Evidence) == 0 {
		t.Fatalf("hunks=%d mode=%s evidence=%d", len(hunks), analysis.Mode, len(analysis.Evidence))
	}

	path := "separated.go"
	document := Document{
		SchemaVersion: SchemaVersion,
		Repository:    Repository{Head: "head", Branch: "main", IndexIdentity: "index"},
		Files: []File{{
			ID: "F001", Status: "A", NewPath: &path, Language: "go", ChangeHash: "separated-hunks",
			Mode: syntax.ModeStructural, Evidence: analysis.Evidence,
		}},
	}
	originalPrompt, err := JSONRenderer(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(originalPrompt)+TemplateReserve+MinimumOutputSpace <= Context32K {
		t.Fatalf("separated-hunk fixture no longer reproduces overflow: prompt bytes=%d", len(originalPrompt))
	}

	prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "32k", MaxContextTokens: Context32K}, JSONRenderer, nil)
	if err != nil {
		t.Fatal(err)
	}
	reduction := prepared.Document.Files[0].EvidenceReduction
	if reduction == nil || reduction.OriginalCount == reduction.RetainedCount && reduction.OriginalBytes == reduction.RetainedBytes {
		t.Fatalf("separated-hunk evidence was not reduced: %#v", reduction)
	}
	if reduction.OriginalCoverageDigest != reduction.RetainedCoverageDigest {
		t.Fatalf("separated-hunk coverage changed: %#v", reduction)
	}
	for _, name := range []string{"FirstHunk", "MiddleHunk", "LastHunk"} {
		if !containsEvidence(prepared.Document.Files[0].Evidence, func(value syntax.Evidence) bool {
			return value.Kind == "function_declaration" && value.Name == name
		}) {
			t.Fatalf("%s declaration was not retained: %#v", name, prepared.Document.Files[0].Evidence)
		}
	}
}

func TestPrepareDoesNotReduceSmallChangesAcrossAllSupportedLanguages(t *testing.T) {
	fixtures := crossLanguageFixtures()
	for index, fixture := range fixtures {
		source := fixture.build(1)
		lines := strings.Count(source, "\n")
		analysis := syntax.Analyze(syntax.Input{
			Language: fixture.language,
			Content:  []byte(source),
			Hunks:    []syntax.Hunk{{StartLine: 1, EndLine: lines}},
		})
		if analysis.Mode != syntax.ModeStructural || len(analysis.Evidence) == 0 {
			t.Fatalf("%s analysis = %#v", fixture.language, analysis)
		}
		path := fixture.path
		document := Document{
			SchemaVersion: SchemaVersion,
			Repository:    Repository{Head: "head", Branch: "main", IndexIdentity: "index"},
			Files: []File{{
				ID: fmt.Sprintf("F%03d", index+1), Status: "A", NewPath: &path,
				Language: fixture.language, ChangeHash: fmt.Sprintf("small-hash-%03d", index+1),
				Mode: syntax.ModeStructural, Evidence: analysis.Evidence,
			}},
		}
		prepared, err := Prepare(context.Background(), document, BudgetConfig{Context: "auto", MaxContextTokens: Context32K}, JSONRenderer, nil)
		if err != nil {
			t.Fatal(err)
		}
		if prepared.SummaryStage != SummaryNone || prepared.SummaryCount != 0 || prepared.EvidenceReductionCount != 0 {
			t.Fatalf("small %s change was summarized or reduced: %#v", fixture.language, prepared)
		}
		if prepared.Document.Files[0].EvidenceReduction != nil {
			t.Fatalf("small change %s has reduction audit: %#v", fixture.language, prepared.Document.Files[0].EvidenceReduction)
		}
	}
}

func crossLanguageFixtures() []crossLanguageFixture {
	return []crossLanguageFixture{
		{
			language: "go", path: "fixture.go",
			build: func(count int) string {
				var source strings.Builder
				source.WriteString("package fixture\n\nimport \"fmt\"\n\n")
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, "type Item%03d struct { Value string }\n", index)
					fmt.Fprintf(&source, "func Function%03d(value string) string { return fmt.Sprintf(\"%%s-%%d\", value, %d%s) }\n", index, index, optionalMarkers("go", index, 16))
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "function declaration", match: func(value syntax.Evidence) bool { return value.Kind == "function_declaration" }},
				{name: "call", match: func(value syntax.Evidence) bool { return value.Role == "call" }},
			},
		},
		{
			language: "javascript", path: "fixture.js",
			build: func(count int) string {
				var source strings.Builder
				source.WriteString("import { api } from \"./api\";\n\n")
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, "export function function%03d(value) { return api.call(value%s); }\n", index, optionalMarkers("javascript", index, 16))
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "export", match: func(value syntax.Evidence) bool { return value.Role == "export" }},
				{name: "call", match: func(value syntax.Evidence) bool { return value.Role == "call" }},
			},
		},
		{
			language: "jsx", path: "fixture.jsx",
			build: func(count int) string {
				var source strings.Builder
				source.WriteString("import { render } from \"./render\";\n\n")
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, "export function View%03d(value) { return render(<section data-index=\"%03d\"><span>{value}</span></section>%s); }\n", index, index, optionalMarkers("jsx", index, 16))
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "export", match: func(value syntax.Evidence) bool { return value.Role == "export" }},
				{name: "call", match: func(value syntax.Evidence) bool { return value.Role == "call" }},
				{name: "JSX structure", match: func(value syntax.Evidence) bool { return strings.HasPrefix(value.Kind, "jsx_") }},
			},
		},
		{
			language: "typescript", path: "fixture.ts",
			build: func(count int) string {
				var source strings.Builder
				source.WriteString("import { api } from \"./api\";\n\n")
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, "export interface Item%03d { value: string }\n", index)
					fmt.Fprintf(&source, "export function function%03d(value: string): string { return api.call(value%s); }\n", index, optionalMarkers("typescript", index, 16))
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "interface declaration", match: func(value syntax.Evidence) bool { return strings.Contains(value.Kind, "interface") }},
				{name: "export", match: func(value syntax.Evidence) bool { return value.Role == "export" }},
				{name: "call", match: func(value syntax.Evidence) bool { return value.Role == "call" }},
			},
		},
		{
			language: "tsx", path: "fixture.tsx",
			build: func(count int) string {
				var source strings.Builder
				source.WriteString("import { render } from \"./render\";\n\n")
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, "export function View%03d(value: string) { return render(<section data-index=\"%03d\"><span>{value}</span></section>%s); }\n", index, index, optionalMarkers("tsx", index, 16))
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "export", match: func(value syntax.Evidence) bool { return value.Role == "export" }},
				{name: "call", match: func(value syntax.Evidence) bool { return value.Role == "call" }},
				{name: "TSX structure", match: func(value syntax.Evidence) bool { return strings.HasPrefix(value.Kind, "jsx_") }},
			},
		},
		{
			language: "python", path: "fixture.py",
			build: func(count int) string {
				var source strings.Builder
				source.WriteString("from lib import api\n\n")
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, "class Item%03d:\n    def __init__(self, value):\n        self.value = value\n\n", index)
					fmt.Fprintf(&source, "def function%03d(value):\n    return api.call(value%s)\n\n", index, optionalMarkers("python", index, 16))
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "class declaration", match: func(value syntax.Evidence) bool { return value.Kind == "class_definition" }},
				{name: "function definition", match: func(value syntax.Evidence) bool { return value.Kind == "function_definition" }},
				{name: "call", match: func(value syntax.Evidence) bool { return value.Role == "call" }},
			},
		},
		{
			language: "rust", path: "fixture.rs",
			build: func(count int) string {
				var source strings.Builder
				source.WriteString("use crate::api;\n\n")
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, "pub struct Item%03d { value: String }\n", index)
					fmt.Fprintf(&source, "pub fn function%03d(value: &str) -> String { api::call(value%s) }\n", index, optionalMarkers("rust", index, 16))
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "struct", match: func(value syntax.Evidence) bool { return strings.Contains(value.Kind, "struct") }},
				{name: "function", match: func(value syntax.Evidence) bool { return value.Kind == "function_item" }},
				{name: "call", match: func(value syntax.Evidence) bool { return value.Role == "call" }},
			},
		},
		{
			language: "html", path: "fixture.html",
			build: func(count int) string {
				var source strings.Builder
				source.WriteString("<!doctype html>\n")
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, "<fixture-item-%03d data-index=\"%03d\"><span>value</span></fixture-item-%03d>\n", index, index, index)
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "tag", match: func(value syntax.Evidence) bool { return value.Kind == "tag_name" }},
				{name: "attribute", match: func(value syntax.Evidence) bool { return value.Kind == "attribute" || value.Kind == "attribute_name" }},
			},
		},
		{
			language: "css", path: "fixture.css",
			build: func(count int) string {
				var source strings.Builder
				for index := 0; index < count; index++ {
					fmt.Fprintf(&source, ".component-%03d { color: red; padding: %dpx; }\n", index, index)
				}
				return source.String()
			},
			wants: []evidenceExpectation{
				{name: "selector", match: func(value syntax.Evidence) bool { return strings.Contains(value.Kind, "selector") }},
				{name: "property", match: func(value syntax.Evidence) bool { return value.Kind == "property" || value.Kind == "property_name" }},
			},
		},
	}
}

func separatedGoFixture() (string, []syntax.Hunk) {
	var source strings.Builder
	source.WriteString("package fixture\n\nimport \"fmt\"\n\n")
	hunks := make([]syntax.Hunk, 0, 3)
	for block, name := range []string{"FirstHunk", "MiddleHunk", "LastHunk"} {
		startLine := strings.Count(source.String(), "\n") + 1
		fmt.Fprintf(&source, "func %s(value string) string { return fmt.Sprintf(\"%%s-anchor\", value) }\n", name)
		for filler := 0; filler < 24; filler++ {
			fmt.Fprintf(&source, "func filler_%d_%02d(value string) string { return fmt.Sprintf(\"%%s-%%d%s\", value, %d) }\n", block, filler, optionalMarkers("hunk", block, 18), filler)
		}
		endLine := strings.Count(source.String(), "\n")
		hunks = append(hunks, syntax.Hunk{StartLine: startLine, EndLine: endLine})
		source.WriteString("\n")
	}
	return source.String(), hunks
}

func optionalMarkers(language string, index, count int) string {
	var markers strings.Builder
	for marker := 0; marker < count; marker++ {
		fmt.Fprintf(&markers, ", marker_%s_%03d_%02d", language, index, marker)
	}
	return markers.String()
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
