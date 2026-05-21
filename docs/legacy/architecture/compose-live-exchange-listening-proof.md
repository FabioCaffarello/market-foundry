# Compose Live Exchange Listening Proof

Status: **Active** | Stage: S378 | Wave: Exchange Listening and Dry-Run Foundation (S376–S381)

## Purpose

This document records the proof that the market-foundry compose stack can connect to a real exchange, receive live market data, and flow it through the canonical NATS observation pipeline — without touching the write path (no real orders).

## Scope

- **Proven:** Live read path from Binance Futures mainnet through compose-orchestrated multi-binary pipeline.
- **Not proven:** Dry-run execution, venue adapter integration, observability expansion.

## Architecture of the Live Listening Path

```
┌────────────────────────────────────────────────────────────────┐
│  Binance Futures Mainnet                                       │
│  wss://fstream.binance.com/ws/{symbol}@aggTrade                │
└───────────────────────┬────────────────────────────────────────┘
                        │ WebSocket (public, no auth)
                        ▼
┌────────────────────────────────────────────────────────────────┐
│  ingest binary (compose service)                               │
│                                                                │
│  WSClient → ParseAggTrade → Normalize → PublisherActor         │
│  ┌──────────────────────────────────────────────────────┐      │
│  │ Contract invariants enforced:                         │      │
│  │  CI-1: mainnet only (hardcoded endpoint)              │      │
│  │  CI-2: string passthrough for price/quantity          │      │
│  │  CI-3: validation before NATS publish                 │      │
│  │  CI-4: deduplication via Msg-Id                       │      │
│  │  CI-5: no venue config read by ingest                 │      │
│  └──────────────────────────────────────────────────────┘      │
└───────────────────────┬────────────────────────────────────────┘
                        │ NATS JetStream publish
                        │ Subject: observation.events.market.trade.binancef
                        │ Stream: OBSERVATION_EVENTS
                        ▼
┌────────────────────────────────────────────────────────────────┐
│  NATS JetStream (file-backed)                                  │
│  Retention: 6h / 256MB                                         │
│  Dedup window: active (Msg-Id = source:tradeID)                │
└───────────────────────┬────────────────────────────────────────┘
                        │ Durable consumer: derive-observation
                        ▼
┌────────────────────────────────────────────────────────────────┐
│  derive binary (compose service)                               │
│  Consumes normalized trades → candle aggregation → signals     │
└────────────────────────────────────────────────────────────────┘
```

## Binding Activation Flow

The ingest binary does not hardcode which symbols to listen to. Bindings are discovered dynamically:

1. `make seed` creates a config via configctl HTTP API (draft → validate → compile → activate).
2. Activation publishes `IngestionRuntimeChangedEvent` to `CONFIGCTL_EVENTS` stream.
3. Ingest's `BindingWatcherActor` receives the event and sends `activateBindingMessage` to the supervisor.
4. Supervisor spawns `ExchangeScopeActor` → `WebSocketAdapterActor` per symbol.
5. WebSocket connects to Binance Futures mainnet and starts streaming aggTrade messages.

This means adding or removing symbols at runtime does not require a restart.

## Write Path Isolation

The execution engine runs in **paper mode** by default (CI-8: empty `venue.type` defaults to `paper_simulator`). The three-dimensional activation surface guarantees:

| AdapterState | GateStatus | CredentialState | EffectiveMode | Real orders? |
|---|---|---|---|---|
| paper | * | * | paper | NO |

No environment variables (`MF_VENUE_*`) are set in the compose config, so `CredentialState = absent`. Even if `venue.type` were set to `venue`, the system would compute `venue_degraded` — still no real orders.

## Smoke Validation

The proof is exercised by `make smoke-live-listening` which runs `scripts/smoke-live-exchange-listening.sh`.

Phases validated:

| Phase | What it checks |
|---|---|
| 1. Stack Readiness | nats, configctl, ingest, derive, gateway all healthy |
| 2. JetStream Wiring | OBSERVATION_EVENTS stream and derive-observation consumer exist |
| 3. Active Bindings | configctl has at least one active ingestion binding |
| 4. Execution Mode | Activation surface reports paper/venue_halted/venue_degraded |
| 5. WebSocket Connectivity | Ingest logs show WebSocket connection activity |
| 6. Live Trade Flow | OBSERVATION_EVENTS message count grows over polling window |
| 7. Derive Consumption | derive-observation consumer delivered count > 0 |
| 8. Write Path Isolation | No venue_live evidence in execute logs |
| 9. Ingest Health | No publisher errors in recent ingest logs |

## Operational Ergonomics

```bash
# Full lifecycle: build, start, seed, validate live listening
make up && make seed && make smoke-live-listening

# With longer polling window (default 60s)
LISTEN_WAIT=120 make smoke-live-listening

# Check already-running stack
make smoke-live-listening
```

## Limitations

1. **Network dependency:** Requires outbound connectivity to `fstream.binance.com:443`. Firewalled environments will fail Phase 6.
2. **Exchange availability:** Binance Futures must be operational. Weekend maintenance windows may affect trade volume.
3. **Single exchange:** Only Binance Futures is wired. Other exchanges would require new adapter implementations.
4. **No latency measurement:** The smoke does not measure WebSocket-to-NATS latency.
5. **No volume assertion:** The smoke checks for "at least one new trade" but does not assert throughput.
6. **Derive processing depth:** Phase 7 checks consumption, not full candle/signal generation (that requires timeframe-length observation).
