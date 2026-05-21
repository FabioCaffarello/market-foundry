# Refactors Still Deferred After TC-01 — Timeframe Coverage

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S135 (Triggered Refactors)
> **Date:** 2026-03-19
> **Source:** S134 Prioritized Friction Matrix

---

## 1. Purpose

This document records every friction item from S134 that was consciously **not** addressed in S135, with explicit rationale for deferral. Items listed here remain tracked and will be re-evaluated at their specified trigger points.

---

## 2. Deferred Items

### D-01: Per-Binding Timeframes (F-01)

**S134 Priority:** P2 (TC-02 gate)
**Classification:** Structural Debt
**Why deferred:** The global timeframe list works correctly at TC-01. Per-binding overrides are only needed if TC-02 introduces heterogeneous timeframe sets per symbol (e.g., BTC needs 4h but a low-volume alt only needs 1m/5m). This requirement has not materialized.
**Trigger for action:** TC-02 scoping requires different timeframes per symbol.
**What stays simple:** Single `pipeline.timeframes` array in `derive.jsonc`.

### D-02: Per-Timeframe Tracker Split (F-04)

**S134 Priority:** P2 (TC-02 gate)
**Classification:** Structural Debt
**Why deferred:** The single `evidence-publisher` tracker provides aggregate visibility. At 4 TFs, this is adequate — per-timeframe counter totals are available via custom counters when emitted. At 8+ TFs, aggregate-only visibility makes diagnosing stalled timeframes harder.
**Trigger for action:** TC-02 adds 8+ timeframes, or operational incident where aggregate tracker masks per-TF stall.
**What stays simple:** Single tracker registration per evidence publisher.

### D-03: Per-Timeframe Idle Detection (F-05)

**S134 Priority:** P2 (TC-02 gate)
**Classification:** Operational Fragility
**Why deferred:** The idle heartbeat monitor (30s interval, 2min threshold) is calibrated for 60s candles. A 3600s sampler is legitimately idle for up to 60 minutes. Adjusting the threshold would mask 60s issues. The correct fix is timeframe-aware health reporting, which adds complexity not justified at 4 TFs.
**Trigger for action:** TC-02 adds 4h+ timeframes where stall detection becomes operationally critical.
**What stays simple:** Component-level idle detection with uniform threshold.

### D-04: "List Available Timeframes" Endpoint (F-07)

**S134 Priority:** P3 (track, revisit)
**Classification:** Structural Debt
**Why deferred:** All query consumers are internal operators who know the config. A discovery endpoint provides convenience but no correctness gain.
**Trigger for action:** External consumers query the system without config access.
**What stays simple:** No new endpoint. Config is the source of truth.

### D-05: Null Response Disambiguation (F-08)

**S134 Priority:** P3 (track, revisit)
**Classification:** Operational Fragility
**Why deferred:** The `{"candle": null}` response is correct but ambiguous (not configured vs. warming up vs. expired). The operator can check config to disambiguate. Adding `"status"` fields increases response complexity across all query endpoints.
**Trigger for action:** Non-expert consumers are exposed to query surfaces, or operational incident caused by response ambiguity.
**What stays simple:** Null response = no data. Config is the disambiguator.

### D-06: Window State Persistence (F-13 + F-15)

**S134 Priority:** P2 (TC-02 gate)
**Classification:** Structural Debt (highest-impact item)
**Why deferred:** This is the most significant structural debt from TC-01. At 3600s, a crash loses up to 60 minutes of accumulated state. At TC-02 (4h), it would be 4 hours. The fix (WAL or interim snapshots for in-progress candles) is a **Large** effort item that changes the accumulator's persistence model. This was explicitly scoped out in S131 L4.
**Trigger for action:** TC-02 commits to 4h+ timeframes. This item is a **hard gate** for TC-02 execution.
**What stays simple:** Accumulators are in-memory only. Final-only candle semantics. No WAL or snapshot infrastructure.

### D-07: Gateway Aggregate View (F-19)

**S134 Priority:** P3 (track, revisit)
**Classification:** Structural Debt
**Why deferred:** `live-pipeline-activate.sh` Phase 6 automates the N-query check across all domains × symbols × timeframes. Manual querying is tedious but feasible at 2 symbols × 4 TFs. A `/pipeline/overview` endpoint would be valuable at higher cardinality.
**Trigger for action:** Symbol count grows beyond ~5, or dashboard integration is needed.
**What stays simple:** No aggregate endpoint. Script-based checks suffice.

---

## 3. Permanently Accepted Items (P4)

These items were classified as P4 in S134 and remain permanently accepted:

| ID | Friction | Rationale |
|----|----------|-----------|
| F-03 | Integer-only timeframe representation | Unambiguous, machine-friendly, consistent across all surfaces |
| F-06 | Log verbosity scaling | Linear growth by design; `slog` supports filtering |
| F-09 | HTTP test file duplication | Each block independently executable; provides documentation value |
| F-10 | Actor count growth | Linear by design; Hollywood engine handles thousands |
| F-11 | KV key cardinality | NATS KV handles millions of keys |
| F-12 | NATS subject cardinality | NATS handles millions of subjects |
| F-14 | Signal warmup latency at high TFs | Physics constraint, not architectural deficiency |
| F-16 | Smoke test wait time assumptions | Three-tier validation procedure correctly separates wiring from data validation |
| F-18 | Configctl has no timeframe concept | Correct separation of concerns |

---

## 4. TC-02 Gate Summary

Before TC-02 execution begins, the following deferred items must be resolved or explicitly accepted with documented risk:

| Item | Decision Required |
|------|------------------|
| D-06 (State persistence) | Evaluate WAL/snapshot cost. If prohibitive, accept with documented 4h state loss risk. **Hard gate.** |
| D-02 + D-03 (Per-TF diagnostics) | Implement per-timeframe tracker split + timeframe-aware idle detection. Low cost, high operational value. |
| D-01 (Per-binding TFs) | Evaluate only if TC-02 design requires heterogeneous TF sets per symbol. |
