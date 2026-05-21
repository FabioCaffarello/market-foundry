# Stage S178 — Family 04 Trigger Assessment Report

## Stage Identity

- **ID:** S178
- **Title:** Family 04 Trigger Assessment
- **Type:** Assessment gate (no implementation)
- **Predecessor:** S177 (Family 03 end-to-end validation)
- **Successor:** S179 (Family 04 definition and contract — if authorized)

---

## 1. Executive Summary

S178 evaluated all committed and predicted triggers for Family 04 (Risk Assessments) based on concrete evidence from Family 03 and the cumulative Wave B expansion history.

**Result: Family 04 is authorized to proceed.**

Of 7 evaluated triggers, only 1 was activated (D-4 codegen evaluation), and it is non-blocking. The most frequently flagged concern (CI smoke integration, PF-4) was found to be **already resolved** — the documentation had lagged behind the implementation. No friction threshold was crossed. The pattern is healthy and scalable through at least Family 05.

---

## 2. Triggers Evaluated

| # | Trigger | Verdict | Action |
|---|---------|---------|--------|
| 1 | D-4 Codegen evaluation | **ACTIVATED** (non-blocking) | Evaluated; codegen deferred to Family 06 boundary |
| 2 | CI smoke integration (PF-4) | **RESOLVED** (already in CI) | Close in documentation |
| 3 | Friction count >2 new | **NOT TRIGGERED** (2 new, both low) | Continue monitoring |
| 4 | JSON column ceiling (3→4) | **NOT TRIGGERED** (scales through reuse) | No action |
| 5 | Free-text column (rationale) | **NOT TRIGGERED** (simpler than JSON) | No action |
| 6 | Domain filter scaling | **NOT TRIGGERED** (mechanical) | No action |
| 7 | Constructor/DI accumulation | **NOT TRIGGERED** (H-1 resolved) | No action |

---

## 3. Key Finding: CI Smoke Was Already Resolved

The single most important finding of this assessment is that PF-4 (no CI integration for analytical smoke test) — flagged as **high severity** across three consecutive family validations — was already resolved. The `.github/workflows/ci.yml` file contains a `smoke-analytical` job that validates all 4 families end-to-end.

This means:
- The highest-severity friction in the inventory is closed.
- The argument for a hardening tranche before Family 04 loses its strongest item.
- Documentation accuracy is itself a friction that should be addressed.

---

## 4. Codegen Evaluation (D-4 Resolution)

D-4 was a committed trigger requiring formal evaluation at Family 04. The evaluation finds:

- **Duplication is real:** ~800 lines of ~80% structurally identical code across readers, handlers, and use cases.
- **Duplication is stable:** Each family adds a predictable, bounded increment (~450-500 lines).
- **Duplication is correct:** Zero bugs introduced by copy-paste-modify across 4 families.
- **Codegen threshold:** Cost-effective at 6+ families. Before that, template maintenance cost exceeds duplication cost.

**D-4 is resolved.** New committed trigger: codegen implementation mandatory before Family 06.

---

## 5. Items Triggered vs Deferred

### Triggered (2 items)
1. **Codegen evaluation** — evaluated and resolved. Codegen deferred to Family 06 with committed trigger.
2. **Documentation correction** — PF-4 must be marked as resolved in all future documents.

### Deferred with committed triggers (4 items)
1. Codegen implementation → Family 06 boundary
2. Schema coherence compile-time verification → ~12 analytical tables
3. Handler file split → ~600 lines (Family 06)
4. Friction count gate → >2 new frictions in any family

### Deferred without triggers (9 items)
- Filter case-sensitivity, pagination, consumer lag, sticky degradation, silent mapper fallbacks, backoff jitter, smoke JSON content verification, consumer/inserter naming, metadata validation.

### Resolved (5 items)
- D-1 naming, D-2 struct DI, D-3 smoke extraction, PF-4 CI smoke, D-4 codegen evaluation.

**No item blocks Family 04.**

---

## 6. Pattern Scalability Assessment

The Wave B 9-artifact expansion pattern is healthy through Family 05:
- Linear growth with bounded increments (~450-500 lines per family).
- No cross-family dependencies.
- Write path stable (zero changes across 4 expansions).
- Struct DI eliminates constructor churn.
- Smoke helper absorbs new families mechanically.
- CI validates all families E2E.

The pattern needs codegen at Family 06 — not before.

---

## 7. Risks and Limits

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| 4 JSON columns introduce edge case | Low | Medium | Reuse proven parsers |
| Handler file grows beyond review comfort | Medium | Low | ~500 lines projected; split at ~600 |
| Codegen debt compounds if Family 05 follows quickly | Medium | Medium | Hard gate at Family 06 |
| Documentation drift continues | Medium | Low | S178 corrects; future stages verify |

---

## 8. Decision: Family 04 Authorization

**AUTHORIZED — with conditions.**

### Conditions for Family 04 (Risk Assessments):
1. Follow Wave B pattern v2 with full 9-artifact checklist.
2. Respect all 9 S162 constraints.
3. D-4 codegen evaluation is resolved (this document).
4. PF-4 CI smoke is closed (this document).
5. >2 new frictions in Family 04 → mandatory codegen/hardening before Family 05.
6. Codegen implementation becomes mandatory before Family 06.

### What Family 04 must NOT do:
- Introduce codegen (premature at 5 families).
- Skip the E2E validation stage.
- Modify existing families.
- Add cross-family features.
- Add pagination, aggregation, or materialized views.

---

## 9. Preparation Recommended for S179

S179 should be the **Family 04 (Risk Assessments) selection and responsibility fit review** or, if selection is already confirmed by S174's deferred candidate ranking, the **Family 04 definition and analytical contract**.

### Pre-staged artifacts (already exist):
- Migration 005: `deploy/migrations/005_create_risk_assessments.sql`
- Writer mapper: `cmd/writer/mappers.go` (risk mapper function)
- Writer pipeline entry: pre-configured in pipeline/supervisor

### Artifacts to build:
1. ClickHouse reader (`internal/adapters/clickhouse/risk_reader.go`)
2. Use case (`internal/application/analyticalclient/get_risk_history.go`)
3. Contracts update (`internal/application/analyticalclient/contracts.go`)
4. Handler method (add to `internal/interfaces/http/handlers/analytical.go`)
5. Route registration (add to `internal/interfaces/http/routes/analytical.go`)
6. Compose DI wiring (add to `cmd/gateway/compose.go`)
7. Tests (reader + use case + handler)
8. Smoke test extension (add `validate_analytical_family` call)

### New pattern elements to validate:
- 4 JSON columns (highest count yet)
- `rationale TEXT` — first free-text column
- `disposition` filter — follows `outcome`/`direction` pattern
- 17 DDL columns (highest column count yet)

---

## 10. Deliverables Produced

| Deliverable | Path |
|-------------|------|
| Trigger assessment (primary) | `docs/architecture/family-04-trigger-assessment.md` |
| Pattern scalability analysis | `docs/architecture/wave-b-pattern-scalability-after-family-03.md` |
| Triggered vs deferred inventory | `docs/architecture/triggered-vs-deferred-items-before-family-04.md` |
| Stage report (this document) | `docs/stages/stage-s178-family-04-trigger-assessment-report.md` |

---

## 11. Stage Outcome

**S178 COMPLETE.**

The trigger assessment confirms the analytical layer is scaling healthily. Family 04 is authorized with conditions. No hardening tranche is required before the next expansion. The only structural intervention needed (codegen) has a clear, committed activation point at Family 06.
