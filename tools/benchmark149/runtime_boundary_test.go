package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test-pass is an observable property of a snapshot, but cannot by itself prove
// the semantic commit partition. This probe executes only the synthetic fixture.
func TestPassingIntermediatesDoNotProveSemanticBoundary(t *testing.T) {
	f := contractFixtures()[2]
	pairs := [][]string{}
	for i := 0; i < len(f.Files); i += 2 {
		pairs = append(pairs, []string{f.Files[i].ID, f.Files[i+1].ID})
	}
	exact, fm, fs := quality(pairs, f.Expected)
	if exact || fm != 0 || fs != 24 {
		t.Fatal("probe no longer represents a false split")
	}
	for afterCount := 2; afterCount < len(f.Files); afterCount += 2 {
		t.Run(string(rune('0'+afterCount/2)), func(t *testing.T) {
			root := t.TempDir()
			if e := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.23.0\n"), 0600); e != nil {
				t.Fatal(e)
			}
			for i, file := range f.Files {
				prefix := "-"
				if i < afterCount {
					prefix = "+"
				}
				var content strings.Builder
				for _, line := range strings.Split(file.RawDiff, "\n") {
					if strings.HasPrefix(line, prefix) {
						content.WriteString(strings.TrimPrefix(line, prefix))
						content.WriteByte('\n')
					}
				}
				path := filepath.Join(root, *file.NewPath)
				if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(path, []byte(content.String()), 0600); e != nil {
					t.Fatal(e)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			c := exec.CommandContext(ctx, "go", "test", "./...")
			c.Dir = root
			c.Env = append(os.Environ(), "GOWORK=off")
			start := time.Now()
			_, e := c.CombinedOutput()
			if e != nil {
				t.Fatalf("intermediate failed: %v", e)
			}
			t.Logf("runtime_boundary_audit after_files=%d before_files=%d test_pass=true wall_seconds=%.3f model_calls=0", afterCount, len(f.Files)-afterCount, time.Since(start).Seconds())
		})
	}
	t.Log("runtime_boundary_audit passing_intermediates=5 proposed_groups=6 gold_groups=2 FM=0 FS=24")
}
