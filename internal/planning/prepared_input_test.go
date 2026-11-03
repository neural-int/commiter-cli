package planning

import (
	"encoding/json"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Simulate helper-side preparation in scripted planner tests. This is not a
// tokenizer or a production size estimator; real preparation is tested in Swift.
func prepareCandidateInput(c *candidateClient, messages []llm.Message, options llm.Options) ([]llm.Message, []int, int, int, error) {
	count := func(m []llm.Message) (int, error) {
		if c.countInput != nil {
			return c.countInput(m, options)
		}
		return requiredHarness004Size(m) - 4096, nil
	}
	n, err := count(messages)
	if err != nil {
		return nil, nil, 0, 1, err
	}
	if options.HistoryInput == nil {
		if n > options.ContextTokens-options.OutputTokens {
			return nil, nil, 0, 1, llm.ErrInputTooLarge
		}
		return messages, nil, n, 1, nil
	}
	var shared map[string]any
	_ = json.Unmarshal([]byte(messages[1].Content), &shared)
	records := shared["codex_context"].([]any)
	all := []int{}
	for i := range records {
		all = append(all, i)
	}
	if n <= options.ContextTokens-options.OutputTokens {
		return messages, all, n, 1, nil
	}
	with := func(indices []int) []llm.Message {
		m := append([]llm.Message(nil), messages...)
		if len(indices) == 0 {
			delete(shared, "codex_context")
			m[0].Content = strings.TrimSuffix(m[0].Content, codexTrustInstruction)
		} else {
			selected := []any{}
			for _, i := range indices {
				selected = append(selected, records[i])
			}
			shared["codex_context"] = selected
		}
		m[1].Content = string(naturalJSON(shared))
		return m
	}
	selected := []int{}
	accepted := with(selected)
	n, err = count(accepted)
	if err != nil {
		return nil, nil, 0, 2, err
	}
	if n > options.ContextTokens-options.OutputTokens {
		return nil, nil, 0, 2, llm.ErrInputTooLarge
	}
	attempts := 2
	for i := range records {
		trial := append(append([]int(nil), selected...), i)
		m := with(trial)
		tokens, err := count(m)
		attempts++
		if err != nil {
			return nil, nil, 0, attempts, err
		}
		if tokens <= options.ContextTokens-options.OutputTokens {
			selected, accepted, n = trial, m, tokens
		}
	}
	return accepted, selected, n, attempts, nil
}
