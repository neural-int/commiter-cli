package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/exitcode"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/trust"
)

type doctorCheck struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func TestRemovedPlannerConfigurationNeverPromptsOrStartsBackend(t *testing.T) {
	root := cliRepository(t)
	chdir(t, root)
	for _, legacy := range []string{"[llm]\nplanner = \"single-pass\"\n", "[llm]\nbackend = \"ollama\"\n"} {
		for _, args := range [][]string{{"setup"}, {"setup", "--update-model"}, {"--dry-run"}, {"--json", "doctor"}} {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			path := filepath.Join(configHome, "commiter", "config.toml")
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
				t.Fatal(err)
			}
			previousConfirm, previousStore := confirmFunc, newMLXModelStore
			confirmFunc = func(string) bool { t.Fatal("legacy settings requested an operation"); return false }
			newMLXModelStore = func() (mlxmodel.Store, error) {
				t.Fatal("legacy settings opened model store")
				return mlxmodel.Store{}, nil
			}
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			confirmFunc, newMLXModelStore = previousConfirm, previousStore
			if code == 0 {
				t.Fatalf("legacy settings accepted: %v", args)
			}
			if args[len(args)-1] != "doctor" && code != exitcode.Usage {
				t.Fatalf("migration error code=%d", code)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != legacy {
				t.Fatal("configuration changed during rejected migration")
			}
		}
	}
}

func TestVersionHumanAndJSON(t *testing.T) {
	oldVersion := Version
	Version = "test-version"
	t.Cleanup(func() { Version = oldVersion })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != "commiter test-version\n" || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	if code := Run([]string{"--json", "version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("JSON code = %d", code)
	}
	var result map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result["version"] != "test-version" {
		t.Fatalf("JSON = %q, error = %v", stdout.String(), err)
	}
}

func TestHelpIncludesInitIgnoreUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "commiter init [--ignore PATTERN]...") {
		t.Fatalf("help = %q", stdout.String())
	}
}

func TestInitDeclineDoesNotChangeDirectory(t *testing.T) {
	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	oldConfirm := confirmFunc
	confirmFunc = func(string) bool { return false }
	t.Cleanup(func() { confirmFunc = oldConfirm })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatalf(".git exists after rejection: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf(".gitignore exists after rejection: %v", err)
	}
}

func TestInitIgnoreAppendsOnlyApprovedPatterns(t *testing.T) {
	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	oldConfirm := confirmFunc
	confirmFunc = func(string) bool { return true }
	t.Cleanup(func() { confirmFunc = oldConfirm })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init", "--ignore", "*.tmp", "--ignore=build/"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "*.tmp\nbuild/\n" {
		t.Fatalf("gitignore = %q", got)
	}
}

func TestUpdateCheckIsSkippedForJSONOutput(t *testing.T) {
	repo := cliRepository(t)
	cliWrite(t, repo, "README.md", "base\n", 0o644)
	cliGit(t, repo, "add", "README.md")
	cliGit(t, repo, "commit", "-m", "base")
	chdir(t, repo)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	old := updateCheckInteractive
	updateCheckInteractive = func() bool {
		t.Fatal("update check should be skipped for JSON output")
		return true
	}
	t.Cleanup(func() { updateCheckInteractive = old })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--json", "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var result struct {
		DryRun bool `json:"dry_run"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || !result.DryRun {
		t.Fatalf("JSON = %q, error = %v", stdout.String(), err)
	}
}

func TestUpdateCheckInteractiveSkipsCI(t *testing.T) {
	t.Setenv("CI", "true")
	if updateCheckInteractive() {
		t.Fatal("update check should be skipped in CI")
	}
}

func TestJSONRestrictionIsUsageErrorAndDoesNotMixStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json"}, &stdout, &stderr)
	if code != 2 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var result struct {
		ExitCode int `json:"exit_code"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.ExitCode != 2 {
		t.Fatalf("JSON = %q, error = %v", stdout.String(), err)
	}
}

func TestJSONDoctorIsStableAndReadOnly(t *testing.T) {
	configHome, stateHome := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/version":
			_, _ = io.WriteString(w, `{"version":"0.31.2"}`)
		case "/api/tags":
			_, _ = io.WriteString(w, `{"models":[{"name":"qwen3.5:4b-q4_K_M"}]}`)
		case "/api/chat":
			_, _ = io.WriteString(w, `{"message":{"content":"{\"ok\":true}","thinking":""},"done":true}`)
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	configDirectory := filepath.Join(configHome, "commiter")
	if err := os.MkdirAll(configDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDirectory, "config.toml")
	configBefore := []byte(config.GlobalTemplate)
	if err := os.WriteFile(configPath, configBefore, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "doctor"}, &stdout, &stderr)
	if code == 2 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var result struct {
		Doctor   map[string]map[string]any `json:"doctor"`
		ReadOnly bool                      `json:"read_only"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("JSON = %q, error = %v", stdout.String(), err)
	}
	if !result.ReadOnly || result.Doctor["git"] == nil || result.Doctor["config"] == nil || result.Doctor["mlx_backend"] == nil {
		t.Fatalf("doctor JSON = %#v", result)
	}
	configAfter, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(configAfter, configBefore) {
		t.Fatalf("doctor changed configuration: %q", configAfter)
	}
	stateEntries, err := os.ReadDir(stateHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(stateEntries) != 0 {
		t.Fatalf("doctor created trust/state entries: %v", stateEntries)
	}
}

func TestMLXSetupShowsPinnedDownloadPlanBeforeApproval(t *testing.T) {
	root := cliRepository(t)
	chdir(t, root)
	const model = "mlx-community/Qwen3.5-4B-4bit"
	const revision = "0e7ffd5c629ef7719d4cbc04069232580bfa9d9c"
	requests, downloads := 0, 0
	var treePath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests++
		treePath = request.URL.Path
		if strings.Contains(request.URL.Path, "/resolve/") {
			downloads++
		}
		_, _ = io.WriteString(w, `[{"type":"file","path":"config.json","size":1,"oid":"`+strings.Repeat("a", 40)+`"},{"type":"file","path":"model.safetensors","size":1,"oid":"`+strings.Repeat("b", 40)+`"}]`)
	}))
	defer server.Close()
	store := mlxmodel.Store{Root: t.TempDir(), BaseURL: server.URL, Client: server.Client()}
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	if err := os.MkdirAll(filepath.Join(configHome, "commiter"), 0o700); err != nil {
		t.Fatal(err)
	}
	configText := config.GlobalTemplate
	if err := os.WriteFile(filepath.Join(configHome, "commiter", "config.toml"), []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	oldStore, oldConfirm, oldPlatform, oldHelper := newMLXModelStore, confirmFunc, mlxPlatform, mlxHelperPath
	t.Cleanup(func() {
		newMLXModelStore, confirmFunc, mlxPlatform, mlxHelperPath = oldStore, oldConfirm, oldPlatform, oldHelper
	})
	newMLXModelStore = func() (mlxmodel.Store, error) { return store, nil }
	mlxPlatform = func() (string, string) { return "darwin", "arm64" }
	mlxHelperPath = func() (string, error) { return "/test/commiter-mlx-helper", nil }
	var stdout bytes.Buffer
	confirmFunc = func(string) bool {
		if !strings.Contains(stdout.String(), "estimated download size: 2 bytes") ||
			!strings.Contains(stdout.String(), "model repository: "+model) ||
			!strings.Contains(stdout.String(), "pinned revision: "+revision) ||
			!strings.Contains(stdout.String(), "quantization label: 4bit") ||
			!strings.Contains(stdout.String(), "destination:") {
			t.Fatalf("unexpected download details before approval: %s", stdout.String())
		}
		return false
	}
	if code := Run([]string{"--model", model, "setup"}, &stdout, io.Discard); code != 0 {
		t.Fatalf("setup code = %d, output = %q", code, stdout.String())
	}
	if requests != 1 || downloads != 0 {
		t.Fatalf("requests = %d, downloads = %d", requests, downloads)
	}
	if want := "/api/models/" + model + "/tree/" + revision; treePath != want {
		t.Fatalf("tree path = %q, want %q", treePath, want)
	}
}

func runJSONDoctor(t *testing.T) (int, map[string]doctorCheck) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "doctor"}, &stdout, &stderr)
	if stderr.Len() != 0 {
		t.Fatalf("stderr=%q", stderr.String())
	}
	var result struct {
		Doctor map[string]doctorCheck `json:"doctor"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("JSON=%q error=%v", stdout.String(), err)
	}
	return code, result.Doctor
}

func TestUnimplementedCommandArgumentsAreValidated(t *testing.T) {
	tests := [][]string{
		{"setup", "--garbage"},
		{"setup", "--update-model", "extra"},
		{"doctor", "nonsense"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(args, &stdout, &stderr); code != 2 {
				t.Fatalf("%v code = %d, stderr = %q", args, code, stderr.String())
			}
		})
	}
}

func TestConfigShowJSONUsesOnlyStdout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "config", "show", "--effective"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result["config"] == nil {
		t.Fatalf("JSON = %q, error = %v", stdout.String(), err)
	}
}

func TestConfigPathAndInitGlobal(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	wantPath := filepath.Join(configHome, "commiter", "config.toml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "config", "path", "--global"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("path code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var pathResult map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &pathResult); err != nil || pathResult["path"] != wantPath {
		t.Fatalf("path JSON = %q, error = %v", stdout.String(), err)
	}

	stdout.Reset()
	code = Run([]string{"config", "init", "--global"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init code = %d, stderr = %q", code, stderr.String())
	}
	info, err := os.Stat(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("global config mode = %o", info.Mode().Perm())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"config", "init", "--global"}, &stdout, &stderr); code != 2 {
		t.Fatalf("duplicate init code = %d, stderr = %q", code, stderr.String())
	}
}

func TestConfigShowAppliesCLIOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--language", "ja", "--json", "config", "show"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var result struct {
		Config map[string]struct {
			Value  any    `json:"value"`
			Source string `json:"source"`
		} `json:"config"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	entry := result.Config["commit.language"]
	if entry.Value != "ja" || entry.Source != "cli" {
		t.Fatalf("commit.language = %#v", entry)
	}
}

func TestConfigErrorDoesNotExposeRawValue(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	directory := filepath.Join(configHome, "commiter")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "config.toml"), []byte("[commit]\nconfirm = \"RAW_SECRET\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"config", "show"}, &stdout, &stderr)
	if code != 2 || strings.Contains(stderr.String(), "RAW_SECRET") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestTrustListJSONIsReadOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "trust", "list"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var result struct {
		Trust []any `json:"trust"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || len(result.Trust) != 0 {
		t.Fatalf("JSON = %q, error = %v", stdout.String(), err)
	}
}

func TestTrustListShowsCanonicalRepoHashSourceAndArgv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	repo := t.TempDir()
	store := trust.New(filepath.Join(stateHome, "commiter"))
	if err := store.Approve(trust.Entry{
		RepoPath:       repo,
		DefinitionHash: "definition-hash",
		SourceType:     "repo_config",
		Commands:       [][]string{{"echo", "a b"}, {"echo", "a", "b"}, {"printf", "", "line\nbreak"}},
	}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "trust", "list"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var result struct {
		Trust []trust.Entry `json:"trust"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trust) != 1 || result.Trust[0].RepoPath != canonical || result.Trust[0].DefinitionHash != "definition-hash" || result.Trust[0].SourceType != "repo_config" {
		t.Fatalf("trust = %#v", result.Trust)
	}
	if !reflect.DeepEqual(result.Trust[0].Commands, [][]string{{"echo", "a b"}, {"echo", "a", "b"}, {"printf", "", "line\nbreak"}}) {
		t.Fatalf("argv = %#v", result.Trust[0].Commands)
	}

	stdout.Reset()
	code = Run([]string{"trust", "list"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `argv=[[\"echo\",\"a b\"],[\"echo\",\"a\",\"b\"],[\"printf\",\"\",\"line\\nbreak\"]]`) {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestCorruptTrustStateIsInternalForListAndRevoke(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	stateDir := filepath.Join(stateHome, "commiter")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "trust.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()

	for _, args := range [][]string{{"trust", "list"}, {"trust", "revoke", repo}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code != 1 || !strings.Contains(stderr.String(), "invalid trust state") {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
			}
		})
	}
}
