# TC-01 Frictions and Findings — Timeframe-Driven Friction Capture

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S134 (Friction Capture)
> **Date:** 2026-03-19
> **Matrix:** [60, 300, 900, 3600] seconds
> **Symbols:** btcusdt, ethusdt

---

## 1. Purpose

This document captures, classifies, and evaluates every friction point revealed by the TC-01 temporal expansion from 2 to 4 timeframes. Each finding is grounded in evidence from the S132 implementation, S133 validation, and direct code analysis — not in abstract speculation.

The goal is to distinguish clearly between:

| Classification | Definition |
|---------------|-----------|
| **Bug** | Incorrect behavior that produces wrong results |
| **Operational Fragility** | Correct behavior that breaks under plausible operational conditions |
| **Acceptable Boilerplate** | Repetition that is annoying but carries no structural cost |
| **Structural Debt** | Design choice that increases cost of the next expansion step |
| **Real Refactor Trigger** | Friction that justifies immediate architectural change |

---

## 2. Config / Activation Frictions

### F-01: Global Timeframe List (Not Per-Binding)

- **Classification:** Structural Debt
- **Evidence:** `derive.jsonc` has a single `pipeline.timeframes` array applied to all sources and symbols. `SourceScopeActor` (`source_scope_actor.go:293-298`) iterates this global list for every symbol.
- **Impact at TC-01 (4 TFs):** None. All symbols benefit from the same 4-timeframe coverage.
- **Impact at TC-02+ (8+ TFs):** If different symbols need different timeframe sets (e.g., BTC needs 1h/4h/daily, a low-volume alt needs only 1m/5m), the current design spawns unnecessary actors and produces empty candles for unused timeframes.
- **Severity:** Low now. Becomes Medium if symbol count grows beyond ~5 with heterogeneous timeframe needs.
- **Verdict:** Accepted limit. Documented as S131 L1. No refactor justified yet — the "unnecessary actor" cost is negligible at current scale. Re-evaluate if binding-specific overrides become a real requirement.

### F-02: No Timeframe Validation in Config

- **Classification:** Operational Fragility
- **Evidence:** `PipelineConfig.TimeframeDurations()` (`schema.go:185-196`) silently drops values ≤ 0 and falls back to `[60s]` if the list is empty. There is no validation for:
  - Duplicate timeframes (e.g., `[60, 60, 300]`)
  - Non-standard values (e.g., `[7, 42, 999]`)
  - Unreasonably large values (e.g., `[604800]` = 1 week)
  - Non-sorted order (though order doesn't matter for correctness)
- **Impact:** An operator typo in `derive.jsonc` could silently spawn actors for meaningless timeframes. The system would produce correct but useless candles at the misconfigured timeframe.
- **Severity:** Low. The failure mode is waste, not corruption.
- **Verdict:** Worth a small validation pass (reject duplicates, reject < 10s and > 86400s). Not a refactor trigger — a 10-line enhancement.

### F-03: Timeframe Representation as Integer Seconds

- **Classification:** Acceptable Boilerplate
- **Evidence:** Config uses `[]int` for timeframes. HTTP queries require `?timeframe=60`, not `?timeframe=1m`. NATS subjects use seconds: `evidence.events.candle.sampled.binancef.btcusdt.60`. KV keys: `binancef.btcusdt.60`.
- **Impact:** Human-readable labels (1m, 5m, 15m, 1h) are absent from all surfaces. Operators must know that `900` means 15 minutes.
- **Severity:** Cosmetic. The integer representation is unambiguous and machine-friendly.
- **Verdict:** Acceptable. Human-readable labels could be added to diagnostic output (e.g., `/statusz`, smoke test logs) without changing the underlying representation. Not a refactor trigger.

---

## 3. Diagnostics Frictions

### F-04: Single Tracker for Evidence Publisher

- **Classification:** Structural Debt
- **Evidence:** The derive runtime registers a single `evidence-publisher` tracker. Per-timeframe breakdown depends on custom counters being emitted (`validation-findings.md` §4.4). If the publisher does not increment per-timeframe counters, `/statusz` shows aggregate counts only.
- **Impact at TC-01 (4 TFs):** Adequate. The S133 enhancement to `live-pipeline-activate.sh` Phase 8 extracts per-timeframe counter totals when available.
- **Impact at TC-02+ (8+ TFs):** Aggregate-only visibility makes it hard to diagnose which timeframe is stalled or producing unexpected volume.
- **Severity:** Low now. Becomes Medium at 8+ timeframes.
- **Verdict:** Structural debt worth tracking. A per-timeframe tracker split would improve operational visibility but is not justified at 4 TFs. Candidate for TC-02 preparation.

### F-05: No Per-Timeframe Idle Detection

- **Classification:** Operational Fragility
- **Evidence:** The idle heartbeat monitor (`30s interval, 2min threshold` per `validation-findings.md` §4.4) operates at the component level. A stalled 3600s sampler would not trigger the idle alert until 2 minutes of inactivity — but a 3600s sampler is legitimately idle for up to 60 minutes between candle finalizations.
- **Impact:** The idle detection threshold is designed for 60s candles. A 3600s sampler that genuinely stalls is indistinguishable from one that is correctly waiting for its window to close.
- **Severity:** Medium. This is a real diagnostic gap, though the blast radius is limited (stale 1h candle, not system failure).
- **Verdict:** Operational fragility worth noting. The fix is not to adjust the idle threshold (which would mask 60s issues) but to add timeframe-aware health reporting. Candidate for a targeted diagnostic enhancement.

### F-06: Log Verbosity Scaling

- **Classification:** Acceptable Boilerplate
- **Evidence:** Each sampler logs at startup and at candle finalization. With 24 evidence samplers (2 symbols × 4 TFs × 3 families), startup logs grow linearly. Phase 8 of `live-pipeline-activate.sh` scans logs for errors.
- **Impact:** Startup log output is 2× what it was at 2 TFs. Still well within readable bounds.
- **Severity:** Negligible at current scale.
- **Verdict:** Acceptable. Log volume grows linearly with actor count, as expected. No action needed.

---

## 4. Query Ergonomics Frictions

### F-07: No "List Available Timeframes" Endpoint

- **Classification:** Structural Debt
- **Evidence:** All query endpoints (`/evidence/candles/latest`, `/signal/rsi/latest`, etc.) require the caller to know which timeframes are active. There is no endpoint that returns the list of configured timeframes or indicates which timeframes have materialized data.
- **Impact:** An operator querying `?timeframe=3600` for a newly started pipeline gets a null response with no indication of whether 3600s is configured, whether data is still warming up, or whether the timeframe doesn't exist.
- **Severity:** Low for internal use (operators know the config). Medium for any external consumer.
- **Verdict:** Structural debt. A `/pipeline/status` or `/evidence/available` endpoint would reduce operational friction. Not a refactor trigger — a new endpoint addition when needed.

### F-08: Null Response Ambiguity

- **Classification:** Operational Fragility
- **Evidence:** HTTP handlers return `{"candle": null}` (HTTP 200) when no data exists for a timeframe. This is the same response for:
  - Timeframe exists, data not yet materialized (waiting for first window)
  - Timeframe does not exist in config (never will produce data)
  - Timeframe existed, data expired from KV TTL
- **Impact:** The 200-with-null response is technically correct but operationally ambiguous. The operator cannot distinguish "wait longer" from "misconfigured" without checking the config separately.
- **Severity:** Low. Query surfaces are internal and operators have config access.
- **Verdict:** Operational fragility. Could be improved with a `"status": "awaiting_first_window"` vs `"status": "not_configured"` distinction. Low priority — does not affect correctness.

### F-09: HTTP Test File Duplication

- **Classification:** Acceptable Boilerplate
- **Evidence:** `tests/http/evidence.http` contains 4 blocks (one per timeframe) for each endpoint. Signal, decision, strategy, risk HTTP test files follow the same pattern. Adding a 5th timeframe requires adding blocks to 5 test files.
- **Impact:** Test file maintenance grows linearly with timeframe count. Each new timeframe adds ~20 lines across all HTTP test files.
- **Severity:** Negligible. These are manual test files, not automated test code.
- **Verdict:** Acceptable boilerplate. The repetition provides value (each block is independently executable and documented). No action needed.

---

## 5. Cardinality / Operational Frictions

### F-10: Actor Count Growth

- **Classification:** Acceptable Boilerplate
- **Evidence:** `source_scope_actor.go` spawns `N_symbols × N_timeframes × N_families` actors. At TC-01: 2 × 4 × 3 = 24 evidence samplers (was 12 at 2 TFs). Plus 8 signal, 8 decision, 8 strategy, 8 risk, 8 execution actors = ~64 total per source scope.
- **Impact:** 64 actors is trivially within Hollywood engine capacity (tested to thousands). Memory overhead per actor is negligible (single accumulator + mailbox).
- **Severity:** None at current scale.
- **Verdict:** Not a friction. Linear growth is by design and well within capacity.

### F-11: KV Key Cardinality

- **Classification:** Acceptable Boilerplate
- **Evidence:** Each evidence family produces 1 latest key per (source, symbol, timeframe). At TC-01: 1 × 2 × 4 × 3 = 24 evidence latest keys. NATS KV handles millions of keys.
- **Impact:** None measurable.
- **Verdict:** Not a friction.

### F-12: NATS Subject Cardinality

- **Classification:** Acceptable Boilerplate
- **Evidence:** ~64 unique subjects at TC-01 (was ~32 at 2 TFs). NATS handles millions of subjects.
- **Impact:** None measurable.
- **Verdict:** Not a friction.

---

## 6. Recovery / Expectations Frictions

### F-13: 3600s Window State Duration

- **Classification:** Structural Debt
- **Evidence:** `CandleSamplerActor` accumulates trades for the full window duration. A 3600s sampler holds in-progress state for up to 60 minutes. If the derive runtime crashes at minute 59, the accumulated 59 minutes of trades are lost.
- **Impact:** On restart, the 3600s sampler starts accumulating from scratch. The first post-restart candle is incomplete (represents only the trades received after restart, not the full window).
- **Severity:** Medium. This is the highest-impact operational concern revealed by TC-01.
- **Verdict:** Structural debt. The fix (interim state snapshots / WAL for in-progress candles) was explicitly scoped out in S131 L4. The cost is proportional to timeframe duration — a 60s crash loses at most 60s of data, but a 3600s crash can lose up to 60 minutes. Re-evaluate when TC-02 considers 4h+ timeframes.

### F-14: Signal Warmup Latency at High Timeframes

- **Classification:** Accepted Limitation (Not a Friction)
- **Evidence:** RSI-14 at 3600s requires 14 candles × 1 hour = ~14-15 hours of live data. This is an inherent property of the indicator, not an architectural deficiency.
- **Impact:** After a full restart, the 3600s signal pipeline is effectively non-functional for 15 hours.
- **Severity:** N/A — physics constraint.
- **Verdict:** Not a friction. Documented as S131 L5. Would only become actionable if the system supported signal state persistence/restore, which is out of scope.

### F-15: No Interim Candle Snapshots

- **Classification:** Structural Debt (related to F-13)
- **Evidence:** Query for 3600s candle returns null until the window closes. During the 60-minute accumulation window, no data is visible to the query surface. Confirmed in S131 L4.
- **Impact:** Operators monitoring 1h timeframes see no data updates for up to 60 minutes. This creates a "dead zone" in observability.
- **Severity:** Low at TC-01. Becomes Medium at TC-02 (4h candles = 4-hour dead zones).
- **Verdict:** Structural debt. An "in-progress candle" projection (with `final=false`) would solve this but adds complexity. Not justified at TC-01.

---

## 7. Documentation / Playbook Frictions

### F-16: Smoke Test Wait Time Assumptions

- **Classification:** Operational Fragility
- **Evidence:** `smoke-first-slice.sh` uses a default `WAIT_SECONDS=75`. This is calibrated for 60s candle finalization. The script validates 900s and 3600s endpoint reachability (200 response) but cannot validate data correctness without waiting 15-60 minutes.
- **Impact:** The smoke test gives a false sense of completeness — it proves wiring but not data at higher timeframes.
- **Severity:** Low. The three-tier validation procedure (S133) explicitly documents this gap and provides Tier 2/3 procedures for extended validation.
- **Verdict:** Acceptable. The smoke test's scope is correctly limited to wiring validation. The tier system handles the rest.

### F-17: No Runbook for Post-Crash Recovery at High Timeframes

- **Classification:** Operational Fragility
- **Evidence:** No documented procedure for "derive crashed at minute 45 of a 1h candle — what do I do?"
- **Impact:** An operator facing this scenario has no guidance on expected data loss or recovery timeline.
- **Severity:** Low (single operator, development context). Would be Medium in production.
- **Verdict:** Worth a brief addition to the validation procedure. Not a code change.

---

## 8. Configctl / Gateway Ergonomics Frictions

### F-18: No Timeframe Awareness in Configctl

- **Classification:** Acceptable Boilerplate
- **Evidence:** `configctl` manages ingestion bindings (source + symbol pairs). Timeframes are configured separately in `derive.jsonc`. Configctl has no concept of timeframes — it doesn't know which timeframes are active.
- **Impact:** There is no single control plane call that returns the full activation matrix (bindings × timeframes). The operator must cross-reference configctl response with derive config.
- **Severity:** Low. The separation is intentional (bindings define *what* to ingest; timeframes define *how* to derive).
- **Verdict:** Not a friction — correct separation of concerns. If a unified "pipeline status" view becomes needed, it belongs in the gateway, not configctl.

### F-19: Gateway Has No Aggregate View

- **Classification:** Structural Debt
- **Evidence:** Gateway routes individual domain queries (`/evidence/candles/latest?...`). There is no aggregate endpoint like `/pipeline/overview` that shows which symbols × timeframes × families have materialized data.
- **Impact:** Operational visibility requires N individual queries (one per domain × symbol × timeframe combination). At TC-01 that's 48 queries for a full check.
- **Severity:** Low. `live-pipeline-activate.sh` Phase 6 automates this. Manual querying is tedious but feasible.
- **Verdict:** Structural debt. A pipeline overview endpoint would improve operational ergonomics. Low priority — the script-based approach works.

---

## 9. Items That Did NOT Confirm as Problems

These items were identified as potential friction points in S131 but were not confirmed during S132/S133:

### Non-Friction NF-01: NATS Stream Pressure

- **Anticipated concern:** Doubling subjects might increase consumer lag.
- **Finding:** NATS handles millions of subjects. 64 subjects is zero operational pressure. Consumer lag is dominated by event processing time, not subject count.

### Non-Friction NF-02: Fan-Out Latency

- **Anticipated concern:** Iterating 4 timeframes in `routeTrade()` instead of 2 might introduce measurable latency.
- **Finding:** In-process actor messaging at sub-microsecond per send. 4× vs 2× adds ~1-2μs. Negligible.

### Non-Friction NF-03: KV Write Contention

- **Anticipated concern:** More KV keys might create write contention.
- **Finding:** Higher timeframes actually write *less* frequently (900s: 4/hour, 3600s: 1/hour). Total write load increases < 30% while key count doubles. NATS KV handles this trivially.

### Non-Friction NF-04: Dedup Key Collision

- **Anticipated concern:** More timeframes might create dedup collisions.
- **Finding:** Dedup keys embed timeframe: `{source}:{symbol}:{timeframe}:{open_time}`. Each timeframe has a mathematically distinct key space. Zero collision risk by construction.

### Non-Friction NF-05: Cross-Timeframe Signal Interference

- **Anticipated concern:** Signal processors might receive events from wrong timeframes.
- **Finding:** All processing is per-timeframe with timeframe encoded in NATS subject. Downstream consumers filter by subject. No cross-timeframe interference possible.

### Non-Friction NF-06: Memory Accumulation at 3600s

- **Anticipated concern:** 1-hour candle accumulation might cause memory growth.
- **Finding:** Sampler state is a single OHLCV accumulator (O(1) memory per candle, not O(trades)). Trade burst and volume samplers similarly bounded. Memory impact is negligible.

---

## 10. Summary

The TC-01 expansion revealed **no bugs** and **no real refactor triggers**. The architecture genuinely handles temporal expansion as a config concern.

The captured friction falls into three categories:

1. **Structural debt worth tracking** (F-01, F-04, F-07, F-13, F-15, F-19) — items that are fine at 4 TFs but will become relevant at 8+ TFs or with TC-02 session semantics. These form the basis for TC-02 planning.

2. **Operational fragility worth addressing** (F-02, F-05, F-08, F-16, F-17) — small gaps in validation, diagnostics, or documentation that can be fixed with targeted enhancements, not refactors.

3. **Acceptable boilerplate** (F-03, F-06, F-09, F-10, F-11, F-12, F-18) — repetition that is inherent to the multi-dimensional design and carries no structural cost.

The most important finding: **6 anticipated problems did not materialize** (NF-01 through NF-06). The S10-S15 architecture is more robust under temporal expansion than conservative planning assumed.
