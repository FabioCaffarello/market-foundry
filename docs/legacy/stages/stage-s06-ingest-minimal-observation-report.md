# Stage S06 — Ingest: Minimal Observation Pipeline

**Status:** Complete
**Date:** 2026-03-16

## Executive Summary

S06 implements the first real ingestion path for the market-foundry first slice. The `ingest` binary now connects to a Binance Futures aggTrade WebSocket stream for a single symbol (`btcusdt`), normalizes incoming trades into the canonical `ObservationTrade` domain type, and publishes `TradeReceivedEvent` envelopes to the `OBSERVATION_EVENTS` JetStream stream.

The implementation follows the layer sovereignty principle strictly: domain types are pure, adapters handle external concerns, actors manage lifecycle, and the cmd layer wires everything together.

## Files Changed/Created

### New Files

| File | Layer | Purpose |
|------|-------|---------|
| `internal/adapters/exchanges/binancef/aggtrade.go` | Adapter | Binance aggTrade JSON parser + normalization to `ObservationTrade` |
| `internal/adapters/exchanges/binancef/aggtrade_test.go` | Adapter | Unit tests for parsing and normalization |
| `internal/adapters/exchanges/binancef/websocket.go` | Adapter | WebSocket client with auto-reconnect |
| `internal/adapters/exchanges/go.mod` | Module | Go module for exchange adapters |
| `internal/adapters/nats/observation_publisher.go` | Adapter | JetStream publisher for observation events |
| `internal/actors/scopes/ingest/ingest_supervisor.go` | Actor | Root supervisor — spawns publisher + WebSocket adapter |
| `internal/actors/scopes/ingest/publisher_actor.go` | Actor | Owns NATS connection, publishes trades |
| `internal/actors/scopes/ingest/websocket_actor.go` | Actor | Owns WebSocket lifecycle, normalizes + forwards trades |
| `internal/actors/scopes/ingest/messages.go` | Actor | Actor message types |

### Modified Files

| File | Change |
|------|--------|
| `cmd/ingest/run.go` | Wired actor engine + IngestSupervisor (replaced stub) |
| `go.work` | Added `./internal/adapters/exchanges` module |

## Architectural Rationale

### Layer Flow

```
Binance WS → [adapter/exchanges/binancef] → ObservationTrade (domain)
                                                    ↓
                                           [actors/ingest/ws_actor]
                                                    ↓ publishTradeMessage
                                           [actors/ingest/publisher_actor]
                                                    ↓
                                           [adapters/nats/observation_publisher]
                                                    ↓
                                           NATS JetStream: OBSERVATION_EVENTS
                                           Subject: observation.events.market.trade.binancef
```

### Design Decisions

1. **Single exchange, single symbol** — `binancef` + `btcusdt` hardcoded in the supervisor. No generic adapter engine. This is intentional for the first slice scope.

2. **Adapter owns parsing, domain owns shape** — The `binancef` package handles JSON decoding of the raw WebSocket frame and converts it to `ObservationTrade`. The domain type validates invariants. Clean separation.

3. **Actor-per-concern** — `PublisherActor` owns the NATS connection, `WebSocketAdapterActor` owns the WebSocket lifecycle. Communication via typed messages (`publishTradeMessage`). No shared mutable state.

4. **CBOR envelopes with deduplication** — Trades are wrapped in `Envelope[TradeReceivedEvent]` with CBOR encoding. JetStream deduplication uses `{source}:{trade_id}` as the message ID, preventing duplicate processing on WebSocket reconnects.

5. **Auto-reconnect** — The WebSocket client reconnects on failure with a 3-second delay. The actor lifecycle supervises the goroutine.

6. **Subject partitioning** — Subject is `observation.events.market.trade.binancef`, extending the base with the source name. This enables future per-source consumer filtering without changing the stream.

### Contracts Used

| Contract | Source |
|----------|--------|
| `ObservationTrade` | `internal/domain/observation/trade.go` |
| `TradeReceivedEvent` | `internal/domain/observation/events.go` |
| `ObservationRegistry` | `internal/adapters/nats/observation_registry.go` |
| `Envelope[T]` | `internal/shared/envelope/envelope.go` |
| Stream: `OBSERVATION_EVENTS` | 6h retention, 1GB, file storage |

## How to Run

```bash
# Local (requires NATS running)
make build SERVICE=ingest
./bin/ingest -config deploy/configs/ingest.jsonc

# Docker Compose
make up
docker compose -f deploy/compose/docker-compose.yaml logs -f ingest
```

Expected logs:
```
level=INFO msg="ingest starting"
level=INFO msg="observation publisher started"
level=INFO msg="connecting websocket" url="wss://fstream.binance.com/ws/btcusdt@aggTrade"
level=INFO msg="websocket connected" stream="btcusdt@aggTrade"
level=INFO msg="ingest runtime started" source=binancef symbol=btcusdt stream=OBSERVATION_EVENTS
```

## Verification

1. **Binary starts** — `ingest` boots, connects NATS, creates/verifies stream.
2. **WebSocket connects** — Logs confirm connection to Binance Futures WS.
3. **Normalization** — Raw aggTrade JSON is parsed to `ObservationTrade` with validation.
4. **Publishing** — Events arrive at `observation.events.market.trade.binancef` in JetStream.
5. **Deduplication** — Same `trade_id` published twice is deduplicated by JetStream.

To verify events are flowing (with NATS CLI):
```bash
nats stream info OBSERVATION_EVENTS
nats sub "observation.events.market.trade.>" --stream OBSERVATION_EVENTS
```

## Remaining Risks

1. **No config-driven binding** — The symbol is hardcoded. S07+ should integrate with `configctl.events.config.ingestion_runtime_changed` for dynamic binding activation via BindingWatcherActor.

2. **No backpressure** — If the NATS publisher is slow, trades from the WebSocket handler are sent to the actor mailbox unboundedly. Hollywood's default mailbox is unbounded.

3. **No health check** — The ingest binary's health check in docker-compose only verifies the process is running, not that the WebSocket is connected or NATS is reachable.

4. **Goroutine in actor** — The WebSocket read loop runs in a goroutine spawned by the actor. While the context-based cancellation is clean, a crash in the goroutine before context check could be silent. The Poison fallback handles this.

5. **Single point of failure** — One WebSocket connection for one symbol. No redundancy. Acceptable for first slice.

## Recalibration Points Before S07

1. **BindingWatcherActor** — Before adding more sources/symbols, the config-driven activation path must be wired. The supervisor should react to `ingestion_runtime_changed` events rather than hardcoding sources.

2. **Volume field** — `ObservationTrade` has a `Volume` field but Binance aggTrade doesn't send volume directly (it's `price * quantity`). Currently unused. Decide whether to compute it in the adapter or leave it for derive.

3. **Error metrics** — The `reportError` pattern in the NATS adapter is stubbed. Before scaling to multiple sources, metrics collection should be wired.

4. **Reconnect strategy** — The current fixed 3-second delay should evolve to exponential backoff before production use.

5. **Derive consumer readiness** — S07 (derive) will consume from `OBSERVATION_EVENTS` using `DeriveObservationConsumer`. The durable consumer spec is already defined in `observation_registry.go`.
