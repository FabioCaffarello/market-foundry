# Stage S213 — Strategic Runtime and Package Refactor Report

## Status: COMPLETE

## Objective

Execute the highest-value tranche of structural refactoring identified by S212, targeting runtimes, packages, and boundaries to reduce evolution cost and family-expansion blast radius.

## Executive Summary

S213 applied three HIGH priority refactoring items from the S212 priority map:

1. **H-03: ClickHouse Query Builder** — Extracted `BuildQuery()` centralizing query construction across 6 analytical readers.
2. **H-02: Consumer Spec Factory** — Extracted `newConsumerSpec()` reducing 18 consumer spec functions from 12 lines each to one-liners.
3. **H-04: Store Actor Infrastructure** — Created `GenericConsumerActor` and `ProjectionStats` as shared building blocks for future actor migration.

All changes are pure refactoring — zero behavioral changes. All tests pass. All 8 binaries build.

## Tranche Applied

### H-03: ClickHouse Query Builder Consolidation

| Aspect | Detail |
|--------|--------|
| Files created | `query_builder.go`, `query_builder_test.go` |
| Files modified | 6 reader files (Build*Query functions) |
| Lines reduced | ~60 lines (query construction duplication) |
| Tests added | 7 test cases for BuildQuery |
| Risk | Low — exported signatures preserved, SQL output identical |

### H-02: NATS Consumer Spec Factory

| Aspect | Detail |
|--------|--------|
| Files created | `consumer_spec_factory.go`, `consumer_spec_factory_test.go` |
| Files modified | 6 registry files (18 consumer spec functions) |
| Lines reduced | ~180 lines (consumer spec boilerplate) |
| Tests added | 21 test cases (factory + 18-spec completeness) |
| Risk | Low — function signatures preserved, durable names exact |

### H-04: Store Actor Generic Infrastructure (Partial)

| Aspect | Detail |
|--------|--------|
| Files created | `generic_consumer_actor.go`, `projection_stats.go` |
| Files modified | None (infrastructure only, migration deferred to S214) |
| Lines added | ~160 lines of reusable infrastructure |
| Projected S214 reduction | ~1,800 lines when per-family actors are migrated |
| Risk | None — new code is additive, existing actors unchanged |

## Files Changed

### Created (6 files)
- `internal/adapters/clickhouse/query_builder.go`
- `internal/adapters/clickhouse/query_builder_test.go`
- `internal/adapters/nats/consumer_spec_factory.go`
- `internal/adapters/nats/consumer_spec_factory_test.go`
- `internal/actors/scopes/store/generic_consumer_actor.go`
- `internal/actors/scopes/store/projection_stats.go`

### Modified (12 files)
- `internal/adapters/clickhouse/candle_reader.go` — BuildCandleQuery delegates to BuildQuery
- `internal/adapters/clickhouse/signal_reader.go` — BuildSignalQuery delegates to BuildQuery
- `internal/adapters/clickhouse/decision_reader.go` — BuildDecisionQuery delegates to BuildQuery
- `internal/adapters/clickhouse/strategy_reader.go` — BuildStrategyQuery delegates to BuildQuery
- `internal/adapters/clickhouse/risk_reader.go` — BuildRiskQuery delegates to BuildQuery
- `internal/adapters/clickhouse/execution_reader.go` — BuildExecutionQuery delegates to BuildQuery
- `internal/adapters/nats/evidence_registry.go` — consumer specs use factory
- `internal/adapters/nats/signal_registry.go` — consumer specs use factory (codegen markers preserved)
- `internal/adapters/nats/decision_registry.go` — consumer specs use factory
- `internal/adapters/nats/strategy_registry.go` — consumer specs use factory
- `internal/adapters/nats/risk_registry.go` — consumer specs use factory
- `internal/adapters/nats/execution_registry.go` — consumer specs use factory

### Documentation (3 files)
- `docs/architecture/strategic-runtime-and-package-refactor.md`
- `docs/architecture/refactor-tranche-01-changes-rationale-and-impact.md`
- `docs/stages/stage-s213-strategic-runtime-and-package-refactor-report.md`

## Verification

| Check | Result |
|-------|--------|
| `go test internal/adapters/clickhouse/...` | PASS |
| `go test internal/adapters/nats/...` | PASS |
| `go test internal/actors/...` | PASS |
| `go test` (all modules) | PASS |
| `go build` (all 8 binaries) | PASS |
| `go vet` (modified packages) | PASS |

## Architectural Gains

1. **Consumer spec centralization.** AckWait/MaxDeliver defaults are now in one place. Adding a new family's consumer spec is a single line.
2. **Query construction centralization.** The time-range + optional-filter + ORDER BY pattern is defined once. Future changes (pagination, query hints) need one edit.
3. **Actor infrastructure readiness.** GenericConsumerActor and ProjectionStats are available for S214 to eliminate ~1,800 lines of per-family actor duplication.
4. **Test coverage increased.** 28 new test cases covering the shared infrastructure.

## Trade-offs and Limits

1. **H-04 partial execution.** Per-family actors remain because migrating 18 actors in one tranche risks regression. S214 should migrate incrementally.
2. **Row scanning not abstracted.** Each ClickHouse reader's Scan/construct logic varies per domain type. Abstracting this requires complex generics for minimal gain.
3. **Supervisor unchanged.** The store_supervisor.go pipeline declarations still reference per-family actor constructors.

## Preparation for S214

S214 should:
1. Migrate per-family consumer actors to use `GenericConsumerActor` (one pipeline at a time, verify tests after each)
2. Migrate per-family projection actors to use `ProjectionStats` (reduces ~35 lines per actor)
3. Consider whether GenericProjectionActor is warranted or if ProjectionStats alone is sufficient
4. Evaluate remaining MEDIUM priority items (M-01 through M-07) for inclusion

## Success Criteria Assessment

| Criterion | Met? |
|-----------|------|
| Highest-value tranche executed | Yes — H-02, H-03, H-04 (partial) |
| Runtimes/packages/boundaries clearer | Yes — query, spec, and actor patterns centralized |
| Relevant couplings reduced | Yes — consumer specs, query construction |
| Real gain, not aesthetic rearrangement | Yes — measurable line reduction + single-point-of-change |
| Base ready for S214 consolidation | Yes — generic infrastructure available |
| No new features opened | Yes |
| No operational breakage | Yes — all tests pass, all binaries build |
