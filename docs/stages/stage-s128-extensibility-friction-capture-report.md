# Stage S128 — Extensibility Friction Capture Report

> Disciplined capture of extensibility frictions exposed by CC-02 (EMA Crossover Signal Family).

**Stage:** S128
**Scope:** Friction capture and architectural reading — no new features, no refactors executed
**Inputs:** S125 (family definition), S126 (implementation), S127 (operational validation), S124 (deferred debts)
**Outputs:** Friction inventory, triggered-vs-deferred matrix, next-stage preparation

---

## 1. Executive Summary

CC-02 confirms that market-foundry absorbs new signal families with **predictable, bounded friction and zero architectural regression**. The extensibility model works as designed.

The total cost of adding `ema_crossover` was 3 new files + 7 modifications + ~462 lines (including tests and smoke coverage). All 22 mandatory extensibility criteria passed. No domain model changes, no infrastructure redesign, no test regressions.

**Friction is real but mechanical.** The primary pain points are actor boilerplate (~97 copy-pasted lines per family), scattered NATS registry switches (4 touch points), and store pipeline registration (~25 lines). None of these caused errors or blocked implementation.

**Of the 10 deferred debts from S124, only CF-03 (correlation ID) had its trigger definitively met — and the non-action produced zero incidents.** CF-08 (actor boilerplate) is approaching its threshold (N=3) but is correctly deferred at N=2.

The architecture is ready for the next capability or a small, evidence-justified refactor batch.

---

## 2. Friction Findings

### 2.1 Confirmed Frictions (Ordered by Impact)

| # | Friction | Classification | Lines/Family | Touch Points | Threshold |
|---|---------|---------------|-------------|-------------|-----------|
| 1 | Sampler actor copy-paste (CF-08) | Structural debt | ~97 | 1 new file | **N=3 families** |
| 2 | NATS registry switch scatter (CF-11, new) | Structural debt | ~37 | 3 files | **N=3 families** |
| 3 | Store pipeline boilerplate (CF-12, new) | Acceptable boilerplate | ~25 | 1 file | N=5 families |
| 4 | Correlation ID manual copy (CF-03) | Structural debt | ~2 | per actor | **Opportunistic** |
| 5 | Derive processor registration | Acceptable boilerplate | ~10 | 1 file | No action needed |
| 6 | Config schema map entries | Acceptable boilerplate | ~4 | 1 file | No action needed |

### 2.2 Zero-Friction Areas (Design Validated)

| Area | Evidence |
|------|---------|
| Domain model (`signal.Signal`) | `Value: string` + `Metadata: map[string]string` handled numeric (RSI) and categorical (EMA) without modification |
| NATS stream topology | `SIGNAL_EVENTS` wildcard subjects auto-cover new families |
| HTTP routes | `/signal/:type/latest` fully parameterized, zero changes |
| Diagnostic surfaces | `/statusz`, `/diagz`, `/healthz` auto-include new actors |
| Config validation | Generic `ValidatePipeline()` handles any registered family |
| Coexistence | Separate KV buckets + consumers, zero interference |

### 2.3 Items That Did NOT Confirm as Problems

| Concern | Pre-CC-02 Expectation | CC-02 Reality |
|---------|----------------------|---------------|
| Domain model rigidity | Might not handle categorical values | Handled perfectly via `Value: string` |
| Stream topology pressure | Might need restructuring | Wildcards auto-extend |
| Config lifecycle breakage | Might break validation | Generic validation handles it |
| Diagnostic gaps | New family might lack observability | Auto-included via tracker injection |
| Coexistence interference | Two families might collide | Complete isolation confirmed |

---

## 3. Triggered vs Deferred Matrix

### 3.1 Triggers Definitively Met

| Debt | Original Trigger | Met? | Incident? | Recommendation |
|------|-----------------|------|-----------|---------------|
| CF-03 (correlation ID) | First new actor | **YES** | None | Implement at CC-03 (P2) |

### 3.2 Triggers Approaching

| Debt | Original Trigger | Current State | Recommendation |
|------|-----------------|--------------|---------------|
| CF-08 (actor boilerplate) | N=3 signal families | N=2 | Implement generic actor at N=3 |
| CF-11 (registry switches) | N=3 signal families | N=2 | Implement map-based registry at N=3 |

### 3.3 Triggers Not Met (Correctly Deferred)

| Debt | Trigger | Why Not Met | Action |
|------|---------|------------|--------|
| CF-02 (active symbols) | Configctl route change or N>5 symbols | No route changes, N=2 symbols | Continue deferring |
| CF-08 (client boilerplate) | New domain family | CC-02 is within signal domain | Continue deferring |
| D4 (composition root tests) | New runtime or wiring error | No new runtime, no errors | Continue deferring |
| D5 (failure recovery) | Production deployment | Not planned | Continue deferring |
| D6 (soak testing) | N>5 symbols or 24h operation | N=2, manual validation | Continue deferring |

### 3.4 Resolved (No Longer Tracked)

CF-01 (per-symbol trackers), CF-04 (error log scanning), CF-05 (memory tracking) — all resolved in S123.

---

## 4. Trade-Offs Accepted

| Trade-Off | Rationale | Revisit Condition |
|-----------|-----------|-------------------|
| ~174 lines of registration boilerplate per family | Mechanical, predictable, error-free at N=2 | N=3 families (justified by three examples) |
| No per-family algorithm configuration | Deliberate simplification; hardcoded periods are correct for current use | A/B testing or per-binding tuning requirement |
| Global kill switch (CF-07) | Paper-only execution; halting both symbols is the safe default | Live venue adapter activation |
| No sustained soak testing (CF-06) | Manual validation at intervals sufficient at N=2 symbols | N>5 symbols or 24h operation goals |
| CF-03 trigger met but not acted on | Zero incidents from manual pattern; middleware is growth-protection, not a fix | CC-03 implementation (opportunistic) |

---

## 5. New Friction Items Cataloged

CC-02 revealed 3 frictions not previously tracked:

| ID | Description | Type | Trigger |
|----|------------|------|---------|
| CF-11 | NATS registry uses hardcoded switch statements; 4 manual touch points per family | Structural debt | N=3 families → replace with map-based registry |
| CF-12 | Store `declarePipelines()` requires ~25 lines of manual pipeline declaration per family | Acceptable boilerplate | Evaluate at N=5 |
| CF-13 | No config-driven algorithm parameterization (periods hardcoded in constructors) | Intentional limitation | A/B testing or per-binding tuning use case |

---

## 6. Extensibility Cost Model

Based on CC-02 evidence, the marginal cost of adding a signal family:

| Component | Lines | Type |
|-----------|-------|------|
| Domain sampler + tests | ~240 | Unique logic (irreducible) |
| Sampler actor | ~97 | Boilerplate (reducible at N=3) |
| NATS registry entries | ~37 | Boilerplate (reducible at N=3) |
| Store pipeline entry | ~25 | Boilerplate (acceptable) |
| Derive processor entry | ~10 | Boilerplate (acceptable) |
| Config schema entries | ~4 | Trivial |
| Smoke test coverage | ~80 | Validation (irreducible) |
| HTTP routes | 0 | None needed |
| Diagnostics | 0 | None needed |
| **Total** | **~493** | **~240 unique + ~173 boilerplate + ~80 validation** |

**Reducible boilerplate at N=3:** ~134 lines (actor + registry) could be eliminated with generic factory + map-based registry.

---

## 7. Preparation for S129

### 7.1 If S129 Is a Third Signal Family (CC-03)

Recommended refactors to bundle with CC-03 implementation:

1. **Generic `SignalSamplerActor`** — Extract shared lifecycle into parameterized actor accepting a `SignalSampler` interface. Eliminates ~97 lines of copy-paste per family. Effort: ~2 hours.
2. **Map-based NATS signal registry** — Replace switch statements with `map[string]EventSpec` lookup. Centralizes 4 scatter points into 1 registration site. Effort: ~1–2 hours.
3. **Correlation ID middleware (CF-03)** — Implement the S123 design sketch. First actor to use it is the CC-03 sampler actor. Effort: ~2–3 hours.

Total estimated effort for bundled refactors: ~5–7 hours, amortized across CC-03 delivery.

### 7.2 If S129 Is a Different Capability

The current friction level does not block any capability. Proceed with capability delivery; revisit refactors when the third signal family is introduced.

### 7.3 What NOT to Do

- Do not execute refactors without a third concrete example (premature abstraction risk)
- Do not open a dedicated refactoring stage for ~5 hours of work (bundle with delivery)
- Do not treat acceptable boilerplate (store pipeline, derive processor, config maps) as structural debt
- Do not confuse intentional limitations (hardcoded periods, global kill switch) with architectural failures

---

## 8. Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Extensibility frictions captured with evidence | **PASS** — 6 frictions documented with code references and line counts |
| Bugs, boilerplate, debts, trade-offs distinguished | **PASS** — Each friction classified as structural debt, acceptable boilerplate, intentional limitation, or trade-off |
| Natural triggers confirmed or rejected | **PASS** — CF-03 confirmed triggered; CF-08 approaching; CF-02, D4-D6 correctly deferred |
| Prioritization useful for next decision | **PASS** — Clear N=3 threshold for actor/registry refactors; bundled effort estimated |
| Ready for small, justified refactors | **PASS** — Three refactors identified for CC-03 bundling; no standalone refactor stage needed |

---

## 9. Deliverables

| Deliverable | Path | Status |
|-------------|------|--------|
| Extensibility frictions and findings | `docs/architecture/cc-02-extensibility-frictions-and-findings.md` | Complete |
| Triggered vs deferred refactor matrix | `docs/architecture/cc-02-triggered-vs-deferred-refactor-matrix.md` | Complete |
| Stage report | `docs/stages/stage-s128-extensibility-friction-capture-report.md` | This document |

---

## 10. Conclusion

CC-02 served its purpose: it revealed that market-foundry's extensibility is **real, measurable, and bounded**. The friction is mechanical registration boilerplate — not algorithmic complexity, not architectural misalignment, not domain model rigidity. The deferred debts from S124 are correctly calibrated: only CF-03 fired, and it fired without consequence. The architecture is healthy. The next move is either a third signal family (which bundles justified refactors) or a different capability (which proceeds unblocked).
