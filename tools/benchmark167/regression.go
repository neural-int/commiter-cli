package main

import (
	"encoding/json"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

// Hashes retain the serialized contract without persisting repository/model text.
type requestAudit struct {
	Bytes  int    `json:"request_bytes"`
	SHA    string `json:"request_sha256"`
	System string `json:"system_sha256"`
	User   string `json:"user_sha256"`
	Schema string `json:"schema_sha256"`
}

func auditRequest(request []byte, messages []llm.Message, schema json.RawMessage) *requestAudit {
	a := &requestAudit{Bytes: len(request), SHA: digest(request), Schema: digest(schema)}
	for _, m := range messages {
		if m.Role == "system" {
			a.System = digest([]byte(m.Content))
		}
		if m.Role == "user" {
			a.User = digest([]byte(m.Content))
		}
	}
	return a
}

func regressionFixtures() []fixture {
	return []fixture{
		{Name: "smoke-143", Files: []fixtureFile{{Path: "sample.go", Before: "package sample\nfunc value() int { return 1 }\n", After: "package sample\nfunc value() int { return 2 }\n"}}},
		fixtures()[0],
	}
}
