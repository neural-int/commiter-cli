package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/exitcode"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/llamacpp"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
)

func TestLlamaCPPStopsBeforeModelStoreOrPrompt(t *testing.T) {
	chdir(t, cliRepository(t))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	oldStore, oldConfirm, oldHelper := newMLXModelStore, confirmFunc, mlxHelperPath
	t.Cleanup(func() { newMLXModelStore, confirmFunc, mlxHelperPath = oldStore, oldConfirm, oldHelper })
	newMLXModelStore = func() (mlxmodel.Store, error) { t.Fatal("opened MLX cache for llamacpp"); return mlxmodel.Store{}, nil }
	confirmFunc = func(string) bool { t.Fatal("prompted for unqualified runtime"); return false }
	mlxHelperPath = func() (string, error) { t.Fatal("resolved helper for unqualified runtime"); return "", nil }
	for _, command := range [][]string{{"setup"}, {"setup", "--update-model"}, {"--dry-run"}, {}} {
		args := append([]string{"--runtime=llamacpp", "--model", "trial", "--model-path=/models/trial.gguf"}, command...)
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != exitcode.LLM || !strings.Contains(stderr.String(), llamacpp.QualificationMessage) {
			t.Fatalf("code=%d output=%s stderr=%s", code, &stdout, &stderr)
		}
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "--runtime=llamacpp", "--model=trial", "--model-path=/models/trial.gguf", "doctor"}, &stdout, &stderr)
	var result struct {
		Doctor   map[string]doctorCheck `json:"doctor"`
		ReadOnly bool                   `json:"read_only"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if code != exitcode.LLM || !result.ReadOnly || !result.Doctor["config"].OK || result.Doctor["mlx_backend"].Message != "" {
		t.Fatalf("doctor misclassified runtime: %s", &stdout)
	}
	for _, key := range []string{"bundle", "model", "metal", "uds", "llguidance", "three_phase"} {
		check, ok := result.Doctor["llamacpp_"+key]
		if !ok || check.OK || !strings.Contains(check.Message, "not checked") {
			t.Fatalf("unchecked capability reported ready: %s %+v", key, check)
		}
	}
}

func TestDirectPlanningCannotBypassQualificationGate(t *testing.T) {
	values := config.Defaults().Values
	values.Backend = "llamacpp"
	_, err := generateCommitPlan(context.Background(), t.TempDir(), gitstate.Snapshot{}, values, "")
	if err == nil || !strings.Contains(err.Error(), llamacpp.QualificationMessage) {
		t.Fatal(err)
	}
}

func TestModelSelectionDisplayDistinguishesRegisteredNameFromLocalSource(t *testing.T) {
	chdir(t, cliRepository(t))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "--runtime=mlx", "--model=" + config.CandidateModel, "--model-path=/models/trial", "config", "show", "--effective"}, &stdout, &stderr)
	var result struct {
		Selection map[string]string `json:"model_selection"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if code != 0 || result.Selection["source"] != "local" || result.Selection["id"] != config.CandidateModel {
		t.Fatalf("local registration collision: code=%d output=%s stderr=%s", code, &stdout, &stderr)
	}
}

func TestLocalMLXDoesNotOpenManagedCacheOrDownload(t *testing.T) {
	chdir(t, cliRepository(t))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	oldStore, oldPlatform, oldHelper, oldConfirm := newMLXModelStore, mlxPlatform, mlxHelperPath, confirmFunc
	t.Cleanup(func() {
		newMLXModelStore, mlxPlatform, mlxHelperPath, confirmFunc = oldStore, oldPlatform, oldHelper, oldConfirm
	})
	newMLXModelStore = func() (mlxmodel.Store, error) {
		t.Fatal("opened managed cache for local model")
		return mlxmodel.Store{}, nil
	}
	confirmFunc = func(string) bool { t.Fatal("prompted for a local model download"); return false }
	mlxPlatform = func() (string, string) { return "darwin", "arm64" }
	mlxHelperPath = func() (string, error) { return "", os.ErrNotExist }
	values := config.Defaults().Values
	values.ModelSource = "local"
	values.ModelPath = t.TempDir()
	path, err := resolveMLXModelPath(values)
	if err != nil || path != values.ModelPath {
		t.Fatalf("local resolution: %s %v", path, err)
	}
	values.ModelPath += "/missing"
	if _, err := resolveMLXModelPath(values); err == nil {
		t.Fatal("missing local directory accepted")
	}
	for _, command := range []string{"setup", "doctor"} {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"--model=" + config.CandidateModel, "--model-path=" + values.ModelPath, command}, &stdout, &stderr); code == 0 {
			t.Fatalf("missing local model succeeded: %s", command)
		}
	}
}
