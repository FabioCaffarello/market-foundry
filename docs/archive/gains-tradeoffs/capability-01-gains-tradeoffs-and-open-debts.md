# Capability 01 — Gains, Trade-offs, and Open Debts

> Stage S124 — Evidence-based ledger of what CC-01 gave, cost, and left behind.
> Date: 2026-03-19

---

## Purpose

This document is the definitive accounting of the CC-01 wave (S119–S123). It records what was gained, what was traded, and what debt remains — so future decisions are made on fact, not memory.

---

## Gains

### G1: Config-Driven Horizontal Scaling — Validated

**What we gained:** Proof that adding a new symbol requires zero application code changes. The full pipeline (observation → signal → decision → strategy → risk → execution → fill) activates for a new symbol through config binding alone.

**Evidence:** S120 added ethusdt to btcusdt with 3 files changed (scripts + Makefile, ~62 lines). S121 validated both symbols flowing through all 6 runtimes and 8 domains.

**Structural impact:** This is the most important architectural property of Market Foundry. It means the cost of adding symbol N+1 is operational (config), not engineering (code).

---

### G2: Cross-Symbol Isolation — Validated

**What we gained:** Proof that composite key design (`source.symbol.timeframe`) provides correct per-symbol data isolation under concurrent load.

**Evidence:** S121 cross-symbol smoke test checks (comparing OHLCV values between symbols) pass consistently. 7 of 12 predicted contamination pressure points produced zero friction.

**Structural impact:** Actors, KV stores, and query surfaces can be trusted to partition by symbol without defensive coding.

---

### G3: Per-Symbol Diagnostic Visibility — Achieved (S123)

**What we gained:** `/statusz` now shows per-symbol counter breakdowns. Operators can answer "is ethusdt flowing?" from a single HTTP call.

**Evidence:** S123 refactor R1 instrumented 14 actors with per-symbol counter keys using the existing `tracker.Counter()` API.

**Structural impact:** Diagnostic capability now scales with symbols, not against them. Each new symbol automatically appears in counter keys.

---

### G4: Automated Operational Checks — Achieved (S123)

**What we gained:** The activation script now detects domain-level errors and captures memory baselines automatically.

**Evidence:** S123 refactors R2 and R3 added error log scanning and docker stats snapshot to Phase 8 of `live-pipeline-activate.sh`.

**Structural impact:** Validation runs are more reliable. Silent failures that previously required manual log review are now caught automatically.

---

### G5: Architecture Governance Holds Under Load

**What we gained:** Proof that raccoon-cli's ~950 governance rules remain valid and no architectural violations were introduced during CC-01 delivery.

**Evidence:** All governance checks pass throughout S119–S123. No new boundary violations, naming inconsistencies, or topology drift.

**Structural impact:** The governance tooling scales with the codebase and continues to provide value.

---

### G6: Zero Domain Logic Bugs

**What we gained:** Confidence that the domain model is correct under multi-symbol operation.

**Evidence:** S121 validated the full domain chain for 2 symbols over 30+ minutes. Zero bugs in domain logic. All 3 bugs found during the broader S96–S118 wave were infrastructure/wiring issues, not domain errors.

**Structural impact:** Domain model changes (new signal types, decision families) can build on a proven foundation.

---

## Trade-offs Accepted

### T1: Global Kill Switch (CF-07)

**What we traded:** Per-symbol execution control. Activating the kill switch halts paper order processing for all symbols simultaneously.

**Why accepted:** Paper-only execution. Halting both symbols is the safe default. Per-symbol control is a separate capability requiring per-symbol KV state.

**Revisit condition:** If live venue adapter is activated, or if operator explicitly needs per-symbol halt.

---

### T2: RSI Warm-Up Delay (CF-09)

**What we traded:** Immediate full-chain validation for newly-activated symbols. RSI requires 14 candles (~15 minutes at 60s timeframe) before producing non-null values.

**Why accepted:** Mathematical requirement of RSI indicator. Cannot be shortened without changing the signal definition.

**Revisit condition:** Never — inherent to RSI.

---

### T3: 300s Timeframe Wait (CF-10)

**What we traded:** Fast validation of 300-second timeframe candles. The 5-minute candle window requires at least 5 minutes before materialization.

**Why accepted:** 60-second timeframe provides sufficient pipeline validation. 300s is supplementary.

**Revisit condition:** Never — inherent to timeframe design.

---

### T4: Manual Sustained Monitoring (CF-06)

**What we traded:** Automated watchdog for sustained soak testing. The 30-minute validation requires operator attention.

**Why accepted:** Manual `make live-multi-check` at intervals is sufficient for 2 symbols. Building watchdog infrastructure has no consumer at N=2.

**Revisit condition:** When soak testing at N>5 symbols or 24-hour duration becomes a goal.

---

### T5: Correlation ID Design-Only (CF-03)

**What we traded:** Immediate implementation of correlation ID middleware for a design sketch.

**Why accepted:** No new actors are being added. Implementing without a consumer risks designing the wrong abstraction. Current actors are consistent — the fragility is in growth, not in present behavior.

**Revisit condition:** When the first new actor is added (CC-02 or equivalent).

---

## Open Debts

### D1: Correlation ID Middleware Implementation (CF-03)

**Nature:** Structural debt. Manual correlation ID copy in every actor is fragile at scale.

**Current state:** Design sketch produced in S123. All existing actors are consistent.

**Risk if not addressed:** Silent correlation chain breaks when new actors are added.

**Trigger:** First new actor addition.

**Estimated effort:** 2–3 hours.

---

### D2: Active Symbols Endpoint (CF-02)

**Nature:** Operational convenience debt. No endpoint to discover which symbols are active.

**Current state:** Workaround exists — parse active config response body.

**Risk if not addressed:** Operator confusion at N>5 symbols.

**Trigger:** When touching configctl routes.

**Estimated effort:** 1 hour.

---

### D3: Client UseCase Boilerplate (CF-08)

**Nature:** Structural duplication. 6 domain client packages still hand-write the struct+Execute pattern (~180 lines total).

**Current state:** Code is correct. `configctlclient` already migrated to type aliases.

**Risk if not addressed:** Copy-paste errors when adding a new domain client operation.

**Trigger:** When adding a new domain family.

**Estimated effort:** 1 hour.

---

### D4: Composition Root Smoke Tests

**Nature:** Testing gap. Composition roots (`cmd/*/run.go`) have no automated integration test.

**Current state:** Wiring correctness proven only by full-stack run.

**Risk if not addressed:** Wiring errors caught late (at deploy, not CI).

**Trigger:** When adding a new runtime or refactoring composition roots.

**Estimated effort:** 2–3 hours.

---

### D5: Failure Recovery Validation

**Nature:** Operational gap. NATS reconnection, actor crash recovery, and JetStream consumer restart never exercised.

**Current state:** Expected to work (NATS client has built-in reconnect), but not validated.

**Risk if not addressed:** Unknown behavior under network partitions or container restarts.

**Trigger:** Before any production-grade deployment.

**Estimated effort:** 4–6 hours.

---

### D6: Soak Testing Infrastructure

**Nature:** Operational gap. No automated long-duration stability test.

**Current state:** 30-minute manual validation is the longest run.

**Risk if not addressed:** Slow goroutine leaks or buffer accumulation undetected.

**Trigger:** When operating at N>5 symbols or 24-hour duration.

**Estimated effort:** 2–3 hours.

---

## Debt Budget Summary

| Category | Items | Total Effort | Urgency |
|----------|-------|-------------|---------|
| Growth-triggered (implement on next expansion) | D1, D2, D3 | ~4–5 hours | Low — triggers exist |
| Quality-triggered (implement before production) | D4, D5 | ~6–9 hours | Medium — no near-term deployment planned |
| Scale-triggered (implement at N>5) | D6 | ~2–3 hours | Low — current scale is N=2 |
| **Total** | **6 debts** | **~13–17 hours** | **None blocking next capability** |

---

## Accounting Verdict

CC-01 delivered more gains than debts. The gains (G1–G6) are permanent architectural properties that compound with scale. The trade-offs (T1–T5) are correctly scoped — each has a documented rationale and revisit condition. The debts (D1–D6) total ~13–17 hours and all have natural triggers; none requires a dedicated refactoring stage.

The platform's debt-to-gain ratio is healthy. The next wave should be capability delivery, not debt reduction.
