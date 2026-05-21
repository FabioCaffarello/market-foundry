# Test Skip Cause Matrix and Remediation

Status: **Delivered** — S282
Date: 2026-03-21

## Context

Prior to S282, 40 tests in the repository contained runtime auto-skip logic
(`t.Skip`/`t.Skipf`) that silently passed in CI without exercising any code.
These tests inflated the test count, gave false confidence in the baseline,
and obscured the boundary between unit and integration tests.

## Skip Inventory (Pre-S282)

| # | File | Tests | Skip Trigger | Category |
|---|------|-------|-------------|----------|
| 1 | `internal/adapters/nats/natsexecution/kv_store_roundtrip_test.go` | 8 | NATS not reachable at localhost:4222 | NATS-external |
| 2 | `internal/adapters/nats/natsexecution/control_gate_runtime_test.go` | 6 | NATS not reachable (via shared `natsURL()`) | NATS-external |
| 3 | `internal/adapters/nats/natsexecution/control_plane_full_path_test.go` | 5 | NATS not reachable (via shared `natsURL()`) | NATS-external |
| 4 | `internal/adapters/nats/natsexecution/multi_binary_integration_test.go` | 6 | NATS not reachable (via shared `natsURL()`) | NATS-external |
| 5 | `internal/adapters/nats/natsexecution/restart_recovery_test.go` | 10 | NATS not reachable (via shared `natsURL()`) | NATS-external |
| 6 | `internal/adapters/clickhouse/writerpipeline/restart_recovery_test.go` | 4 | NATS not reachable (via `wrNATSURL()`) | NATS-external |
| 7 | `internal/adapters/clickhouse/live_execution_analytical_test.go` | 1 | `CLICKHOUSE_DSN` not set or ClickHouse not reachable | ClickHouse-external |

**Total: 40 tests auto-skipping in CI.**

## Classification by Cause

### Category 1: NATS-External (39 tests)

**Root cause:** Tests require a running NATS server with JetStream support.
They use a `natsURL()` helper that does a TCP dial to `localhost:4222` and
calls `t.Skipf()` on connection failure.

**Why they auto-skipped in CI:** The `unit-tests` job runs `make test` (no
infrastructure). The `integration-tests` job runs `make test-integration`
with `-tags=integration`, but these files had **no build tags** — so they
compiled in both contexts and auto-skipped in both because no NATS was
available.

**Remediation:** Added `//go:build integration` build tag to all 6 files.
Modified the CI `integration-tests` job to start NATS as a service container.
The `t.Skipf()` helpers are retained as defense-in-depth but are no longer
the primary gate.

**Affected stages:** S271 (KV Round-Trip), S273 (Control Gate Runtime),
S275 (Control Plane Full-Path), S276 (Multi-Binary Integration),
S280 (Durable Restart and Recovery).

### Category 2: ClickHouse-External (1 test)

**Root cause:** Test requires a live ClickHouse instance with network
connectivity and DDL permissions.

**Why it auto-skipped in CI:** The `unit-tests` job has no ClickHouse.
The `smoke-analytical` job validates the same data path via shell scripts
but does not run this Go test directly.

**Remediation:** Added `//go:build requireclickhouse` build tag. Added
`make test-clickhouse` Makefile target. This test remains local-only or
runnable when the compose stack is up. The `smoke-analytical` CI job covers
the same analytical round-trip path via end-to-end shell scripts.

**Affected stage:** S277 (Live Analytical Execution Proof).

### Category 3: Harness Fragility — None Found

No tests were found to skip due to harness issues, test ordering, race
conditions, or flaky assertions.

### Category 4: Structural Debt — None Correctable Without Refactor

No tests were found to skip due to correctable structural debt within the
S282 scope.

## Remediation Summary

| Action | Files Changed | Tests Affected | Result |
|--------|--------------|----------------|--------|
| Added `//go:build integration` | 6 files | 39 tests | Excluded from `make test`; run in CI with NATS service |
| Added `//go:build requireclickhouse` | 1 file | 1 test | Excluded from `make test`; available via `make test-clickhouse` |
| Added NATS service container to CI | `ci.yml` | 39 tests | Tests now actually execute in CI |
| Added `make test-clickhouse` target | `Makefile` | 1 test | Explicit invocation with `CLICKHOUSE_DSN` |

## Residual Skip Behavior

After S282, the auto-skip helpers (`natsURL()`, `wrNATSURL()`,
`skipUnlessClickHouse()`) remain in the code as **defense-in-depth**.
They only trigger if someone runs with the build tag but without
infrastructure — a developer convenience, not a CI escape hatch.

**In CI, zero tests auto-skip.** Every test that compiles either passes or fails.
