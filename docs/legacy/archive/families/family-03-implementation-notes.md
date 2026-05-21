# Family 03 — Implementation Notes

**Family**: Strategies (`mean_reversion_entry`)
**Stage**: S176
**Date**: 2026-03-19

---

## 1. Artifact inventory

### 1.1 New files created

| File | Purpose |
|------|---------|
| `internal/adapters/clickhouse/strategy_reader.go` | Reader adapter: `StrategyReader`, `BuildStrategyQuery`, `ParseDecisionInputsJSON` |
| `internal/adapters/clickhouse/strategy_reader_test.go` | 14 test cases: 8 query builder + 6 parser |
| `internal/application/analyticalclient/get_strategy_history.go` | Use case: `GetStrategyHistoryUseCase` with validation |
| `internal/application/analyticalclient/get_strategy_history_test.go` | 12 test cases: validation, degradation, happy path |

### 1.2 Files modified

| File | Change |
|------|--------|
| `internal/application/analyticalclient/contracts.go` | Added `StrategyHistoryQuery`, `StrategyHistoryReply`, `strategy` import |
| `internal/interfaces/http/handlers/analytical.go` | Added `getAnalyticalStrategyHistoryUseCase` interface, struct field, deps field, `GetStrategyHistory` handler method |
| `internal/interfaces/http/handlers/analytical_test.go` | Added 7 strategy handler tests: happy path, direction filter, missing type, missing timeframe, invalid limit, nil handler, use case error |
| `internal/interfaces/http/routes/analytical.go` | Added `GetStrategyHistory` to deps struct, `HasAny()`, interface, route registration |
| `cmd/gateway/analytical_reader.go` | Added `newAnalyticalStrategyReader` composition function |
| `cmd/gateway/compose.go` | Wired strategy reader + use case into `buildRouteDependencies` |
| `tests/http/analytical.http` | Added 8 strategy endpoint test cases (#24–#31) |

### 1.3 Pre-existing artifacts (zero changes)

| Artifact | File | Verified |
|----------|------|----------|
| Migration 004 | `deploy/migrations/004_create_strategies.sql` | No changes |
| Writer mapper | `cmd/writer/mappers.go` → `mapStrategyRow()` | No changes |
| Pipeline entry | `cmd/writer/pipeline.go` → `mean_reversion_entry` | No changes |

---

## 2. Implementation decisions

### 2.1 ParseDecisionInputsJSON placement

Placed in `strategy_reader.go` (not in a shared file), following the pattern established by `ParseSignalInputsJSON` in `decision_reader.go`. Each parser lives alongside the reader that uses it.

### 2.2 Direction filter — no validation

The `direction` parameter is passed directly to the SQL WHERE clause as a string equality check. Invalid direction values (e.g., "up") produce empty results rather than errors. This matches the `outcome` filter behavior in Family 02.

### 2.3 JSON column parsing

- `decisions` → `ParseDecisionInputsJSON` (new, in `strategy_reader.go`)
- `parameters` → `ParseMetadataJSON` (reused from `signal_reader.go`)
- `metadata` → `ParseMetadataJSON` (reused from `signal_reader.go`)

Two of three JSON columns needed no new code. Only the `[]DecisionInput` array required a new parser.

### 2.4 Struct DI — zero friction

Adding `GetStrategyHistory` to `AnalyticalHandlerDeps`, `AnalyticalWebHandler`, and `AnalyticalFamilyDeps` required only field additions. No constructor signature changes. The H-1 hardening (struct DI, completed in S172) continues to pay off.

---

## 3. Schema coherence verification

| # | DDL column | Writer value | Reader scan | Reader mapping | Aligned |
|---|------------|-------------|-------------|----------------|---------|
| 1 | `type` (LowCardinality String) | `s.Type` | `string` | direct | Yes |
| 2 | `source` (LowCardinality String) | `s.Source` | `string` | direct | Yes |
| 3 | `symbol` (LowCardinality String) | `s.Symbol` | `string` | direct | Yes |
| 4 | `timeframe` (UInt32) | `uint32(s.Timeframe)` | `uint32` | `int(tf)` | Yes |
| 5 | `direction` (LowCardinality String) | `string(s.Direction)` | `string` | `strategy.Direction(dir)` | Yes |
| 6 | `confidence` (Float64) | `parseFloat(s.Confidence)` | `float64` | `FormatFloat(confidence)` | Yes |
| 7 | `decisions` (String) | `marshalJSON(s.Decisions)` | `string` | `ParseDecisionInputsJSON` | Yes |
| 8 | `parameters` (String) | `marshalJSON(s.Parameters)` | `string` | `ParseMetadataJSON` | Yes |
| 9 | `metadata` (String) | `marshalJSON(s.Metadata)` | `string` | `ParseMetadataJSON` | Yes |
| 10 | `final` (Bool) | `s.Final` | `bool` | direct | Yes |
| 11 | `timestamp` (DateTime64) | `s.Timestamp` | `time.Time` | direct | Yes |

All 11 domain columns are type-aligned across DDL, writer, and reader.

---

## 4. Test summary

| Layer | File | Test count | All pass |
|-------|------|-----------|----------|
| Adapter | `strategy_reader_test.go` | 14 | Yes |
| Application | `get_strategy_history_test.go` | 12 | Yes |
| Handler | `analytical_test.go` (strategy section) | 7 | Yes |
| **Total** | | **33** | **Yes** |

---

## 5. Frictions observed

### F-01: Mechanical repetition in handler parameter parsing

The `GetStrategyHistory` handler method repeats the same limit/since/until parsing boilerplate as the three prior family handlers. This is the fourth copy. The pattern is stable and correct but verbose (~35 lines of identical parsing per handler).

**Assessment**: Not worth abstracting yet. The cost of a helper would introduce indirection for a pattern that is still evolving (each handler has a different set of optional filters). Revisit at Family 05 if the pattern remains stable.

### F-02: Constructor accumulation in compose.go

The `buildRouteDependencies` analytical block now creates 4 readers and 4 use cases. Each line is mechanical. This is manageable at 4 families but will become noticeable at 6.

**Assessment**: No action needed now. The struct DI pattern absorbs the growth cleanly. If Family 05 or 06 triggers discomfort, consider a loop-based wiring approach.

### F-03: No friction from 3 JSON columns

The three JSON columns parsed cleanly. Two reused `ParseMetadataJSON`, one required a new parser. The read path handled 11 scan variables without issues. **This validates that Family 04 (risk, with 4 JSON columns) is viable.**
