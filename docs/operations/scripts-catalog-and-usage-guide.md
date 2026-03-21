# Scripts Catalog And Usage Guide

## Purpose

This guide catalogs the current repository script surface and states which entrypoint
should be considered canonical for each workflow.

Rule of thumb:

- prefer `make` for public operator/developer flows
- use `make smoke-help` when you need to choose the right proof quickly
- use `make smoke*` as the operational proof-of-record surface
- treat `make live*` as orchestration wrappers, not as competing smoke entrypoints
- call `scripts/*.sh` directly for lower-level debugging or extra flags
- treat `scripts/utils/*` as support utilities, not user-facing workflows
- use [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md) when deciding whether a new workflow belongs in `make`, a direct script, or `raccoon-cli`

## Canonicality Rules

- If a workflow already exists in `make`, that Make target is the canonical entrypoint.
- A script may expose additional flags or debugging detail without becoming a competing public API.
- `scripts/utils/*` are implementation support only and should never be documented as primary entrypoints.
- When a script becomes routine for contributors or operators, add or update the corresponding Make target instead of promoting raw script usage as the new default.

## Catalog By Category

| Category | Canonical Entry | Direct Script | Use When | Notes |
|---|---|---|---|---|
| bootstrap/setup | `make bootstrap` | `scripts/bootstrap-check.sh` | Validate local prerequisites and repository entrypoints before first use | Setup validation, not stack bring-up |
| bootstrap/setup | `make seed` | `scripts/seed-configctl.sh` | Activate default single-symbol config bindings | Supports `--multi-symbol` and `SYMBOLS=...` |
| bootstrap/setup | `make seed-multi` | `scripts/seed-configctl.sh --multi-symbol` | Activate default multi-symbol bindings | Same lifecycle as `make seed` |
| bootstrap/setup | `make live` | `scripts/live-pipeline-activate.sh` | Build, start, seed, and validate the single-symbol live stack | Canonical full-stack bring-up |
| bootstrap/setup | `make live-check` | `scripts/live-pipeline-activate.sh --check-only` | Validate an already-running single-symbol stack | No reseed or restart |
| bootstrap/setup | `make live-multi` | `scripts/live-pipeline-activate.sh --multi-symbol` | Build, start, seed, and validate the multi-symbol live stack | Multi-symbol activation flow |
| bootstrap/setup | `make live-multi-check` | `scripts/live-pipeline-activate.sh --multi-symbol --check-only` | Validate a running multi-symbol stack | Useful for repeat checks |
| smoke/integration | `make smoke` | `scripts/smoke-first-slice.sh` | Run the smallest operational E2E proof | First vertical slice coverage |
| smoke/integration | `make smoke-multi` | `scripts/smoke-multi-symbol.sh` | Run multi-symbol operational proof | Broadest KV/query smoke |
| smoke/integration | `make smoke-analytical` | `scripts/smoke-analytical-e2e.sh` | Prove analytical writer/reader path | Large harness; active domain-adjacent surface |
| smoke/integration | `make smoke-operational` | `scripts/smoke-os-process-operational.sh` | Prove isolated-process operational behavior | OS-process/container operational proof |
| smoke/integration | `make smoke-restart-recovery` | `scripts/smoke-restart-recovery.sh` | Prove restart/recovery resilience | Durable consumer and gate recovery proof |
| local dev | `make diag` | `scripts/diag-check.sh` | Capture a quick runtime health snapshot | Supports `--local` |
| docs/tooling | `make repo-consistency-check` | `scripts/repository-consistency-check.sh` | Run lightweight repository-policy and support-surface checks | Canonical repository-policy guard rail |
| docs/tooling | `make stage-help`, `make stage-scaffold`, `make stage-check` | `scripts/stage-tooling.sh` | Scaffold or validate one governed stage | Lightweight stage-support helper, not a workflow engine |
| docs/tooling | `make codegen-integrated` | `scripts/codegen-integrated-check.sh` | Verify governed integrated slices | Golden-to-target check |
| docs/tooling | `make codegen-equivalence` | `scripts/codegen-equivalence-check.sh` | Run cross-artifact codegen equivalence validation | Wider consistency harness |
| local dev | none | `scripts/utils/list-modules.sh` | Print Go workspace modules | Helper for repo tooling |
| local dev | none | `scripts/utils/for-each-module.sh <cmd...>` | Run a command in each Go workspace module | Honors `MODULE=...` |
| shared harness support | none | `scripts/utils/lib.sh` | Source-only helper library | Not an entrypoint |

## Recommended Usage Paths

### Daily local development

Use:

- `make bootstrap` when the machine or local environment changed
- `make check`
- `make stage-check STAGE_ID=...` when closing a governed stage
- `make tdd`
- `make test`
- `make verify`
- `make diag` when the stack is already running

### Local stack bring-up

Use:

- `make up`
- `make seed` or `make seed-multi`
- `make smoke` or `make smoke-multi`

Use `make live*` when you want the repository to orchestrate the whole activation
sequence for you. The canonical proof surface after bring-up still remains the
relevant `make smoke*` target.

### Analytical and deeper runtime proofs

Use:

- `make smoke-help`
- `make smoke-analytical`
- `make smoke-operational`
- `make smoke-restart-recovery`

These harnesses are heavier and more stage-like than the first-slice smoke.
They are still canonical operational proofs because they prove distinct runtime
properties instead of duplicating `make smoke`.

### Codegen and governed artifact checks

Use:

- `make codegen-check`
- `make codegen-test`
- `make codegen-integrated`
- `make codegen-equivalence`
- `make codegen-validate-all`

Prefer the `make` wrappers unless you are debugging the shell harness itself.

## Direct Script Invocation Guide

### `scripts/seed-configctl.sh`

Common forms:

- `./scripts/seed-configctl.sh`
- `./scripts/seed-configctl.sh --multi-symbol`
- `SYMBOLS="btcusdt,ethusdt,solusdt" ./scripts/seed-configctl.sh`

Use directly when:

- you want custom symbols
- you do not need the full `make live*` orchestration

### `scripts/bootstrap-check.sh`

Common forms:

- `./scripts/bootstrap-check.sh`

Use directly when:

- you are debugging bootstrap/setup validation itself
- you want the same checks behind `make bootstrap` without invoking `make`

### `scripts/stage-tooling.sh`

Common forms:

- `./scripts/stage-tooling.sh help`
- `STAGE_ID=C15 STAGE_SLUG=stage-tooling STAGE_TITLE="Stage Tooling" ./scripts/stage-tooling.sh scaffold`
- `STAGE_ID=C15 STAGE_SLUG=stage-tooling ./scripts/stage-tooling.sh check`

Use directly when:

- you are debugging the stage helper itself
- you need stage support without routing through `make`

### `scripts/diag-check.sh`

Common forms:

- `./scripts/diag-check.sh`
- `./scripts/diag-check.sh --local`

Use directly when:

- you want a quick health snapshot
- services are running on the host rather than through compose exec

### Smoke scripts

Common forms:

- `./scripts/smoke-first-slice.sh --wait 120`
- `./scripts/smoke-multi-symbol.sh --wait 180`
- `./scripts/smoke-os-process-operational.sh --wait 180`
- `./scripts/smoke-restart-recovery.sh --wait 180`
- `./scripts/smoke-analytical-e2e.sh --wait 180`

Use direct invocation when:

- you need to override waiting behavior
- you are debugging harness behavior rather than the public Make target

Common public overrides that avoid direct script invocation:

- `SMOKE_WAIT=180 make smoke`
- `SMOKE_WAIT=240 make smoke-analytical`
- `BASE_URL=http://127.0.0.1:18080 make smoke-operational`

## Discoverability Improvements Introduced In C3

The following scripts now support `--help` and explicit argument validation:

- `scripts/bootstrap-check.sh`
- `scripts/seed-configctl.sh`
- `scripts/diag-check.sh`
- `scripts/live-pipeline-activate.sh`
- `scripts/smoke-first-slice.sh`
- `scripts/smoke-multi-symbol.sh`
- `scripts/smoke-os-process-operational.sh`
- `scripts/smoke-restart-recovery.sh`
- `scripts/stage-tooling.sh`
- `scripts/codegen-integrated-check.sh`
- `scripts/codegen-equivalence-check.sh`
- `scripts/utils/for-each-module.sh`
- `scripts/utils/list-modules.sh`

## Current Structural Risks

The following risks remain visible after C3:

- `scripts/smoke-analytical-e2e.sh` remains large and monolithic
- `scripts/smoke-multi-symbol.sh` remains very large despite improved entry hygiene
- some support surfaces are documented through `make`, while others remain script-only helpers

These are follow-up concerns, not blockers for the current support normalization.
