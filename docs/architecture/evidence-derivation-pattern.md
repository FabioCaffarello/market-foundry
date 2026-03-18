# Evidence Derivation Pattern

> Canonical reference for how new evidence types are derived from observations in market-foundry.

## Pattern Overview

Every evidence type in market-foundry follows the same structural pipeline:

```
observation event → sampler (pure logic) → evidence event → projection (KV) → query (request/reply) → HTTP
```

This pattern is proven by two evidence types: **candles** (OHLCV aggregation) and **trade bursts** (activity/burst detection). Both share identical infrastructure with different domain logic.

## Pipeline Stages

### Stage 1: Observation Consumption

All evidence types derive from the same observation stream (`OBSERVATION_EVENTS`). The derive service consumes trades through a durable JetStream consumer and routes them to samplers by source → symbol.

**Key property:** One observation consumer feeds all evidence samplers. Adding a new evidence type does NOT require a new observation consumer.

### Stage 2: Sampler (Pure Application Logic)

Each evidence type has a pure sampler in `internal/application/derive/`:

| Sampler | Input | Output | Logic |
|---------|-------|--------|-------|
| `CandleSampler` | Trade | `EvidenceCandle` | OHLCV aggregation per window |
| `TradeBurstSampler` | Trade | `EvidenceTradeBurst` | Trade count + buy/sell volume + burst detection |

**Invariants:**
- Samplers have zero I/O dependencies (no NATS, no actors, no network)
- Window boundaries computed as `floor(timestamp / timeframe) * timeframe`
- Finalization triggered by first trade in next window (same for all types)
- Only `Final=true` events are published

### Stage 3: Actor Integration

Each sampler is wrapped in an actor (`SamplerActor`, `TradeBurstSamplerActor`). The source scope actor spawns one sampler actor per (symbol, timeframe, evidence_type) combination. All sampler actors for a symbol receive every trade for that symbol (fan-out).

```
SourceScopeActor (binancef)
  └── btcusdt samplers:
      ├── CandleSamplerActor (60s)
      ├── CandleSamplerActor (300s)
      ├── TradeBurstSamplerActor (60s)
      └── TradeBurstSamplerActor (300s)
```

### Stage 4: Evidence Publication

One `EvidencePublisherActor` per source handles all evidence types. Each type has its own publish method with type-specific subject and dedup key:

| Type | Subject | Dedup key |
|------|---------|-----------|
| Candle | `evidence.events.candle.sampled.{src}.{sym}.{tf}` | `{src}:{sym}:{tf}:{open_time}` |
| TradeBurst | `evidence.events.tradeburst.sampled.{src}.{sym}.{tf}` | `burst:{src}:{sym}:{tf}:{open_time}` |

All evidence events flow into the same `EVIDENCE_EVENTS` stream (subjects: `evidence.events.>`).

### Stage 5: Store Projection

Each evidence type has its own:
- **Durable consumer** (filters by event subject prefix)
- **Consumer actor** (decodes events, forwards to projection)
- **Projection actor** (validates, writes to KV with monotonicity guard)
- **KV bucket** (latest per source/symbol/timeframe)

| Type | Consumer | Bucket | Projection actor |
|------|----------|--------|-----------------|
| Candle | `store-evidence` | `CANDLE_LATEST` + `CANDLE_HISTORY` | `CandleProjectionActor` |
| TradeBurst | `store-trade-burst` | `TRADE_BURST_LATEST` | `TradeBurstProjectionActor` |

### Stage 6: Query Path

One `QueryResponderActor` serves all evidence queries via typed control routes. Each evidence type registers its own subject/request/reply spec:

| Type | Subject | Queue group |
|------|---------|-------------|
| Candle latest | `evidence.query.candle.latest` | `evidence.query` |
| Candle history | `evidence.query.candle.history` | `evidence.query` |
| TradeBurst latest | `evidence.query.tradeburst.latest` | `evidence.query` |

### Stage 7: HTTP Endpoint

The gateway exposes each evidence type through a dedicated endpoint:

| Type | Path | Parameters |
|------|------|-----------|
| Candle latest | `GET /evidence/candles/latest` | source, symbol, timeframe |
| Candle history | `GET /evidence/candles/history` | source, symbol, timeframe, limit, since, until |
| TradeBurst latest | `GET /evidence/tradeburst/latest` | source, symbol, timeframe |

## Adding a New Evidence Type

To add a third evidence type (e.g., volume profile, funding rate), follow these steps:

1. **Domain:** `internal/domain/evidence/` — type + Validate() + event
2. **Sampler:** `internal/application/derive/` — pure logic + tests
3. **Sampler actor:** `internal/actors/scopes/derive/` — wrap sampler in actor
4. **Source scope:** Add sampler spawning in `onActivateSampler`
5. **Publisher:** Add publish method to `EvidencePublisher`
6. **Registry:** Add EventSpec + ControlSpec + ConsumerSpec
7. **KV store:** New adapter in `internal/adapters/nats/`
8. **Store actors:** Consumer + projection + extend query responder
9. **Store supervisor:** Spawn new actors
10. **Client contracts:** Query/reply types + use case
11. **Evidence port:** Add to `EvidenceGateway` interface
12. **Evidence gateway:** Add NATS gateway method
13. **HTTP handler:** Add handler method + route
14. **Gateway wiring:** Create use case, add to Dependencies

Each step follows an established pattern. No framework or abstraction needed — just follow the existing code.
