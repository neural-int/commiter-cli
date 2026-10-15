package codexcontext

import (
	"path/filepath"
	"strings"
)

func localPath(root, name string) string {
	if name == "" {
		return ""
	}
	if !filepath.IsAbs(name) {
		name = filepath.Join(root, filepath.FromSlash(name))
	}
	rel, err := filepath.Rel(root, filepath.Clean(name))
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(rel)
}

// editBlocks ignores hunk coordinates, but preserves exact changed lines and
// operation order. It never executes a patch or reads a file referenced by logs.
func editBlocks(diff string) []string {
	var blocks []string
	var block []string
	inHunk := false
	flush := func() {
		if len(block) > 0 {
			blocks = append(blocks, strings.Join(block, "\n"))
			block = nil
		}
	}
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "@@") {
			flush()
			inHunk = true
			continue
		}
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			inHunk = false
			continue
		}
		if !inHunk {
			continue
		}
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			block = append(block, line)
		} else if !strings.HasPrefix(line, "\\ No newline") {
			flush()
		}
	}
	flush()
	return blocks
}

func matches(current, logged string, kind string) bool {
	want := editBlocks(current)
	if len(want) == 0 {
		return false
	}
	got := editBlocks(logged)
	// App Server can expose full added/deleted file content rather than hunks.
	if len(got) == 0 && (kind == "add" || kind == "delete") && logged != "" {
		prefix := "+"
		if kind == "delete" {
			prefix = "-"
		}
		lines := strings.Split(strings.TrimSuffix(logged, "\n"), "\n")
		for i := range lines {
			lines[i] = prefix + lines[i]
		}
		got = []string{strings.Join(lines, "\n")}
	}
	if len(got) == 0 {
		return false
	}
	// Every block of an observed operation must still exist, with multiplicity.
	remaining := make(map[string]int, len(want))
	for _, b := range want {
		remaining[b]++
	}
	for _, b := range got {
		if remaining[b] == 0 {
			return false
		}
		remaining[b]--
	}
	return true
}
