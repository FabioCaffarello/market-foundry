# Timeframe Coverage Wave 01 — Definition

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S131
> **Predecessor:** S130 (Post-CC-02 Extensibility Readiness Review)
> **Status:** Defined

---

## 1. Objective

Define the first strategic expansion of timeframe coverage in market-foundry, selecting a matrix of new timeframes that validates the architecture's ability to handle increased temporal depth without structural compromise, new families, or new domains.

This is a **consolidation wave**, not a feature wave. The goal is to prove that the existing capability — already designed to treat timeframe as a first-class dimension — scales operationally when more timeframes are activated.

---

## 2. Current Coverage

| Timeframe | Seconds | Status | Since |
|-----------|---------|--------|-------|
| 1-minute  | 60      | Active | S14   |
| 5-minute  | 300     | Active | S15   |

**Activation method:** Config-driven via `deploy/configs/derive.jsonc` → `pipeline.timeframes`.

**Propagation path:** Config → `DeriveSupervisor` → `SourceScopeActor` → fan-out to `N samplers/symbol × T timeframes`. Downstream (evidence publisher, NATS streams, store projections, KV buckets, query surfaces, HTTP handlers) requires zero changes per S15 findings.

**Current actor count formula:** `N_symbols × T_timeframes` sampler actors per source scope.

---

## 3. Proposed Expansion Matrix

### 3.1 TC-01 Target Matrix

| Timeframe | Seconds | Status | Rationale |
|-----------|---------|--------|-----------|
| 1-minute  | 60      | Existing | Baseline high-frequency evidence |
| 5-minute  | 300     | Existing | First scalability proof (S15) |
| **15-minute** | **900** | **New** | Standard intraday timeframe; validates 3× multiplier from 5m |
| **1-hour** | **3600** | **New** | Standard swing timeframe; validates order-of-magnitude jump from 5m |

**Total after TC-01:** 4 timeframes (from 2).

### 3.2 Why This Matrix

**Strategic justification:**

1. **Four distinct temporal magnitudes.** 1m → 5m → 15m → 1h covers the most commonly used intraday timeframes in financial markets. Each step increases by a meaningful multiplier (5×, 3×, 4×), testing the system at different accumulation windows.

2. **Market-standard intervals.** Both 15m and 1h are universally recognized timeframes across every major exchange (Binance, CME, NYSE). Evidence derived at these intervals has immediate operational utility.

3. **Controlled combinatorial growth.** Doubling from 2→4 timeframes doubles the KV keyspace and actor count per symbol, creating measurable architectural pressure without explosion. For 2 symbols × 4 timeframes = 8 sampler actors per evidence family (vs. current 4).

4. **Well within known safe limits.** S15 documented that synchronous fan-out becomes a concern at ~10+ timeframes. TC-01 stays at 4, well within the safe operating envelope.

5. **1-hour window tests long accumulation.** A 3600s candle takes 60 minutes to finalize. This is the first timeframe that exercises genuinely long accumulation windows, testing:
   - Memory holding patterns for in-progress candles
   - Deduplication across longer windows
   - Store materialization latency for infrequent but large events
   - Signal/decision evaluation on slower-moving data

6. **No daily/weekly timeframes yet.** Daily (86400s) and weekly (604800s) timeframes introduce market-session semantics (what defines a "day" in 24/7 crypto markets?), timezone considerations, and multi-day state management. These concerns are out of scope for TC-01 and belong to a later wave.

### 3.3 Why NOT Other Choices

| Alternative | Why Excluded |
|-------------|-------------|
| 3-minute (180s) | Too close to existing 1m/5m; adds volume without architectural signal |
| 30-minute (1800s) | Redundant with 15m + 1h; adds a timeframe without testing a new magnitude |
| 4-hour (14400s) | Valid candidate for TC-02; skipped here to keep the jump from 1h → 4h as the next wave's proof |
| 1-day (86400s) | Session semantics and timezone concerns require design work beyond config change |
| 1-week (604800s) | Multi-day state management; premature for consolidation wave |
| All at once | Violates controlled expansion principle; makes failure diagnosis harder |

---

## 4. Activation Plan

### 4.1 Config Change

The entire TC-01 activation is a single config change:

```jsonc
// deploy/configs/derive.jsonc
{
  "pipeline": {
    "timeframes": [60, 300, 900, 3600]
    // ... rest unchanged
  }
}
```

### 4.2 Code Changes Expected

**Zero code changes required.** This is the core thesis of TC-01:

- S15 proved that adding a timeframe is config-only
- The evidence publisher already carries `{timeframe}` in subjects
- NATS streams use wildcards that auto-cover new timeframes
- KV stores use `{source}.{symbol}.{timeframe}` partition keys
- Query surfaces accept `timeframe` as a parameter
- HTTP handlers are type-parameterized

If any code change is required, it represents an architectural finding (a regression or gap not detected in S15).

### 4.3 Propagation Verification Points

Each new timeframe must be verified at every layer:

| Layer | Verification | Method |
|-------|-------------|--------|
| Derive | Sampler actors spawned for 900s and 3600s | Actor hierarchy log at startup |
| Evidence publish | Events on `evidence.events.candle.sampled.{source}.{symbol}.900` and `.3600` | NATS subject monitoring |
| Store projection | KV entries at `{source}.{symbol}.900` and `{source}.{symbol}.3600` | KV bucket inspection |
| Query (NATS) | Request/reply returns data for `timeframe=900` and `timeframe=3600` | NATS request test |
| Query (HTTP) | `/evidence/candles/latest?...&timeframe=900` returns data | HTTP smoke test |
| Signal | Signal events generated from 15m and 1h evidence | NATS subject monitoring |
| Decision | Decision events evaluated from 15m and 1h signals | NATS subject monitoring |
| Strategy | Strategy events resolved from 15m and 1h decisions | NATS subject monitoring |
| Risk | Risk assessments from 15m and 1h strategies | NATS subject monitoring |
| Execution | Execution intents from 15m and 1h risk assessments | NATS subject monitoring |

---

## 5. Domains and Families Affected

### 5.1 Full Pipeline Coverage

TC-01 exercises the **complete pipeline** at new timeframes. Every currently activated family must produce correct output at 900s and 3600s:

| Domain | Family | Expected Behavior |
|--------|--------|-------------------|
| Evidence | `candle` | 15m and 1h candles derived from trade stream |
| Evidence | `tradeburst` | 15m and 1h trade burst aggregations |
| Evidence | `volume` | 15m and 1h volume profiles |
| Signal | `rsi` | RSI computed from 15m and 1h candle series |
| Decision | `rsi_oversold` | Oversold evaluations on 15m and 1h RSI |
| Strategy | `mean_reversion_entry` | Entry resolution from 15m and 1h decisions |
| Risk | `position_exposure` | Exposure assessment from 15m and 1h strategies |
| Execution | `paper_order` | Paper orders from 15m and 1h risk assessments |

### 5.2 What Does NOT Change

- No new domains
- No new families
- No new NATS streams
- No new KV buckets
- No new HTTP routes
- No new actors (types) — only more instances of existing actor types

---

## 6. Resource Impact Projection

### 6.1 Actor Count Growth

| Metric | Before (2 TF) | After (4 TF) | Growth |
|--------|---------------|---------------|--------|
| Sampler actors per symbol (evidence) | 2 × 3 families = 6 | 4 × 3 families = 12 | 2× |
| Total samplers (2 symbols) | 12 | 24 | 2× |
| Total samplers (5 symbols) | 30 | 60 | 2× |

### 6.2 KV Keyspace Growth

| Metric | Before (2 TF) | After (4 TF) | Growth |
|--------|---------------|---------------|--------|
| KV keys per symbol per domain | 2 | 4 | 2× |
| Total KV keys (2 symbols, 8 domains) | 32 | 64 | 2× |

### 6.3 NATS Subject Cardinality

| Metric | Before (2 TF) | After (4 TF) | Growth |
|--------|---------------|---------------|--------|
| Unique evidence subjects per symbol | 2 × 3 = 6 | 4 × 3 = 12 | 2× |
| Total unique subjects (2 symbols) | ~32 | ~64 | 2× |

All growth is **linear and predictable**: exactly 2× for doubling timeframes from 2→4.

---

## 7. Relationship to Prior Stages

| Stage | Relationship |
|-------|-------------|
| S15 | Proved second timeframe (300s) works config-only; TC-01 extends this proof to 4 timeframes |
| S17 | Proved multi-symbol; TC-01 operates on the same symbol set |
| S109 | Vertical slice end-to-end; TC-01 validates the same pipeline at more temporal depths |
| S130 | Recommended consolidation wave; TC-01 is that wave |
| CC-01/CC-02 | Proved family extensibility; TC-01 is orthogonal (temporal depth, not family breadth) |
