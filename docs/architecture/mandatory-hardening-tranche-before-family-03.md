# Mandatory Hardening Tranche Before Family 03

> Formal definition of the hardening iteration that must complete before the third analytical family is opened.
> This document is the binding scope reference for the hardening tranche.

## Strategic Context

After two successful family expansions (Candles → Signals → Decisions), the Wave B pattern has proven mechanically sound. However, three friction points have accumulated to threshold levels documented across S167, S169, and S170. The Wave B Family Expansion Pattern v2 explicitly commits these as **hard requirements at family 3**.

The next iteration is **not** a third family. It is a mandatory hardening pass that resolves accumulated friction before the pattern scales further.

## Why Hardening, Not Expansion

| Signal | Evidence |
|---|---|
| Constructor fragility at threshold | `NewAnalyticalWebHandler` takes 4 positional args; a 5th would be brittle (S167 D-2, S170 PF-1) |
| Smoke script at maintainability ceiling | 614 lines, growing ~80 lines per family; a 4th phase would exceed readable structure (S167 D-3, S170 PF-3) |
| Naming residue misleading at scale | `parseEvidenceKeyParams()` is consumed by 3 families; "Evidence" prefix is factually wrong for Signals and Decisions (S167 D-1, S170 PF-2) |

These are not aesthetic preferences. Each item addresses a **structural scaling constraint** that, if left unresolved, would make the next family expansion fragile, confusing, or operationally risky.

## Tranche Composition

The tranche consists of exactly three hardening items, ordered by dependency:

### Sequencing

```
Phase 1: H-3 — Rename parseEvidenceKeyParams → parseAnalyticalKeyParams
  └─ No dependencies. Pure rename. All tests update mechanically.

Phase 2: H-1 — Refactor NewAnalyticalWebHandler to struct-based DI
  └─ Depends on: nothing structurally, but benefits from clean naming (H-3).
  └─ Touches: handler constructor, route wiring, compose.go, handler tests.

Phase 3: H-2 — Extract validate_analytical_family() in smoke script
  └─ Depends on: H-1 complete (handler shape stable before smoke refactor).
  └─ Touches: scripts/smoke-analytical-e2e.sh only.
```

### Rationale for Ordering

1. **H-3 first**: smallest blast radius, zero behavioral change, establishes correct naming before structural changes arrive.
2. **H-1 second**: structural change to handler composition; doing this after naming ensures the new struct uses the correct vocabulary.
3. **H-2 last**: smoke refactoring should happen after the code it tests has stabilized; extracting a reusable function before handler shape is final risks rework.

## Tranche Boundaries

### In Scope

- The three items listed above (H-1, H-2, H-3).
- Test updates required by each item.
- Documentation updates to reflect new signatures/names.

### Out of Scope (Non-Goals)

- **No new family expansion.** Family 03 is not opened.
- **No new schema or migration work.** ClickHouse is untouched.
- **No writer changes.** Write path is stable and unrelated.
- **No H-4 (consumer/inserter naming review).** Deferred — severity too low, blast radius unclear.
- **No CI smoke integration (PF-5).** Important but separate concern; tracked independently.
- **No pagination work (PF-6, D-9).** Functional, not structural.
- **No codegen evaluation (D-4).** Deferred to family 4 gate.
- **No broad redesign.** Each item has a defined, narrow scope.

## Affected Files (Anticipated)

| Item | Files Modified |
|---|---|
| H-3 | `internal/interfaces/http/handlers/analytical.go`, `internal/interfaces/http/handlers/analytical_test.go` |
| H-1 | `internal/interfaces/http/handlers/analytical.go`, `internal/interfaces/http/handlers/analytical_test.go`, `internal/interfaces/http/routes/analytical.go`, `cmd/gateway/compose.go` |
| H-2 | `scripts/smoke-analytical-e2e.sh` |

## Validation Criteria

Each phase must pass before the next begins:

| Phase | Gate |
|---|---|
| H-3 complete | All tests pass. `grep -r parseEvidenceKeyParams` returns zero results. |
| H-1 complete | All tests pass. `NewAnalyticalWebHandler` accepts a single struct arg. Route wiring uses struct. Handler behavior unchanged. |
| H-2 complete | Smoke script produces identical pass/fail results. `validate_analytical_family()` function exists and is called for all 3 families. Script line count reduced or stable. |
| Tranche complete | CI green. No behavioral regressions. All three items verified. Family 03 gate unblocked. |

## Relationship to Family 03

This tranche is a **prerequisite** for Family 03, not part of it. Family 03 may only begin after:

1. All three hardening items are implemented and verified.
2. The tranche gate (this document's validation criteria) passes.
3. A readiness review confirms no new structural friction was introduced.

This separation ensures that Family 03 can focus entirely on expansion mechanics without carrying hardening debt.
