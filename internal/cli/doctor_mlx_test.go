package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/exitcode"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func TestDoctorChecksCacheForModelOverride(t *testing.T) {
	chdir(t, cliRepository(t))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	spec := mlxmodel.Spec{
		Repo: "mlx-community/Qwen3.5-4B-4bit", Revision: "0e7ffd5c629ef7719d4cbc04069232580bfa9d9c", Quantization: "4bit",
	}
	store := mlxmodel.Store{Root: t.TempDir()}
	dir, err := store.Destination(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// Only Qwen has cache metadata. Invalid metadata must stop before the helper is run.
	cliWrite(t, dir, "commiter-model.json", "{}", 0600)
	oldStore, oldPlatform, oldBundled := newMLXModelStore, mlxPlatform, mlxHelperPath
	t.Cleanup(func() {
		newMLXModelStore, mlxPlatform, mlxHelperPath = oldStore, oldPlatform, oldBundled
	})
	newMLXModelStore = func() (mlxmodel.Store, error) { return store, nil }
	mlxPlatform = func() (string, string) { return "darwin", "arm64" }
	mlxHelperPath = func() (string, error) { return "/test/commiter-mlx-helper", nil }
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--json", "--model=" + spec.Repo, "doctor"}, &stdout, &stderr); code != exitcode.LLM {
		t.Fatalf("code = %d, stdout = %s, stderr = %s", code, &stdout, &stderr)
	}
	var result struct {
		Doctor map[string]doctorCheck `json:"doctor"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Doctor["config"].OK || !strings.Contains(result.Doctor["mlx_model"].Message, "metadata is invalid") {
		t.Fatalf("doctor did not check Qwen cache: %s", &stdout)
	}
}

func TestSetupAndDoctorRejectHelperOnlyOnPath(t *testing.T) {
	chdir(t, cliRepository(t))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	helper := writeDoctorHelper(t, "{}")
	t.Setenv("PATH", filepath.Dir(helper)+string(os.PathListSeparator)+os.Getenv("PATH"))
	oldBundled, oldPlatform, oldStore := mlxHelperPath, mlxPlatform, newMLXModelStore
	t.Cleanup(func() {
		mlxHelperPath, mlxPlatform, newMLXModelStore = oldBundled, oldPlatform, oldStore
	})
	mlxHelperPath = func() (string, error) { return "", os.ErrNotExist }
	mlxPlatform = func() (string, string) { return "darwin", "arm64" }
	newMLXModelStore = func() (mlxmodel.Store, error) {
		t.Error("accepted PATH helper and opened model store")
		return mlxmodel.Store{}, errors.New("unexpected model store access")
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"setup"}, &stdout, &stderr); code != exitcode.LLM ||
		!strings.Contains(stderr.String(), "bundled MLX helper is unavailable") {
		t.Errorf("setup code=%d stderr=%q", code, &stderr)
	}
	if code, checks := runJSONDoctor(t); code != exitcode.LLM || checks["mlx_helper"].OK ||
		!strings.Contains(checks["mlx_helper"].Message, "bundled MLX helper is unavailable") {
		t.Errorf("doctor code=%d helper=%#v", code, checks["mlx_helper"])
	}
}

func TestMLXCapabilityProbeRequiresJSONOnlyExpectedShape(t *testing.T) {
	tests := []struct {
		name string
		json string
		want bool
	}{
		{name: "valid plan", json: `{"commits":[{"type":"fix","scope":"cli","breaking":false,"summary":"check model","file_ids":["F001"]}]}`, want: true},
		{name: "reasoning property", json: `{"commits":[{"type":"fix","scope":"cli","breaking":false,"summary":"check model","file_ids":["F001"]}],"reasoning":"because"}`},
		{name: "prose before JSON", json: `Here is the answer: {"commits":[{"type":"fix","scope":"cli","breaking":false,"summary":"check model","file_ids":["F001"]}]}`},
		{name: "prose after JSON", json: `{"commits":[{"type":"fix","scope":"cli","breaking":false,"summary":"check model","file_ids":["F001"]}]} explanation`},
		{name: "wrong file ID", json: `{"commits":[{"type":"fix","scope":"cli","breaking":false,"summary":"check model","file_ids":["F002"]}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			helper := writeDoctorHelper(t, `{"ok":true,"stop_reason":"completed","generated_json":`+quoteJSON(t, test.json)+`}`)
			got, err := mlxCapabilityProbe(context.Background(), helper, "owner/model", "/local/model")
			if err != nil {
				t.Fatalf("probe error = %v", err)
			}
			if got != test.want {
				t.Fatalf("probe result = %v, want %v", got, test.want)
			}
		})
	}
}

func TestMLXCapabilityProbeUsesPlanningSchema(t *testing.T) {
	requestPath := filepath.Join(t.TempDir(), "request.json")
	response := `{"ok":true,"stop_reason":"completed","generated_json":"{\"commits\":[{\"type\":\"fix\",\"scope\":\"cli\",\"breaking\":false,\"summary\":\"check model\",\"file_ids\":[\"F001\"]}]}"}`
	helper := filepath.Join(t.TempDir(), "commiter-mlx-helper")
	body := "#!/bin/sh\ncat >" + shellQuoteForTest(requestPath) + "\nprintf '%s\\n' " + shellQuoteForTest(response) + "\n"
	if err := os.WriteFile(helper, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	if ok, err := mlxCapabilityProbe(context.Background(), helper, "owner/model", "/local/model"); err != nil || !ok {
		t.Fatalf("probe result = %v, error = %v", ok, err)
	}
	data, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var request struct {
		Schema json.RawMessage `json:"schema"`
	}
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	want, err := planning.Schema([]string{"F001"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(request.Schema, want) {
		t.Fatalf("probe schema = %s, want planning schema %s", request.Schema, want)
	}
}

func TestDoctorMLXMissingModelDoesNotStartHelperOrContactRegistry(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	helper := writeDoctorHelper(t, `{"ok":true,"stop_reason":"completed","generated_json":"{\"ok\":true}"}`, marker)
	oldHelper, oldStore := mlxHelperPath, newMLXModelStore
	t.Cleanup(func() { mlxHelperPath, newMLXModelStore = oldHelper, oldStore })
	mlxHelperPath = func() (string, error) { return helper, nil }
	newMLXModelStore = func() (mlxmodel.Store, error) { return mlxmodel.Store{Root: t.TempDir()}, nil }

	checks := doctorMLXFor(config.Values{
		Model: "owner/model", ModelRevision: strings.Repeat("a", 40), ModelQuantization: "4bit",
	}, "darwin", "arm64")
	if checks["mlx_model"]["ok"] != false || !strings.Contains(checks["mlx_model"]["message"].(string), "run commiter setup") {
		t.Fatalf("model check = %#v", checks["mlx_model"])
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("helper started despite missing model: %v", err)
	}
}

func TestMLXDoctorJSONKeepsEnvelope(t *testing.T) {
	repo := cliRepository(t)
	chdir(t, repo)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(configHome, "commiter"), 0o700); err != nil {
		t.Fatal(err)
	}
	configText := config.GlobalTemplate
	if err := os.WriteFile(filepath.Join(configHome, "commiter", "config.toml"), []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	oldStore, oldHelper, oldPlatform := newMLXModelStore, mlxHelperPath, mlxPlatform
	t.Cleanup(func() { newMLXModelStore, mlxHelperPath, mlxPlatform = oldStore, oldHelper, oldPlatform })
	newMLXModelStore = func() (mlxmodel.Store, error) { return mlxmodel.Store{Root: t.TempDir()}, nil }
	mlxHelperPath = func() (string, error) { return "/test/commiter-mlx-helper", nil }
	mlxPlatform = func() (string, string) { return "darwin", "arm64" }

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "doctor"}, &stdout, &stderr)
	if code == 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	var result struct {
		Doctor   map[string]doctorCheck `json:"doctor"`
		OK       bool                   `json:"ok"`
		ReadOnly bool                   `json:"read_only"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode doctor JSON: %v (%q)", err, stdout.String())
	}
	_, hasOllamaChecks := result.Doctor["ollama"]
	if !result.ReadOnly || result.OK || result.Doctor["mlx_backend"].OK != true ||
		result.Doctor["mlx_model"].OK || hasOllamaChecks {
		t.Fatalf("unexpected MLX doctor JSON: %#v", result)
	}
}

func TestDoctorReportsSkippedMLXChecksWhenConfigurationCannotBeResolved(t *testing.T) {
	repo := cliRepository(t)
	chdir(t, repo)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(configHome, "commiter"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "commiter", "config.toml"), []byte("[llm\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldHelper := mlxHelperPath
	t.Cleanup(func() { mlxHelperPath = oldHelper })
	mlxHelperPath = func() (string, error) {
		t.Error("invalid configuration reached helper lookup")
		return "", os.ErrNotExist
	}

	var stdout bytes.Buffer
	if code := Run([]string{"--json", "doctor"}, &stdout, &bytes.Buffer{}); code == 0 {
		t.Fatal("invalid configuration unexpectedly passed doctor")
	}
	var result struct {
		Doctor map[string]doctorCheck `json:"doctor"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode doctor JSON: %v", err)
	}
	if _, exists := result.Doctor["ollama_binary"]; exists {
		t.Fatalf("unresolved configuration revived removed Ollama backend: %#v", result.Doctor)
	}
	for _, key := range []string{"mlx_backend", "mlx_helper", "mlx_model", "mlx_model_runtime", "mlx_json_schema_grammar", "mlx_constrained_generation"} {
		if check, exists := result.Doctor[key]; !exists || check.OK || !strings.Contains(check.Message, "configuration") {
			t.Errorf("check %s = %#v, want skipped MLX check", key, check)
		}
	}
	for _, key := range []string{"structured_output", "thinking"} {
		if _, exists := result.Doctor[key]; exists {
			t.Errorf("unexpected legacy check %s", key)
		}
	}
}

func TestMLXPlatformSupportIsAppleSiliconOnly(t *testing.T) {
	for _, test := range []struct {
		goos, goarch string
		want         bool
	}{
		{"darwin", "arm64", true},
		{"darwin", "amd64", false},
		{"linux", "arm64", false},
		{"windows", "arm64", false},
	} {
		if got := mlxPlatformSupported(test.goos, test.goarch); got != test.want {
			t.Errorf("mlxPlatformSupported(%q, %q) = %v, want %v", test.goos, test.goarch, got, test.want)
		}
	}
}

func writeDoctorHelper(t *testing.T, response string, marker ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "commiter-mlx-helper")
	body := "#!/bin/sh\ncat >/dev/null\n"
	if len(marker) > 0 {
		body += "touch " + shellQuoteForTest(marker[0]) + "\n"
	}
	body += "printf '%s\\n' " + shellQuoteForTest(response) + "\n"
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func quoteJSON(t *testing.T, value string) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func shellQuoteForTest(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
