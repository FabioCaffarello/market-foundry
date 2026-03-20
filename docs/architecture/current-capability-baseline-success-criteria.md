# Current Capability Baseline — Success Criteria

> **Stage:** S137 — Canonical Current Capability Baseline
> **Scope:** Objective criteria for confirming the baseline is operationally healthy.

---

## 1. Purpose

This document specifies the **pass/fail criteria** that determine whether the canonical baseline (as defined in `current-capability-baseline-definition.md`) is operating correctly. These criteria are designed to be:

- **Objective** — no subjective judgment required
- **Observable** — verifiable via existing scripts, HTTP calls, or log inspection
- **Repeatable** — any operator can run them with the same expected outcome
- **Scoped to existing capability** — no new instrumentation required

---

## 2. Tier 1 — Infrastructure Health (< 2 minutes)

These criteria validate that the platform is alive and ready.

| ID | Criterion | Verification | Pass Condition |
|----|-----------|-------------|----------------|
| **B-01** | All 7 runtimes start without error | `docker compose ps` | All containers status: `Up (healthy)` |
| **B-02** | NATS JetStream is operational | NATS monitor port 8222 `/jsz` | JetStream enabled, streams listed |
| **B-03** | Gateway health endpoint responds | `GET /healthz` | HTTP 200 |
| **B-04** | Gateway readiness endpoint responds | `GET /readyz` | HTTP 200 (configctl + store reachable) |
| **B-05** | All services report healthy | `/healthz` on ports 8080–8084 | HTTP 200 on each |

**Verification script:** `scripts/live-pipeline-activate.sh` (health-check phase)

---

## 3. Tier 2 — Pipeline Activation (< 5 minutes)

These criteria validate that the data pipeline is flowing from ingestion to evidence.

| ID | Criterion | Verification | Pass Condition |
|----|-----------|-------------|----------------|
| **B-06** | Config binding accepted and activated | `POST /configctl/configs` + activate | HTTP 200/201, config state = `active` |
| **B-07** | Observation events flowing | NATS stream `OBSERVATION_EVENTS` message count | Count > 0 within 30s of activation |
| **B-08** | 60s candle produced | `GET /evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60` | HTTP 200 with valid candle JSON (OHLCV, trades, final=true) |
| **B-09** | 300s candle produced | Same endpoint, `timeframe=300` | HTTP 200 with valid candle (may require 5min wait) |
| **B-10** | TradeBurst produced | `GET /evidence/tradeburst/latest?...&timeframe=60` | HTTP 200 with valid tradeburst JSON |
| **B-11** | Volume produced | `GET /evidence/volume/latest?...&timeframe=60` | HTTP 200 with valid volume JSON |
| **B-12** | All 3 evidence families present in KV | NATS KV key inspection | Keys exist for candle, tradeburst, volume at timeframe=60 |

**Verification script:** `scripts/smoke-first-slice.sh`

---

## 4. Tier 3 — Full Causal Chain (< 20 minutes)

These criteria validate the complete vertical slice from evidence through execution.

| ID | Criterion | Verification | Pass Condition |
|----|-----------|-------------|----------------|
| **B-13** | RSI signal generated (60s TF) | `GET /signal/rsi/latest?...&timeframe=60` | HTTP 200 with valid RSI value after ~15min |
| **B-14** | RSI Oversold decision evaluated | `GET /decision/rsi_oversold/latest?...&timeframe=60` | HTTP 200 with decision result |
| **B-15** | Mean Reversion Entry strategy resolved | `GET /strategy/mean_reversion_entry/latest?...&timeframe=60` | HTTP 200 with strategy result |
| **B-16** | Position Exposure risk assessed | `GET /risk/position_exposure/latest?...&timeframe=60` | HTTP 200 with risk assessment |
| **B-17** | Paper Order execution intent submitted | `GET /execution/status/latest?...` | HTTP 200 with execution status |
| **B-18** | Venue fill event produced | Execution fill on `EXECUTION_FILL_EVENTS` stream | Fill event present (venue_market_order) |

**Note:** B-13 through B-18 depend on RSI warm-up (15 candles × 60s = ~15 minutes). This is a physics constraint, not a test issue.

**Verification script:** `scripts/smoke-multi-symbol.sh` (downstream validation sections)

---

## 5. Tier 4 — Multi-Symbol Scaling (< 25 minutes)

These criteria validate horizontal scaling by adding a second symbol.

| ID | Criterion | Verification | Pass Condition |
|----|-----------|-------------|----------------|
| **B-19** | Second symbol binding accepted | `POST /configctl/configs` with `binancef.ethusdt` | HTTP 200/201 |
| **B-20** | ethusdt candle produced (60s) | `GET /evidence/candles/latest?...&symbol=ethusdt&timeframe=60` | HTTP 200 with valid candle |
| **B-21** | ethusdt RSI generated (60s) | `GET /signal/rsi/latest?...&symbol=ethusdt&timeframe=60` | HTTP 200 after warm-up |
| **B-22** | KV key count correct | KV inspection per evidence family | 8 keys (2 symbols × 4 timeframes) |
| **B-23** | No cross-symbol interference | btcusdt queries return btcusdt data only | Symbol field matches query parameter |

**Verification script:** `scripts/smoke-multi-symbol.sh`

---

## 6. Tier 5 — TC-01 Timeframe Coverage (extended runtime)

These criteria validate that all 4 timeframes produce data.

| ID | Criterion | Verification | Pass Condition |
|----|-----------|-------------|----------------|
| **B-24** | 900s candle produced | `GET /evidence/candles/latest?...&timeframe=900` | HTTP 200 with valid candle (requires 15min runtime) |
| **B-25** | 3600s candle produced | `GET /evidence/candles/latest?...&timeframe=3600` | HTTP 200 with valid candle (requires 60min runtime) |
| **B-26** | RSI at 300s TF | `GET /signal/rsi/latest?...&timeframe=300` | HTTP 200 (requires 15×300s = 75min) |
| **B-27** | All 4 TF KV keys per family | KV inspection | 4 distinct timeframe keys per symbol per family |

**Note:** B-25 and B-26 require extended runtime (60–75 minutes). B-25 is achievable within a single operator session; B-26 may need background operation.

**Deferred from TC-01:** RSI convergence at 900s/3600s (M7/M8) requires 3.75–15 hours of runtime. This is accepted as a physics constraint and NOT a baseline failure.

---

## 7. Error Condition Criteria

These criteria validate that the system handles errors correctly.

| ID | Criterion | Verification | Pass Condition |
|----|-----------|-------------|----------------|
| **B-28** | Missing query params return 400 | `GET /evidence/candles/latest` (no params) | HTTP 400 with problem detail |
| **B-29** | Unknown family returns 404/400 | `GET /signal/unknown/latest?...` | HTTP 400 or 404 |
| **B-30** | Unwarmed signal returns null/empty | RSI query before 15 candles | HTTP 200 with null/empty result (not error) |

---

## 8. Criteria Summary

| Tier | Criteria | Time Required | Automation |
|------|----------|---------------|------------|
| Tier 1: Infrastructure | B-01 through B-05 | < 2 min | `live-pipeline-activate.sh` |
| Tier 2: Pipeline | B-06 through B-12 | < 5 min | `smoke-first-slice.sh` |
| Tier 3: Full Chain | B-13 through B-18 | < 20 min | `smoke-multi-symbol.sh` |
| Tier 4: Multi-Symbol | B-19 through B-23 | < 25 min | `smoke-multi-symbol.sh` |
| Tier 5: TC-01 Coverage | B-24 through B-27 | 15–75 min | Manual / extended smoke |
| Error Conditions | B-28 through B-30 | < 1 min | HTTP test files |

**Minimum baseline validation:** Tiers 1–3 (B-01 through B-18) — 20 minutes
**Full baseline validation:** Tiers 1–5 + Errors (B-01 through B-30) — 75 minutes

---

## 9. Pass/Fail Decision

| Result | Condition |
|--------|-----------|
| **BASELINE PASS** | All Tier 1–3 criteria pass (B-01 through B-18) |
| **BASELINE PASS (full)** | All Tier 1–5 + Error criteria pass (B-01 through B-30) |
| **BASELINE PARTIAL** | Tier 1–2 pass; Tier 3 partially passes (RSI warm-up in progress) |
| **BASELINE FAIL** | Any Tier 1 or Tier 2 criterion fails |

A **BASELINE PASS** is the minimum acceptable result for the Foundry to be considered operationally canonical.
