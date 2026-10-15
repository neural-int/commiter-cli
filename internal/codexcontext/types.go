// Package codexcontext reads optional local Codex evidence without changing Git.
package codexcontext

import "time"

const (
	Timeout         = 5 * time.Second
	MaxReceiveBytes = 4 * 1024 * 1024
	MaxContextBytes = 8 * 1024
	PageSize        = 20
)

// Record is the only log projection sent to the planner. Source IDs stay local.
type Record struct {
	Paths         []string `json:"paths"`
	UserRequest   string   `json:"user_request"`
	FinalResponse string   `json:"final_response,omitempty"`
}

type Target struct {
	Path string
	Diff string
	// FinalDiff is the authoritative HEAD-to-worktree change being committed.
	// When area diffs are supplied, cancelled intermediate edits cannot qualify.
	FinalDiff string
}

type Result struct {
	Records []Record
	// Status distinguishes retrieval failures from readable but unrelated logs.
	Status string
}
