# Latest and History by Family

> Defines the read model strategy per projection family: which families have latest-only projections, which have history, and the rules governing each.

## Read Model Classes

Every projection family materializes into one of two read model classes:

### Latest-Only

A single KV bucket per family storing the most recent finalized value per partition key (`source.symbol.timeframe`). Overwrites on each new finalized event, guarded by monotonicity.

**Properties:**
- Key format: `{source}.{symbol}.{timeframe}`
- One value per key (last-writer-wins with monotonicity guard)
- No TTL (values persist until overwritten)
- Bounded by number of active source/symbol/timeframe combinations

**Use case:** Real-time dashboards, latest-value queries, signal generation inputs.

### Latest + History

Two KV buckets per family. The latest bucket works identically to latest-only. The history bucket stores every finalized event indexed by open time.

**Properties:**
- Latest bucket: same as latest-only
- History bucket key: `{source}.{symbol}.{timeframe}.{open_time_unix}`
- TTL-bounded (e.g., 24h for candles)
- Size-bounded (e.g., 256 MB for candles)
- Idempotent by key design — replaying the same event produces the same key
- Query supports prefix scan + optional time-range filtering

**Use case:** Historical analysis, charting, backtesting data access, trend detection.

## Current Family Read Models

| Family | Class | Latest Bucket | History Bucket | History TTL | Max History Size |
|--------|-------|---------------|----------------|-------------|-----------------|
| **candle** | Latest + History | `CANDLE_LATEST` (64 MB) | `CANDLE_HISTORY` (256 MB) | 24h | 256 MB |
| **tradeburst** | Latest-Only | `TRADE_BURST_LATEST` (64 MB) | — | — | — |

### Why candle has history

Candles are the primary evidence type for charting and time-series analysis. Historical access is needed for:
- OHLCV chart rendering (requires N most recent candles)
- Signal generation (may compare current candle to previous windows)
- Backtesting data access

### Why tradeburst is latest-only

Trade bursts are activity indicators, not time-series data. The primary use case is:
- "Is there a burst right now?" (answered by latest)
- Burst history is low-value without context (no charting use case)
- If history is needed later, adding a TRADE_BURST_HISTORY bucket follows the same pattern as candles

## Materialization Rules

### Latest Bucket Rules (all families)

1. **Final gate:** Only events with `Final=true` are written. Interim snapshots are ignored.
2. **Validation gate:** Domain `Validate()` must pass before any write.
3. **Monotonicity guard:** Compare incoming `OpenTime` against existing entry.
   - If incoming is newer → write (overwrite)
   - If incoming is equal → skip (duplicate)
   - If incoming is older → skip (stale)
4. **No delete:** Latest values are never deleted, only overwritten by newer events.

### History Bucket Rules (families with history)

1. **Same final + validation gates** as latest.
2. **Idempotent by key:** Key includes `open_time_unix`, so replaying the same event writes to the same key — no duplicates.
3. **TTL-bounded:** NATS KV applies TTL per entry. Entries expire automatically.
4. **Size-bounded:** MaxBytes limits total bucket size. Oldest entries are evicted when exceeded.
5. **Write order:** Latest is written first, then history. If history write fails, latest is still consistent (latest is authoritative).

### Materialization Counters (all families)

| Counter | Meaning |
|---------|---------|
| `materialized` | Event successfully written to latest (and history if applicable) |
| `skipped_stale` | Existing latest has newer OpenTime |
| `skipped_dedup` | Existing latest has equal OpenTime |
| `skipped_non_final` | Event had Final=false |
| `rejected` | Domain validation failed |
| `errors` | KV write error |

## Query Patterns by Read Model Class

### Latest-Only Queries

```
GET /evidence/{type}/latest?source=X&symbol=Y&timeframe=Z
→ evidence.query.{type}.latest
→ KV lookup: {source}.{symbol}.{timeframe}
→ Returns: single evidence entity or 404
```

### History Queries (families with history)

```
GET /evidence/{type}/history?source=X&symbol=Y&timeframe=Z&limit=N&since=T1&until=T2
→ evidence.query.{type}.history
→ KV prefix scan: {source}.{symbol}.{timeframe}.*
→ Filter by time range (since/until as unix seconds)
→ Sort descending by open_time
→ Limit to N results (default 10, max 100)
→ Returns: array of evidence entities
```

## Decision Framework: Latest-Only vs. Latest + History

When adding a new projection family, use this framework to decide:

| Question | Latest-Only | Latest + History |
|----------|-------------|-----------------|
| Is time-series access needed? | No | Yes |
| Is charting a use case? | No | Yes |
| Does signal generation need window history? | No | Yes |
| Is the primary question "what is the current value"? | Yes | Partially |
| Does the data have backtesting value? | No | Yes |
| Is storage budget a concern? | Budget-friendly | Requires size planning |

**Default:** Start with latest-only. Add history when a concrete use case demands it. History can be added non-breaking (new bucket, new query route, no changes to existing projection).

## Adding History to a Latest-Only Family

If a family needs to graduate from latest-only to latest+history:

1. **Add history bucket** — new constant in NATS adapter, new KV config (with TTL and MaxBytes)
2. **Extend projection actor** — add `PutHistory()` call after latest write
3. **Extend KV store adapter** — add `PutHistory()` and `GetHistory()` methods
4. **Add query route** — new `ControlSpec` in registry, new handler in QueryResponderActor
5. **Add use case** — `GetXxxHistoryUseCase` with range validation
6. **Add HTTP route** — `GET /evidence/{type}/history`
7. **Update pipeline Buckets** — add history bucket name to `ProjectionPipeline.Buckets`

This is additive — no existing code changes, no migration needed.
