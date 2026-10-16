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
