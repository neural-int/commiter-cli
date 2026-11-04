package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePrecedenceSourcesAndArrayReplacement(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(t.TempDir(), "global.toml")
	repo := filepath.Join(root, ".commiter.toml")
	writeTestFile(t, global, `schema_version = 1
[commit]
language = "ja"
confirm = false
[analysis]
include = ["global/**"]
[safety]
additional_sensitive_patterns = ["*.private"]
`)
	writeTestFile(t, repo, `schema_version = 1
[commit]
language = "en"
[analysis]
include = ["repo/**"]
[verification]
autodetect = false
timeout_seconds = 30
[[verification.commands]]
name = "test"
argv = ["go", "test", "./..."]
cwd = "."
`)
	language := "ja"
	effective, err := Resolve(global, repo, root, CLIOverrides{Language: &language})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if effective.Values.Language != "ja" || effective.Sources["commit.language"] != SourceCLI {
		t.Fatalf("language = %q (%s)", effective.Values.Language, effective.Sources["commit.language"])
	}
	if effective.Values.CommitConfirm || effective.Sources["commit.confirm"] != SourceGlobal {
		t.Fatalf("commit.confirm = %v (%s)", effective.Values.CommitConfirm, effective.Sources["commit.confirm"])
	}
	if got := effective.Values.Include; len(got) != 1 || got[0] != "repo/**" {
		t.Fatalf("include = %#v, want repo replacement", got)
	}
	if effective.Sources["analysis.include"] != SourceRepo {
		t.Fatalf("include source = %s", effective.Sources["analysis.include"])
	}
	if !effective.Values.CommandsSet || len(effective.Values.Commands) != 1 {
		t.Fatalf("commands = %#v", effective.Values.Commands)
	}
	if got := effective.Values.SensitivePatterns; len(got) != 1 || got[0] != "*.private" {
		t.Fatalf("sensitive patterns = %#v", got)
	}
}

func TestResolveRejectsInvalidValuesBeforeHigherPriorityOverride(t *testing.T) {
	tests := []struct {
		name    string
		global  string
		repo    string
		cli     CLIOverrides
		wantErr string
	}{
		{
			name:    "repo cannot hide unsupported global schema version",
			global:  "schema_version = 2\n",
			repo:    "schema_version = 1\n",
			wantErr: "invalid global configuration: unsupported schema_version",
		},
		{
			name:    "repo cannot hide global glob outside root",
			global:  "[analysis]\ninclude = [\"../secret\"]\n",
			repo:    "[analysis]\ninclude = [\"src/**\"]\n",
			wantErr: "invalid global configuration: analysis glob must not escape",
		},
		{
			name: "CLI cannot hide invalid repo value",
			repo: "[commit]\nlanguage = \"invalid\"\n",
			cli: func() CLIOverrides {
				language := "ja"
				return CLIOverrides{Language: &language}
			}(),
			wantErr: "invalid repo configuration: commit.language must be en or ja",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			global := filepath.Join(root, "global.toml")
			repo := filepath.Join(root, "repo.toml")
			if test.global != "" {
				writeTestFile(t, global, test.global)
			}
			if test.repo != "" {
				writeTestFile(t, repo, test.repo)
			}
			_, err := Resolve(global, repo, root, test.cli)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Resolve() error = %v, want containing %q", err, test.wantErr)
			}
		})
	}
}

func TestDefaultsCoverEverySchemaKey(t *testing.T) {
	effective := Defaults()
	if effective.Values.Backend != "mlx" {
		t.Fatalf("default backend = %q", effective.Values.Backend)
	}
	if effective.Values.Context != "64k" || effective.Values.MaxTokens != 65536 {
		t.Fatalf("adaptive context defaults = %q/%d", effective.Values.Context, effective.Values.MaxTokens)
	}
	entries := effective.Entries()
	if len(entries) != len(schema) {
		t.Fatalf("entries = %d, schema = %d", len(entries), len(schema))
	}
	for key := range schema {
		entry, ok := entries[key]
		if !ok {
			t.Errorf("missing default for %s", key)
			continue
		}
		wantSource := SourceDefault
		if key == "verification.commands" {
			wantSource = SourceUnset
		}
		if entry.Source != wantSource {
			t.Errorf("%s source = %s, want %s", key, entry.Source, wantSource)
		}
	}
}

func TestResolveRejectsRemovedSinglePassAndUnverifiedModels(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(root, "global.toml")
	repo := filepath.Join(root, "repo.toml")
	for _, body := range []string{"[llm]\nplanner = \"single-pass\"\n", "[llm]\nbackend = \"ollama\"\n", "[llm]\nmodel = \"owner/model\"\n"} {
		writeTestFile(t, repo, body)
		if _, err := Resolve(global, repo, root, CLIOverrides{}); err == nil {
			t.Fatal("legacy or unverified model accepted")
		}
	}
	writeTestFile(t, repo, "[llm]\nplanner = \"three-phase\"\nmodel = \"mlx-community/Qwen3.5-4B-4bit\"\nmodel_revision = \"0e7ffd5c629ef7719d4cbc04069232580bfa9d9c\"\n")
	if e, err := Resolve(global, repo, root, CLIOverrides{}); err != nil || !SupportedPlannerModel(e.Values) {
		t.Fatalf("validated Qwen pin rejected: %v", err)
	}
}

func TestResolveAccepts64KContextCeiling(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(t.TempDir(), "config.toml")
	repo := filepath.Join(root, ".commiter.toml")
	writeTestFile(t, repo, "schema_version = 1\n[llm]\nplanner = \"three-phase\"\ncontext = \"64k\"\nmax_context_tokens = 65536\n")
	effective, err := Resolve(global, repo, root, CLIOverrides{})
	if err != nil {
		t.Fatal(err)
	}
	if effective.Values.Context != "64k" || effective.Values.MaxTokens != 65536 {
		t.Fatalf("context = %q/%d", effective.Values.Context, effective.Values.MaxTokens)
	}
}

func TestResolveRejectsEveryForbiddenSourceKey(t *testing.T) {
	tests := []struct {
		name    string
		scope   Source
		content string
	}{
		{"global verification autodetect", SourceGlobal, "[verification]\nautodetect = true\n"},
		{"global verification timeout", SourceGlobal, "[verification]\ntimeout_seconds = 10\n"},
		{"global verification commands", SourceGlobal, "[[verification.commands]]\nname = \"test\"\nargv = [\"true\"]\ncwd = \".\"\n"},
		{"repo commit confirm", SourceRepo, "[commit]\nconfirm = false\n"},
		{"repo push enabled", SourceRepo, "[push]\nenabled = false\n"},
		{"repo push confirm", SourceRepo, "[push]\nconfirm = false\n"},
		{"repo untracked", SourceRepo, "[analysis]\nuntracked = \"auto-safe\"\n"},
		{"repo metrics", SourceRepo, "[metrics]\npersist = true\n"},
		{"repo sensitive patterns", SourceRepo, "[safety]\nadditional_sensitive_patterns = []\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			global, repo := filepath.Join(root, "global.toml"), filepath.Join(root, "repo.toml")
			path := global
			if test.scope == SourceRepo {
				path = repo
			}
			writeTestFile(t, path, test.content)
			_, err := Resolve(global, repo, root, CLIOverrides{})
			if err == nil || !strings.Contains(err.Error(), "not allowed in "+string(test.scope)) {
				t.Fatalf("Resolve() error = %v", err)
			}
		})
	}
}

func TestCLIOverridesOnlySupportedKeys(t *testing.T) {
	language, model := "ja", "mlx-community/Qwen3.5-4B-4bit"
	commitConfirm, pushEnabled, pushConfirm, metrics := false, false, false, true
	root := t.TempDir()
	global := filepath.Join(root, "config.toml")
	writeTestFile(t, global, `[llm]
planner = "three-phase"
backend = "mlx"
`)
	effective, err := Resolve(global, "missing-repo", root, CLIOverrides{
		Language:       &language,
		CommitConfirm:  &commitConfirm,
		PushEnabled:    &pushEnabled,
		PushConfirm:    &pushConfirm,
		Model:          &model,
		MetricsPersist: &metrics,
	})
	if err != nil {
		t.Fatal(err)
	}
	if effective.Values.Model != model || effective.Values.ModelRevision != "0e7ffd5c629ef7719d4cbc04069232580bfa9d9c" ||
		effective.Values.ModelQuantization != "4bit" {
		t.Fatalf("unexpected CLI model pin: %#v", effective.Values)
	}
	for _, key := range []string{"commit.language", "commit.confirm", "push.enabled", "push.confirm", "llm.model", "llm.model_revision", "llm.model_quantization", "metrics.persist"} {
		if effective.Sources[key] != SourceCLI {
			t.Errorf("%s source = %s", key, effective.Sources[key])
		}
	}
}

func TestModelOverrideReplacesConfiguredQwenPin(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(root, "global.toml")
	repo := filepath.Join(root, "repo.toml")
	writeTestFile(t, repo, `[llm]
planner = "three-phase"
model = "mlx-community/Qwen3.5-4B-4bit"
model_revision = "0e7ffd5c629ef7719d4cbc04069232580bfa9d9c"
`)
	model := CandidateModel
	effective, err := Resolve(global, repo, root, CLIOverrides{Model: &model})
	if err != nil || effective.Values.Model != CandidateModel || effective.Values.ModelRevision != CandidateRevision ||
		effective.Values.ModelQuantization != "4bit" {
		t.Fatalf("expected Gemma pin: %#v, %v", effective.Values, err)
	}
	model = "owner/model"
	if _, err := Resolve(global, repo, root, CLIOverrides{Model: &model}); err == nil || !strings.Contains(err.Error(), "--model requires a supported pinned") {
		t.Fatalf("unsupported model override error = %v", err)
	}
}

func TestResolveRejectsInvalidConfigurationWithoutRawValue(t *testing.T) {
	tests := []struct {
		name    string
		scope   Source
		content string
		want    string
	}{
		{"unknown", SourceGlobal, "schema_version = 1\nsecret = \"RAW_SECRET\"\n", "unknown configuration key"},
		{"wrong type", SourceGlobal, "schema_version = 1\n[commit]\nconfirm = \"RAW_SECRET\"\n", "must have type boolean"},
		{"global verification", SourceGlobal, "schema_version = 1\n[verification]\nautodetect = true\n", "not allowed in global"},
		{"repo safety", SourceRepo, "schema_version = 1\n[push]\nenabled = false\n", "not allowed in repo"},
		{"unsupported version", SourceGlobal, "schema_version = 2\n", "unsupported schema_version"},
		{"empty commands", SourceRepo, "schema_version = 1\nverification.commands = []\n", "at least one command"},
		{"outside glob", SourceRepo, "schema_version = 1\n[analysis]\ninclude = [\"../secret\"]\n", "must not escape"},
		{"global endpoint only", SourceGlobal, "[llm]\nendpoint = \"http://127.0.0.1:11434\"\n", "remove llm.endpoint from global configuration"},
		{"repo endpoint with migrated planner", SourceRepo, "[llm]\nplanner = \"three-phase\"\nendpoint = \"RAW_SECRET\"\n", "remove llm.endpoint from repo configuration"},
		{"empty unknown table", SourceGlobal, "schema_version = 1\n[completely_unknown]\n", "unknown configuration key"},
		{"empty unknown nested table", SourceGlobal, "schema_version = 1\n[commit.unknown]\n", "unknown configuration key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			global, repo := filepath.Join(root, "missing-global.toml"), filepath.Join(root, "missing-repo.toml")
			path := global
			if test.scope == SourceRepo {
				path = repo
			}
			writeTestFile(t, path, test.content)
			_, err := Resolve(global, repo, root, CLIOverrides{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Resolve() error = %v, want containing %q", err, test.want)
			}
			if strings.Contains(err.Error(), "RAW_SECRET") {
				t.Fatalf("error exposed raw value: %v", err)
			}
		})
	}
}

func TestResolveAcceptsKnownEmptyTable(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(root, "global.toml")
	writeTestFile(t, global, "schema_version = 1\n[commit]\n")
	if _, err := Resolve(global, filepath.Join(root, "missing"), root, CLIOverrides{}); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestResolveRejectsVerificationCWDThroughSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, ".commiter.toml")
	writeTestFile(t, repo, `schema_version = 1
[verification]
[[verification.commands]]
name = "unsafe"
argv = ["true"]
cwd = "outside"
`)
	_, err := Resolve(filepath.Join(root, "missing"), repo, root, CLIOverrides{})
	if err == nil || !strings.Contains(err.Error(), "inside the repo root") {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestResolvePathsUsesXDGAndFallbacks(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "home", "tester")
	paths, err := ResolvePaths("/repo", func(key string) string {
		switch key {
		case "XDG_CONFIG_HOME":
			return "/xdg/config"
		case "XDG_STATE_HOME":
			return "/xdg/state"
		default:
			return ""
		}
	}, func() (string, error) { return home, nil })
	if err != nil {
		t.Fatal(err)
	}
	if paths.GlobalConfig != "/xdg/config/commiter/config.toml" || paths.StateDir != "/xdg/state/commiter" || paths.RepoConfig != "/repo/.commiter.toml" {
		t.Fatalf("paths = %#v", paths)
	}

	fallback, err := ResolvePaths("", func(string) string { return "" }, func() (string, error) { return home, nil })
	if err != nil {
		t.Fatal(err)
	}
	if fallback.GlobalConfig != filepath.Join(home, ".config", "commiter", "config.toml") || fallback.StateDir != filepath.Join(home, ".local", "state", "commiter") {
		t.Fatalf("fallback paths = %#v", fallback)
	}
}

func TestInitDoesNotOverwriteExistingConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeTestFile(t, path, "original\n")
	if err := Init(path, "replacement\n", 0o600); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Init() error = %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "original\n" {
		t.Fatalf("contents = %q", contents)
	}
}

func TestTemplatesSatisfyTheirAllowedScopes(t *testing.T) {
	root := t.TempDir()
	global, repo := filepath.Join(t.TempDir(), "config.toml"), filepath.Join(root, ".commiter.toml")
	writeTestFile(t, global, GlobalTemplate)
	writeTestFile(t, repo, RepoTemplate)
	if _, err := Resolve(global, repo, root, CLIOverrides{}); err != nil {
		t.Fatalf("Resolve(templates) error = %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestThreePhaseDefaultRequiresPinnedCandidateWithFixedContext(t *testing.T) {
	values := Defaults().Values
	if values.Planner != CandidatePlanner || values.Model != CandidateModel || values.ModelRevision != CandidateRevision || values.ModelQuantization != "4bit" {
		t.Fatalf("candidate defaults = %#v", values)
	}
	for _, field := range []string{"model", "revision", "quantization", "backend"} {
		t.Run(field, func(t *testing.T) {
			changed := values
			switch field {
			case "model":
				changed.Model = "owner/model"
			case "revision":
				changed.ModelRevision = strings.Repeat("b", 40)
			case "quantization":
				changed.ModelQuantization = "8bit"
			case "backend":
				changed.Backend = "ollama"
			case "context":
				changed.Context = "8k"
				changed.MaxTokens = 8192
			}
			if err := validateValues(changed, t.TempDir()); err == nil {
				t.Fatal("unsupported candidate accepted")
			}
		})
	}
	for _, ceiling := range []struct {
		context string
		max     int
	}{{"8k", 8192}, {"16k", 16384}, {"32k", 32768}, {"64k", 65536}, {"auto", 32768}} {
		changed := values
		changed.Context, changed.MaxTokens = ceiling.context, ceiling.max
		err := validateValues(changed, t.TempDir())
		valid := ceiling.max == 65536 && (ceiling.context == "64k" || ceiling.context == "auto")
		if (err == nil) != valid {
			t.Fatalf("context admission %s/%d: %v", ceiling.context, ceiling.max, err)
		}
	}
}

func TestLegacyV1PlannerCompatibility(t *testing.T) {
	for _, scope := range []string{"global", "repo"} {
		t.Run(scope, func(t *testing.T) {
			root := t.TempDir()
			global, repo := filepath.Join(root, "global.toml"), filepath.Join(root, "repo.toml")
			path := global
			if scope == "repo" {
				path = repo
			}
			writeTestFile(t, path, "schema_version = 1\n[llm]\nmodel = \"qwen3.5:4b-q4_K_M\"\ncontext = \"auto\"\n")
			if _, err := Resolve(global, repo, root, CLIOverrides{}); err == nil {
				t.Fatal("legacy config silently recovered")
			}
		})
	}
	root := t.TempDir()
	e, err := Resolve(filepath.Join(root, "absent"), filepath.Join(root, "missing"), root, CLIOverrides{})
	if err != nil || e.Values.Planner != CandidatePlanner {
		t.Fatalf("fresh defaults: %#v %v", e.Values, err)
	}
}

func TestLegacyMLXAndExplicitCandidateCompatibility(t *testing.T) {
	root := t.TempDir()
	global, repo := filepath.Join(root, "global.toml"), filepath.Join(root, "repo.toml")
	writeTestFile(t, global, "[llm]\nbackend = \"mlx\"\nmodel = \"owner/model\"\nmodel_revision = \""+strings.Repeat("a", 40)+"\"\nmodel_quantization = \"4bit\"\n")
	e, err := Resolve(global, repo, root, CLIOverrides{})
	if err == nil {
		t.Fatal("unverified legacy model accepted")
	}
	writeTestFile(t, global, GlobalTemplate)
	writeTestFile(t, repo, RepoTemplate)
	e, err = Resolve(global, repo, root, CLIOverrides{})
	if err != nil || e.Values.Planner != CandidatePlanner || e.Values.Model != CandidateModel {
		t.Fatalf("explicit migration: %#v %v", e.Values, err)
	}
	writeTestFile(t, global, GlobalTemplate)
	writeTestFile(t, repo, "[commit]\nlanguage = \"ja\"\n")
	e, err = Resolve(global, repo, root, CLIOverrides{})
	if err != nil || e.Values.Planner != CandidatePlanner {
		t.Fatalf("explicit global planner overridden: %#v %v", e.Values, err)
	}
}

func TestPinnedModelWithoutExplicitPlannerRequiresMigration(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "global.toml")
	writeTestFile(t, path, "[llm]\nbackend = \"mlx\"\nmodel = \""+CandidateModel+"\"\n")
	if _, err := Resolve(path, filepath.Join(root, "missing"), root, CLIOverrides{}); err == nil || !strings.Contains(err.Error(), "explicitly") {
		t.Fatalf("implicit migration: %v", err)
	}
}
