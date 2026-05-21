# Execute: Operational Platform Integration

> Stage S87 — Documents how `execute` integrates into the market-foundry platform as a first-class operational citizen.

## Overview

The `execute` binary is the venue adapter runtime. It consumes execution intents from derive, applies control gates (kill switch, staleness guard), submits orders to a venue adapter (currently `paper_simulator`), and publishes fill events.

As of S87, `execute` is fully integrated into all standard platform workflows: docker-compose, Makefile build targets, smoke tests, and health monitoring.

## Docker Compose

`execute` is defined in `deploy/compose/docker-compose.yaml` following the same pattern as all other Go services:

| Property | Value |
|----------|-------|
| Image | `market-foundry/execute:dev` |
| Config | `/etc/market-foundry/execute.jsonc` |
| Port | `127.0.0.1:8084:8084` |
| Health check | `GET /readyz` (grep "ready") |
| Depends on | `nats` (service_healthy), `derive` (service_healthy) |

### Dependency Chain

```
nats → configctl
nats → ingest
nats → derive
nats + derive → store
nats + derive → execute
nats + configctl + store → gateway
```

`execute` depends on `derive` because it consumes events published by derive (`EXECUTION_EVENTS` stream). It does NOT depend on `store` — store and execute are peers that both consume from derive independently.

## Makefile

`execute` is included in `BUILDABLE_SERVICES`:

```makefile
BUILDABLE_SERVICES := configctl derive execute gateway ingest store
```

This enables:

| Target | Execute included |
|--------|-----------------|
| `make build` | Yes — builds `bin/execute` |
| `make build SERVICE=execute` | Yes — single-service build |
| `make docker-build` | Yes — builds execute Docker image |
| `make up` | Yes — starts execute with the full stack |

## Configuration

Configuration lives at `deploy/configs/execute.jsonc`:

```jsonc
{
  "log": { "level": "info", "format": "text" },
  "http": { "addr": ":8084" },
  "nats": { "enabled": true, "url": "nats://nats:4222", "request_timeout": "2s" },
  "venue": { "type": "paper_simulator" },
  "pipeline": {
    "execution_families": ["paper_order"],
    "risk_families": ["position_exposure"],
    "strategy_families": ["mean_reversion_entry"],
    "decision_families": ["rsi_oversold"],
    "signal_families": ["rsi"],
    "families": ["candle", "tradeburst", "volume"]
  }
}
```

The `venue.type` field is validated at startup. Only `paper_simulator` is approved until the venue activation gate ceremony.

## Go Workspace

`cmd/execute` is registered in `go.work` alongside all other service modules.

## Smoke Test Integration

`scripts/smoke-multi-symbol.sh` validates execute as a mandatory stack member:

- Step 16: Health checks (`/healthz`, `/readyz`, `/statusz`) — now hard failures, not soft warnings
- Step 17: Venue market order fill validation per symbol/timeframe
- Step 20: Kill switch integration with execute active
- Step 21: Trace persistence through execute chain

## Startup Lifecycle

1. Load and validate `execute.jsonc` (shared `bootstrap.LoadAndValidate`)
2. Build venue adapter via config-driven selection
3. Create health trackers (`venue-adapter`, `venue-consumer`)
4. Spawn `ExecuteSupervisor` actor:
   - Connects execution control KV store (kill switch)
   - Starts fill publisher (EXECUTION_FILL_EVENTS)
   - Spawns `VenueAdapterActor` child
   - Creates and starts `ExecutionConsumer` (paper_order intake)
5. Start health server on `:8084`
6. Block on `WaitTillShutdown`
7. Graceful shutdown: cancel heartbeat, stop health server (5s timeout)

## What Remains Outside Scope

- Real venue adapter implementation (requires activation gate ceremony)
- Horizontal scaling / multi-instance execute deployment
- Prometheus metrics exporter (current observability is via /statusz JSON)
- Execute-specific alerting rules
