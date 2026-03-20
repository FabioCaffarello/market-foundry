# Writer Service — Initial Event Coverage and Limits

## Covered Families (S148)

The minimal writer covers 6 pipeline families, mapping to the 6 core ClickHouse tables:

| Pipeline Family | ClickHouse Table | Event Type | Source Stream |
|----------------|-----------------|------------|--------------|
| candle | `evidence_candles` | `CandleSampledEvent` | EVIDENCE_EVENTS |
| rsi | `signals` | `SignalGeneratedEvent` | SIGNAL_EVENTS |
| rsi_oversold | `decisions` | `DecisionEvaluatedEvent` | DECISION_EVENTS |
| mean_reversion_entry | `strategies` | `StrategyResolvedEvent` | STRATEGY_EVENTS |
| position_exposure | `risk_assessments` | `RiskAssessedEvent` | RISK_EVENTS |
| paper_order | `executions` | `PaperOrderSubmittedEvent` | EXECUTION_EVENTS |

## Not Covered (Deferred)

| Family | Reason | Target Stage |
|--------|--------|-------------|
| tradeburst | Evidence secondary — not in critical analytical path | Future |
| volume | Evidence secondary — not in critical analytical path | Future |
| ema_crossover | Signal secondary — signals table ready, just needs consumer | Future |
| venue_market_order | Execution secondary — separate stream (EXECUTION_FILL_EVENTS) | Future |

## Delivery Semantics

- **Guarantee:** Best-effort at-least-once. NATS messages are acked after successful decode and buffer. ClickHouse inserts are batch-flushed asynchronously.
- **On ClickHouse failure:** Buffer accumulates. Oldest rows evicted at `max_pending`. Failed batches logged at ERROR and dropped. Consumer continues advancing.
- **On writer crash:** In-flight buffer lost. NATS redelivers unacked messages on restart.
- **Duplicates:** Tolerated. No insert-level deduplication. Handled via `SELECT DISTINCT` at query time if needed.
- **Ordering:** Events arrive in NATS delivery order per partition. No cross-partition ordering guarantee. ClickHouse ORDER BY provides query-time ordering.

## Failure Modes

| Failure | Writer Behavior | Impact on Pipeline |
|---------|----------------|-------------------|
| ClickHouse down | Buffer fills, evicts oldest, logs errors | None — pipeline runs without writer |
| ClickHouse INSERT fails | Batch dropped after single attempt, logged as ERROR | Analytical gap — events lost |
| NATS down | Consumer stops receiving, inserter drains buffer | Same as any NATS outage |
| Decode failure | Message terminated (NATS won't redeliver), logged as WARN | Single event lost |
| Writer process crash | Buffer lost, NATS redelivers unacked | Temporary gap, recovers on restart |

## Diagnostic Visibility

### Structured Logging

Every flush and failure is logged with:
- `family` — pipeline family name
- `table` — ClickHouse target table
- `rows` — batch size
- `error` — failure details (on error)

### Health Tracker Counters

Per-family inserter trackers expose:
- `events_flushed` — total rows successfully inserted
- `events_dropped` — rows lost to flush failure or buffer overflow

Per-family consumer trackers expose standard `event_count` and `error_count`.

### Health Endpoints

- `/statusz` — aggregated phase (starting/warming/active/idle/stalled) with per-tracker breakdown
- `/diagz` — full diagnostic with readiness check results, tracker stats, goroutine count

## Type Mapping Conventions

| Go Type | ClickHouse Type | Conversion |
|---------|----------------|------------|
| `string` (decimal) | `Float64` | `strconv.ParseFloat` (0.0 on failure) |
| `string` (enum) | `LowCardinality(String)` | Pass through |
| `string` (free text) | `String` | Pass through |
| `map[string]string` | `String` (JSON) | `json.Marshal` |
| `[]T` (slice) | `String` (JSON) | `json.Marshal` |
| `struct` (nested) | `String` (JSON) | `json.Marshal` |
| `time.Time` | `DateTime64(3)` | Pass through (driver handles) |
| `int` / `int64` | `UInt32` / `Int64` | Type cast |
| `bool` | `Bool` | Pass through |

## Limits

1. **No deduplication** — duplicates from NATS redelivery are appended as-is.
2. **No transformation** — events are mechanically mapped, not enriched or filtered.
3. **No retry on INSERT failure** — single attempt per batch, then drop.
4. **No backfill** — only processes events from NATS consumer's current position.
5. **No materialized views** — raw append-only tables only.
6. **No query surface** — writer only writes; query endpoints are a future stage.
7. **Single ClickHouse instance** — no clustering or replication.
8. **No dynamic family registration** — families are declared at compile time.

## Preparation for Future Stages

The writer's coverage can be extended by:

1. **Adding families** — Add consumer spec + pipeline entry + mapper function. No architectural change needed.
2. **Adding query endpoints** — Gateway can query ClickHouse tables directly via a future `internal/adapters/clickhouse/query.go` or via the writer service exposing NATS request/reply handlers.
3. **Cold-start bootstrap** — Derive can optionally query evidence_candles for warm-up data on startup.
