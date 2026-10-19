# Benchmark data policy

Keep the materials needed to reproduce and interpret a benchmark in Git:

- benchmark code and scripts;
- test fixtures and frozen Gold data;
- evaluation metrics and methodology;
- model, quantization and execution settings;
- concise reports with key scores, adoption decisions and limitations.

Keep run-specific raw inputs and model responses, detailed process or system
telemetry, resource snapshots, logs, source snapshots, and generated per-run
results out of Git. Store them locally or as CI Artifacts when they need to be
retained. A published report should include the relevant score and measurement
scope so readers do not need raw output files to understand the decision.

Use synthetic fixtures for tests by default. If a fixture comes from private
Codex history, a non-public repository, or personal data, do not publish it
unless its public release has been explicitly reviewed and authorized.

Historical reports for Issues #175, #177, #178, and #180 retain concise
conditions and scores, but their private source inputs are kept locally. The
synthetic public examples are for test execution and do not reproduce those
historical measurements. Historical execution-contract hashes identify the
omitted inputs; they are not hashes of the synthetic examples.

The `.gitignore` in this directory excludes common generated run artifacts.
Before publishing benchmark changes, verify that no ignored run output has been
force-added and that references in reports point to files tracked by Git.
