package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func stringPtr(s string) *string { return &s }

func TestCLIChangesRuntimeFromInitializedMLXConfiguration(t *testing.T) {
	for _, scope := range []string{"global", "repo", "both"} {
		t.Run(scope, func(t *testing.T) {
			root := t.TempDir()
			global, repo := filepath.Join(root, "global.toml"), filepath.Join(root, "repo.toml")
			if scope != "repo" {
				if err := Init(global, GlobalTemplate, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if scope != "global" {
				if err := Init(repo, RepoTemplate, 0600); err != nil {
					t.Fatal(err)
				}
			}
			e, err := Resolve(global, repo, root, CLIOverrides{Runtime: stringPtr("llamacpp"), Model: stringPtr("trial"), ModelPath: stringPtr("/models/trial.gguf")})
			if err != nil {
				t.Fatal(err)
			}
			if e.Values.Backend != "llamacpp" || e.Values.Model != "trial" || e.Values.ModelSource != "local" || e.Values.ModelPath != "/models/trial.gguf" {
				t.Fatalf("selection=%+v", e.Values)
			}
			for _, key := range []string{"llm.model_revision", "llm.model_quantization"} {
				if entry := e.Entries()[key]; entry.Value != "" || entry.Source != SourceUnset {
					t.Fatalf("inherited MLX field survived: %s=%+v", key, entry)
				}
			}
			unchanged, err := Resolve(global, repo, root, CLIOverrides{})
			if err != nil || !SupportedPlannerModel(unchanged.Values) {
				t.Fatalf("initialized MLX configuration changed: %+v %v", unchanged.Values, err)
			}
		})
	}
}

func TestRuntimeLayersAndLocalModelOrigin(t *testing.T) {
	for _, tc := range []struct {
		name, global, repo                                      string
		cli                                                     CLIOverrides
		wantRuntime, wantModel, wantPath, wantSource, wantError string
	}{
		{name: "runtime alias", global: `[llm]
planner = "three-phase"
runtime = "mlx"
backend = "mlx"`, wantRuntime: "mlx", wantModel: CandidateModel, wantSource: "managed"},
		{name: "conflicting alias", global: `[llm]
planner = "three-phase"
runtime = "llamacpp"
backend = "mlx"`, wantError: "conflict"},
		{name: "same registered ID is local", cli: CLIOverrides{Model: stringPtr(CandidateModel), ModelPath: stringPtr("/models/local")}, wantRuntime: "mlx", wantModel: CandidateModel, wantPath: "/models/local", wantSource: "local"},
		{name: "explicit ID required", cli: CLIOverrides{ModelPath: stringPtr("/models/local")}, wantError: "explicit"},
		{name: "unknown local GGUF", cli: CLIOverrides{Runtime: stringPtr("llamacpp"), Model: stringPtr("trial"), ModelPath: stringPtr("/models/trial.gguf")}, wantRuntime: "llamacpp", wantModel: "trial", wantPath: "/models/trial.gguf", wantSource: "local"},
		{name: "model override drops path", global: `[llm]
planner = "three-phase"
model = "mlx-community/gemma-4-E4B-it-4bit"
model_path = "/models/local"`, cli: CLIOverrides{Model: stringPtr("mlx-community/Qwen3.5-4B-4bit")}, wantRuntime: "mlx", wantModel: "mlx-community/Qwen3.5-4B-4bit", wantSource: "managed"},
		{name: "same model keeps path", global: `[llm]
planner = "three-phase"
model = "mlx-community/gemma-4-E4B-it-4bit"
model_path = "/models/local"`, cli: CLIOverrides{Model: stringPtr(CandidateModel)}, wantRuntime: "mlx", wantModel: CandidateModel, wantPath: "/models/local", wantSource: "local"},
		{name: "runtime change clears path without choosing model", global: `[llm]
planner = "three-phase"
model = "trial"
runtime = "llamacpp"
model_path = "/models/trial.gguf"`, cli: CLIOverrides{Runtime: stringPtr("mlx")}, wantError: "supported pinned"},
		{name: "repo model drops global path", global: `[llm]
planner = "three-phase"
model = "trial"
runtime = "llamacpp"
model_path = "/models/trial.gguf"`, repo: `[llm]
runtime = "mlx"
model = "mlx-community/gemma-4-E4B-it-4bit"`, wantRuntime: "mlx", wantModel: CandidateModel, wantSource: "managed"},
		{name: "repo may not specify a path", repo: `[llm]
planner = "three-phase"
model_path = "/models/evil"`, wantError: "not allowed in repo"},
		{name: "empty runtime", cli: CLIOverrides{Runtime: stringPtr("")}, wantError: "must not be empty"},
		{name: "empty path", cli: CLIOverrides{ModelPath: stringPtr("")}, wantError: "must not be empty"},
		{name: "relative path", cli: CLIOverrides{Model: stringPtr("trial"), ModelPath: stringPtr("../model")}, wantError: "absolute"},
		{name: "no managed GGUF yet", cli: CLIOverrides{Runtime: stringPtr("llamacpp"), Model: stringPtr("trial")}, wantError: "no managed GGUF"},
		{name: "incomplete runtime override keeps MLX field conflict", global: `[llm]
planner = "three-phase"
model_revision = "475b9088d29754a3379866cf5aeb6b41acd313c2"`, cli: CLIOverrides{Runtime: stringPtr("llamacpp"), ModelPath: stringPtr("/models/trial.gguf")}, wantError: "MLX-specific"},
		{name: "same layer MLX field conflict cannot be hidden", global: `[llm]
planner = "three-phase"
runtime = "llamacpp"
model = "trial"
model_path = "/models/trial.gguf"
model_revision = "475b9088d29754a3379866cf5aeb6b41acd313c2"`, cli: CLIOverrides{Runtime: stringPtr("llamacpp"), Model: stringPtr("trial"), ModelPath: stringPtr("/models/other.gguf")}, wantError: "MLX-specific"},
		{name: "same layer MLX quantization conflict", global: `[llm]
planner = "three-phase"
runtime = "llamacpp"
model = "trial"
model_path = "/models/trial.gguf"
model_quantization = "4bit"`, wantError: "MLX-specific"},
		{name: "invalid inherited MLX pin cannot be hidden", global: `[llm]
planner = "three-phase"
model_revision = "invalid"`, cli: CLIOverrides{Runtime: stringPtr("llamacpp"), Model: stringPtr("trial"), ModelPath: stringPtr("/models/trial.gguf")}, wantError: "40-character"},
		{name: "invalid lower type cannot be hidden", global: `[llm]
runtime = false`, cli: CLIOverrides{Runtime: stringPtr("mlx")}, wantError: "type string"},
		{name: "file model registration resolves pin", global: `[llm]
planner = "three-phase"
runtime = "mlx"
model = "mlx-community/Qwen3.5-4B-4bit"`, wantRuntime: "mlx", wantModel: "mlx-community/Qwen3.5-4B-4bit", wantSource: "managed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			global, repo := filepath.Join(root, "global.toml"), filepath.Join(root, "repo.toml")
			if tc.global != "" {
				writeTestFile(t, global, tc.global)
			}
			if tc.repo != "" {
				writeTestFile(t, repo, tc.repo)
			}
			e, err := Resolve(global, repo, root, tc.cli)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error=%v want=%s", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if e.Values.Backend != tc.wantRuntime || e.Values.Model != tc.wantModel || e.Values.ModelPath != tc.wantPath || e.Values.ModelSource != tc.wantSource {
				t.Fatalf("unexpected selection: %+v", e.Values)
			}
			if e.Entries()["llm.runtime"].Value != e.Entries()["llm.backend"].Value {
				t.Fatal("alias differs")
			}
		})
	}
}
