package codexcontext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalPathMatchesSymlinkAliasesForDeletedTargets(t *testing.T) {
	root := t.TempDir()
	aliasParent := t.TempDir()
	alias := filepath.Join(aliasParent, "repo")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	for _, tc := range []struct {
		name string
		root string
		path string
	}{
		{name: "symlink root and real target", root: alias, path: filepath.Join(root, "deleted", "file.go")},
		{name: "real root and symlink target", root: root, path: filepath.Join(alias, "deleted", "file.go")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := localPath(tc.root, tc.path), "deleted/file.go"; got != want {
				t.Fatalf("localPath(%q, %q) = %q, want %q", tc.root, tc.path, got, want)
			}
		})
	}
}

func TestLocalPathMatchesSymlinkAliasesForExistingTargets(t *testing.T) {
	root := t.TempDir()
	aliasParent := t.TempDir()
	alias := filepath.Join(aliasParent, "repo")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "match.go"), []byte("package internal\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got, want := localPath(alias, filepath.Join(alias, "internal", "match.go")), "internal/match.go"; got != want {
		t.Fatalf("localPath(%q, existing symlink target) = %q, want %q", alias, got, want)
	}
}

func TestLocalPathRejectsSymlinkEscapingRepository(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "outside")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	if got := localPath(root, filepath.Join(link, "secret.go")); got != "" {
		t.Fatalf("localPath accepted path outside repository: %q", got)
	}
}
