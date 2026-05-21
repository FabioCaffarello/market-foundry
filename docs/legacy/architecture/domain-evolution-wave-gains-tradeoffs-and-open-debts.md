# Domain Evolution Wave — Gains, Trade-offs, and Open Debts

**Charter:** S233–S237 Domain Logic Depth
**Date:** 2026-03-20

---

## 1. Gains

### 1.1 Decision Domain

| Gain | Evidence |
|------|----------|
| Severity classification (none/low/moderate/high) | Fixed 10-point zone model; deterministic, reproducible |
| Human-readable rationale | "RSI 25.00 below threshold 30.00 (distance 16.67%); severity low" |
| Metadata enrichment | threshold, rsi_zone, distance_pct in every decision |
| Confidence model formalized | Linear interpolation with 0.5 floor, 1.0 cap, monotonicity tested |
| Test density | 25 domain tests + 33 evaluator tests = 58 total |

### 1.2 Strategy Domain

| Gain | Evidence |
|------|----------|
| Decision context threading | DecisionInput carries severity + rationale as primitives |
| DBI-9 boundary preserved | No decision type imports in strategy; primitive-only crossing |
| Metadata propagation | decision_rationale in strategy metadata when non-empty |
| Backward compatibility | omitempty tags; zero-value defaults for pre-depth data |

### 1.3 Risk Domain

| Gain | Evidence |
|------|----------|
| End-to-end traceability | Decision severity/rationale visible in risk assessment without joins |
| Contextual rationale | "Position size X within limits; decision severity high" |
| Metadata enrichment | decision_severity + decision_rationale in risk metadata |
| Dual access pattern | StrategyInput (structured) + Metadata (flat query) |
| Test density | 25 domain tests + 20 evaluator tests = 45 total |

### 1.4 Integration and Infrastructure

| Gain | Evidence |
|------|----------|
| ClickHouse migration 007 | severity + rationale columns, idempotent, backward compatible |
| Writer pipeline alignment | 16 decision columns, 15 strategy columns, 17 risk columns |
| CI integration tests | 4-job matrix in remote CI (~30s cost) |
| Smoke-analytical Phase 7 | Domain depth validation with tiered result classification |
| Codegen alignment | Golden snapshots updated for all 6 families |
| Actor message isolation | Primitive-only messages for decision→strategy→risk chain |

### 1.5 Architecture and Governance

| Gain | Evidence |
|------|----------|
| 7 architecture documents | Semantics, boundaries, consistency model, validation findings |
| 5 stage reports | Full traceability of decisions and changes |
| Entry/exit/stop conditions | Formal governance framework (reusable for future charters) |
| Permitted/prohibited scope | Clear boundary enforcement |
| Consistency model | End-to-end decision→strategy→risk information flow documented |

---

## 2. Trade-offs Accepted

### 2.1 Depth over Breadth

**Decision:** Enriched existing evaluators instead of adding new evaluator families.
**Cost:** Charter breadth criteria (≥2 types per domain) not achieved.
**Justification:** Depth creates the semantic foundation that makes future breadth coherent. A second evaluator without severity/rationale would be structurally hollow.

### 2.2 Traceability over Logic

**Decision:** Severity is recorded end-to-end but not acted upon.
**Cost:** No parameter modulation, risk gating, or disposition logic based on severity.
**Justification:** Acting on severity without validation data would be premature. The traceability foundation enables future data-driven activation.

### 2.3 Observability over Performance

**Decision:** Metadata duplicated across layers (StrategyInput + flat Metadata).
**Cost:** Slightly increased storage footprint per event.
**Justification:** Enables flat ClickHouse queries without JSON parsing or cross-table joins.

### 2.4 Backward Compatibility over Schema Cleanliness

**Decision:** New fields use omitempty/zero-value defaults rather than migrations with NOT NULL.
**Cost:** Historical data has empty severity/rationale (semantically correct but requires awareness).
**Justification:** Avoids breaking existing data; zero-values handled gracefully by Go.

### 2.5 Light Hardening over Comprehensive CI

**Decision:** S237 hardening limited to integration tests + smoke Phase 7.
**Cost:** No load testing, no full smoke in remote CI, no schema drift detection.
**Justification:** Stayed within 20% hardening budget; heavier CI is a future charter concern.

---

## 3. Open Debts

### 3.1 Structural Debts (Carried Forward)

| Debt | Origin | Impact | Priority |
|------|--------|--------|----------|
| Domain breadth: 1 evaluator per domain | S233 charter pivot | Limits resilience testing and family diversity | **P0 — next charter** |
| No charter amendment for scope pivot | S234–S236 | Governance gap; criteria/execution mismatch | Closed by S238 gate document |
| Strategy test sparseness | S235 | 14 domain tests vs 25 for Decision/Risk; missing multi-symbol isolation | P1 |
| No inter-actor chain integration test | S237 | Complete decision→strategy→risk flow not tested in single harness | P1 |

### 3.2 Functional Debts (Deliberate Deferrals)

| Debt | Rationale for Deferral | Activation Condition |
|------|----------------------|---------------------|
| Severity-dependent resolution | No validation data for parameter modulation | After ≥1 month of severity data collection |
| Severity-dependent risk gating | No evidence for rejection thresholds | After severity distribution analysis |
| Multi-decision strategy resolution | Only single-decision strategies exist | When ≥2 decision families are available |
| Cross-symbol aggregate risk | Single-symbol risk only | When multi-symbol portfolio management is in scope |

### 3.3 Infrastructure Debts (Pre-existing, Unchanged)

| Debt | Origin | Notes |
|------|--------|-------|
| Documentation entropy (265+ arch docs, 224+ stage reports) | Pre-S233 | No lifecycle policy; accumulating |
| raccoon-cli assumption freshness | Pre-S233 | 6 fixed in S229; others possibly stale |
| Full smoke not in remote CI | Pre-S233 | Requires ClickHouse infrastructure in CI |
| Performance regression testing | Never existed | No latency baseline or load testing |
| Production readiness | Never existed | Deployment, monitoring, security scanning |
| marketmonkey absorption | Pre-S233 | Deferred; no progress |

### 3.4 Test Coverage Gaps

| Gap | Domain | Severity |
|-----|--------|----------|
| Severity-outcome consistency validation | Decision | Low (semantically correct by construction) |
| Boundary condition tests for position sizing | Risk | Medium |
| Multi-symbol isolation tests | Strategy | Medium |
| Risk confidence scaling justification (0.95 factor) | Risk | Low |
| Correlation/causation ID propagation test | All | Medium |

---

## 4. Net Assessment

The charter delivered **real semantic depth** across all three domains. The traceability chain from decision through risk is a genuine capability gain. The trade-off of depth over breadth was pragmatically correct but governmentally undisciplined.

**Net value: Positive.** The codebase is materially stronger than at S232 baseline. The depth work created patterns and infrastructure that reduce the cost of future breadth expansion. No regressions introduced. No structural damage.

**Net debt: Slightly increased.** Domain breadth remains the primary gap. Strategy test coverage is lighter than peers. Infrastructure debts are unchanged (neither improved nor worsened).
