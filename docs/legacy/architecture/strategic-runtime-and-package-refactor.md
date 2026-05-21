# Strategic Runtime and Package Refactor

## Purpose

This document captures the architectural rationale behind the S213 refactoring tranche — the first structural code refactor applied to the market-foundry repository after the S210 stabilization gate and the S212 architecture census.

## Guiding Principles

1. **Attack highest-value items only.** The S212 census identified ~6,375 recoverable lines across 10 structural clusters. S213 targets only the HIGH priority items that directly reduce family-expansion blast radius.
2. **No new features.** All changes are pure refactoring — identical behavior before and after.
3. **Preserve operational contracts.** NATS durable names, ClickHouse SQL queries, HTTP handler signatures, and actor supervision trees remain byte-identical.
4. **Prepare for S214.** The refactoring lays infrastructure (GenericConsumerActor, ProjectionStats, BuildQuery) that S214 can build on to complete the migration.

## Refactoring Strategy

### What S213 Executes

| ID | Item | Package | Approach |
|----|------|---------|----------|
| H-02 | Consumer Spec Factory | `internal/adapters/nats` | Extract `newConsumerSpec()` factory; all 18 consumer spec functions become one-liners |
| H-03 | ClickHouse Query Builder | `internal/adapters/clickhouse` | Extract `BuildQuery()` with `OptionalFilter`; all 6 `Build*Query` functions delegate to it |
| H-04 | Store Actor Infrastructure | `internal/actors/scopes/store` | Create `GenericConsumerActor` + `ProjectionStats` as shared building blocks |

### What S213 Consciously Defers

| ID | Item | Reason |
|----|------|--------|
| H-01 | NATS Adapter Sub-Packaging | Mechanical file moves — high effort, no duplication reduction |
| H-05 | Documentation Entropy | Pure documentation — valuable but not code architecture |
| H-06 | Module Consolidation | Requires careful evaluation of build isolation requirements |
| M-01..M-07 | Medium Priority Items | Depend on H-01..H-04 or have lower ROI per effort |

## Architectural Boundaries After Refactor

### ClickHouse Read Path

```
BuildQuery()                    ← single query construction engine
  ├── BuildCandleQuery()        ← delegates with open_time column
  ├── BuildSignalQuery()        ← delegates with timestamp column
  ├── BuildDecisionQuery()      ← adds outcome OptionalFilter
  ├── BuildStrategyQuery()      ← adds direction OptionalFilter
  ├── BuildRiskQuery()          ← adds disposition OptionalFilter
  └── BuildExecutionQuery()     ← adds side + status OptionalFilters
```

Each reader's Query method still handles row scanning independently (domain types differ too much for useful abstraction).

### NATS Consumer Specs

```
newConsumerSpec(durable, subject, type, stream)    ← single factory
  ├── Writer*Consumer()                            ← one-liner delegates
  └── Store*Consumer()                             ← one-liner delegates
```

All 18 consumer spec functions produce identical structure with identical defaults (30s AckWait, 5 MaxDeliver). Codegen markers preserved in signal_registry.go.

### Store Actor Infrastructure

```
GenericConsumerActor            ← callback-driven, family-agnostic
  └── ConsumerStartFn           ← captures registry + event routing in closure

ProjectionStats                 ← shared stats tracking + invariant check
  ├── CheckInvariant()          ← replaces 9 per-family checkStatsInvariant()
  └── Log()                     ← replaces 9 per-family logStats()
```

Per-family actors remain in this tranche. S214 can migrate them to use GenericConsumerActor and ProjectionStats, eliminating the per-family files entirely.

## Evolution Cost Impact

| Metric | Before S213 | After S213 | After S214 (projected) |
|--------|-------------|------------|------------------------|
| Consumer spec lines per family | 12 | 1 | 1 |
| Build*Query lines per family | 18-25 | 8-12 | 8-12 |
| Consumer actor files per family | 1 (90 lines) | 1 (90 lines) | 0 (uses generic) |
| Projection stats duplication | 9x ~35 lines | 9x ~35 lines | 0 (uses shared) |
| Family addition blast radius | 15+ files | 14 files | ~8 files (projected) |
