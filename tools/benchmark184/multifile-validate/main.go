// Validate a research plan with the unchanged production domain validator.
package main

import (
	"encoding/json"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"os"
)

func main() {
	var in struct {
		Plan    json.RawMessage `json:"plan"`
		FileIDs []string        `json:"file_ids"`
	}
	d := json.NewDecoder(os.Stdin)
	d.DisallowUnknownFields()
	if e := d.Decode(&in); e != nil {
		panic(e)
	}
	plan, v := planning.Validate(in.Plan, in.FileIDs, planning.SensitiveValues{}, planning.Japanese)
	if e := json.NewEncoder(os.Stdout).Encode(map[string]any{"accepted": len(v) == 0, "violations": v, "plan": plan}); e != nil {
		panic(e)
	}
}
