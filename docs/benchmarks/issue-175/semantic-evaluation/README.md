# Issue #175: semantic evaluation fixtures

## Public fixtures

`c01-synthetic-context.json` and `c02-synthetic-context.json` contain only
synthetic Codex-context examples. They contain no copied user request, session
response, repository diff, source snapshot, or personal path. Default tests and
opt-in evaluation runners use these examples.

Historical reports for Issues #175, #177, #178, and #180 were produced with
local-only inputs derived from private work history and repository snapshots.
Those original inputs are not distributed. Their recorded reports and
execution-contract hashes describe the original runs; the public synthetic
fixtures do not reproduce those measurements.

To repeat an original run when its local-only fixtures are available, set
`COMMITER_175_FIXTURE_DIR` or `COMMITER_180_FIXTURE_DIR` to the directory
containing `c01-fixture.json` and `c02-fixture.json`. Keep generated requests,
responses, logs, and telemetry outside Git or in CI Artifacts. See the
[benchmark publication policy](../../README.md).
