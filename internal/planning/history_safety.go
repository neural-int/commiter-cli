package planning

import "strings"

// Safety projection precedes token admission. It never edits a final response,
// substitutes a request, or changes an ambiguous match to a unique match.
func safePhaseHistory(payload map[string]any, sensitive SensitiveValues) {
	safe := func(text string) bool {
		data := []byte(text)
		return strings.TrimSpace(text) != "" && !sensitive.Contains(data) && !ExtractSensitiveValues(data).Contains(data)
	}
	contexts, _ := payload["agent_history_contexts"].([]AgentHistoryContext)
	kept := []AgentHistoryContext{}
	refs := map[string]bool{}
	for _, record := range contexts {
		valid := len(record.FinalResponses) > 0
		for _, final := range record.FinalResponses {
			valid = valid && safe(final)
		}
		if valid {
			record.UserRequests = nil
			kept = append(kept, record)
			refs[record.ContextRef] = true
		}
	}
	payload["agent_history_contexts"] = kept
	partial := len(kept) != len(contexts)
	files := clonePhaseFiles(phaseHistoryFiles(payload))
	for i := range files {
		for j := range files[i].ChangeBlocks {
			block := &files[i].ChangeBlocks[j]
			candidates := []AgentHistoryCandidate{}
			omitted := false
			for _, candidate := range block.AgentHistoryCandidates {
				if refs[candidate.ContextRef] {
					candidates = append(candidates, candidate)
				} else {
					omitted = true
				}
			}
			block.AgentHistoryCandidates = candidates
			if omitted {
				block.AgentHistoryOmissionReasons = appendUniqueReason(block.AgentHistoryOmissionReasons, "candidate_context_omitted")
				if block.AgentHistoryMatchStatus == "matched" {
					block.AgentHistoryMatchStatus = "ambiguous"
				}
				block.AgentHistoryMatchReasons = appendUniqueReason(block.AgentHistoryMatchReasons, "candidate_context_omitted")
			}
		}
	}
	setPhaseHistoryFiles(payload, files)
	if commands, grouping := payload["agent_command_execution_contexts"].([]AgentCommandExecutionContext); grouping {
		paths := []string{}
		for _, file := range files {
			for _, path := range []*string{file.OldPath, file.NewPath} {
				if path != nil {
					paths = append(paths, *path)
				}
			}
		}
		kept := []AgentCommandExecutionContext{}
		for _, command := range commands {
			if refs[command.ContextRef] && safe(CommandSafetyText(command.Command, paths)) {
				kept = append(kept, command)
			}
		}
		partial = partial || len(kept) != len(commands)
		payload["agent_command_execution_contexts"] = kept
		for _, file := range files {
			for _, block := range file.ChangeBlocks {
				partial = partial || len(block.AgentHistoryOmissionReasons) > 0
			}
		}
		status := "complete"
		if partial {
			status = "partial"
		}
		payload["agent_history_projection"] = map[string]string{"status": status}
	}
}
func appendUniqueReason(reasons []string, reason string) []string {
	if !containsString(reasons, reason) {
		return append(reasons, reason)
	}
	return reasons
}
