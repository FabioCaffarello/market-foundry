# Post-Multi-Symbol Next Wave Options Matrix

> Strategic comparison of candidate next waves after Phase 29 closure.
> Decision authority: S305 gate.
> Date: 2026-03-21

---

## 1. Evaluation Criteria

Each candidate wave is evaluated against five dimensions:

| Criterion | Weight | Rationale |
|-----------|--------|-----------|
| **Value unlock** | High | Does this wave enable meaningful new capability or remove a hard blocker? |
| **Evidence readiness** | High | Does the Foundry already have the prerequisites to start this wave? |
| **Risk of deferral** | Medium | What happens if this wave is postponed? Does technical debt compound? |
| **Scope containability** | Medium | Can this wave be scoped to 4–6 stages without inflation? |
| **Dependency chain** | Medium | Does this wave unblock other high-value work? |

---

## 2. Candidate Waves

### Option A: Venue Readiness Charter

**Objective**: Replace paper execution with real exchange connectivity (adapter, order lifecycle, fill reconciliation).

| Criterion | Assessment | Score |
|-----------|------------|-------|
| Value unlock | **Very High** — the Foundry cannot operate in production without venue integration; this is the single largest remaining capability gap | 5/5 |
| Evidence readiness | **High** — paper execution proven (S264–S274), multi-symbol isolation confirmed (S300–S304), venue adapter skeleton exists from S90–S93 | 4/5 |
| Risk of deferral | **High** — every other wave that touches execution (portfolio risk, OMS) depends on real venue behavior being understood first | 4/5 |
| Scope containability | **Medium** — venue work is inherently integration-heavy; requires careful scope freeze to avoid OMS/portfolio creep | 3/5 |
| Dependency chain | **Very High** — unblocks: portfolio risk, OMS, real operational dashboards, live multi-symbol validation | 5/5 |

**Total: 21/25**

### Option B: Second Decision Family End-to-End

**Objective**: Add a second decision family (e.g., momentum or volatility-based) with full pipeline coverage from signal to execution.

| Criterion | Assessment | Score |
|-----------|------------|-------|
| Value unlock | **Medium** — expands strategy diversity but does not remove a hard blocker; codegen path (S258–S263) already proved family extensibility | 3/5 |
| Evidence readiness | **High** — codegen pipeline ready, family expansion pattern proven (Wave B, Phase 14), multi-symbol pattern validated | 4/5 |
| Risk of deferral | **Low** — no compounding debt; the existing 3 families are sufficient for operational validation | 2/5 |
| Scope containability | **High** — proven pattern from Wave B; 4–5 stages predictable | 4/5 |
| Dependency chain | **Low** — does not unblock venue, portfolio, or operational maturity work | 2/5 |

**Total: 15/25**

### Option C: Additional Multi-Symbol Hardening

**Objective**: Close MQ7 (resource measurement), add actor-level concurrency testing, stress-test sub-millisecond ordering.

| Criterion | Assessment | Score |
|-----------|------------|-------|
| Value unlock | **Low** — marginal improvement on already-proven isolation; MQ7 expected proportional, no counter-evidence | 2/5 |
| Evidence readiness | **High** — test infrastructure in place, scenarios defined | 4/5 |
| Risk of deferral | **Very Low** — existing architecture is proven sound; these are polish items, not blockers | 1/5 |
| Scope containability | **High** — narrow scope, 2–3 stages maximum | 5/5 |
| Dependency chain | **None** — does not unblock any successor wave | 1/5 |

**Total: 13/25**

### Option D: Portfolio Risk Aggregation

**Objective**: Cross-symbol portfolio-level risk checks (total exposure, correlation-aware risk, aggregate position limits).

| Criterion | Assessment | Score |
|-----------|------------|-------|
| Value unlock | **High** — required for safe multi-symbol production operation | 4/5 |
| Evidence readiness | **Medium** — symbol isolation proven, but portfolio risk requires venue behavior understanding (fill prices, actual positions) that paper mode cannot provide | 2/5 |
| Risk of deferral | **Medium** — not blocking paper operation; becomes critical only with real venue integration | 3/5 |
| Scope containability | **Medium** — portfolio risk design is complex; scope inflation risk without venue baseline | 3/5 |
| Dependency chain | **Medium** — blocked by venue readiness (needs real positions and fills to be meaningful) | 3/5 |

**Total: 15/25**

### Option E: Operational Maturity (Dashboards, Alerting, Runbooks)

**Objective**: Grafana dashboards, Prometheus alerting, structured runbooks for operational monitoring.

| Criterion | Assessment | Score |
|-----------|------------|-------|
| Value unlock | **Medium** — valuable for operations but does not expand core capability | 3/5 |
| Evidence readiness | **Medium** — observability surfaces exist but are HTTP-only; no metrics pipeline or dashboard infrastructure yet | 2/5 |
| Risk of deferral | **Low** — paper mode does not require production-grade monitoring | 2/5 |
| Scope containability | **Medium** — operational tooling tends to expand; needs strict scope freeze | 3/5 |
| Dependency chain | **Low** — useful but not blocking other waves | 2/5 |

**Total: 12/25**

---

## 3. Comparison Summary

| Option | Total | Value | Readiness | Deferral Risk | Containment | Dependencies |
|--------|-------|-------|-----------|---------------|-------------|--------------|
| **A: Venue Readiness** | **21/25** | 5 | 4 | 4 | 3 | 5 |
| B: Second Decision Family | 15/25 | 3 | 4 | 2 | 4 | 2 |
| D: Portfolio Risk | 15/25 | 4 | 2 | 3 | 3 | 3 |
| C: Multi-Symbol Hardening | 13/25 | 2 | 4 | 1 | 5 | 1 |
| E: Operational Maturity | 12/25 | 3 | 2 | 2 | 3 | 2 |

---

## 4. Dependency Graph

```
Venue Readiness (A)
  └── Portfolio Risk (D) — needs real fills/positions
  └── Operational Maturity (E) — meaningful only with real venue data
  └── Actor Concurrency Hardening (subset of C) — venue load reveals real concurrency patterns

Second Decision Family (B) — independent, can happen anytime
Multi-Symbol Hardening (C) — independent, diminishing returns
```

---

## 5. Recommendation

### Primary: Option A — Venue Readiness Charter

**Rationale**:
1. Venue readiness is the single largest remaining capability gap between paper operation and production.
2. It sits at the root of the dependency graph — portfolio risk, operational maturity, and real concurrency testing all depend on understanding real venue behavior.
3. The Foundry has strong prerequisites: paper execution proven (S264–S274), multi-symbol isolation confirmed (S300–S304), venue adapter skeleton exists (S90–S93).
4. Deferring venue readiness means every other wave operates on paper assumptions that may not hold under real exchange conditions.

**Scope guard-rails for venue readiness charter**:
- Single exchange adapter (e.g., Binance paper trading API) — not multiple exchanges.
- Order submission and fill reception only — no OMS, no position tracking aggregation.
- Existing pipeline unchanged — venue adapter replaces paper fill stub only.
- No portfolio-level risk — per-symbol risk remains as-is.
- No new decision/strategy/risk families.

### Secondary: None

No secondary wave should be opened. The venue readiness charter is integration-heavy and will surface unexpected complexity. Opening a concurrent wave violates the project's single-front discipline and creates scope management risk.

---

## 6. What Explicitly NOT to Open Now

| Wave | Reason to Defer |
|------|-----------------|
| Second Decision Family (B) | Not a blocker; codegen path already proven; adds complexity during venue integration |
| Multi-Symbol Hardening (C) | Diminishing returns; existing proof is sufficient; real concurrency patterns will emerge naturally from venue work |
| Portfolio Risk (D) | Blocked by venue readiness — portfolio risk without real positions and fills is speculative |
| Operational Maturity (E) | Meaningful only with real venue data flowing; paper-mode dashboards provide false confidence |
| Runtime Config Wave | Low severity; static scaling factors are acceptable until venue integration reveals tuning needs |
| Write-Side Validation | Read-side proven; write-side concerns become relevant only under real venue load |
