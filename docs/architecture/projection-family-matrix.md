# Projection Family Matrix

> Maps every evidence type to its full projection pipeline: source stream, consumer, projection actor, KV bucket, query route, and HTTP endpoint.

---

## Projection Pipeline Pattern

Every evidence type follows the same structural pipeline. No evidence type may deviate from this pattern:

```
EVIDENCE_EVENTS ──filter──→ ConsumerActor ──msg──→ ProjectionActor ──write──→ KV Bucket
                                                                                  │
QueryResponderActor ←──read──────────────────────────────────────────────────────────┘
        │
        └──reply──→ Gateway ──HTTP──→ Client
```

### Pipeline Invariants

1. **One consumer per evidence type.** Each type gets its own durable consumer with a type-specific filter subject. No shared consumer with type routing.
2. **One projection actor per evidence type.** Each type gets its own actor that owns writes to its KV bucket(s). No generic projection framework.
3. **Single-writer per bucket.** Each KV bucket is written to by exactly one projection actor.
4. **Shared query responder.** All evidence types share one `QueryResponderActor` that registers typed routes. The responder reads from all KV buckets.
5. **Final gate.** Only events with `Final=true` are materialized. Interim snapshots exist only in the event stream.
6. **Monotonicity guard.** Latest buckets check existing OpenTime before writing. Stale writes are silently skipped.
7. **Domain validation.** Every event is validated (`Validate()`) before materialization. Invalid events are rejected, not silently dropped.

---

## Current Projections

### P-01: Candle

| Component | Value |
|-----------|-------|
| **Evidence type** | `evidence.candle` |
| **Source stream** | `EVIDENCE_EVENTS` |
| **Consumer durable** | `store-candle` |
| **Consumer filter** | `evidence.events.candle.sampled.>` |
| **Consumer actor** | `EvidenceConsumerActor` |
| **Projection actor** | `CandleProjectionActor` |
| **Health trackers** | `candle-projection`, `candle-consumer` |

#### KV Buckets

| Bucket | Purpose | Key Format | MaxBytes | TTL | Storage |
|--------|---------|-----------|----------|-----|---------|
| `CANDLE_LATEST` | Last finalized candle per key | `{source}.{symbol}.{timeframe}` | 64 MB | — | File |
| `CANDLE_HISTORY` | Time-windowed candle archive | `{source}.{symbol}.{timeframe}.{open_time_unix}` | 256 MB | 24h | File |

#### Query Routes

| Operation | NATS Subject | HTTP Path | Query Params |
|-----------|-------------|-----------|-------------|
| Latest | `evidence.query.candle.latest` | `GET /evidence/candles/latest` | `source`, `symbol`, `timeframe` |
| History | `evidence.query.candle.history` | `GET /evidence/candles/history` | `source`, `symbol`, `timeframe`, `limit`, `since`, `until` |

#### Materialization Metrics

| Counter | Meaning |
|---------|---------|
| `materialized` | Candle written to KV |
| `skipped_stale` | Existing candle has newer OpenTime |
| `skipped_dedup` | Key already exists with same OpenTime |
| `skipped_non_final` | Interim candle ignored |
| `rejected` | Domain validation failed |
| `errors` | KV write error |

---

### P-02: Trade Burst

| Component | Value |
|-----------|-------|
| **Evidence type** | `evidence.tradeburst` |
| **Source stream** | `EVIDENCE_EVENTS` |
| **Consumer durable** | `store-trade-burst` |
| **Consumer filter** | `evidence.events.tradeburst.sampled.>` |
| **Consumer actor** | `TradeBurstConsumerActor` |
| **Projection actor** | `TradeBurstProjectionActor` |
| **Health trackers** | `trade-burst-projection`, `trade-burst-consumer` |

#### KV Buckets

| Bucket | Purpose | Key Format | MaxBytes | TTL | Storage |
|--------|---------|-----------|----------|-----|---------|
| `TRADE_BURST_LATEST` | Last finalized burst per key | `{source}.{symbol}.{timeframe}` | 64 MB | — | File |

#### Query Routes

| Operation | NATS Subject | HTTP Path | Query Params |
|-----------|-------------|-----------|-------------|
| Latest | `evidence.query.tradeburst.latest` | `GET /evidence/tradeburst/latest` | `source`, `symbol`, `timeframe` |

#### Intentional Gaps

- **No history bucket.** Trade burst does not have a historical archive. Latest-only is sufficient for the current use case.
- **No burst-filtered query.** Clients must filter the `Burst` boolean field themselves. No server-side burst-only query exists.

---

### P-03: Volume (implemented S31)

| Component | Value |
|-----------|-------|
| **Evidence type** | `evidence.volume` |
| **Source stream** | `EVIDENCE_EVENTS` |
| **Consumer durable** | `store-volume` |
| **Consumer filter** | `evidence.events.volume.sampled.>` |
| **Consumer actor** | `VolumeConsumerActor` |
| **Projection actor** | `VolumeProjectionActor` |
| **Health trackers** | `volume-projection`, `volume-consumer` |

#### KV Buckets

| Bucket | Purpose | Key Format | MaxBytes | TTL | Storage |
|--------|---------|-----------|----------|-----|---------|
| `VOLUME_LATEST` | Last finalized volume per key | `{source}.{symbol}.{timeframe}` | 64 MB | — | File |

#### Query Routes

| Operation | NATS Subject | HTTP Path | Query Params |
|-----------|-------------|-----------|-------------|
| Latest | `evidence.query.volume.latest` | `GET /evidence/volume/latest` | `source`, `symbol`, `timeframe` |

#### Intentional Gaps

- **No history bucket.** Volume does not have a historical archive. Latest-only is sufficient for the current use case.

---

## Planned Projections

### P-04: Stats (Planned)

| Component | Expected Value |
|-----------|---------------|
| **Evidence type** | `evidence.stats` |
| **Consumer durable** | `store-stats` |
| **Consumer filter** | `evidence.events.stats.sampled.>` |
| **Consumer actor** | `StatsConsumerActor` |
| **Projection actor** | `StatsProjectionActor` |
| **KV bucket (latest)** | `STATS_LATEST` |
| **Query subject** | `evidence.query.stats.latest` |
| **HTTP path** | `GET /evidence/stats/latest` |

---

## Supervisor Actor Tree (Store)

Current state of the store supervisor's actor tree:

```
StoreSupervisor
├── CandleProjectionActor          ○ CANDLE_LATEST, CANDLE_HISTORY (write)
├── CandleConsumerActor            ← EVIDENCE_EVENTS (store-candle)
├── TradeBurstProjectionActor      ○ TRADE_BURST_LATEST (write)
├── TradeBurstConsumerActor        ← EVIDENCE_EVENTS (store-trade-burst)
├── VolumeProjectionActor          ○ VOLUME_LATEST (write)
├── VolumeConsumerActor            ← EVIDENCE_EVENTS (store-volume)
└── QueryResponderActor            ⇄ evidence.query.* (read from all KV buckets)
```

### Growth Pattern

Adding a new evidence type adds exactly 2 actors to the supervisor (one consumer, one projection). The QueryResponderActor gains one additional typed route. No existing actors are modified.

**Current state (3 evidence types):**
- StoreSupervisor: 7 children (3 consumers + 3 projections + 1 query responder)
- QueryResponderActor: 4 query routes (candle latest, candle history, tradeburst latest, volume latest)

**At 5 evidence types** (projected: candle, tradeburst, volume, stats, +1):
- StoreSupervisor: 11 children (5 consumers + 5 projections + 1 query responder)
- QueryResponderActor: 6-8 typed routes (latest + history per applicable type)

**Scaling limit:** QueryResponderActor linearly grows in routes and KV connections. Consider splitting into per-type responders if the store exceeds 10 evidence types (see multi-projection-pattern.md).

---

## Projection Health Model

Each projection pipeline has an independent pair of health trackers:

| Tracker Name | Component | Records |
|-------------|-----------|---------|
| `candle-projection` | CandleProjectionActor | Materialization events |
| `candle-consumer` | EvidenceConsumerActor | Message processing events |
| `trade-burst-projection` | TradeBurstProjectionActor | Materialization events |
| `trade-burst-consumer` | TradeBurstConsumerActor | Message processing events |
| `volume-projection` | VolumeProjectionActor | Materialization events |
| `volume-consumer` | VolumeConsumerActor | Message processing events |

All trackers surface on `/statusz` with independent idle detection (configurable idle timeout). An idle consumer without an idle projection indicates a consumer stall. An idle projection without an idle consumer indicates all events are being filtered (e.g., non-final events skipped).

---

## Cross-Reference: Projection to Actor Ownership

| Projection | Writer Actor | Consumer Actor | KV Bucket(s) | Query Routes | HTTP Routes |
|------------|-------------|---------------|-------------|-------------|-------------|
| Candle | CandleProjectionActor | EvidenceConsumerActor | CANDLE_LATEST, CANDLE_HISTORY | candle.latest, candle.history | /evidence/candles/latest, /evidence/candles/history |
| Trade Burst | TradeBurstProjectionActor | TradeBurstConsumerActor | TRADE_BURST_LATEST | tradeburst.latest | /evidence/tradeburst/latest |
| Volume | VolumeProjectionActor | VolumeConsumerActor | VOLUME_LATEST | volume.latest | /evidence/volume/latest |
| Stats (planned) | StatsProjectionActor | StatsConsumerActor | STATS_LATEST | stats.latest | /evidence/stats/latest |

---

## Adding a New Projection

When adding a new evidence type, this matrix must be updated with:

1. A new P-XX entry with all component values filled
2. New rows in the Supervisor Actor Tree
3. New rows in the Projection Health Model
4. New rows in the Cross-Reference table
5. Updated growth projections

The complete implementation checklist is in [evidence-read-model-guidelines.md](evidence-read-model-guidelines.md).
