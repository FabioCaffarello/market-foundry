# Strategy Risks and Blockers — Rerun (S52)

> Reassessment of risks and blockers for strategy domain entry, incorporating S50 and S51 results.

**Date:** 2026-03-17
**Predecessor:** [strategy-risks-and-blockers.md](strategy-risks-and-blockers.md) (S49)

---

## 1. S49 Blocking Gaps — Final Status

| Gap | Severity | Resolution | Stage | Verification |
|-----|----------|------------|-------|-------------|
| BG-1: Evidence adapter tests | CRITICAL | **CLOSED** | S50 | 19 tests: evidence_registry_test.go (16), candle_kv_store_test.go, signal_kv_store_test.go, decision_kv_store_test.go, trade_burst_kv_store_test.go, volume_kv_store_test.go |
| BG-2: Observation/ingest pipeline tests | CRITICAL | **CLOSED** | S50 | 22 tests: trade_test.go (8), aggtrade_test.go, observation_registry_test.go (9) |
| BG-3: Evidence projection actor tests | HIGH | **CLOSED** | S51 | 46 tests across 5 projection actors with interface extraction |
| BG-4: TradeBurst domain validation tests | MEDIUM | **CLOSED** | S50 | 10 tests in trade_burst_test.go |
| BG-5: Evidence HTTP handler tests | MEDIUM | **CLOSED** | S50 | 12 handler tests for tradeburst/volume |
| BG-6: Candle dual-write atomicity | MEDIUM | **CLOSED** | S51 | Documented in projection-confidence-and-dual-write-review.md |

**All six blocking gaps are closed. Zero critical or high-severity blockers remain.**

---

## 2. New Risks from S51

| Risk | Severity | Category | Impact on Strategy | Mitigation |
|------|----------|----------|--------------------|------------|
| BG-7: Multi-instance store violates single-writer | MEDIUM | Architectural | None (single-instance deployment) | Actor model serialization; document as constraint for strategy projections |
| BG-8: No projection lag metric | LOW | Observability | None (monitoring enhancement) | Defer to infrastructure hardening stage |
| R4: VolumeKVStore error return inconsistency | LOW | Cosmetic | None | Normalize in future cleanup stage |

**None of the new risks are blocking for strategy domain design.**

---

## 3. Non-Blocking Risks (Carried from S49)

| Risk | Severity | S49 Status | Current Status | Notes |
|------|----------|------------|----------------|-------|
| NBR-1: Binding deactivation requires restart | LOW | Acknowledged | **Unchanged** | Known limitation of activation model; not strategy-specific |
| NBR-2: Single exchange adapter (binancef) | LOW | Acknowledged | **Unchanged** | Observation-layer concern; does not block strategy |
| NBR-3: No projection lag metrics | LOW | Acknowledged | **Unchanged** | Now tracked as BG-8; monitoring enhancement |
| NBR-4: QueryResponderActor not family-filtered | LOW | Acknowledged | **Unchanged** | Performance optimization; not correctness issue |
| NBR-5: No signal/decision history projections | LOW | Acknowledged | **Unchanged** | Latest-only is sufficient for strategy Phase 1 |

---

## 4. Strategy-Specific Risks (Forward-Looking)

These risks apply to strategy implementation, not to the current entry gate. They are documented here for awareness during strategy design.

### SR-1: Strategy Dependency Chain Depth
- **Risk:** Strategy depends on decision → signal → evidence → observation. A failure at any layer cascades.
- **Severity:** MEDIUM
- **Mitigation:** Each layer has independent health tracking. Strategy evaluators must handle `insufficient` outcomes from decision layer gracefully.

### SR-2: Strategy Config Complexity
- **Risk:** Adding `strategy_families` increases config surface. Cross-layer dependency validation must cover `strategy → decision_families → signal_families → families`.
- **Severity:** MEDIUM
- **Mitigation:** raccoon-cli already validates decision→signal→evidence chain. Extending to strategy is a mechanical addition.

### SR-3: Derive Binary Scope Growth
- **Risk:** Derive binary now hosts sampler, signal, decision, and (soon) strategy actors. Actor count grows with each layer.
- **Severity:** LOW
- **Mitigation:** Actor model isolates concerns. Each actor has its own lifecycle. No shared mutable state. Binary split can happen later if needed.

### SR-4: Strategy Governance Bootstrapping
- **Risk:** If strategy implementation proceeds without governance rules, drift can occur before raccoon-cli catches it.
- **Severity:** HIGH
- **Mitigation:** P-7 (strategy governance infrastructure) is a hard prerequisite before any strategy code lands. This is enforced by the readiness review process.

---

## 5. Risk-Impact Matrix (Updated)

| Priority | Risks | Action |
|----------|-------|--------|
| P0 (Critical) | — | **None.** All P0 risks from S49 are closed. |
| P1 (High) | SR-4 (governance bootstrapping) | Enforce P-7 before implementation |
| P2 (Medium) | BG-7 (multi-instance), SR-1 (chain depth), SR-2 (config complexity) | Address during strategy design |
| P3 (Low) | BG-8, NBR-1–NBR-5, SR-3 | Monitor; defer to hardening stages |

---

## 6. Verdict

The risk profile has fundamentally changed since S49. The system moved from **2 critical + 1 high-severity blockers** to **zero blocking risks**. The remaining items are low/medium severity, non-blocking, and well-understood. Strategy domain design can proceed with acceptable risk.
