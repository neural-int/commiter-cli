package planning

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// The host compiles reference-safe alternatives once. Large Git diffs occur
// once on IPC, including when the helper must try several history shortlists.
type historyDisplayCompiler struct {
	contexts     map[string]int
	commandStart int
	counts       []int
}

func attributed004History(messages []llm.Message) (*llm.OptionalHistoryInput, error) {
	var shared, phase map[string]any
	if err := json.Unmarshal([]byte(messages[1].Content), &shared); err != nil {
		return nil, err
	}
	_, body, ok := strings.Cut(messages[2].Content, "\n\nPhase input:\n")
	if !ok {
		return nil, errors.New("missing phase input")
	}
	if err := json.Unmarshal([]byte(body), &phase); err != nil {
		return nil, err
	}
	compiler := historyDisplayCompiler{contexts: map[string]int{}, commandStart: 0}
	contexts, _ := shared["agent_history_contexts"].([]any)
	for _, record := range contexts {
		context := record.(map[string]any)
		ref, _ := context["context_ref"].(string)
		if ref == "" {
			return nil, errors.New("empty history context reference")
		}
		compiler.contexts[ref] = len(compiler.counts)
		compiler.counts = append(compiler.counts, len(naturalJSON(record)))
	}
	compiler.commandStart = len(compiler.counts)
	commands, _ := phase["agent_command_execution_contexts"].([]any)
	for _, record := range commands {
		compiler.counts = append(compiler.counts, len(naturalJSON(record)))
	}
	if len(compiler.counts) == 0 {
		return nil, nil
	}
	sharedTemplate := compiler.value(shared)
	phaseTemplate := compiler.value(phase)
	return &llm.OptionalHistoryInput{Records: []string{}, SystemWithoutHistory: messages[0].Content,
		RecordByteCounts: compiler.counts, SharedTemplate: &sharedTemplate, PhaseTemplate: &phaseTemplate}, nil
}

func literalTemplate(text string) llm.DisplayTemplate { return llm.DisplayTemplate{Literal: text} }
func listTemplate(open, close, separator string, parts []llm.DisplayTemplate) llm.DisplayTemplate {
	return llm.DisplayTemplate{Parts: []llm.DisplayTemplate{literalTemplate(open), {Parts: parts, Separator: separator}, literalTemplate(close)}}
}
func optionalTemplate(node llm.DisplayTemplate, indices []int, fallback *llm.DisplayTemplate) llm.DisplayTemplate {
	node.Require = indices
	if fallback != nil {
		node.IfMissing = []llm.DisplayTemplate{*fallback}
	}
	return node
}
func (compiler historyDisplayCompiler) allIndices() []int {
	indices := make([]int, len(compiler.counts))
	for i := range indices {
		indices[i] = i
	}
	return indices
}
func (compiler historyDisplayCompiler) value(value any) llm.DisplayTemplate {
	switch value := value.(type) {
	case []any:
		children := []llm.DisplayTemplate{}
		for _, item := range value {
			children = append(children, compiler.value(item))
		}
		return listTemplate("[", "]", ",", children)
	case map[string]any:
		keys := []string{}
		for key := range value {
			keys = append(keys, key)
		}
		children := []llm.DisplayTemplate{}
		candidateIndices := []int{}
		if candidates, ok := value["agent_history_candidates"].([]any); ok {
			for _, candidate := range candidates {
				ref, _ := candidate.(map[string]any)["context_ref"].(string)
				if index, exists := compiler.contexts[ref]; exists {
					candidateIndices = append(candidateIndices, index)
				}
			}
		}
		_, hasOmissions := value["agent_history_omission_reasons"]
		if len(candidateIndices) > 0 && !hasOmissions {
			keys = append(keys, "agent_history_omission_reasons")
		}
		sort.Slice(keys, func(i, j int) bool { return naturalLess(keys[i], keys[j]) })
		for _, key := range keys {
			if key == "agent_history_omission_reasons" && !hasOmissions {
				fallback := llm.DisplayTemplate{Parts: []llm.DisplayTemplate{literalTemplate("\"agent_history_omission_reasons\":"), compiler.value([]any{"candidate_context_omitted", "input_budget_exceeded"})}}
				children = append(children, optionalTemplate(literalTemplate(""), candidateIndices, &fallback))
				continue
			}
			field := compiler.value(value[key])
			switch key {
			case "agent_history_contexts", "agent_history_candidates", "agent_command_execution_contexts":
				records, _ := value[key].([]any)
				children := []llm.DisplayTemplate{}
				for position, record := range records {
					object := record.(map[string]any)
					ref, _ := object["context_ref"].(string)
					index, exists := compiler.contexts[ref]
					if !exists {
						continue
					}
					requires := []int{index}
					if key == "agent_command_execution_contexts" {
						requires = append(requires, compiler.commandStart+position)
					}
					children = append(children, optionalTemplate(compiler.value(record), requires, nil))
				}
				field = listTemplate("[", "]", ",", children)
			case "agent_history_match_status":
				if value[key] == "matched" && len(candidateIndices) > 0 {
					fallback := compiler.value("ambiguous")
					field = optionalTemplate(field, candidateIndices, &fallback)
				}
			case "agent_history_match_reasons", "agent_history_omission_reasons":
				if len(candidateIndices) > 0 {
					reasons, _ := value[key].([]any)
					omitted := append([]any{}, reasons...)
					extra := []string{"candidate_context_omitted"}
					if key == "agent_history_omission_reasons" {
						extra = append(extra, "input_budget_exceeded")
					}
					for _, reason := range extra {
						if !containsAnyString(omitted, reason) {
							omitted = append(omitted, reason)
						}
					}
					fallback := compiler.value(omitted)
					field = optionalTemplate(field, candidateIndices, &fallback)
				}
			case "agent_history_projection":
				fallback := compiler.value(map[string]any{"status": "partial"})
				field = optionalTemplate(field, compiler.allIndices(), &fallback)
			}
			children = append(children, llm.DisplayTemplate{Parts: []llm.DisplayTemplate{literalTemplate(string(naturalJSON(key))), literalTemplate(":"), field}})
		}
		return listTemplate("{", "}", ",", children)
	default:
		return literalTemplate(string(naturalJSON(value)))
	}
}
func containsAnyString(values []any, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func selectedAttributed004Shared(input *llm.OptionalHistoryInput, indices []int) (map[string]any, error) {
	if indices == nil {
		return nil, errors.New("missing history selection")
	}
	previous := -1
	for _, index := range indices {
		if index <= previous || index >= len(input.RecordByteCounts) {
			return nil, errors.New("invalid history selection")
		}
		previous = index
	}
	var shared map[string]any
	err := json.Unmarshal([]byte(input.SharedTemplate.Render(indices)), &shared)
	return shared, err
}
