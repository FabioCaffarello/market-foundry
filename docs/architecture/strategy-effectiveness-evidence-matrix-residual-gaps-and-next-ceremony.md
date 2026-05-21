# Strategy Effectiveness Evidence Matrix, Residual Gaps, and Next Ceremony

**Wave**: Strategy Effectiveness Measurement (S474--S478)
**Gate stage**: S478
**Date**: 2026-03-25

---

## 1. Evidence Matrix

### 1.1 Capability Ratings

| ID | Capability | Target | Achieved | Delta |
|----|-----------|--------|----------|-------|
| C-SE1 | Canonical effectiveness outcome model | FULL | **FULL** | 0 |
| C-SE2 | P&L attribution per decision chain | FULL | **FULL** | 0 |
| C-SE3 | Effectiveness computation from existing data | FULL | **FULL** | 0 |
| C-SE4 | Batch effectiveness evaluation endpoint | FULL | **FULL** | 0 |
| C-SE5 | Effectiveness in DecisionReviewBundle | FULL | **FULL** | 0 |
| C-SE6 | Cohort-level effectiveness aggregation | FULL | **FULL** | 0 |
| C-SE7 | Comparative analysis by dimension | FULL | **FULL** | 0 |

**Summary**: 7 FULL / 0 SUBSTANTIAL / 0 PARTIAL / 0 PENDING.

### 1.2 Governing Questions

| ID | Question | Answered | Stage |
|----|----------|----------|-------|
| Q-SE1 | Classify decision chains as win/loss/breakeven | YES | S476 (domain types from S475 inline) |
| Q-SE2 | Attribute P&L to originating decision | YES | S476 |
| Q-SE3 | Computable from existing data | YES | S476 |
| Q-SE4 | Batch-evaluate across cohorts | YES | S476 |
| Q-SE5 | Comparative effectiveness analysis | YES | S477 |

**Summary**: 5/5 governing questions answered YES.

### 1.3 Test Coverage

| Stage | Tests Added | Cumulative | Regressions |
|-------|------------|------------|-------------|
| S474 | 0 (charter) | 0 | 0 |
| S476 | 30 (15 domain + 15 use case) | 30 | 0 |
| S477 | 15 (use case) | 45 | 0 |
| S478 | 0 (gate) | 45 | 0 |
| **Total** | **45** | **45** | **0** |

### 1.4 Artifact Inventory

| Artifact | Type | Stage | Location |
|----------|------|-------|----------|
| Wave charter and scope freeze | Architecture | S474 | `docs/architecture/strategy-effectiveness-measurement-wave-charter-and-scope-freeze.md` |
| Capabilities, questions, non-goals | Architecture | S474 | `docs/architecture/strategy-effectiveness-capabilities-questions-and-non-goals.md` |
| Effectiveness domain package | Code | S476 | `internal/domain/effectiveness/effectiveness.go` |
| Effectiveness domain tests | Test | S476 | `internal/domain/effectiveness/effectiveness_test.go` |
| Effectiveness contracts | Code | S476 | `internal/application/analyticalclient/effectiveness_contracts.go` |
| Effectiveness use case | Code | S476 | `internal/application/analyticalclient/get_effectiveness.go` |
| Effectiveness use case tests | Test | S476 | `internal/application/analyticalclient/s476_effectiveness_test.go` |
| Measurement read surfaces doc | Architecture | S476 | `docs/architecture/measurement-read-surfaces-and-batch-evaluation.md` |
| Effectiveness query surfaces doc | Architecture | S476 | `docs/architecture/effectiveness-query-surfaces-batch-evaluation-inputs-outputs-and-limitations.md` |
| Effectiveness summary use case | Code | S477 | `internal/application/analyticalclient/get_effectiveness_summary.go` |
| Effectiveness summary tests | Test | S477 | `internal/application/analyticalclient/s477_effectiveness_review_test.go` |
| Decision effectiveness review doc | Architecture | S477 | `docs/architecture/decision-effectiveness-review-and-comparative-analysis.md` |
| Comparison semantics doc | Architecture | S477 | `docs/architecture/effectiveness-review-comparison-semantics-interpretation-and-limitations.md` |
| Evidence gate | Architecture | S478 | `docs/architecture/strategy-effectiveness-evidence-gate.md` |
| This document | Architecture | S478 | (this file) |

### 1.5 HTTP Surface Inventory

| Endpoint | Method | Stage | Purpose |
|----------|--------|-------|---------|
| `/analytical/composite/decision/effectiveness` | GET | S476 | Single-chain effectiveness lookup |
| `/analytical/composite/decision/effectiveness/batch` | GET | S476 | Batch effectiveness evaluation |
| `/analytical/composite/decision/effectiveness/summary` | GET | S477 | Cohort aggregation and comparison |

### 1.6 Guard Rails (Wave-Wide)

All 10 guard rails observed across all stages. No violations, no waivers, no exceptions.

---

## 2. Residual Gaps

### 2.1 Honest Assessment

| ID | Gap | Severity | Why It Exists | Impact | Mitigation |
|----|-----|----------|---------------|--------|------------|
| G-SE1 | **Single-leg fills dominate outcomes.** Most evaluations return `unresolved` because there is no paired exit within session scope. | MEDIUM | The pipeline processes individual orders, not round-trip trade pairs. Pairing requires cross-correlation or session-end mark logic not in wave scope. | Win rate and P&L metrics are based on a small subsample of resolved chains. | `ClassifyPair()` is available programmatically. Pairing logic can be added in a future wave without changing the domain model. |
| G-SE2 | **No statistical significance on cohort comparisons.** Differences between cohorts may be noise. | LOW | Statistical testing (p-values, confidence intervals) was explicitly excluded (NG-SE2 boundary). | Operators may over-interpret small differences. | The `effectiveness-review-comparison-semantics-interpretation-and-limitations.md` document explicitly warns about this. |
| G-SE3 | **Futures fees are zero.** S428 limitation means futures fee impact is understated in all effectiveness metrics. | LOW | Fee normalization for futures was incomplete in the prior OMS wave. Not addressable within this wave's scope. | Fee-related metrics (`total_fees`, net P&L after fees) are inaccurate for futures segment. | Spot fees are accurate. Futures fee correction is a write-path fix outside this wave. |
| G-SE4 | **No temporal decomposition within a query.** Summary aggregates over the full time range without hourly/daily breakdown. | LOW | Time-series decomposition was not in charter scope. The summary endpoint answers "what happened?" not "how did it change over time?" | Operators cannot see effectiveness trends within a single query. | Multiple queries with different `since`/`until` ranges can approximate decomposition. |
| G-SE5 | **No paired matching HTTP endpoint.** `ClassifyPair()` exists in domain code but is not exposed as an HTTP surface. | LOW | Round-trip pairing requires entry/exit correlation that is not yet automated in the read path. | Programmatic users can pair manually; HTTP users cannot. | Future wave can expose a `/effectiveness/pair` endpoint. |
| G-SE6 | **No cross-symbol aggregation.** Each query is scoped to one source/symbol/timeframe partition. | LOW | Explicit non-goal (NG-SE1: portfolio analytics). | Cannot compare effectiveness across symbols in a single request. | By design. Portfolio-level analytics would be a separate wave. |

### 2.2 Gap Severity Summary

| Severity | Count | Blocking? |
|----------|-------|-----------|
| HIGH | 0 | -- |
| MEDIUM | 1 (G-SE1) | NO -- expected pipeline limitation, well-documented |
| LOW | 5 | NO |

**No gaps block the wave verdict.** G-SE1 is the most material gap and is inherent to the current pipeline scope (single-session, single-order processing). It requires architectural expansion beyond this wave's boundaries.

---

## 3. Risk Register Disposition

| Risk (from S474) | Outcome |
|------------------|---------|
| Effectiveness semantics ambiguous for partial fills | MITIGATED -- explicit classification: partially-filled is non-terminal, classified as unresolved |
| Single-session scope limits multi-session attribution | ACCEPTED -- documented as G-SE1, known limitation |
| Fill data lacks exit price for open positions | MITIGATED -- single-leg fills classified as unresolved, `ClassifyPair()` available for paired data |
| Scope inflation toward portfolio analytics | AVOIDED -- all 20 non-goals respected, zero scope expansion |
| Read-path performance on large cohorts | MITIGATED -- scan limit (default 100, max 300) prevents unbounded queries |

---

## 4. What Improved Concretely

Before this wave:
- The system had no concept of decision effectiveness.
- A winning trade was indistinguishable from a losing trade.
- No P&L attribution existed.
- No way to compare decision types or strategies by outcome.

After this wave:
- **Classification**: Every completed decision chain has a canonical outcome (win/loss/breakeven/unresolved).
- **Attribution**: P&L is linked to the originating decision, carrying decision type, strategy type, severity, and correlation context.
- **Batch evaluation**: Operators can scan up to 300 chains with 4 filters and get per-chain effectiveness.
- **Comparative analysis**: Operators can compare cohorts by 4 dimensions with 10 summary metrics.
- **Review integration**: The DecisionReviewBundle now includes an effectiveness section with explanation text.
- **3 HTTP endpoints** added to the analytical surface.
- **45 tests** validating the full effectiveness stack.

---

## 5. Next Ceremony Recommendation

### 5.1 Verdict

The Strategy Effectiveness Measurement wave is **CLOSED with PASS**. No closure sprint is needed.

### 5.2 Next strategic direction

The wave progression so far:
1. **Lineage** (S470) -- "what caused this order?"
2. **Review** (S471) -- "show me the full decision chain"
3. **Consistency** (S472) -- "was the chain internally consistent?"
4. **Effectiveness** (S474--S478) -- "was the decision good?"

The natural next question is operational: **"how is the system performing across sessions and over time?"**

Recommended next macro-front candidates (ordered by evidence-informed priority):

| Candidate | What it addresses | Why now |
|-----------|------------------|---------|
| **Session Lifecycle and Operational Continuity** | Cross-session state, session health metrics, operational dashboards | Addresses G-SE1 (single-session limitation) and bridges to multi-session analytics |
| **Round-Trip Pairing and Resolved Rate Improvement** | Automated entry/exit matching, resolved rate optimization | Directly improves effectiveness utility by increasing resolved chain count |
| **Live Operational Hardening** | Kill-switch improvements, runtime health, operational runbooks | Operational maturity for sustained live trading |

### 5.3 What NOT to do next

- Do not open portfolio analytics (NG-SE1) -- premature without resolved rate improvement.
- Do not add ML/predictive scoring (NG-SE4) -- effectiveness data is too sparse.
- Do not build dashboards (NG-SE6) -- the API surface must stabilize first.
- Do not expand to multi-exchange (NG-SE8) -- single-venue operation is not yet hardened.

---

## 6. References

- [Evidence Gate](strategy-effectiveness-evidence-gate.md)
- [Wave Charter and Scope Freeze](strategy-effectiveness-measurement-wave-charter-and-scope-freeze.md)
- [Capabilities, Questions, and Non-Goals](strategy-effectiveness-capabilities-questions-and-non-goals.md)
- [S474 Charter Report](../stages/stage-s474-strategy-effectiveness-charter-report.md)
- [S476 Measurement Surfaces Report](../stages/stage-s476-measurement-read-surfaces-report.md)
- [S477 Effectiveness Review Report](../stages/stage-s477-effectiveness-review-report.md)
- [S478 Evidence Gate Report](../stages/stage-s478-strategy-effectiveness-evidence-gate-report.md)
