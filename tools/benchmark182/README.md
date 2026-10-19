# Issue #182: model-default comparison

`H182-001-model-defaults` compares Gemma 4 E4B, Qwen3.5-4B, Qwen3.5-2B,
and Phi-4-mini-instruct through the existing three-phase generator and its
schema/validators. It uses an explicitly selected native-helper benchmark mode,
`--benchmark-model-defaults`. Normal CLI configuration keeps its pinned Gemma
contract. No new dependencies are required.

Qwen chat templates receive no `enable_thinking` override: the pinned 4B template
defaults to ON and the 2B template to OFF. Phi uses its normal template without
a reasoning-effort field. Gemma retains its established bounded-profile behavior.
The finite native-thought safety caps are Gemma 512/384/0 and Qwen4B 512/384/384
for Grouping/Category/Text; total output limits stay 768/512/768 for all models.
Forced thought closure is recorded separately; these caps are harness settings,
not claims about model-specified reasoning effort.

All models retain model/tokenizer and immutable grammar preparation within one
cycle. Every phase starts with a fresh native cache; no prefix reuse or KV
quantization override is used. Thus Gemma is measured again in H182-001;
historical H180-004 scores are not used as its baseline. Default cache classes,
dtypes and sizes differ by architecture and are measured, not artificially made
identical.

## Build and run

Build the helper with the repository's standard build procedure (or record the
actual Swift build configuration), then prepare an isolated fixture and test
binary. `build.py` changes only the four-file limit to16 in a Go build overlay;
it does not modify production source or extend the 120-second cycle timeout.

```sh
python3 tools/benchmark182/fixture.py /absolute/private/fixture
python3 tools/benchmark182/build.py /absolute/private/build
python3 tools/benchmark182/run.py freeze \
  --out /absolute/private/study \
  --models /absolute/private/models-local.json \
  --fixture /absolute/private/fixture \
  --helper /absolute/private/runtime/commiter-mlx-helper \
  --binary /absolute/private/build/evaluation.test \
  --overlay /absolute/private/build/overlay.json
python3 tools/benchmark182/run.py run --out /absolute/private/study
```

`models-local.json` is the public `docs/benchmarks/issue-182/models.json` array
with a `path` per entry pointing to a complete local snapshot at the pinned
revision. Download only model/config/tokenizer data; no model repository Python
code is executed. Keep model weights and generated study directories outside Git.
The freeze stage hashes runtime, study source, overlay, model data and fixture.
It also checks three system-monitor samples before inference.

The fixed schedule is one full-cycle warmup and two measured repeats per model,
with the final round in reverse model order: at most12 cycles /36 phase calls.
There are no retries, repairs, extra model judges or post-result tuning. Actual
Grouping feeds Category/Text; Gold groups are never injected. A phase failure
ends that cycle and remains in the denominator. Infrastructure failure stops the
study. Raw inputs, outputs and monitoring stay local; only aggregate results,
public fixture/Gold, contracts and reproduction code belong in Git.

The authored 16-file fixture contains four independent fixes with four related
files each. Reference groups are agent-authored, not independent human Gold.
Scores are exploratory and do not establish performance on real repositories.

## Earlier entry-point preflight

`go run ./tools/benchmark182 -helper /absolute/helper -metadata docs/benchmarks/issue-182/models.json`
reproduces rejection through the normal production entry points. It intentionally
uses an empty model directory and never generates tokens. Its zero exit means
expected restriction was reproduced, not inference readiness. The separate
benchmark mode above is the model-swapping entry point added after that preflight.
