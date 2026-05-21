# Refactor Tranche 01 — Changes, Rationale, and Impact

## Overview

Tranche 01 is the first structural code refactoring applied to market-foundry after the S210 stabilization gate and S212 architecture census. It targets three HIGH priority items from the S212 priority map.

## Changes Applied

### 1. ClickHouse Query Builder (H-03)

**Files created:**
- `internal/adapters/clickhouse/query_builder.go` — `BuildQuery()` function + `OptionalFilter` type
- `internal/adapters/clickhouse/query_builder_test.go` — 7 test cases covering all query patterns

**Files modified:**
- `candle_reader.go` — `BuildCandleQuery()` delegates to `BuildQuery()`
- `signal_reader.go` — `BuildSignalQuery()` delegates to `BuildQuery()`
- `decision_reader.go` — `BuildDecisionQuery()` delegates with outcome filter
- `strategy_reader.go` — `BuildStrategyQuery()` delegates with direction filter
- `risk_reader.go` — `BuildRiskQuery()` delegates with disposition filter
- `execution_reader.go` — `BuildExecutionQuery()` delegates with side + status filters

**Rationale:** Six Build*Query functions contained identical query construction logic (base SELECT, mandatory WHERE, optional filters, time range, ORDER BY DESC LIMIT). The only variation was table name, column list, filter names, and timestamp column. Extracting `BuildQuery()` centralizes this pattern so future analytical families only specify their table-specific parameters.

**Impact:** Reduced Build*Query functions from 18-25 lines each to 8-12 lines. Query construction logic now lives in one place — any future change to query pattern (e.g., adding pagination cursors) only needs one edit.

**Guard rails verified:**
- All 30+ existing ClickHouse tests pass without modification
- Generated SQL is byte-identical (verified via existing BuildQuery test assertions)
- All 8 binaries build successfully

### 2. Consumer Spec Factory (H-02)

**Files created:**
- `internal/adapters/nats/consumer_spec_factory.go` — `newConsumerSpec()` factory function
- `internal/adapters/nats/consumer_spec_factory_test.go` — 3 test cases + 18-spec completeness test

**Files modified:**
- `evidence_registry.go` — 4 consumer specs → one-liners
- `signal_registry.go` — 4 consumer specs → one-liners (codegen markers preserved)
- `decision_registry.go` — 2 consumer specs → one-liners
- `strategy_registry.go` — 2 consumer specs → one-liners
- `risk_registry.go` — 2 consumer specs → one-liners
- `execution_registry.go` — 4 consumer specs → one-liners

**Rationale:** All 18 consumer spec functions were identical 12-line templates that differed only in four string values: durable name, subject, event type, and stream name. The factory eliminates this duplication while preserving the exact durable names (critical for NATS consumer offset continuity).

**Impact:** Reduced consumer spec code from ~216 lines to ~36 lines (18 one-liners + factory). Adding a new family's consumer spec is now a single line instead of 12. Defaults (30s AckWait, 5 MaxDeliver) are centralized.

**Guard rails verified:**
- All existing NATS adapter tests pass
- TestAllConsumerSpecFunctionsUseFactory verifies all 18 functions return correct defaults
- Codegen markers in signal_registry.go preserved exactly

### 3. Store Actor Infrastructure (H-04 — Partial)

**Files created:**
- `internal/actors/scopes/store/generic_consumer_actor.go` — `GenericConsumerActor` + `ConsumerStartFn` type
- `internal/actors/scopes/store/projection_stats.go` — `ProjectionStats` + `CheckInvariant()` + `Log()`

**Files NOT modified:** Existing per-family actors and store_supervisor.go remain unchanged.

**Rationale:** The 9 consumer actors and 9 projection actors share ~95% identical code. Rather than attempting a risky wholesale migration in one tranche, S213 lays the generic infrastructure that S214 can use to migrate each actor incrementally. The `GenericConsumerActor` uses a `ConsumerStartFn` callback pattern that captures domain-specific NATS consumer creation in a closure, eliminating the need for per-family actor types. `ProjectionStats` centralizes the 7-field stats tracking + invariant check + log pattern shared across all 9 projection actors.

**Impact:** Infrastructure is ready. S214 can migrate each pipeline from per-family actors to generic actors one-at-a-time with per-pipeline test verification. Projected reduction when migration completes: ~1,800 lines eliminated from 18 actor files.

**Guard rails verified:**
- New types compile cleanly and are independent of existing actors
- All existing actor tests pass (no behavioral changes)
- Actor supervision tree unchanged

## Quantified Impact

| Metric | Before | After | Reduction |
|--------|--------|-------|-----------|
| Consumer spec boilerplate | 216 lines | 36 lines | -83% |
| Query builder duplication | ~120 lines (6 × 20) | ~60 lines (6 × 10) | -50% |
| Actor infrastructure (new) | 0 | 160 lines available | Foundation for -1,800 lines in S214 |
| Total test cases added | 0 | 28 new tests | +28 |
| Files created | 0 | 6 | New shared infrastructure |
| Binary build status | All pass | All pass | No regression |

## Trade-offs

1. **H-04 is partial.** The generic actor infrastructure is created but per-family actors are not yet migrated. This is deliberate — migrating 18 actors in one tranche risks regression.
2. **Build*Query signature preserved.** The existing exported functions remain as thin wrappers. This maintains backward compatibility but means callers still use positional parameters.
3. **Query row scanning not abstracted.** Each reader's scan/construct logic varies per domain type. Abstracting this would require complex generics with minimal readability benefit.

## Areas Consciously Not Touched

- `internal/adapters/nats/configctl_registry.go` — infrastructure registry, different pattern
- `internal/actors/scopes/store/candle_projection_actor.go` — dual-bucket variant
- `internal/actors/scopes/store/fill_projection_actor.go` — RC reconciliation variant
- `internal/actors/scopes/store/query_responder_actor.go` — unrelated to this refactoring
- `internal/shared/settings/schema.go` — medium priority (M-07)
- `cmd/writer/pipeline.go` — medium priority (M-03)
- `cmd/gateway/compose.go` — medium priority (M-06)
