# Minimal Live Operation Checks and Invariants

Quick-reference for operating market-foundry in controlled mode. Covers what to check, what must hold, and what to do when something looks wrong.

## Health Endpoints

Every runtime exposes four HTTP endpoints:

| Endpoint | Purpose | Healthy | Unhealthy |
|----------|---------|---------|-----------|
| `GET /healthz` | Liveness | Always 200 (process alive) | Process dead (container restart) |
| `GET /readyz` | Readiness | 200 + `{"status":"ready"}` | 503 + check name + error |
| `GET /statusz` | Activity | 200 + tracker metrics | Idle warnings, zero event counts |
| `GET /diagz` | Diagnostic | 200 + readiness + trackers | Failed checks, idle components |

### Readiness Checks by Runtime

| Runtime | Check | What It Verifies |
|---------|-------|-----------------|
| configctl | NATS TCP dial (2s) | Broker reachable |
| ingest | NATS TCP dial (2s) | Broker reachable |
| derive | NATS TCP dial (2s) | Broker reachable |
| store | NATS TCP dial (2s) | Broker reachable |
| execute | NATS TCP dial (2s) | Broker reachable |
| gateway | NATS + configctl ListConfigs() | Broker + control plane reachable |

Gateway also probes evidence store non-blocking (logs warning if unavailable, does not block readiness).

## Operational Invariants

These must hold at all times during controlled operation. Violation of any invariant indicates a real problem.

### INV-1: Startup Order

Services must start in dependency order. The compose file enforces this via `depends_on` with `condition: service_healthy`.

```
NATS (healthy) → configctl → ingest, derive → store, execute → gateway
```

**Check:** `make ps` — all services show `healthy`.

### INV-2: Config Activation Before Data Flow

No data flows until configctl has an active configuration with at least one ingestion binding.

**Check:** `curl -s http://127.0.0.1:8080/configctl/configs/active` returns a non-empty config document.

### INV-3: All Streams and Durables Present

9 streams and 11 durable consumers must exist after activation:

| Stream | Durables |
|--------|----------|
| OBSERVATION_EVENTS | derive-observation |
| EVIDENCE_EVENTS | store-candle, store-tradeburst, store-volume |
| SIGNAL_EVENTS | store-signal-rsi |
| DECISION_EVENTS | store-decision-rsi-oversold |
| STRATEGY_EVENTS | store-strategy-mean-reversion-entry |
| RISK_EVENTS | store-risk-position-exposure |
| EXECUTION_EVENTS | store-execution-paper-order, execute-venue-market-order-intake |
| EXECUTION_FILL_EVENTS | store-execution-venue-market-order-fill |
| CONFIG_EVENTS | (internal, configctl) |

**Check:** raccoon-cli topology doctor (`make quality-gate` includes this).

### INV-4: Actor Shutdown Before Health Server

Actors stop before the health server drains. Reversing this order causes readiness probes to succeed while actors are still shutting down.

**Check:** Shutdown logs show `"actors stopped"` before `"shutdown complete"`.

### INV-5: Safety Gates Evaluate in Fixed Order

Execute runtime evaluates: kill switch → staleness guard → submit timeout. No gate can be bypassed or reordered.

**Check:** `skipped_halt` and `skipped_stale` counters in execute `/statusz`.

### INV-6: Graceful Degradation in Gateway

Gateway starts and serves traffic even if optional domain gateways (evidence, signal, etc.) are unavailable. Only NATS and configctl are hard dependencies.

**Check:** Gateway `/readyz` returns 200 even when store is temporarily down (with warnings in logs).

### INV-7: Tracker Activity Reflects Pipeline Health

Every pipeline actor records events via health trackers. An idle tracker (>2 min without events) triggers a warning in `/statusz` and structured logs.

**Check:** `/statusz` on each runtime — all trackers should show recent `last_event` and non-zero `event_count`.

### INV-8: Problem Type Across Boundaries

All errors crossing layer boundaries use `*problem.Problem` with taxonomy codes (`VAL_*`, `SYS_*`, `CFG_*`). Raw `error` is used only within infrastructure.

### INV-9: No init() Registration

All wiring is explicit in composition roots (`cmd/*/run.go`). No `init()` functions register global state.

### INV-10: Structured Log Key Conventions

| Key | Value |
|-----|-------|
| `runtime` | Binary name (set once at logger creation) |
| `actor` | Actor identity within engine |
| `error` | Always `"error"`, never `"err"` |
| `component` | Infrastructure component (`"healthz"`, `"nats"`) |

## Operational Checks Runbook

### After Startup

| # | Check | Command | Expected |
|---|-------|---------|----------|
| 1 | All services healthy | `make ps` | All show `healthy` |
| 2 | Gateway reachable | `curl -s http://127.0.0.1:8080/healthz` | `{"status":"ok"}` |
| 3 | Gateway ready | `curl -s http://127.0.0.1:8080/readyz` | `{"status":"ready"}` |
| 4 | Active config exists | `curl -s http://127.0.0.1:8080/configctl/configs/active` | Non-empty JSON |

### After Seed (60-120s Wait)

| # | Check | Command | Expected |
|---|-------|---------|----------|
| 5 | Candle materialized | `curl -s 'http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60'` | Non-null OHLCV |
| 6 | Signal present | `curl -s 'http://127.0.0.1:8080/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60'` | Non-null RSI |
| 7 | Decision present | `curl -s 'http://127.0.0.1:8080/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60'` | Non-null |
| 8 | Strategy present | `curl -s 'http://127.0.0.1:8080/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60'` | Non-null |
| 9 | Risk present | `curl -s 'http://127.0.0.1:8080/risk/position_exposure/latest?source=binancef&symbol=btcusdt&timeframe=60'` | Non-null |
| 10 | Execution intent | `curl -s 'http://127.0.0.1:8080/execution/paper_order/latest?source=binancef&symbol=btcusdt&timeframe=60'` | Non-null |
| 11 | Tracker activity | `/statusz` on each runtime | Non-zero event counts, no stale idle warnings |

### Automated

```bash
make live-check    # Runs checks 1-11 automatically
make smoke         # E2E single-symbol validation
```

## Troubleshooting Quick Reference

| Symptom | Check | Likely Cause | Action |
|---------|-------|-------------|--------|
| Service `unhealthy` | `make logs SERVICE=<name>` | NATS unreachable or config error | Check NATS, verify config file |
| Gateway 503 on `/readyz` | Gateway logs | configctl unreachable | `make restart SERVICE=configctl` |
| No candle after 120s | `make logs SERVICE=ingest` | WS connection failed | Check internet, Binance status |
| Null candle response | Wait | First 60s window not closed | Wait for window boundary |
| Execute idle | `make logs SERVICE=execute` | No paper_order events | Check derive execution family activation |
| Tracker idle warning | `/statusz` on affected runtime | No incoming events | Check upstream runtime, NATS streams |
| `skipped_halt` increasing | Execute `/statusz` | Kill switch activated | Check `EXECUTION_CONTROL` KV |
| `skipped_stale` increasing | Execute `/statusz` | Data too old for execution | Check derive pipeline lag |
| Quality gate failures | `make check` output | Code/structural regression | Fix before committing |

## What This Document Does Not Cover

- Multi-environment deployment (single local compose only)
- Production hardening (TLS, auth, rate limiting)
- Long-running stability (soak testing)
- Data correctness verification (OHLCV accuracy, RSI precision)
- ClickHouse write path (started but not wired)
- Live venue adapter operation (paper simulator only)
