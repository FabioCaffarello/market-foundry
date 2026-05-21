# Timeframe Coverage Wave 01 — Success Criteria and Out of Scope

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S131
> **Status:** Defined

---

## 1. Success Criteria

### 1.1 Mandatory Criteria (all must PASS)

| # | Criterion | Verification Method | Pass Condition |
|---|-----------|-------------------|----------------|
| M1 | Config activates 4 timeframes without code change | Deploy with `[60, 300, 900, 3600]` | System starts; all 4 timeframes active; zero Go files modified |
| M2 | Derive spawns correct actor count | Startup log inspection | `N_symbols × 4 × 3_families` sampler actors reported |
| M3 | Evidence events published for all 4 timeframes | NATS subject monitoring | Events observed on `.60`, `.300`, `.900`, `.3600` subjects for each evidence family |
| M4 | Store materializes KV entries for all 4 timeframes | KV bucket inspection | `{source}.{symbol}.{tf}` keys exist in all evidence KV buckets for tf ∈ {60, 300, 900, 3600} |
| M5 | HTTP query returns data for all 4 timeframes | Smoke test | `GET /evidence/candles/latest?...&timeframe=900` and `&timeframe=3600` return valid JSON |
| M6 | NATS request/reply returns data for all 4 timeframes | NATS request test | Reply contains correct timeframe in payload |
| M7 | Signal pipeline processes all 4 timeframes | NATS subject monitoring + KV inspection | RSI signal KV entries exist for all 4 timeframes |
| M8 | Decision → Strategy → Risk → Execution pipeline complete for all 4 timeframes | KV inspection | KV entries in each domain for all 4 timeframes |
| M9 | 15-minute candle finalizes correctly | Evidence inspection | Candle at tf=900 has correct open_time, close_time (15m window), correct OHLCV |
| M10 | 1-hour candle finalizes correctly | Evidence inspection | Candle at tf=3600 has correct open_time, close_time (1h window), correct OHLCV |
| M11 | Existing 1m and 5m behavior unchanged | Regression comparison | Same outputs at tf=60 and tf=300 as before expansion |
| M12 | No duplicate events | Dedup key inspection | Dedup keys include timeframe; no collisions across timeframes |
| M13 | Historical candle store works for all 4 timeframes | `CANDLE_HISTORY` bucket query | Range queries return candles for 900 and 3600 |

### 1.2 Diagnostic Criteria (informational, no pass/fail gate)

| # | Criterion | What It Reveals |
|---|-----------|----------------|
| D1 | Memory usage delta with 4 vs 2 timeframes | Whether long-window candles (3600s) create meaningful memory pressure |
| D2 | Fan-out latency per `routeTrade` call at 4 timeframes | Whether synchronous fan-out remains negligible |
| D3 | Time-to-first-candle for 900s and 3600s | Operational visibility: 15 minutes and 60 minutes respectively |
| D4 | KV write frequency per timeframe | 900s writes 4×/hour, 3600s writes 1×/hour — validates store handles sparse writes |
| D5 | Signal quality at lower frequencies | Whether RSI at 1h granularity produces meaningful (non-degenerate) values |

---

## 2. Limits

### 2.1 Operational Limits Accepted for TC-01

| Limit | Description | Severity |
|-------|-------------|----------|
| L1 | Global timeframe list (not per-binding) | Low — all sources share `[60, 300, 900, 3600]`; per-binding overrides remain future work |
| L2 | Synchronous fan-out at 4 timeframes | None — well within the ~10 threshold identified in S15 |
| L3 | 1-hour candle takes 60 minutes to first finalize | Accepted — this is by definition; not a bug |
| L4 | No interim snapshots for in-progress candles | Accepted — evidence-grade data is finalized only (documented since S13) |
| L5 | Signal lookback window unchanged | RSI period remains fixed; 1h RSI requires 14 candles = 14 hours of data before meaningful output |

### 2.2 Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| 1h candle memory accumulation | Low | Low | SamplerActor holds only current window; memory is O(trades_per_window) |
| Dedup key collision across timeframes | Very Low | High | Dedup key includes timeframe component: `{source}:{symbol}:{timeframe}:{open_time}` |
| Fan-out latency at 4× | Very Low | Low | 4 sequential sends per trade; negligible at sub-microsecond per send |
| Signal degenerate at low frequency | Medium | Low | Diagnostic only (D5); does not gate success |

---

## 3. Out of Scope

### 3.1 Explicitly Out of Scope for TC-01

| # | Item | Why Out of Scope |
|---|------|-----------------|
| OS1 | **New families** | TC-01 is a temporal depth wave, not a family breadth wave. No new signal, decision, strategy, risk, or execution families. |
| OS2 | **New domains** | No new domain types (e.g., portfolio, alerts). The domain model is frozen for this wave. |
| OS3 | **Daily timeframe (86400s)** | Introduces market-session semantics and timezone considerations that require design work beyond config change. Candidate for TC-02. |
| OS4 | **Weekly timeframe (604800s)** | Multi-day state management; premature for consolidation wave. |
| OS5 | **4-hour timeframe (14400s)** | Valid candidate but excluded to keep TC-01 at exactly 2 new timeframes. Reserved for TC-02. |
| OS6 | **Per-binding timeframe overrides** | S15 documented this as future work. Not triggered by TC-01 (single source is sufficient). |
| OS7 | **Interim candle snapshots** | Publishing in-progress candle state is a real-time dashboard concern, not a coverage concern. |
| OS8 | **Actor fan-out refactoring** | S15 noted concern at ~10+ timeframes. TC-01 stays at 4; no trigger. |
| OS9 | **Generic sampler actor (CF-08)** | Triggered at N=3 signal families. TC-01 does not add families; no trigger. |
| OS10 | **Map-based registry (CF-11)** | Same trigger as CF-08; not activated by temporal expansion. |
| OS11 | **Performance benchmarking** | D1/D2 diagnostics are informational. Formal benchmarking is a separate concern. |
| OS12 | **Signal algorithm tuning** | RSI parameters are fixed. Tuning per-timeframe is a product decision, not an architecture proof. |
| OS13 | **Product roadmap** | TC-01 defines a strategic proof, not a feature roadmap. No user-facing product decisions. |
| OS14 | **Additional symbols** | Symbol set remains as currently configured. Temporal depth is the independent variable. |
| OS15 | **Refactors without trigger** | No refactoring unless a real finding during TC-01 creates an evidence-based trigger. |

### 3.2 Deferred to TC-02 (Candidate Items)

| Item | Trigger for TC-02 |
|------|-------------------|
| 4-hour timeframe (14400s) | TC-01 success; 1h proves long-window accumulation |
| Daily timeframe (86400s) | TC-01 success + market-session semantics design |
| Per-binding timeframe overrides | Multi-source requirement emerges |
| Fan-out actor refactoring | Timeframe count approaches 10 |

---

## 4. Non-Objectives

To be absolutely clear about what TC-01 is **NOT**:

1. **NOT a product feature.** TC-01 does not deliver user-facing functionality. It proves architectural depth.
2. **NOT a performance optimization.** Diagnostics D1–D5 are informational. No performance work is scoped.
3. **NOT a refactoring trigger.** Unless TC-01 discovers a real gap, no refactoring is justified.
4. **NOT an exhaustive timeframe rollout.** 2 new timeframes (15m, 1h) is intentionally minimal.
5. **NOT a signal quality assessment.** Whether RSI at 1h is "good" is a product concern, not an architecture concern.
6. **NOT a precondition for anything.** TC-01 validates depth; it does not gate any ongoing work.
