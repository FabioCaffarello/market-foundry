# Restructure Doc Impact and Reconciliation Log

**Stage:** S221
**Date:** 2026-03-20
**Purpose:** Detailed log of every documentation discrepancy found and how it was resolved during the S221 reconciliation.

---

## Discrepancy Log

### DISC-01 (MAJOR): H-01/H-04/H-06 Status Drift

**Source:** `refactor-wave-gains-tradeoffs-and-open-debts.md` (section 3.1)
**Problem:** H-01 listed as NOT STARTED, H-04 as INFRASTRUCTURE READY/MIGRATION DEFERRED, H-06 as NOT STARTED. All three were completed in S218–S220.
**Resolution:** Status updated to DONE with stage references and actual metrics (S218 sub-packaging, S219 510 lines recovered, S220 19→17 modules).
**Impact:** HIGH — this was the primary drift between code and documentation.

### DISC-02 (MAJOR): Debt Disposition Count

**Source:** `refactor-wave-gains-tradeoffs-and-open-debts.md` (section 4)
**Problem:** "Deferred to next expansion wave: 10 — H-01, H-04 migration, H-06, M-01–M-07" — but 3 of those 10 are now done.
**Resolution:** Updated to "7 — M-01–M-07" with note that H-01/H-04/H-06 completed in S218–S220.
**Impact:** HIGH — misleading count of remaining work.

### DISC-03 (MAJOR): Path B Recommendation Status

**Source:** `next-wave-recommendations-after-post-refactor-and-documentation-gate.md` (section 2, Path B)
**Problem:** Path B described as a future recommendation with 2–3 sessions estimated. Path B was already executed.
**Resolution:** Path B section marked COMPLETED with deliverables. Recommendation updated to reflect post-Path-B state (Path A or Path C now viable).
**Impact:** HIGH — reader would plan work already done.

### DISC-04 (MODERATE): Module Count References

**Source:** Multiple documents reference "19 modules" or "19-module workspace".
**Locations reconciled:**
- `post-refactor-and-documentation-exit-gate.md` XC-4: "All 19 Go modules build clean" → "All Go modules build clean (S221: 19→17)"
- `pre-refactor-technical-debt-registry-and-cleanup-plan.md` AD-01: Updated from "19 modules" to "19→17 modules" with S220 details
**Impact:** MODERATE — stale number, but the direction of change is clear.

### DISC-05 (MODERATE): Architecture Doc Count

**Source:** `documentation-canonical-map-after-consolidation.md` (summary table)
**Problem:** Count was 243 (S217 corrected). Actual count is 249 after S219–S221 added 6 new architecture docs.
**Resolution:** Updated to 249 with S221 note explaining the +6: 4 from S219/S220 (actor-infrastructure, h04-completion, h06-simplification, module-graph-before-after) + 2 from S221 (this doc + companion reconciliation doc).
**Impact:** MODERATE — count accuracy for gate compliance tracking.

### DISC-06 (MODERATE): Stage Report Count and Index

**Source:** `docs/stages/INDEX.md` and `documentation-canonical-map-after-consolidation.md`
**Problem:** INDEX showed Phase 18 with only S219. Missing S218 (H-01), S220 (H-06), S221. Stage count was 214.
**Resolution:** Phase 18 renamed to "Structural Refactoring Completion (S218–S221)". S218 entry added (no report file — work documented in S221 reconciliation log). S220 and S221 entries added. Count updated to 219.
**Impact:** MODERATE — incomplete phase record.

### DISC-07 (MODERATE): S213 Assessment Outdated

**Source:** `post-refactor-and-documentation-exit-gate.md` (section 3, S213 assessment)
**Problem:** S213 graded PARTIAL (42% of HIGH items). With S218–S220 completing the remaining items, the combined assessment should reflect full completion.
**Resolution:** Added S221 update noting all deferred HIGH items completed. Grade annotation: "PARTIAL at S213 exit → COMPLETE after S218–S220 tranche".
**Impact:** MODERATE — historical grade preserved but current state clarified.

### DISC-08 (MINOR): H-04 Projected Savings vs Actual

**Source:** `refactor-wave-gains-tradeoffs-and-open-debts.md` (section 1.4)
**Problem:** Listed "Projected savings of ~1,800 lines when per-family actors migrate." Actual savings in S219 were ~510 lines (the 1,800 line estimate was for the entire actor layer; only consumer actors were migrated).
**Resolution:** Section 1.4 updated to reflect actual S219 outcome: 8 files deleted, ~510 lines recovered, projection actors intentionally kept domain-specific.
**Impact:** MINOR — projection was an upper bound; actual was for the appropriate scope.

### DISC-09 (MINOR): Missing S218 Stage Report

**Source:** `docs/stages/`
**Problem:** S218 (H-01 NATS adapter sub-packaging) was executed but no stage report was created. This is the only stage in Phase 18 without a dedicated report.
**Resolution:** S218 entry added to INDEX.md with note that work is documented in this S221 reconciliation log rather than a standalone report. The H-01 details are captured in `post-restructure-documentation-reconciliation.md` section 2.1.
**Impact:** MINOR — traceability gap addressed via cross-reference.

### DISC-10 (MINOR): Net Assessment Text

**Source:** `refactor-wave-gains-tradeoffs-and-open-debts.md` (section 5)
**Problem:** Text said "3.5 of 6 HIGH items remain open" — none remain open after S218–S220.
**Resolution:** Updated to reflect all 6 HIGH items DONE. Remaining gaps narrowed to 7 MEDIUM items + doc count + CI/tag.
**Impact:** MINOR — narrative correction.

---

## Summary of Reconciliation Actions

| Action Type | Count |
|-------------|-------|
| Major discrepancies corrected | 3 |
| Moderate discrepancies corrected | 4 |
| Minor discrepancies corrected | 3 |
| Documents modified | 6 |
| Documents created | 3 (this doc + companion + stage report) |
| Documents archived | 0 |
| Documents deleted | 0 |

---

## Verification Checklist

| Check | Result |
|-------|--------|
| H-01 status consistent across all docs | PASS |
| H-04 status consistent across all docs | PASS |
| H-06 status consistent across all docs | PASS |
| Module count (17) consistent across all docs | PASS |
| Stage INDEX complete for Phase 18 | PASS |
| Doc counts in canonical map reflect reality | PASS |
| No new contradictions introduced | PASS |
| No docs deleted without record | PASS — zero deletions |
| Rastreabilidade preservada | PASS — all changes annotated with stage references |

---

## Scope Boundary

This reconciliation was strictly scoped to documentation affected by the S218–S220 structural tranche. The following were explicitly NOT addressed:

- General documentation consolidation toward ≤150 target (XC-1 — separate effort)
- Cross-document link verification
- Domain subdirectory reorganization
- MEDIUM-priority debt items (M-01 through M-07)
- CI push verification (XC-6) or repository tagging (XC-11)
- Golden snapshot drift resolution
