# Family 04 Trigger Assessment

## Purpose

Formal evaluation of whether the triggers predicted at S167, S173, and throughout Wave B have been activated by the Family 03 expansion. This document decides whether Family 04 (Risk Assessments) can proceed, must wait for hardening, or requires architectural redesign.

## Assessment Date

Post-S177 (Family 03 end-to-end validation complete).

---

## 1. Trigger Inventory

The following triggers were committed or predicted across S167, S173, and the Wave B pattern documents.

### T-D4: Codegen Evaluation (Committed — Activates at Family 04)

**Origin:** S167 (D-4), reaffirmed in S173 gate, pattern v2.

**Evidence from Family 03:**
- 4 readers averaging 138 lines each, ~80% structurally identical.
- 4 handler methods averaging ~80 lines each, ~85% structurally identical.
- 4 use cases averaging ~60 lines each, ~90% structurally identical.
- Compose DI block: 4 reader lines + 4 use case lines, purely mechanical.
- Contracts: 4 request/reply struct pairs following identical shape.

**Assessment: TRIGGERED — but not blocking.**

The duplication is real, measured, and growing linearly. However:
- The duplication is *stable* — each new family adds a predictable, bounded increment.
- The duplication is *correct* — no bugs have been introduced by copy-paste-modify across 4 families.
- The duplication is *reviewable* — the pattern is simple enough that review catches deviations.
- Codegen would reduce ~800 lines of reader+handler+use case boilerplate to templates, but introduces template maintenance cost.

**Verdict:** Codegen is **justified but not urgent**. The pattern can absorb 1-2 more families before the maintenance burden exceeds the codegen investment cost. Family 04 can proceed without codegen. Codegen becomes **mandatory before Family 06**.

### T-CI: CI Smoke Integration (Predicted blocker — repeatedly flagged)

**Origin:** PF-4/PF-5/PF-6 across Family 01, 02, and 03 validation documents.

**Evidence from current state:**
- `.github/workflows/ci.yml` **already includes** `smoke-analytical` job (lines 26-65).
- The job runs full E2E: `make up` → seed → `make smoke-analytical`.
- All 4 families validated in CI.

**Assessment: NOT TRIGGERED — already resolved.**

The CI smoke integration was flagged as a gap in the Family 01-03 validation documents, but the implementation evidence shows it was resolved (likely during S166 or the hardening tranche). The documentation lagged behind the implementation. This is **no longer a friction or a trigger**.

### T-F3: Friction Count Threshold (>2 new frictions → pause)

**Origin:** S167 gate condition, reaffirmed at each family gate.

**Evidence from Family 03:**
- PF-1 (handler duplication): **not new** — existed since Family 02, severity unchanged.
- PF-2 (smoke test size): **not new** — grew linearly as expected, ~570 lines with reusable helper.
- PF-3 (direction case-sensitive): **new but low severity** — same pattern as `outcome` in Family 02.
- PF-4 (CI gap): **resolved** — see T-CI above.
- PF-5 (no pagination): **not new** — unchanged since Family 01.
- PF-6 (no JSON content verification in smoke): **new but low severity** — cosmetic gap.

**Assessment: NOT TRIGGERED.** Only 2 new frictions (PF-3, PF-6), both low severity. The threshold of >2 new frictions was not crossed.

### T-JSON: JSON Column Ceiling (3 → 4 transition)

**Origin:** Family 03 definition — strategies was chosen specifically to prove 3 JSON columns.

**Evidence:**
- `ParseMetadataJSON` reused across 3 families for map-type JSON columns.
- `ParseSignalInputsJSON` and `ParseDecisionInputsJSON` handle array-type JSON columns identically.
- Family 04 (Risk Assessments) adds 1 JSON column (4 total), following the same parsing pattern.

**Assessment: NOT TRIGGERED as a blocker.** The JSON parsing pattern scales through reuse. 4 JSON columns are mechanically absorbable.

### T-TEXT: Free-Text Column (risk's `rationale` field)

**Origin:** Family 03 selection rationale (T-4).

**Evidence:**
- No free-text (non-enum, non-JSON, non-numeric) columns exist in the current 4 families.
- Risk Assessments introduces `rationale TEXT` — the first free-text column.
- This is structurally simpler than JSON columns (no parsing, direct string scan).

**Assessment: NOT TRIGGERED as a blocker.** Free-text columns are simpler than JSON columns. No pattern change required.

### T-FILTER: Domain-Specific Filter Scaling

**Origin:** Observation across Family 02 (`outcome`) and Family 03 (`direction`).

**Evidence:**
- Each family adds 0-1 optional domain-specific filters.
- Risk Assessments would add `disposition` — same pattern as `outcome` and `direction`.
- Handler parameter parsing for optional filters is ~5 lines of boilerplate.

**Assessment: NOT TRIGGERED.** Filters are mechanical and predictable.

### T-COMPOSE: Constructor/DI Accumulation

**Origin:** Family 02 implementation notes, reaffirmed at Family 03.

**Evidence:**
- Compose DI block: 4 reader lines + 4 use case lines + 1 struct literal = ~14 lines.
- Struct-based DI (H-1) eliminated constructor churn — adding fields is additive only.
- At 6 families: ~21 lines. At 8: ~28 lines. Linear, manageable.

**Assessment: NOT TRIGGERED.** The H-1 refactor resolved the structural concern. Growth is linear and bounded.

---

## 2. Trigger Summary Matrix

| Trigger | Status | Severity | Blocking? | Action Required |
|---------|--------|----------|-----------|----------------|
| D-4 Codegen | **ACTIVATED** | Medium | No | Evaluate before Family 06 |
| CI Smoke | **RESOLVED** | — | No | Update documentation |
| Friction Count | Not triggered | — | No | None |
| JSON Column Ceiling | Not triggered | — | No | None |
| Free-Text Column | Not triggered | — | No | None |
| Filter Scaling | Not triggered | — | No | None |
| Constructor/DI | Not triggered | — | No | None |

---

## 3. Architectural Decision

**Family 04 (Risk Assessments) is authorized to proceed.**

Rationale:
1. Only one trigger activated (D-4 codegen), and it is non-blocking at this scale.
2. The predicted major blocker (CI smoke) was already resolved.
3. No friction threshold was crossed.
4. The pattern has been proven across 4 families with zero correctness regressions.
5. Risk Assessments is the natural next step (layer 5 of 6, 4 JSON columns, pre-staged migration 005).

**Conditions:**
- D-4 codegen evaluation must be documented as part of Family 04 or immediately after.
- Codegen becomes mandatory before Family 06.
- >2 new frictions in Family 04 triggers mandatory hardening pause before Family 05.
- Documentation must be updated to reflect CI smoke resolution (PF-4 closed).

---

## 4. What a Hardening Tranche Would Address (If Chosen)

If the team prefers a hardening tranche before Family 04:

| Item | Effort | Impact | Recommendation |
|------|--------|--------|----------------|
| Codegen for readers/handlers/use cases | High (2-3 days) | Reduces ~800 lines duplication | Defer to post-Family-04 |
| Handler method extraction/generics | Medium (1 day) | Reduces handler from 417 to ~200 lines | Defer — risk of premature abstraction |
| Smoke test parameterization | Low (0.5 days) | Already has `validate_analytical_family()` helper | Already done |
| Filter validation (case-sensitivity) | Low (0.5 days) | Cosmetic improvement | Defer |
| Pagination | Medium (1-2 days) | Not needed at current data volumes | Defer |

**Recommendation: No hardening tranche is necessary before Family 04.** The only activated trigger (codegen) is explicitly non-blocking for the next expansion. All other items are either resolved or low-severity.

---

## 5. Risk Register for Family 04

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| 4 JSON columns introduce parsing edge case | Low | Medium | Reuse proven parsers; test coverage exists |
| Free-text `rationale` column causes scan issues | Low | Low | Simpler than JSON; standard string scan |
| Handler file exceeds comfortable review size | Medium | Low | ~520 lines projected; still reviewable |
| Smoke test grows beyond maintainability | Low | Low | `validate_analytical_family()` helper absorbs growth |
| Codegen debt compounds if Family 05 follows quickly | Medium | Medium | Gate at Family 05 enforces codegen decision |

---

## 6. Implications for S179

S179 should be the **Family 04 (Risk Assessments) definition and contract**, following the established Wave B family expansion pattern v2. It should include:

1. Risk Assessments domain contract (17 DDL columns, 4 JSON columns, `disposition` filter).
2. Success criteria following the 38-criterion template from Family 03.
3. Explicit codegen evaluation checkpoint (D-4 resolution).
4. Updated friction inventory reflecting CI smoke resolution.
5. Gate condition: >2 new frictions in Family 04 → mandatory codegen/hardening before Family 05.
