# Development CLI Reliability And Command Testing Strategy

## Purpose

This document defines how `tools/raccoon-cli` should be validated as a
development tool for `market-foundry`.

The goal is not exhaustive QA coverage. The goal is operational confidence for
the commands that contributors actually depend on during the repository
workflow.

## Reliability Posture

`raccoon-cli` is trustworthy only when its commands are:

- predictable about what they inspect;
- explicit about where their inputs came from;
- stable in exit-code semantics;
- easy to validate in both human and JSON modes;
- safe to run on partial, broken, or non-runtime-ready worktrees.

## High-Value Command Surface

The most valuable development commands are:

| Command family | Why it matters | Reliability priority |
|---|---|---|
| `check repo` | First repository sanity pass | High |
| `check gate` / `quality-gate` | Main guard-rail aggregator | High |
| `change tdd` / `tdd` | Daily validation planning | High |
| `change recommend` / `recommend` | Daily validation prioritization | High |
| `change briefing` / `briefing` | Human/agent handoff context | High |
| `change impact` / `impact-map` | Structural scope discovery | High |
| `snapshot`, `snapshot-diff`, `baseline-drift` | Drift and contract comparison | Medium |
| `inspect symbol`, `inspect contract-usage`, `change rename` | Focused engineering analysis | Medium |
| `legacy runtime-smoke` | Compatibility only | Lower than `make smoke*` |

The first hardening target is not feature breadth. It is the reliability of the
commands above when developers use them on normal local worktrees.

## Testing Layers

### 1. Analyzer unit tests

Analyzer unit tests prove deterministic structural logic:

- report contents;
- ordering and deduplication;
- severity and recommendation rules;
- JSON serializability;
- graceful behavior on empty or partial repositories.

This is the broadest layer and should stay fast.

### 2. CLI integration tests

Integration tests prove command contracts end-to-end:

- help and taxonomy;
- exit codes;
- JSON shape;
- human output markers;
- `--project-root`, `--json`, `-v`, and argument interactions;
- git-backed auto-detection behavior for change-oriented commands.

This layer should own behavior that users actually observe from the binary.

### 3. Validation matrix

`tests/validation_matrix.rs` should keep a curated set of trustworthiness
assertions for the repository-critical commands:

- exit code semantics `0/1/2`;
- stdout vs stderr hygiene;
- deterministic step ordering;
- actionable error/help text;
- clean handling of absent runtime dependencies.

This file is the contract suite, not a second place to duplicate every feature
test.

## Command Input Resolution Strategy

Commands that infer scope from `git status` must expose where that scope came
from.

For `tdd`, `recommend`, `briefing`, and `impact-map`, the command must make the
input source observable:

- `explicit`
- `git_status_structural`
- `git_status_all`
- `git_status_clean`
- `not_git_repository`
- `git_unavailable`

When no targets are found, the command must explain why:

- clean worktree;
- project root is not a git repository;
- git execution failed;
- explicit input was omitted.

This keeps “no findings” distinct from “no trustworthy input”.

## Output Contract Rules

### Human output

Human output should include:

- stable section headers;
- a visible input-source or reason line for change-oriented commands;
- actionable next steps when nothing can be analyzed;
- no hidden dependence on stderr for normal command failures.

### JSON output

JSON output should include machine-checkable source metadata:

- `input_source` for `tdd`, `briefing`, and `impact-map`;
- `input.detection_mode` for `recommend`.

JSON should remain valid on failing checks. Check failure is an exit-code
decision, not a serialization failure.

## Error And Exit-Code Policy

- `0`: command succeeded and the evaluated condition passed or produced a usable report
- `1`: command succeeded but the evaluated condition failed
- `2`: runtime error, serialization error, or argument parsing failure

Command implementations should prefer “successful report with failed verdict”
over runtime exceptions when the repository state is inspectable but unhealthy.

## Required Regression Cases

Every future change to change-oriented commands should preserve tests for:

- explicit target input;
- auto-detection from a real git worktree;
- docs-only changes being filtered from structural commands;
- clean worktree behavior;
- non-git project root behavior;
- JSON source metadata;
- human-readable reason lines for empty auto-detection.

## Maintenance Guidance

- Add unit tests for structural logic before adding integration coverage.
- Add integration tests only for observable contracts.
- Prefer extending existing report fields over inventing parallel status channels.
- Keep compatibility aliases tested only at the dispatch/CLI layer.
- Do not let compatibility helpers redefine the repository workflow contract.

## Validation Entry Points

Use this sequence when changing the CLI:

1. `cargo test --manifest-path tools/raccoon-cli/Cargo.toml --test cli_integration`
2. `cargo test --manifest-path tools/raccoon-cli/Cargo.toml --test validation_matrix`
3. `cargo test --manifest-path tools/raccoon-cli/Cargo.toml change_targets`
4. Escalate to broader `cargo test --manifest-path tools/raccoon-cli/Cargo.toml` when the change touches analyzer internals broadly

The repository-level workflow remains:

1. `make check`
2. `make tdd`
3. implement the smallest correct change
4. `make verify`
