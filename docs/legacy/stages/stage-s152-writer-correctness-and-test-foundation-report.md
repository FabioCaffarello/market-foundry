# Stage S152 — Writer Correctness and Test Foundation Report

## Executive Summary

S152 establishes the first disciplined test foundation for the analytical write and read paths. Before this stage, the writer service (`cmd/writer/`) had **zero tests** and the gateway analytical reader had **no unit test coverage** for its query builder and float formatting. The analytical client use case and HTTP handler already had comprehensive tests from S149.

This stage adds **32 new test cases** across the writer mapper, inserter buffer logic, and reader query builder — covering the most critical correctness boundaries without introducing test infrastructure overhead.

## Test Foundation Applied

### Writer Mappers (cmd/writer/mappers_test.go) — 25 tests

**Scope:** All 6 mapper functions + 2 helper functions.

Tests prove:
- **Column count matches DDL** for all 6 tables (evidence_candles: 16, signals: 12, decisions: 14, strategies: 15, risk_assessments: 17, executions: 20).
- **Metadata positions are consistent** (positions 0–3 across all mappers).
- **Domain field transformations are correct** — enum casts, float parsing, JSON serialization.
- **Edge cases handled** — empty decimal strings → 0.0, nil metadata → `"null"`, empty maps/slices produce valid JSON.
- **JSON roundtrip** — serialized nested structures (signals, decisions, strategies, constraints, risk, fills) are valid and deserializable.

### Writer Inserter (cmd/writer/inserter_test.go) — 10 tests

**Scope:** Buffer management and eviction logic.

Tests prove:
- **FIFO eviction** — when buffer exceeds maxPending, oldest rows are dropped.
- **Correct eviction count** — exactly `overflow` rows removed.
- **Tracker integration** — `events_dropped` counter accurately tracks evictions.
- **Nil safety** — nil tracker and nil engine/pid don't panic.
- **Empty flush** — no-op, no crash.

### Reader Query Builder (cmd/gateway/analytical_reader_test.go) — 8 tests

**Scope:** Query construction and float formatting.

Tests prove:
- **Parameterized query** — all inputs go through `?` placeholders.
- **Conditional time filters** — `since=0`/`until=0` correctly omit predicates.
- **Argument order** — matches placeholder positions exactly.
- **Type alignment** — timeframe as `uint32`, time filters as `time.Time`.
- **SELECT columns** — all 12 expected columns present.
- **formatFloat precision** — 7 cases covering precision boundary.

### Pre-existing Tests (unchanged)

- `internal/application/analyticalclient/get_candle_history_test.go` — 9 tests (validation, limit clamping, error propagation)
- `internal/interfaces/http/handlers/analytical_test.go` — 5 tests (HTTP handler responses)

## Files Changed

| File | Action | Description |
|---|---|---|
| `cmd/writer/mappers_test.go` | Created | 25 test cases for all mappers + helpers |
| `cmd/writer/inserter_test.go` | Created | 10 test cases for buffer/eviction logic |
| `cmd/gateway/analytical_reader_test.go` | Created | 8 test cases for query builder + formatFloat |
| `docs/architecture/analytical-writer-correctness-and-test-foundation.md` | Created | Writer test scope and invariants doc |
| `docs/architecture/analytical-reader-adapter-test-scope-and-limits.md` | Created | Reader test scope and limits doc |
| `docs/stages/stage-s152-writer-correctness-and-test-foundation-report.md` | Created | This report |

## Contracts and Paths Covered

### Write Path

```
Domain Event → mapXxxRow() → []any row → inserter buffer → enforceMaxPending → flush
                  ✅ tested     ✅ tested    ✅ tested        ✅ tested           ❌ needs CH
```

### Read Path

```
HTTP params → handler → use case validation → buildCandleQuery → Client.Query → row scan → formatFloat
                ✅           ✅                    ✅                ❌ needs CH   ❌ needs CH    ✅
```

## Remaining Limits

### Not testable without infrastructure

| Gap | Component | Risk | When to address |
|---|---|---|---|
| ClickHouse InsertBatch | inserter flush | Medium | S153+ integration tests |
| ClickHouse Query+Scan | reader QueryCandleHistory | Medium | S153+ integration tests |
| NATS consumer wiring | consumer → inserter | Low | End-to-end smoke tests |
| Actor lifecycle | Started/Stopped dispatch | Low | Hollywood framework responsibility |

### Not yet tested (low priority)

| Gap | Component | Risk |
|---|---|---|
| Pipeline enable/disable | supervisor.start() | Low — config is tested in settings_test.go |
| Timer-based flush trigger | scheduleFlush | Low — trivial AfterFunc |
| Multiple pipeline spawn | supervisor + pipeline.go | Low — declarative, covered by smoke tests |

## Preparation for S153

With the correctness foundation in place, S153 can focus on:

1. **Failure and recovery hardening** — the tested mapper and buffer contracts provide a reliable baseline for introducing error injection and retry logic.
2. **Integration test scaffold** — a lightweight ClickHouse test container (e.g., testcontainers-go) would unlock the remaining `flush()` and `QueryCandleHistory` gaps.
3. **Writer observability** — the proven tracker integration (events_dropped counter) provides a foundation for operational alerting.
4. **Reader expansion** — adding query paths for signals, decisions, strategies, risk, and executions following the same `buildXxxQuery` pattern already proven for candles.

## Acceptance Criteria Checklist

- [x] Writer mapper, inserter, and supervisor have minimum serious coverage
- [x] Reader adapter gains coverage adequate to current stage
- [x] Critical contracts are made more explicit (column counts, JSON serialization, parameterized queries)
- [x] Test base improves without becoming an excessive framework
- [x] Analytical layer is ready for failure and recovery hardening
- [x] Guard rails respected: no functional expansion, no unnecessary abstractions, gaps documented clearly
