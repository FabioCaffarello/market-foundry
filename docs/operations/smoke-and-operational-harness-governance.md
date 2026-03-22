# Smoke And Operational Harness Governance

## Purpose

This document defines the governance model for smoke tests and operational
harnesses in `market-foundry`.

Its job is to make the operational proof surface explicit:

- which entrypoints are canonical;
- which ones are wrappers only;
- which ones remain tolerated for compatibility;
- which routes should stop being expanded or documented as first choice.

It does not redefine domain behavior, service contracts, or runtime logic.

## Governance Problem

The repository already contains enough operational proof surface to create
entropy if left implicit:

- `make smoke*` targets;
- `make live*` bring-up flows;
- direct shell harnesses under `scripts/`;
- compose-backed runtime substrate commands;
- `raccoon-cli` legacy runtime helpers.

Without explicit governance, those surfaces can drift into parallel front doors.

## Canonical Model

### 1. `make smoke*` owns operational proof

The canonical proof-of-record surface for runtime behavior is the `make smoke*`
family:

- `make smoke`
- `make smoke-multi`
- `make smoke-analytical`
- `make smoke-round-trip`
- `make smoke-live-stack`
- `make smoke-activation`
- `make smoke-composed`
- `make smoke-operational`
- `make smoke-restart-recovery`

Each target proves a distinct runtime property and is the authoritative
repository entrypoint for that proof.

### 1b. `make ci-*` owns CI integration and preflight

The CI integration surface is:

- `make ci-smoke` — stackless smoke suite safe to run in any CI environment
- `make ci-preflight` — local pre-push gate: tests + consistency + quality + stackless smoke
- `make ci-analytical` — full analytical gate: unit tests + compose-backed smoke-analytical
- `make ci-wait-ready` — infrastructure readiness polling before stack-dependent smokes

These targets compose canonical surfaces; they do not define new proof semantics.

### 2. `make live*` is orchestration, not proof-of-record

`make live`, `make live-check`, `make live-multi`, and `make live-multi-check`
remain useful, but their role is ergonomic orchestration:

- bring up the stack;
- seed the stack when needed;
- run validation as a convenience flow.

They are not the canonical proof-of-record surface. Documentation should point
to the relevant `make smoke*` target when the question is "how do we prove this
runtime behavior?"

### 3. `stack-*` stays alias-only

`stack-up`, `stack-down`, `stack-restart`, and `stack-logs` exist only to make
the lifecycle surface easier to scan in `make help`.

They must not replace the canonical `up`, `down`, `restart`, or `logs`
terminology in documentation or governance artifacts.

### 4. Shell harnesses stay behind `make`

Direct script invocation is tolerated for:

- harness debugging;
- extra wait flags or environment overrides;
- work on the harness implementation itself.

Direct script usage is not the public API when an equivalent Make target exists.

### 5. `raccoon-cli` does not own runtime proof

Direct `raccoon-cli` usage remains canonical for expert inspection and tooling
governance, but not for operational proof.

The deep gate profile and `legacy runtime-smoke` helper remain compatibility
surfaces only. They must not gain new authority over runtime proof that already
belongs to `make smoke*`.

## Entry Classification Rules

| Class | Definition | Current surfaces |
|---|---|---|
| canonical surface | proof-of-record or public repository entrypoint | `make smoke*`, `make ci-*`, `make up`, `make seed*`, `make diag` |
| ergonomic wrapper | convenience layer that composes canonical surfaces | `make live*`, `stack-*` |
| tolerated legacy | retained for compatibility or narrow expert use | direct `scripts/*.sh`, `raccoon-cli legacy runtime-smoke`, flat `runtime-smoke` alias, `quality-gate --profile deep` runtime helper |
| route to discontinue | surface that should not receive new first-choice documentation or expanded ownership | direct script usage as primary docs, `make check-deep` as runtime proof, direct CLI runtime helper as operational proof, new parallel runtime front doors |

## Canonical Entry Design Rules

When adding or modifying operational proofs:

1. A new operational proof must have exactly one canonical Make target.
2. The Make target name should stay in the `smoke-*` family if it proves runtime behavior.
3. The shell script remains the implementation surface behind that target.
4. `live*` may call into canonical proof flows, but should not become the only documented way to prove behavior.
5. `check-deep` may keep compatibility runtime checks, but it must not be documented as the proof-of-record runtime flow.
6. `raccoon-cli` may inspect or report on harnesses, but should not become a second runtime orchestration plane.

## Ownership Model

| Concern | Owning surface | Responsibility |
|---|---|---|
| public operational proof taxonomy | `Makefile`, `docs/operations/` | define canonical entrypoints and usage rules |
| harness implementation | `scripts/*.sh` | execute the proof, expose debug-only flags, stay behind Make wrappers |
| runtime substrate | `deploy/compose/`, `deploy/configs/`, `deploy/envs/` | provide the runtime environment used by proofs |
| tooling compatibility helpers | `tools/raccoon-cli/`, `docs/tooling/` | keep legacy CLI runtime helpers contained and correctly labeled |
| governance evidence | `docs/stages/` | record the decisions and resulting support architecture |

## Documentation Rules

- `README.md` and `DEVELOPMENT.md` should describe `make smoke*` as the canonical operational-proof surface.
- `docs/operations/` owns the user-facing governance of smoke and harness entrypoints.
- `docs/tooling/` may mention runtime helpers only as tooling compatibility behavior, not as first-choice operator guidance.
- Script help text should point back to the canonical Make target when one exists.

## Non-Goals

- introducing a new harness platform;
- rewriting existing smoke logic for stylistic reasons;
- pushing domain changes into services to make smoke easier;
- preserving ambiguous entrypoints for convenience once governance is explicit.
