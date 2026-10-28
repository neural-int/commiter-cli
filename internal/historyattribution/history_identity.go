package historyattribution

import "encoding/json"

// Validate retained source identities again at the projection boundary. Equal
// text under different identities remains separate attribution evidence.
func uniqueHistorySources(h History) (History, error) {
	turns := make([]TurnContext, 0, len(h.Turns))
	operations := make([]Operation, 0, len(h.Operations))
	seenTurns := map[string]string{}
	seenOps := map[string]string{}
	for _, turn := range h.Turns {
		key, _ := json.Marshal([]string{turn.Thread, turn.Turn})
		body, _ := json.Marshal(turn)
		if previous, ok := seenTurns[string(key)]; ok {
			if previous != string(body) {
				return h, failure("input.schema", "conflicting_source_identity")
			}
			continue
		}
		seenTurns[string(key)] = string(body)
		turns = append(turns, turn)
	}
	for _, op := range h.Operations {
		key := operationKey(op)
		body, _ := json.Marshal(op)
		if previous, ok := seenOps[key]; ok {
			if previous != string(body) {
				return h, failure("input.schema", "conflicting_source_identity")
			}
			continue
		}
		seenOps[key] = string(body)
		operations = append(operations, op)
	}
	h.Turns = turns
	h.Operations = operations
	return h, nil
}
