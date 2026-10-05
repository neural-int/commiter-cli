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

// Ground the fixture references in executable before/after contracts rather
// than treating parser acceptance as proof that a synthetic change is valid.
func TestContractFixtureProgramsPassBeforeAndAfter(t *testing.T) {
	for _, f := range append(append(contractFixtures(), holdouts()...), holdout16(), sharedCalleeGuardrail()) {
		for _, prefix := range []string{"-", "+"} {
			t.Run(f.Name+prefix, func(t *testing.T) {
				root := t.TempDir()
				if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.23.0\n"), 0600); err != nil {
					t.Fatal(err)
				}
				for _, file := range f.Files {
					content := ""
					for _, line := range strings.Split(file.RawDiff, "\n") {
						if strings.HasPrefix(line, prefix) {
							content += strings.TrimPrefix(line, prefix) + "\n"
						}
					}
					path := filepath.Join(root, *file.NewPath)
					if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(content), 0600); err != nil {
						t.Fatal(err)
					}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, "go", "test", "./...")
				command.Dir = root
				command.Env = append(os.Environ(), "GOWORK=off")
				out, err := command.CombinedOutput()
				if err != nil {
					t.Fatalf("fixture does not compile/pass: %v\n%s", err, out)
				}
			})
		}
	}
}
