// Package ggufmodel implements the managed GGUF store. It contains no model
// registrations: artifacts must be qualified before setup can offer them.
package ggufmodel

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

const metadataName = "commiter-model.json"
const formatVersion = "commiter-gguf-cache-v1"

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Spec is trusted registration data, never metadata discovered in a cache.
// The first implementation accepts a single GGUF artifact only.
type Spec struct {
	Repo         string `json:"repo"`
	Revision     string `json:"revision"`
	Quantization string `json:"quantization"`
	Files        []File `json:"files"`
}

func digest(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func Validate(spec Spec) error {
	invalid := errors.New("GGUF registration requires a pinned repository, quantization, single GGUF file, expected size and SHA-256")
	parts := strings.Split(spec.Repo, "/")
	if len(parts) != 2 || !digest(spec.Revision, 40) || spec.Quantization == "" || len(spec.Files) != 1 {
		return invalid
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return invalid
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
				return invalid
			}
		}
	}
	for _, value := range []string{spec.Quantization, spec.Revision, spec.Repo} {
		if strings.ContainsRune(value, 0) || !utf8.ValidString(value) {
			return invalid
		}
	}
	seen := map[string]bool{}
	for _, file := range spec.Files {
		if !utf8.ValidString(file.Path) || strings.ContainsAny(file.Path, "\x00\\") || strings.HasPrefix(file.Path, "/") || path.Clean(file.Path) != file.Path || !strings.HasSuffix(strings.ToLower(file.Path), ".gguf") || file.Size <= 0 || file.Size > 50<<30 || !digest(file.SHA256, 64) || seen[file.Path] {
			return invalid
		}
		for _, part := range strings.Split(file.Path, "/") {
			if part == "" || part == "." || part == ".." || part == metadataName || strings.HasPrefix(part, ".") {
				return invalid
			}
		}
		seen[file.Path] = true
	}
	return nil
}

func CacheKey(spec Spec) (string, error) {
	if err := Validate(spec); err != nil {
		return "", err
	}
	names := make([]string, len(spec.Files))
	for i, file := range spec.Files {
		names[i] = file.Path
	}
	sort.Strings(names)
	values := append([]string{formatVersion, "llamacpp", spec.Repo, strings.ToLower(spec.Revision), spec.Quantization}, names...)
	h := sha256.New()
	for _, value := range values {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func normalized(spec Spec) Spec {
	spec.Revision = strings.ToLower(spec.Revision)
	spec.Files = append([]File(nil), spec.Files...)
	for i := range spec.Files {
		spec.Files[i].SHA256 = strings.ToLower(spec.Files[i].SHA256)
	}
	sort.Slice(spec.Files, func(i, j int) bool { return spec.Files[i].Path < spec.Files[j].Path })
	return spec
}
