# Documentation Entropy: Archive / Delete / Consolidate Map

**Stage:** S209
**Date:** 2026-03-20
**Status:** Registry — no deletions or moves executed. This is the plan.

---

## 1. Purpose

This document maps the entropy in `docs/architecture/` (440 files) and `docs/stages/` (205 files), classifying every document cluster by recommended action: **archive**, **delete**, **consolidate**, or **keep as-is**. It provides the criteria, the rationale, and the execution order for the documentation cleanup phase.

---

## 2. Classification Criteria

| Action | Criteria | Reversibility |
|--------|----------|---------------|
| **KEEP** | Active reference, no redundancy, still authoritative | N/A |
| **CONSOLIDATE** | Multiple docs covering the same topic; merge into one authoritative doc | Medium — originals archived before merge |
| **ARCHIVE** | Historically valuable but superseded or no longer active reference; move to `docs/archive/` | High — preserved, just relocated |
| **DELETE** | Redundant with another doc that fully subsumes it; no unique content | Low — but backed by git history |

**Safety rules:**
- No document is deleted without verifying its content is fully subsumed elsewhere.
- No document is treated as obsolete solely because it is long.
- Archive is the default safe action when in doubt.
- All actions are reversible via git history.

---

## 3. Cluster Analysis and Recommendations

### 3.1 "Next-Wave Recommendations" Cluster (15 files) → CONSOLIDATE

**Files:**
- `next-wave-recommendations-after-vertical-slice-01.md`
- `next-wave-recommendations-after-live-baseline.md`
- `next-wave-recommendations-after-capability-01.md`
- `next-wave-recommendations-after-cc-02.md`
- `next-wave-recommendations-after-timeframe-coverage-01.md`
- `next-wave-recommendations-after-current-capability-consolidation.md`
- `next-wave-recommendations-after-analytical-runtime-entry.md`
- `next-wave-recommendations-after-analytical-wave-a.md`
- `next-wave-recommendations-after-pre-wave-b-gate.md`
- `next-wave-recommendations-after-wave-b-iteration-01.md`
- `next-wave-recommendations-after-post-hardening-wave-b-gate.md`
- `next-wave-recommendations-after-family-03-wave-b-gate.md`
- `next-wave-recommendations-after-family-05-pre-family-06-gate.md`
- `next-wave-recommendations-after-pre-generated-family-gate.md`
- `next-wave-recommendations-after-post-generated-family-gate.md`

**Problem:** 15 separate forward-looking docs, each a snapshot at a gate boundary. No cross-linking. Finding which recommendations were actually executed requires reading all 15.

**Recommendation:** Consolidate into a single `next-wave-recommendations-timeline.md` that presents all gate recommendations chronologically with status annotations (EXECUTED / DEFERRED / SUPERSEDED). Archive the 15 originals.

**Value:** HIGH — eliminates the most confusing scatter pattern in the doc tree.

---

### 3.2 Per-Family Lifecycle Docs (families 03–06) → CONSOLIDATE

**Clusters:**

**Family 03 (11 files):**
- `family-03-candidate-comparison-matrix.md`
- `family-03-selection-and-responsibility-fit-review.md`
- `family-03-selection-rationale-and-deferred-candidates.md` (appears twice — possible duplicate)
- `family-03-definition-and-analytical-contract.md`
- `family-03-implementation-notes.md`
- `family-03-end-to-end-validation.md`
- `family-03-validation-findings-and-pattern-frictions.md`
- `family-03-success-criteria-and-operability-scope.md`
- `family-03-blockers-and-hardening-success-criteria.md`
- `family-03-runtime-and-operability-notes.md`
- `family-03-schema-writer-reader-gateway-contract.md`

**Family 04 (8 files):**
- `family-04-trigger-assessment.md`
- `family-04-definition-and-responsibility-fit.md`
- `family-04-implementation-notes.md`
- `family-04-end-to-end-validation.md`
- `family-04-validation-findings-and-pattern-frictions.md`
- `family-04-success-criteria-risks-and-out-of-scope.md`
- `family-04-runtime-operability-and-boundary-notes.md`
- `family-04-schema-writer-reader-gateway-scope.md`

**Family 05 (11 files):**
- `family-05-trigger-assessment.md`
- `family-05-candidate-comparison-and-pressure-matrix.md`
- `family-05-selection-confirmation-and-responsibility-fit.md`
- `family-05-selection-rationale-and-deferred-candidates.md`
- `family-05-definition-and-analytical-contract.md`
- `family-05-implementation-notes.md`
- `family-05-end-to-end-validation-and-ceiling-evidence.md`
- `family-05-pattern-frictions-cost-and-scalability-findings.md`
- `family-05-success-criteria-risks-and-non-goals.md`
- `family-05-runtime-operability-and-boundary-notes.md`
- `family-05-schema-writer-reader-gateway-contract.md`

**Family 06 (4 files):**
- `family-06-trigger-assessment-and-candidate-selection.md`
- `family-06-candidate-comparison-matrix.md`
- `family-06-selection-rationale-or-abort-rationale.md`
- `family-06-blockers-and-hardening-success-criteria.md`

**Total: ~34 files across 4 families.**

**Problem:** Identical lifecycle structure repeated per family. Most content is formulaic (same sections, same decision framework). Unique content is the family-specific data, which is a small fraction of each doc.

**Recommendation:** For each family, consolidate into a single `family-XX-lifecycle-record.md` that contains all phases (selection → definition → implementation → validation → findings). Archive the originals. Potentially create a `family-lifecycle-template.md` for future families.

**Value:** HIGH — reduces 34 files to 4 + 1 template.

---

### 3.3 Wave B Pattern and Iteration Docs (24 files) → CONSOLIDATE + ARCHIVE

**Files include:**
- `wave-b-family-expansion-pattern.md` → **ARCHIVE** (superseded by v2)
- `wave-b-family-expansion-pattern-v2.md` → **KEEP** (canonical)
- `wave-b-family-checklist-schema-writer-reader-gateway-tests-runbook.md` → **KEEP** (operational)
- `wave-b-iteration-constraints-and-non-goals.md` → **KEEP** (scope boundary)
- `wave-b-family-01-*.md` (5 files) → **CONSOLIDATE** into `wave-b-family-01-lifecycle-record.md`
- `wave-b-family-02-decisions-*.md` (7 files) → **CONSOLIDATE** into `wave-b-family-02-lifecycle-record.md`
- `wave-b-iteration-01-*.md` (2 files) → **CONSOLIDATE** into `wave-b-iteration-01-summary.md`
- `wave-b-pattern-hardening-after-family-01.md` → **ARCHIVE** (historical)
- `wave-b-pattern-scalability-after-family-03.md` → **ARCHIVE** (historical)
- `wave-b-pattern-scalability-after-family-04.md` → **ARCHIVE** (historical)
- `wave-b-after-family-02-and-hardening-gains-tradeoffs-and-open-debts.md` → **ARCHIVE**
- `wave-b-after-family-03-gains-tradeoffs-and-open-debts.md` → **ARCHIVE**
- `wave-b-after-family-05-gains-tradeoffs-and-open-debts.md` → **ARCHIVE**

**Value:** HIGH — reduces 24 files to ~6 active docs.

---

### 3.4 Analytical Infrastructure Docs (28 files) → CONSOLIDATE + ARCHIVE

**Consolidation targets:**

**Boundary/Responsibility cluster (consolidate into 1):**
- `analytical-boundaries-writer-reader-gateway-migrate-observability.md`
- `analytical-boundary-hardening-writer-reader-gateway.md`
- `analytical-responsibility-map-writer-reader-pipeline-observability.md`
- `analytical-responsibility-review-and-restructuring-plan.md`
- `analytical-responsibility-anti-patterns-and-non-goals.md`
- `analytical-contracts-and-adapter-boundaries.md`
→ Consolidate into: `analytical-boundary-and-responsibility-model.md`

**Runtime/Lifecycle cluster (consolidate into 1):**
- `analytical-runtime-activation-rules-and-failure-modes.md`
- `analytical-runtime-optionality-rules.md`
- `analytical-pipeline-lifecycle-degraded-dead-recovered.md`
- `analytical-pipeline-recovery-and-supervision.md`
→ Consolidate into: `analytical-runtime-lifecycle-and-recovery.md`

**Observability cluster (consolidate into 1):**
- `analytical-observability-and-diagnostics-hardening.md`
- `analytical-read-path-observability-and-reliability.md`
- `analytical-read-path-runbook-and-signal-interpretation.md`
- `analytical-runtime-runbook-and-signal-interpretation.md`
→ Consolidate into: `analytical-observability-and-runbook.md`

**Gains/Tradeoffs cluster (consolidate into 1):**
- `analytical-wave-a-gains-tradeoffs-and-open-debts.md`
- `analytical-runtime-gains-tradeoffs-and-open-debts.md`
- `analytical-wave-a-scope-blockers-and-non-goals.md`
→ Consolidate into: `analytical-gains-tradeoffs-and-debts-summary.md`

**Keep as-is:**
- `analytical-implementation-closure.md` (S206, authoritative closure)
- `analytical-writer-correctness-and-test-foundation.md` (test reference)
- `analytical-closure-open-vs-closed-items.md` (closure checklist)
- `analytical-config-and-startup-validation-hardening.md` (config reference)
- `analytical-storage-strategy.md` (strategy doc)
- `analytical-wave-a-hardening-plan.md` (plan reference)

**Remaining files:** Archive after verifying content subsumed.

**Value:** HIGH — reduces ~28 files to ~10 active docs.

---

### 3.5 Codegen Docs (16 files) → CONSOLIDATE

**Consolidation targets:**

**Spec & Schema (consolidate into 1):**
- `codegen-specification-freeze.md`
- `codegen-spec-schema-fields-invariants-and-ownership.md`
- `codegen-equivalence-scope-semantic-vs-structural-rules.md`
→ Consolidate into: `codegen-specification-and-schema.md`

**Validation & Drift (consolidate into 1):**
- `codegen-golden-outputs-and-comparison-strategy.md`
- `codegen-drift-findings-and-equivalence-results.md`
- `codegen-validation-drift-and-ci-strategy.md`
- `codegen-slice-01-ci-validation-strategy.md`
- `codegen-slice-01-coverage-and-non-coverage.md`
→ Consolidate into: `codegen-validation-and-ci-strategy.md`

**Boundaries & Governance (consolidate into 1):**
- `codegen-manual-vs-generated-boundaries.md`
- `codegen-anti-patterns-non-goals-and-human-decision-boundaries.md`
- `codegen-source-of-truth-artifact-coverage-and-ownership.md`
→ Consolidate into: `codegen-boundaries-and-governance.md`

**Keep as-is:**
- `codegen-path-stabilization-or-freeze-decision.md` (S207, authoritative decision)
- `codegen-current-usage-boundaries-and-limitations.md` (latest state)
- `codegen-next-phase-readiness-or-freeze-conditions.md` (forward reference)
- `codegen-tranche-scoping.md` (scope definition)
- `codegen-readiness-gains-tradeoffs-and-open-debts.md` (retrospective)

**Value:** MEDIUM-HIGH — reduces 16 files to ~8.

---

### 3.6 Gains/Tradeoffs/Open-Debts Pattern (17+ files) → CONSOLIDATE

**Files with `*-gains-tradeoffs-and-open-debts*` pattern:**
- `capability-01-gains-tradeoffs-and-open-debts.md`
- `cc-02-gains-tradeoffs-and-open-debts.md`
- `current-capability-consolidation-gains-tradeoffs-and-open-debts.md`
- `live-baseline-gains-tradeoffs-and-open-debts.md`
- `platform-gains-tradeoffs-and-open-debts.md`
- `structural-gains-tradeoffs-and-open-debts.md`
- `vertical-slice-01-gains-tradeoffs-and-open-debts.md`
- `analytical-wave-a-gains-tradeoffs-and-open-debts.md`
- `analytical-runtime-gains-tradeoffs-and-open-debts.md`
- `codegen-readiness-gains-tradeoffs-and-open-debts.md`
- `generated-path-gains-tradeoffs-and-open-debts.md`
- `pre-wave-b-analytical-gains-tradeoffs-and-open-debts.md`
- `timeframe-coverage-01-gains-tradeoffs-and-open-debts.md`
- `wave-b-iteration-01-gains-tradeoffs-and-open-debts.md`
- `wave-b-after-family-02-and-hardening-gains-tradeoffs-and-open-debts.md`
- `wave-b-after-family-03-gains-tradeoffs-and-open-debts.md`
- `wave-b-after-family-05-gains-tradeoffs-and-open-debts.md`

**Recommendation:** Consolidate into a single `gains-tradeoffs-and-open-debts-timeline.md` that captures the evolution chronologically. Archive the originals. Open debts that remain unresolved should be promoted to the technical debt registry.

**Value:** HIGH — these files are the primary source of scattered deferred work items.

---

### 3.7 Triggered/Deferred Refactor Docs (8+ files) → CONSOLIDATE

**Files:**
- `refactors-deferred-after-vertical-slice-01.md`
- `refactors-deferred-after-live-pipeline.md`
- `refactors-deferred-after-capability-01.md`
- `refactors-still-deferred-after-cc-02.md`
- `refactors-still-deferred-after-timeframe-coverage-01.md`
- `triggered-refactors-after-cc-02.md`
- `triggered-refactors-after-timeframe-coverage-01.md`
- `triggered-vs-deferred-items-before-family-04.md`
- `triggered-vs-deferred-items-before-family-05.md`
- `triggered-vs-deferred-hardening-items-after-family-05.md`
- `evidence-driven-refactors-after-vertical-slice-01.md`
- `evidence-driven-surgical-refactors-after-capability-01.md`
- `bounded-pain-refactors-after-live-pipeline.md`

**Problem:** 13 files tracking deferred work across phases. Items accumulate and carry forward but aren't consolidated. The same item (e.g., reader signature) appears in multiple files.

**Recommendation:** Merge all into a single `deferred-work-registry.md` with status tracking. This replaces the scattered approach with a single authoritative list. Archive originals.

**Value:** VERY HIGH — this is the most operationally impactful consolidation.

---

### 3.8 Gate/Readiness Review Docs (20+ files) → ARCHIVE most

**Files matching `post-*-readiness-review.md` and `*-gate.md` patterns:**
- `post-vertical-slice-01-architectural-readiness-review.md`
- `post-live-architectural-and-refactoring-readiness-review.md`
- `post-capability-01-readiness-review.md`
- `post-cc-02-extensibility-readiness-review.md`
- `post-consolidation-readiness-review.md`
- `post-analytical-runtime-entry-readiness-review.md`
- `post-wave-a-analytical-readiness-review.md`
- `post-timeframe-coverage-01-readiness-review.md`
- `post-s100-technical-platform-readiness-review.md`
- `post-paper-action-boundary-readiness-review.md`
- `post-hardening-action-boundary-gate.md`
- `post-hardening-wave-b-gate.md`
- `post-family-03-wave-b-gate.md`
- `post-family-05-pre-family-06-gate.md`
- `post-generated-family-gate.md`
- `pre-wave-b-analytical-readiness-gate.md`
- `pre-generated-family-gate.md`
- `wave-b-iteration-01-gate.md`

**Recommendation:** These are historical audit trail documents. Archive all to `docs/archive/gates/`. They are valuable for compliance/audit but not for active reference.

**Value:** MEDIUM — reduces visual clutter without losing information.

---

### 3.9 Superseded Documents → ARCHIVE with markers

| File | Superseded By | Action |
|------|---------------|--------|
| `wave-b-family-expansion-pattern.md` | `wave-b-family-expansion-pattern-v2.md` | ARCHIVE |
| `execute-runtime-and-activation-model.md` | `execute-governance-and-activation-model.md` | ARCHIVE |
| `family-runtime-registration-rules.md` | `monorepo-structure-and-engineering-conventions.md` | ARCHIVE |
| `strategy-entry-prerequisites.md` | `strategy-entry-prerequisites-rerun.md` | ARCHIVE (keep rerun) |
| `strategy-readiness-review.md` | `strategy-readiness-review-rerun.md` | ARCHIVE (keep rerun) |
| `strategy-risks-and-blockers.md` | `strategy-risks-and-blockers-rerun.md` | ARCHIVE (keep rerun) |
| `first-generated-family-risks-success-criteria-and-non-goals.md` | `first-generated-family-success-criteria-risks-and-non-goals.md` | ARCHIVE (verify content overlap — likely duplicate with name variation) |

---

### 3.10 Domain Design Docs (currently scattered) → KEEP but REORGANIZE

**Files per domain:**
- `signal-domain-design.md`, `signal-first-slice.md`, `signal-projection-pattern.md`, etc.
- `decision-domain-design.md`, `decision-first-slice.md`, `decision-projection-pattern.md`, etc.
- `risk-domain-design.md`, `risk-first-slice.md`, `risk-projection-pattern.md`, etc.
- `execution-domain-design.md`, `execution-first-slice.md`, `execution-projection-pattern.md`, etc.
- `strategy-domain-design.md`, `strategy-first-slice.md`, `strategy-projection-pattern.md`, etc.

**Recommendation:** Move domain-specific docs into subdirectories: `docs/architecture/domains/{signal,decision,risk,execution,strategy}/`. Keep content as-is. This is reorganization, not content change.

**Value:** MEDIUM — improves navigability without content risk.

---

### 3.11 Stage Reports (205 files) → KEEP + ADD INDEX

**Recommendation:** Stage reports are the audit trail. Do NOT delete or consolidate them. Instead:
1. Create `docs/stages/INDEX.md` grouping stages by phase/theme.
2. Add phase markers: Foundation (S06-S10), Mesh (S26-S32), Domains (S35-S73), Platform (S95-S106), Analytical (S131-S162), Wave B (S163-S191), Codegen (S192-S204), Stabilization (S205-S208).

**Value:** MEDIUM — makes the audit trail navigable.

---

## 4. Proposed Archive Structure

```
docs/
├── architecture/          (active docs only — target: ~120-150 files, down from 440)
│   ├── domains/           (domain-specific docs, grouped)
│   │   ├── signal/
│   │   ├── decision/
│   │   ├── risk/
│   │   ├── execution/
│   │   └── strategy/
│   └── (remaining active docs at root level)
├── archive/               (new directory)
│   ├── gates/             (all gate/readiness review docs)
│   ├── wave-b/            (superseded wave-b docs)
│   ├── families/          (pre-consolidation family lifecycle docs)
│   ├── next-wave/         (pre-consolidation next-wave docs)
│   ├── gains-tradeoffs/   (pre-consolidation retrospectives)
│   └── deferred-work/     (pre-consolidation refactor tracking docs)
├── stages/                (untouched — add INDEX.md only)
│   └── INDEX.md           (new — phase-grouped index)
└── (other existing dirs)
```

---

## 5. Execution Order

| Phase | Action | Files Affected | Risk |
|-------|--------|----------------|------|
| 1 | Create `docs/archive/` directory structure | 0 (structure only) | None |
| 2 | Archive explicitly superseded docs (Section 3.9) | ~7 files | Very low |
| 3 | Archive gate/readiness docs (Section 3.8) | ~18 files | Low |
| 4 | Consolidate deferred-work docs (Section 3.7) | ~13 files → 1 | Medium |
| 5 | Consolidate next-wave docs (Section 3.1) | 15 files → 1 | Medium |
| 6 | Consolidate gains/tradeoffs docs (Section 3.6) | 17 files → 1 | Medium |
| 7 | Consolidate family lifecycle docs (Section 3.2) | 34 files → 5 | Medium |
| 8 | Consolidate Wave B docs (Section 3.3) | 24 files → 6 | Medium |
| 9 | Consolidate analytical docs (Section 3.4) | 28 files → 10 | Medium |
| 10 | Consolidate codegen docs (Section 3.5) | 16 files → 8 | Low |
| 11 | Reorganize domain docs (Section 3.10) | ~40 files moved | Low |
| 12 | Create stage report index (Section 3.11) | 1 new file | None |

**Estimated impact:** From ~440 architecture docs to ~120-150 active docs + organized archive.

---

## 6. Safety Guardrails

1. **Git tag before cleanup:** Tag the repository before any documentation changes.
2. **Archive before delete:** Every consolidated doc's originals go to `docs/archive/` first.
3. **One cluster at a time:** Do not batch multiple consolidation clusters in a single commit.
4. **Review consolidated content:** Each consolidation must verify no unique content is lost.
5. **No content invention:** Consolidated docs contain only content from their sources.
6. **Maintain git blame:** Use `git mv` for moves to preserve history.
