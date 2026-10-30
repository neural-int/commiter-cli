// Evaluate saved fixed-fixture responses without model calls.
package main

import (
	"encoding/json"
	"flag"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"os"
	"path/filepath"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func main() {
	source := flag.String("source", "", "canonical Text request")
	dir := flag.String("results", "", "completed arm")
	flag.Parse()
	b, e := os.ReadFile(*source)
	must(e)
	var request mlx.Request
	must(json.Unmarshal(b, &request))
	var input struct {
		Groups []struct {
			ID    string `json:"group_id"`
			Files []struct {
				ID string `json:"file_id"`
			} `json:"files"`
		} `json:"groups"`
	}
	must(json.Unmarshal([]byte(request.Messages[1].Content), &input))
	outputs := map[string]map[string]string{}
	cats := map[string]map[string]string{}
	for index, dst := range map[string]any{"02": &cats, "03": &outputs} {
		b, e := os.ReadFile(filepath.Join(*dir, index+"-response.json"))
		must(e)
		var r mlx.Response
		must(json.Unmarshal(b, &r))
		must(json.Unmarshal([]byte(r.GeneratedJSON), dst))
	}
	plan := planning.Plan{SchemaVersion: 1}
	ids := []string{}
	correct := 0
	gold := map[string]string{"G001": "docs", "G002": "feat", "G003": "fix"}
	for _, g := range input.Groups {
		c := planning.Commit{Type: cats[g.ID]["type"], Scope: outputs[g.ID]["scope"], Summary: outputs[g.ID]["summary"]}
		for _, f := range g.Files {
			c.FileIDs = append(c.FileIDs, f.ID)
			ids = append(ids, f.ID)
		}
		plan.Commits = append(plan.Commits, c)
		if c.Type == gold[g.ID] {
			correct++
		}
	}
	b, e = json.Marshal(plan)
	must(e)
	_, violations := planning.Validate(b, ids, planning.SensitiveValues{}, planning.Japanese)
	result := map[string]any{"category_correct": correct, "category_total": 3, "host_validator_accepted": len(violations) == 0, "violations": violations, "plan": plan, "note": "fixed phase inputs; upstream equality required by runner; current production host validator"}
	b, e = json.MarshalIndent(result, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(*dir, "evaluation.json"), b, 0600))
}
