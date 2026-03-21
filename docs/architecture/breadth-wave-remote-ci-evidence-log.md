# Breadth Wave Remote CI Evidence Log

**Date:** 2026-03-21
**Stage:** S245
**Charter:** BREADTH-WAVE-1

---

## CI Run 1 — Initial Push (FAILED)

| Field | Value |
|-------|-------|
| Run ID | 23375415266 |
| URL | https://github.com/FabioCaffarello/market-foundry/actions/runs/23375415266 |
| Commit | `95c7cc2` |
| Trigger | push to main |
| Date | 2026-03-21T08:06:12Z |

### Job Results

| Job | Result | Duration |
|-----|--------|----------|
| Unit Tests | PASS | 1m29s |
| Codegen Golden Equivalence | PASS | 25s |
| Integration Tests | PASS | 1m29s |
| Smoke Analytical E2E | **FAIL** | 5m21s |

### Failure Analysis

- **Failed step:** Start stack (compose up)
- **Error:** `code: 62, message: Syntax error (Multi-statements are not allowed)` in migration `007_add_decision_severity_rationale.sql`
- **Root cause:** Two `ALTER TABLE` statements in a single migration file; ClickHouse only accepts one statement per query
- **Fix applied:** Combined into single `ALTER TABLE ... ADD COLUMN ..., ADD COLUMN ...` statement

### Observation

Three out of four jobs passed. The failure was exclusively in the infrastructure setup (migration application), not in any test assertion. This confirms the breadth wave code itself is correct; only a migration authoring oversight caused the failure.

---

## CI Run 2 — After Migration Fix (PASSED)

| Field | Value |
|-------|-------|
| Run ID | 23375533952 |
| URL | https://github.com/FabioCaffarello/market-foundry/actions/runs/23375533952 |
| Commit | `516236d` |
| Trigger | push to main |
| Date | 2026-03-21T08:13:48Z |

### Job Results

| Job | Result | Duration |
|-----|--------|----------|
| Unit Tests | PASS | 1m31s |
| Codegen Golden Equivalence | PASS | 30s |
| Integration Tests | PASS | 1m34s |
| Smoke Analytical E2E | PASS | 7m23s |

### Verification Scope

The CI run validated the complete breadth wave (S241–S244) plus the migration fix:

**Unit Tests** covered:
- `internal/application/decision` — rsi_oversold + ema_crossover evaluators
- `internal/application/strategy` — mean_reversion_entry + trend_following_entry resolvers
- `internal/application/risk` — position_exposure + drawdown_limit evaluators
- `internal/domain/decision`, `internal/domain/strategy`, `internal/domain/risk` — type validation
- `internal/actors/scopes/derive` — actor tests for all 6 types
- `internal/adapters/clickhouse` — reader tests with new columns
- `internal/adapters/nats` — registry/publisher/kv for all domains
- `internal/interfaces/http` — handler and route tests

**Codegen Golden Equivalence** covered:
- 10 families validated (spec syntax + cross-spec uniqueness)
- 20 golden comparisons passed (including 6 new breadth families)
- 4 integrated slice verifications passed

**Integration Tests** covered:
- Actor chain integration with embedded NATS for both pipeline chains
- Store projection wiring

**Smoke Analytical E2E** covered:
- Full Docker Compose stack boot (NATS, ClickHouse, gateway, derive, store, writer, etc.)
- Migration application (all 7 migrations including the fixed 007)
- Config seed via configctl
- HTTP endpoint smoke assertions

---

## Annotations (Non-blocking)

Both runs produced the following warnings, none of which affect correctness:

1. **Node.js 20 deprecation** — GitHub Actions will force Node.js 24 starting June 2026. Actions `checkout@v4`, `setup-go@v5`, `upload-artifact@v4` will need updates before then.
2. **Go module cache miss** — `go.sum` not found at expected path for cache restore. Doesn't affect build correctness, only cache performance.
