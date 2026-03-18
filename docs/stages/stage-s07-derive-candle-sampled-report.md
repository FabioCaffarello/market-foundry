# Stage S07 — Derive: Candle Sampled from Observation

**Status:** Complete
**Date:** 2026-03-16

## Executive Summary

S07 implements the first derivation stage of the market-foundry first slice. The `derive` binary consumes `TradeReceivedEvent` from the `OBSERVATION_EVENTS` JetStream stream (published by ingest in S06), samples 60-second OHLCV candles using pure application logic, and publishes `CandleSampledEvent` to the `EVIDENCE_EVENTS` stream.

This closes the observation-to-evidence pipeline: raw market data flows from an external source through ingest normalization, then through derive aggregation, producing structured evidence that downstream consumers can query.

## Observation → Derivation Flow

```
OBSERVATION_EVENTS (JetStream)
  Subject: observation.events.market.trade.binancef
  Durable: derive-observation
        │
        ▼
  ConsumerActor (decode CBOR → TradeReceivedEvent)
        │
        ▼ tradeReceivedMessage
  SamplerActor (CandleSampler — 60s OHLCV accumulation)
        │
        │ on window rollover → finalized candle
        ▼ publishCandleMessage
  EvidencePublisherActor (encode CBOR → JetStream)
        │
        ▼
EVIDENCE_EVENTS (JetStream)
  Subject: evidence.events.candle.sampled.binancef.btcusdt.60
  Dedup:   binancef:btcusdt:60:{open_time_unix}
```

## Files Changed/Created

### New Files

| File | Layer | Purpose |
|------|-------|---------|
| `internal/application/derive/sampler.go` | Application | Pure OHLCV candle accumulation logic |
| `internal/application/derive/sampler_test.go` | Application | Unit tests: single window, rollover, validation |
| `internal/adapters/nats/observation_consumer.go` | Adapter | JetStream durable consumer for observation trades |
| `internal/adapters/nats/evidence_publisher.go` | Adapter | JetStream publisher for evidence candle events |
| `internal/actors/scopes/derive/derive_supervisor.go` | Actor | Root supervisor — spawns consumer + sampler + publisher |
| `internal/actors/scopes/derive/consumer_actor.go` | Actor | Owns durable consumer, decodes and forwards trades |
| `internal/actors/scopes/derive/sampler_actor.go` | Actor | Owns CandleSampler, emits finalized candles |
| `internal/actors/scopes/derive/publisher_actor.go` | Actor | Owns NATS connection, publishes evidence events |
| `internal/actors/scopes/derive/messages.go` | Actor | Actor message types |

### Modified Files

| File | Change |
|------|--------|
| `cmd/derive/run.go` | Wired actor engine + DeriveSupervisor (replaced stub) |

## Subject Naming and Ownership

| Subject | Owner | Direction |
|---------|-------|-----------|
| `observation.events.market.trade.binancef` | ingest | Published by ingest, consumed by derive |
| `evidence.events.candle.sampled.binancef.btcusdt.60` | derive | Published by derive |

### Stream Ownership

| Stream | Owner | Retention | Max Size |
|--------|-------|-----------|----------|
| `OBSERVATION_EVENTS` | ingest | 6h | 1 GB |
| `EVIDENCE_EVENTS` | derive | 72h | 2 GB |

### Consumer Ownership

| Durable Consumer | Owner | Stream | Filter |
|------------------|-------|--------|--------|
| `derive-observation` | derive | `OBSERVATION_EVENTS` | `observation.events.market.trade.>` |

### Deduplication Strategy

- **Observation**: `{source}:{trade_id}` — one message per trade
- **Evidence**: `{source}:{symbol}:{timeframe}:{open_time_unix}` — one finalized candle per window

## Architectural Decisions

1. **Pure application sampler** — `CandleSampler` has zero I/O dependencies. It takes `ObservationTrade` in, returns `EvidenceCandle` out. All NATS/actor concerns are in their respective layers.

2. **Window rollover on trade arrival** — A candle is finalized when the first trade of the *next* window arrives. This means the last candle of a quiet period won't finalize until trading resumes. This is intentional for the first slice — timer-based flush is a future enhancement.

3. **Volume = sum(price * quantity)** — The candle's volume field is the cumulative notional value (price * qty) across all trades in the window, calculated with `math/big.Float` for precision.

4. **Explicit ack policy** — The durable consumer uses `AckExplicitPolicy`. Each trade is acked only after successful delivery to the sampler actor. Decode errors are terminated (not redelivered).

5. **Single sampler per supervisor** — No exchange/symbol scoping hierarchy yet. The supervisor directly spawns one sampler for `binancef/btcusdt/60s`. Multi-symbol routing is deferred.

6. **CBOR everywhere** — Both input (observation) and output (evidence) use the same CBOR envelope encoding with the shared `encodeEvent`/`decodeEvent` codec.

## How to Run

```bash
# Requires ingest to be publishing observations
make up
docker compose -f deploy/compose/docker-compose.yaml logs -f derive
```

Expected logs:
```
level=INFO msg="derive starting"
level=INFO msg="evidence publisher started"
level=INFO msg="candle sampler started" source=binancef symbol=btcusdt timeframe_s=60
level=INFO msg="observation consumer started" durable=derive-observation filter="observation.events.market.trade.>"
level=INFO msg="derive runtime started" source=binancef symbol=btcusdt timeframe_s=60 ...
level=INFO msg="candle finalized" source=binancef symbol=btcusdt timeframe=60 open_time=2026-03-16T12:00:00Z trades=142
```

Verify evidence events:
```bash
nats stream info EVIDENCE_EVENTS
nats sub "evidence.events.candle.sampled.>" --stream EVIDENCE_EVENTS
```

## Intentional Limitations

1. **Single timeframe (60s)** — Only 1-minute candles. 5-minute and other timeframes are deferred.
2. **Single source/symbol** — `binancef/btcusdt` hardcoded. No dynamic config activation.
3. **No timer-based flush** — Candles only finalize on the arrival of a trade in the next window. Quiet markets delay finalization.
4. **No interim candle publishing** — Only final candles are published. Real-time preview candles are deferred.
5. **No query responder** — `evidence.query.candle.latest` is defined in the registry but not wired. The gateway cannot yet query candles.
6. **No backpressure** — Actor mailboxes are unbounded.

## Points to Review Before S08

1. **Query responder** — To close the full slice (gateway → evidence), the `CandleLatest` control spec needs a responder actor in derive and a gateway-side client. This is likely S08 scope.

2. **Timer-based flush** — For production use, a periodic tick should finalize stale windows that haven't received new trades. Could be a simple timer message sent to the sampler actor.

3. **Multi-symbol routing** — When adding more symbols, an `ExchangeScopeActor → SymbolScopeActor → SamplerActor` hierarchy (as documented in first-slice-contracts) will replace the flat single-sampler approach.

4. **Config-driven activation** — Both ingest and derive currently hardcode their source/symbol. The `PipelineWatcherActor` pattern (watching configctl events) should be the next structural improvement.

5. **Consumer error handling** — Currently, decode failures are terminated. Production should log the raw payload for debugging and track error rates via metrics.

6. **Volume semantics** — Confirm with downstream consumers whether volume should be notional (price*qty, current) or base quantity. The domain type says "total traded volume" which could mean either.
