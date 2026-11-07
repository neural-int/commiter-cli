package cli

import (
	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/llamacpp"
)

func doctorLlamaCPP(values config.Values) map[string]map[string]any {
	checks := map[string]map[string]any{
		"llamacpp_backend":      check(false, llamacpp.QualificationMessage),
		"llamacpp_model_source": check(true, values.ModelSource+" trial selected; the model name does not imply a registered or validated artifact"),
	}
	for _, name := range []string{"bundle", "model", "metal", "uds", "llguidance", "three_phase"} {
		checks["llamacpp_"+name] = check(false, "not checked because no runtime build has passed qualification")
	}
	return checks
}
