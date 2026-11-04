package planning

import (
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Scripted admission for planner contract tests, not a copy of the runtime
// shortlist policy or a tokenizer. The real selection policy is tested in Swift.
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
	all := []int{}
	for i := range options.HistoryInput.RecordByteCounts {
		all = append(all, i)
	}
	if n <= options.ContextTokens-options.OutputTokens {
		return messages, all, n, 1, nil
	}
	with := func(indices []int) []llm.Message {
		m := append([]llm.Message(nil), messages...)
		m[1].Content = options.HistoryInput.SharedTemplate.Render(indices)
		prefix, _, _ := strings.Cut(m[2].Content, "\n\nPhase input:\n")
		m[2].Content = prefix + "\n\nPhase input:\n" + options.HistoryInput.PhaseTemplate.Render(indices)
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
	for i := range options.HistoryInput.RecordByteCounts {
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
