# Current Capability Baseline Definition

> **Stage:** S137 — Canonical Current Capability Baseline
> **Status:** Definitive
> **Scope:** Consolidation of proven capabilities. No new features, families, or expansions.

---

## 1. Purpose

This document defines the **canonical operational baseline** of Market Foundry as it exists after TC-01 completion (S131–S136). It transforms the proven loop into a formally specified, repeatable, observable, and architecturally representative reference point.

The baseline answers one question: **what does the Foundry do today, and under what conditions is it considered operationally healthy?**

---

## 2. Baseline Runtimes

The canonical baseline comprises **7 runtime processes** and **1 infrastructure dependency**:

| Runtime | Port | Role | Baseline Status |
|---------|------|------|-----------------|
| **NATS** | 4222 / 8222 | Message bus + JetStream | Required infrastructure |
| **configctl** | 8080 | Config lifecycle (draft → validate → compile → activate) | Required control plane |
| **gateway** | 8080 | Stateless HTTP↔NATS translator | Required query surface |
| **ingest** | 8082 | Binance Futures WebSocket → observation events | Required data source |
| **derive** | 8083 | Observation → evidence → signal → decision → strategy → risk → execution | Required pipeline core |
| **store** | 8081 | NATS KV projections + query serving | Required read model |
| **execute** | 8084 | Execution intent → venue adapter → fill events | Required execution bridge |
| **ClickHouse** | 9000 / 8123 | Analytical warehouse | **Not in baseline** (optional, not exercised) |

**Canonical startup order:** NATS → configctl → gateway + ingest + derive → store → execute

---

## 3. Baseline Symbols

| Symbol | Source | Status |
|--------|--------|--------|
| `btcusdt` | `binancef` (Binance Futures) | Primary — always present |
| `ethusdt` | `binancef` (Binance Futures) | Secondary — validated in multi-symbol mode |

**Canonical single-symbol mode:** `btcusdt` only
**Canonical multi-symbol mode:** `btcusdt` + `ethusdt`

Both modes are part of the baseline. Single-symbol is the minimum; multi-symbol proves horizontal scaling.

---

## 4. Baseline Timeframes

| Seconds | Human | Candle Window | Baseline Status |
|---------|-------|---------------|-----------------|
| 60 | 1 min | 60s | Primary (original) |
| 300 | 5 min | 300s | Primary (original) |
| 900 | 15 min | 900s | TC-01 expansion (validated) |
| 3600 | 1 hour | 3600s | TC-01 expansion (validated, RSI warm-up deferred) |

**Matrix cardinality (single-symbol):** 1 symbol × 4 timeframes = 4 sampler sets per family
**Matrix cardinality (multi-symbol):** 2 symbols × 4 timeframes = 8 sampler sets per family

---

## 5. Baseline Families and Dependency Chain

### 5.1. Active Families

| Layer | Family | Activation | Depends On |
|-------|--------|------------|------------|
| Evidence | `candle` | Default (always on) | Observation events |
| Evidence | `tradeburst` | Default (always on) | Observation events |
| Evidence | `volume` | Default (always on) | Observation events |
| Signal | `rsi` | Opt-in (active) | `candle` evidence |
| Decision | `rsi_oversold` | Opt-in (active) | `rsi` signal |
| Strategy | `mean_reversion_entry` | Opt-in (active) | `rsi_oversold` decision |
| Risk | `position_exposure` | Opt-in (active) | `mean_reversion_entry` strategy |
| Execution | `paper_order` | Opt-in (active) | `position_exposure` risk |
| Execution | `venue_market_order` | Store projection only | `execute` service fills |

### 5.2. Registered But NOT in Baseline

| Layer | Family | Status | Reason |
|-------|--------|--------|--------|
| Signal | `ema_crossover` | Registered in schema | Not activated in derive.jsonc |
| Venue | `binance_futures_testnet` | Registered in schema | Requires activation gate ceremony |

These families exist in code but are **explicitly excluded from the baseline**. They are not part of the canonical loop.

### 5.3. Canonical Causal Chain

```
Binance WS → observation.events.market.trade
  → candle.sampled / tradeburst.sampled / volume.sampled    [evidence]
    → signal.events.rsi.generated                            [signal]
      → decision.events.rsi_oversold.evaluated               [decision]
        → strategy.events.mean_reversion_entry.resolved      [strategy]
          → risk.events.position_exposure.assessed            [risk]
            → execution.events.paper_order.submitted          [execution intent]
              → execution.fill.venue_market_order             [fill via paper_simulator]
```

This is the **full vertical slice** — from market data to simulated execution. Every link in this chain must be operational for the baseline to be considered healthy.

---

## 6. Baseline Query Surfaces

### 6.1. HTTP Endpoints (via Gateway)

| Endpoint | Domain | Baseline Expectation |
|----------|--------|---------------------|
| `GET /healthz` | Core | 200 always |
| `GET /readyz` | Core | 200 when configctl + store reachable |
| `GET /evidence/candles/latest` | Evidence | Returns candle for any valid symbol/timeframe |
| `GET /evidence/candles/history` | Evidence | Returns candle history with limit/range |
| `GET /evidence/tradeburst/latest` | Evidence | Returns latest trade burst |
| `GET /evidence/volume/latest` | Evidence | Returns latest volume |
| `GET /signal/rsi/latest` | Signal | Returns RSI after warm-up (≥15 candles) |
| `GET /decision/rsi_oversold/latest` | Decision | Returns decision evaluation |
| `GET /strategy/mean_reversion_entry/latest` | Strategy | Returns strategy resolution |
| `GET /risk/position_exposure/latest` | Risk | Returns risk assessment |
| `GET /execution/status/latest` | Execution | Returns execution status |

All query endpoints require parameters: `source=binancef&symbol={symbol}&timeframe={tf}`

### 6.2. NATS Event Streams

| Stream | Subject Pattern | Retention | Max Size |
|--------|----------------|-----------|----------|
| `OBSERVATION_EVENTS` | `observation.events.market.trade` | 6h | 1GB |
| `EVIDENCE_EVENTS` | `evidence.events.{family}.sampled` | 72h | 2GB |
| `SIGNAL_EVENTS` | `signal.events.{family}.generated` | 72h | 2GB |
| `DECISION_EVENTS` | `decision.events.{family}.evaluated` | 72h | 2GB |
| `STRATEGY_EVENTS` | `strategy.events.{family}.resolved` | 72h | 2GB |
| `RISK_EVENTS` | `risk.events.{family}.assessed` | 72h | 2GB |
| `EXECUTION_EVENTS` | `execution.events.{family}.submitted` | 72h | 2GB |
| `EXECUTION_FILL_EVENTS` | `execution.fill.{family}` | 72h | 2GB |

### 6.3. NATS KV Buckets

Each family materializes latest values in NATS KV with keys following the pattern:
`{source}.{symbol}.{timeframe}`

**Expected KV key count per family (multi-symbol):** 2 symbols × 4 timeframes = 8 keys

---

## 7. Baseline Operational Phases

### 7.1. Startup

1. NATS starts and becomes healthy (JetStream ready)
2. configctl starts, registers streams and KV buckets
3. gateway, ingest, derive start concurrently
4. store starts, creates consumers for all configured families
5. execute starts, creates consumer for execution intents
6. All services report `/healthz` 200 and `/readyz` 200
7. configctl receives ingestion binding(s) via `POST /configctl/configs`
8. Config is validated → compiled → activated

**Expected startup time to health:** < 30 seconds (all services healthy)
**Expected startup time to first data:** 60–75 seconds (first 1-min candle closes)

### 7.2. Activation

1. Ingestion binding activates WebSocket connection to Binance
2. Observation events begin flowing on `OBSERVATION_EVENTS`
3. Evidence samplers start accumulating trades per timeframe window
4. After first window close (60s), candle/tradeburst/volume events emit
5. After 15 candle closes (~15 min at 60s TF), RSI warm-up completes
6. Full causal chain activates: signal → decision → strategy → risk → execution

### 7.3. Steady State

- Candle events emit every 60s, 300s, 900s, 3600s respectively
- All query surfaces return fresh data within their timeframe cadence
- NATS KV keys update on each window close
- Log output shows sampler activity without errors or warnings
- Actor count remains stable (no actor leaks)
- Memory usage remains bounded (in-memory state is per-window only)

### 7.4. Shutdown

- Services receive SIGTERM
- Graceful shutdown within configured timeout (10s default)
- In-flight windows are lost (accepted limitation — no state persistence)
- NATS streams retain historical events per retention policy
- Restart recovers from last committed stream position

---

## 8. Accepted Limitations

These are known constraints that are **accepted as part of the baseline** and do NOT require resolution:

| ID | Limitation | Impact | Accepted Because |
|----|-----------|--------|------------------|
| L-01 | In-memory window state only | Crash loses current window (up to 60min at 3600s TF) | Adequate for TC-01 scope; D-01 deferred |
| L-02 | Global timeframe list (all symbols get same TFs) | No per-symbol TF customization | No heterogeneous need demonstrated |
| L-03 | No interim candle snapshots | Partial candles not observable | Simplifies publisher; final candles sufficient |
| L-04 | Aggregate-only tracking | No per-TF actor health breakdown | Adequate at 4 TFs |
| L-05 | RSI warm-up at 900s/3600s requires extended runtime | 15min TF RSI needs ~3.75h; 1h TF needs ~15h | Physics constraint, not bug |
| L-06 | Integer-only timeframe representation | No "1m" labels, only seconds | Unambiguous, machine-friendly |
| L-07 | No discovery endpoint for active TFs | Clients must know TF list | No external consumers yet |
| L-08 | Log volume scales linearly with TF×symbol cardinality | 2× symbols = 2× log lines | Inherent, not problematic at current scale |

---

## 9. What Is NOT in the Baseline

The following are **explicitly out of scope** and must not be conflated with baseline capabilities:

- **TC-02** (additional timeframes beyond [60, 300, 900, 3600])
- **New families** (ema_crossover activation, MACD, Bollinger Bands, etc.)
- **New symbols** beyond btcusdt/ethusdt
- **New sources** beyond binancef
- **ClickHouse integration** as a runtime dependency
- **Real venue execution** (binance_futures_testnet)
- **Backtest harness, alert system, or paper trading P&L tracking**
- **State persistence / WAL** for window recovery
- **Per-binding timeframe configuration**
- **Session-aware timeframes** (daily/weekly)
- **Dashboard or aggregate gateway views**

---

## 10. Diagnostic Signals

The following signals indicate baseline health:

| Signal | Where | Healthy Value |
|--------|-------|---------------|
| `/healthz` response | All services | 200 |
| `/readyz` response | gateway | 200 |
| Candle events on 60s TF | NATS stream | Regular cadence (~every 60s) |
| KV key count per evidence family | NATS KV | 4 (single) or 8 (multi-symbol) |
| RSI signal events | NATS stream | Present after ~15min warm-up (60s TF) |
| Execution fill events | NATS stream | Present after full chain activation |
| Actor count | Service logs | Stable (no growth over time) |
| Error log lines | All services | Zero in steady state |
| Memory usage | Service metrics | Bounded, no growth trend |

This document is the **single source of truth** for what Market Foundry can do today.
