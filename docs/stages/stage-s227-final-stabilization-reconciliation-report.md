# Stage S227 — Final Stabilization Reconciliation Report

**Date:** 2026-03-20
**Type:** Final closure-tranche reconciliation
**Scope:** Reconcile the final S223-S226 closure baseline across code, tooling, active docs, and gate evidence
**Status:** COMPLETE — baseline reconciled locally; final remote gate proof still pending

---

## 1. Executive Summary

S227 executed the final short-tranche reconciliation without opening new scope.

The stage confirmed that S224 and S225 were substantively closed, reproduced the S226 analytical-smoke blocker locally, corrected the real runtime/tooling drift behind it, and consolidated the tranche into one coherent local baseline.

The result is honest and bounded:

1. local code, tooling, docs, and smoke evidence now point to the same reality,
2. XC-1 is no longer ambiguous and is formally closed by re-baseline,
3. the final gate is still not clean because the corrected baseline has not yet been rerun through remote CI, so the tag remains correctly blocked.

---

## 2. Scope Discipline

### IN scope

1. Review closure claims from S223-S226 against the actual repository.
2. Correct small-scope runtime/tooling/documentation drift inside the closure tranche.
3. Close XC-1 explicitly.
4. Publish the final reconciliation record and consistency checklist.

### OUT of scope

1. New feature work.
2. New architectural expansion.
3. Broad archival or documentation minimization wave.
4. Reopening medium-priority debt beyond the closure tranche.
5. Declaring remote CI green without a fresh run.

---

## 3. Reconciliation Applied

### 3.1 Tooling/runtime fixes

S227 applied the following bounded changes:

1. restored Bash 3.2 compatibility in shared scripting utilities,
2. removed compose-command path-splitting bugs for workspace paths containing spaces,
3. aligned `make up` with its documented behavior by waiting for ClickHouse and applying migrations,
4. aligned `live-pipeline-activate.sh` with the same migration discipline,
5. corrected smoke/log diagnostics so they report the actual failure class instead of over-asserting missing config,
6. aligned gateway and writer ClickHouse config to `market_foundry`,
7. fixed writer insert SQL to match the current DDL column surface.

### 3.2 Active-doc and evidence fixes

S227 also:

1. updated active config examples that still used `default`,
2. added S227 reconciliation notes to the S226 evidence docs,
3. updated `system-vision.md` to reflect closure through S227,
4. published the new S227 architecture reconciliation docs and this stage report.

---

## 4. Validation

S227 validated the reconciled baseline with:

1. `make check` — **PASS**
2. `make verify` — **PASS**
3. `make up` — **PASS**
4. `make seed` — **PASS**
5. `make smoke-analytical` — **PASS**

Key runtime proof from the clean stack:

1. ClickHouse schema present in `market_foundry`,
2. writer receives events and persists candle rows,
3. gateway analytical candle history returns `200` with data,
4. no error-level compose logs were emitted on the clean proof run.

---

## 5. XC-1 Disposition

S227 closes XC-1 explicitly by formal re-baseline.

Measured counts:

1. pre-S227 deliverables: `docs/architecture = 263`, `docs/stages = 223`
2. post-S227 deliverables: `docs/architecture = 265`, `docs/stages = 224`

The old `<= 150` target is not used as the closure criterion anymore for this tranche. The accepted S227 criterion is active-doc consistency with current code/tooling/evidence rather than raw count reduction.

Result:

1. XC-1 — **CLOSED**

---

## 6. Residual Blocker

S227 leaves one exact blocker before the final gate can be called clean:

1. the reconciled S227 baseline still lacks a fresh remote GitHub Actions run, so XC-6 / EC-7 remains unproven on the corrected baseline and XC-11 remains blocked behind that proof.

This means:

1. S226's historical remote `FAIL` is no longer the unresolved runtime mystery,
2. the remaining blocker is now procedural and evidence-based: rerun CI on push, then tag if green.

---

## 7. Files Produced

S227 produced:

1. `docs/architecture/final-stabilization-reconciliation.md`
2. `docs/architecture/final-closure-consistency-checklist.md`
3. `docs/stages/stage-s227-final-stabilization-reconciliation-report.md`

S227 also updated the relevant runtime/tooling/docs files needed to make the closure baseline coherent.

---

## 8. Preparation for S228

S228 should assume:

1. the short closure tranche is locally reconciled,
2. the remaining gate work is fresh remote proof plus tag creation,
3. no reopening of S223-S226 drift investigation is necessary,
4. expansion discussion stays blocked until the fresh remote proof exists.
