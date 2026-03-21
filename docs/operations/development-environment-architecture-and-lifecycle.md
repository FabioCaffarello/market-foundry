# Development Environment Architecture And Lifecycle

## Purpose

This document defines the canonical architecture of the `market-foundry`
developer environment.

Its job is to make explicit how setup, bootstrap, local execution, validation,
smoke/proof, troubleshooting, and cleanup fit together as one repository-level
lifecycle instead of a loose set of commands, scripts, and docs.

It governs the developer environment only. It does not redefine functional
runtime behavior, service contracts, or bounded contexts.

## Design Goal

The repository should be operable through one coherent lifecycle with clear
entrypoints, predictable escalation rules, and a stable public surface.

That means:

- one primary public workflow surface;
- clear separation between canonical entrypoints and auxiliary implementations;
- one lifecycle with specialized branches, not parallel lifecycles with equal authority;
- support tooling, including `raccoon-cli`, positioned as part of the developer
  environment rather than as a competing environment of its own.

## Environment Layers

| Layer | Role | Main surfaces | Status |
|---|---|---|---|
| navigation | orient the contributor | `README.md`, `DEVELOPMENT.md`, `docs/operations/README.md` | canonical |
| workflow control | public repository entrypoints | `Makefile` | canonical |
| harness implementation | execute operational flows behind `make` | `scripts/*.sh` | auxiliary |
| support tooling | structural analysis, guidance, consistency checks | `make check`, `make tdd`, `make coverage-map`, direct `raccoon-cli` | canonical within tooling scope |
| runtime substrate | containers, env files, configs, migrations | `deploy/compose/`, `deploy/envs/`, `deploy/configs/`, `cmd/migrate` | substrate |
| historical evidence | report what stages changed in the environment | `docs/stages/` | historical |

## Canonical Architecture Rules

### Rule 1. `make` owns the public developer-environment contract

If a lifecycle step is part of normal repository use, the canonical entrypoint
must be a Make target.

Examples:

- `make bootstrap`
- `make live`
- `make up`
- `make smoke`
- `make check`
- `make verify`
- `make diag`
- `make down`

### Rule 2. Scripts implement flows; they do not outrank them

Scripts are real entrypoints for debugging and harness work, but they are not
the canonical public contract when a Make target already exists.

Direct script usage is appropriate when:

- a debug-only flag is needed;
- the harness itself is under development;
- the user is investigating below the repository workflow layer.

### Rule 3. `raccoon-cli` is support tooling, not a parallel control plane

`raccoon-cli` remains part of the developer environment, but it owns only the
tooling and governance slice of that environment:

- structural checks;
- drift and architecture analysis;
- validation planning;
- coverage and recommendation support.

It does not replace the repository lifecycle for bootstrap, bring-up, smoke,
troubleshooting, or reset.

### Rule 4. Raw substrate commands stay below the environment contract

Direct `docker compose`, `go`, and `cargo` commands are valid substrate-level
interfaces, but they are escalation paths, not first-choice repository flows.

## Lifecycle Model

The developer environment has one lifecycle with six phases.

| Phase | Goal | Canonical entrypoints | Auxiliary surfaces |
|---|---|---|---|
| bootstrap/setup | prove the machine and repo are ready | `make help`, `make bootstrap`, `make docs` | `scripts/bootstrap-check.sh`, direct toolchain commands |
| local dev loop | start or inspect the local stack and make changes | `make live`, `make live-multi`, `make up`, `make seed`, `make seed-multi`, `make check`, `make tdd`, `make verify` | `scripts/live-pipeline-activate.sh`, `scripts/seed-configctl.sh`, direct `raccoon-cli` |
| validation/checks | run fast or deep non-runtime validation | `make check`, `make verify`, `make check-deep`, `make test*`, `make arch-guard`, `make repo-consistency-check` | direct `raccoon-cli`, direct `go test` |
| smoke/proofs | prove runtime behavior for the changed surface | `make smoke-help`, `make smoke*` | `scripts/smoke-*.sh` |
| troubleshooting | inspect stack health and narrow failures | `make diag`, `make ps`, `make logs`, `make restart` | `scripts/diag-check.sh`, direct scripts, raw compose commands |
| cleanup/reset | stop or clean local state and rebuild confidence | `make down`, `make clean`, `make up`, `make seed*`, `make smoke` | raw compose cleanup and low-level tooling |

## Lifecycle Topology

```text
discover -> bootstrap -> bring-up -> change loop -> targeted proof -> troubleshoot/reset

make help/docs
  -> make bootstrap
  -> make live | (make up -> make seed*)
  -> make check -> make tdd -> implement -> make verify
  -> make smoke-help -> relevant make smoke*
  -> make diag / make ps / make logs / SERVICE=... make restart
  -> make down / make clean -> bring-up again when needed
```

## Canonical Versus Auxiliary Entrypoints

### Canonical public entrypoints

- `make help`
- `make docs`
- `make bootstrap`
- `make live`, `make live-check`, `make live-multi`, `make live-multi-check`
- `make up`, `make down`, `make restart`, `make logs`, `make ps`
- `make seed`, `make seed-multi`
- `make check`, `make tdd`, `make verify`, `make check-deep`
- `make test*`, `make arch-guard`, `make repo-consistency-check`
- `make smoke-help`, `make smoke*`
- `make diag`
- `make clean`

### Auxiliary or expert entrypoints

- `scripts/bootstrap-check.sh`
- `scripts/live-pipeline-activate.sh`
- `scripts/seed-configctl.sh`
- `scripts/diag-check.sh`
- `scripts/smoke-*.sh`
- direct `raccoon-cli`
- raw `docker compose`
- raw `go`
- raw `cargo`

## Current Repository Diagnosis

The repository already has most lifecycle pieces, but they were distributed
across several documents and command families:

- setup existed through `make bootstrap`, but the environment architecture was
  implicit rather than explicitly documented;
- bring-up had both `live*` and `up`/`seed*` paths, but the hierarchy between
  them was spread across multiple docs;
- smoke/proof guidance was well developed, but lived separately from the broader
  lifecycle description;
- troubleshooting entrypoints existed, but were mostly described as adjacent
  runbooks rather than as a fixed lifecycle phase;
- cleanup/reset existed operationally through `make down` and `make clean`, but
  was under-documented as part of the official lifecycle;
- tooling guidance existed, but the place of `raccoon-cli` inside the broader
  developer environment remained easy to overstate.

## Unification Applied In C17

This stage applies the following environment-level decisions:

- the developer environment is now explicitly documented as a single lifecycle
  with six phases;
- `make` is reaffirmed as the canonical public control surface for that
  lifecycle;
- `live*` is defined as orchestration convenience, while `up` + `seed*` remains
  the controlled manual path inside the same lifecycle;
- `smoke*` is defined as the proof-of-record branch of the lifecycle, not a
  separate workflow;
- troubleshooting and cleanup/reset are promoted from implicit practices to
  explicit lifecycle phases;
- `raccoon-cli` is positioned as support tooling inside the environment rather
  than as the goal of the environment itself.

## Operational Invariants

- no new parallel lifecycle should be introduced without a clearly superior
  canonical owner;
- new routine developer workflows should land in `make` first;
- new scripts should be introduced as harness implementations or expert surfaces,
  not as undocumented public APIs;
- new tooling commands should be mapped back to a lifecycle phase so their role
  stays legible;
- new docs should link into this lifecycle model instead of restating an
  alternate workflow hierarchy.

## Related Documents

- [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- [`developer-workflow-unification.md`](developer-workflow-unification.md)
- [`development-lifecycle-entrypoints-and-canonical-flows.md`](development-lifecycle-entrypoints-and-canonical-flows.md)
- [`developer-onboarding-and-troubleshooting-guide.md`](developer-onboarding-and-troubleshooting-guide.md)
- [`repository-support-surface-canonical-model.md`](repository-support-surface-canonical-model.md)
- [`makefile-targets-reference-and-conventions.md`](makefile-targets-reference-and-conventions.md)
- [`scripts-catalog-and-usage-guide.md`](scripts-catalog-and-usage-guide.md)
