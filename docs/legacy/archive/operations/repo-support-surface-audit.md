# Repo Support Surface Audit

## Purpose

This document audits the repository support surfaces for `market-foundry` without disturbing the active domain tranche. It focuses on:

- `Makefile` workflow entrypoints
- support scripts under `scripts/`
- `tools/raccoon-cli/`
- operational and tooling documentation
- lightweight guard rails and repository hygiene

Non-goals for this audit:

- changing bounded contexts or domain flows
- broad refactors
- rewriting architecture history
- feature work disguised as tooling work

## Baseline Examined

Audit baseline collected from the current worktree on 2026-03-21.

Observed commands and artifacts:

- `make help`
- `make check` -> passed: 84 checks, 6 steps passed, 1 runtime step skipped
- `git status --short`
- `./tools/raccoon-cli/target/release/raccoon-cli --help`
- `./tools/raccoon-cli/target/release/raccoon-cli quality-gate --help`
- `./tools/raccoon-cli/target/release/raccoon-cli tdd`
- `./tools/raccoon-cli/target/release/raccoon-cli --json coverage-map`

## Inventory

### Surface Map

| Surface | Current State | Evidence | Implicit Owner |
|---|---:|---|---|
| Root workflow entrypoint | 1 file | `Makefile` | repository tooling owner |
| CI workflow | 1 active workflow | `.github/workflows/ci.yml` | repository tooling owner |
| Primary support scripts | 10 executable scripts | `scripts/*.sh` | operational tooling owner |
| Shared shell utilities | 3 executable helpers | `scripts/utils/*.sh` | operational tooling owner |
| Raccoon CLI docs | 1 local README + 15 tooling docs | `tools/raccoon-cli/README.md`, `docs/tooling/*` | governance/tooling owner |
| Raccoon CLI implementation | 18 commands exposed by help | `raccoon-cli --help` | governance/tooling owner |
| Raccoon analyzers | 24 Rust analyzer files | `tools/raccoon-cli/src/analyzers/*` | governance/tooling owner |
| Operational docs namespace | 0 files | `docs/operations/` absent before this audit | documentation/operations owner |
| Architecture docs | 408 active files | `docs/architecture/` | architecture/stage owner |
| Stage reports | 288 files | `docs/stages/` | stage/reporting owner |
| HTTP manual checks | 9 files | `tests/http/*.http` | operator/developer workflow owner |
| Deploy/runtime configs | compose + 6 jsonc configs + env + migrations + NATS config | `deploy/*` | deploy/runtime owner |
| Go workspace modules | 17 modules | `./scripts/utils/list-modules.sh | wc -l` | repository tooling owner |

### Primary Support Entry Points

`Makefile` currently fronts these support areas:

- quality and guard rails: `check`, `verify`, `check-deep`, `quality-gate*`, `coverage-map`, `tdd`, `arch-guard`, `drift-detect`, `snapshot*`, `baseline-drift`, `recommend`, `briefing`
- developer workflow: `tidy`, `test*`, `build`, `docker-build`
- operational scripts: `live*`, `diag`, `seed*`, `smoke*`
- codegen validation: `codegen-*`
- migration utilities: `migrate-*`

### Hidden or Indirect Entry Points

These support surfaces exist but are not promoted through `make help`:

- `scripts/codegen-equivalence-check.sh`
- `scripts/smoke-restart-recovery.sh`
- direct `raccoon-cli` expert commands:
  `doctor`, `topology-doctor`, `contract-audit`, `runtime-bindings`, `impact-map`, `symbol-trace`, `contract-usage-map`, `rename-safety`, `lsp-enrich`, `runtime-smoke`

## Current Worktree Conflict Zones

These support files already have in-flight changes and should not be used as low-conflict parallel targets right now:

- `.github/workflows/ci.yml`
- `Makefile`
- `scripts/codegen-integrated-check.sh`
- `scripts/smoke-analytical-e2e.sh`
- newly introduced support scripts tied to the active wave:
  `scripts/codegen-equivalence-check.sh`, `scripts/smoke-os-process-operational.sh`, `scripts/smoke-restart-recovery.sh`

The audit documents added by C1 are safe because they live in new locations and do not perturb these active files.

## Findings

### F1. No dedicated operational documentation surface

- Category: discoverability, organization
- Severity: high
- Fix risk: low
- Return: high

Evidence:

- `docs/operations/` did not exist before this audit
- operational runbooks currently live inside `docs/architecture/`, including:
  `current-baseline-runbook.md`
  `current-baseline-operational-diagnostics.md`
  `operational-smoke-ci-and-runbook-closure.md`
- active docs volume is heavily skewed:
  408 architecture docs, 288 stage reports, 15 tooling docs, 0 operations docs

Impact:

- operators and contributors must know the repository history to find current runbooks
- operational guidance is buried inside architecture records and stage evidence
- safe support work is harder to isolate from domain-history documents

### F2. CI guard rail contract is weaker than the documented workflow

- Category: guard rails
- Severity: high
- Fix risk: medium
- Return: high
- Safe to change in parallel now: no, current conflict with `.github/workflows/ci.yml`

Evidence:

- `Makefile` exposes `quality-gate-ci`, `check`, `verify`, `raccoon-test`, `raccoon-build`
- `README.md`, `DEVELOPMENT.md`, and `docs/tooling/cli-overview.md` present `make check` and `quality-gate` as first-class workflow steps
- current `.github/workflows/ci.yml` ends after unit, codegen, behavioral, integration, and smoke jobs
- no active CI step invokes `make check`, `make verify`, `make quality-gate-ci`, or `make raccoon-test`

Impact:

- repo-local guard rails can pass locally but remain unenforced in CI
- docs currently overstate CI coverage for tooling checks
- any drift in `raccoon-cli` or Makefile support workflow can escape until manual execution

### F3. Raccoon CLI contradicts itself on runtime smoke and bootstrap guidance

- Category: robustness, discoverability
- Severity: high
- Fix risk: low
- Return: high

Evidence:

- `tools/raccoon-cli/README.md` says `runtime-smoke` is deprecated and replaced
- `tools/raccoon-cli/src/main.rs` still exposes `runtime-smoke` as a real command
- `tools/raccoon-cli/src/gate/mod.rs` still executes `runtime-smoke` in deep profile
- `tools/raccoon-cli/src/gate/mod.rs` tells users to run nonexistent `make up-dataplane`
- `tools/raccoon-cli/src/analyzers/tdd.rs` also recommends `make up-dataplane`
- `tools/raccoon-cli/src/smoke/stages.rs` still points bootstrap help to `make up-dataplane`
- `coverage-map` emits `raccoon-cli smoke-e2e`, but no such command exists in `raccoon-cli --help`

Impact:

- operator guidance from the CLI cannot be trusted consistently
- “deep” workflow language is internally inconsistent
- remediation output sends users toward commands that do not exist

### F4. Support entrypoints are fragmented between Make, CLI, and unadvertised scripts

- Category: ergonomics, discoverability
- Severity: medium
- Fix risk: low
- Return: high

Evidence:

- `make help` promotes a curated subset of support commands
- `raccoon-cli --help` exposes 18 commands, including expert commands absent from the Makefile
- stage and architecture docs reference `scripts/codegen-equivalence-check.sh`, but no `make` wrapper exists
- `scripts/smoke-restart-recovery.sh` exists and is executable, but no promoted target or workflow entrypoint exposes it

Impact:

- a contributor must inspect multiple surfaces to know the real support toolset
- some high-value support flows look “internal” even when they are operator-facing
- support scripts risk becoming orphaned despite remaining important

### F5. `DEVELOPMENT.md` contains factual drift against the current tree

- Category: organization, discoverability
- Severity: medium
- Fix risk: low
- Return: medium

Evidence:

- `DEVELOPMENT.md` still references `cmd/migrate/migrate/`
- actual tree uses `cmd/migrate/engine/`
- `DEVELOPMENT.md` describes `internal/interfaces/` as containing `webserver`
- actual tree places `webserver` in `internal/shared/webserver`

Impact:

- contributors using the workflow doc will navigate to stale paths
- this weakens trust in the “read this first” support documents

### F6. `raccoon-cli tdd` degrades sharply on a dirty worktree

- Category: ergonomics
- Severity: medium
- Fix risk: low
- Return: medium

Evidence:

- current `raccoon-cli tdd` output expands to 230 changed files because it defaults to `git status`
- in an active multi-stage worktree, the guidance becomes noisy and hard to act on

Impact:

- the intended “impact-driven” workflow loses signal during exactly the kind of active wave where support tooling should help most
- contributors are pushed toward manual filtering instead of disciplined usage

### F7. Shell helper reuse is incomplete and script size has crossed maintainability thresholds

- Category: robustness, organization
- Severity: medium
- Fix risk: medium
- Return: medium
- Safe to change in parallel now: partially

Evidence:

- `scripts/utils/lib.sh` provides shared logging and JSON helpers
- `scripts/live-pipeline-activate.sh` redefines color/logging/record helpers instead of sourcing the shared library
- `scripts/diag-check.sh` also redefines overlapping helpers
- script size has grown substantially:
  `smoke-multi-symbol.sh` 1532 lines
  `smoke-analytical-e2e.sh` 1256 lines
  `smoke-os-process-operational.sh` 476 lines
  `smoke-restart-recovery.sh` 439 lines
  `live-pipeline-activate.sh` 386 lines

Impact:

- maintenance cost rises for every small script fix
- helper behavior can drift between scripts
- broad script refactors become riskier over time

### F8. Makefile reuses module helpers inconsistently

- Category: organization, robustness
- Severity: low
- Fix risk: low
- Return: medium

Evidence:

- `RUN_IN_MODULES` and `scripts/utils/for-each-module.sh` exist
- `tidy` uses shared iteration
- `test`, `test-integration`, and `test-clickhouse` each reimplement module iteration locally

Impact:

- small behavior changes must be repeated across multiple targets
- support logic is harder to evolve consistently

## Classification by Problem Type

### Ergonomics

- `raccoon-cli tdd` is too noisy on dirty worktrees
- hidden scripts and expert commands are not surfaced consistently
- there is no canonical operations home

### Robustness

- CLI remediation text points to nonexistent commands
- runtime smoke semantics are internally contradictory
- shell helper duplication increases drift risk

### Discoverability

- operations docs are buried in `docs/architecture/`
- Make and CLI expose different slices of the support toolset
- some high-value scripts have no promoted entrypoint

### Documentation Organization

- `DEVELOPMENT.md` has stale paths
- operational docs are mixed with architectural history
- stage evidence volume overwhelms current-state support guidance

### Guard Rails

- CI does not currently run the published quality-gate workflow
- support messaging overstates what is enforced automatically

## What Is Safe To Touch Now

Safe, low-conflict parallel surfaces for next stages:

- new docs under `docs/operations/`
- targeted doc repairs in `README.md`, `DEVELOPMENT.md`, and `docs/tooling/*`
- small `raccoon-cli` copy/help fixes that do not alter domain or runtime behavior
- `raccoon-cli` discoverability improvements limited to command text or guidance output

## What Should Not Be Touched Now

Do not touch during the current tranche unless explicitly coordinated:

- domain packages under `internal/domain/`, `internal/application/`, `internal/adapters/`, `internal/actors/`
- runtime composition or service contracts under `deploy/`
- active-wave files already modified in the worktree:
  `.github/workflows/ci.yml`
  `Makefile`
  `scripts/codegen-integrated-check.sh`
  `scripts/smoke-analytical-e2e.sh`
  wave-coupled new support scripts
- bulk rewrites of `docs/architecture/` or `docs/stages/`

## Safe Parallelization Summary

Best parallel Codex lanes after C1:

1. docs-only support consolidation
2. `raccoon-cli` message and discoverability corrections
3. post-wave Makefile and CI alignment
4. post-wave shell helper extraction
