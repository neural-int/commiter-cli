package llm

import "errors"

// OptionalHistoryInput carries the shared message as non-overlapping display
// fragments. Large diffs are transmitted once. The helper chooses history and
// reuses the resulting prepared input for generation in the same request.
type OptionalHistoryInput struct {
	Before               string   `json:"before"`
	After                string   `json:"after"`
	Records              []string `json:"records"`
	SystemWithoutHistory string   `json:"system_without_history"`
}

var ErrInputTooLarge = errors.New("final model input exceeds context")
