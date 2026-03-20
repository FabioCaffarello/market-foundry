# Timeframe Coverage Wave 01 — Runtime Activation and Query Surface

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S132
> **Status:** Implemented

---

## 1. Runtime Activation

### 1.1 Actor Hierarchy After TC-01

With `timeframes: [60, 300, 900, 3600]` and 2 symbols (btcusdt, ethusdt):

```
DeriveSupervisor
  └─ SourceScopeActor (binancef)
       ├─ btcusdt
       │    ├─ CandleSamplerActor(60s)
       │    ├─ CandleSamplerActor(300s)
       │    ├─ CandleSamplerActor(900s)      ← NEW
       │    ├─ CandleSamplerActor(3600s)     ← NEW
       │    ├─ TradeBurstSamplerActor(60s)
       │    ├─ TradeBurstSamplerActor(300s)
       │    ├─ TradeBurstSamplerActor(900s)  ← NEW
       │    ├─ TradeBurstSamplerActor(3600s) ← NEW
       │    ├─ VolumeSamplerActor(60s)
       │    ├─ VolumeSamplerActor(300s)
       │    ├─ VolumeSamplerActor(900s)      ← NEW
       │    └─ VolumeSamplerActor(3600s)     ← NEW
       └─ ethusdt
            └─ (same 12 samplers)
```

**Total evidence sampler actors:** 2 symbols × 4 timeframes × 3 families = **24** (was 12).

### 1.2 Actor Counts

| Actor Type | Before (2 TF) | After (4 TF) | Per Symbol |
|-----------|---------------|---------------|------------|
| CandleSamplerActor | 4 | 8 | 4 |
| TradeBurstSamplerActor | 4 | 8 | 4 |
| VolumeSamplerActor | 4 | 8 | 4 |
| RSI SignalSamplerActor | 4 | 8 | 4 |
| RSI Oversold DecisionActor | 4 | 8 | 4 |
| MeanReversionEntry StrategyActor | 4 | 8 | 4 |
| PositionExposure RiskActor | 4 | 8 | 4 |
| PaperOrder ExecutionActor | 4 | 8 | 4 |

### 1.3 Verification at Startup

Expected log signals after deployment:

```
sampler spawned source=binancef symbol=btcusdt family=candle timeframe=900
sampler spawned source=binancef symbol=btcusdt family=candle timeframe=3600
sampler spawned source=binancef symbol=btcusdt family=tradeburst timeframe=900
sampler spawned source=binancef symbol=btcusdt family=tradeburst timeframe=3600
sampler spawned source=binancef symbol=btcusdt family=volume timeframe=900
sampler spawned source=binancef symbol=btcusdt family=volume timeframe=3600
```

(Repeated for each symbol.)

---

## 2. NATS Subject Topology

### 2.1 Evidence Subjects (New)

| Subject | Timeframe | Event Frequency |
|---------|-----------|----------------|
| `evidence.events.candle.sampled.binancef.btcusdt.900` | 15-minute | 4×/hour |
| `evidence.events.candle.sampled.binancef.btcusdt.3600` | 1-hour | 1×/hour |
| `evidence.events.tradeburst.sampled.binancef.btcusdt.900` | 15-minute | 4×/hour |
| `evidence.events.tradeburst.sampled.binancef.btcusdt.3600` | 1-hour | 1×/hour |
| `evidence.events.volume.sampled.binancef.btcusdt.900` | 15-minute | 4×/hour |
| `evidence.events.volume.sampled.binancef.btcusdt.3600` | 1-hour | 1×/hour |

(Same pattern for ethusdt — 6 new subjects per symbol, 12 total.)

### 2.2 Stream Coverage

Existing NATS streams use wildcard subscriptions (e.g., `evidence.events.candle.sampled.>`). New timeframe subjects are **automatically covered** — no stream configuration changes needed.

### 2.3 Total Subject Cardinality

| Metric | Before (2 TF) | After (4 TF) |
|--------|---------------|---------------|
| Evidence subjects per symbol | 6 | 12 |
| Signal subjects per symbol | 2 | 4 |
| Decision subjects per symbol | 2 | 4 |
| Strategy subjects per symbol | 2 | 4 |
| Risk subjects per symbol | 2 | 4 |
| Execution subjects per symbol | 2 | 4 |
| **Total unique subjects (2 symbols)** | **~32** | **~64** |

---

## 3. KV Store Activation

### 3.1 New KV Keys

For each symbol, 2 new keys per KV bucket:

| Bucket | New Keys (per symbol) |
|--------|----------------------|
| `CANDLE_LATEST` | `binancef.{sym}.900`, `binancef.{sym}.3600` |
| `CANDLE_HISTORY` | `binancef.{sym}.900`, `binancef.{sym}.3600` |
| `TRADEBURST_LATEST` | `binancef.{sym}.900`, `binancef.{sym}.3600` |
| `VOLUME_LATEST` | `binancef.{sym}.900`, `binancef.{sym}.3600` |
| `SIGNAL_RSI_LATEST` | `binancef.{sym}.900`, `binancef.{sym}.3600` |
| `DECISION_RSI_OVERSOLD_LATEST` | `binancef.{sym}.900`, `binancef.{sym}.3600` |
| `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` | `binancef.{sym}.900`, `binancef.{sym}.3600` |
| `RISK_POSITION_EXPOSURE_LATEST` | `binancef.{sym}.900`, `binancef.{sym}.3600` |
| `EXECUTION_PAPER_ORDER_LATEST` | `binancef.{sym}.900`, `binancef.{sym}.3600` |

### 3.2 Write Frequency

| Timeframe | Evidence Writes/Hour (per symbol, per family) | Signal+ Writes/Hour |
|-----------|----------------------------------------------|---------------------|
| 60s | 60 | 60 (after warmup) |
| 300s | 12 | 12 (after warmup) |
| 900s | 4 | 4 (after warmup) |
| 3600s | 1 | 1 (after warmup) |

Higher timeframes write **less frequently**. Total write volume increases sub-linearly.

---

## 4. Query Surface

### 4.1 HTTP Endpoints (All Existing, No Changes)

All handlers already accept `timeframe` as a query parameter. TC-01 adds no new routes.

| Endpoint | New Valid Queries |
|----------|------------------|
| `GET /evidence/candles/latest` | `&timeframe=900`, `&timeframe=3600` |
| `GET /evidence/candles/history` | `&timeframe=900`, `&timeframe=3600` |
| `GET /evidence/tradeburst/latest` | `&timeframe=900`, `&timeframe=3600` |
| `GET /evidence/volume/latest` | `&timeframe=900`, `&timeframe=3600` |
| `GET /signal/rsi/latest` | `&timeframe=900`, `&timeframe=3600` |
| `GET /decision/rsi_oversold/latest` | `&timeframe=900`, `&timeframe=3600` |
| `GET /strategy/mean_reversion_entry/latest` | `&timeframe=900`, `&timeframe=3600` |
| `GET /risk/position_exposure/latest` | `&timeframe=900`, `&timeframe=3600` |
| `GET /execution/paper_order/latest` | `&timeframe=900`, `&timeframe=3600` |

### 4.2 NATS Request/Reply (All Existing, No Changes)

| Query Subject | Behavior |
|---------------|----------|
| `evidence.query.candle.latest` | Request with `timeframe=900` or `3600` → looks up KV by partition key |
| `evidence.query.candle.history` | Same — range queries work for all timeframes |
| `signal.query.rsi.latest` | Same |
| `decision.query.rsi_oversold.latest` | Same |
| `strategy.query.mean_reversion_entry.latest` | Same |
| `risk.query.position_exposure.latest` | Same |

---

## 5. Deduplication

Dedup key format includes timeframe: `{source}:{symbol}:{timeframe}:{open_time}`

| Timeframe | Dedup Entry Lifetime | Max Active Entries (per symbol, per family) |
|-----------|---------------------|---------------------------------------------|
| 60s | ~60s | 1 |
| 300s | ~300s | 1 |
| 900s | ~900s | 1 |
| 3600s | ~3600s | 1 |

No collision risk between timeframes. Total active dedup entries doubles from ~2 to ~4 per symbol per family.

---

## 6. Time-to-First-Data

| Timeframe | First Evidence | First Signal (RSI, period=14) | Full Pipeline |
|-----------|---------------|-------------------------------|--------------|
| 60s | ~60s | ~15 min | ~15 min |
| 300s | ~5 min | ~75 min | ~75 min |
| 900s | ~15 min | ~225 min (~3.75h) | ~225 min |
| 3600s | ~60 min | ~15 hours | ~15 hours |

**Implication:** Operational validation of the full pipeline at 3600s requires extended runtime. Evidence reachability can be validated much sooner (after first 1h candle finalizes at ~60 min).

---

## 7. Diagnostic Signals to Observe

| Signal | Where to Look | What to Expect |
|--------|--------------|----------------|
| Actor spawn count | derive startup logs | 24 evidence samplers (2 sym × 4 tf × 3 fam) |
| First 900s candle | `GET /evidence/candles/latest?...&timeframe=900` | Non-null after ~15 min |
| First 3600s candle | `GET /evidence/candles/latest?...&timeframe=3600` | Non-null after ~60 min |
| KV key population | NATS KV bucket key listing | 4 keys per symbol per bucket |
| Memory baseline | `docker stats` after 1 hour | < 2× increase vs. 2-TF baseline |
| Fan-out latency | derive logs (if instrumented) | < 10μs per routeTrade call |
