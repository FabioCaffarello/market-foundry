# ExecutionIntent Queryability, Correlation, PriceSource Runtime, and Limitations

> Stage: S387 — OMS Foundation Wave
> Status: Complete
> Scope: Read-path queryability model, correlation semantics, PriceSource runtime behavior

## 1. Queryability Model

After S387, the execution lifecycle is queryable across four NATS KV buckets:

| Bucket | Owner | Contents | Semantics |
|--------|-------|----------|-----------|
| `EXECUTION_PAPER_ORDER_LATEST` | store (paper_order pipeline) | Latest paper intent per source/symbol/timeframe | Intent from derive |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | store (venue_market_order pipeline) | Latest venue fill result per source/symbol/timeframe | Fill from execute |
| `EXECUTION_VENUE_REJECTION_LATEST` | store (venue_rejection pipeline) | Latest venue rejection per source/symbol/timeframe | Rejection from execute |
| `EXECUTION_CONTROL` | store (query responder) | Global gate + activation dimensions | Control plane |

### Composite Status Query

The `execution.query.status.latest` endpoint returns all four surfaces in a single response:

```json
{
  "intent":      { ... },  // paper_order latest (nullable)
  "result":      { ... },  // venue fill latest (nullable)
  "rejection":   { ... },  // venue rejection latest (nullable)
  "gate":        { "status": "active", ... },
  "propagation": "filled"  // derived from most-recent outcome
}
```

### Propagation Derivation

`DeriveEffectivePropagation(intent, result, rejection)`:

1. If both `result` and `rejection` exist: compare timestamps, most recent wins.
2. If only `result` exists: use result status.
3. If only `rejection` exists: use rejection status.
4. If neither exists but `intent` does: use intent status.
5. If all nil: `"none"`.

This model answers the question: "What is the current effective lifecycle state for this source/symbol/timeframe?"

## 2. Correlation Model

Every `ExecutionIntent` carries:

| Field | Source | Purpose |
|-------|--------|---------|
| `CorrelationID` | Derive binary (set at intent creation) | End-to-end trace through events |
| `CausationID` | Each producer (set at event creation) | Direct predecessor link |
| `DeduplicationKey` | `exec:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}` | Stream-level idempotency |
| `PartitionKey` | `{source}.{symbol}.{timeframe}` | KV bucket key |

### Correlation Chain

```
RiskAssessedEvent (derive)
  → CorrelationID set
  → PaperOrderSubmittedEvent (derive, EXECUTION_EVENTS)
    → CausationID = risk event ID
  → VenueOrderFilledEvent OR VenueOrderRejectedEvent (execute, EXECUTION_FILL/REJECTION_EVENTS)
    → CausationID = paper order event ID
    → CorrelationID preserved from original
```

All three KV projections (intent, fill, rejection) preserve the `CorrelationID` from the original intent, enabling end-to-end tracing from risk assessment to venue outcome.

## 3. PriceSource Runtime

### Architecture

```
CANDLE_LATEST KV bucket (written by store/candle-projection)
  ↓ (read-only)
CandleKVPriceSource (execute binary)
  ↓ (injected via WithPriceSource)
DryRunSubmitter / PaperVenueAdapter
  ↓ (resolvePrice called per SubmitOrder)
FillRecord.Price = last observed Close
```

### Key Properties

| Property | Value |
|----------|-------|
| Source bucket | `CANDLE_LATEST` |
| Key format | `{source}.{symbol}.{timeframe}` |
| Field read | `Close` (decimal string) |
| Fallback | `"0"` on any error or missing data |
| Thread safety | Yes (delegates to NATS KV client) |
| Connection | Separate from execute's main NATS connection |
| Lifetime | Binary-scoped (opened at startup, closed at shutdown) |

### Fill Price Realism by Mode

| Mode | PriceSource Used | Fill Price |
|------|-----------------|------------|
| Dry-run (default) | Yes, if available | Last close from CANDLE_LATEST, else "0" |
| Paper | Yes, if available | Last close from CANDLE_LATEST, else "0" |
| Venue live | No (venue sets price) | Real venue fill price |

## 4. Limitations

### Read-Path Limitations

- **Latest-only**: KV buckets store only the most recent state per partition key. Historical lifecycle transitions are not queryable from KV — use JetStream streams (72h retention) for event replay.
- **Eventual consistency**: The store projection is eventually consistent with the event stream. There is a small window between event publish and KV materialization.
- **No cross-partition queries**: Cannot query "all rejections across symbols" — only per source/symbol/timeframe.
- **Rejection degradation**: If the `EXECUTION_VENUE_REJECTION_LATEST` bucket is unavailable, the status query returns without the rejection field (best-effort).

### PriceSource Limitations

- **Stale price**: The price reflects the last finalized candle close, not real-time. In fast-moving markets, the fill price may not match the current market price.
- **Cold start**: On first deploy or for new symbols, no candle data exists — fills default to `"0"` until the first candle is materialized.
- **Single timeframe**: Price lookup uses the intent's own timeframe. If candles for that specific timeframe are not enabled in the pipeline, the lookup returns `"0"`.
- **No venue price for paper/dry-run**: The PriceSource provides candle close prices, not order book prices. This is sufficient for paper trading simulation but not for production venue pricing.

### Persistence Limitations

- **No writer wiring for rejections**: Rejection events have writer consumer specs (S386) but the ClickHouse writer actor is not wired in this stage. Rejections persist in JetStream (72h) and KV (latest only).
- **No lifecycle history**: The KV model is latest-only. A full lifecycle history (submitted → sent → accepted → filled) is not available via KV — only the terminal state.
- **No multi-family aggregation**: The status query combines paper_order, venue fill, and venue rejection for one partition key. It does not aggregate across multiple symbols or timeframes.

### Out of Scope

- OMS: Order management, position tracking, portfolio aggregation.
- Write-path enforcement: Quantity invariants tested but not enforced at domain level.
- Dashboards or alerts: No Grafana or observability tooling in scope.
- Price source for venue mode: Real venue orders get prices from the exchange, not from PriceSource.
