package mlx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveHelperForExecutableUsesPrivateLibexecLayout(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	libexecDir := filepath.Join(root, "libexec")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(libexecDir, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(binDir, "commiter")
	if err := os.WriteFile(executable, []byte("app"), 0o755); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(libexecDir, helperName)
	if err := os.WriteFile(helper, []byte("helper"), 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(root, "prefix", "bin")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	launchPath := filepath.Join(linkDir, "commiter")
	if err := os.Symlink(executable, launchPath); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveHelperForExecutable(launchPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(helper)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != want {
		t.Fatalf("resolved helper = %q, want %q", resolved, want)
	}
}

func TestResolveHelperForExecutableRejectsMissingAndNonExecutableHelper(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "bin", "commiter")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveHelperForExecutable(executable); err == nil {
		t.Fatal("missing helper unexpectedly resolved")
	}

	helper := filepath.Join(root, "libexec", helperName)
	if err := os.MkdirAll(filepath.Dir(helper), 0o755); err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0o644)
	if runtime.GOOS == "windows" {
		mode = 0o755
	}
	if err := os.WriteFile(helper, []byte("helper"), mode); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveHelperForExecutable(executable); err == nil {
		t.Fatal("non-executable helper unexpectedly resolved")
	}
}
