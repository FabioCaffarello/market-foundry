# Wave B Family-01 Implementation Notes — Signals (RSI)

## Family Selection

**Chosen:** Signals (RSI)
**Justification:**

1. **Write path already active** — the writer service already consumes `signal.events.rsi.generated` from NATS and inserts into the `signals` ClickHouse table via `mapSignalRow()`. Only the read path needed building.
2. **Simplest domain type after candles** — Signal has 8 domain fields vs. 16+ for evidence candles. The Metadata field (map[string]string serialized as JSON) introduces exactly one new concern (JSON deserialization) beyond what candle reads handle.
3. **Schema already exists** — migration `002_create_signals.sql` was applied in S147. No new DDL required.
4. **S163 recommendation** — the S163 report explicitly identified signals as the lowest-risk first Wave B candidate.
5. **Dependency chain root** — signals are Layer 1 in the domain dependency graph (depends on evidence). Testing the pattern here validates the simplest non-evidence family before more complex ones (decisions, strategies, risk, executions).

## What Was Built

### New Artifacts (4 files)

| Artifact | File | Purpose |
|---|---|---|
| Reader adapter | `internal/adapters/clickhouse/signal_reader.go` | Parameterized SELECT on `signals` table, row→domain mapping |
| Reader tests | `internal/adapters/clickhouse/signal_reader_test.go` | Query builder tests (8 cases), `ParseMetadataJSON` tests (4 cases) |
| Use case | `internal/application/analyticalclient/get_signal_history.go` | Validation, delegation, timing, logging |
| Use case tests | `internal/application/analyticalclient/get_signal_history_test.go` | 10 test cases (mirrors candle use case coverage) |

### Modified Artifacts (8 files)

| File | Change |
|---|---|
| `internal/application/analyticalclient/contracts.go` | Added `SignalHistoryQuery`, `SignalHistoryReply` |
| `internal/interfaces/http/handlers/analytical.go` | Added `GetSignalHistory` handler, `getAnalyticalSignalHistoryUseCase` interface |
| `internal/interfaces/http/handlers/analytical_test.go` | Added 6 signal handler tests |
| `internal/interfaces/http/routes/analytical.go` | Added `GetSignalHistory` dep, `/analytical/signal/history` route |
| `cmd/gateway/analytical_reader.go` | Added `newAnalyticalSignalReader()` factory |
| `cmd/gateway/analytical_reader_test.go` | Added compile-time interface assertion for `SignalReader` |
| `cmd/gateway/compose.go` | Wired `SignalReader` into `AnalyticalFamilyDeps` |
| `tests/http/analytical.http` | Added 7 signal HTTP test requests |

## Design Decisions

### 1. Signal type as query filter (not path parameter)

The operational signal endpoint uses a path parameter (`/signal/:type/latest`). The analytical endpoint uses a query parameter (`/analytical/signal/history?type=rsi`). This is intentional:

- Analytical endpoints live under `/analytical/` with a flat namespace.
- The `type` filter is mandatory in the query to ensure signals are always scoped to a specific family (RSI, EMA crossover).
- This avoids routing ambiguity and keeps the analytical URL structure uniform across families.

### 2. Metadata deserialization with silent fallback

`ParseMetadataJSON` returns an empty map on invalid JSON rather than failing the entire row. This mirrors the write-path pattern where `marshalJSON` falls back to `"{}"`. The trade-off: a corrupt metadata field doesn't poison the entire query result, but the corruption is silent at the read path (the write-path logs a warning during serialization).

### 3. Shared query parameter parsing

Both candle and signal handlers reuse `parseEvidenceKeyParams()` for source/symbol/timeframe extraction. Despite the function name containing "evidence", these parameters are universal to all domain types. Renaming it would be a horizontal refactor that violates the C-9 (additive only) constraint.

## Simplifications

1. **No signal-type validation** — the reader accepts any `type` string. ClickHouse returns empty results for unknown types. Validation against `knownSignalFamilies` was considered but rejected: it would couple the read path to the settings registry, violating the adapter's isolation boundary.

2. **No metadata schema validation** — the reader deserializes metadata as `map[string]string` without validating that expected keys (e.g., `period`, `avg_gain`, `avg_loss` for RSI) are present. This matches the write path's schema-agnostic approach.

3. **No pagination** — results are limited to 500 rows (same as candles). Cursor-based pagination is a future concern listed in open debts.

## Limits

- Only RSI signals are currently being written to ClickHouse (the EMA crossover writer pipeline exists but depends on the EMA crossover actor being enabled in config).
- The signal read path does not filter by `final` status — all signals (interim and final) are returned. The consumer must filter if needed.
- No cross-family queries (e.g., "signals with their upstream candle") — this is explicitly out of scope per C-8.
