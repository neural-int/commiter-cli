# Issue #180: run-local model retention and Category/Text prefix gate

This opt-in experiment leaves production defaults unchanged. A uses the existing
Issue #178 iteration03 history-present results, including the incomplete C02.
B rearranges Category/Text messages; C also retains one loaded model/tokenizer
from Grouping through Text. Each generation has fresh grammar/native state and
fresh KV. D is not sampled when template tokenization diverges before shared
facts. Do not treat message JSON equality as a cache hit.

`TestIssue180Evaluation` uses production collection, full Raw Diff projection,
preparation and ThreePhaseGenerator. The actual Grouping response is saved and
validated before injecting the frozen reference partition for downstream
comparisons. This injection is not a successful end-to-end grouping result.

`prefix_audit.py` loads the fixed tokenizer and prepares actual messages, without
sampling. `run.py freeze` stores source/runtime/fixture/Gold hashes, order, request
cap and stop rules. `run.py run` verifies them and executes the fixed B/C cells
once. No retries, repairs, judge, download or wall-clock timeout. The recorded Go
build overlay replaces only the planner's shared deadline with cancellation;
the test binary runs with `-test.timeout=0`. Production keeps120seconds.

Build the Swift helper with the existing pinned dependencies, and copy its Metal
library beside it. Compile the opt-in Go test binary using the frozen overlay.
Capture both fixtures with `COMMITER_180_MODE=capture` first; run the prefix audit,
then freeze before sampling. All paths and commands are saved with each run.

Metrics separate load, input preparation, grammar setup, generation wall and the
dependency's prompt/generation times. `promptTime` includes first-token work,
not an isolated pure-prefill timer. MLX allocator peak is process/cycle cumulative
for C and process/phase for B; it is not system peak or OS RSS. Different scopes
must be labeled. A has no matching instrumentation/timeout/fixed-group treatment,
so A→B is reference-only, especially for C02. No general noninferiority or
production adoption follows from these two known fixtures and n=1 cells.
