# Operational Proof Entrypoints And Ownership

## Purpose

This document maps the current operational-proof entrypoints to their role,
ownership, and allowed usage.

Use it when deciding where to run a proof, where to document a workflow, or
where to land changes without creating another parallel route.

## Canonical Entrypoint Matrix

| Surface | Class | Official use | Primary owner | Notes |
|---|---|---|---|---|
| `make smoke` | canonical | baseline single-symbol operational proof | operational tooling owner | first proof to reach for after `make up` + `make seed` |
| `make smoke-multi` | canonical | broad multi-symbol operational proof | operational tooling owner | default broader proof for governed multi-symbol behavior |
| `make smoke-analytical` | canonical | analytical writer/reader proof | operational tooling owner | proves ClickHouse-backed analytical path |
| `make smoke-operational` | canonical | process-isolation and halt/resume operational proof | operational tooling owner | proves container/process-level operational behavior |
| `make smoke-restart-recovery` | canonical | restart/recovery proof | operational tooling owner | proves restart resilience and durable recovery |
| `make smoke-help` | supporting | proof selection and operator guidance | operational tooling owner | discoverability surface only; does not replace proof-of-record targets |
| `make up` | canonical | runtime bring-up | runtime/deploy owner | prepares substrate for proofs |
| `make seed`, `make seed-multi` | canonical | activation/setup before smoke | operational tooling owner | setup entrypoints, not proof entrypoints |
| `make diag` | canonical | quick runtime snapshot | operational tooling owner | diagnostic support, not proof-of-record |

## Ergonomic Wrappers

| Surface | Why it exists | Owner | Rule |
|---|---|---|---|
| `make live` | one-command single-symbol bring-up plus validation | operational tooling owner | wrapper only; do not cite as proof-of-record when a smoke target is the actual evidence surface |
| `make live-check` | validate a running single-symbol stack | operational tooling owner | wrapper only |
| `make live-multi` | one-command multi-symbol bring-up plus validation | operational tooling owner | wrapper only |
| `make live-multi-check` | validate a running multi-symbol stack | operational tooling owner | wrapper only |
| `stack-up`, `stack-down`, `stack-restart`, `stack-logs` | help-surface discoverability | repository/tooling owner | alias only; do not replace canonical lifecycle names in docs |

## Tolerated Legacy And Debugging Routes

| Surface | Why it still exists | Allowed use |
|---|---|---|
| direct `scripts/smoke-*.sh` | harness debugging, wait overrides, implementation work | use only when the Make wrapper intentionally does not expose what you need |
| direct `scripts/live-pipeline-activate.sh` | wrapper debugging and narrow flag overrides | use only when debugging wrapper behavior |
| direct `scripts/seed-configctl.sh`, `scripts/diag-check.sh` | local customization and local-host debugging | acceptable expert use; still document Make first |
| `raccoon-cli legacy runtime-smoke` | compatibility for historical CLI consumers | tolerated only as a legacy helper |
| flat `raccoon-cli runtime-smoke` alias | hidden compatibility alias | do not document as first choice |
| `quality-gate --profile deep` runtime helper | compatibility inside deep tooling profile | not a substitute for `make smoke*` |
| raw `docker compose`, `go`, `cargo` | substrate debugging and implementation work | below the repository workflow contract |

## Routes To Discontinue

The following routes are tolerated only to avoid breakage. They should not
receive new first-choice documentation or expanded responsibilities:

- describing direct `scripts/*.sh` as the main way to run operational proof;
- presenting `make check-deep` as equivalent to runtime proof;
- presenting `raccoon-cli runtime-smoke` or `legacy runtime-smoke` as canonical operational validation;
- adding new runtime proof entrypoints outside `make smoke*` without a strong architectural exception.

## Change Ownership Rules

### Makefile

Owns:

- public target naming;
- target discoverability;
- canonical vs wrapper classification.

Does not own:

- detailed shell harness behavior;
- runtime substrate semantics.

### `scripts/*.sh`

Own:

- proof execution details;
- debug-only flags;
- low-level waits and environment-specific knobs.

Do not own:

- public workflow taxonomy;
- canonicality policy.

### `tools/raccoon-cli/`

Owns:

- analysis, guard rails, and compatibility helper behavior.

Does not own:

- the canonical runtime proof contract;
- live-stack orchestration as a primary operator surface.

### `docs/operations/`

Owns:

- the official governance model for smoke and operational harnesses;
- operator-facing entrypoint guidance;
- canonical usage language.

## Selection Guide

| If you need to... | Start here |
|---|---|
| prove the baseline runtime flow | `make smoke` |
| prove governed multi-symbol runtime behavior | `make smoke-multi` |
| prove analytical write/read behavior | `make smoke-analytical` |
| prove halt/resume and process isolation behavior | `make smoke-operational` |
| prove restart/recovery resilience | `make smoke-restart-recovery` |
| bring up a stack before proving behavior | `make up` + `make seed*` or `make live*` |
| debug a harness implementation | direct `scripts/*.sh` |
| inspect tooling and repository structure | direct `raccoon-cli` |
