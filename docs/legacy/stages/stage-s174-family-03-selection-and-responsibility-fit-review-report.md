# Stage S174 — Family 03 Selection and Responsibility Fit Review

> **Objective:** Formally select the third Wave B analytical family through comparative evaluation of all viable candidates, guided by architectural fit, analytical value, incremental complexity, and controlled risk.

---

## 1. Executive Summary

**Family 03 selected: Strategies (mean_reversion_entry).**

Five candidates were evaluated: strategies, risk assessments, executions, EMA crossover, and tradeburst. Strategies is the only candidate that simultaneously advances the read path into the next natural layer (layer 4 of 6), introduces healthy incremental complexity (15 columns, three JSON fields), and maintains contiguous analytical coverage without skipping layers or requiring structural changes.

The selection is defensible on every evaluation criterion: boundary fit, complexity gradient, pattern pressure, analytical value, operational risk, dependency chain position, and infrastructure readiness. All deferred candidates have clear triggers for future consideration. The decision framework eliminates arbitrary expansion and preserves the Wave B pattern's controlled iteration model.

---

## 2. Candidates Evaluated

| Candidate | Domain | Layer | Columns | JSON Columns | Verdict |
|-----------|--------|-------|---------|--------------|---------|
| **Strategies (mean_reversion_entry)** | Strategy | 4 | 15 | 3 | **Selected** |
| Risk Assessments (position_exposure) | Risk | 5 | 17 | 4 | Deferred — premature (coverage gap) |
| Executions (paper_order) | Execution | 6 | 20 | 4 + quantities | Deferred — premature (two-layer gap, max complexity) |
| EMA Crossover | Signal | 2 | — | — | Deferred — not a family expansion (within-layer variant) |
| Tradeburst | Evidence | 1 | — | — | Deferred — incomplete infrastructure (no write path) |

Full comparison in [`family-03-candidate-comparison-matrix.md`](../architecture/family-03-candidate-comparison-matrix.md).

---

## 3. Family 03 Selected: Strategies — Rationale

### Why strategies

1. **Next natural layer.** Read path currently covers evidence → signal → decision (layers 1-3). Strategies is layer 4 — the next link in the analytical dependency chain. Adding it completes the "evaluate → decide → resolve" analytical surface.

2. **Healthy complexity increment.** 15 columns vs decisions' 14. Adds three JSON columns (vs two), a `direction` enum filter (following decisions' `outcome` pattern), and `DecisionInput` JSON array (following decisions' `SignalInput` pattern). Every new dimension has a direct precedent.

3. **Productive pattern pressure.** Tests three JSON columns for the first time. If the scan/parse pattern holds, risk assessments (four JSON columns) becomes lower-risk as Family 04. The pattern is tested at the right granularity.

4. **High readiness.** Migration 004 exists, writer mapper exists, pipeline consuming. Only read path artifacts needed — consistent with Family 01 and 02 scope.

5. **High analytical value.** Strategy resolution history answers critical operational questions: "What strategies resolved? What directions? What decisions drove them?" This is the natural next query after reviewing decision outcomes.

6. **No coverage gap.** Contiguous read path maintained through layers 1-4.

### What strategies tests about the pattern

- Three JSON columns (decisions, parameters, metadata) — scale test
- Second domain-specific enum filter (`direction`) — pattern reuse test
- Fourth `AnalyticalHandlerDeps` field — struct DI scale test
- Fourth `validate_analytical_family()` call — smoke parameterization scale test
- Schema coherence across 4 simultaneous families — review overhead test

Full rationale in [`family-03-selection-rationale-and-deferred-candidates.md`](../architecture/family-03-selection-rationale-and-deferred-candidates.md).

---

## 4. Deferred Candidates — Why Not Now

### Risk Assessments → Family 04

- **Coverage gap:** Adding risk before strategies skips layer 4, breaking contiguous read path
- **Complexity jump:** 17 columns with four JSON columns + free-text `rationale` — too large a step from decisions
- **Pattern sequence:** Three JSON columns (strategies) should be proven before four (risk)
- **Trigger:** Strategies gate passes

### Executions → Family 05

- **Two-layer gap:** Skips strategy AND risk layers
- **Maximum complexity:** 20 columns, quantity fields, fill arrays, execution-specific IDs
- **Terminal position:** Should be the capstone after all upstream layers are covered
- **Trigger:** Risk assessments gate passes

### EMA Crossover → Not scheduled

- **Not a family expansion.** Existing signal reader handles it via type discrimination. Enabling it is a writer config change, not a 9-artifact expansion unit.
- **Tests nothing new.** Zero pattern pressure, zero architectural learning.

### Tradeburst → Not scheduled

- **Incomplete infrastructure.** No writer mapper, no migration, no pipeline entry. Write-path-first principle violated.
- **Within-layer deepening.** Horizontal depth deferred until vertical coverage is complete.

---

## 5. Risks and Limits

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Three JSON columns reveal parse bottleneck | Low | Low | Independent parsing per column |
| Schema coherence at 4 families exceeds review capacity | Low | Medium | Unit test assertions on row/column counts |
| Direction filter semantics differ from outcome | Low | Low | Both are LowCardinality string enums |
| Friction count exceeds threshold (>2 new) | Low | Medium | Triggers hardening pause per S167 rule |

### Limits

- This selection authorizes **definition and planning** of the strategies family. Implementation follows in subsequent stages.
- The selection does NOT authorize Family 04 or beyond — each requires its own gate review.
- Codegen evaluation (D-4) remains deferred to Family 04 per pattern v2 commitment.
- CI smoke integration (PF-5) remains an unresolved gap, growing with each family.

---

## 6. Preparation for S175

S175 should execute the **Strategies Family Expansion Definition**, following the precedent established by S163 (Signals) and S168 (Decisions):

### Recommended S175 scope

1. **Schema coherence table** — Full DDL ↔ mapper ↔ reader column alignment for strategies
2. **Endpoint specification** — `GET /analytical/strategy/history` with query parameters, response contract, error codes
3. **Query parameter design** — `direction` as optional filter; case sensitivity decision; validation rules
4. **Data flow diagram** — NATS → writer → ClickHouse → reader → use case → handler → HTTP
5. **Success criteria** — What must pass for strategies to be considered delivered
6. **Non-goals** — What strategies explicitly does NOT include
7. **Known limits** — Simplifications inherited from the pattern; deferred validations

### S175 should NOT include

- Implementation of any artifact (that's S176+)
- Schema changes or new migrations
- Writer modifications
- CI or infrastructure changes

---

## 7. Artifacts Produced

| Artifact | Path | Purpose |
|----------|------|---------|
| Candidate comparison matrix | `docs/architecture/family-03-candidate-comparison-matrix.md` | Formal multi-criteria comparison of all viable candidates |
| Selection rationale and deferred candidates | `docs/architecture/family-03-selection-rationale-and-deferred-candidates.md` | Detailed justification for selection and deferral decisions |
| Selection and responsibility fit review | `docs/architecture/family-03-selection-and-responsibility-fit-review.md` | Architectural responsibility analysis and pattern fit assessment |
| Stage report (this document) | `docs/stages/stage-s174-family-03-selection-and-responsibility-fit-review-report.md` | Executive summary and stage record |

---

## 8. Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|---------|
| Family 03 chosen with explicit criteria | **PASS** | 7 criteria applied across 5 candidates in comparison matrix |
| Choice is architecturally defensible | **PASS** | Contiguous layer coverage, healthy complexity gradient, dependency chain respected |
| Deferred candidates well justified | **PASS** | Each deferral has explicit reason, trigger for reconsideration, and future position |
| Risk of arbitrary expansion reduced | **PASS** | Selection framework eliminates "interest-based" choices; ordering follows layer progression |
| Base ready for formal family definition | **PASS** | S175 scope defined; pattern v2 template proven; all blockers cleared in S172 |
