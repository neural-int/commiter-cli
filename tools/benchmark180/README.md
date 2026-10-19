# Issue #180: run-local model retention and Category/Text prefix gate

Current adopted baseline (2026-10-19): **H180-004 — H180-003 with 8-bit affine KV (group 64, four global layers; native rotating layers remain BF16)**.
The legacy condition label is D-T. Use `H180-004` to identify the configuration;
keep measurement iterations separate and record `harness_id` and
`baseline_harness_id` in future execution contracts.
Use `tools/benchmark180/h180004-helper.sh` as the evaluation helper to pin the 8-bit setting.
Use [the baseline contract](../../docs/benchmarks/issue-180/BASELINE.md) for future
comparisons. The instructions below describe the historical iteration01 B/C and
prefix-gate experiment; they do not select the current adopted baseline.


This opt-in experiment leaves production defaults unchanged. A uses the existing
Issue #178 iteration03 history-present results, including the incomplete C02.
B rearranges Category/Text messages; C also retains one loaded model/tokenizer
from Grouping through Text. Each generation has fresh grammar/native state and
fresh KV. D is not sampled when template tokenization diverges before shared
facts. Do not treat message JSON equality as a cache hit.

`TestIssue180Evaluation` uses production collection, full Raw Diff projection,
preparation and ThreePhaseGenerator. The actual Grouping response is validated
before injecting the frozen reference partition for downstream comparisons.
This injection is not a successful end-to-end grouping result.

`prefix_audit.py` loads the fixed tokenizer and prepares actual messages, without
sampling. `run.py freeze` stores source/runtime/fixture/Gold hashes, order, request
cap and stop rules. `run.py run` verifies them and executes the fixed B/C cells
once. No retries, repairs, judge, download or wall-clock timeout. The recorded Go
build overlay replaces only the planner's shared deadline with cancellation;
the test binary runs with `-test.timeout=0`. Production keeps120seconds.

Build the native helper and its Metal library from a clean checkout on Apple
Silicon macOS 14 or later:

```sh
./.github/scripts/build-mlx-helper.sh
export COMMITER_180_H180004_NATIVE_HELPER="$PWD/dist/commiter-mlx-helper"
export COMMITER_180_HELPER="$PWD/tools/benchmark180/h180004-helper.sh"
test -x "$COMMITER_180_H180004_NATIVE_HELPER"
test -s "$PWD/dist/mlx.metallib"
```

The build script uses the pinned Swift package dependencies and places
`commiter-mlx-helper` beside `mlx.metallib` under `dist/`. The H180-004 wrapper
requires `COMMITER_180_H180004_NATIVE_HELPER`, verifies that it is executable,
and pins `COMMITER_180_KV_BITS=8`. It exits with a clear error if the helper has
not been built or the environment variable is unset. `run.py freeze` records
hashes for the wrapper, native helper, Metal library, test binary and overlay;
set the same environment variable when freezing and running.

Compile the opt-in Go test binary using the frozen overlay. The default public
context examples are synthetic and validate the runner only. To reproduce the
historical measured inputs, set `COMMITER_180_FIXTURE_DIR` to the local-only
directory containing the original `c01-fixture.json` and `c02-fixture.json`.
Capture both cases with `COMMITER_180_MODE=capture` first; run the prefix audit,
then freeze before sampling. Keep raw requests, responses, logs, resource
snapshots and generated run directories outside the repository or in CI
Artifacts. Commit only synthetic examples, settings and concise reports with
the adopted scores.

Metrics separate load, input preparation, grammar setup, generation wall and the
dependency's prompt/generation times. `promptTime` includes first-token work,
not an isolated pure-prefill timer. MLX allocator peak is process/cycle cumulative
for C and process/phase for B; it is not system peak or OS RSS. Different scopes
must be labeled. A has no matching instrumentation/timeout/fixed-group treatment,
so A→B is reference-only, especially for C02. No general noninferiority or
production adoption follows from these two known fixtures and n=1 cells.
