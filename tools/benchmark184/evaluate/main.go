// Evaluate saved fixed-fixture responses without model calls.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type inputGroup struct {
	ID    string `json:"group_id"`
	Files []struct {
		ID string `json:"file_id"`
	} `json:"files"`
}

// readTextGroups accepts the canonical two-message request or the registered
// three-message display. Shared data is never interpreted as the phase input.
func readTextGroups(request mlx.Request) ([]inputGroup, error) {
	var body string
	switch {
	case len(request.Messages) == 2 && request.Messages[0].Role == "system" && request.Messages[1].Role == "user":
		body = request.Messages[1].Content
	case len(request.Messages) == 3 && request.Messages[0].Role == "system" && request.Messages[1].Role == "user" && request.Messages[2].Role == "user":
		_, phase, found := strings.Cut(request.Messages[2].Content, "\n\nPhase input:\n")
		if !found {
			return nil, errors.New("missing Text phase input")
		}
		body = phase
	default:
		return nil, errors.New("unsupported Text request messages")
	}
	// Only the model display allows literal LF inside JSON strings. Re-escape
	// those bytes for the decoder; escaped code backslashes remain untouched.
	var canonical bytes.Buffer
	quoted, escaped := false, false
	for i := 0; i < len(body); i++ {
		c := body[i]
		if quoted && !escaped && c == '\n' {
			canonical.WriteString(`\n`)
			continue
		}
		canonical.WriteByte(c)
		if escaped {
			escaped = false
			continue
		}
		if quoted && c == '\\' {
			escaped = true
			continue
		}
		if c == '"' {
			quoted = !quoted
		}
	}
	var input struct {
		Groups []inputGroup `json:"groups"`
	}
	if err := json.Unmarshal(canonical.Bytes(), &input); err != nil {
		return nil, err
	}
	if len(input.Groups) == 0 {
		return nil, errors.New("Text input has no groups")
	}
	groups, files := map[string]bool{}, map[string]bool{}
	for _, group := range input.Groups {
		if group.ID == "" || groups[group.ID] || len(group.Files) == 0 {
			return nil, errors.New("invalid Text group membership")
		}
		groups[group.ID] = true
		for _, file := range group.Files {
			if file.ID == "" || files[file.ID] {
				return nil, errors.New("invalid Text file membership")
			}
			files[file.ID] = true
		}
	}
	return input.Groups, nil
}

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func main() {
	source := flag.String("source", "", "canonical or mixed-layout Text request")
	dir := flag.String("results", "", "completed arm")
	flag.Parse()
	b, e := os.ReadFile(*source)
	must(e)
	var request mlx.Request
	must(json.Unmarshal(b, &request))
	groups, e := readTextGroups(request)
	must(e)
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
	for _, g := range groups {
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
	result := map[string]any{"category_correct": correct, "category_total": len(groups), "host_validator_accepted": len(violations) == 0, "violations": violations, "plan": plan, "note": "fixed phase inputs; this evaluator does not establish upstream equality or propagated E2E; current production host validator"}
	b, e = json.MarshalIndent(result, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(*dir, "evaluation.json"), b, 0600))
}
