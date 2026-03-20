# Stage S175 — Family 03 Definition and Analytical Contract Report

**Stage**: S175
**Status**: Complete
**Date**: 2026-03-19
**Predecessor**: S174 (Family 03 Selection and Responsibility Fit Review)
**Successor**: S176 (Family 03 Minimal Implementation)

---

## 1. Executive summary

Stage S175 formally defines Family 03 — **Strategies (`mean_reversion_entry`)** — as the next controlled expansion of the analytical layer. The stage transforms the S174 selection into a precise, implementable specification covering the analytical contract, payload shape, schema mapping, reader/writer boundaries, gateway integration, and success criteria.

The write path (mapper + pipeline entry) already exists and is active. Family 03 requires only the read path: one reader adapter, one use case, one HTTP handler, one route, and test extensions. This is the same 6-artifact read-path pattern proven by Family 01 (signals) and Family 02 (decisions).

**Key complexity increment**: Family 03 introduces 3 JSON columns (up from 2 in decisions) and a second enum-like optional filter (`direction`, analogous to `outcome`). This tests whether the JSON scanning pattern and optional-filter pattern scale without architectural pressure.

---

## 2. What was defined

### 2.1 Analytical contract

- **Query contract**: `StrategyHistoryQuery` with 4 required params (type, source, symbol, timeframe) and 4 optional params (direction, limit, since, until)
- **Response contract**: `StrategyHistoryReply` with `[]strategy.Strategy`, source, and query metadata
- **HTTP endpoint**: `GET /analytical/strategy/history`
- **Ordering**: Newest-first (DESC by timestamp)
- **Limits**: Default 50, max 500

### 2.2 Payload and schema

- **Domain type**: `strategy.Strategy` (11 domain fields)
- **DDL**: Migration 004 (15 columns including envelope metadata) — already applied
- **JSON columns**: `decisions` (`[]DecisionInput`), `parameters` (`map[string]string`), `metadata` (`map[string]string`)
- **Schema coherence**: All 11 domain columns verified type-aligned across DDL, writer mapper, and planned reader adapter

### 2.3 Boundaries

- **Writer**: Zero changes. `mapStrategyRow()` and `mean_reversion_entry` pipeline entry exist.
- **Reader**: New `StrategyReader` adapter with `QueryStrategyHistory()`, `BuildStrategyQuery()`, and `ParseDecisionInputsJSON()`.
- **Gateway**: New handler method, route, composition wiring. Follows struct DI pattern (H-1 already complete).
- **Observability**: Inserter counters (existing) + reader adapter logs + Server-Timing header.

### 2.4 Success criteria

- 7 schema coherence criteria
- 7 read path criteria
- 6 application layer criteria
- 6 HTTP surface criteria
- 6 integration criteria
- 6 boundary preservation criteria
- Total: **38 pass/fail criteria**

---

## 3. Artifacts produced

| # | Artifact                                                                     | Purpose                                      |
|---|----------------------------------------------------------------------------- |--------------------------------------------- |
| 1 | `docs/architecture/family-03-definition-and-analytical-contract.md`          | Analytical contract, payload, schema mapping  |
| 2 | `docs/architecture/family-03-schema-writer-reader-gateway-contract.md`       | 9-artifact inventory, implementation spec     |
| 3 | `docs/architecture/family-03-success-criteria-and-operability-scope.md`      | Pass/fail criteria, non-goals, runbook scope  |
| 4 | `docs/stages/stage-s175-family-03-definition-and-analytical-contract-report.md` | This report                                |

---

## 4. Pre-staged artifacts (exist, require zero changes)

| Artifact            | File                                          | Verified |
|-------------------- |---------------------------------------------- |--------- |
| Schema migration    | `deploy/migrations/004_create_strategies.sql` | ✓        |
| Writer mapper       | `cmd/writer/mappers.go` → `mapStrategyRow()`  | ✓        |
| Pipeline entry      | `cmd/writer/pipeline.go` → line 148–170       | ✓        |

---

## 5. Pattern observations

### 5.1 What scales well

- **9-artifact pattern**: Third application. Pattern is mechanical and predictable.
- **Struct DI (H-1)**: `AnalyticalHandlerDeps` and `AnalyticalFamilyDeps` accept a new field without constructor signature changes.
- **Optional filter pattern**: `direction` mirrors `outcome` exactly. Query builder appends clause only when non-empty.
- **JSON parser pattern**: `ParseDecisionInputsJSON` mirrors `ParseSignalInputsJSON`. Array-of-structs deserialization is a proven shape.
- **Reusable parsers**: `ParseMetadataJSON` reused for both `parameters` and `metadata` columns (2 of 3 JSON columns need no new code).

### 5.2 What to watch

- **JSON column count at 3**: This is the highest in the analytical surface. If 3 columns scan/parse cleanly, Family 04 (risk assessments, 4 JSON columns) is viable.
- **Constructor accumulation**: `buildRouteDependencies` in compose.go now wires 4 analytical families. Manageable but approaching the point where mechanical repetition is noticeable.
- **Test case count**: 12 use case tests + 8 reader tests + handler tests. Total test count for the analytical layer continues to grow linearly.

### 5.3 Complexity gradient verification

| Family | Layer | Domain columns | JSON columns | Optional filter | New parser needed |
|--------|-------|----------------|--------------|-----------------|-------------------|
| 00 (candles)    | 1 | 12 | 0 | none      | none                    |
| 01 (signals)    | 2 | 8  | 1 | none      | `ParseMetadataJSON`     |
| 02 (decisions)  | 3 | 10 | 2 | `outcome` | `ParseSignalInputsJSON` |
| **03 (strategies)** | **4** | **11** | **3** | **`direction`** | **`ParseDecisionInputsJSON`** |
| 04 (risk) *planned* | 5 | 13 | 4 | `disposition` | `ParseStrategyInputsJSON` + `ParseConstraintsJSON` |

Monotonic complexity gradient is maintained.

---

## 6. Explicit limits and non-goals

- No direction aggregation, confidence filtering, or decision drill-down
- No cross-family queries
- No writer changes
- No smoke parameterization (H-2) or naming cleanup (H-3) — tracked as deferred hardening
- No Prometheus/OpenTelemetry
- No pagination beyond 500
- No custom TTL

---

## 7. S176 preparation

S176 implementation scope is fully defined:

1. **Reader adapter**: `strategy_reader.go` + `strategy_reader_test.go` (8 test cases)
2. **Use case**: `get_strategy_history.go` + `get_strategy_history_test.go` (12 test cases)
3. **Contracts**: Add `StrategyHistoryQuery` + `StrategyHistoryReply` to `contracts.go`
4. **Handler**: Extend `analytical.go` with `GetStrategyHistory` method
5. **Routes**: Extend `analytical.go` routes with strategy endpoint
6. **Composition**: Add `newAnalyticalStrategyReader` + wire in `buildRouteDependencies`
7. **Tests**: Extend `tests/http/analytical.http` and `scripts/smoke-analytical-e2e.sh`

**Estimated artifact count**: 6 new files/modifications, ~20 test cases, 1 new JSON parser.

No ambiguity remains. The S176 implementation is mechanical and bounded.

---

## 8. Decisions log

| Decision                                                  | Rationale                                                    |
|---------------------------------------------------------- |------------------------------------------------------------- |
| Direction filter is NOT validated at query level           | Matches outcome pattern; invalid values return empty results |
| `ParseDecisionInputsJSON` placed in `strategy_reader.go`  | Follows `ParseSignalInputsJSON` placement in `decision_reader.go` |
| `parameters` and `metadata` both use `ParseMetadataJSON`  | Same `map[string]string` shape; no new parser needed         |
| No generic reader abstraction                             | 4 families is not enough to justify abstraction cost         |
| Confidence returned as formatted string, not filterable   | Consistent with candle/signal/decision behavior              |
