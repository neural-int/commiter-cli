//go:build darwin

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/llamacpp"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type observedLlamaTrial struct {
	backend llm.CycleBackend
	rows    *[]llm.Response
	t       *testing.T
}

func (b observedLlamaTrial) Chat(c context.Context, m []llm.Message, s json.RawMessage) (llm.Response, error) {
	return b.backend.Chat(c, m, s)
}
func (b observedLlamaTrial) ChatWithOptions(c context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	return b.backend.ChatWithOptions(c, m, s, o)
}
func (b observedLlamaTrial) BeginCycle(ctx context.Context) (llm.OptionsBackend, func() error, error) {
	started := time.Now()
	cycle, close, err := b.backend.BeginCycle(ctx)
	b.t.Logf("trial cycle initialization wall=%s error=%v", time.Since(started), err)
	if err != nil {
		return nil, nil, err
	}
	return observedLlamaPhase{OptionsBackend: cycle, rows: b.rows, t: b.t}, close, nil
}

type observedLlamaPhase struct {
	llm.OptionsBackend
	rows *[]llm.Response
	t    *testing.T
}

func (b observedLlamaPhase) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	r, err := b.OptionsBackend.ChatWithOptions(ctx, m, s, o)
	if err != nil {
		b.t.Logf("native phase %s error=%v", o.GenerationProfile, err)
	}
	*b.rows = append(*b.rows, r)
	return r, err
}

// Explicit public synthetic fixture only. Neither the user's repository content
// nor its Codex history is used. All output remains local to the test process.
func TestInstalledLlamaCPPCLIDryRun(t *testing.T) {
	executable, model := os.Getenv("COMMITER_TEST_LLAMA_SERVER"), os.Getenv("COMMITER_TEST_GGUF_MODEL")
	if executable == "" || model == "" {
		t.Skip("explicit LLGuidance server and verified GGUF required")
	}
	repo := cliRepository(t)
	cliWrite(t, repo, "counter.go", "package counter\nfunc value() int { return 1 }\n", 0644)
	cliGit(t, repo, "add", "counter.go")
	cliGit(t, repo, "commit", "-m", "base")
	cliWrite(t, repo, "counter.go", "package counter\nfunc value() int { return 2 }\n", 0644)
	beforeHead := cliGitOutput(t, repo, "rev-parse", "HEAD")
	beforeIndex := cliGitOutput(t, repo, "diff", "--cached", "--binary")
	chdir(t, repo)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	oldFactory, oldUpdate := newLlamaCPPBackend, updateCheckInteractive
	t.Cleanup(func() { newLlamaCPPBackend, updateCheckInteractive = oldFactory, oldUpdate })
	rows := []llm.Response{}
	observations := []llamacpp.TrialObservation{}
	newLlamaCPPBackend = func(values config.Values) (llm.OptionsBackend, error) {
		return observedLlamaTrial{backend: &llamacpp.TrialBackend{Executable: executable, ModelPath: values.ModelPath, Observe: func(observation llamacpp.TrialObservation) {
			observations = append(observations, observation)
			if observation.ConstraintFailure != "" {
				t.Logf("native LLGuidance failure operation=%s", observation.ConstraintFailure)
			}
			if len(observations) == 1 {
				command := exec.Command("/usr/sbin/lsof", "-nP", "-a", "-p", fmt.Sprint(observation.PID), "-iTCP")
				data, err := command.CombinedOutput()
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(data) != 0 {
					t.Error("TCP absence could not be verified")
				}
			}
		}}, rows: &rows, t: t}, nil
	}
	updateCheckInteractive = func() bool { return false }
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "--dry-run", "--runtime=llamacpp", "--model=" + llamacpp.TrialModelID, "--model-path=" + model}, &stdout, &stderr)
	for i, row := range rows {
		t.Logf("phase=%d stop=%s input_tokens=%d output_tokens=%d preparations=%d prompt_ns=%d decode_ns=%d wall_ns=%d", i+1, row.StopReason, row.PromptEvalCount, row.EvalCount, row.InputPreparationCalls, row.PromptEvalDuration, row.EvalDuration, row.TotalDuration)
	}
	if code != 0 {
		t.Fatalf("CLI E2E code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
	}
	var result struct {
		Plan   planning.Plan `json:"plan"`
		DryRun bool          `json:"dry_run"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || !result.DryRun || len(result.Plan.Commits) != 1 {
		t.Fatalf("invalid E2E result: %s", &stdout)
	}
	if len(observations) != 2 {
		t.Fatalf("expected startup/closed observations, got %d", len(observations))
	}
	observed := observations[1]
	t.Logf("model_loads=%d metal_device=%t gpu_layers=%d/%d", observed.ModelLoads, observed.MetalDevice, observed.OffloadedLayers, observed.TotalLayers)
	if observed.ModelLoads != 1 || !observed.MetalDevice || observed.TotalLayers == 0 || observed.OffloadedLayers != observed.TotalLayers {
		t.Fatal("Metal/model-load evidence missing")
	}
	if _, err := os.Lstat(observed.Directory); !os.IsNotExist(err) {
		t.Fatal("server socket directory remained after E2E")
	}
	if got := cliGitOutput(t, repo, "rev-parse", "HEAD"); got != beforeHead {
		t.Fatal("dry run changed HEAD")
	}
	if got := cliGitOutput(t, repo, "diff", "--cached", "--binary"); got != beforeIndex {
		t.Fatal("dry run changed index")
	}
	for _, row := range rows {
		if row.CacheReusedTokens != 0 || row.PrefixReuseStatus != "unsupported" || row.PromptEvalCount == 0 || row.EvalCount == 0 {
			t.Fatal("incorrect trial telemetry")
		}
	}
	t.Logf("synthetic CLI dry run passed all three phases and the final validator; commit type=%s summary=%q", result.Plan.Commits[0].Type, result.Plan.Commits[0].Summary)
}
