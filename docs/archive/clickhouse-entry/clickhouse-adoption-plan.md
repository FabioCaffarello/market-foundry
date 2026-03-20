# ClickHouse Adoption Plan

> Incremental adoption plan for ClickHouse as the analytical/historical storage layer. Gated by trigger conditions defined in `analytical-storage-strategy.md`.

## Adoption Model: Sidecar Projection

ClickHouse enters as a **sidecar projection target** inside the existing store service. The projection actor gains a second write target. The query responder gains a second read source. No new services, no new NATS subjects, no new actors.

```
CandleProjectionActor
  ├── Gate 1: Final=true
  ├── Gate 2: Validate()
  ├── Write: CANDLE_LATEST (monotonicity guard)    ← always
  ├── Write: CANDLE_HISTORY (idempotent by key)    ← always
  └── Write: ClickHouse candles table              ← new, fire-and-forget

QueryResponderActor
  ├── handleCandleLatest → CANDLE_LATEST KV        ← unchanged
  └── handleCandleHistory
        ├── if within 24h → CANDLE_HISTORY KV      ← existing path
        └── if beyond 24h → ClickHouse              ← new path
```

## Slice Plan

### Slice 0: Schema + Container (infrastructure only)

**Goal:** ClickHouse runs in Docker Compose, schema exists, no runtime integration.

**Deliverables:**
- `deploy/clickhouse/init.sql` — candles table DDL
- `deploy/compose/docker-compose.yaml` — ClickHouse container
- Schema matches `EvidenceCandle` domain type exactly

**Table design:**
```sql
CREATE TABLE IF NOT EXISTS candles (
    source      LowCardinality(String),
    symbol      LowCardinality(String),
    timeframe   UInt32,
    open_time   DateTime64(3, 'UTC'),
    close_time  DateTime64(3, 'UTC'),
    open        String,
    high        String,
    low         String,
    close       String,
    volume      String,
    trade_count Int64,
    final       Bool
) ENGINE = ReplacingMergeTree()
ORDER BY (source, symbol, timeframe, open_time)
PARTITION BY toYYYYMM(open_time)
TTL open_time + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
```

**Key design decisions:**
- `ReplacingMergeTree` — dedup on `(source, symbol, timeframe, open_time)` for replay safety
- `LowCardinality(String)` — enum-like compression for source/symbol (few distinct values)
- Monetary values as `String` — preserves decimal precision (same as domain type)
- 90-day TTL — generous retention, adjustable without schema change
- Monthly partitioning — efficient range scans and partition drops

**Risk:** None. Infrastructure-only. No runtime coupling.

### Slice 1: Write Adapter (store writes to ClickHouse)

**Goal:** The projection actor writes finalized candles to ClickHouse alongside NATS KV.

**Deliverables:**
- `internal/adapters/clickhouse/candle_store.go` — implements `PutCandle(ctx, candle)`
- `internal/adapters/clickhouse/go.mod` — new module with `clickhouse-go` dependency
- `go.work` — add clickhouse adapter module
- `CandleProjectionActor` — gains optional ClickHouse writer
- `store.jsonc` — gains optional `clickhouse` config section

**Write semantics:**
- Fire-and-forget from projection actor. ClickHouse failure does not block KV writes.
- Batch insert (buffer N candles or flush every T seconds) — ClickHouse prefers batch inserts.
- Error logged and counted in projection stats. No retry — replay will fill gaps.

**Config:**
```jsonc
{
  "clickhouse": {
    "enabled": false,    // opt-in, disabled by default
    "dsn": "clickhouse://clickhouse:9000/market_foundry",
    "batch_size": 100,
    "flush_interval": "5s"
  }
}
```

**Risk:** Low. Fire-and-forget write. Disabled by default. KV remains primary.

### Slice 2: Read Adapter (store reads from ClickHouse for history)

**Goal:** The query responder routes history queries beyond 24h to ClickHouse.

**Deliverables:**
- `internal/adapters/clickhouse/candle_store.go` — add `GetHistory(ctx, source, symbol, timeframe, limit, since, until)`
- `QueryResponderActor` — route logic: if `since` < now-24h → ClickHouse, else → NATS KV
- Contract unchanged — `CandleHistoryQuery` and `CandleHistoryReply` stay the same

**Query routing:**
```
if since > 0 && since < (now - 24h):
    read from ClickHouse
else:
    read from CANDLE_HISTORY KV (existing path)
```

**Risk:** Medium. Introduces a second read path. Requires ClickHouse to be available for historical queries. Mitigated: if ClickHouse is unavailable, return error for historical queries while recent queries still work via KV.

### Slice 3: Backfill (replay stream into ClickHouse)

**Goal:** Populate ClickHouse with existing data from EVIDENCE_EVENTS stream.

**Deliverables:**
- CLI tool or script that replays EVIDENCE_EVENTS (72h) into ClickHouse
- Idempotent: ReplacingMergeTree handles duplicates
- One-time operation, not a runtime concern

**Risk:** Low. Offline operation. No runtime impact.

## What Explicitly Does NOT Change

| Component | Status |
|-----------|--------|
| CANDLE_LATEST bucket | Unchanged. Always served from NATS KV. |
| Gateway HTTP endpoints | Unchanged. Same contracts, same parameters. |
| NATS request/reply subjects | Unchanged. Same subjects, same queue groups. |
| Evidence consumer | Unchanged. Same durable consumer, same ack policy. |
| Projection validation gates | Unchanged. Same Final + Validate gates. |
| Boundary: gateway → store | Unchanged. Gateway never touches ClickHouse. |

## Dependency Impact

| Dependency | Scope |
|------------|-------|
| `clickhouse-go` (Go driver) | New module: `internal/adapters/clickhouse/go.mod` |
| ClickHouse Docker image | `deploy/compose/docker-compose.yaml` |
| No new Go modules in store | Store imports the clickhouse adapter via `go.work` |

## Estimated Effort

| Slice | Effort | Can run independently |
|-------|--------|----------------------|
| Slice 0 | ~1 stage | Yes |
| Slice 1 | ~1 stage | Yes (after Slice 0) |
| Slice 2 | ~1 stage | Yes (after Slice 1) |
| Slice 3 | ~0.5 stage | Yes (after Slice 0) |

Total: ~3-4 stages if triggered. Each slice is independently deployable and testable.

## Rollback Plan

Each slice is reversible:
- **Slice 0:** Delete Docker Compose entry and init.sql. No runtime impact.
- **Slice 1:** Set `clickhouse.enabled = false` in config. Projection actor skips ClickHouse writes.
- **Slice 2:** Revert query routing. All history queries go to NATS KV (24h ceiling returns).
- **Slice 3:** Drop and recreate ClickHouse table. Re-run backfill if needed.
