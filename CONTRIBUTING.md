# Contributing to commiter-cli

English | [日本語](CONTRIBUTING_ja.md)

Thank you for considering a contribution to `commiter-cli`.

`commiter-cli` is designed around reproducible Git-state handling, local-only LLM analysis, and conservative safety boundaries. Contributions should preserve those properties rather than trading them away for convenience.

## Before You Start

Read the software requirements specification before changing behavior:

- [Software Requirements Specification (English)](SOFTWARE_REQUIREMENTS_SPECIFICATION_en.md)
- [ソフトウェア要求仕様書 (日本語)](SOFTWARE_REQUIREMENTS_SPECIFICATION.md)

Before opening an issue or pull request, review existing issues and the SRS. If a change needs a substantial specification update or a design decision, discuss it in an issue first.

For v1, the primary target is macOS 14 or later on Apple Silicon. The CLI is implemented primarily in Go and uses system Git and Ollama at runtime.

If you know that a change affects behavior described by the SRS, update the relevant specification or documentation when practical and call out any uncertainty in the pull request. Contributors are not required to determine exact FR / SR / NFR / AC IDs or certify English/Japanese SRS alignment in order to submit a pull request. Maintainers make the final determination before merge.

## Issues

### Issue labels

Apply one primary type label in principle.

- `enhancement`: new features, implementation, or improvements
- `bug`: bug fixes
- `documentation`: documentation additions or changes

Add the following secondary labels only when needed, in addition to the primary type.

- `good first issue`: issues that first-time contributors can take on
- `help wanted`: issues seeking help from external contributors

If the primary type is unclear, do not force a label. Create new labels or make substantial changes to existing labels only after confirming the policy in an issue.

### Issue templates

Contributor-facing issue forms are intentionally limited to a small set of clear entry points.

- Bug report: report a reproducible defect or regression
- Feature / Enhancement: propose a new feature or improvement based on a concrete problem and use case
- Documentation: report missing, incorrect, outdated, unclear, or mismatched documentation
- Design / RFC: discuss a significant behavior, architecture, compatibility, or specification change before implementation
- Maintainer task: internal Implementation, Decision, Verification, release, CI, or other maintenance work that does not fit a public contributor template

Use Design / RFC when a proposal may substantially affect the SRS, CLI or configuration compatibility, Git-state behavior, security boundaries, model-backend architecture, or another cross-cutting contract.

The GitHub issue chooser does not offer a blank issue (`blank_issues_enabled: false`). External contributors should use Bug report, Feature / Enhancement, Documentation, or Design / RFC. Maintainers should use the Maintainer task form for internal work. If a template is still a poor fit, create the issue with `gh issue create` instead of expecting a blank-issue button in the web UI.

Implementation issues generated from the SRS should include the related requirements, dependencies, success and failure conditions, implementation approach, and verification conditions in the body. Maintainer-created Decision and Verification issues should include content that matches their purpose and expected outcome. For ordinary contributor reports, fill in the selected template fields as specifically as you can.

Security vulnerabilities must not be reported through public issue forms. Follow [SECURITY.md](SECURITY.md) and use GitHub Private Vulnerability Reporting instead.

## Development Setup

Requirements:

- Go 1.23 or later
- Git
- Ollama when working on or testing Ollama-dependent behavior

Clone the repository and download dependencies:

```sh
git clone https://github.com/neural-int/commiter-cli.git
cd commiter-cli
go mod download
```

Run the standard checks:

```sh
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/commiter
```

Do not commit generated binaries or unrelated local files.

## Making Changes

Create a focused branch from the latest `main` and keep each pull request limited to one coherent purpose. In principle, one pull request should correspond to one issue.

Prefer small changes that fit the existing package boundaries. Avoid unrelated refactors, formatting-only churn, or dependency additions that are not needed for the requested behavior.

For Go code:

- Run `gofmt`.
- Prefer explicit errors and deterministic behavior.
- Preserve argument boundaries when executing subprocesses; do not replace argv-based execution with shell-string execution.
- Add or update tests for behavior changes and regressions.
- Keep terminal output safe for untrusted repository paths, Git output, hook output, verification output, and LLM output.

## Safety-sensitive Changes

Several behaviors are intentional security invariants. Changes touching them require particular care and corresponding tests.

Do not weaken these guarantees without an explicit specification change:

- repository content used for LLM analysis stays within loopback/local processing;
- clearly sensitive files are automatically excluded and cannot be opted in through a generic override;
- sensitive candidates are confirmed before their contents are read;
- out-of-scope staged content and staged selection are preserved;
- verification trust remains repository-scoped;
- Git hooks are respected without using `--no-verify`;
- commiter does not perform force push, reset, stash, amend, or automatic rollback;
- invalid or unsafe LLM output cannot cause Git mutation.

Never add real credentials, private keys, tokens, `.env` contents, or other secrets to fixtures or examples.

## Tests

New behavior should include tests at the narrowest useful package boundary. Add regression tests for bug fixes.

Before opening a pull request, run:

```sh
go test ./...
go vet ./...
go build ./cmd/commiter
```

When a change affects CGo or Tree-sitter integration, also verify the supported Apple Silicon build path.

When a change affects Git mutation, verification, hooks, sensitive-file handling, or LLM-plan validation, include failure-path tests that confirm Git state is left in the state required by the SRS.

## Documentation

Use repository-relative links for repository documents.

When you knowingly change requirements, commands, configuration, safety behavior, or contributor workflow, update the relevant documentation when practical and note any uncertainty in the pull request. Maintainers are responsible for the final determination of documentation completeness and English/Japanese SRS semantic alignment before merge. Code blocks, configuration keys, requirement IDs, command names, file names, and numeric thresholds should remain identical across translations unless the underlying specification changes.

## Pull Requests

Pull requests use one repository-wide contract. The pull request template and the `PR Policy / pr-policy` check enforce the machine-verifiable contributor-facing parts of that contract once a pull request is ready for review.

### Title

Pull request titles must:

- be written in English using ASCII characters;
- follow Conventional Commits in the form `<type>(<optional-scope>): <summary>`;
- use one of: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `build`, `ci`, or `chore`;
- use `!` before the colon when the pull request introduces a breaking change, when applicable.

Examples:

```text
feat(cli): add JSON output
fix(git): preserve staged selection
ci: validate pull request metadata
```

### Required body sections

Keep all required template sections and complete them before marking the pull request ready for review:

Free-form explanations in the body may be written in English or Japanese. Keep the section headings, issue-closing keywords, verification check labels, and requirements-impact checkbox labels exactly as shown because the policy check validates those machine-readable parts.

- `Summary`: explain the problem and the chosen approach.
- `Related issue`: normally use `Closes #123`, `Fixes #123`, or `Resolves #123`. In the limited cases where an issue is not appropriate, write `N/A: <reason>`.
- `Changes`: list the concrete changes in the pull request.
- `Verification`: mark the standard checks that were run. Explain every unchecked standard check under `Skipped / not applicable`.
- `Requirements impact`: select exactly one declaration based on what you know: no expected requirements impact, possible requirements impact, or maintainer review required. Contributors do not need to identify exact FR / SR / NFR / AC IDs or certify English/Japanese SRS alignment.
- `Safety impact`: describe effects on Git-state handling, local-only LLM processing, sensitive-file handling, verification, hooks, or output safety. Write `None.` when there is no safety impact.

The policy check validates structure and explicit confirmations. It does not judge whether the technical explanation is sufficient; reviewers make that determination.

Before merge, maintainers must make the final determination of SRS impact, affected requirement IDs when applicable, English/Japanese SRS semantic alignment, and release/breaking-change impact. Merging the pull request signifies that the maintainer considers those checks acceptable for the change.

Draft pull requests may be incomplete. The PR policy is enforced when the pull request becomes ready for review.

Pull requests created by explicitly allowlisted automation, currently `dependabot[bot]` and `github-actions[bot]`, are exempt from the human-oriented body checks. Their titles must still satisfy the title policy. Dependabot is configured to generate compatible Conventional Commit prefixes.

### Scope, labels, and merge method

Keep each pull request limited to one coherent purpose. In principle, one pull request should correspond to one primary issue.

Pull request labels are not required. There is no hard changed-line or file-count limit; scope quality is evaluated by responsibility and purpose rather than raw size.

The repository uses squash merge for pull requests. The validated pull request title is therefore the basis of the commit title added to `main`.

Review feedback should be addressed with the same focus: fix the identified issue without expanding the scope unless the broader change is required for correctness or safety.

## Commit Messages

Use Conventional Commits where practical, for example:

```text
feat(cli): add command
fix(git): preserve staged state
docs: add English SRS
test(safety): cover sensitive-file exclusion
```

Keep each commit internally coherent and avoid mixing unrelated changes.

## License and Conduct

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE) that applies to this repository.

All participants must follow the [Code of Conduct](CODE_OF_CONDUCT.md). Security vulnerabilities must be reported according to the [Security Policy](SECURITY.md), not through a public issue or discussion.
