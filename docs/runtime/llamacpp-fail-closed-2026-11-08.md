# Fixed-SHA LLGuidance failure patch — 2026-11-08

The user approved a narrowly scoped exception to Issue #190 design v0.3.1:
build a fixed llama.cpp SHA with a local patch for the P1 LLGuidance failure
continuation. The patch fixes the reproduced continuation without waiting for
upstream. Normal release activation and PR #193 remain gated; the review's
local-MLX identity and history-selection P2 candidates are outside this repair.

## Build identity and patch boundary

| Input / output | Value |
| --- | --- |
| llama.cpp base | `80d2c0c2c56e4a1bbf74be98f64b99927bf230bc` |
| LLGuidance, unchanged | `d795912fedc7d393de740177ea9ea761e7905774` |
| Sole upstream file changed | `common/llguidance.cpp` |
| Patch | [`0001-llguidance-fail-closed.patch`](../../tools/llamacpp-trial/patches/0001-llguidance-fail-closed.patch) |
| Patch SHA-256 | `68aa603b9c43e96975f95a0ef506972cc7be35d7eba35f245207ea268942661a` |
| Patched source Git blob | `0c4ea16817eff73702ca508760bb6113a0d1005c` |
| `/props` build identity | `b1-80d2c0c2c56e4a1bbf74be98f64b99927bf230bc+commiter-llg-fail-closed-v1` |
| Server SHA-256, this local build | `7a4fbeee4c52f824078a2c52628a51be7e321cd5f306d8fedaa2d4429efdae78` |
| LLGuidance Cargo.lock SHA-256 | `3771a06633ce97358b9986bd9e6d54e377c131a6e073fb04e15625ce9b492b0d` |

Build tools remain CMake 4.4.5, Rust/Cargo 1.96.0 and AppleClang
21.0.0.21000401 on Darwin arm64 / macOS 26.6.2. CMake was re-fetched into the
temporary task directory under the existing approval and its archive digest
was rechecked against the [original trial](llamacpp-trial-2026-11-07.md).
There is no new Go dependency or system installation.

The [build recipe](../../tools/llamacpp-trial/build.sh) checks the base SHA and
patch digest before applying it to a clean checkout. A repeat build accepts only
the exact patched source blob and the sole expected modified file; unrelated
tracked or untracked changes are refused. Source checks run again after build,
and LLGuidance's SHA and clean worktree are checked separately. Applying the
patch to a separate clean checkout produced the expected blob without conflicts.
The build identity is passed explicitly to CMake and compared exactly by
`TrialBackend`. It distinguishes the previous unpatched artifact; it is not a
signature or attestation of arbitrary externally supplied executables. An
explicit negative run with the previous unpatched server was rejected at this
identity check before phase generation.

## Failure semantics

The upstream sampler callbacks cannot return a generation error to all their
callers. This dedicated server instead calls `std::_Exit(EXIT_FAILURE)` before
control returns to generation on a reported LLGuidance failure. The patch checks
grammar creation, tokenizer creation/clone, mask calculation/result, token
consumption, reset, clone and existing matcher error state. It never frees a
failed grammar and resumes unconstrained sampling. It emits only a constant
operation label on the added failure path, rather than grammar or prompt data.

This deliberately terminates the entire owned server. The host uses one slot,
one server per cycle and non-streaming completion, so a disconnected response
is discarded, the planning cycle fails, and process/socket cleanup runs. No
partial plan, automatic retry or fallback is added. `_Exit` avoids destructor
recovery on corrupted state, debugger attachment and a deliberate core dump.
This is not a shared-server or streaming error-recovery design.

Normal grammar completion requires separate handling: LLGuidance can enter its
successful stopped state immediately after the final byte and subsequently
force EOS in the mask. Consuming that EOS again returns an error. Native CLI
testing exposed this case, and a minimal literal-grammar test reproduced it.
The patch permits only an accepting, error-free stopped matcher's mask-approved
terminal token. An ordinary token after completion still stops the process.
It does not ignore failed `consume_token` calls or turn an error into EOS.

## Reproduction and regression results

The unchanged source was rebuilt before applying the patch. Sending the same
invalid Lark grammar to the real server reproduced `http_accepted=true`,
`engine_error_observed=true`, `returned_valid_json=true`.

The [native adapter regression](../../tools/llamacpp-trial/test-fail-closed.cpp)
compiles the actual adapter source against the real pinned Rust LLGuidance
library, using a synthetic single-byte vocabulary. It does not implement a
second sampler or expose production fault-injection flags. A valid recursive
grammar with `step_max_items=1` compiles successfully and causes a real parser
resource error during mask computation (`status=-1`). The unchanged adapter
returned to generation after this error. Invalid token consumption, reset and
clone of an errored matcher also returned control before the patch.

With the final patch:

| Case | Required and observed result |
| --- | --- |
| Open-ended valid grammar | Allowed token, EOS, reset and clone succeed |
| Literal grammar already complete | Its terminal EOS succeeds without a second consume |
| Invalid grammar compilation | Process exits 1 before returning to generation |
| Real mask-computation resource exhaustion | Process exits 1 before returning to generation |
| Invalid token consumption | Process exits 1 before returning to generation |
| Reset / clone / apply with an errored matcher | Each process exits 1 before returning to generation |
| Ordinary token after successful completion | Process exits 1 before returning to generation |
| Real server, invalid grammar | No successful HTTP completion/content; exit 1, child reaped and socket removed |
| Fresh real server after the failed test | Startup and cancellation cleanup succeed |

The real-server negative test takes the patched exit path independently of log
classification or whether an unconstrained answer might satisfy the schema.
Tokenizer/allocation null cases are guarded but were not separately injected.
The mask resource failure is exercised in the native adapter regression; it is
not claimed as an HTTP-server runtime fault-injection experiment.

The same pinned Unsloth Qwen3.5-4B Q4_K_M artifact was downloaded again and its
full SHA-256 verified: `00fe7986ff5f6b463e62455821146049db6f9313603938a70800d1fb69ef11a4`.
The patched actual-CLI dry-run E2E passed all three phases and the final
validator:

| Phase | Input tokens | Output tokens | Preparations | Prefill ms | Decode ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| Grouping | 821 | 10 | 1 | 2572.961 | 360.594 |
| Category | 865 | 36 | 1 | 2658.759 | 1475.483 |
| Text | 760 | 21 | 1 | 2383.566 | 916.204 |

Initialization took 2.151 s and the complete Go test 15.51 s in this single
observation. It verified one model load, Metal 33/33 layer offload, actual 65,536
context, independent prefill with zero cached tokens, no observed TCP socket,
unchanged HEAD/index and process/socket cleanup. These timings are not a
performance comparison. Semantic output remained `refactor` / `incremented base
value to 2`; the questionable category is preserved and no semantic-quality
improvement is claimed.

## Reproduce the patched build

Use already approved CMake and Rust tools and an empty task-specific source
destination. The commands fetch public source and use the explicit local model;
they do not send repository contents or prompts to a remote service.

```sh
git clone --no-checkout https://github.com/ggml-org/llama.cpp.git /absolute/task/llama.cpp
git -C /absolute/task/llama.cpp checkout --detach 80d2c0c2c56e4a1bbf74be98f64b99927bf230bc
tools/llamacpp-trial/build.sh /absolute/task/llama.cpp /absolute/task/build /absolute/path/to/cmake
tools/llamacpp-trial/test-fail-closed.sh /absolute/task/llama.cpp /absolute/task/build

COMMITER_TEST_LLAMA_SERVER=/absolute/task/build/bin/llama-server \
COMMITER_TEST_GGUF_MODEL=/absolute/path/to/Qwen3.5-4B-Q4_K_M.gguf \
go test ./internal/llamacpp -run '^TestInstalledTrialGrammarFailureAndCancellation$' -count=1 -v -timeout=180s

COMMITER_TEST_LLAMA_SERVER=/absolute/task/build/bin/llama-server \
COMMITER_TEST_GGUF_MODEL=/absolute/path/to/Qwen3.5-4B-Q4_K_M.gguf \
go test ./internal/cli -run '^TestInstalledLlamaCPPCLIDryRun$' -count=1 -v -timeout=180s
```

Release qualification still includes parent-death cleanup, managed/local model
profiles and broader fixtures. Distribution/signing/notarization remains
delegated to subsequent CI/CD as requested. P1's reproduced fail-open behavior
is fixed in this pinned patched trial; this result does not complete Issue #190
or authorize release activation.
