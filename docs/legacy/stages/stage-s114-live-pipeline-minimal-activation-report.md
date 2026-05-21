# Stage S114 — Live Pipeline Minimal Activation

**Status:** Complete
**Predecessor:** S113 (Execute Actor Safety Hardening)
**Objective:** Activate and validate the minimal live pipeline end-to-end in a controlled environment.

## Executive Summary

S114 activates the smallest possible live version of the market-foundry pipeline, proving that the architectural mesh actually starts, connects, processes, and materializes data. The full chain — from live Binance trade ingestion through evidence/signal/decision/strategy/risk/execution sampling to KV materialization and HTTP query — is exercised with a single symbol (`btcusdt`) on the paper simulator venue adapter.

No new features were introduced. No architectural changes were made. The focus is strictly on validating that the existing mesh operates end-to-end in a live environment.

## What Was Done

### 1. Live Pipeline Activation Script

Created `scripts/live-pipeline-activate.sh` — an orchestrator that automates the full activation and validation sequence:

1. Starts compose stack (`docker compose up -d --build`)
2. Waits for all 7 services to become healthy (with timeout)
3. Validates internal readiness probes (`/readyz`) on all runtimes
4. Seeds configctl with single-symbol ingestion binding
5. Validates diagnostic surfaces (`/statusz`, `/diagz`) across all runtimes
6. Validates gateway query surface for all domain endpoints
7. Waits for first evidence materialization (candle in KV)
8. Reports tracker activity summary across all runtimes

Supports `--skip-build` (reuse images) and `--check-only` (validate running stack without restart).

### 2. Makefile Targets

Added two targets:
- `make live` — full activation sequence (build + start + seed + validate)
- `make live-check` — validate an already-running stack

### 3. Architecture Documentation

- `docs/architecture/live-pipeline-minimal-activation-procedure.md` — step-by-step activation guide with troubleshooting table
- `docs/architecture/live-pipeline-minimal-activation-scope.md` — exact scope definition: what's activated, what's validated, what's out of scope

## Pipeline Activated

```
ingest (Binance WS, btcusdt)
  → OBSERVATION_EVENTS
    → derive (candle/tradeburst/volume/rsi/rsi_oversold/mean_reversion_entry/position_exposure/paper_order)
      → EVIDENCE_EVENTS / SIGNAL_EVENTS / DECISION_EVENTS / STRATEGY_EVENTS / RISK_EVENTS / EXECUTION_EVENTS
        → store (KV projections for all families)
        → execute (paper_simulator) → EXECUTION_FILL_EVENTS → store
          → gateway (HTTP query surface for all domains)
```

### Runtimes Validated
| Runtime | Health | Readiness | Diagnostics |
|---------|--------|-----------|-------------|
| NATS | `/healthz` | JetStream active | 8 streams, durable consumers |
| configctl | `/healthz` | `/readyz` | `/statusz`, `/diagz` |
| ingest | `/healthz` | `/readyz` | `/statusz`, `/diagz` |
| derive | `/healthz` | `/readyz` | `/statusz`, `/diagz` |
| store | `/healthz` | `/readyz` | `/statusz`, `/diagz` |
| execute | `/healthz` | `/readyz` | `/statusz`, `/diagz` |
| gateway | `/healthz` | `/readyz` | HTTP routes |

### Query Surface Validated
| Endpoint | Status |
|----------|--------|
| `GET /healthz` | 200 |
| `GET /readyz` | 200 |
| `GET /configctl/configs/active` | 200 |
| `GET /evidence/candles/latest` | 200 |
| `GET /evidence/tradeburst/latest` | 200 |
| `GET /evidence/volume/latest` | 200 |
| `GET /signal/rsi/latest` | 200 |
| `GET /decision/rsi_oversold/latest` | 200 |
| `GET /strategy/mean_reversion_entry/latest` | 200 |
| `GET /risk/position_exposure/latest` | 200 |
| `GET /execution/paper_order/latest` | 200 |
| `GET /execution/control` | 200 |

## Files Changed

| File | Change |
|------|--------|
| `scripts/live-pipeline-activate.sh` | **New** — orchestrator script |
| `Makefile` | Added `live` and `live-check` targets |
| `docs/architecture/live-pipeline-minimal-activation-procedure.md` | **New** — activation procedure |
| `docs/architecture/live-pipeline-minimal-activation-scope.md` | **New** — scope definition |
| `docs/stages/stage-s114-live-pipeline-minimal-activation-report.md` | **New** — this report |

## Scope Validated

- Startup: all services start in correct dependency order
- Health: all health checks pass
- Readiness: all readiness probes return `ready`
- Config activation: full lifecycle (draft → validate → compile → activate)
- Event flow: live trades → observation → evidence → signal → decision → strategy → risk → execution → fill
- Materialization: all KV buckets populated via store projections
- Query surface: all gateway HTTP endpoints reachable and returning structured responses
- Diagnostics: `/statusz` and `/diagz` on all runtimes report tracker activity

## Limits and Simplifications

- **Single symbol only** in minimal activation (btcusdt); multi-symbol validated separately via `make smoke-multi`
- **Paper simulator only** — no live venue adapter
- **No long-running stability test** — startup + initial flow validated, not sustained operation
- **No data correctness verification** — structural validation, not OHLCV accuracy
- **ClickHouse present but passive** — started for compose completeness, no write path exercised
- **No TLS, auth, or rate limiting** — local development topology only

## Preparation for S115

The live pipeline is now proven operational at minimal scope. Recommended next steps:

1. **Operational hardening under sustained load** — run the pipeline for hours, observe memory, idle warnings, error accumulation
2. **Hot config reactivation** — validate changing symbols/bindings while pipeline is running
3. **Multi-symbol concurrent validation** — prove isolation between symbol pipelines under load
4. **Graceful degradation testing** — NATS disconnection, service restart recovery, consumer replay
5. **Observability export** — Prometheus metrics or OTLP traces for external monitoring

The choice of S115 should be guided by which of these provides the next highest-confidence proof of operational readiness.
