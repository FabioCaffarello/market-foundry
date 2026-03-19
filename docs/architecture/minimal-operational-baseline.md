# Minimal Operational Baseline

Consolidated after S114 (live activation), S115 (operational validation), and S116 (bounded refactors).

This document defines the **stable operational floor** of market-foundry in controlled operation. Everything listed here has been exercised, validated, and is expected to work reliably. Anything not listed is either experimental or not yet proven.

## Topology

| Component | Image | Port | Role |
|-----------|-------|------|------|
| NATS | nats:2.10.18-alpine | 4222, 8222 | Message broker, JetStream, KV |
| ClickHouse | clickhouse-server:24.8.8 | 8123, 9000 | Time-series storage (started, write path not active) |
| configctl | market-foundry/configctl:dev | 8080 (internal) | Configuration control plane |
| ingest | market-foundry/ingest:dev | 8082 (internal) | Market data observation publisher |
| derive | market-foundry/derive:dev | 8083 (internal) | Evidence/signal/decision/strategy/risk/execution sampling |
| store | market-foundry/store:dev | 8081 (internal) | KV projection materialization |
| execute | market-foundry/execute:dev | 8084 (internal) | Venue adapter (paper simulator) |
| gateway | market-foundry/gateway:dev | 8080 (exposed) | HTTP API surface |

## Validated Pipeline Chain

```
ingest (Binance WS → btcusdt)
  → OBSERVATION_EVENTS
    → derive
      → EVIDENCE_EVENTS (candle-60, candle-300, tradeburst, volume)
      → SIGNAL_EVENTS (rsi)
      → DECISION_EVENTS (rsi_oversold)
      → STRATEGY_EVENTS (mean_reversion_entry)
      → RISK_EVENTS (position_exposure)
      → EXECUTION_EVENTS (paper_order)
        → execute (paper simulator)
          → EXECUTION_FILL_EVENTS (venue_market_order fill)
    → store (materializes all domain events to NATS KV)
```

**NATS topology:** 9 streams, 11 durable consumers. All validated by raccoon-cli topology doctor.

## Canonical Operations

### Startup

```bash
make live          # Full: build + start + seed + validate (~2 min)
make up            # Start stack only (no seed, no validate)
make live-check    # Validate already-running stack
```

### Seed Configuration

```bash
make seed          # Single symbol: btcusdt on binancef
make seed-multi    # Two symbols: btcusdt + ethusdt
```

Config lifecycle: draft → validate → compile → activate. Activation publishes `IngestionRuntimeChangedEvent` which ingest and derive discover dynamically. No restart required.

### Validate

```bash
make ps            # All services should show "healthy"
make live-check    # Full diagnostic validation of running stack
make smoke         # E2E smoke test (single symbol)
make smoke-multi   # E2E smoke test (multi-symbol)
```

### Shutdown

```bash
make down          # Graceful shutdown of all services
```

Shutdown sequence: SIGTERM → actor poison (10s timeout) → health server drain (5s) → process exit.

### Diagnostics

```bash
# Gateway endpoints (exposed on localhost:8080)
curl -s http://127.0.0.1:8080/healthz       # Liveness
curl -s http://127.0.0.1:8080/readyz        # Readiness

# Internal runtime diagnostics (via compose exec)
docker compose -f deploy/compose/docker-compose.yaml exec derive \
  wget -q -O - http://127.0.0.1:8083/statusz   # Tracker activity
docker compose -f deploy/compose/docker-compose.yaml exec store \
  wget -q -O - http://127.0.0.1:8081/diagz     # Diagnostic summary
```

### Quality Gate

```bash
make check         # Pre-commit fast gate
make verify        # Tests + quality gate
make check-deep    # Full validation (tests + raccoon-cli deep)
```

## Runtime Lifecycle (All Services)

Every runtime follows a fixed 6-phase lifecycle:

1. **Logger setup** — `BuildLogger()` + `slog.SetDefault()`
2. **Startup log** — `"<runtime> starting"` with context
3. **Actor engine** — terminal failure if creation fails
4. **Runtime wiring** — NATS connections, trackers, adapters
5. **Spawn supervisors + health server** — background HTTP
6. **`WaitTillShutdown()`** — SIGTERM/SIGINT → poison actors → drain health server

## Dependency Order

```
NATS
  └→ configctl
       └→ ingest (configctl + NATS)
       └→ gateway (configctl + NATS + store; evidence/signal/etc optional)
  └→ derive (NATS only)
       └→ store (derive + NATS)
       └→ execute (derive + NATS)
```

Gateway degrades gracefully: configctl and NATS are required; evidence, signal, decision, strategy, risk, execution gateways are optional (log warnings if unavailable, do not block readiness).

## Execution Safety Gates

Three gates protect execution, evaluated in fixed order:

| Gate | Source | Default | Failure Mode |
|------|--------|---------|-------------|
| Kill switch | `EXECUTION_CONTROL` KV, key `global` | Active (allow) | Fail-open (KV unavailable → allow) |
| Staleness guard | `venue.staleness_max_age` config | 120s | Fail-closed (always active) |
| Submit timeout | `venue.submit_timeout` config | 10s | Fail-closed (timeout → reject) |

Paper simulator ignores context (instant fills). Only `paper_order` venue type is validated.

## What Is Baseline Stable

- Domain layer isolation (pure business logic, no I/O)
- Actor concurrency model (Hollywood message-passing)
- Settings/config schema validation at startup
- NATS adapter layer (codec, KV, request-reply)
- Event pipeline chain (observation → fill, 8 stages)
- KV projection model (7 projections)
- Architecture governance (raccoon-cli: ~950 structural tests)
- Diagnostic surfaces (`/healthz`, `/readyz`, `/statusz`, `/diagz`)
- Config lifecycle (draft → validate → compile → activate)
- Safety gates (kill switch, staleness guard, submit timeout)
- Graceful shutdown (actor poison + health drain)
- Docker compose topology with health-driven startup order

## What Remains Experimental

| Area | Why Experimental | Trigger to Promote |
|------|-----------------|-------------------|
| ClickHouse write path | Started but not wired | When projection durability needed |
| Live venue adapters | Only paper simulator tested | When testnet integration begins |
| Multi-symbol sustained | Validated via smoke, not soak-tested | Multi-symbol production use |
| Cold-start behavior | RSI needs historical candles; untested | Formal cold-start testing |
| NATS reconnection/recovery | Not exercised under failure | Chaos/resilience testing |
| Correlation ID tracing | IDs exist in events, not in slog | When cross-runtime debugging needed |
| Hot config reactivation | Not validated under running pipeline | When config hot-reload required |
| Long-running stability | No soak test (hours/days) | Dedicated soak environment |

## Known Debts (Accepted)

| ID | Debt | Impact | Trigger |
|----|------|--------|---------|
| D1 | Execute actor lacks unit tests | Test gap | Actor test harness design |
| D2 | ~260 stale "consumer"/"validator" refs in docs | Cosmetic | Opportunistic cleanup |
| D3 | Use-case pattern not unified (20+ files) | Inconsistency, no bugs | Confusion when adding domain |
| D4 | Scripts not hardened for CI | Work locally, fragile | CI/CD pipeline setup |
| D5 | Config not parameterized per-env | Single deployment OK | Second environment |
| D6 | No distributed tracing | No collector infra | Observability stack setup |
| D7 | No per-event metrics (Prometheus) | Tracker aggregates suffice | Metrics backend setup |
| D8 | Kill switch global-only | No per-symbol halt | Per-symbol execution control |
