# Stage S131: Strategic Timeframe Coverage Definition — Report

> **Stage:** S131
> **Type:** Strategic Definition
> **Wave:** TC-01 (Timeframe Coverage)
> **Predecessor:** S130 (Post-CC-02 Extensibility Readiness Review)
> **Status:** Complete

---

## 1. Executive Summary

Stage S131 defines the first strategic expansion of timeframe coverage in market-foundry. The current system operates with 2 timeframes (1-minute and 5-minute). TC-01 adds 2 new timeframes: **15-minute (900s)** and **1-hour (3600s)**, bringing the total to 4.

This is a **consolidation wave**. No new domains, families, or structural changes. The entire activation is a single config change — proving that the architecture designed in S10–S15 genuinely scales along the temporal dimension without code modification.

The matrix was chosen to cover four distinct temporal magnitudes (1m/5m/15m/1h), stay within known safe operational limits, and create meaningful architectural pressure (especially 1-hour candle accumulation) without explosion.

---

## 2. Inputs

| Source | What It Provided |
|--------|-----------------|
| S15 (Second Timeframe Scalability) | Proof that adding a timeframe is config-only; zero downstream changes |
| S130 (Post-CC-02 Review) | Recommendation for consolidation wave; architecture proven for extension |
| `deploy/configs/derive.jsonc` | Current config: `"timeframes": [60, 300]` |
| `internal/shared/settings/schema.go` | Config model: `PipelineConfig.Timeframes []int` |
| Codebase analysis (153+ files) | Timeframe is a first-class dimension across all domains, contracts, partitions, KV keys, query surfaces |

---

## 3. Timeframe Matrix

### 3.1 Chosen Matrix

| Timeframe | Seconds | Status | Role in TC-01 |
|-----------|---------|--------|---------------|
| 1-minute | 60 | Existing | Regression baseline |
| 5-minute | 300 | Existing | Regression baseline |
| **15-minute** | **900** | **New** | Validates 3× step from 5m; standard intraday |
| **1-hour** | **3600** | **New** | Validates order-of-magnitude jump; tests long accumulation |

### 3.2 Justification

1. **Four distinct magnitudes** — 1m/5m/15m/1h are the four most commonly used intraday timeframes globally.
2. **Controlled growth** — Doubling from 2→4 creates exactly 2× linear growth across actors, KV keys, and subjects.
3. **Within safe limits** — S15 documented fan-out concern at ~10+ timeframes. TC-01 stays at 4.
4. **1-hour tests real duration pressure** — First timeframe where accumulation window (60 minutes) exercises long-held state.
5. **No session semantics** — Daily/weekly timeframes require timezone and market-session design. Excluded.

### 3.3 Excluded and Why

| Timeframe | Why Excluded |
|-----------|-------------|
| 3m (180s) | Too close to 1m/5m; no new architectural signal |
| 30m (1800s) | Redundant between 15m and 1h |
| 4h (14400s) | Reserved for TC-02; keeps TC-01 minimal |
| 1d (86400s) | Session semantics and timezone considerations |
| 1w (604800s) | Multi-day state management; premature |

---

## 4. Architectural Pressure Points

### 4.1 Primary Pressure

| Point | Class | Impact |
|-------|-------|--------|
| `CandleSamplerActor` at 3600s | Duration | Accumulates trades for 60-minute windows; O(trades_per_window) memory |
| `SourceScopeActor` fan-out | Volume | 4 sends per trade instead of 2; sub-microsecond per send |
| KV key cardinality | Cardinality | 2× more keys across all buckets |

### 4.2 No Meaningful Pressure

| Point | Why |
|-------|-----|
| HTTP query handlers | Already parameterized by timeframe |
| NATS request/reply | Already parameterized by timeframe |
| NATS stream topology | Wildcards auto-cover new timeframes |
| Execute runtime | Higher timeframes produce less work per unit time |
| Store write volume | 900s writes 4×/hour, 3600s writes 1×/hour — sparse |

### 4.3 Diagnostic Signals to Monitor

1. Actor spawn count at startup (expect `N × 4 × 3` evidence samplers)
2. Memory baseline after 1 hour (expect < 2× vs. 2-timeframe baseline)
3. First 15m and 1h candle correctness (OHLCV verification)
4. KV key population (4 keys per symbol per domain)
5. Signal generation at 900s and 3600s after warmup period

---

## 5. Success Criteria

### 5.1 Mandatory (13 criteria)

| # | Criterion |
|---|-----------|
| M1 | Config activates 4 timeframes without code change |
| M2 | Derive spawns correct actor count |
| M3 | Evidence events published for all 4 timeframes |
| M4 | Store materializes KV entries for all 4 timeframes |
| M5 | HTTP query returns data for all 4 timeframes |
| M6 | NATS request/reply returns data for all 4 timeframes |
| M7 | Signal pipeline processes all 4 timeframes |
| M8 | Full pipeline (decision → strategy → risk → execution) complete for all 4 timeframes |
| M9 | 15-minute candle finalizes correctly |
| M10 | 1-hour candle finalizes correctly |
| M11 | Existing 1m and 5m behavior unchanged |
| M12 | No duplicate events across timeframes |
| M13 | Historical candle store works for all 4 timeframes |

### 5.2 Diagnostic (5 criteria, informational only)

| # | Criterion |
|---|-----------|
| D1 | Memory usage delta |
| D2 | Fan-out latency |
| D3 | Time-to-first-candle for new timeframes |
| D4 | KV write frequency per timeframe |
| D5 | Signal quality at lower frequencies |

---

## 6. Out of Scope

| # | Item | Reason |
|---|------|--------|
| OS1 | New families | Temporal depth wave, not family breadth |
| OS2 | New domains | Domain model frozen |
| OS3 | Daily/weekly timeframes | Session semantics require design |
| OS4 | 4-hour timeframe | Reserved for TC-02 |
| OS5 | Per-binding timeframe overrides | Not triggered |
| OS6 | Interim candle snapshots | Dashboard concern, not coverage |
| OS7 | Actor fan-out refactoring | Triggered at ~10 TF, we're at 4 |
| OS8 | Generic sampler (CF-08) | Triggered at N=3 families, not by TF expansion |
| OS9 | Performance benchmarking | Diagnostics are informational only |
| OS10 | Signal algorithm tuning | Product decision, not architecture proof |
| OS11 | Additional symbols | Symbol set unchanged; TF is the independent variable |
| OS12 | Refactors without trigger | No refactoring unless evidence demands it |

---

## 7. Deliverables

| # | Deliverable | Path | Status |
|---|------------|------|--------|
| 1 | Timeframe Coverage Definition | `docs/architecture/timeframe-coverage-01-definition.md` | Complete |
| 2 | Success Criteria and Out of Scope | `docs/architecture/timeframe-coverage-01-success-criteria-and-out-of-scope.md` | Complete |
| 3 | Architectural Pressure Points | `docs/architecture/timeframe-coverage-01-architectural-pressure-points.md` | Complete |
| 4 | Stage Report (this document) | `docs/stages/stage-s131-strategic-timeframe-coverage-definition-report.md` | Complete |

---

## 8. Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Matrix is clear and well justified | PASS | 4 timeframes, 5 justification points, excluded alternatives documented |
| Scope remains controlled | PASS | 12 explicit out-of-scope items; zero new families/domains |
| Architectural pressure points are explicit | PASS | Pressure points mapped by runtime, binding, and query surface with classification |
| Success criteria are objective | PASS | 13 mandatory criteria with concrete verification methods |
| Base is ready for expansion without ambiguity | PASS | Activation is a single config line; verification procedure fully defined |

---

## 9. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| No new family opened | PASS — zero family additions |
| Not a product roadmap | PASS — pure architecture consolidation |
| Not all timeframes at once | PASS — exactly 2 new timeframes added (15m, 1h) |
| No refactors without trigger | PASS — no refactoring scoped |
| Out of scope is documented | PASS — 12 items explicitly excluded with rationale |

---

## 10. Preparation for S132

S132 should be the **minimal implementation and operational validation** of TC-01. Recommended structure:

### 10.1 S132 Scope

1. **Config change:** `"timeframes": [60, 300, 900, 3600]` in `derive.jsonc`
2. **Smoke test expansion:** Add 900 and 3600 endpoints to `scripts/smoke-first-slice.sh`
3. **HTTP test expansion:** Add 900 and 3600 queries to `tests/http/evidence.http`
4. **Live pipeline activation:** Deploy and run for ≥ 90 minutes (to see at least one 1h candle finalize)
5. **Verification:** Walk through all 13 mandatory criteria

### 10.2 S132 Duration Estimate

- Config + test changes: minimal (< 30 minutes of work)
- Pipeline runtime: ≥ 90 minutes minimum (for 1h candle + warmup)
- Verification: systematic walkthrough of M1–M13

### 10.3 S132 Exit Condition

All 13 mandatory criteria PASS. Any failure triggers investigation and is documented as an architectural finding, not papered over.

### 10.4 What Comes After S132

| Outcome | Next Step |
|---------|-----------|
| All 13 criteria PASS | S133: friction capture and diagnostic analysis |
| Code change required | Document as architectural gap; assess whether it invalidates S15 claims |
| Signal degenerate at 1h | Document as known limit; does not block TC-01 success |
| Memory concern at 1h window | Investigate sampler lifecycle; document finding |

---

## 11. Strategic Position

**What TC-01 proves if successful:**
- The Foundry's temporal dimension genuinely scales by configuration
- The architecture handles four distinct accumulation magnitudes
- The contracts designed in S10–S15 hold under 2× cardinality growth
- 1-hour accumulation windows are operationally viable

**What TC-01 does NOT prove:**
- Daily/weekly timeframe viability (session semantics)
- Per-binding temporal differentiation
- Signal quality across temporal granularities (product concern)
- Performance under high symbol count × high timeframe count

**Strategic direction:** TC-01 is the first step in proving that market-foundry's existing capabilities are deep, not just wide. The next evolution after TC-01 is either TC-02 (4h + daily with session semantics) or a product wave that leverages the proven temporal depth.
