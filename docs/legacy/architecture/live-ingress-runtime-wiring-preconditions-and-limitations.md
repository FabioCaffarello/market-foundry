# Live Ingress Runtime Wiring: Preconditions and Limitations

Status: **Active** | Stage: S378 | Wave: Exchange Listening and Dry-Run Foundation (S376–S381)

## Purpose

Documents the preconditions, dependencies, and known limitations for running the market-foundry compose stack with live exchange ingestion. This serves as the operational checklist for anyone activating or troubleshooting live exchange listening.

## Preconditions for Live Exchange Listening

### Infrastructure

| Precondition | How to verify | Failure mode |
|---|---|---|
| Docker and Docker Compose installed | `docker compose version` | Services won't start |
| Outbound HTTPS to `fstream.binance.com:443` | `curl -s https://fstream.binance.com/fapi/v1/ping` | WebSocket connections fail silently with backoff |
| Ports 4222, 8222, 8080 available on host | `lsof -i :4222,:8222,:8080` | Compose bind fails |
| Sufficient memory (~2GB for stack) | `docker stats` | OOM kills, especially ClickHouse |

### Services

| Service | Role in live listening | Required? |
|---|---|---|
| nats | Event backbone, JetStream persistence | Yes |
| configctl | Binding discovery and lifecycle management | Yes |
| ingest | WebSocket connection to exchange, trade normalization, NATS publish | Yes |
| gateway | HTTP API for seeding configctl and querying activation surface | Yes (for seeding) |
| derive | Consumes observation trades, produces candles/signals | Yes (for full pipeline) |
| store | KV materialization of latest values | Optional for listening-only |
| execute | Execution engine (paper mode) | Optional for listening-only |
| writer | ClickHouse analytical persistence | Optional for listening-only |
| clickhouse | Analytical database | Optional for listening-only |

### Configuration

| Config item | Location | Default | Required for live listening? |
|---|---|---|---|
| `nats.enabled` | `deploy/configs/ingest.jsonc` | `true` | Yes |
| `nats.url` | `deploy/configs/ingest.jsonc` | `nats://nats:4222` | Yes (compose DNS) |
| `venue.type` | `deploy/configs/execute.jsonc` | `paper_simulator` | No — paper is safe default |
| Active bindings | Seeded via `make seed` | None | Yes — without bindings, ingest has nothing to listen to |

### Seed Requirements

The ingest binary will not connect to any exchange until bindings are activated:

```bash
# Single symbol (btcusdt)
make seed

# Multiple symbols (btcusdt + ethusdt)
make seed-multi

# Custom symbols
SYMBOLS=btcusdt,ethusdt,solusdt ./scripts/seed-configctl.sh
```

The seed lifecycle is: draft → validate → compile → activate. Each step must succeed in order.

## Runtime Wiring Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│ Host Machine                                                        │
│                                                                     │
│  127.0.0.1:8080 → gateway (HTTP API)                                │
│  127.0.0.1:4222 → nats (client)                                     │
│  127.0.0.1:8222 → nats (monitoring / JetStream API)                 │
│  127.0.0.1:8123 → clickhouse (HTTP)                                 │
│  127.0.0.1:9000 → clickhouse (native)                               │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ market-foundry-network (bridge)                                 │ │
│ │                                                                 │ │
│ │  nats:4222          ← all services connect here                 │ │
│ │  configctl:8080     ← ingest queries bindings                   │ │
│ │  ingest:8082        ← readyz only (internal)                    │ │
│ │  derive:8083        ← readyz only (internal)                    │ │
│ │  store:8081         ← readyz only (internal)                    │ │
│ │  execute:8084       ← readyz only (internal)                    │ │
│ │  writer:8085        ← readyz only (internal)                    │ │
│ │  gateway:8080       ← host-exposed                              │ │
│ │  clickhouse:8123    ← host-exposed                              │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│ External:                                                           │
│  fstream.binance.com:443  ← ingest connects outbound (WebSocket)    │
└─────────────────────────────────────────────────────────────────────┘
```

## Known Limitations

### 1. Single Exchange Adapter

Only `binancef` (Binance Futures) is implemented. The adapter architecture supports multiple exchanges via `ExchangeScopeActor` per source, but no other adapters exist yet.

### 2. Mainnet-Only Market Data

The WebSocket endpoint is hardcoded to `wss://fstream.binance.com/ws/` (CI-1). There is no testnet switch for market data because Binance's public market data streams are free and unauthenticated. This is safe — market data observation carries no trading risk.

### 3. No Backpressure on Ingestion

If the NATS publish path blocks (JetStream full, NATS down), the WebSocket read loop will eventually buffer in memory. The current design relies on:
- NATS JetStream's 256MB / 6h retention window.
- The 5-second publish timeout in `PublisherActor`.
- Actor framework's mailbox buffering.

For sustained high-throughput ingestion, explicit backpressure (pause WebSocket reads when NATS is slow) would be needed.

### 4. Reconnection Visibility

WebSocket reconnections are logged but not surfaced as metrics. The exponential backoff (1s → 60s cap, reset after 30s stable) means a network blip causes ~1s gap. A prolonged outage caps at 60s retry intervals. The smoke script does not measure reconnection behavior.

### 5. Binding Reconciliation on Clear

When bindings are cleared (deactivated), the `BindingWatcherActor` logs the event but full reconciliation (stopping specific WebSocket actors) requires tracking which bindings are active per scope. The current implementation handles activation reliably but clearing is best-effort.

### 6. No Authentication for Market Data

Binance Futures public market data requires no API keys. This is a feature (no credential management for the read path) but means the system cannot access authenticated-only streams if they exist.

### 7. DNS Resolution Inside Compose

The ingest binary resolves `nats` via Docker's internal DNS. If the NATS container restarts, the ingest binary's existing NATS connection may break. The NATS client library handles reconnection automatically, but there may be a brief gap in trade publishing.

## Troubleshooting Guide

| Symptom | Likely cause | Fix |
|---|---|---|
| No trades after seeding | Outbound connectivity blocked | Check `curl -s https://fstream.binance.com/fapi/v1/ping` |
| Trades appear then stop | WebSocket disconnected (backoff) | Check `make logs SERVICE=ingest` for reconnection logs |
| OBSERVATION_EVENTS empty | No bindings activated | Run `make seed` and verify with `make smoke-live-listening` |
| derive-observation at 0 | Derive not consuming | Check `make logs SERVICE=derive` for consumer errors |
| Ingest crashes on start | NATS not ready | Ensure `make up` completed and NATS is healthy |
| Seed script fails | Gateway not ready | Wait for `curl ${BASE_URL}/readyz` to return 200 |

## Relationship to Other Stages

| Stage | Relationship |
|---|---|
| S377 | Formalized the ingress contracts (CI-1 through CI-12) that this proof exercises |
| S372 | Proved compose wiring (streams, consumers, boot order) — prerequisite for live listening |
| S373 | Proved end-to-end pipeline across binaries — live listening adds real exchange data |
| S379 | Next: dry-run execution path uses live-ingested data to exercise paper orders |
