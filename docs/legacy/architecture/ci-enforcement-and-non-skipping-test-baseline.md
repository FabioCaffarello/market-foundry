# CI Enforcement and Non-Skipping Test Baseline

Status: **Delivered** — S282
Date: 2026-03-21

## Purpose

This document defines the test baseline after S282 hardening. It establishes
which tests are mandatory in CI, which are conditional, and the enforcement
contract for the Signal Evolution Wave (S283–S287).

## Baseline Definition

### Mandatory CI Jobs (all must pass for merge)

| Job | Target | What It Proves | Auto-Skip Count |
|-----|--------|---------------|-----------------|
| `unit-tests` | `make test` | All pure unit tests across 16 modules | **0** |
| `codegen-golden` | `make codegen-validate-all` + `make codegen-check` + `make codegen-test` + `make codegen-integrated` | Spec validity, golden snapshot equivalence, integrated slice match | **0** |
| `behavioral-scenarios` | `make test-behavioral` + `make test-behavioral-roundtrip` | Charter-protected domain scenarios, serialization round-trips | **0** |
| `integration-tests` | `make test-integration` | NATS-dependent integration tests (KV, control gate, control plane, multi-binary, restart recovery) | **0** |
| `smoke-analytical` | `make smoke-analytical` | Full-stack analytical E2E (NATS→writer→ClickHouse→reader→HTTP) | **0** |

### Conditional / Local-Only Tests

| Target | Tag | What It Proves | When to Run |
|--------|-----|---------------|-------------|
| `make test-clickhouse` | `requireclickhouse` | Live ClickHouse analytical round-trip (S277: LAE-1 through LAE-9) | Local dev with compose stack up, or dedicated CI with ClickHouse service |

## Pre-S282 vs Post-S282

### Before (Dishonest Baseline)

- `unit-tests` CI job: compiled and "passed" 40 tests that auto-skipped
- `integration-tests` CI job: ran with `-tags=integration` but **no files** had that tag — functionally identical to `make test` without NATS
- Net effect: 40 tests existed in the repository that **never executed in CI**

### After (Honest Baseline)

- `unit-tests`: only compiles tests without build tags → zero skips
- `integration-tests`: compiles `//go:build integration` tests, NATS available via service container → 39 tests actually execute
- `requireclickhouse` tests: excluded from CI by build tag, explicitly available via `make test-clickhouse`
- Net effect: **every test that compiles in CI either passes or fails**

## Build Tag Taxonomy

| Tag | Meaning | CI Coverage |
|-----|---------|------------|
| *(none)* | Pure unit test, no external dependencies | `unit-tests` job |
| `integration` | Requires NATS server with JetStream | `integration-tests` job (NATS service container) |
| `requireclickhouse` | Requires live ClickHouse instance | Not in CI; covered by `smoke-analytical` at system level |

## Enforcement Contract for S283–S287

1. **New unit tests** must not introduce `t.Skip` for infrastructure reasons. Use build tags instead.
2. **New NATS-dependent tests** must use `//go:build integration`.
3. **New ClickHouse-dependent tests** must use `//go:build requireclickhouse`.
4. **`make test` must have zero auto-skips.** Any `t.Skip` in a non-tagged file requires explicit justification.
5. **All CI jobs must remain green** before merging any S283–S287 stage.

## Infrastructure in CI

### NATS Service Container (integration-tests job)

```yaml
services:
  nats:
    image: nats:2.10.18-alpine
    ports:
      - 4222:4222
      - 8222:8222
```

The NATS image includes JetStream support. Tests connect via `NATS_URL=nats://localhost:4222`.

### ClickHouse (smoke-analytical job only)

ClickHouse is started via Docker Compose in the `smoke-analytical` job.
It is not available as a standalone service container in other jobs.
The `smoke-analytical` shell scripts validate the same data path that
`TestLiveAnalyticalExecution_FullRoundTrip` covers in Go.

## Makefile Targets Summary

| Target | Build Tag | Infrastructure | Cache |
|--------|-----------|---------------|-------|
| `make test` | *(none)* | None | Go default (cached) |
| `make test-integration` | `integration` | NATS | `-count=1` (no cache) |
| `make test-clickhouse` | `requireclickhouse` | ClickHouse | `-count=1` (no cache) |
| `make test-behavioral` | *(none)* | None | `-count=1` (no cache) |
| `make test-behavioral-roundtrip` | *(none)* | None | `-count=1` (no cache) |
| `make codegen-test` | *(none)* | None | `-count=1` (no cache) |
