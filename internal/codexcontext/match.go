package codexcontext

import (
	"os"
	"path/filepath"
	"strings"
)

func localPath(root, name string) string {
	if name == "" {
		return ""
	}
	canonicalRoot, ok := canonicalPath(root)
	if !ok {
		return ""
	}
	if !filepath.IsAbs(name) {
		name = filepath.Join(canonicalRoot, filepath.FromSlash(name))
	}
	canonicalName, ok := canonicalPath(name)
	if !ok {
		return ""
	}
	rel, err := filepath.Rel(canonicalRoot, canonicalName)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(rel)
}

// canonicalPath resolves every existing path component. When a target was
// deleted, it resolves the nearest existing parent and appends the missing
// suffix so symlink aliases still compare consistently.
func canonicalPath(path string) (string, bool) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	current := filepath.Clean(absolute)
	var missing []string
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", false
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), true
		} else if !os.IsNotExist(err) {
			return "", false
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}
