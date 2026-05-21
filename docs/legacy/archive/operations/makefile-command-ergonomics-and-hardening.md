# Makefile Command Ergonomics And Hardening

## Purpose

This document records the C2 hardening pass on the root `Makefile`.

The goal of this pass was narrow and operational:

- improve command discoverability
- reduce avoidable fragility
- make the supported workflows more predictable
- preserve the existing runtime and domain tranche behavior

This is not a workflow redesign. The repository keeps its existing operational model:

- `make check` before coding
- `make tdd` to scope validation
- smallest correct change
- `make verify` after coding
- `make check-deep` only when the change justifies it

## Problems Addressed

### 1. Help existed, but it was a hard-coded wall of text

The old `help` target exposed many commands, but it had three practical issues:

- categories were partly implicit
- aliases and optional variables were not modeled consistently
- updates required editing a large block manually, which is drift-prone

### 2. Service scoping was inconsistent

`build` and `docker-build` both supported `SERVICE=...`, but they did not operate over the same service universe.

The most concrete fragility was:

- `make docker-build SERVICE=migrate` looked acceptable from the variable contract
- the compose file has no `migrate` service
- the target therefore failed in a predictable but avoidable way

### 3. Real support flows were still hidden behind raw scripts

Two operationally relevant checks existed as scripts but were not promoted through `make`:

- `scripts/smoke-restart-recovery.sh`
- `scripts/codegen-equivalence-check.sh`

That made the supported surface less discoverable than the real surface.

### 4. Common expectations were not met by familiar target names

Many contributors expect entrypoints such as:

- `make lint`
- `make docs`

The repository already had equivalent behavior, but not equivalent names.

### 5. Internal shell logic was more duplicated than necessary

The module-aware Go test targets repeated the same module iteration pattern with small variations.

That increased maintenance cost and made behavior drift more likely.

## Changes Applied

### Help and organization

- Replaced the hand-written `help` wall with grouped self-documenting help.
- Grouped targets by operational category:
  - Help
  - Core Workflow
  - Go And Test
  - Runtime Stack
  - Architecture And Analysis
  - Raccoon CLI
  - Codegen
  - Migrations
- Added a compact common-variable section to `help`.

### Predictability and compatibility

- Kept canonical existing targets intact: `check`, `verify`, `up`, `down`, `smoke`, `live`, `quality-gate`, `codegen-*`, `migrate-*`.
- Added low-risk discoverability aliases:
  - `lint` -> `check`
  - `test-unit` -> `test`
  - `stack-up` -> `up`
  - `stack-down` -> `down`
  - `stack-restart` -> `restart`
  - `stack-logs` -> `logs`
- Added `docs` to print the primary documentation entrypoints for workflows and tooling.

### Hardening

- Split service validation by responsibility:
  - `BUILDABLE_SERVICES` for local binary builds
  - `COMPOSE_BUILD_SERVICES` for image builds
  - `COMPOSE_RUNTIME_SERVICES` for logs/restart scoping
- Added validation for `SERVICE=...` on `restart` and `logs`, not only on build targets.
- Promoted hidden wrappers:
  - `make smoke-restart-recovery`
  - `make codegen-equivalence`
- Consolidated repeated Go test iteration logic behind one shared `RUN_GO_TEST` helper.
- Centralized local env loading behind one shared helper for migration-related targets.

## What Did Not Change

This pass deliberately did not:

- change runtime topology
- add new infrastructure
- rename canonical operational targets
- remove `quality-gate*`, `check*`, `live*`, `smoke*`, `codegen-*`, or `migrate-*`
- alter domain behavior

## Net Effect

The `Makefile` now behaves more like an explicit command surface and less like a loose script index:

- supported commands are easier to discover
- scoping failures fail earlier and with clearer messages
- high-value hidden flows are first-class
- documentation and commands now align more closely
