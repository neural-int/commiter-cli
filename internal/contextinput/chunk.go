package contextinput

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

type CompressionProfile string

const (
	CompressionNone   CompressionProfile = ""
	CompressionLight  CompressionProfile = "light"
	CompressionMedium CompressionProfile = "medium"
	CompressionStrong CompressionProfile = "strong"
)

const canonicalChunkLines = 48

type compressionLimits struct {
	chunkLines   int
	excerptLines int
	excerptBytes int
}

var compressionProfiles = []struct {
	name   CompressionProfile
	limits compressionLimits
}{
	{name: CompressionLight, limits: compressionLimits{chunkLines: canonicalChunkLines, excerptLines: 7, excerptBytes: 224}},
	{name: CompressionMedium, limits: compressionLimits{chunkLines: canonicalChunkLines, excerptLines: 5, excerptBytes: 176}},
	{name: CompressionStrong, limits: compressionLimits{chunkLines: canonicalChunkLines, excerptLines: 3, excerptBytes: 128}},
}

type summaryLine struct {
	originalIndex int
	hunk          int
	text          string
}

type summaryLineID struct {
	chunk             int
	originalDiffIndex int
}

type chunkCompression struct {
	summary         string
	selected        map[summaryLineID]bool
	digests         []string
	omittedAdded    int
	omittedRemoved  int
	selectedAdded   int
	selectedRemoved int
}

func compressionLimitsFor(profile CompressionProfile) (compressionLimits, bool) {
	for _, candidate := range compressionProfiles {
		if candidate.name == profile {
			return candidate.limits, true
		}
	}
	return compressionLimits{}, false
}

func indexSummaryLines(source string) []summaryLine {
	input := splitLines(source)
	lines := make([]summaryLine, 0, len(input))
	hunk := 0
	for index, text := range input {
		if strings.HasPrefix(text, "@@") {
			hunk++
		}
		lines = append(lines, summaryLine{originalIndex: index + 1, hunk: hunk, text: text})
	}
	return lines
}

func summarizeFileLines(lines []summaryLine) []summaryLine {
	kept := make([]summaryLine, 0, len(lines))
	inHeader := true
	for _, line := range lines {
		if strings.HasPrefix(line.text, "@@") {
			inHeader = false
		}
		if inHeader && redundantFileHeader(line.text) {
			continue
		}
		kept = append(kept, line)
	}
	return kept
}

func summarizeHunkLines(lines []summaryLine) []summaryLine {
	kept := make([]summaryLine, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line.text, " ") {
			continue
		}
		kept = append(kept, line)
	}
	return kept
}

func summaryLinesText(lines []summaryLine) string {
	values := make([]string, len(lines))
	for index, line := range lines {
		values[index] = line.text
	}
	return strings.Join(values, "\n")
}

func selectedSummaryLines(lines []summaryLine, selected map[summaryLineID]bool, chunkLines int) []summaryLine {
	retained := make([]summaryLine, 0, len(selected))
	for index, line := range lines {
		id := summaryLineID{chunk: index/chunkLines + 1, originalDiffIndex: line.originalIndex}
		if selected[id] {
			retained = append(retained, line)
		}
	}
	return retained
}

func compressSummaryLines(lines []summaryLine, limits compressionLimits) chunkCompression {
	result := chunkCompression{selected: make(map[summaryLineID]bool)}
	var output []string
	activeHunk := ""
	for offset, chunkIndex := 0, 1; offset < len(lines); offset, chunkIndex = offset+limits.chunkLines, chunkIndex+1 {
		end := offset + limits.chunkLines
		if end > len(lines) {
			end = len(lines)
		}
		chunk := lines[offset:end]
		texts := make([]string, len(chunk))
		additions, deletions, contextLines := 0, 0, 0
		headers := make([]string, 0, 2)
		if activeHunk != "" {
			headers = append(headers, activeHunk)
		}
		for index, line := range chunk {
			texts[index] = line.text
			switch {
			case strings.HasPrefix(line.text, "@@"):
				activeHunk = line.text
				if len(headers) == 0 || headers[len(headers)-1] != line.text {
					headers = append(headers, line.text)
				}
			case strings.HasPrefix(line.text, "+"):
				additions++
			case strings.HasPrefix(line.text, "-"):
				deletions++
			case strings.HasPrefix(line.text, " "):
				contextLines++
			}
		}
		digest := sha256.Sum256([]byte(strings.Join(texts, "\n")))
		encodedDigest := hex.EncodeToString(digest[:])
		result.digests = append(result.digests, encodedDigest)
		for _, header := range headers {
			output = append(output, "hunk: "+truncateUTF8(header, limits.excerptBytes))
		}
		output = append(output, fmt.Sprintf(
			"chunk %d: lines=%d additions=%d deletions=%d context=%d sha256=%s",
			chunkIndex, len(chunk), additions, deletions, contextLines, encodedDigest,
		))

		selected := make(map[int]bool)
		for _, sign := range []byte{'+', '-'} {
			ranked := rankSummaryLines(chunk, sign)
			for _, index := range ranked[:min(limits.excerptLines, len(ranked))] {
				selected[index] = true
				result.selected[summaryLineID{chunk: chunkIndex, originalDiffIndex: chunk[index].originalIndex}] = true
				if sign == '+' {
					result.selectedAdded++
				} else {
					result.selectedRemoved++
				}
			}
		}

		omittedAdded, omittedRemoved := 0, 0
		flushOmitted := func() {
			if omittedAdded == 0 && omittedRemoved == 0 {
				return
			}
			output = append(output, fmt.Sprintf("omitted: additions=%d deletions=%d", omittedAdded, omittedRemoved))
			result.omittedAdded += omittedAdded
			result.omittedRemoved += omittedRemoved
			omittedAdded, omittedRemoved = 0, 0
		}
		for index, line := range chunk {
			if !isChangedSummaryLine(line.text) {
				flushOmitted()
				continue
			}
			if !selected[index] {
				if line.text[0] == '+' {
					omittedAdded++
				} else {
					omittedRemoved++
				}
				continue
			}
			flushOmitted()
			label := "added: "
			if line.text[0] == '-' {
				label = "removed: "
			}
			output = append(output, label+truncateUTF8(line.text[1:], limits.excerptBytes))
		}
		flushOmitted()
	}
	result.summary = strings.Join(output, "\n")
	return result
}

func rankSummaryLines(chunk []summaryLine, sign byte) []int {
	type group struct {
		position int
		lines    []int
	}
	groupsByHunk := make(map[int]*group)
	var groups []*group
	for index, line := range chunk {
		if len(line.text) == 0 || line.text[0] != sign {
			continue
		}
		item := groupsByHunk[line.hunk]
		if item == nil {
			item = &group{position: line.originalIndex}
			groupsByHunk[line.hunk] = item
			groups = append(groups, item)
		}
		item.lines = append(item.lines, index)
	}
	if len(groups) == 0 {
		return nil
	}
	positions := make([]int, len(groups))
	lineRanks := make([][]int, len(groups))
	for index, item := range groups {
		positions[index] = item.position
		linePositions := make([]int, len(item.lines))
		for lineIndex, chunkIndex := range item.lines {
			linePositions[lineIndex] = chunk[chunkIndex].originalIndex
		}
		for _, ranked := range spatialRank(linePositions) {
			lineRanks[index] = append(lineRanks[index], item.lines[ranked])
		}
	}
	hunkRank := spatialRank(positions)
	var ranked []int
	for depth := 0; ; depth++ {
		added := false
		for _, hunkIndex := range hunkRank {
			if depth < len(lineRanks[hunkIndex]) {
				ranked = append(ranked, lineRanks[hunkIndex][depth])
				added = true
			}
		}
		if !added {
			return ranked
		}
	}
}

func spatialRank(positions []int) []int {
	if len(positions) == 0 {
		return nil
	}
	selected := make([]bool, len(positions))
	ranked := []int{0}
	selected[0] = true
	if len(positions) > 1 {
		ranked = append(ranked, len(positions)-1)
		selected[len(positions)-1] = true
	}
	for len(ranked) < len(positions) {
		best, bestDistance := -1, -1
		for candidate := range positions {
			if selected[candidate] {
				continue
			}
			distance := int(^uint(0) >> 1)
			for chosen := range positions {
				if selected[chosen] {
					distance = min(distance, absolute(positions[candidate]-positions[chosen]))
				}
			}
			if distance > bestDistance || distance == bestDistance && positions[candidate] < positions[best] {
				best, bestDistance = candidate, distance
			}
		}
		selected[best] = true
		ranked = append(ranked, best)
	}
	return ranked
}

func isChangedSummaryLine(line string) bool {
	return len(line) > 0 && (line[0] == '+' || line[0] == '-')
}

func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return strings.Repeat(".", max(0, limit))
	}
	payload := limit - 3
	headBytes := (payload*3 + 4) / 5
	tailBytes := payload - headBytes
	return utf8Prefix(value, headBytes) + "..." + utf8Suffix(value, tailBytes)
}

func utf8Prefix(value string, limit int) string {
	if limit >= len(value) {
		return value
	}
	value = value[:max(0, limit)]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func utf8Suffix(value string, limit int) string {
	if limit >= len(value) {
		return value
	}
	value = value[len(value)-max(0, limit):]
	for !utf8.ValidString(value) {
		value = value[1:]
	}
	return value
}

func absolute(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
