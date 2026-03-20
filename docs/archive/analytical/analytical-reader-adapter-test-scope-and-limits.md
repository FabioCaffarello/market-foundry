# Analytical Reader Adapter: Test Scope and Limits

## Purpose

This document defines the test scope for the analytical read path (ClickHouse → domain → HTTP) after S152.

## Read Path Architecture

```
HTTP GET /analytical/evidence/candles
  → AnalyticalWebHandler.GetCandleHistory()     [handlers/analytical.go]
    → GetCandleHistoryUseCase.Execute()          [analyticalclient/get_candle_history.go]
      → analyticalCandleReader.QueryCandleHistory() [gateway/analytical_reader.go]
        → Client.Query()                         [adapters/clickhouse/reader.go]
          → evidence_candles table
```

## Test Coverage by Layer

### Layer 1: Use Case Validation (analyticalclient)

**File:** `internal/application/analyticalclient/get_candle_history_test.go`
**Status:** Pre-existing, comprehensive.

| Test | What it proves |
|---|---|
| MissingSource | Rejects empty source |
| MissingSymbol | Rejects empty symbol |
| InvalidTimeframe | Rejects zero/negative timeframe |
| SinceAfterUntil | Rejects since > until |
| DefaultLimit | Applies default limit of 50 |
| LimitClamped | Clamps limit to [1, 500] |
| ReaderError | Propagates reader errors |
| NilReader | Returns unavailable for nil reader |
| NilUseCaseExecute | Returns unavailable for nil use case |

### Layer 2: HTTP Handler (interfaces/http/handlers)

**File:** `internal/interfaces/http/handlers/analytical_test.go`
**Status:** Pre-existing, covers happy path and error responses.

| Test | What it proves |
|---|---|
| Happy path | Full round-trip with mock candles, correct JSON shape |
| Missing timeframe | Returns 400 |
| Limit out of bounds | Returns 400 |
| Nil handler | Returns 503 |
| Use case errors | Returns 503 |

### Layer 3: Query Builder + Float Formatting (gateway)

**File:** `cmd/gateway/analytical_reader_test.go`
**Status:** New in S152.

| Test | What it proves |
|---|---|
| BasicFilters | Base WHERE clause structure, 4 args |
| NoTimeFilters | No open_time clauses when since/until are 0 |
| WithSince | Adds open_time >= ?, correct time.Unix conversion |
| WithUntil | Adds open_time <= ?, correct time.Unix conversion |
| WithSinceAndUntil | Both time filters, 6 args, correct order |
| TimeframeAsUint32 | Timeframe passed as uint32 (matches DDL type) |
| SelectColumns | All 12 expected columns present in SELECT |
| FormatFloat | 7 cases: zero, negative, small, large, trailing precision |

## Critical Contracts Proven

1. **Query parameterization.** All user inputs go through `?` placeholders — no SQL injection surface.
2. **Conditional time filters.** `since=0` and `until=0` correctly omit time predicates.
3. **Argument order matches placeholders.** Base args (source, symbol, timeframe) are always first; time filters follow conditionally; limit is always last.
4. **Type alignment with DDL.** Timeframe is `uint32`, time filters are `time.Time` — matching ClickHouse column types.
5. **Float formatting.** `formatFloat` uses `strconv.FormatFloat(f, 'f', -1, 64)` for maximum precision without scientific notation.

## What Remains Outside Coverage

| Component | Why | Risk | Mitigation |
|---|---|---|---|
| `QueryCandleHistory` full path | Requires live ClickHouse | Medium | Smoke tests validate end-to-end |
| Row scanning (Scan call) | Requires real result rows | Medium | Type alignment tested via DDL review |
| `float64 → string` round-trip | Writer parseFloat + reader formatFloat | Low | Both individually tested; round-trip fidelity acceptable for analytical use |
| Route registration | Conditional on ClickHouse availability | Low | Integration smoke tests |
| Connection errors / retries | Infrastructure-dependent | Low | ClickHouse client handles reconnection |

## Round-Trip Fidelity Note

The write path converts decimal strings to float64 via `parseFloat`, and the read path converts float64 back to decimal strings via `formatFloat`. This round-trip is **not** lossless for all inputs (IEEE 754 limits apply). For the analytical layer, this is acceptable: the precision loss is bounded and well-understood. Production use cases that require exact decimal fidelity should use the operational (NATS KV) path, not the analytical (ClickHouse) path.

Example:
- Write: `"0.1"` → `parseFloat` → `0.1` (float64)
- Read: `0.1` → `formatFloat` → `"0.1"` (string)
- For most practical values, the round-trip is faithful.
