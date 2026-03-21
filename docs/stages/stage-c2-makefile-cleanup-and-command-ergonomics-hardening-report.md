# Stage C2: Makefile Cleanup And Command Ergonomics Hardening Report

## 1. Executive Summary

C2 hardened the root `Makefile` as an operational support surface without changing the active domain tranche.

The work focused on:

- better command discoverability
- clearer target grouping
- safer service scoping
- promotion of real but previously hidden support flows
- proportional documentation of the command contract

The result is a more predictable DX surface with compatibility preserved for the targets already used across active docs and stage history.

## 2. Problems Found In The Makefile

### 2.1 Help was informative but manually fragile

The existing `help` target was a large hand-maintained block. It exposed commands, but drift risk was high and operational grouping was only partial.

### 2.2 `SERVICE=...` support was under-hardened

`build` and `docker-build` did not actually support the same service set. In practice, `docker-build` could receive a service value that Docker Compose could never satisfy.

### 2.3 Real support entrypoints were hidden

Two meaningful support flows were not first-class Make targets:

- restart/recovery smoke
- codegen equivalence validation

### 2.4 Common operator expectations were missing

The repository had a fast guard rail and current docs, but lacked common entrypoint names such as:

- `make lint`
- `make docs`

### 2.5 Internal shell logic repeated itself

The module-aware Go test targets repeated the same iteration logic with only small flag differences.

## 3. Changes Performed

### 3.1 Reorganized the Makefile by category

The file is now grouped into:

- Help
- Core Workflow
- Go And Test
- Runtime Stack
- Architecture And Analysis
- Raccoon CLI
- Codegen
- Migrations

### 3.2 Replaced the manual help block

`make help` is now generated from target annotations and prints:

- grouped targets
- concise descriptions
- common variable usage

### 3.3 Added discoverability aliases without removing canonical targets

Added:

- `lint`
- `test-unit`
- `stack-up`
- `stack-down`
- `stack-restart`
- `stack-logs`

These are additive only.

### 3.4 Promoted hidden support flows

Added:

- `make smoke-restart-recovery`
- `make codegen-equivalence`

### 3.5 Hardened target-specific service validation

Split service validation into:

- buildable binaries
- compose-backed build services
- compose runtime services

This removed a concrete fragility around unsupported `SERVICE` combinations.

### 3.6 Consolidated duplicated helpers

Introduced shared helpers for:

- module-aware Go test execution
- `SERVICE` validation
- local env loading for migration-oriented targets

### 3.7 Updated active docs

Updated:

- `README.md`
- `DEVELOPMENT.md`

Also fixed the active-path drift in `DEVELOPMENT.md`:

- `cmd/migrate/engine/` replaces the stale `cmd/migrate/migrate/`
- `internal/shared/webserver/` replaces the stale implication that `webserver` still lives under `internal/interfaces/`

## 4. Final Targets And Conventions

The command surface now follows four explicit rules:

1. Existing documented targets remain canonical.
2. New aliases exist only to improve discoverability.
3. Families remain prefix-based (`smoke-*`, `live-*`, `codegen-*`, `migrate-*`).
4. `SERVICE=...` is validated against the real support set of each target.

Reference docs added in C2:

- `docs/operations/makefile-command-ergonomics-and-hardening.md`
- `docs/operations/makefile-targets-reference-and-conventions.md`

## 5. Expected Impacts

### Positive

- Faster onboarding into the repository command surface
- Lower chance of invalid `SERVICE=...` usage
- Less hidden operational knowledge around restart-recovery and codegen equivalence
- Lower maintenance cost for `help`
- Better alignment between command surface and active docs

### Tradeoffs

- The `Makefile` is more explicitly structured and therefore slightly longer
- Help quality now depends on keeping target annotations accurate

These tradeoffs are acceptable because the structure is still simple and the gain in clarity is immediate.

## 6. Validation

Validation performed for C2:

- `make help`
- `make docs`
- `make lint`
- `make stack-up SERVICE=not-a-service` expectation review via target validation logic
- documentation reconciliation against `README.md` and `DEVELOPMENT.md`

## 7. Optional Follow-Ups

- Decide whether CI should adopt `make quality-gate-ci` as a first-class enforced step.
- Consider exposing any remaining expert-only support wrappers only after their scripts stabilize.
- If future target growth continues, consider a short generated index under `docs/operations/` that mirrors `make help`.
