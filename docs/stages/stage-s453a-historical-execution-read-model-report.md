# Stage S453A: Historical Execution Read Model Report

Stage: S453A
Wave: Operational Memory Hardening
Predecessor: S452A
Date: 2026-03-24

## Objective

Design, implement, validate, and document a historical read model for execution lifecycle, enabling chronological reconstruction of order trajectories without depending on latest-only KV surfaces or separate per-type queries.

## Executive Summary

S453A delivers a unified lifecycle history query surface backed by ClickHouse. The `GET /analytical/execution/lifecycle` endpoint returns all execution event types (paper_order, venue_market_order, venue_rejection) for a given source/symbol/timeframe in reverse-chronological order. No schema migration was required -- the existing `executions` table already contains all necessary data. The stage adds a new ClickHouse reader method, analytical contract, use case, HTTP handler, and gateway route. 19 new tests pass. 3 architecture documents produced.

## What Changed

### Code Changes

| Layer | File | Change |
|-------|------|--------|
| Adapter | `internal/adapters/clickhouse/execution_reader.go` | Added `QueryLifecycleHistory()` and `BuildLifecycleHistoryQuery()` |
| Contract | `internal/application/analyticalclient/contracts.go` | Added `LifecycleHistoryQuery`, `LifecycleHistoryEntry`, `LifecycleHistoryReply` |
| Use case | `internal/application/analyticalclient/get_lifecycle_history.go` | New `LifecycleHistoryReader` interface, `GetLifecycleHistoryUseCase` |
| Handler | `internal/interfaces/http/handlers/analytical.go` | Added `getAnalyticalLifecycleHistoryUseCase` interface, `getLifecycleHistory` field, `GetLifecycleHistory()` method |
| Route | `internal/interfaces/http/routes/analytical.go` | Added `GetLifecycleHistory` to `AnalyticalFamilyDeps`, registered `/analytical/execution/lifecycle` |
| Composition | `cmd/gateway/compose.go` | Wired `GetLifecycleHistory` use case |
| Reader factory | `cmd/gateway/analytical_reader.go` | Added `newAnalyticalLifecycleReader()` |

### Test Changes

| File | Tests | Coverage |
|------|-------|----------|
| `internal/adapters/clickhouse/s453a_lifecycle_history_test.go` | 9 tests | Query builder: no-type filter, side/status/since/until/all filters, select columns, timeframe type |
| `internal/application/analyticalclient/s453a_lifecycle_history_test.go` | 10 tests | Use case: happy path, empty result, reader error, missing source/symbol/timeframe, invalid time range, limit clamping, nil use case, timestamp format |

### Documentation

| Document | Path |
|----------|------|
| Read model design | `docs/architecture/historical-execution-and-lifecycle-read-model.md` |
| Sources and limitations | `docs/architecture/execution-lifecycle-history-sources-projection-semantics-and-limitations.md` |
| Stage report | `docs/stages/stage-s453a-historical-execution-read-model-report.md` |

## Capabilities Delivered

| # | Capability | Status |
|---|-----------|--------|
| C-1 | Lifecycle history query across all exec types for a given partition key | FULL |
| C-2 | ClickHouse-backed historical read model (no new table/migration) | FULL |
| C-3 | Optional side and status filters on lifecycle history | FULL |
| C-4 | Time range filtering (since/until) on lifecycle history | FULL |
| C-5 | HTTP endpoint at `/analytical/execution/lifecycle` | FULL |
| C-6 | Gateway composition with graceful degradation when ClickHouse unavailable | FULL |
| C-7 | RFC3339 timestamp format in lifecycle entries | FULL |
| C-8 | Correlation ID preserved for lifecycle grouping | FULL |
| C-9 | Query builder tests (9 tests) | FULL |
| C-10 | Use case tests with validation, error handling, edge cases (10 tests) | FULL |

**Score: 10/10 FULL, 0 PARTIAL, 0 PENDING**

## Key Design Decisions

### D-1: Reuse executions table

The existing `executions` table already stores all three event types with a `type` discriminator column. The lifecycle history query simply removes `type` from the mandatory WHERE clause. No schema migration needed.

### D-2: Separate reader interface

`LifecycleHistoryReader` is a distinct interface from `ExecutionReader`. This avoids widening the existing interface contract and keeps the lifecycle concern isolated. The concrete `*clickhouse.ExecutionReader` satisfies both interfaces.

### D-3: LifecycleHistoryEntry with string fields

The entry type uses string representations for side, status, and timestamp rather than domain types. This avoids JSON serialization edge cases and provides a stable wire format independent of domain model evolution.

## Acceptance Criteria Evaluation

| Criterion | Met? | Evidence |
|-----------|------|---------|
| Historical surface exists and is useful | YES | `/analytical/execution/lifecycle` endpoint returns unified timeline |
| Operational memory materially improved | YES | Single query reconstructs full lifecycle trajectory; previously required 3 separate queries + client merge |
| Reduces latest-only dependence | YES | Historical reconstruction no longer requires KV surfaces; ClickHouse provides 90-day window |
| Ready for ergonomic queries in S454A | YES | Contracts and query builder support time range, status, side filters; extension points clear |

## Guard Rails Compliance

| Guard rail | Status |
|-----------|--------|
| No lakehouse/analytics inflation | COMPLIANT -- single query added to existing table, no new infrastructure |
| No broad dashboards | COMPLIANT -- raw event timeline only, no aggregation or visualization |
| No masked limitations | COMPLIANT -- 6 limitations documented with impact assessment |
| No lifecycle redesign | COMPLIANT -- existing event model, KV projection, and write path unchanged |

## Residual Gaps

| # | Gap | Severity | Mitigation |
|---|-----|----------|-----------|
| G-1 | LifecycleListQuery (S413) not yet exposed via gateway HTTP | LOW | Available via NATS; can be wired in S454A |
| G-2 | No rejection_code filtering in ClickHouse query | LOW | Client-side filter from metadata; dedicated columns possible in future stage |
| G-3 | No cross-partition lifecycle query | LOW | Use LifecycleListQuery for key enumeration, then per-key lifecycle history |

## Test Results

```
--- internal/adapters/clickhouse ---
PASS: TestBuildLifecycleHistoryQuery_BasicFilters
PASS: TestBuildLifecycleHistoryQuery_NoTypeFilter
PASS: TestBuildLifecycleHistoryQuery_WithSide
PASS: TestBuildLifecycleHistoryQuery_WithStatus
PASS: TestBuildLifecycleHistoryQuery_WithSince
PASS: TestBuildLifecycleHistoryQuery_WithUntil
PASS: TestBuildLifecycleHistoryQuery_WithAllFilters
PASS: TestBuildLifecycleHistoryQuery_SelectColumns
PASS: TestBuildLifecycleHistoryQuery_TimeframeAsUint32

--- internal/application/analyticalclient ---
PASS: TestGetLifecycleHistoryUseCase_HappyPath
PASS: TestGetLifecycleHistoryUseCase_EmptyResult
PASS: TestGetLifecycleHistoryUseCase_ReaderError
PASS: TestGetLifecycleHistoryUseCase_MissingSource
PASS: TestGetLifecycleHistoryUseCase_MissingSymbol
PASS: TestGetLifecycleHistoryUseCase_InvalidTimeframe
PASS: TestGetLifecycleHistoryUseCase_InvalidTimeRange
PASS: TestGetLifecycleHistoryUseCase_LimitClamping
PASS: TestGetLifecycleHistoryUseCase_NilUseCase
PASS: TestGetLifecycleHistoryUseCase_EntryTimestampFormat
```

Zero test regressions across all modified packages (clickhouse, analyticalclient, handlers, routes).

## Next Stage Direction

S454A should consider:
1. Wiring `LifecycleListQuery` to the gateway HTTP surface (bridging KV enumeration to HTTP)
2. Adding lifecycle summary/aggregation queries (rejection rate, fill latency distribution)
3. Cross-partition lifecycle listing for operational dashboards
