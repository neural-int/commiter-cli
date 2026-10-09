// Research adapter: call the existing authoritative validator unchanged.
package main

import (
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"os"
)

type request struct {
	Candidate json.RawMessage `json:"candidate"`
	FileIDs   []string        `json:"file_ids"`
}

func main() {
	var requests []request
	if err := json.NewDecoder(os.Stdin).Decode(&requests); err != nil {
		panic(err)
	}
	rows := make([]map[string]any, 0, len(requests))
	for _, r := range requests {
		_, violations := planning.Validate(r.Candidate, r.FileIDs, planning.SensitiveValues{}, planning.Japanese)
		rows = append(rows, map[string]any{"valid": len(violations) == 0, "violations": violations})
	}
	if err := json.NewEncoder(os.Stdout).Encode(rows); err != nil {
		panic(err)
	}
}
