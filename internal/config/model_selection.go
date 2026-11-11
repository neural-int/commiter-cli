package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// normalizeModelLayer validates explicit values before merging a layer. A
// changed runtime or model clears a lower-priority local path; no implicit
// model is selected when only the runtime changes.
func normalizeModelLayer(e *Effective, layer map[string]any, source Source) error {
	for _, key := range []string{"llm.runtime", "llm.backend", "llm.model", "llm.model_path", "llm.model_revision", "llm.model_quantization"} {
		if raw, exists := layer[key]; exists {
			value, ok := raw.(string)
			if !ok {
				return typeError(key, "string")
			}
			if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) {
				return fmt.Errorf("%s must not be empty or contain NUL", key)
			}
		}
	}
	legacy, hasLegacy := layer["llm.backend"]
	runtime, hasRuntime := layer["llm.runtime"]
	if hasLegacy && hasRuntime && legacy != runtime {
		return fmt.Errorf("llm.backend and llm.runtime conflict in %s configuration", source)
	}
	if hasLegacy {
		runtime, hasRuntime = legacy, true
		layer["llm.runtime"] = legacy
	}
	if hasRuntime && runtime != "mlx" && runtime != "llamacpp" {
		return fmt.Errorf("llm.runtime must be mlx or llamacpp; legacy Ollama settings are no longer supported")
	}
	if path, exists := layer["llm.model_path"]; exists {
		if source == SourceRepo {
			return fmt.Errorf("configuration key %q is not allowed in repo configuration", "llm.model_path")
		}
		if !filepath.IsAbs(path.(string)) {
			return fmt.Errorf("llm.model_path must be an absolute path")
		}
	}
	if revision, exists := layer["llm.model_revision"]; exists && !validHexRevision(revision.(string)) {
		return fmt.Errorf("llm.model_revision must be a 40-character commit hash for MLX")
	}
	if quantization, exists := layer["llm.model_quantization"]; exists && !validMLXQuantization(quantization.(string)) {
		return fmt.Errorf("llm.model_quantization must be none or a positive Nbit label for MLX")
	}
	model, hasModel := layer["llm.model"]
	// A complete CLI runtime/model override replaces inherited MLX identity.
	// Runtime-only overrides and explicit non-MLX/MLX-field conflicts remain
	// invalid; lower layers have already passed their own value validation.
	if source == SourceCLI && hasRuntime && hasModel && runtime != e.Values.Backend && e.Values.Backend == "mlx" {
		e.Values.ModelRevision, e.Values.ModelQuantization = "", ""
		e.Sources["llm.model_revision"], e.Sources["llm.model_quantization"] = SourceUnset, SourceUnset
	}
	if hasModel {
		if revision := pinnedPlannerRevision(model.(string)); revision != "" && e.Sources["llm.model_revision"] == SourceDefault {
			if _, explicit := layer["llm.model_revision"]; !explicit {
				e.Values.ModelRevision = revision
			}
		}
	}
	if (hasRuntime && runtime != e.Values.Backend) || (hasModel && model != e.Values.Model) {
		e.Values.ModelPath = ""
		e.Sources["llm.model_path"] = SourceUnset
	}
	return nil
}

func validateModelSelection(e *Effective) error {
	v := &e.Values
	if v.Backend != "mlx" && v.Backend != "llamacpp" {
		return fmt.Errorf("llm.runtime must be mlx or llamacpp")
	}
	if v.Backend != "mlx" {
		for _, key := range []string{"llm.model_revision", "llm.model_quantization"} {
			if e.Sources[key] != SourceDefault && e.Sources[key] != SourceUnset {
				return fmt.Errorf("%s is MLX-specific and conflicts with llm.runtime", key)
			}
		}
	}
	v.ModelSource = "managed"
	if v.ModelPath != "" {
		if e.Sources["llm.model"] == SourceDefault {
			return fmt.Errorf("llm.model_path requires an explicit nonempty llm.model")
		}
		v.ModelSource = "local"
		return nil
	}
	if !SupportedPlannerModel(*v) {
		return fmt.Errorf("three-phase planning requires a supported pinned Gemma or Qwen MLX 4bit model; no managed GGUF profile has passed the runtime qualification gate")
	}
	return nil
}
