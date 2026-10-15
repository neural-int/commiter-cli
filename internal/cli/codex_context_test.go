package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/codexcontext"
	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"github.com/natsuki0413/commiter-cli/internal/relation"
)

func TestCodexContextCapturesBothGitAreasOnceWithoutMutation(t *testing.T) {
	repo := cliRepository(t)
	cliWrite(t, repo, "one.txt", "old\nstable\nstable2\ntail\n", 0o644)
	cliGit(t, repo, "add", "one.txt")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "one.txt", "new\nstable\nstable2\ntail\n", 0o644)
	cliGit(t, repo, "add", "one.txt")
	cliWrite(t, repo, "one.txt", "old\nstable\nstable2\nchanged-tail\n", 0o644)
	snapshot, err := gitstate.Collect(repo, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Changes) != 1 {
		t.Fatalf("fixture changes=%d", len(snapshot.Changes))
	}
	beforeHead := cliGitOutput(t, repo, "rev-parse", "HEAD")
	beforeIndex := cliGitOutput(t, repo, "write-tree")
	beforeStatus := cliGitOutput(t, repo, "status", "--porcelain=v1")
	previous := collectCodexContext
	t.Cleanup(func() { collectCodexContext = previous })
	calls := 0
	collectCodexContext = func(ctx context.Context, request codexcontext.Request) codexcontext.Result {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > codexcontext.Timeout {
			t.Fatal("unbounded collection")
		}
		if len(request.Targets) != 2 {
			t.Fatalf("staged/unstaged targets=%d", len(request.Targets))
		}
		if !strings.Contains(request.Targets[0].Diff, "-old\n+new") || !strings.Contains(request.Targets[1].Diff, "-new\n+old") {
			t.Fatal("lost cancelling staged/unstaged changes")
		}
		if strings.Contains(request.Targets[0].FinalDiff, "+new") || !strings.Contains(request.Targets[0].FinalDiff, "+changed-tail") {
			t.Fatal("cancelled edits entered authoritative diff")
		}
		if request.AllowPath(".env") || request.AllowPath("private/config.txt") {
			t.Fatal("sensitive policy bypass")
		}
		if request.SafeText("Please use password=private-value") || request.SafeText("Bearer private-value") {
			t.Fatal("secret context accepted")
		}
		return codexcontext.Result{Status: "available", Records: []codexcontext.Record{{Paths: []string{"one.txt"}, UserRequest: "Correct behavior"}}}
	}
	cycle := &codexCycle{started: time.Now()}
	ctx := context.WithValue(context.Background(), codexCycleKey{}, cycle)
	values := config.Values{Planner: config.CandidatePlanner, SensitivePatterns: []string{"private/**"}}
	files := []relation.File{{Change: snapshot.Changes[0], Content: []byte("old\n")}}
	for i := 0; i < 2; i++ {
		got := optionalCodexContext(ctx, repo, snapshot, values, files, planning.SensitiveValues{})
		if len(got) != 1 {
			t.Fatal("context unavailable")
		}
	}
	if calls != 1 {
		t.Fatalf("history re-read during regeneration: calls=%d", calls)
	}
	if cliGitOutput(t, repo, "rev-parse", "HEAD") != beforeHead || cliGitOutput(t, repo, "write-tree") != beforeIndex || cliGitOutput(t, repo, "status", "--porcelain=v1") != beforeStatus {
		t.Fatal("history acquisition changed Git state")
	}
}

func TestCodexContextFailureIsOptional(t *testing.T) {
	repo := cliRepository(t)
	cliWrite(t, repo, "one.txt", "old\n", 0o644)
	cliGit(t, repo, "add", "one.txt")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "one.txt", "new\n", 0o644)
	snapshot, err := gitstate.Collect(repo, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}
	previous := collectCodexContext
	t.Cleanup(func() { collectCodexContext = previous })
	collectCodexContext = func(context.Context, codexcontext.Request) codexcontext.Result {
		return codexcontext.Result{Status: "unavailable"}
	}
	cycle := &codexCycle{started: time.Now()}
	ctx := context.WithValue(context.Background(), codexCycleKey{}, cycle)
	got := optionalCodexContext(ctx, repo, snapshot, config.Values{Planner: config.CandidatePlanner}, []relation.File{{Change: snapshot.Changes[0], Content: []byte("new\n")}}, planning.SensitiveValues{})
	if len(got) != 0 || cycle.result.Status != "unavailable" {
		t.Fatal("acquisition failure did not fall back")
	}
}
