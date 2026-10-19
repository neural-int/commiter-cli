// Command preflight checks existing three-phase entry points without inference.
// It intentionally uses an empty local directory: candidate rejection precedes
// model loading. This does not test candidate weights, tokenizers, or quality.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
)

type model struct {
	Repo     string `json:"repo"`
	Revision string `json:"revision"`
}

type result struct {
	Model          model           `json:"model"`
	ConfigAccepted bool            `json:"config_accepted"`
	ConfigError    string          `json:"config_error,omitempty"`
	HelperResponse json.RawMessage `json:"helper_response,omitempty"`
	HelperError    string          `json:"helper_error,omitempty"`
}

func main() {
	helper := flag.String("helper", "", "native helper built from the measured source")
	metadata := flag.String("metadata", "", "JSON model array with pinned repo and revision")
	flag.Parse()
	if *helper == "" || *metadata == "" {
		panic("helper and metadata are required")
	}
	data, err := os.ReadFile(*metadata)
	must(err)
	var models []model
	must(json.Unmarshal(data, &models))
	temporary, err := os.MkdirTemp("", "issue182-entrypoints-")
	must(err)
	// No recursive deletion: keep the small synthetic inputs for audit.
	empty := filepath.Join(temporary, "empty-model")
	must(os.Mkdir(empty, 0700))
	rows := []result{}
	unexpected := false
	for index, candidate := range models {
		path := filepath.Join(temporary, fmt.Sprintf("model-%d.toml", index))
		text := fmt.Sprintf("[llm]\nplanner = %q\nbackend = %q\nmodel = %q\nmodel_revision = %q\nmodel_quantization = %q\n", "three-phase", "mlx", candidate.Repo, candidate.Revision, "4bit")
		must(os.WriteFile(path, []byte(text), 0600))
		_, validation := config.Resolve(filepath.Join(temporary, "absent-global.toml"), path, temporary, config.CLIOverrides{})
		row := result{Model: candidate, ConfigAccepted: validation == nil}
		isBaseline := candidate.Repo == config.CandidateModel && candidate.Revision == config.CandidateRevision
		if (isBaseline && validation != nil) || (!isBaseline && validation == nil) {
			unexpected = true
		}
		if validation != nil {
			row.ConfigError = validation.Error()
		}
		if !isBaseline {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			response, callErr := mlx.NewClient(*helper).Generate(ctx, mlx.Request{
				Schema:        json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`),
				Messages:      []llm.Message{{Role: "user", Content: "Return JSON."}},
				ContextTokens: 16384, OutputTokens: 768,
				Model: candidate.Repo + "@" + candidate.Revision, ModelPath: empty,
				GenerationProfile: "bounded-grouping",
			})
			cancel()
			if callErr == nil || response.ErrorClass != "malformed_request" {
				unexpected = true
			}
			row.HelperResponse, err = json.Marshal(response)
			must(err)
			if callErr != nil {
				row.HelperError = callErr.Error()
			}
		}
		rows = append(rows, row)
	}
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{
		"unexpected_outcome":        unexpected,
		"scope":                     "entry-point compatibility only; no weights loaded; no generation",
		"created_utc":               time.Now().UTC().Format(time.RFC3339),
		"synthetic_input_directory": temporary, "results": rows,
	}))
	if unexpected {
		os.Exit(1)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
