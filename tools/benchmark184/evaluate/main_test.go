package main

import (
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
)

func TestReadTextGroupsCanonicalAndMixedDisplay(t *testing.T) {
	canonical := `{"groups":[{"group_id":"G001","files":[{"file_id":"F1","raw_diff":"-old\n+new\n"}]}]}`
	display, err := llm.LiteralNewlineDisplay([]byte(canonical))
	if err != nil {
		t.Fatal(err)
	}
	for name, messages := range map[string][]llm.Message{
		"canonical": {{Role: "system", Content: "instructions"}, {Role: "user", Content: canonical}},
		"mixed":     {{Role: "system", Content: "instructions"}, {Role: "user", Content: `{"files":[{"file_id":"F1"}]}`}, {Role: "user", Content: "Instructions\n\nPhase input:\n" + display}},
	} {
		t.Run(name, func(t *testing.T) {
			groups, err := readTextGroups(mlx.Request{Messages: messages})
			if err != nil || len(groups) != 1 || groups[0].ID != "G001" || groups[0].Files[0].ID != "F1" {
				t.Fatalf("groups=%+v err=%v", groups, err)
			}
		})
	}
}

func TestReadTextGroupsRejectsMissingOrInvalidMembership(t *testing.T) {
	for _, body := range []string{`{}`, `{"groups":[]}`, `{"groups":[{"group_id":"G001","files":[]}]}`, `{"groups":[{"group_id":"G001","files":[{"file_id":"F1"},{"file_id":"F1"}]}]}`, `{"groups":[{"group_id":"G001","files":[{}]}]}`, `{"groups":[{"group_id":"G001","files":[{"file_id":"F1"}]}]} trailing`} {
		if _, err := readTextGroups(mlx.Request{Messages: []llm.Message{{Role: "system"}, {Role: "user", Content: body}}}); err == nil {
			t.Fatalf("accepted invalid body %q", body)
		}
	}
	if _, err := readTextGroups(mlx.Request{}); err == nil {
		t.Fatal("accepted missing messages")
	}
}
