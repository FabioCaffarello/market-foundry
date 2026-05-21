# Development Lifecycle Entrypoints And Canonical Flows

## Purpose

This document maps the official developer-environment lifecycle to the real
entrypoints present in the repository.

Use it when the question is operational:

- what command should I start with?
- what is the canonical path for this phase?
- what is allowed but auxiliary?
- how do I recover when the canonical path fails?

## Phase Map

| Lifecycle phase | Canonical goal | Canonical entrypoints | Main supporting surfaces |
|---|---|---|---|
| bootstrap/setup | validate machine and repo readiness | `make help`, `make bootstrap`, `make docs` | `README.md`, `DEVELOPMENT.md`, `scripts/bootstrap-check.sh` |
| local dev loop | bring up a usable stack and execute day-to-day work | `make live` or `make up` + `make seed*`; then `make check`, `make tdd`, `make verify` | `scripts/live-pipeline-activate.sh`, `scripts/seed-configctl.sh`, `raccoon-cli` wrappers |
| validation/checks | validate code and repository integrity | `make check`, `make tdd`, `make verify`, `make check-deep`, `make test*`, `make arch-guard` | direct `raccoon-cli`, direct `go test` |
| smoke/proofs | prove runtime behavior | `make smoke-help`, `make smoke*` | `scripts/smoke-*.sh` |
| troubleshooting | inspect health, logs, and narrow recovery | `make diag`, `make ps`, `make logs`, `make restart` | `scripts/diag-check.sh`, raw compose commands |
| cleanup/reset | stop, clean, and reconstruct confidence | `make down`, `make clean`, then canonical bring-up again | raw compose cleanup, Go cache cleanup |

## Canonical Entrypoint Inventory

### Bootstrap and discovery

| Entrypoint | Role | Notes |
|---|---|---|
| `make help` | discover supported targets | first scan of the public surface |
| `make docs` | discover canonical docs | points to the main workflow and environment docs |
| `make bootstrap` | validate machine and repo readiness | canonical setup check |

### Bring-up and runtime lifecycle

| Entrypoint | Role | Notes |
|---|---|---|
| `make live` | fastest single-symbol bring-up | build + start + migrate + seed + validate |
| `make live-check` | validate already-running single-symbol stack | no reseed or restart |
| `make live-multi` | fastest governed multi-symbol bring-up | same shape for multi-symbol |
| `make live-multi-check` | validate already-running multi-symbol stack | no reseed or restart |
| `make up` | controlled manual stack start | starts compose stack and applies migrations |
| `make seed` | activate default single-symbol config | manual path step |
| `make seed-multi` | activate default multi-symbol config | manual path step |
| `make down` | stop stack | canonical runtime stop/reset entrypoint |

### Validation and analysis

| Entrypoint | Role | Notes |
|---|---|---|
| `make check` | pre-change fast guard rail | repo consistency + fast quality gate |
| `make tdd` | impact-driven validation planning | wraps `raccoon-cli` |
| `make verify` | post-change validation | tests + repo consistency + fast quality gate |
| `make check-deep` | deep validation profile | significant changes only |
| `make test`, `make test-*` | direct Go validation | narrower validation paths |
| `make arch-guard` | enforce layer boundaries | architecture check |
| `make repo-consistency-check` | support-surface/doc consistency | lightweight environment hygiene |

### Smoke and proof entrypoints

| Entrypoint | Role | Notes |
|---|---|---|
| `make smoke-help` | choose the right proof | first stop before heavy proof runs |
| `make smoke` | baseline single-symbol proof | default proof-of-record |
| `make smoke-multi` | multi-symbol proof | breadth and isolation |
| `make smoke-analytical` | analytical path proof | writer/ClickHouse/read path |
| `make smoke-round-trip` | persistence round-trip proof | specialized path |
| `make smoke-live-stack` | live stack plus gateway verification | specialized path |
| `make smoke-activation` | activation control-surface proof | specialized path |
| `make smoke-composed` | composed pipeline proof without the full stack | specialized path |
| `make smoke-operational` | OS-process operational proof | lifecycle and isolation |
| `make smoke-restart-recovery` | restart and recovery proof | durability and recovery |

### Troubleshooting and recovery

| Entrypoint | Role | Notes |
|---|---|---|
| `make diag` | quick runtime diagnostic snapshot | first troubleshooting command |
| `make ps` | service/container status | confirm running state |
| `make logs SERVICE=...` | scoped logs | inspect one surface before broad restarts |
| `SERVICE=... make restart` | narrow service restart | controlled recovery before whole-stack restart |
| `make clean` | local build/cache cleanup | complements `make down`; not a runtime teardown |

## Canonical Flows

### Flow 1. New machine or changed environment

```bash
make help
make docs
make bootstrap
make live
make smoke
```

Use when onboarding or after toolchain, Docker, or environment drift.

### Flow 2. Fastest daily path

```bash
make check
make tdd
# implement the smallest correct change
make verify
make smoke-help
make smoke
```

Replace `make smoke` with the narrowest `make smoke*` that proves the changed
behavior.

### Flow 3. Controlled manual bring-up

```bash
make up
make seed          # or make seed-multi
make diag
make smoke         # or relevant make smoke*
```

Use when you need visibility into startup and seeding as separate steps.

### Flow 4. Multi-symbol path

```bash
make live-multi
make smoke-multi
```

Or, if manual control is required:

```bash
make up
make seed-multi
make smoke-multi
```

### Flow 5. Troubleshoot a running stack

```bash
make diag
make ps
make logs SERVICE=gateway
make logs SERVICE=derive
SERVICE=gateway make restart
```

Escalate to direct scripts or raw compose only when the canonical path does not
expose enough detail.

### Flow 6. Reset and rebuild confidence

```bash
make down
make clean
make up
make seed
make smoke
```

Use `make live` instead of the manual sequence when you want the repository to
own the whole recovery path.

## Entrypoint Hierarchy

### Canonical public surface

- `make`
- `README.md`
- `DEVELOPMENT.md`
- `docs/operations/README.md`
- this document

### Auxiliary support surface

- `scripts/*.sh`
- direct `raccoon-cli`
- direct `go test`
- direct `docker compose`
- direct `cargo`

### Private implementation surface

- `scripts/utils/*`
- internal implementation details of `tools/raccoon-cli/`

## Known Fragmentation That This Model Resolves

- `live*` and `up`/`seed*` now have an explicit hierarchy: same lifecycle,
  different control level.
- smoke harnesses are no longer documented as isolated capabilities; they are
  the proof branch of the lifecycle.
- troubleshooting now has a fixed first-line order instead of scattered hints.
- cleanup/reset is now documented as an official phase rather than an implied
  fallback.
- `raccoon-cli` is explicitly mapped to validation and analysis, preventing it
  from being mistaken for the repository's primary runtime operator surface.
- the grouped CLI taxonomy now reads as the expert intelligence layer behind
  `make`, instead of as a second competing workflow taxonomy.

## When To Escalate Beyond Canonical Flows

Use direct scripts when:

- you need `--wait` or another debug-only flag;
- you are debugging the harness implementation itself.

Use direct `raccoon-cli` when:

- you need JSON output;
- you need narrower structural analysis than the Make wrapper exposes;
- you are modifying the CLI.

Use raw substrate commands when:

- debugging compose, Go, or Cargo behavior directly;
- evolving those substrate layers rather than using the repository workflow.

## Related Documents

- [`development-environment-architecture-and-lifecycle.md`](development-environment-architecture-and-lifecycle.md)
- [`developer-workflow-unification.md`](developer-workflow-unification.md)
- [`developer-onboarding-and-troubleshooting-guide.md`](developer-onboarding-and-troubleshooting-guide.md)
- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md)
