# Stream Family Adoption Report: evidence.volume

> First new stream family adopted using the canonical mesh patterns from S26-S30.

## Family Chosen

**`evidence.volume`** — per-window volume profile with VWAP, buy/sell distribution, and total notional volume.

## Why volume

| Criterion | Assessment |
|-----------|------------|
| **Simplicity** | Same input (observation trades), same windowing, same finalization pattern as candle and tradeburst |
| **Distinctness** | Provides VWAP and total notional volume — metrics not available from candle (price) or tradeburst (activity) |
| **Utility** | Directly useful for signal generation: VWAP is a key technical indicator, volume delta informs market pressure |
| **Pipeline fit** | Reuses the proven evidence derivation pipeline with zero infrastructure changes |
| **Mesh alignment** | Follows the `evidence.events.{type}.sampled.{source}.{symbol}.{timeframe}` subject pattern exactly |

Alternatives considered:
- `evidence.stats` — statistical summary (std dev, spread) — deferred because distributional metrics are more complex and less immediately useful than VWAP
- `evidence.orderbook` — not applicable (requires a different data source, not observation trades)

## Domain Type

```go
type EvidenceVolume struct {
    Source      string    // exchange identifier
    Symbol      string    // trading pair
    Timeframe   int       // window seconds
    BuyVolume   string    // decimal — Σ(price × qty) where BuyerMaker=true
    SellVolume  string    // decimal — Σ(price × qty) where !BuyerMaker
    TotalVolume string    // decimal — BuyVolume + SellVolume
    VWAP        string    // decimal — TotalVolume / TotalQuantity
    TradeCount  int64
    OpenTime    time.Time
    CloseTime   time.Time
    Final       bool
}
```

**Key design decisions:**
- VWAP = Σ(price × qty) / Σ(qty), not a simple average — this is the standard financial definition
- All monetary values are decimal strings (same convention as candle and tradeburst)
- `big.Float` arithmetic in the sampler to avoid IEEE 754 precision loss
- Latest-only projection (no history bucket) — mirrors tradeburst pattern; history can be added later if needed

## Pipeline Trace

```
observation.trade → VolumeSampler (derive)
    → evidence.events.volume.sampled.{source}.{symbol}.{timeframe}
    → VolumeConsumer (store, durable: store-volume)
    → VolumeProjectionActor → VOLUME_LATEST KV
    → QueryResponderActor (evidence.query.volume.latest)
    → EvidenceGateway → GetLatestVolumeUseCase
    → GET /evidence/volume/latest?source=X&symbol=Y&timeframe=Z
```

## Mesh Entry Points

| Layer | Component | Value |
|-------|-----------|-------|
| **Event subject** | `evidence.events.volume.sampled.{source}.{symbol}.{timeframe}` | Same stream as candle/tradeburst |
| **Envelope type** | `evidence.events.v1.volume_sampled` | Versioned |
| **Dedup key** | `vol:{source}:{symbol}:{timeframe}:{open_time_unix}` | Unique prefix |
| **Consumer durable** | `store-volume` | Independent cursor |
| **KV bucket** | `VOLUME_LATEST` (64 MB, FileStorage) | Latest-only |
| **Query subject** | `evidence.query.volume.latest` | Queue group: `evidence.query` |
| **HTTP endpoint** | `GET /evidence/volume/latest` | Same param pattern |

## What the Patterns Produced

### Derive (FamilyProcessor from S28)

One entry added to `DeriveSupervisor.start()`:
```go
{
    Family:      "volume",
    ActorPrefix: "volume-sampler",
    NewActor:    func(...) { return NewVolumeSamplerActor(...) },
}
```
**SourceScopeActor untouched.** The spawning loop handled volume automatically.

### Store (ProjectionPipeline from S29)

One entry added to `StoreSupervisor.start()`:
```go
{
    Family:         "volume",
    ProjectionName: "volume-projection",
    ConsumerName:   "volume-consumer",
    Buckets:        []string{VolumeLatestBucket},
    ConsumerSpec:   StoreVolumeConsumer(),
    NewProjection:  func(...) { return NewVolumeProjectionActor(...) },
    NewConsumer:    func(...) { return NewVolumeConsumerActor(...) },
}
```
**StoreSupervisor spawning loop untouched.** The pipeline was added declaratively.

### Gateway (EvidenceFamilyDeps from S30)

One field added to `EvidenceFamilyDeps`:
```go
GetLatestVolume handlersGetLatestVolumeUseCase
```
One route block added to `Evidence()`:
```go
if deps.GetLatestVolume != nil {
    routes = append(routes, webserver.Route{...})
}
```
**DefaultRoutes untouched.** `HasAny()` automatically includes volume.

## Intentional Limitations

1. **Latest-only** — no history bucket. Volume history can be added following the candle pattern if charting of volume over time is needed.
2. **No volume delta field** — clients compute `BuyVolume - SellVolume` themselves. Adding a derived field is trivial if needed.
3. **Same timeframes as candle/tradeburst** — volume window aligns with the configured timeframes. No independent volume timeframes.
4. **No VWAP bands** — no standard deviation around VWAP. This would require a more complex sampler (rolling stats). Deferred to evidence.stats.
