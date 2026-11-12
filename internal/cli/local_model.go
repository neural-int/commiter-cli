package cli

import (
	"errors"
	"os"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
)

func validateLocalMLXModel(values config.Values) error {
	if !config.SupportedPlannerModel(values) {
		return errors.New("local MLX model input format, thinking control and 64k profile are unverified; use a supported pinned MLX profile")
	}
	info, err := os.Lstat(values.ModelPath)
	if err != nil || !info.IsDir() {
		return errors.New("llm.model_path for MLX must be an existing model directory")
	}
	return nil
}

func resolveMLXModelPath(values config.Values) (string, error) {
	if values.ModelSource == "local" {
		if err := validateLocalMLXModel(values); err != nil {
			return "", err
		}
		return values.ModelPath, nil
	}
	store, err := newMLXModelStore()
	if err != nil {
		return "", err
	}
	return store.Ready(mlxmodel.Spec{Repo: values.Model, Revision: values.ModelRevision, Quantization: values.ModelQuantization})
}
