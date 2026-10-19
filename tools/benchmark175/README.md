# Issue #175 paired semantic evaluation

The opt-in runner `internal/cli/issue175_evaluation_test.go` uses production Git collection, syntax analysis, grouping context, context preparation, ThreePhaseGenerator and its final validation. It does not invoke an App Server or mutate the source repositories. `prepare.py` creates two disposable Git repositories from historical blobs and the copied log patch. It fails rather than overwriting an existing output directory.

`capture` mode uses mechanical responses to inspect all reachable phase inputs. These responses are not Gold and are not model results. The backend adapter appends the current common context instruction to arm A, matching arm B exactly. Actual byte bounds and context presence are checked before every call; generation profiles, schemas, selected files and defaults are unchanged. Context tiers must match in the captured pair. Actual later schemas naturally depend on the model's earlier grouping/type.

`preflight.py` verifies paired input equality after removing only codex_context, then invokes an existing native tokenize-only helper. This older helper accepts a 16k envelope; the captured generation context is kept separately and token admission is compared against that actual ceiling. A known `input_too_large` rejection at that helper's 16k ceiling can still supply a token count. Such a receipt explicitly separates the helper rejection from arithmetic against the captured generation ceiling; it is not a successful production admission. Other failures or a count exceeding the actual generation ceiling stop preflight; receipts are append-free and must use a new output path on an environment repair. The production generation helper independently enforces its actual token budget at every call.

`TestIssue175Evaluation` defaults to the public synthetic context examples. To
reproduce historical local-only inputs, set `COMMITER_175_FIXTURE_DIR` to the
directory containing the original `c01-fixture.json` and `c02-fixture.json`.
Captured requests, responses, logs, and other run artifacts must remain outside
Git or in CI Artifacts.

`native_helper.py` forwards the production helper protocol unchanged and records elapsed time, stop state and child-process maximum RSS on macOS. It does not export thought tokens or helper stderr. RSS is not a GPU/Metal or system memory measurement. Each subprocess uses a cold helper lifecycle matching production. A shared per-cycle 120-second timeout and three calls per cycle are enforced by the generator/adapter. No retry or repair inference is performed.

Example preparation/capture (all paths must be explicit):

```sh
python3 tools/benchmark175/prepare.py --out /private/tmp/issue175-fixtures-NEW
COMMITER_175_MODE=capture \
COMMITER_175_REPOS=/private/tmp/issue175-fixtures-NEW/repos.json \
COMMITER_175_OUT=/private/tmp/issue175-capture-NEW \
COMMITER_175_HELPER=/absolute/path/to/production-helper \
go test ./internal/cli -run '^TestIssue175Evaluation$' -v -count=1
```

Before generation, freeze input/source/runtime/Gold hashes and successful preflight receipts. Set MODE=run, HELPER to the native_helper.py wrapper, REAL_HELPER to the production helper and NATIVE_ROOT to a fresh numeric-receipt directory. At most twelve generation requests are authorized for the first 2×2×1 comparison, including failed cycles; admission rejection is recorded with zero calls. `analyze.py` reports grouping, category and final-plan acceptance separately. Text meaning is reviewed against the pre-fixed rubric, not exact summary strings or an extra judging-model call.
