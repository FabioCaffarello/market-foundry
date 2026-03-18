# Strategy Entry Prerequisites — Rerun (S52)

> Reassessment of prerequisites for strategy domain entry, based on S50 and S51 deliverables.

**Date:** 2026-03-17
**Predecessor:** [strategy-entry-prerequisites.md](strategy-entry-prerequisites.md) (S49)

---

## 1. Blocking Prerequisites Status

| ID | Prerequisite | S49 Status | Current Status | Evidence |
|----|-------------|------------|----------------|----------|
| P-1 | Evidence adapter test coverage | NOT MET | **MET** | 16 evidence registry tests, KV store tests for candle/tradeburst/volume/signal/decision |
| P-2 | Observation adapter test coverage | NOT MET | **MET** | 9 observation registry tests, binancef aggtrade tests |
| P-3 | Evidence projection actor tests | NOT MET | **MET** | 46 projection actor tests across 5 types (candle, tradeburst, volume, signal, decision) |
| P-4 | TradeBurst domain tests | NOT MET | **MET** | 10 tests in trade_burst_test.go |
| P-5 | Evidence HTTP handler tests | NOT MET | **MET** | 12 handler tests covering tradeburst/volume endpoints |
| P-6 | Strategy config dependency chain | DEFERRED | **DEFERRED** | Part of strategy first-slice; not a pre-entry blocker |
| P-7 | Strategy governance infrastructure | DEFERRED | **DEFERRED** | Part of strategy first-slice; raccoon-cli rules required before implementation |
| P-8 | Dual-write atomicity review | PARTIALLY MET | **MET** | Documented in projection-confidence-and-dual-write-review.md with scenario matrix |

**Summary:** 6 of 6 blocking prerequisites met. 2 deferred prerequisites remain (by design — they belong to the strategy first-slice, not to pre-entry gating).

---

## 2. Foundation Confidence Rules Gate

| Rule | Requirement | Status |
|------|-------------|--------|
| R-1 | Every domain value type has validation tests | **PASS** |
| R-2 | Every NATS adapter has contract tests | **PASS** |
| R-3 | Every exchange adapter has translation fidelity tests | **PASS** |
| R-4 | Every HTTP handler has query surface tests | **PASS** |
| R-5 | Every event type has dedup key isolation tests | **PASS** |

**All five gates pass.**

---

## 3. Remaining Deferred Prerequisites

### P-6: Strategy Config Dependency Chain

**When:** During strategy first-slice implementation (S54+).

**Scope:**
- Add `strategy_families` to `internal/shared/settings/schema.go`
- Add strategy family entries to `deploy/configs/derive.jsonc` and `deploy/configs/store.jsonc`
- Add raccoon-cli validation for strategy config dependencies
- Document dependency chain: `strategy → decision → signal → evidence`

### P-7: Strategy Governance Infrastructure

**When:** Before any strategy implementation code lands (S54+).

**Scope:**
- Define strategy drift rules (SD-1 through SD-5, following signal/decision pattern)
- Add strategy guardrails to raccoon-cli
- Add strategy as sensitive area in coverage map
- Update governance-hygiene-status.md

---

## 4. New Prerequisites Identified in S51

| ID | Prerequisite | Severity | Status | Notes |
|----|-------------|----------|--------|-------|
| P-9 | Single-writer guarantee for strategy projections | MEDIUM | ACKNOWLEDGED | Actor model provides this; document explicitly for strategy |
| P-10 | Strategy projection lag monitoring | LOW | DEFERRED | BG-8 from S51; enhancement, not blocker |

Neither P-9 nor P-10 blocks strategy domain design. P-9 is satisfied by the existing actor model pattern. P-10 is a monitoring improvement that applies to all projection types, not strategy-specific.

---

## 5. Minimal Acceptable Strategy Design (Unchanged)

When all prerequisites are met, the first strategy implementation must follow:

1. **One family** — e.g., `mean_reversion_entry`
2. **One binary** — derive (alongside signal and decision processing)
3. **One stream** — `STRATEGY_EVENTS`
4. **One KV bucket** — `STRATEGY_{TYPE}_LATEST` (latest-only, no history in Phase 1)
5. **One HTTP endpoint** — `GET /strategy/:type/latest`
6. **Config activation** — `pipeline.strategy_families` opt-in
7. **Dependency chain** — `strategy → decision → signal → evidence`
8. **Governance** — raccoon-cli strategy drift rules and guardrails active before code

---

## 6. Verdict

All blocking prerequisites for strategy domain entry are met. Strategy domain design may proceed.
