package mlx

import (
	"fmt"
	"os"
	"path/filepath"
)

const helperName = "commiter-mlx-helper"

// ResolveBundledHelper locates the private helper shipped beside commiter in
// the standard bin/ and libexec/ layout. It never searches PATH, so an
// unrelated executable cannot silently replace the signed release helper.
func ResolveBundledHelper() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve commiter executable: %w", err)
	}
	return ResolveHelperForExecutable(executable)
}

// ResolveHelperForExecutable resolves the private helper for an executable
// under the distribution's bin/ directory. It is exported so other internal
// packages can inspect the same install layout without duplicating path rules.
func ResolveHelperForExecutable(executable string) (string, error) {
	if executable == "" {
		return "", fmt.Errorf("commiter executable path is empty")
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err == nil {
		executable = resolved
	}
	helper := filepath.Clean(filepath.Join(filepath.Dir(executable), "..", "libexec", helperName))
	info, err := os.Stat(helper)
	if err != nil {
		return "", fmt.Errorf("bundled MLX helper is unavailable")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("bundled MLX helper is not an executable file")
	}
	return helper, nil
}
