package planning

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

func harness004OptionalHistory(messages []llm.Message) (*llm.OptionalHistoryInput, error) {
	var shared map[string]any
	if err := json.Unmarshal([]byte(messages[1].Content), &shared); err != nil {
		return nil, err
	}
	if _, exists := shared["agent_history_contexts"]; exists {
		return attributed004History(messages)
	}
	value, exists := shared["codex_context"]
	if !exists {
		return nil, nil
	}
	records, ok := value.([]any)
	if !ok || len(records) == 0 || !strings.HasSuffix(messages[0].Content, codexTrustInstruction) {
		return nil, errors.New("invalid optional history projection")
	}
	before, after := map[string]any{}, map[string]any{}
	for key, value := range shared {
		if key == "codex_context" {
			continue
		}
		if naturalLess(key, "codex_context") {
			before[key] = value
		} else {
			after[key] = value
		}
	}
	fragment := func(value any) (string, error) { return llm.LiteralNewlineDisplay(naturalJSON(value)) }
	a, err := fragment(before)
	if err != nil {
		return nil, err
	}
	b, err := fragment(after)
	if err != nil {
		return nil, err
	}
	input := &llm.OptionalHistoryInput{Before: a[1 : len(a)-1], After: b[1 : len(b)-1],
		SystemWithoutHistory: strings.TrimSuffix(messages[0].Content, codexTrustInstruction)}
	for _, record := range records {
		text, err := fragment(record)
		if err != nil {
			return nil, err
		}
		input.Records = append(input.Records, text)
	}
	return input, nil
}

// Only ordered, unique source indices can be reflected into Text's frozen
// shared data. Omitted metadata is not accepted as "all history selected".
func selected004History(messages []llm.Message, indices []int) ([]any, error) {
	var shared map[string]any
	if err := json.Unmarshal([]byte(messages[1].Content), &shared); err != nil {
		return nil, err
	}
	records, ok := shared["codex_context"].([]any)
	if !ok || indices == nil {
		return nil, errors.New("missing history selection")
	}
	selected := []any{}
	previous := -1
	for _, index := range indices {
		if index <= previous || index >= len(records) {
			return nil, errors.New("invalid history selection")
		}
		selected = append(selected, records[index])
		previous = index
	}
	return selected, nil
}
