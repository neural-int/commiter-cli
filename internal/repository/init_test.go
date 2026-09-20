package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPlanInitializationForDirectoryWithoutRepository(t *testing.T) {
	dir := t.TempDir()
	plan, err := PlanInitialization(dir)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Root != dir || !plan.InitializeGit || !plan.CreateGitignore {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestApplyInitializationCreatesRepositoryAndEmptyGitignore(t *testing.T) {
	dir := t.TempDir()
	plan, err := PlanInitialization(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyInitialization(plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf(".git was not created: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 0 {
		t.Fatalf("gitignore = %q, want empty", contents)
	}
}

func TestPlanInitializationPreservesExistingRepositoryAndGitignore(t *testing.T) {
	dir := t.TempDir()
	command := exec.Command("git", "-C", dir, "init")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	want := []byte("build/\n")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), want, 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanInitialization(dir)
	if err != nil {
		t.Fatal(err)
	}
	if plan.InitializeGit || plan.CreateGitignore {
		t.Fatalf("plan = %#v", plan)
	}
	if err := ApplyInitialization(plan); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("gitignore changed: %q", got)
	}
}

func TestPlanInitializationRejectsBareRepository(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bare.git")
	command := exec.Command("git", "init", "--bare", dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v: %s", err, output)
	}

	if _, err := PlanInitialization(dir); err == nil {
		t.Fatal("PlanInitialization accepted a bare repository")
	}
}

func TestApplyInitializationAppendsRequestedGitignorePatterns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("build/"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanInitializationWithIgnore(dir, []string{"*.tmp", "build/"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.UpdateGitignore {
		t.Fatalf("plan = %#v, want gitignore update", plan)
	}
	if err := ApplyInitialization(plan); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "build/\n*.tmp\n" {
		t.Fatalf("gitignore = %q", got)
	}
}

func TestApplyInitializationHandlesGitignoreEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		before string
		want   string
	}{
		{name: "empty", before: "", want: "*.tmp\n"},
		{name: "missing final newline", before: "build/", want: "build/\n*.tmp\n"},
		{name: "duplicate", before: "*.tmp\n", want: "*.tmp\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tt.before), 0o644); err != nil {
				t.Fatal(err)
			}
			plan, err := PlanInitializationWithIgnore(dir, []string{"*.tmp"})
			if err != nil {
				t.Fatal(err)
			}
			if err := ApplyInitialization(plan); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("gitignore = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyInitializationRejectsChangedGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanInitializationWithIgnore(dir, []string{"*.tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("changed/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyInitialization(plan); err == nil {
		t.Fatal("ApplyInitialization accepted changed .gitignore")
	}
	got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "changed/\n" {
		t.Fatalf("gitignore = %q after rejected apply", got)
	}
}

func TestPlanInitializationRejectsUnsafeGitignorePatternAndSymlink(t *testing.T) {
	dir := t.TempDir()
	for _, pattern := range []string{"safe\nsecret", "\nfoo\n"} {
		if _, err := PlanInitializationWithIgnore(dir, []string{pattern}); err == nil {
			t.Fatalf("accepted multi-line gitignore pattern %q", pattern)
		}
	}
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanInitializationWithIgnore(dir, []string{"*.tmp"}); err == nil {
		t.Fatal("accepted symlink .gitignore")
	}
}

func TestPlanInitializationRejectsInvalidGitMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanInitialization(dir); err == nil {
		t.Fatal("PlanInitialization accepted invalid .git metadata")
	}
}

func TestPlanInitializationRejectsInvalidAncestorGitMetadata(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(parent, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanInitialization(child); err == nil {
		t.Fatal("PlanInitialization accepted invalid ancestor .git metadata")
	}
}

func TestApplyInitializationRefusesRepositoryCreatedAfterConfirmation(t *testing.T) {
	dir := t.TempDir()
	plan, err := PlanInitialization(dir)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "-C", dir, "init")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := ApplyInitialization(plan); err == nil {
		t.Fatal("ApplyInitialization initialized a repository after state changed")
	}
}
