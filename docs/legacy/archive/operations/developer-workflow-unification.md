# Developer Workflow Unification

## Purpose

This document defines the official developer workflow for `market-foundry`.

It exists to remove ambiguity across setup, local execution, validation, smoke
selection, and first-line troubleshooting without changing business behavior or
functional architecture.

## Workflow Principles

- `make` is the canonical public workflow surface.
- `make bootstrap` is the canonical setup validation path.
- `make live` is the fastest official bring-up path.
- `make up` + `make seed*` is the controlled manual bring-up path.
- `make smoke*` is the proof-of-record runtime validation surface.
- `make diag`, `make ps`, and `make logs` are the first-line troubleshooting surface.
- Direct `scripts/*.sh`, direct `raccoon-cli`, and raw `docker compose` / `go` / `cargo` are expert or debugging routes, not the default developer journey.
- `raccoon-cli` is the strategic intelligence layer for inspection, impact, TDD guidance, drift, and architecture safety; `make` remains the workflow contract.

## Official Command Map

| Need | Official command | Notes |
|---|---|---|
| Validate a machine or changed local environment | `make bootstrap` | Run first on a new machine, after toolchain changes, or when setup is suspect |
| Discover the supported surface | `make help` | Grouped command catalog and common variables |
| Fastest single-symbol bring-up | `make live` | Builds, starts, seeds, and validates the stack |
| Fastest multi-symbol bring-up | `make live-multi` | Same orchestration for the governed multi-symbol path |
| Controlled manual bring-up | `make up` then `make seed` / `make seed-multi` | Use when you need finer control or debugging between steps |
| Baseline runtime proof | `make smoke` | Default smoke for ordinary runtime changes |
| Multi-symbol runtime proof | `make smoke-multi` | Use when the change affects symbol breadth or symbol isolation |
| Analytical runtime proof | `make smoke-analytical` | Use for writer/ClickHouse/read-path changes |
| Persistence round-trip proof | `make smoke-round-trip` | Use when the change touches adapter → NATS → ClickHouse → HTTP continuity |
| Live-stack verification proof | `make smoke-live-stack` | Use for the specialized live-stack and gateway verification path |
| Activation control-surface proof | `make smoke-activation` | Use when the change touches activation transitions or the control surface |
| Composed pipeline proof | `make smoke-composed` | Use when the change is bounded to the composed execution pipeline without the full stack |
| Process/runtime operational proof | `make smoke-operational` | Use for lifecycle, halt/resume, or process-isolation changes |
| Restart/recovery proof | `make smoke-restart-recovery` | Use for restart durability and recovery behavior |
| Pre-change guard rail | `make check` | Runs repo consistency + fast quality gate |
| Validation planning | `make tdd` | Shows impact-driven validation guidance |
| Post-change validation | `make verify` | Runs tests + repo consistency + fast quality gate |
| Significant-change gate | `make check-deep` | Deep tooling validation, not a replacement for `make smoke*` |
| First troubleshooting step | `make diag` | Quick runtime health and readiness snapshot |
| Service status | `make ps` | Compose status surface |
| Service logs | `make logs SERVICE=gateway` | Use service scoping before raw compose logs |
| Restart one runtime service | `SERVICE=gateway make restart` | Fast controlled recovery path |

## Official Workflow

### 1. Bootstrap

Run:

```bash
make bootstrap
```

This validates:

- required host tools;
- Docker daemon and compose availability;
- compose renderability;
- canonical repository entrypoints;
- required local env artifacts.

### 2. Bring Up The Local Runtime

Default path:

```bash
make live
```

Controlled manual path:

```bash
make up
make seed
make smoke
```

Use the controlled manual path when you need to inspect bring-up one step at a
time, switch between `seed` and `seed-multi`, or stop before the smoke step.

### 3. Change Loop

Run:

```bash
make check
make tdd
# implement the smallest correct change
make verify
```

Escalate to `make check-deep` for larger or riskier changes. Do not treat
`make check-deep` as equivalent to runtime proof.

### 4. Choose The Narrowest Relevant Smoke

Use this selection order:

1. `make smoke` for baseline single-symbol runtime behavior.
2. `make smoke-multi` when the change affects governed multi-symbol behavior.
3. `make smoke-analytical` when the change touches writer, ClickHouse, analytical readers, or analytical HTTP surfaces.
4. `make smoke-round-trip` when the change affects adapter → NATS → ClickHouse → HTTP continuity.
5. `make smoke-live-stack` when the change affects the specialized live-stack verification path.
6. `make smoke-activation` when the change affects activation transitions or control-surface behavior.
7. `make smoke-composed` when the change is bounded to the composed execution pipeline without the full stack.
8. `make smoke-operational` when the change affects process lifecycle or operational isolation.
9. `make smoke-restart-recovery` when the change affects restart durability or recovery behavior.

If multiple behaviors changed, run the narrowest set of smokes that directly
prove those behaviors instead of defaulting immediately to the heaviest harness.

For the full proof inventory and ownership model, use
[`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md)
and
[`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md).

### 5. Troubleshoot In The Official Order

Use this sequence first:

```bash
make diag
make ps
make logs SERVICE=gateway
SERVICE=gateway make restart
make down
```

Only escalate after that to:

- direct `scripts/*.sh` for debug-only flags or harness work;
- direct `raccoon-cli` for expert structural analysis;
- raw `docker compose`, `go`, or `cargo` when debugging below the repository workflow contract.

## Canonical Versus Deprioritized Paths

### Canonical

- `make bootstrap`
- `make help`
- `make live*`
- `make up`, `make seed*`
- `make smoke*`
- `make check`, `make tdd`, `make verify`, `make check-deep`
- `make diag`, `make ps`, `make logs`, `make restart`

### Deprioritized But Allowed

- direct `scripts/*.sh`
- direct `raccoon-cli`
- raw `docker compose`, `go`, `cargo`

These remain valid for expert work and debugging, but they should not become
the first documented route when a stable Make target already exists.

## Related Documents

- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`developer-onboarding-and-troubleshooting-guide.md`](developer-onboarding-and-troubleshooting-guide.md)
- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md)
- [`smoke-and-operational-harness-governance.md`](smoke-and-operational-harness-governance.md)
- [`operational-proof-entrypoints-and-ownership.md`](operational-proof-entrypoints-and-ownership.md)
