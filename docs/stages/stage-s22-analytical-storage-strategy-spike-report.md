# S22: Analytical Storage Strategy Spike — Stage Report

## Summary

Executed an architectural spike to evaluate ClickHouse as the next-generation historical/analytical storage layer for market-foundry. The spike produced a clear decision, a comparative analysis, and a gated adoption plan — without making any runtime changes.

**Recommendation: ClickHouse is the preferred analytical backend, but it does not enter the runtime now. Adoption is deferred until a concrete trigger is met.**

## Deliverables Produced

| Document | Purpose |
|----------|---------|
| `docs/architecture/analytical-storage-strategy.md` | Architectural decision record: what, why, when, boundary rules |
| `docs/architecture/clickhouse-vs-timescale-vs-current-store.md` | Comparative analysis with decision matrix |
| `docs/architecture/clickhouse-adoption-plan.md` | 4-slice incremental adoption plan with rollback |
| `docs/stages/stage-s22-analytical-storage-strategy-spike-report.md` | This report |

## Recommendation

### ClickHouse enters — but not now

**Why ClickHouse is the right candidate:**
1. OHLCV candles are append-only, immutable, time-indexed — MergeTree's sweet spot
2. ReplacingMergeTree provides server-side dedup that aligns with existing replay/idempotency semantics
3. Columnar compression and partitioning handle multi-month retention efficiently
4. Server-side aggregation (hourly/daily rollups, volume profiles) is impossible with NATS KV
5. `ORDER BY (source, symbol, timeframe, open_time)` maps 1:1 to the existing key structure

**Why not now:**
1. **Scale doesn't demand it.** 2 symbols × 2 timeframes = 8 candles/min. NATS KV handles this with zero stress.
2. **No analytics consumers exist.** No dashboard, strategy engine, or reporting layer reads candle history today.
3. **24h NATS KV history is sufficient** for current operational queries.
4. **Operational cost is real.** New container, new Go dependency, new schema management, new monitoring — all for 8 writes/min.

**Why not TimescaleDB:**
- Row-oriented storage slower for analytical scans
- ACID guarantees unnecessary for idempotent, append-only projections
- PostgreSQL operational weight (vacuuming, connection pooling) without structural benefit
- ClickHouse's columnar compression and MergeTree engine are purpose-built for this exact workload

## Architectural Racional

### Where ClickHouse enters

ClickHouse enters as a **secondary projection target** inside the existing store service. The projection actor gains a third write path (after CANDLE_LATEST and CANDLE_HISTORY). The query responder routes historical queries beyond 24h to ClickHouse.

```
CandleProjectionActor
  ├── CANDLE_LATEST   (always, hot path)
  ├── CANDLE_HISTORY  (always, 24h cache)
  └── ClickHouse      (when enabled, fire-and-forget)
```

### What stays in NATS KV — always

| Projection | Reason |
|------------|--------|
| CANDLE_LATEST | Sub-millisecond point lookup. ClickHouse can't match this. |
| CANDLE_HISTORY (24h) | Hot cache for recent operational queries. Low-latency prefix scan. |

### What goes to ClickHouse — when triggered

| Data | Purpose |
|------|---------|
| All finalized candles | Unbounded historical archive (90-day default TTL) |
| Aggregated views | Materialized views for analytical queries (hourly, daily OHLCV) |
| Future evidence types | Volume profiles, funding rates — purely analytical projections |

### Boundary preservation

- Gateway never talks to ClickHouse
- Store remains sole read-model authority
- ClickHouse failure does not block the hot path (NATS KV writes proceed independently)
- Query routing is store-internal (transparent to gateway and HTTP consumers)
- Same contracts, same NATS subjects, same HTTP endpoints

## Trigger Conditions

ClickHouse adoption is triggered when **any one** of:

| Trigger | Detection |
|---------|-----------|
| >10 active symbols | NATS KV `Keys()` scan becomes latency-visible (~14K+ keys/day) |
| Retention need >24h | External consumer requests historical data beyond KV TTL |
| Aggregation query need | Consumer needs server-side OHLCV rollups or cross-timeframe analysis |
| Stream exhaustion | Extended outage makes 72h stream replay insufficient for rebuilding history |

## Adoption Slices (when triggered)

| Slice | Scope | Effort | Risk |
|-------|-------|--------|------|
| 0 | Schema + Docker container (infrastructure only) | ~1 stage | None |
| 1 | Write adapter (projection actor writes to ClickHouse) | ~1 stage | Low |
| 2 | Read adapter (query responder routes to ClickHouse) | ~1 stage | Medium |
| 3 | Backfill (replay stream into ClickHouse) | ~0.5 stage | Low |

Each slice is independently deployable, testable, and reversible. Total: ~3-4 stages.

## Non-Adoption Scenario

If ClickHouse is never triggered, the system continues to work correctly:
- CANDLE_LATEST serves "what's the current candle" queries indefinitely
- CANDLE_HISTORY serves "last N candles within 24h" queries
- EVIDENCE_EVENTS stream provides 72h replay for projection rebuilds
- The only limitation is: no queries beyond 24h and no server-side aggregation

This is a valid operating mode for a system that processes a small number of symbols without analytics consumers.

## Risks and Trade-offs

| Risk | Assessment | Mitigation |
|------|-----------|-----------|
| Premature adoption | Triggers are measurable. No adoption without evidence. | Explicit trigger conditions with detection methods. |
| Dual-write complexity | ClickHouse is fire-and-forget. KV is primary. | ClickHouse failure logged, not fatal. Replay fills gaps. |
| Query routing complexity | Binary: within 24h → KV, beyond → ClickHouse. | Simple time-based routing, not dynamic. |
| Schema drift | ClickHouse schema derived from domain type. | Same migration discipline as any DB schema. |
| Operational overhead | Single-node ClickHouse in Docker. | Same deployment model as NATS. No cluster management. |

## Impact on Roadmap

### No impact on current roadmap

This spike produces zero runtime changes. All documents are strategy/planning artifacts. The store service is unchanged. The projection pipeline is unchanged.

### Future roadmap — if triggered

When a trigger condition is met, the adoption slices slot naturally into the stage sequence:
- S(N): Slice 0 — ClickHouse infrastructure
- S(N+1): Slice 1 — Write adapter
- S(N+2): Slice 2 — Read adapter
- S(N+3): Slice 3 — Backfill

These stages can interleave with other work (new evidence types, multi-exchange, etc.).

## S23 Preparation

Independent of ClickHouse adoption, the following S23 candidates emerge from this analysis:

1. **Projection lag metric** — Track delta between EVIDENCE_EVENTS stream head and last projected candle. This is valuable regardless of storage backend.
2. **Integration test with embedded NATS** — Verify monotonicity guard, history dedup, and replay behavior end-to-end. Prerequisite for adding any new projection target.
3. **Multi-symbol activation** — Wire BindingWatcherActor in store to dynamically respond to configctl bindings. This is the most likely trigger for ClickHouse adoption (>10 symbols).
4. **Health metrics enrichment** — Add candle production rate, projection throughput, and query latency to `/statusz`. Operational visibility before scaling.

**Recommended S23 focus:** Items 1-2 (projection observability hardening) — they strengthen the foundation regardless of whether ClickHouse adoption is triggered.
