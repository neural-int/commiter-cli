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
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func TestFindMLXHelperPrefersBundledHelperAndFallsBackToPath(t *testing.T) {
	oldBundled, oldLookPath := resolveBundledMLXHelper, lookPath
	t.Cleanup(func() {
		resolveBundledMLXHelper = oldBundled
		lookPath = oldLookPath
	})

	pathLookups := 0
	resolveBundledMLXHelper = func() (string, error) {
		return "/package/libexec/commiter-mlx-helper", nil
	}
	lookPath = func(name string) (string, error) {
		pathLookups++
		return "/usr/local/bin/" + name, nil
	}

	helper, err := findMLXHelper()
	if err != nil {
		t.Fatal(err)
	}
	if helper != "/package/libexec/commiter-mlx-helper" {
		t.Fatalf("helper = %q, want bundled helper", helper)
	}
	if pathLookups != 0 {
		t.Fatalf("PATH lookups = %d, want 0 when bundled helper exists", pathLookups)
	}

	resolveBundledMLXHelper = func() (string, error) {
		return "", errors.New("bundled helper unavailable")
	}
	helper, err = findMLXHelper()
	if err != nil {
		t.Fatal(err)
	}
	if helper != "/usr/local/bin/commiter-mlx-helper" {
		t.Fatalf("helper = %q, want PATH helper", helper)
	}
	if pathLookups != 1 {
		t.Fatalf("PATH lookups = %d, want 1 after bundled lookup fails", pathLookups)
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
	oldLookPath, oldStore := lookPath, newMLXModelStore
	t.Cleanup(func() { lookPath, newMLXModelStore = oldLookPath, oldStore })
	lookPath = func(name string) (string, error) {
		if name == "commiter-mlx-helper" {
			return helper, nil
		}
		return "", os.ErrNotExist
	}
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

func TestMLXDoctorJSONKeepsEnvelopeAndUsesSelectedBackend(t *testing.T) {
	repo := cliRepository(t)
	chdir(t, repo)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(configHome, "commiter"), 0o700); err != nil {
		t.Fatal(err)
	}
	configText := "[llm]\nbackend = \"mlx\"\nmodel = \"owner/model\"\nmodel_revision = \"" + strings.Repeat("a", 40) + "\"\nmodel_quantization = \"4bit\"\n"
	if err := os.WriteFile(filepath.Join(configHome, "commiter", "config.toml"), []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	oldStore, oldPath, oldPlatform := newMLXModelStore, lookPath, mlxPlatform
	t.Cleanup(func() { newMLXModelStore, lookPath, mlxPlatform = oldStore, oldPath, oldPlatform })
	newMLXModelStore = func() (mlxmodel.Store, error) { return mlxmodel.Store{Root: t.TempDir()}, nil }
	lookPath = func(name string) (string, error) {
		if name == "commiter-mlx-helper" {
			return "/test/commiter-mlx-helper", nil
		}
		return "", os.ErrNotExist
	}
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

func TestDoctorKeepsOllamaBinaryCheckWhenBackendCannotBeResolved(t *testing.T) {
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
	oldPath := lookPath
	t.Cleanup(func() { lookPath = oldPath })
	lookPath = func(name string) (string, error) {
		if name == "ollama" {
			return "/test/ollama", nil
		}
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
	if !result.Doctor["ollama_binary"].OK {
		t.Fatalf("legacy Ollama check missing from unresolved-backend output: %#v", result.Doctor)
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
