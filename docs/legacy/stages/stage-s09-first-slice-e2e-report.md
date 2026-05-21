# Stage S09 — First Slice E2E: Observation → Evidence → Query

**Status:** Complete
**Date:** 2026-03-16

## Executive Summary

S09 closes the first vertical slice end-to-end. The complete data flow — from external market data through ingestion, normalization, derivation, and query — is now observable through a single HTTP endpoint on the gateway.

No new domain logic was introduced. S09's contribution is the test coverage gap from S08 (evidence route tests), a reproducible smoke scenario, and the documentation that ties the full slice together.

## E2E Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                        FIRST VERTICAL SLICE                         │
│                                                                     │
│  Binance Futures WS (btcusdt@aggTrade)                             │
│       │                                                             │
│       ▼                                                             │
│  ┌─────────┐  WebSocket    ┌──────────────┐  CBOR/JetStream       │
│  │ ingest  │──aggTrade────▶│ OBSERVATION  │                        │
│  │         │  normalize    │ _EVENTS      │                        │
│  └─────────┘               └──────┬───────┘                        │
│                                   │                                 │
│                                   │ durable: derive-observation     │
│                                   ▼                                 │
│                            ┌─────────────┐                         │
│                            │   derive    │                         │
│                            │             │                         │
│                            │ ┌─────────┐ │  CBOR/JetStream        │
│                            │ │ Sampler │─┼─▶ EVIDENCE_EVENTS      │
│                            │ │  (60s)  │ │                         │
│                            │ └────┬────┘ │                         │
│                            │      │      │                         │
│                            │ ┌────▼────┐ │                         │
│                            │ │ Query   │ │                         │
│                            │ │Responder│ │                         │
│                            │ └────┬────┘ │                         │
│                            └──────┼──────┘                         │
│                                   │ NATS request/reply              │
│                                   │ evidence.query.candle.latest    │
│                                   ▼                                 │
│                            ┌─────────────┐                         │
│                            │   server    │                         │
│                            │  (gateway)  │                         │
│                            └──────┬──────┘                         │
│                                   │                                 │
│                                   ▼                                 │
│                     GET /evidence/candles/latest                    │
│                     ?source=binancef&symbol=btcusdt&timeframe=60    │
└─────────────────────────────────────────────────────────────────────┘
```

## Endpoint

```
GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60
```

**Response (200):**
```json
{
  "candle": {
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "open": "84521.30000000",
    "high": "84589.90000000",
    "low": "84510.00000000",
    "close": "84575.40000000",
    "volume": "12345678.00000000",
    "trade_count": 87,
    "open_time": "2026-03-16T12:00:00Z",
    "close_time": "2026-03-16T12:01:00Z",
    "final": false
  }
}
```

**Response when no data yet (200):**
```json
{
  "candle": null
}
```

**Validation error (400):**
```json
{
  "code": "invalid_argument",
  "message": "timeframe must be positive"
}
```

## Smoke Scenario

### Automated

```bash
make up           # start full stack
make smoke        # run automated E2E smoke test (waits ~75s for pipeline)
```

The `make smoke` target runs `scripts/smoke-first-slice.sh`, which:
1. Checks `/healthz` (gateway alive)
2. Checks `/readyz` (configctl reachable)
3. Polls `/evidence/candles/latest` until a candle appears or timeout
4. Validates response JSON structure (all OHLCV fields present)
5. Validates error handling (missing params → 400)

### Manual

```bash
make up
sleep 75

# Health
curl -s http://127.0.0.1:8080/healthz | jq .

# Readiness
curl -s http://127.0.0.1:8080/readyz | jq .

# Evidence query — the first slice endpoint
curl -s 'http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60' | jq .

# Observe streams (requires nats CLI)
nats stream info OBSERVATION_EVENTS
nats stream info EVIDENCE_EVENTS
nats sub "observation.events.market.trade.>" --stream OBSERVATION_EVENTS
nats sub "evidence.events.candle.sampled.>" --stream EVIDENCE_EVENTS
```

### Via .http file

Open `tests/http/evidence.http` in any REST client (VS Code REST Client, IntelliJ, etc).

## Files Changed/Created

### New Files

| File | Purpose |
|------|---------|
| `internal/interfaces/http/routes/evidence_test.go` | Route registration tests (evidence wired, nil graceful) |
| `scripts/smoke-first-slice.sh` | Automated E2E smoke script |

### Modified Files

| File | Change |
|------|--------|
| `Makefile` | Added `smoke` target |
| `tests/http/evidence.http` | Expanded with full slice documentation |

## Component Ownership Summary

| Component | Binary | Layer | Responsibility |
|-----------|--------|-------|----------------|
| WebSocketAdapterActor | ingest | Actor | Connects to Binance WS, normalizes trades |
| ObservationPublisher | ingest | Adapter | Publishes to OBSERVATION_EVENTS |
| ObservationConsumer | derive | Adapter | Durable consumer on OBSERVATION_EVENTS |
| CandleSampler | derive | Application | Pure OHLCV aggregation (60s windows) |
| SamplerActor | derive | Actor | Owns sampler + answers snapshot queries |
| EvidencePublisher | derive | Adapter | Publishes to EVIDENCE_EVENTS |
| QueryResponderActor | derive | Actor | NATS request/reply for candle.latest |
| EvidenceGateway | server | Adapter | NATS client for evidence queries |
| GetLatestCandleUseCase | server | Application | Input validation + gateway delegation |
| EvidenceWebHandler | server | Interface | HTTP → use case → JSON |

## Subject/Stream Summary

| Subject | Direction | Owner |
|---------|-----------|-------|
| `observation.events.market.trade.binancef` | ingest → NATS | ingest |
| `evidence.events.candle.sampled.binancef.btcusdt.60` | derive → NATS | derive |
| `evidence.query.candle.latest` | server ↔ derive | derive (responder) |

| Stream | Owner | Retention | Size |
|--------|-------|-----------|------|
| `OBSERVATION_EVENTS` | ingest | 6h | 1 GB |
| `EVIDENCE_EVENTS` | derive | 72h | 2 GB |

## Fragility Points for S10 Hardening

1. **Candle null window** — Between boot and the first 60s window boundary crossing, the endpoint returns `{"candle": null}`. The smoke script handles this, but consumers must tolerate null.

2. **WebSocket dependency** — The entire pipeline depends on Binance Futures WS being reachable. If the exchange is down or the container has no outbound internet, the pipeline stalls silently.

3. **No health signal from ingest/derive** — Docker health checks only verify the process is alive, not that the WebSocket is connected or that events are flowing. A stale pipeline is invisible.

4. **Single sampler, single PID** — The query responder hardcodes a single sampler PID. Adding symbols requires a sampler registry.

5. **No replay on restart** — When derive restarts, the current candle window is lost. The durable consumer resumes from the last acked position, but the sampler has no state to resume from.

6. **Unbounded actor mailboxes** — Under high trade volume, the sampler's mailbox grows without bound. Hollywood's default mailbox has no backpressure.

7. **CBOR envelope overhead** — Every message is CBOR-encoded in an Envelope[T] with metadata. For high-frequency trades this adds overhead. Acceptable for the first slice.

8. **Smoke script requires python3** — The response validation step uses python3 for JSON parsing. This is standard on macOS/Linux but should be noted.

## What the First Slice Proves

1. **Layer sovereignty works** — Domain types have no I/O. Adapters own external concerns. Actors manage lifecycle. The cmd layer wires everything.

2. **NATS as the sole inter-service bus works** — JetStream for async events, request/reply for sync queries. No HTTP between services.

3. **Actor-per-concern works** — Each actor has one job. Communication via typed messages. No shared mutable state.

4. **The read model can start simple** — The sampler actor serves both writes (trade accumulation) and reads (snapshot queries) without a separate store. The path to a proper store is clean.

5. **Contract-first works** — Domain types, event specs, stream specs, and consumer specs were defined before implementation. The implementation followed the contracts.
