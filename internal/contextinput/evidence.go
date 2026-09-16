package contextinput

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type EvidenceReduction struct {
	Level                  string `json:"level"`
	OriginalCount          int    `json:"original_count"`
	OriginalBytes          int    `json:"original_bytes"`
	OriginalDigest         string `json:"original_digest"`
	RetainedCount          int    `json:"retained_count"`
	RetainedBytes          int    `json:"retained_bytes"`
	RetainedDigest         string `json:"retained_digest"`
	CoverageCount          int    `json:"coverage_count"`
	OriginalCoverageDigest string `json:"original_coverage_digest"`
	RetainedCoverageDigest string `json:"retained_coverage_digest"`
}

func reduceEvidence(document Document, density int) Document {
	reduced := cloneDocument(document)
	for index := range reduced.Files {
		file := &reduced.Files[index]
		if file.Mode != syntax.ModeStructural || len(file.Evidence) == 0 {
			continue
		}
		original := append([]syntax.Evidence(nil), file.Evidence...)
		canonical := canonicalEvidence(original)
		minimum := requiredEvidenceCount(canonical)
		limit := minimum + (len(canonical)-minimum)*density/1000
		retained := selectEvidence(canonical, limit)
		level := "strong"
		if len(retained) == len(canonical) {
			level = "light"
		} else if len(retained)*2 >= len(canonical) {
			level = "medium"
		}
		file.Evidence = retained
		file.EvidenceReduction = newEvidenceReduction(level, original, retained)
	}
	return reduced
}

func canonicalEvidence(values []syntax.Evidence) []syntax.Evidence {
	ordered := append([]syntax.Evidence(nil), values...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].StartByte != ordered[j].StartByte {
			return ordered[i].StartByte < ordered[j].StartByte
		}
		if ordered[i].EndByte != ordered[j].EndByte {
			return ordered[i].EndByte < ordered[j].EndByte
		}
		return evidenceKey(ordered[i]) < evidenceKey(ordered[j])
	})
	groups := make(map[string][]syntax.Evidence, len(ordered))
	keys := make([]string, 0, len(ordered))
	for _, value := range ordered {
		key := evidenceKey(value)
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], value)
	}
	result := make([]syntax.Evidence, 0, len(keys))
	for _, key := range keys {
		group := groups[key]
		// The middle occurrence is an actual observed fact and avoids implying
		// that a coalesced node spans from the first duplicate to the last.
		result = append(result, group[len(group)/2])
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].StartByte != result[j].StartByte {
			return result[i].StartByte < result[j].StartByte
		}
		if result[i].EndByte != result[j].EndByte {
			return result[i].EndByte < result[j].EndByte
		}
		return evidenceKey(result[i]) < evidenceKey(result[j])
	})
	return result
}

func selectEvidence(values []syntax.Evidence, limit int) []syntax.Evidence {
	if limit >= len(values) {
		return append([]syntax.Evidence(nil), values...)
	}
	required := requiredEvidenceIndices(values)
	if limit < len(required) {
		limit = len(required)
	}
	selected := make(map[int]bool, limit)
	for _, index := range required {
		selected[index] = true
	}
	remaining := limit - len(selected)
	if remaining > 0 {
		candidates := make([]int, 0, len(values)-len(selected))
		for index := range values {
			if !selected[index] {
				candidates = append(candidates, index)
			}
		}
		// Divide the complete ordered range into equal buckets and retain the
		// most meaningful fact from each bucket. This preserves spatial coverage
		// instead of biasing the beginning of a file.
		for bucket := 0; bucket < remaining && len(candidates) > 0; bucket++ {
			start := bucket * len(candidates) / remaining
			end := (bucket + 1) * len(candidates) / remaining
			if end <= start {
				end = start + 1
			}
			best := candidates[start]
			for _, candidate := range candidates[start+1 : end] {
				if evidencePriority(values[candidate]) > evidencePriority(values[best]) {
					best = candidate
				}
			}
			selected[best] = true
		}
	}
	result := make([]syntax.Evidence, 0, len(selected))
	for index, value := range values {
		if selected[index] {
			result = append(result, value)
		}
	}
	return result
}

func requiredEvidenceCount(values []syntax.Evidence) int {
	return len(requiredEvidenceIndices(values))
}

func requiredEvidenceIndices(values []syntax.Evidence) []int {
	seen := map[string]bool{}
	indices := make([]int, 0)
	for index, value := range values {
		for _, key := range coverageKeys(value) {
			if !seen[key] {
				seen[key] = true
				indices = append(indices, index)
			}
		}
	}
	if len(indices) == 0 && len(values) > 0 {
		indices = append(indices, 0)
	}
	return indices
}

func evidencePriority(value syntax.Evidence) int {
	priority := 0
	if value.EnclosingDeclaration != "" {
		priority += 2
	}
	if value.Name != "" {
		priority += 3
	}
	if value.Role != "" {
		priority += 4
	}
	if isDeclarationKind(value.Kind) && value.Name != "" {
		priority += 5
	}
	return priority
}

func coverageKeys(value syntax.Evidence) []string {
	keys := make([]string, 0, 3)
	if value.Role != "" {
		keys = append(keys, "role:"+value.Role)
	}
	if value.EnclosingDeclaration != "" {
		keys = append(keys, "enclosing:"+value.EnclosingDeclaration)
	}
	if value.Name != "" && isDeclarationKind(value.Kind) {
		keys = append(keys, "declaration:"+value.Kind+":"+value.Name)
	}
	return keys
}

func isDeclarationKind(kind string) bool {
	if kind == "variable_declarator" {
		return true
	}
	for _, part := range []string{"function", "method", "struct", "interface", "module"} {
		if strings.Contains(kind, part) {
			return true
		}
	}
	return strings.HasSuffix(kind, "_declaration") || strings.HasSuffix(kind, "_definition") || kind == "class"
}

func evidenceKey(value syntax.Evidence) string {
	return value.Kind + "\x00" + value.Name + "\x00" + value.EnclosingDeclaration + "\x00" + value.Role
}

func newEvidenceReduction(level string, original, retained []syntax.Evidence) *EvidenceReduction {
	originalBytes, originalDigest := evidenceIdentity(original)
	retainedBytes, retainedDigest := evidenceIdentity(retained)
	originalCoverage := coverageSet(original)
	retainedCoverage := coverageSet(retained)
	return &EvidenceReduction{
		Level: level, OriginalCount: len(original), OriginalBytes: originalBytes, OriginalDigest: originalDigest,
		RetainedCount: len(retained), RetainedBytes: retainedBytes, RetainedDigest: retainedDigest,
		CoverageCount: len(originalCoverage), OriginalCoverageDigest: digestStrings(originalCoverage),
		RetainedCoverageDigest: digestStrings(retainedCoverage),
	}
}

func validateEvidencePreserved(original, retained []syntax.Evidence, reduction *EvidenceReduction) error {
	if reduction == nil {
		if !equalEvidence(original, retained) {
			return errors.New("unaudited evidence change")
		}
		return nil
	}
	if reduction.Level != "light" && reduction.Level != "medium" && reduction.Level != "strong" {
		return errors.New("evidence reduction level is invalid")
	}
	want := newEvidenceReduction(reduction.Level, original, retained)
	if *want != *reduction {
		return errors.New("evidence reduction metadata does not match content")
	}
	if reduction.OriginalCoverageDigest != reduction.RetainedCoverageDigest {
		return errors.New("required structural coverage was not retained")
	}
	canonical := canonicalEvidence(original)
	originalFacts := make(map[syntax.Evidence]bool, len(canonical))
	for _, value := range canonical {
		originalFacts[value] = true
	}
	for _, value := range retained {
		if !originalFacts[value] {
			return fmt.Errorf("retained evidence %q has no original fact", evidenceKey(value))
		}
	}
	if len(original) > 0 && len(retained) == 0 {
		return errors.New("all structural evidence was removed")
	}
	return nil
}

func evidenceIdentity(values []syntax.Evidence) (int, string) {
	encoded, _ := json.Marshal(values)
	digest := sha256.Sum256(encoded)
	return len(encoded), hex.EncodeToString(digest[:])
}

func coverageSet(values []syntax.Evidence) []string {
	set := map[string]bool{}
	for _, value := range values {
		for _, key := range coverageKeys(value) {
			set[key] = true
		}
	}
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func digestStrings(values []string) string {
	encoded, _ := json.Marshal(values)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func equalEvidence(left, right []syntax.Evidence) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
