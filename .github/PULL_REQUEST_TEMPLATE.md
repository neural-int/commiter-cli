## Summary

<!-- Required. Briefly describe the problem and the approach taken. -->
<!-- Language rules: use an English Conventional Commits format for the PR title. Keep these section headings and required labels unchanged because PR Policy validates them. Explanatory text in the PR body may be written in Japanese or English. -->

## Related issue

<!-- Required. Use an auto-closing keyword when this PR resolves an issue, for example: `Closes #123`. If no issue applies, write `N/A: <reason>`. -->

## Changes

<!-- Required. List the concrete changes in this PR. -->

-

## Verification

<!-- Mark every standard check that was run. If a check is not run or is not applicable, explain it under "Skipped / not applicable". -->

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `go build ./cmd/commiter`
- [ ] `git diff --check`

Skipped / not applicable:

<!-- Required when any standard check above is unchecked. Explain what was skipped and why. -->

## Requirements impact

<!-- Required. Select exactly one based on what you know. You do not need to identify FR / SR / NFR / AC IDs or certify SRS translation alignment to submit the PR; maintainers make the final determination before merge. -->

- [ ] I believe this PR does not affect requirements.
- [ ] I believe this PR may affect requirements.
- [ ] I am unsure; maintainer review is required.

Notes:

<!-- Optional. Mention any behavior, specification, documentation, or compatibility area you think may be affected. -->

## Safety impact

<!-- Required. Describe any effect on Git-state handling, local-only LLM processing, sensitive-file handling, verification, hooks, or output safety. Write exactly `None.` when not applicable. -->
