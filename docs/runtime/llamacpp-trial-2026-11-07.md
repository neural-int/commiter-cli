# Qwen3.5-4B Q4_K_M native trial — 2026-11-07

The user requested the latest upstream revision, a real server connection, and
Qwen3.5-4B Q4 GGUF CLI E2E. The explicit native trial **passed all three phases
and the final validator**. Normal release CLI activation remains closed because
the negative test also reproduced generation continuing after a grammar error.

## Pinned artifacts and build

| Item | Value |
| --- | --- |
| llama.cpp | `80d2c0c2c56e4a1bbf74be98f64b99927bf230bc` — latest master at selection time |
| LLGuidance | `d795912fedc7d393de740177ea9ea761e7905774` — upstream's fixed v1.0.1 dependency |
| Upstream modifications | None; both source worktrees clean after build |
| Server version | `0.9.0-dev`, build 1, commit `80d2c0c` |
| Target / host OS | Darwin arm64 / macOS 26.6.2 |
| Compiler | AppleClang 21.0.0.21000401 |
| Rust / Cargo | 1.96.0 / 1.96.0 |
| CMake | 4.4.5, downloaded into the task's temporary directory with explicit user approval |
| CMake archive SHA-256 | `679e77a96b9ef9484cc6c1445255fc4f84206b48a65420efbedf16bdab55fbff` |
| Server SHA-256, this local build | `d15b8a1717e5dc83274037b30f5052b55c60b72c099ac95903ed58826362f1a6` |
| LLGuidance Cargo.lock SHA-256 | `3771a06633ce97358b9986bd9e6d54e377c131a6e073fb04e15625ce9b492b0d` |

Build options: Release, static libraries, `LLAMA_LLGUIDANCE=ON`,
`GGML_METAL=ON`, `GGML_METAL_EMBED_LIBRARY=ON`, tests/examples/UI/OpenSSL disabled.
The [build recipe](../../tools/llamacpp-trial/build.sh) pins both source revisions.
Rust/CMake are build tools; no Go module dependency was added.

| Model profile | Observed value |
| --- | --- |
| Repository | `unsloth/Qwen3.5-4B-GGUF` |
| Revision | `e87f176479d0855a907a41277aca2f8ee7a09523` |
| File | `Qwen3.5-4B-Q4_K_M.gguf` |
| Bytes | 2,740,937,888 |
| SHA-256 | `00fe7986ff5f6b463e62455821146049db6f9313603938a70800d1fb69ef11a4` |
| GGUF version / tensors | 3 / 426 |
| Architecture | `qwen35` |
| File type / quantization version | 15 (`Q4_K_M`) / 2 |
| Declared model context | 262,144 |
| Tokenizer model / pre-tokenizer | `gpt2` / `qwen35` |
| Chat template SHA-256 | `7f0e529032c25183bcd66c7f238da2d377f43be754a94e2725a58c4e16d2ed67` |

The size and content digest were obtained from the
[pinned Hugging Face artifact](https://huggingface.co/unsloth/Qwen3.5-4B-GGUF/tree/e87f176479d0855a907a41277aca2f8ee7a09523)
and verified after download. The model is an explicit local trial, not a new
production registration. A complete content digest is checked inside each trial
cycle before model load. Normal execution never downloads it.

## CLI E2E and resource observations

The test creates its own public synthetic Git repository, changes one Go
function's return value from 1 to 2, and calls the actual `cli.Run` dry-run path.
It injects `TrialBackend` through an internal test factory. The ordinary release
factory still refuses llamacpp: this test is not evidence that a packaged release
can select it. No user repository diff, Codex history or secret is used.

Observed in one instrumented pass:

| Phase | Exact input tokens | Generated tokens | Preparations | Native prefill ms | Native decode ms | Phase wall ms |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Grouping | 821 | 10 | 1 | 2723.295 | 358.426 | 3721.221 |
| Category | 865 | 36 | 1 | 2747.127 | 1486.854 | 4243.465 |
| Text | 760 | 21 | 1 | 2476.288 | 886.197 | 3373.257 |

Cycle initialization wall time, including artifact verification and readiness,
was 1.753 s. The complete Go test took 15.05 s. These are single warm-file-cache
observations, not stable latency targets or a comparison with MLX. A prior first
load took 8.245 s for initialization. Readiness is not reported as model load
duration; memory pressure, swap and MLX comparisons were not measured here.

The test verified:

- One model load across Grouping → Category → Text; the same child PID serves
  all phases. Host diagnostics observed the selected `MTL0` device and 33/33
  layers offloaded to GPU.
- `/props` reports a 65,536-token slot and one slot. Every phase reserves 4,096
  output tokens. The server has context shift and automatic memory fitting off.
- Thinking is off through the actual Qwen template. The prepared prompt ends in
  `<think>\n\n</think>\n\n`; native token IDs are used unchanged for completion.
- Native `tokens_evaluated` and `timings.prompt_n` equal the prepared token count.
  `cache_prompt=false` and `timings.cache_n=0` in every phase show independent
  prefill; no MLX prefix-cache optimization is claimed.
- The `%llguidance` grammar path handles the phase schemas. This syntax throws
  when the build lacks LLGuidance; there is no implicit GBNF fallback.
- Peer PID/inode/owner checks protect UDS connections. `lsof` found no TCP
  sockets for the child during the active cycle. This snapshot is not a complete
  OS-level audit of every possible outbound connection.
- All responses end with EOS, pass the post-generation phase Schema checks, and
  reach the existing final validator. The test confirms HEAD/index are unchanged
  and the owned socket directory is gone after cycle close and process Wait.

Structural acceptance does not establish semantic quality. The model classified
the return-value change as `refactor`, with summary `incremented base value to 2`.
That classification is questionable because the observable behavior changes.
The trial records this result without modifying the planner or relaxing its
validators to improve the score.

## Reproduced grammar failure

A separate real-server test sent deliberately invalid LLGuidance Lark grammar.
The server returned HTTP 200, the LLGuidance error was observed, and the returned
content was still valid JSON. Therefore, valid JSON is insufficient evidence of
successful constrained generation in this revision. This reproduces a grammar
compile failure; a runtime mask-computation failure was not separately injected.

The trial adapter rejects its observed LLGuidance errors, but log monitoring is
not a proof that every failure is stopped. The production qualification gate
remains closed. Real cancellation and socket/process cleanup also passed in the
separate negative test. Fake-child race tests cover startup failure, readiness
timeout, environment-variable isolation, cancellation and idempotent cleanup.
Parent SIGKILL recovery remains unqualified.

## Reproduction

Build the pinned source using already approved CMake and Rust tools, then run:

```sh
COMMITER_TEST_LLAMA_SERVER=/absolute/path/to/build/bin/llama-server \
COMMITER_TEST_GGUF_MODEL=/absolute/path/to/Qwen3.5-4B-Q4_K_M.gguf \
go test ./internal/cli -run '^TestInstalledLlamaCPPCLIDryRun$' -count=1 -v -timeout=180s

COMMITER_TEST_LLAMA_SERVER=/absolute/path/to/build/bin/llama-server \
COMMITER_TEST_GGUF_MODEL=/absolute/path/to/Qwen3.5-4B-Q4_K_M.gguf \
go test ./internal/llamacpp -run '^TestInstalledTrialGrammarFailureAndCancellation$' -count=1 -v -timeout=180s
```

Both tests skip when the explicit local assets are absent. Release packaging,
signing and notarization were deferred to subsequent CI/CD by the user's
instruction; they were not performed or claimed as completed in this trial.
