# Smoke UX And Proof Execution Ergonomics

## Purpose

This document defines the operator-facing UX rules for smoke and proof execution
in `market-foundry`.

The goal is practical usability:

- clear proof selection
- explicit prerequisites
- predictable invocation
- useful failure messages
- stable daily execution habits

It does not introduce a new execution platform. The canonical proof surface
remains `make smoke*`.

## Current Public Surface

| Need | Canonical command | Setup before running |
|---|---|---|
| Baseline single-symbol runtime proof | `make smoke` | `make up && make seed` |
| Multi-symbol runtime proof | `make smoke-multi` | `make up && make seed-multi` |
| Analytical write/read proof | `make smoke-analytical` | `make up && make seed` or `make seed-multi` |
| Persistence round-trip proof | `make smoke-round-trip` | `make up && make seed` |
| Live stack and gateway verification proof | `make smoke-live-stack` | `make up && make seed` |
| Activation control-surface proof | `make smoke-activation` | `make up && make seed` |
| Composed pipeline proof without the full stack | `make smoke-composed` | none |
| OS-process and halt/resume proof | `make smoke-operational` | `make up && make seed` |
| Restart/recovery proof | `make smoke-restart-recovery` | `make up && make seed` |
| Proof selection help | `make smoke-help` | none |

## Ergonomic Rules

### 1. Start with proof selection, not script spelunking

Use `make smoke-help` first when the right proof is not obvious.

The operator should not need to inspect shell comments or stage history to
answer:

- which proof is the narrowest valid one
- what setup is required
- what to inspect first when it fails

### 2. Keep prerequisites explicit at the point of use

Each smoke surface should state:

- canonical Make target
- required setup command
- runtime context such as `BASE_URL`
- wait budget in use

This reduces the common failure mode where the harness technically works but the
operator ran it against an unseeded or half-ready stack.

### 3. Support common overrides from `make`

Direct script invocation remains valid for debugging, but common operational
overrides should also work through the public surface:

```bash
SMOKE_WAIT=180 make smoke
SMOKE_WAIT=240 make smoke-analytical
BASE_URL=http://127.0.0.1:18080 make smoke-operational
```

This keeps the public workflow stable while preserving expert escape hatches.

### 4. Fail with concrete next steps

Improved failure messages must not hide the real error. They should add:

- the failing endpoint or subsystem
- the expected state
- the most likely setup gap
- the next diagnosis commands

The current repository standard is:

```bash
make ps
make logs SERVICE=gateway
make diag
```

Then escalate to service-specific logs when needed.

### 5. Print compact summaries that help repeat execution

At the end of a proof, the operator should be able to see:

- which canonical target was used
- which setup was assumed
- which gateway/wait values were used
- what was proven
- what to run next if the result was not what they expected

## UX Changes Applied In C14

### Public surface

- added `make smoke-help` as a discoverability surface
- documented `BASE_URL` and `SMOKE_WAIT` as public smoke overrides
- kept `make smoke*` as proof-of-record

### Harness behavior

- standardized smoke banners with canonical target, setup hint, base URL, and wait budget
- improved early abort messages for gateway/readiness/container preflight failures
- added consistent diagnosis hints on hard failures
- improved end-of-run summaries for baseline and multi-symbol smokes
- preserved existing proof semantics

### Documentation

- linked smoke UX guidance from top-level workflow docs
- added a dedicated failure-diagnosis flow document
- documented public overrides so operators do not need direct script usage for routine waits

## Non-Goals

- no new orchestration layer
- no replacement of `make smoke*` with another tool
- no semantic change to what each proof validates
- no masking of failures behind generic “friendly” output

## Daily Operating Pattern

Use this as the default loop:

```bash
make smoke-help
make up
make seed          # or make seed-multi
make smoke         # or the narrowest matching smoke
```

When a proof is slow because the pipeline is still warming up:

```bash
SMOKE_WAIT=180 make smoke
```

When a proof fails:

```bash
make ps
make logs SERVICE=gateway
make diag
```

Then inspect the service closest to the failing surface:

- `make logs SERVICE=derive` for runtime derivation path failures
- `make logs SERVICE=store` for KV projection/read issues
- `make logs SERVICE=writer` for analytical persistence/read issues
- `make logs SERVICE=execute` for control/execution/recovery issues

## Operator Guidance

Choose the narrowest valid proof first.

Run the broader proofs when:

- the change crosses multiple runtime surfaces
- the narrow proof passes but the broader user flow still looks suspicious
- you are validating operational confidence before handoff
