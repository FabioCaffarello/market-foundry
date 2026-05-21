# Controlled Capability 01 — Live Validation Procedure

> Stage S121: Operational validation of CC-01 (multi-symbol live monitoring).

## Purpose

This document defines the exact procedure to validate CC-01 in live operation.
The goal is to prove that multi-symbol ingestion, derivation, materialization,
and query surfaces behave correctly under real market data — not to expand
functionality.

## Prerequisites

| Prerequisite | How to verify |
|---|---|
| Docker + docker compose available | `docker compose version` |
| No prior stack running | `make ps` shows no services, or `make down` first |
| Internet access to Binance Futures WS | `curl -s https://fapi.binance.com/fapi/v1/ping` returns `{}` |
| S120 implementation merged | `make live-multi` target exists in Makefile |

## Validation Procedure

### Phase 0: Clean Slate

```bash
make down              # ensure no leftover containers
docker volume prune -f # optional: clear stale NATS KV state
```

### Phase 1: Full Multi-Symbol Activation

```bash
make live-multi
```

This runs `scripts/live-pipeline-activate.sh --multi-symbol`, which:

1. **Builds and starts** all 8 compose services (nats, clickhouse, configctl, gateway, ingest, derive, store, execute).
2. **Waits for health** on all services (up to 120s, 5s poll).
3. **Probes readiness** via `/readyz` on every runtime (gateway external + all internal).
4. **Seeds configctl** with a 2-binding config (btcusdt + ethusdt) via `seed-configctl.sh --multi-symbol`.
5. **Validates diagnostics** (`/statusz`, `/diagz`) on configctl, ingest, derive, store, execute.
6. **Validates gateway query surface** for both symbols across all 6 domain endpoints.
7. **Waits for pipeline event flow** — polls `/evidence/candles/latest` per symbol (up to 90s).
8. **Prints tracker activity summary** for ingest, derive, store, execute.

**Expected duration:** 3–5 minutes (dominated by 60s candle window + build time).

### Phase 2: Sustained Observation (30 minutes)

After Phase 1 completes successfully, the stack must remain running for ≥30 minutes.

**During this period, observe:**

#### 2a. Tracker Activity (every 5 minutes)

```bash
make live-multi-check
```

Verify:
- All trackers show `event_count > 0` and monotonically increasing.
- `error_count == 0` on all domain trackers.
- `idle_seconds < 120` on ingest and derive trackers.
- Execute trackers show non-zero `filled` or `skipped_stale` counters.

#### 2b. Per-Symbol Query Surface (every 10 minutes)

```bash
# Evidence — both symbols, both timeframes
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60" | python3 -m json.tool
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=ethusdt&timeframe=60" | python3 -m json.tool

# Signal RSI (requires ~15 candles = ~15 min warm-up)
curl -s "http://127.0.0.1:8080/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60" | python3 -m json.tool
curl -s "http://127.0.0.1:8080/signal/rsi/latest?source=binancef&symbol=ethusdt&timeframe=60" | python3 -m json.tool

# Decision, Strategy, Risk, Execution — spot check
curl -s "http://127.0.0.1:8080/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60" | python3 -m json.tool
curl -s "http://127.0.0.1:8080/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60" | python3 -m json.tool
curl -s "http://127.0.0.1:8080/risk/position_exposure/latest?source=binancef&symbol=btcusdt&timeframe=60" | python3 -m json.tool
curl -s "http://127.0.0.1:8080/execution/paper_order/latest?source=binancef&symbol=btcusdt&timeframe=60" | python3 -m json.tool
```

Verify:
- Both symbols produce independent OHLCV data (prices differ).
- RSI values appear after ~15 minutes (value != null).
- Decision/Strategy/Risk/Execution all show non-empty responses per symbol.

#### 2c. Diagnostic Surfaces

```bash
# Gateway diagnostics
curl -s "http://127.0.0.1:8080/healthz" | python3 -m json.tool
curl -s "http://127.0.0.1:8080/readyz"  | python3 -m json.tool
curl -s "http://127.0.0.1:8080/statusz" | python3 -m json.tool
curl -s "http://127.0.0.1:8080/diagz"   | python3 -m json.tool
```

#### 2d. Log Observation

```bash
make logs SERVICE=ingest   # verify dual WS connections, no error-level entries
make logs SERVICE=derive   # verify dual-symbol processing, no panics
make logs SERVICE=execute  # verify safety gate evaluations, no crashes
make logs SERVICE=store    # verify KV writes for both symbols
```

**Watch for:**
- `level=error` entries → record as finding.
- Panic/crash restarts → record as blocking finding.
- Memory growth → check `docker stats` at t=10min and t=30min.

### Phase 3: E2E Smoke Validation

After 30 minutes of sustained operation:

```bash
make smoke-multi
```

This runs `scripts/smoke-multi-symbol.sh`, which validates:
- 2 symbols × 2 timeframes across all 6 domains (24 endpoint checks).
- Cross-symbol isolation (different OHLCV values per symbol).
- Execution control gate (kill switch cycle).
- Trace propagation (correlation_id + causation_id).

**All 22 steps must pass.**

### Phase 4: Memory Linearity Check

```bash
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}" | grep market-foundry
```

Compare at t=10min and t=30min. Memory should scale linearly (~2× single-symbol baseline),
not grow unbounded.

### Phase 5: Graceful Shutdown

```bash
make down
```

Verify:
- All services stop within 15s (graceful shutdown).
- No orphan containers remain (`docker ps -a | grep market-foundry`).

## Validation Matrix

| ID | Check | Method | Pass Criteria |
|---|---|---|---|
| A1 | Config activation | Phase 1 seed | POST returns 200, 2 bindings in active config |
| A2 | Both bindings visible | `/configctl/configs/active` | Response contains btcusdt + ethusdt |
| A3 | Ingest discovers both | Ingest logs | Two WS connections established without restart |
| P1 | Observation events flow | Ingest `/statusz` | `event_count > 0` for both symbols |
| P2 | Evidence materializes | `/evidence/candles/latest` per symbol | Non-null candle object per symbol |
| P3 | Signal computes | `/signal/rsi/latest` per symbol | Non-null after 15-min warm-up |
| P4 | Decision evaluates | `/decision/rsi_oversold/latest` per symbol | Response with outcome field |
| P5 | Strategy resolves | `/strategy/mean_reversion_entry/latest` per symbol | Response with direction field |
| P6 | Risk evaluates | `/risk/position_exposure/latest` per symbol | Response with disposition field |
| P7 | Execution produces | `/execution/paper_order/latest` per symbol | Non-empty response |
| P8 | Chain latency acceptable | Phase 2 observation | Both symbols produce data within similar timeframes |
| D1 | Healthz during multi | `/healthz` | 200 throughout |
| D2 | Readyz during multi | `/readyz` | 200 throughout |
| D3 | Statusz reflects dual activity | `/statusz` per runtime | Trackers show events for both symbols |
| D4 | Diagz checks pass | `/diagz` per runtime | All readiness checks pass |
| S1 | No crashes (30 min) | `make ps` + logs | All services healthy, no restart count > 0 |
| S2 | No error-level domain logs | `make logs` | Zero `level=error` from domain logic |
| S3 | Memory linearity | `docker stats` at t=10, t=30 | No unbounded growth |
| S4 | Zero data loss | Tracker `event_count` | Monotonically increasing |
| T1 | smoke-multi passes | `make smoke-multi` | All steps pass |
| T2 | Quality gate passes | `make quality-gate` | Exit 0 |
| T3 | Unit tests pass | `make test` | Exit 0 |

## Minimum Viable Success

**Must pass:** A1–A3, P1–P7, D1–D4, S1, S2, T1, T3.

**Desired (non-blocking):** P8, S3, S4, T2.

## Troubleshooting

| Symptom | Likely Cause | Action |
|---|---|---|
| Candle not materializing | 60s window not elapsed | Wait 90s+ after seed |
| RSI shows null | <15 candles accumulated | Wait 15+ minutes |
| ethusdt slower than btcusdt | Lower trade volume | Acceptable; verify eventually produces |
| Tracker idle warning | Binance WS disconnected | Check ingest logs for reconnect |
| Memory growing unbounded | Event leak or buffer | Record as finding, check NATS stream sizes |
