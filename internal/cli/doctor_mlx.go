package cli

import (
	"context"
	"errors"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

const mlxUnsupportedPlatformMessage = "MLX requires macOS on Apple Silicon (darwin/arm64)"

const mlxDoctorTimeout = 2 * time.Minute

func mlxPlatformSupported(goos, goarch string) bool {
	return goos == "darwin" && goarch == "arm64"
}

// findMLXHelper is the single lookup boundary used by setup and doctor. Package
// distribution may resolve a bundled helper here while retaining PATH lookup
// for development builds.
func findMLXHelper() (string, error) {
	return lookPath("commiter-mlx-helper")
}

func doctorMLX(values config.Values) map[string]map[string]any {
	goos, goarch := mlxPlatform()
	return doctorMLXFor(values, goos, goarch)
}

func doctorMLXFor(values config.Values, goos, goarch string) map[string]map[string]any {
	checks := map[string]map[string]any{}
	platformOK := mlxPlatformSupported(goos, goarch)
	checks["mlx_backend"] = check(platformOK, capabilityMessage(platformOK, "MLX is supported on macOS Apple Silicon", mlxUnsupportedPlatformMessage))
	if !platformOK {
		checks["mlx_helper"] = check(false, "not checked because this platform does not support MLX")
		checks["mlx_model"] = check(false, "not checked because this platform does not support MLX")
		checks["mlx_model_runtime"] = check(false, "not checked because this platform does not support MLX")
		checks["mlx_json_schema_grammar"] = check(false, "not checked because this platform does not support MLX")
		checks["mlx_constrained_generation"] = check(false, "not checked because this platform does not support MLX")
		return checks
	}
	helper, err := findMLXHelper()
	checks["mlx_helper"] = check(err == nil, capabilityMessage(err == nil, "MLX helper is executable", "MLX helper is unavailable; install commiter-mlx-helper"))
	if err != nil {
		checks["mlx_model"] = check(false, "not checked because the MLX helper is unavailable")
		checks["mlx_model_runtime"] = check(false, "not checked because the MLX helper is unavailable")
		checks["mlx_json_schema_grammar"] = check(false, "not checked because the MLX helper is unavailable")
		checks["mlx_constrained_generation"] = check(false, "not checked because the MLX helper is unavailable")
		return checks
	}
	store, err := newMLXModelStore()
	if err != nil {
		checks["mlx_model"] = check(false, err.Error())
		checks["mlx_model_runtime"] = check(false, "not checked because the model cache is unavailable")
		checks["mlx_json_schema_grammar"] = check(false, "not checked because the model cache is unavailable")
		checks["mlx_constrained_generation"] = check(false, "not checked because the model cache is unavailable")
		return checks
	}
	spec := mlxmodel.Spec{Repo: values.Model, Revision: values.ModelRevision, Quantization: values.ModelQuantization}
	modelPath, err := store.Ready(spec)
	checks["mlx_model"] = check(err == nil, message(err, "configured pinned MLX model is installed and complete"))
	if err != nil {
		checks["mlx_model_runtime"] = check(false, "not checked because the configured MLX model is unavailable")
		checks["mlx_json_schema_grammar"] = check(false, "not checked because the configured MLX model is unavailable")
		checks["mlx_constrained_generation"] = check(false, "not checked because the configured MLX model is unavailable")
		return checks
	}
	ctx, cancel := context.WithTimeout(context.Background(), mlxDoctorTimeout)
	defer cancel()
	probeOK, err := mlxCapabilityProbe(ctx, helper, values.Model, modelPath)
	if err != nil {
		if mlxFailureKind(err) == mlx.FailureStart {
			checks["mlx_helper"] = check(false, "MLX helper could not be started")
		}
		checks["mlx_model_runtime"] = check(false, "MLX helper could not load the configured model and tokenizer or complete the capability probe")
		grammarFailure := mlxStopReason(err) == mlx.StopReasonGrammar
		grammarMessage := "JSON Schema grammar capability could not be verified because the MLX probe failed"
		if grammarFailure {
			grammarMessage = "JSON Schema grammar compile or constrained generation failed"
		}
		checks["mlx_json_schema_grammar"] = check(false, grammarMessage)
		checks["mlx_constrained_generation"] = check(false, "constrained JSON generation probe failed")
		return checks
	}
	checks["mlx_model_runtime"] = check(true, "MLX helper loaded the configured model and tokenizer")
	checks["mlx_json_schema_grammar"] = check(true, "JSON Schema grammar compiled successfully")
	checks["mlx_constrained_generation"] = check(probeOK, capabilityMessage(probeOK, "constrained generation returned the required planning JSON shape without reasoning text", "constrained generation did not return the required JSON-only response"))
	return checks
}

func verifyMLXCapability(helper, model, modelPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), mlxDoctorTimeout)
	defer cancel()
	ok, err := mlxCapabilityProbe(ctx, helper, model, modelPath)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("MLX constrained JSON capability failed")
	}
	return nil
}

func mlxCapabilityProbe(ctx context.Context, helper, model, modelPath string) (bool, error) {
	fileIDs := []string{"F001"}
	schema, err := planning.Schema(fileIDs)
	if err != nil {
		return false, err
	}
	response, err := mlx.NewClient(helper).Generate(ctx, mlx.Request{
		Schema:       schema,
		Messages:     []llm.Message{{Role: "user", Content: `Return exactly this JSON commit plan without explanations or reasoning: {"commits":[{"type":"fix","scope":"cli","breaking":false,"summary":"check model","file_ids":["F001"]}]}`}},
		OutputTokens: 256, Model: model, ModelPath: modelPath,
	})
	if err != nil {
		return false, err
	}
	_, violations := planning.Validate([]byte(response.GeneratedJSON), fileIDs, planning.SensitiveValues{}, planning.English)
	return len(violations) == 0, nil
}

func mlxStopReason(err error) mlx.StopReason {
	var helperErr *mlx.Error
	if errors.As(err, &helperErr) {
		return helperErr.StopReason
	}
	return ""
}

func mlxFailureKind(err error) mlx.FailureKind {
	var helperErr *mlx.Error
	if errors.As(err, &helperErr) {
		return helperErr.Kind
	}
	return ""
}
