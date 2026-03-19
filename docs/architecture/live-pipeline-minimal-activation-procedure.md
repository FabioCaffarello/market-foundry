# Live Pipeline Minimal Activation Procedure

This document describes the canonical procedure to activate and validate the minimal live pipeline of market-foundry.

## Prerequisites

- Docker and `docker compose` available
- Ports free: `4222` (NATS), `8080` (gateway), `8222` (NATS monitor), `8123`/`9000` (ClickHouse)
- Internet access (ingest connects to Binance Futures WebSocket for live trade data)

## Quick Start

```bash
make live
```

This single command executes the full activation sequence described below.

## Step-by-Step Procedure

### Step 1: Start the Compose Stack

```bash
make up
```

This builds Docker images for all Go services and starts the compose stack in dependency order:

1. **NATS** (message broker, JetStream) — port 4222
2. **ClickHouse** (optional long-term storage) — ports 8123, 9000
3. **configctl** (configuration control plane) — internal port 8080
4. **ingest** (market data consumer) — internal port 8082
5. **derive** (evidence/signal/decision/strategy/risk sampling) — internal port 8083
6. **store** (event materialization to NATS KV) — internal port 8081
7. **execute** (venue adapter, paper simulator) — internal port 8084
8. **gateway** (HTTP API surface) — exposed port 8080

All services have health checks. The compose dependency graph ensures services start only when their dependencies are healthy.

### Step 2: Verify Service Health

```bash
make ps
```

All services should show `healthy` status. If any service is `unhealthy`, check its logs:

```bash
make logs SERVICE=<name>
```

### Step 3: Seed Configuration

```bash
make seed
```

This runs the configctl lifecycle to create and activate ingestion bindings:

1. Creates a config draft with binding `btcusdt-trades → binancef.btcusdt`
2. Validates the draft
3. Compiles the config
4. Activates the config globally

After activation, configctl publishes `IngestionRuntimeChangedEvent` which triggers ingest and derive to discover and activate the binding dynamically.

### Step 4: Wait for Pipeline Data

After seeding, the pipeline begins:

1. **ingest** connects to Binance Futures WebSocket for `btcusdt` trade stream
2. **ingest** publishes `TradeReceived` events to `OBSERVATION_EVENTS` stream
3. **derive** consumes observations and runs samplers:
   - Evidence: candle (60s, 300s), tradeburst, volume
   - Signal: RSI
   - Decision: RSI oversold
   - Strategy: mean reversion entry
   - Risk: position exposure
   - Execution: paper order intent
4. **store** consumes all domain events and materializes to NATS KV buckets
5. **execute** consumes paper order intents, applies staleness guard, submits to paper simulator, publishes fills

The first candle finalizes after ~60-120s of receiving trades.

### Step 5: Validate via Gateway

```bash
# Evidence
curl -s http://127.0.0.1:8080/evidence/candles/latest?source=binancef\&symbol=btcusdt\&timeframe=60

# Signal
curl -s http://127.0.0.1:8080/signal/rsi/latest?source=binancef\&symbol=btcusdt\&timeframe=60

# Decision
curl -s http://127.0.0.1:8080/decision/rsi_oversold/latest?source=binancef\&symbol=btcusdt\&timeframe=60

# Strategy
curl -s http://127.0.0.1:8080/strategy/mean_reversion_entry/latest?source=binancef\&symbol=btcusdt\&timeframe=60

# Risk
curl -s http://127.0.0.1:8080/risk/position_exposure/latest?source=binancef\&symbol=btcusdt\&timeframe=60

# Execution intent
curl -s http://127.0.0.1:8080/execution/paper_order/latest?source=binancef\&symbol=btcusdt\&timeframe=60

# Execution control gate
curl -s http://127.0.0.1:8080/execution/control
```

### Step 6: Observe Diagnostics

```bash
# Runtime status with tracker activity
docker compose -f deploy/compose/docker-compose.yaml exec derive \
  wget -q -O - http://127.0.0.1:8083/statusz

# Diagnostic summary with readiness checks
docker compose -f deploy/compose/docker-compose.yaml exec store \
  wget -q -O - http://127.0.0.1:8081/diagz
```

Each runtime exposes:
- `/healthz` — liveness probe (always 200 if process is alive)
- `/readyz` — readiness probe (200 only when all checks pass)
- `/statusz` — activity status: uptime, tracker event counts, idle warnings
- `/diagz` — diagnostic summary: readiness check results, tracker state

### Step 7: Run Smoke Tests

```bash
# Single symbol (btcusdt only)
make smoke

# Multi-symbol with full pipeline validation
make seed-multi
make smoke-multi
```

### Step 8: Shut Down

```bash
make down
```

## Automated Validation

The `make live` command runs all steps automatically via `scripts/live-pipeline-activate.sh`. For validating an already-running stack:

```bash
make live-check
```

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| configctl unhealthy | NATS not ready | Wait or `make restart SERVICE=configctl` |
| gateway unhealthy | store not ready | Wait for store to become healthy first |
| No candle data after 120s | ingest not connected | Check `make logs SERVICE=ingest` for WS errors |
| Null candle response | Normal before first window close | Wait for 60s window boundary to pass |
| execute idle | No paper_order events from derive | Check derive logs for execution family activation |
