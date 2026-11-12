# llama.cpp adapter implementation status

The normal release llama.cpp runtime is **unavailable**. An explicit native
trial now implements and exercises real server/CLI integration with Qwen3.5-4B
Q4_K_M. See the [2026-11-07 trial results](llamacpp-trial-2026-11-07.md). It does not complete
[Issue #190](https://github.com/neural-int/commiter-cli/issues/190). The
[v0.3.1 design](https://github.com/neural-int/commiter-cli/issues/190#issuecomment-6676772042)
requires qualification before normal CLI activation. The user subsequently
approved one local LLGuidance failure-handling patch against the fixed SHA;
this explicitly replaces the design's prohibition on independent patches only
for this repair. Silent GBNF, CPU, MLX or external-server fallback stays forbidden.
See the [patched build and regression results](llamacpp-fail-closed-2026-11-08.md).

## Upstream evidence and activation gate

The user-requested latest upstream trial is pinned to
`80d2c0c2c56e4a1bbf74be98f64b99927bf230bc`, with LLGuidance
`d795912fedc7d393de740177ea9ea761e7905774`. It is an experimental adoption,
not a qualified release runtime. The original unpatched trial's
[LLGuidance sampler](https://github.com/ggml-org/llama.cpp/blob/80d2c0c2c56e4a1bbf74be98f64b99927bf230bc/common/llguidance.cpp#L46-L59)
frees the grammar and returns after a mask-computation error without stopping
generation. Its [documentation](https://github.com/ggml-org/llama.cpp/blob/80d2c0c2c56e4a1bbf74be98f64b99927bf230bc/docs/llguidance.md#error-handling)
also describes continued generation after errors. The original trial additionally
reproduced HTTP 200 and valid JSON after an invalid LLGuidance grammar triggered
an engine error. Real Metal inference and CLI E2E passed in a separate cycle.

The [fixed patch](../../tools/llamacpp-trial/patches/0001-llguidance-fail-closed.patch)
now stops the owned server with exit 1 on LLGuidance compile, mask, consume,
reset, clone or existing matcher errors, and checks tokenizer initialization.
It never removes a failed grammar and resumes sampling. In this non-streaming,
one-slot, one-cycle process, the host discards the failed HTTP response and reaps
the child. This trades server availability for fail-closed behavior. It is not
a general-purpose streaming or shared-server recovery patch. Normal terminal
EOS handling is checked separately from actual constraint failures.

The recipe verifies the upstream SHA, patch SHA-256 and sole modified source
blob, while leaving the LLGuidance Rust source unchanged. `TrialBackend` requires
the exact patched `/props` build identity, so the previous unpatched server does
not satisfy its identity check. This marker is a trial identity check, not a
signature or cryptographic attestation of an arbitrary executable.

No release build is qualified or bundled, no production GGUF profile is
registered, and no public activation override is exposed. Logs or valid schema
validation alone cannot establish that LLGuidance enforced constraints throughout
generation. Native adapter failure regressions and real-server rejection now
provide separate evidence for this repair. The production `Backend` refuses activation even when called
directly; only an explicit internal `TrialBackend` executes the native trial.

## Implemented components and verification

| Area | Behavior | Evidence |
| --- | --- | --- |
| Cycle ownership | Named `llm.CycleBackend`, error-returning close; cleanup failure discards a plan and preserves a preceding generation error; required startup logs are awaited within the cycle deadline after HTTP readiness | Existing MLX/planner tests, delayed-log fake child and cleanup failure injection |
| Selection | `runtime`, legacy `backend` alias, explicit model path, global/repo/CLI precedence, inherited-path invalidation, local/managed origin | Config and CLI contract tests |
| CLI gate | Execution/setup refuse llamacpp; doctor lists bundle/model/Metal/UDS/LLGuidance/three-phase checks as unperformed | CLI and adapter refusal tests |
| GGUF storage | Canonical cache identity, trusted size/content digest, stable shared/exclusive flock, cancellation, staged install, explicit repair, recovery | Synthetic artifacts, concurrent setup subprocesses, killed writers, rename/restore/sync failure injection |
| UDS transport | Fixed operation paths, Unix-only dialing, no proxy or redirect, owner/mode/inode and OS peer-PID checks before transmission, 1 MiB request/4 MiB response/32 KiB header limits; request context bounds non-streaming response waits | Local test server and real native E2E on macOS; Linux support is for CI |
| Input admission | Exact retained native tokens, 65,536 context/4,096 output reservation, whole-record history selection, at most five preparations, individual-record measurement when byte estimates exclude candidates, optional-history shrinking on local request byte overflow | Synthetic tokenizer callbacks, UDS trial-cycle regressions and earlier real native token counts |
| Schema validation | Closed object/string vocabulary, required/allowed fields, enum, Unicode lengths, unknown-keyword/duplicate-key/trailing-data rejection | Synthetic invalid responses and real three-phase schema validation |

The GGUF store has no production model catalog or network fetcher. Setup does not
invoke it while the runtime is unqualified. `TrialBackend` now connects the
transport, native-input admission and schema helpers to a real fixed server.
The internal test factory exercises the actual CLI dry-run path; the release
factory remains closed. Nominal E2E success, semantic quality and production
failure handling are reported separately. The existing 4-file and 120-second
planner limits remain unchanged.

### History admission review follow-up

The PR #193 review identified two reproducible failures in input preparation:
different token densities could make the byte shortlist empty even when a record
fit, and an oversized all-history request ended preparation before mandatory-only
input was checked. Regression tests reproduced both failures before the repair.

All-history input is still prepared once when it fits. After token or local
request-byte overflow, the adapter validates mandatory-only input first. Byte
estimates shortlist candidates, but an empty shortlist now tries an unmeasured
individual record in the original priority order. Failed singleton candidates
are not repeated. Only a locally classified request-byte overflow permits further
history shrinking; transport, process, cancellation and invalid-token errors
still stop the cycle. Mandatory-input byte or token overflow remains fatal, and
the Git diff is never truncated. The 1 MiB transport limit is unchanged, including
for the final completion request; completion failures are not retried.

The original design bound remains three candidate preparations, at most five
total including all-history and mandatory-only input. This is a bounded search,
not an exhaustive scan or a guarantee of maximum history retention. If those
three candidates fail, already validated mandatory tokens are reused. Records
outside the candidate limit can still be unmeasured. Changing that bound is a
separate design decision.

Verification covers mixed token densities, mandatory-input rejection, bounded
fallback and immediate propagation of other errors. UDS trial-cycle tests cover
both an oversized apply-template request and a template response whose escaped
tokenize request exceeds the limit. They retain a smaller record, preserve the
mandatory diff and send exactly the accepted token IDs to completion without
tokenizing them again. Separate transport tests accept the exact 1 MiB request
boundary and classify and reject one extra byte before transmission. These follow-up tests use
synthetic tokenizers and a local test server, not a new GGUF inference run.

```sh
go test ./internal/llamacpp -run 'TestHistory|TestMandatoryByteOverflow|TestTrialShrinksHistory|TestUDSTransportClassifiesRequest' -count=1
```

Local MLX artifact/profile identity, parent-SIGKILL cleanup and release model
qualification remain open. This history repair does not enable the production
factory or change the native LLGuidance patch.

### Startup observation review follow-up

HTTP readiness and `/props` responses do not synchronize the subprocess log
copier. A fake child reproduced HTTP-ready/props-valid startup with the required
logs still absent. The adapter now waits for one model load, a Metal device and
full offload observations within the existing cycle context, polling at 25 ms
intervals. The deadline is not extended. Cancellation, child exit, duplicate
model loads, partial offload and constraint errors remain failures. `BeginCycle`
closes the process on observation failure, preserving the error.

Tests release startup logs only after HTTP readiness and `/props` inspection,
verify missing-log deadline propagation and process/socket cleanup, and reject
cancelled, exited or contradictory startup states. These are controlled fake
child tests; the review's timing risk has not been reproduced with a real GGUF
server, and this follow-up does not add a new native E2E measurement.

### Non-streaming response deadline review follow-up

The pinned server [waits for all results before returning a non-streaming
response](https://github.com/ggml-org/llama.cpp/blob/80d2c0c2c56e4a1bbf74be98f64b99927bf230bc/tools/server/server-context.cpp#L5423-L5427).
The previous transport-wide 10-second response-header timeout could therefore
interrupt prefill or decoding within the 120-second planning cycle. An 11-second
local completion response reproduced failure at 10 seconds before the repair.

The transport now uses the existing request context for the response wait,
without an independent response-header timer or an extended cycle deadline.
The readiness probe retains its 200 ms context, and connection ownership,
request/response/header byte limits and redirect/proxy restrictions are unchanged.
UDS regression tests verify successful response delivery after 11 seconds,
deadline propagation and explicit cancellation while completion is pending.
This is a local test-server reproduction, not a long-running GGUF generation
measurement. Production activation and the remaining qualification gates are
unchanged.

## Configuration

MLX remains the default. `runtime` and `backend` in the same layer must agree.
Legacy pinned MLX settings and the existing MLX cache layout remain valid. Context
may be omitted; the old `auto`/`64k` and 65,536 settings remain readable. Other
budgets are rejected. New templates omit these fixed context settings.

```toml
[llm]
planner = "three-phase"
runtime = "mlx"
model = "mlx-community/gemma-4-E4B-it-4bit"
# Global configuration only:
# model_path = "/absolute/path/to/mlx-model-directory"
```

Flags precede the command:

```sh
commiter --runtime mlx --model mlx-community/gemma-4-E4B-it-4bit \
  --model-path /absolute/path/to/mlx-model-directory setup
```

A local path requires an explicitly selected nonempty model identifier. Matching
a registered name still has local origin; the name does not establish artifact
identity. Local MLX trials use the existing validated profile and helper probe.
Unknown profiles fail because input format, thinking control and context have not
been established. Local setup never downloads, copies, moves or installs into the
managed cache. A probe success does not register the artifact or prove quality.

The parser can represent a future local GGUF trial:

```toml
[llm]
planner = "three-phase"
runtime = "llamacpp"
model = "my-trial"
model_path = "/absolute/path/to/model.gguf"
```

`config show --effective` shows its local origin. Execution/setup return a
qualification error without reading the model or starting a server. Doctor marks
each capability as not checked. No GGUF metadata check, load, inference or
download is attempted. An explicit CLI switch from MLX using both `--runtime`
and `--model` clears inherited MLX revision/quantization values; effective
configuration reports them as empty with source `unset`. This works with the
global and repo templates generated by init and does not modify those files.
Explicit non-MLX/MLX-field conflicts in a configuration layer, invalid lower-layer
values, and incomplete runtime-only overrides remain errors. A runtime-only change never selects another model. A
model/runtime change clears an inherited path before applying a new path.

## GGUF cache ownership and recovery

Identity hashes `commiter-gguf-cache-v1`, `llamacpp`, repository, lowercase
revision, quantization and sorted relative filenames, each terminated by NUL.
Expected size and digest come from the trusted registration. The initial format
accepts one GGUF file. Install holds an exclusive stable `.locks/<key>.lock`;
Open holds a shared lock until its lease closes. A future server cycle must keep
the lease until the process exits. Lock contention waits at most 30 seconds and
honors cancellation. Different keys progress independently.

Only verified `.stage-<key>/` content becomes `<key>/` by rename. Explicit repair
preserves `.previous-<key>/` until the new destination and parent are synced.
Setup restores an interrupted previous directory, discards uncommitted stage
content, or adopts a verified destination according to the design's recovery
table. Contradictory state and unexpected symlinks/file types are refused. Open
never repairs models. Same-UID malicious modification is not fully isolated.

## Constraint guarantees

| Constraint | LLGuidance generation | Post-generation helper | Existing Go planner/validator |
| --- | --- | --- | --- |
| Object/string, required/allowed keys | Unqualified | Enforced | Phase key checks retained |
| Grouping IDs and allowed labels | Unqualified | Required keys/enum | Complete assignment and ID restoration retained |
| Category type/evidence reference | Unqualified | Dynamic enum supported; not integrated | Evidence/group correspondence and unresolved rejection retained |
| Scope 1–16, summary 1–48 Unicode code points | Unqualified | Enforced | Language, one-line and forbidden-scope checks retained |
| Duplicate keys/trailing data | Unqualified | Rejected before validation | Existing strict parsing retained |
| Semantic correctness/privacy/Git snapshot | No claim | No semantic claim | Existing validators and snapshot checks retained |

Supported vocabulary is `type` (`object`/`string`), `properties`, `required`,
`additionalProperties: false`, string `enum`, `minLength` and `maxLength`.
Unknown keywords fail before generation. MLX scope token filtering and prefix/KV
optimization are not claimed for llamacpp. No cache hits or unobserved numeric
telemetry are reported by the gated adapter.

## Remaining work before activation and distribution

1. Maintain the fixed SHA and reviewed local patch as one build input. The
   observed compile/runtime failure paths now fail closed in regression tests;
   broader API, grammar and model coverage remains part of release qualification.
   Freeze native API fixtures, BOS/EOS behavior, template/thinking control and
   observable constraint-engine status.
2. Connect the experimentally verified supervisor/API to a trusted release
   bundle only after qualification. The trial covers one model load/cycle, one
   slot, independent prefill, Metal/context checks, bounded logs, environment
   allowlist and cancellation cleanup. Parent-death recovery remains unqualified.
3. Qualify managed profiles/local metadata, trusted artifact digests and input,
   thinking and context behavior. Connect explicit setup downloads and reader
   leases after qualification.
4. Extend the passing experimental CLI E2E to a packaged release and broader
   fixtures. Compare structural/semantic/latency/memory results separately from
   the current single-fixture observation and retain MLX regression checks.
5. Distribution/signature/notarization checks are deferred to subsequent CI/CD
   per the user's instruction. They are not prerequisites to this native trial
   and are not claimed as completed here.

No release workflow or distribution dependency is changed here. Issue #190 stays
open until these gates and the issue's completion criteria are met.
