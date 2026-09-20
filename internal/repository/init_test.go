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
