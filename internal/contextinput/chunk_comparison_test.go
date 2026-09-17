package contextinput

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

type comparisonLine = summaryLine

type comparisonLineID = summaryLineID

type comparisonProfile struct {
	name         string
	quota        int
	excerptBytes int
}

type comparisonFamily struct {
	name     string
	profiles []comparisonProfile
}

type comparisonResult struct {
	summary       string
	selected      map[comparisonLineID]bool
	digests       []string
	omittedAdded  int
	omittedRemove int
	promptBytes   int
	estimated     int
	fits          bool
}

var comparisonFamilies = []comparisonFamily{
	{name: "coverage", profiles: []comparisonProfile{
		{name: "light", quota: 9, excerptBytes: 256},
		{name: "medium", quota: 6, excerptBytes: 192},
		{name: "strong", quota: 3, excerptBytes: 128},
	}},
	{name: "balanced", profiles: []comparisonProfile{
		{name: "light", quota: 7, excerptBytes: 224},
		{name: "medium", quota: 5, excerptBytes: 176},
		{name: "strong", quota: 3, excerptBytes: 128},
	}},
	{name: "compact", profiles: []comparisonProfile{
		{name: "light", quota: 5, excerptBytes: 192},
		{name: "medium", quota: 4, excerptBytes: 160},
		{name: "strong", quota: 3, excerptBytes: 128},
	}},
}

func TestComparisonPreprocessingMatchesSummaryHunk(t *testing.T) {
	raw := "diff --git a/main.txt b/main.txt\nindex 123..456 100644\n--- a/main.txt\n+++ b/main.txt\n@@ -1,4 +1,4 @@\n unchanged\n-old\n+new\n--- changed comment\n+++ changed counter\n"
	want := "@@ -1,4 +1,4 @@\n-old\n+new\n--- changed comment\n+++ changed counter"
	fileSummary, err := NewHierarchicalSummarizer().Summarize(context.Background(), SummaryFile, rawDocument(raw))
	if err != nil {
		t.Fatal(err)
	}
	hunkSummary, err := NewHierarchicalSummarizer().Summarize(context.Background(), SummaryHunk, fileSummary)
	if err != nil {
		t.Fatal(err)
	}
	if got := hunkSummary.Files[0].Summary; got != want {
		t.Fatalf("production preprocessing diverged\nwant: %q\n got: %q", want, got)
	}
	if got := comparisonText(comparisonPreprocess(raw)); got != want {
		t.Fatalf("comparison preprocessing diverged\nwant: %q\n got: %q", want, got)
	}
}

func TestComparisonSamplerInvariants(t *testing.T) {
	raw := mixedComparisonFixture(12, 7)
	lines := comparisonPreprocess(raw)
	profiles := comparisonFamilies[1].profiles
	results := make([]comparisonResult, 0, len(profiles))
	for _, profile := range profiles {
		first := runComparisonSampler(t, lines, 48, profile)
		second := runComparisonSampler(t, lines, 48, profile)
		if first.summary != second.summary || !sameLineIDs(first.selected, second.selected) {
			t.Fatalf("profile %s is not deterministic", profile.name)
		}
		if first.omittedAdded == 0 || first.omittedRemove == 0 {
			t.Fatalf("profile %s did not report both omission signs: %#v", profile.name, first)
		}
		original := rawDocument(raw)
		candidate := cloneDocument(original)
		candidate.Files[0].RawDiff = ""
		candidate.Files[0].Summary = first.summary
		if err := ValidatePreserved(original, candidate); err != nil {
			t.Fatalf("profile %s changed required document data: %v", profile.name, err)
		}
		selectedAdded, selectedRemoved := selectedSignCounts(lines, first.selected)
		if selectedAdded == 0 || selectedRemoved == 0 || selectedAdded > profile.quota*len(first.digests) || selectedRemoved > profile.quota*len(first.digests) {
			t.Fatalf("profile %s violated independent sign quotas: additions=%d deletions=%d chunks=%d", profile.name, selectedAdded, selectedRemoved, len(first.digests))
		}
		added, removed := changedSignCounts(lines)
		if selectedAdded+first.omittedAdded != added || selectedRemoved+first.omittedRemove != removed {
			t.Fatalf("profile %s omission counts do not reconcile: selected=%d/%d omitted=%d/%d total=%d/%d", profile.name, selectedAdded, selectedRemoved, first.omittedAdded, first.omittedRemove, added, removed)
		}
		results = append(results, first)
	}
	assertSubset(t, results[2].selected, results[1].selected, "strong", "medium")
	assertSubset(t, results[1].selected, results[0].selected, "medium", "light")
	if !(len(results[2].summary) <= len(results[1].summary) && len(results[1].summary) <= len(results[0].summary)) {
		t.Fatalf("summary bytes are not monotonic: light=%d medium=%d strong=%d", len(results[0].summary), len(results[1].summary), len(results[2].summary))
	}
	if !(results[2].estimated <= results[1].estimated && results[1].estimated <= results[0].estimated) {
		t.Fatalf("estimated tokens are not monotonic: light=%d medium=%d strong=%d", results[0].estimated, results[1].estimated, results[2].estimated)
	}
	if fmt.Sprint(results[0].digests) != fmt.Sprint(results[1].digests) || fmt.Sprint(results[1].digests) != fmt.Sprint(results[2].digests) {
		t.Fatalf("canonical chunk digests changed by profile: %#v", results)
	}
}

func TestAdoptedComparisonCandidate(t *testing.T) {
	family := comparisonFamilies[1]
	chunkSize := 48
	selection := []comparisonFixture{
		{name: "late-meaning", raw: lateMeaningFixture(), sentinels: []string{"ENABLE_AUTHORIZATION_CHECK"}},
		{name: "balanced-signs", raw: balancedSignsFixture(), sentinels: []string{"REMOVE_LEGACY_TOKEN_CHECK", "ADD_ROTATING_TOKEN_CHECK"}},
		{name: "many-hunks", raw: manyHunksFixture(10), sentinels: []string{"HUNK_01", "HUNK_05", "HUNK_10"}},
		{name: "duplicate-lines", raw: duplicateLinesFixture(), sentinels: []string{"UNIQUE_MIDDLE_CHANGE"}},
	}
	for _, fixture := range selection {
		result := runComparisonSampler(t, comparisonPreprocess(fixture.raw), chunkSize, family.profiles[0])
		for _, sentinel := range fixture.sentinels {
			if !strings.Contains(result.summary, sentinel) {
				t.Fatalf("provisional candidate lost selection sentinel %s in %s", sentinel, fixture.name)
			}
		}
	}
	for _, fixture := range []struct {
		lines int
		want  string
	}{{700, "light"}, {1000, "medium"}, {1800, "strong"}, {6000, "unfit"}} {
		if got := firstFittingProfile(t, comparisonPreprocess(sizedComparisonFixture(fixture.lines)), chunkSize, family); got != fixture.want {
			t.Fatalf("provisional candidate lines=%d profile=%s, want %s", fixture.lines, got, fixture.want)
		}
	}
	holdouts := []comparisonFixture{
		{name: "split-replacement", raw: holdoutReplacementFixture(), sentinels: []string{"DEPRECATE_COOKIE_SESSION", "USE_SIGNED_SESSION"}},
		{name: "boundary-hunks", raw: holdoutBoundaryFixture(), sentinels: []string{"FIRST_BOUNDARY_CHANGE", "MIDDLE_BOUNDARY_CHANGE", "LAST_BOUNDARY_CHANGE"}},
	}
	baseline, candidate := 0, 0
	for _, fixture := range holdouts {
		baselineResult := runBaselineSampler(t, comparisonPreprocess(fixture.raw))
		candidateResult := runComparisonSampler(t, comparisonPreprocess(fixture.raw), chunkSize, family.profiles[0])
		for _, sentinel := range fixture.sentinels {
			if strings.Contains(baselineResult.summary, sentinel) {
				baseline++
			}
			if strings.Contains(candidateResult.summary, sentinel) {
				candidate++
			}
		}
	}
	if candidate < baseline {
		t.Fatalf("provisional candidate regressed holdout retention: candidate=%d baseline=%d", candidate, baseline)
	}
	t.Logf("adopted chunk=%d family=%s holdout=%d baseline=%d", chunkSize, family.name, candidate, baseline)
}

func TestPrepareUsesAdoptedCompressionProfiles(t *testing.T) {
	for _, fixture := range []struct {
		lines    int
		profile  CompressionProfile
		count    int
		tooLarge bool
	}{
		{lines: 700, profile: CompressionLight, count: 3},
		{lines: 1000, profile: CompressionMedium, count: 4},
		{lines: 1800, profile: CompressionStrong, count: 5},
		{lines: 6000, profile: CompressionStrong, count: 5, tooLarge: true},
	} {
		prepared, err := Prepare(
			context.Background(), rawDocument(sizedComparisonFixture(fixture.lines)),
			BudgetConfig{Context: "auto", MaxContextTokens: Context32K}, JSONRenderer, nil,
		)
		if fixture.tooLarge {
			if !errors.Is(err, ErrTooLarge) {
				t.Fatalf("lines=%d error=%v, want ErrTooLarge", fixture.lines, err)
			}
			strong := runComparisonSampler(t, comparisonPreprocess(sizedComparisonFixture(fixture.lines)), canonicalChunkLines, comparisonProfile{name: "strong", quota: 3, excerptBytes: 128})
			t.Logf("lines=%d profile=strong prompt_bytes=%d estimated_tokens=%d fit=%v", fixture.lines, strong.promptBytes, strong.estimated, strong.fits)
		} else if err != nil {
			t.Fatalf("lines=%d: %v", fixture.lines, err)
		}
		if prepared.CompressionProfile != fixture.profile || prepared.SummaryCount != fixture.count {
			t.Fatalf("lines=%d profile=%s count=%d, want %s/%d", fixture.lines, prepared.CompressionProfile, prepared.SummaryCount, fixture.profile, fixture.count)
		}
		if !fixture.tooLarge {
			t.Logf("lines=%d profile=%s prompt_bytes=%d estimated_tokens=%d context=%d", fixture.lines, prepared.CompressionProfile, prepared.Budget.PromptBytes, prepared.Budget.EstimatedTokens, prepared.Budget.ContextTokens)
		}
	}
}

func TestComparisonSamplerBalancesHunksWithinChunk(t *testing.T) {
	lines := comparisonPreprocess(manyHunksFixture(10))
	result := runComparisonSampler(t, lines, 32, comparisonProfile{name: "strong", quota: 3, excerptBytes: 128})
	selectedHunks := map[int]bool{}
	byIndex := map[int]int{}
	for _, line := range lines {
		byIndex[line.originalIndex] = line.hunk
	}
	for id := range result.selected {
		selectedHunks[byIndex[id.originalDiffIndex]] = true
	}
	for _, want := range []int{1, 5, 10} {
		if !selectedHunks[want] {
			t.Fatalf("spatially ranked hunk %d was not sampled: %#v", want, selectedHunks)
		}
	}
}

func TestComparisonHeadTailTruncationIsBoundedUTF8(t *testing.T) {
	for _, input := range []string{
		"abcdefghijklmnopqrstuvwxyz0123456789",
		"前半の値を保持しながら後半の重要な値も保持する",
	} {
		for _, limit := range []int{3, 4, 8, 17} {
			got := comparisonTruncate(input, limit)
			if len(got) > limit || !utf8.ValidString(got) {
				t.Fatalf("truncate(%q, %d)=%q bytes=%d valid=%v", input, limit, got, len(got), utf8.ValidString(got))
			}
			if len(input) > limit && limit >= 3 && !strings.Contains(got, "...") {
				t.Fatalf("truncate(%q, %d) omitted marker: %q", input, limit, got)
			}
		}
	}
	if got := comparisonTruncate("abcdefghijklmnopqrstuvwxyz0123456789", 17); !strings.HasPrefix(got, "abcdefghi") || !strings.HasSuffix(got, "56789") {
		t.Fatalf("head-tail truncation did not retain both ends: %q", got)
	}
}

func TestChunkCompressionSelectionMatrix(t *testing.T) {
	selection := []comparisonFixture{
		{name: "late-meaning", raw: lateMeaningFixture(), sentinels: []string{"ENABLE_AUTHORIZATION_CHECK"}},
		{name: "balanced-signs", raw: balancedSignsFixture(), sentinels: []string{"REMOVE_LEGACY_TOKEN_CHECK", "ADD_ROTATING_TOKEN_CHECK"}},
		{name: "many-hunks", raw: manyHunksFixture(10), sentinels: []string{"HUNK_01", "HUNK_05", "HUNK_10"}},
		{name: "duplicate-lines", raw: duplicateLinesFixture(), sentinels: []string{"UNIQUE_MIDDLE_CHANGE"}},
	}
	budgetFixtures := []struct {
		name  string
		lines int
		want  string
	}{
		{name: "just-over", lines: 700, want: "light"},
		{name: "medium-over", lines: 1000, want: "medium"},
		{name: "large-over", lines: 1800, want: "strong"},
		{name: "unfit", lines: 6000, want: "unfit"},
	}
	baselineRetained, baselineTotal := 0, 0
	for _, fixture := range selection {
		result := runBaselineSampler(t, comparisonPreprocess(fixture.raw))
		for _, sentinel := range fixture.sentinels {
			baselineTotal++
			if strings.Contains(result.summary, sentinel) {
				baselineRetained++
			}
		}
	}
	t.Logf("baseline chunk=64 first-lines=2 excerpt=160 sentinel=%d/%d", baselineRetained, baselineTotal)

	for _, chunkSize := range []int{32, 48, 64} {
		for _, family := range comparisonFamilies {
			retained, total := 0, 0
			for _, fixture := range selection {
				result := runComparisonSampler(t, comparisonPreprocess(fixture.raw), chunkSize, family.profiles[0])
				for _, sentinel := range fixture.sentinels {
					total++
					if strings.Contains(result.summary, sentinel) {
						retained++
					}
				}
			}
			stages := make([]string, 0, len(budgetFixtures))
			for _, fixture := range budgetFixtures {
				stage := "unfit"
				lines := comparisonPreprocess(sizedComparisonFixture(fixture.lines))
				for _, profile := range family.profiles {
					if runComparisonSampler(t, lines, chunkSize, profile).fits {
						stage = profile.name
						break
					}
				}
				stages = append(stages, fixture.name+"="+stage+"(target="+fixture.want+")")
			}
			t.Logf("candidate chunk=%d family=%s sentinel=%d/%d %s", chunkSize, family.name, retained, total, strings.Join(stages, " "))
		}
	}
}

func TestChunkCompressionFixedHoldout(t *testing.T) {
	holdouts := []comparisonFixture{
		{name: "split-replacement", raw: holdoutReplacementFixture(), sentinels: []string{"DEPRECATE_COOKIE_SESSION", "USE_SIGNED_SESSION"}},
		{name: "boundary-hunks", raw: holdoutBoundaryFixture(), sentinels: []string{"FIRST_BOUNDARY_CHANGE", "MIDDLE_BOUNDARY_CHANGE", "LAST_BOUNDARY_CHANGE"}},
	}
	baselineRetained, total := 0, 0
	for _, fixture := range holdouts {
		result := runBaselineSampler(t, comparisonPreprocess(fixture.raw))
		for _, sentinel := range fixture.sentinels {
			total++
			if strings.Contains(result.summary, sentinel) {
				baselineRetained++
			}
		}
	}
	t.Logf("holdout baseline sentinel=%d/%d", baselineRetained, total)
	for _, chunkSize := range []int{32, 48, 64} {
		for _, family := range comparisonFamilies {
			retained := 0
			for _, fixture := range holdouts {
				result := runComparisonSampler(t, comparisonPreprocess(fixture.raw), chunkSize, family.profiles[0])
				for _, sentinel := range fixture.sentinels {
					if strings.Contains(result.summary, sentinel) {
						retained++
					}
				}
			}
			t.Logf("holdout candidate chunk=%d family=%s sentinel=%d/%d delta=%+d", chunkSize, family.name, retained, total, retained-baselineRetained)
		}
	}
}

type comparisonFixture struct {
	name      string
	raw       string
	sentinels []string
}

func runComparisonSampler(t *testing.T, lines []comparisonLine, chunkSize int, profile comparisonProfile) comparisonResult {
	t.Helper()
	if chunkSize <= 0 || profile.quota <= 0 || profile.excerptBytes <= 0 {
		t.Fatalf("invalid comparison configuration: chunk=%d profile=%#v", chunkSize, profile)
	}
	compressed := compressSummaryLines(lines, compressionLimits{
		chunkLines: chunkSize, excerptLines: profile.quota, excerptBytes: profile.excerptBytes,
	})
	result := comparisonResult{
		summary: compressed.summary, selected: compressed.selected, digests: compressed.digests,
		omittedAdded: compressed.omittedAdded, omittedRemove: compressed.omittedRemoved,
	}
	document := rawDocument("@@ -1 +1 @@\n-old\n+new\n")
	document.Files[0].RawDiff = ""
	document.Files[0].Summary = result.summary
	prompt, err := JSONRenderer(document)
	if err != nil {
		t.Fatal(err)
	}
	budget, err := SelectContext(prompt, len(document.Files), BudgetConfig{Context: "auto", MaxContextTokens: Context32K})
	if err != nil && !errors.Is(err, ErrTooLarge) {
		t.Fatal(err)
	}
	result.promptBytes = budget.PromptBytes
	result.estimated = budget.EstimatedTokens
	result.fits = err == nil
	return result
}

func runBaselineSampler(t *testing.T, lines []comparisonLine) comparisonResult {
	t.Helper()
	const chunkSize, quota, excerptBytes = 64, 2, 160
	var output []string
	activeHunk := ""
	for offset, number := 0, 1; offset < len(lines); offset, number = offset+chunkSize, number+1 {
		end := min(offset+chunkSize, len(lines))
		chunk := lines[offset:end]
		hunks := make([]string, 0, 2)
		if activeHunk != "" {
			hunks = append(hunks, activeHunk)
		}
		texts := make([]string, len(chunk))
		for index, line := range chunk {
			texts[index] = line.text
			if strings.HasPrefix(line.text, "@@") {
				activeHunk = line.text
				if len(hunks) == 0 || hunks[len(hunks)-1] != line.text {
					hunks = append(hunks, line.text)
				}
			}
		}
		for _, hunk := range hunks {
			output = append(output, "hunk: "+legacyTruncateUTF8(hunk, excerptBytes))
		}
		additions, deletions, contextLines, addedExcerpts, deletedExcerpts := 0, 0, 0, 0, 0
		var excerpts []string
		for _, line := range texts {
			switch {
			case strings.HasPrefix(line, "+"):
				additions++
				if addedExcerpts < quota {
					excerpts = append(excerpts, "added: "+legacyTruncateUTF8(strings.TrimPrefix(line, "+"), excerptBytes))
					addedExcerpts++
				}
			case strings.HasPrefix(line, "-"):
				deletions++
				if deletedExcerpts < quota {
					excerpts = append(excerpts, "removed: "+legacyTruncateUTF8(strings.TrimPrefix(line, "-"), excerptBytes))
					deletedExcerpts++
				}
			case strings.HasPrefix(line, " "):
				contextLines++
			}
		}
		digest := sha256.Sum256([]byte(strings.Join(texts, "\n")))
		output = append(output, fmt.Sprintf("chunk %d: lines=%d additions=%d deletions=%d context=%d sha256=%s", number, len(texts), additions, deletions, contextLines, hex.EncodeToString(digest[:])))
		output = append(output, excerpts...)
	}
	summary := strings.Join(output, "\n")
	document := rawDocument("@@ -1 +1 @@\n-old\n+new\n")
	document.Files[0].RawDiff = ""
	document.Files[0].Summary = summary
	prompt, err := JSONRenderer(document)
	if err != nil {
		t.Fatal(err)
	}
	budget, err := SelectContext(prompt, len(document.Files), BudgetConfig{Context: "auto", MaxContextTokens: Context32K})
	if err != nil && !errors.Is(err, ErrTooLarge) {
		t.Fatal(err)
	}
	return comparisonResult{
		summary: summary, promptBytes: budget.PromptBytes, estimated: budget.EstimatedTokens, fits: err == nil,
	}
}

func firstFittingProfile(t *testing.T, lines []comparisonLine, chunkSize int, family comparisonFamily) string {
	t.Helper()
	for _, profile := range family.profiles {
		if runComparisonSampler(t, lines, chunkSize, profile).fits {
			return profile.name
		}
	}
	return "unfit"
}

func comparisonPreprocess(raw string) []comparisonLine {
	return summarizeHunkLines(summarizeFileLines(indexSummaryLines(raw)))
}

func comparisonText(lines []comparisonLine) string {
	return summaryLinesText(lines)
}

func comparisonTruncate(value string, limit int) string {
	return truncateUTF8(value, limit)
}

func isComparisonChanged(line string) bool {
	return len(line) > 0 && (line[0] == '+' || line[0] == '-') && !strings.HasPrefix(line, "@@")
}

func sameLineIDs(left, right map[comparisonLineID]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for id := range left {
		if !right[id] {
			return false
		}
	}
	return true
}

func selectedSignCounts(lines []comparisonLine, selected map[comparisonLineID]bool) (int, int) {
	byIndex := make(map[int]byte, len(lines))
	for _, line := range lines {
		if isComparisonChanged(line.text) {
			byIndex[line.originalIndex] = line.text[0]
		}
	}
	added, removed := 0, 0
	for id := range selected {
		if byIndex[id.originalDiffIndex] == '+' {
			added++
		} else {
			removed++
		}
	}
	return added, removed
}

func changedSignCounts(lines []comparisonLine) (int, int) {
	added, removed := 0, 0
	for _, line := range lines {
		if !isComparisonChanged(line.text) {
			continue
		}
		if line.text[0] == '+' {
			added++
		} else {
			removed++
		}
	}
	return added, removed
}

func assertSubset(t *testing.T, subset, superset map[comparisonLineID]bool, subsetName, supersetName string) {
	t.Helper()
	for id := range subset {
		if !superset[id] {
			t.Fatalf("%s selected %#v but %s did not", subsetName, id, supersetName)
		}
	}
}

func legacyTruncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value + "..."
}

func lateMeaningFixture() string {
	var raw strings.Builder
	raw.WriteString("diff --git a/auth.rb b/auth.rb\n--- a/auth.rb\n+++ b/auth.rb\n@@ -1,128 +1,128 @@\n")
	for i := 1; i <= 127; i++ {
		fmt.Fprintf(&raw, "+mechanical_format_change_%03d\n", i)
	}
	raw.WriteString("+ENABLE_AUTHORIZATION_CHECK\n")
	return raw.String()
}

func balancedSignsFixture() string {
	var raw strings.Builder
	raw.WriteString("@@ -1,96 +1,96 @@\n")
	for i := 1; i <= 48; i++ {
		value := fmt.Sprintf("old_token_rule_%03d", i)
		if i == 47 {
			value = "REMOVE_LEGACY_TOKEN_CHECK"
		}
		fmt.Fprintf(&raw, "-%s\n", value)
	}
	for i := 1; i <= 48; i++ {
		value := fmt.Sprintf("new_token_rule_%03d", i)
		if i == 47 {
			value = "ADD_ROTATING_TOKEN_CHECK"
		}
		fmt.Fprintf(&raw, "+%s\n", value)
	}
	return raw.String()
}

func manyHunksFixture(count int) string {
	var raw strings.Builder
	for i := 1; i <= count; i++ {
		fmt.Fprintf(&raw, "@@ -%d +%d @@\n-HUNK_%02d_OLD\n+HUNK_%02d_NEW\n", i, i, i, i)
	}
	return raw.String()
}

func duplicateLinesFixture() string {
	var raw strings.Builder
	raw.WriteString("@@ -1,80 +1,80 @@\n")
	for i := 1; i <= 80; i++ {
		value := "duplicate_change"
		if i == 41 {
			value = "UNIQUE_MIDDLE_CHANGE"
		}
		fmt.Fprintf(&raw, "+%s\n", value)
	}
	return raw.String()
}

func mixedComparisonFixture(hunks, linesPerSign int) string {
	var raw strings.Builder
	for hunk := 1; hunk <= hunks; hunk++ {
		fmt.Fprintf(&raw, "@@ -%d,%d +%d,%d @@\n", hunk*100, linesPerSign, hunk*100, linesPerSign)
		for line := 1; line <= linesPerSign; line++ {
			fmt.Fprintf(&raw, "-old_h%02d_line_%02d\n", hunk, line)
		}
		for line := 1; line <= linesPerSign; line++ {
			fmt.Fprintf(&raw, "+new_h%02d_line_%02d\n", hunk, line)
		}
	}
	return raw.String()
}

func sizedComparisonFixture(lineCount int) string {
	var raw strings.Builder
	fmt.Fprintf(&raw, "@@ -1,%d +1,%d @@\n", lineCount/2, lineCount/2)
	for i := 0; i < lineCount; i++ {
		sign := '+'
		if i%2 == 0 {
			sign = '-'
		}
		fmt.Fprintf(&raw, "%ccomparison_changed_line_%05d_with_stable_payload_abcdefghijklmnopqrstuvwxyz_0123456789\n", sign, i)
	}
	return raw.String()
}

func holdoutReplacementFixture() string {
	var raw strings.Builder
	raw.WriteString("@@ -1,70 +1,70 @@\n")
	for i := 1; i <= 70; i++ {
		value := fmt.Sprintf("old_session_behavior_%03d", i)
		if i == 52 {
			value = "DEPRECATE_COOKIE_SESSION"
		}
		fmt.Fprintf(&raw, "-%s\n", value)
	}
	raw.WriteString("@@ -90,70 +90,70 @@\n")
	for i := 1; i <= 70; i++ {
		value := fmt.Sprintf("new_session_behavior_%03d", i)
		if i == 18 {
			value = "USE_SIGNED_SESSION"
		}
		fmt.Fprintf(&raw, "+%s\n", value)
	}
	return raw.String()
}

func holdoutBoundaryFixture() string {
	var raw strings.Builder
	for hunk := 1; hunk <= 6; hunk++ {
		fmt.Fprintf(&raw, "@@ -%d,11 +%d,11 @@\n", hunk*20, hunk*20)
		for line := 1; line <= 11; line++ {
			value := fmt.Sprintf("boundary_h%02d_line_%02d", hunk, line)
			switch {
			case hunk == 1 && line == 1:
				value = "FIRST_BOUNDARY_CHANGE"
			case hunk == 3 && line == 6:
				value = "MIDDLE_BOUNDARY_CHANGE"
			case hunk == 6 && line == 11:
				value = "LAST_BOUNDARY_CHANGE"
			}
			fmt.Fprintf(&raw, "+%s\n", value)
		}
	}
	return raw.String()
}
