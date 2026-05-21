# S378: Compose Live Exchange Listening Proof — Stage Report

Wave: Exchange Listening and Dry-Run Foundation (S376–S381) | Phase 39

## Executive Summary

S378 proves that the market-foundry compose stack can connect to a real exchange (Binance Futures mainnet), receive live aggTrade market data via WebSocket, normalize it through the canonical ingestion pipeline, publish to NATS JetStream, and have it consumed by downstream binaries — all while the execution engine remains in paper mode with no real orders placed.

This stage required **no code changes to binaries**. The infrastructure built in prior stages (S372 compose wiring, S373 end-to-end pipeline, S377 ingress contracts) already supports live exchange listening. S378 adds the smoke validation script and architectural documentation that turns implicit capability into auditable proof.

## What Was Proven

### 1. Live Exchange Connectivity

The ingest binary connects to `wss://fstream.binance.com/ws/{symbol}@aggTrade` via the existing `WSClient` with exponential backoff reconnection. No code changes were needed — the endpoint has been hardcoded to mainnet since the adapter was built (CI-1).

### 2. Real Trade Flow Through NATS

Live aggTrade messages from Binance Futures are:
- Parsed from raw JSON (`ParseAggTrade`)
- Normalized to `ObservationTrade` with string-passthrough precision (CI-2)
- Validated before publish (CI-3)
- Published to `OBSERVATION_EVENTS` stream with `Msg-Id` deduplication (CI-4)

The smoke script polls the NATS monitoring API and confirms message count growth on `OBSERVATION_EVENTS` within the configured polling window.

### 3. Downstream Consumption

The `derive-observation` durable consumer on `OBSERVATION_EVENTS` receives and processes live trades. The smoke verifies the consumer's delivered count is positive, confirming the derive binary is consuming real exchange data.

### 4. Write Path Isolation

The execution engine's three-dimensional activation surface guarantees paper mode:
- `AdapterState = paper` (default from config)
- `CredentialState = absent` (no `MF_VENUE_*` env vars)
- `GateStatus = active` (default, but irrelevant — paper dominates)
- `EffectiveMode = paper` (computed, never stored — CI-6)

The smoke explicitly verifies the activation surface reports a non-live mode and checks execute logs for absence of venue_live activity.

### 5. Dynamic Binding Activation

Bindings are seeded via the configctl lifecycle (draft → validate → compile → activate) and discovered at runtime by the ingest binary's `BindingWatcherActor`. No restart is needed to start or change listening symbols.

## Files Changed

| File | Change | Purpose |
|---|---|---|
| `scripts/smoke-live-exchange-listening.sh` | **New** | 10-phase smoke script validating live exchange listening |
| `Makefile` | **Modified** | Added `smoke-live-listening` target and smoke-help entry |
| `docs/architecture/compose-live-exchange-listening-proof.md` | **New** | Architecture doc: proof structure, flow diagram, limitations |
| `docs/architecture/live-ingress-runtime-wiring-preconditions-and-limitations.md` | **New** | Preconditions, dependencies, troubleshooting guide |
| `docs/stages/stage-s378-compose-live-exchange-listening-proof-report.md` | **New** | This report |
| `docs/stages/INDEX.md` | **Modified** | Added S378 entry |
| `docs/architecture/README.md` | **Modified** | Added S378 document links |

## Evidence Summary

| Evidence | Source | Verification method |
|---|---|---|
| WebSocket connects to mainnet | `internal/adapters/exchanges/binancef/websocket.go` | Hardcoded URL, log inspection |
| Trades flow to NATS | NATS monitoring API `/jsz?streams=true` | Message count growth on OBSERVATION_EVENTS |
| Derive consumes trades | NATS monitoring API `/jsz?consumers=true` | derive-observation delivered count > 0 |
| Paper mode active | Gateway `/execution/activation/surface` | effective = paper |
| No real orders | Execute logs | Absence of venue_live references |
| Bindings activate dynamically | configctl API + NATS events | Seed script + binding watcher actor |

## Smoke Script: `make smoke-live-listening`

```bash
# Prerequisites
make up && make seed

# Run proof
make smoke-live-listening

# With longer polling window
LISTEN_WAIT=120 make smoke-live-listening
```

**10 phases:**

1. Stack Readiness (nats, configctl, ingest, derive, gateway healthy)
2. JetStream Wiring (OBSERVATION_EVENTS stream + derive-observation consumer)
3. Active Bindings (configctl has activated ingestion bindings)
4. Execution Mode (activation surface reports paper)
5. WebSocket Connectivity (ingest logs show connection activity)
6. Live Trade Flow (OBSERVATION_EVENTS message count grows)
7. Derive Consumption (derive-observation delivered count > 0)
8. Write Path Isolation (no venue_live in execute logs)
9. Ingest Health (no publisher errors)
10. Summary

## Remaining Limitations

1. **Network dependency:** Outbound connectivity to `fstream.binance.com:443` required.
2. **Single exchange:** Only Binance Futures wired.
3. **No backpressure:** WebSocket reads not paused when NATS publish is slow.
4. **No latency measurement:** WebSocket-to-NATS latency not quantified.
5. **No throughput assertion:** Smoke checks for "at least one trade," not volume.
6. **Binding clear reconciliation:** Deactivation is best-effort (activation is reliable).

## Preparation for S379

S379 (Dry-Run Execution Path) builds directly on S378:

1. **Live data is flowing.** S378 proves that `OBSERVATION_EVENTS` receives real trades. S379 can use this data to exercise the full pipeline through derive → signal → decision → strategy → execution intent.

2. **Paper mode is confirmed.** S378 proves the execution engine is in paper mode. S379 needs to verify that paper orders are generated from live-ingested data and that the paper simulator produces `EXECUTION_FILL_EVENTS`.

3. **Activation surface is queryable.** S378 validates the `/execution/activation/surface` endpoint. S379 can use the kill-switch (`PUT /execution/control`) to test halt/resume with live data flowing.

4. **Compose stack is stable.** S378 proves multi-binary runtime stability with real exchange traffic. S379 can focus on execution behavior rather than infrastructure.

Recommended S379 scope:
- Verify paper orders are generated from live-ingested trades
- Verify `EXECUTION_FILL_EVENTS` receives paper fills
- Exercise kill-switch with live data flowing
- Validate staleness guard with real timestamps (120s default)
- Document dry-run execution evidence and limitations

## Verdict

**S378: PASS.** The compose stack can listen to real exchanges, flow live market data through the canonical pipeline, and maintain write-path isolation. The infrastructure is ready for S379 dry-run execution path validation.
