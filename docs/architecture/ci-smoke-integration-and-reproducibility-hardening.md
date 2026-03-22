# CI Smoke Integration And Reproducibility Hardening

## Purpose

This document defines the architecture and rationale for integrating operational
smoke tests into CI and hardening their reproducibility. It is the canonical
reference for what smokes run in CI, why, and how they are wired.

## Problem Statement

Before S355, the CI pipeline had five jobs:

1. **unit-tests** — `make test`
2. **codegen-golden** — spec validation, golden comparison, integrated slices
3. **behavioral-scenarios** — charter-protected behavioral tests
4. **integration-tests** — NATS-backed integration tests
5. **smoke-analytical** — compose-backed E2E analytical proof

Two important verification surfaces were missing from CI:

- **`smoke-composed`** — stackless Go-test-only pipeline validation requiring
  no infrastructure, with zero reason to exclude from CI.
- **repository-checks** — repository consistency checks and quality gate, which
  guard documentation alignment, link integrity, stage indexing, and architecture
  layer boundaries.

Additionally, the `smoke-analytical` job had inline readiness polling —
duplicated shell logic that was harder to maintain and debug than a shared script.

## Integration Model

### CI Job Taxonomy

| Job | Type | Infrastructure | Gate? |
|---|---|---|---|
| unit-tests | Go tests | none | yes |
| codegen-golden | Go tests + golden snapshots | none | yes |
| behavioral-scenarios | Go tests | none | yes |
| integration-tests | Go tests | NATS service container | yes |
| smoke-composed | Go tests (wrapped in smoke script) | none | yes |
| repository-checks | bash + raccoon-cli | none (needs Rust toolchain) | yes |
| smoke-analytical | bash smoke script + compose stack | ClickHouse, NATS, all services | yes (depends on unit-tests) |

### Stackless vs Stack-Dependent

The key design axis is **stackless vs stack-dependent**:

- **Stackless smokes** (`smoke-composed`) run pure Go tests with no external
  services. They are fast, deterministic, and safe to run in any CI environment.
- **Stack-dependent smokes** (`smoke-analytical`) require a compose stack with
  ClickHouse, NATS, and all application services. They are slower, require
  readiness polling, and have inherent non-determinism from network timing.

Only stackless smokes are included in `make ci-smoke`. Stack-dependent smokes
keep their own dedicated CI jobs with explicit infrastructure setup.

### Readiness Polling

The `scripts/ci-wait-ready.sh` script replaces inline readiness loops:

- Polls ClickHouse via `clickhouse-client SELECT 1`
- Polls gateway via `curl /readyz`
- Configurable timeout (`--timeout`, default 120s)
- Optional `--skip-clickhouse` for non-analytical smokes
- Structured PASS/FAIL output using shared `lib.sh` logging
- Single reusable script for CI and local use

## Local Preflight

`make ci-preflight` provides a local pre-push gate combining:

1. `make test` — all unit tests
2. `make repo-consistency-check` — documentation and governance invariants
3. `make quality-gate` — architecture layer enforcement
4. `make smoke-composed` — stackless pipeline validation

This runs without any infrastructure and catches the majority of issues before
pushing to CI.

## Reproducibility Improvements

| Before | After |
|---|---|
| Inline readiness polling in ci.yml | Shared `ci-wait-ready.sh` script |
| No repo consistency check in CI | `repository-checks` job with caching |
| No quality gate in CI | Quality gate (CI profile) in same job |
| `smoke-composed` not in CI | Dedicated CI job |
| No local preflight target | `make ci-preflight` |
| Smoke taxonomy not CI-aware | `make smoke-help` shows CI targets |

## Design Decisions

### Why not add all smokes to CI?

Most smokes (`smoke`, `smoke-multi`, `smoke-live-stack`, `smoke-activation`,
`smoke-restart-recovery`) require live Binance WebSocket connections or long
observation windows. They are inherently non-deterministic in CI and would
produce flaky failures without value. They remain local-only proof targets.

### Why a separate `repository-checks` job?

Repository consistency checks and quality gate require the `raccoon-cli` Rust
binary. Building it in a shared job avoids duplicating the Rust toolchain setup
across multiple jobs. Caching the build artifact keeps the job fast after the
first run.

### Why `ci-smoke` only includes `smoke-composed`?

`smoke-composed` is currently the only smoke that is both stackless and
exercises meaningful composed pipeline behavior beyond what unit tests cover.
As more stackless smokes are added, they should be wired into `ci-smoke`.

## Non-Goals

- Full CI/CD pipeline with deployment stages
- Docker image building and registry push in CI
- Production deployment automation
- Notification or alerting integration
- Coverage reporting or badge generation
