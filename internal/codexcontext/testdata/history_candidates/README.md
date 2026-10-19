# Synthetic collector fixtures

The JSON files in this directory are synthetic test data authored for the
collector replay test. They contain no copied Codex session, repository diff,
personal path, or model response. They are not semantic Gold and make no claim
about real-world grouping quality.

The cases cover a completed edit matching one target, multiple matching paths,
an unrelated edit, a failed edit, and a move that is conservatively excluded.
Each record is hashed by `manifest.json` so fixture changes are reviewed with the
test expectation.

## Reproduce

```sh
go test ./internal/codexcontext -run TestHistoricalCandidateReplay -v -count=1
```

No source log, repository, App Server, or model connection is needed.
`history_candidates_test.go` replays the synthetic JSON through the collector and
checks adopted paths, record counts, and fail-closed behavior. It does not assess
LLM grouping/category/text quality or live App Server retrieval.
