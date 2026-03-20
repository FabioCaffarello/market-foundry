# Stage S176 — Family 03 Minimal Implementation Report

**Stage**: S176
**Status**: Complete
**Date**: 2026-03-19
**Predecessor**: S175 (Family 03 Definition and Analytical Contract)
**Successor**: S177 (Family 03 End-to-End Validation)

---

## 1. Executive summary

Stage S176 implements Family 03 — **Strategies (`mean_reversion_entry`)** — as the third controlled expansion of the analytical layer under Wave B. The implementation follows the S175 specification exactly: one family, one endpoint, zero writer changes, mechanical application of the v2 pattern.

**Results**:
- 4 new files, 7 modified files
- 33 new test cases (all passing)
- 1 new HTTP endpoint: `GET /analytical/strategy/history`
- 1 new JSON parser: `ParseDecisionInputsJSON`
- Gateway compiles and all pre-existing tests pass
- Zero regressions

**Key validation**: 3 JSON columns scan and parse without friction, confirming that Family 04 (risk assessments, 4 JSON columns) is architecturally viable.

---

## 2. Family implemented

| Attribute | Value |
|-----------|-------|
| Family | Strategies |
| Type | `mean_reversion_entry` |
| Layer | 4 (evidence → signal → decision → **strategy**) |
| Table | `strategies` |
| Endpoint | `GET /analytical/strategy/history` |
| Domain columns | 11 |
| JSON columns | 3 (`decisions`, `parameters`, `metadata`) |
| Optional filter | `direction` (long, short, flat) |

---

## 3. Files changed

### New files (4)

| File | Lines | Purpose |
|------|-------|---------|
| `internal/adapters/clickhouse/strategy_reader.go` | ~135 | Reader adapter + query builder + JSON parser |
| `internal/adapters/clickhouse/strategy_reader_test.go` | ~175 | 14 test cases |
| `internal/application/analyticalclient/get_strategy_history.go` | ~90 | Use case with validation |
| `internal/application/analyticalclient/get_strategy_history_test.go` | ~175 | 12 test cases |

### Modified files (7)

| File | Change |
|------|--------|
| `internal/application/analyticalclient/contracts.go` | +`StrategyHistoryQuery`, +`StrategyHistoryReply`, +`strategy` import |
| `internal/interfaces/http/handlers/analytical.go` | +interface, +struct field, +deps field, +`GetStrategyHistory` method |
| `internal/interfaces/http/handlers/analytical_test.go` | +7 strategy handler tests, +`strategy` import |
| `internal/interfaces/http/routes/analytical.go` | +deps field, +interface, +`HasAny()` update, +route |
| `cmd/gateway/analytical_reader.go` | +`newAnalyticalStrategyReader` |
| `cmd/gateway/compose.go` | +strategy reader + use case wiring |
| `tests/http/analytical.http` | +8 strategy HTTP test cases (#24–#31) |

### Unchanged pre-existing artifacts (3)

| Artifact | File | Status |
|----------|------|--------|
| Migration 004 | `deploy/migrations/004_create_strategies.sql` | Untouched |
| Writer mapper | `cmd/writer/mappers.go` → `mapStrategyRow()` | Untouched |
| Pipeline entry | `cmd/writer/pipeline.go` → `mean_reversion_entry` | Untouched |

---

## 4. Test results

| Layer | Tests | Status |
|-------|-------|--------|
| `internal/adapters/clickhouse` | 14 new + existing | All pass |
| `internal/application/analyticalclient` | 12 new + existing | All pass |
| `internal/interfaces/http/handlers` | 7 new + existing | All pass |
| Gateway build | `go build ./cmd/gateway/...` | Clean |
| **Total new tests** | **33** | **All pass** |

---

## 5. Simplifications adopted

| ID | Simplification | Rationale |
|----|---------------|-----------|
| S-01 | Direction filter not validated | Matches outcome pattern in Family 02; invalid values return empty results |
| S-02 | No handler parameter parsing abstraction | Fourth copy of boilerplate; pattern still evolving per family |
| S-03 | ParseDecisionInputsJSON in strategy_reader.go | Follows established placement convention (parser lives with its reader) |
| S-04 | Parameters and metadata both use ParseMetadataJSON | Same `map[string]string` shape; no new parser needed |

---

## 6. Frictions observed

| ID | Friction | Severity | Action |
|----|----------|----------|--------|
| F-01 | Handler parameter parsing boilerplate (4th copy) | Low | Monitor; consider helper at Family 05 |
| F-02 | Constructor accumulation in compose.go (4 readers, 4 use cases) | Low | Monitor; struct DI absorbs growth |
| F-03 | 3 JSON columns → zero friction | None | **Validates Family 04 viability** |

**No blocking frictions. No stop conditions triggered.**

---

## 7. Limits maintained

- Exactly one family implemented (strategies)
- Exactly one endpoint added (`GET /analytical/strategy/history`)
- Zero writer changes
- Zero migration changes
- Zero cross-family queries
- Wave B v2 pattern followed mechanically
- No new abstractions introduced
- No Prometheus/OpenTelemetry
- No pagination beyond 500

---

## 8. Complexity gradient (updated)

| Family | Layer | Domain cols | JSON cols | Optional filter | New parser |
|--------|-------|------------|-----------|-----------------|------------|
| 00 (candles) | 1 | 12 | 0 | none | none |
| 01 (signals) | 2 | 8 | 1 | none | `ParseMetadataJSON` |
| 02 (decisions) | 3 | 10 | 2 | `outcome` | `ParseSignalInputsJSON` |
| **03 (strategies)** | **4** | **11** | **3** | **`direction`** | **`ParseDecisionInputsJSON`** |

Monotonic complexity gradient maintained. Each family adds exactly one JSON column and tests one incremental pattern.

---

## 9. S177 preparation

S177 should validate Family 03 end-to-end:

1. **Live pipeline test**: Verify strategy events flow from NATS through writer to ClickHouse
2. **Endpoint validation**: Query `GET /analytical/strategy/history` against live data
3. **Response shape verification**: Confirm `strategies` array, `source`, `meta` fields
4. **Direction filter test**: Verify filtering by `long`, `short`, `flat`
5. **JSON round-trip**: Verify `decisions`, `parameters`, `metadata` survive write→read cycle
6. **Non-regression**: All candle, signal, decision endpoints still work
7. **Smoke extension**: Add strategy phase to `smoke-analytical-e2e.sh`

The implementation is complete and ready for end-to-end validation.

---

## 10. Decisions log

| Decision | Rationale |
|----------|-----------|
| No shared parameter parsing helper | Pattern varies per family (candles: no type; signals/decisions/strategies: type + optional filter). Abstraction cost exceeds benefit at 4 families. |
| ParseDecisionInputsJSON in strategy_reader.go, not shared | Follows convention: ParseSignalInputsJSON is in decision_reader.go, ParseMetadataJSON is in signal_reader.go. Each parser lives with its first consumer. |
| No refactoring of existing code | S176 scope is additive-only. Existing artifacts verified but not touched. |
| Strategy import added to handler test file | Required for constructing test fixtures with `strategy.DirectionLong` and `strategy.DecisionInput`. |
