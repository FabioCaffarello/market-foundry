# Stage S171 — Mandatory Hardening Tranche Definition Report

> **Objective:** Define the mandatory hardening iteration that must precede Family 03, transforming three ad-hoc commitments into a formal, responsibility-oriented tranche with clear scope, sequencing, rationale, and gate criteria.

## Executive Summary

S171 formalizes the hardening mandate established in S167 and reconfirmed across S169–S170. Three friction items — struct-based DI, smoke test extraction, and helper renaming — are decomposed into a sequenced tranche with architectural rationale, success criteria, and explicit blockers for Family 03.

The tranche is narrow by design: three items, three phases, zero functional changes, zero schema work. Each item addresses a documented scaling constraint at a composition, operability, or semantic boundary. The goal is to ensure the Wave B expansion pattern remains mechanically reliable at 4+ families.

## Deliverables Produced

| # | Document | Purpose |
|---|---|---|
| 1 | `docs/architecture/mandatory-hardening-tranche-before-family-03.md` | Binding scope reference: composition, sequencing, boundaries, validation |
| 2 | `docs/architecture/struct-di-smoke-extraction-helper-renaming-rationale.md` | Architectural rationale for each item: why structural, not cosmetic |
| 3 | `docs/architecture/family-03-blockers-and-hardening-success-criteria.md` | Explicit gate conditions, verification commands, stop conditions |
| 4 | `docs/stages/stage-s171-mandatory-hardening-tranche-definition-report.md` | This report |

## Tranche Summary

### Items

| ID | Item | Responsibility | Severity | Phase |
|---|---|---|---|---|
| H-3 | Rename `parseEvidenceKeyParams` → `parseAnalyticalKeyParams` | Semantic clarity, boundary hygiene | Low-Medium | 1 |
| H-1 | Refactor `NewAnalyticalWebHandler` to struct-based DI | Composition, extensibility | Medium | 2 |
| H-2 | Extract `validate_analytical_family()` in smoke script | Operability, repeatability | Medium | 3 |

### Sequencing Rationale

1. **H-3 first** — smallest blast radius, establishes correct naming before structural changes.
2. **H-1 second** — handler composition change; benefits from clean naming already in place.
3. **H-2 last** — smoke refactoring after the code it tests has stabilized.

### Affected Scope

| Item | Files |
|---|---|
| H-3 | `handlers/analytical.go`, `handlers/analytical_test.go` |
| H-1 | `handlers/analytical.go`, `handlers/analytical_test.go`, `routes/analytical.go`, `cmd/gateway/compose.go` |
| H-2 | `scripts/smoke-analytical-e2e.sh` |

Total: 4 Go files + 1 shell script. No schema, no writer, no reader adapters, no migrations.

## Pain Points Addressed

| Friction | Origin | Item |
|---|---|---|
| Constructor with 4 positional args approaching fragility | S167 D-2, S170 PF-1 | H-1 |
| Smoke script 614 lines, +80/family growth | S167 D-3, S170 PF-3 | H-2 |
| `parseEvidenceKeyParams` name misleading for 3 families | S167 D-1, S170 PF-2 | H-3 |

## Gains

| Gain | Type |
|---|---|
| Handler constructor scales to N families without signature changes | Structural |
| Smoke test family addition cost drops from ~80 lines to ~5 lines | Operational |
| Helper naming matches actual scope across all families | Semantic |
| Family 03 expansion focuses purely on new artifacts, not debt | Process |

## Trade-offs Accepted

| Trade-off | Rationale |
|---|---|
| H-4 (consumer/inserter naming) deferred | Severity too low to justify inclusion; blast radius unclear |
| CI smoke integration (PF-5) not included | Separate concern with different stakeholders and scope |
| No codegen evaluation | D-4 explicitly deferred to family 4 gate |
| Hardening as separate iteration (not combined with Family 03) | Ensures hardening gets full attention; prevents expansion from crowding out structural fixes |

## Open Debts After S171

| Debt | Status | Trigger |
|---|---|---|
| D-4: Codegen evaluation for reader/handler/test | Deferred | Family 4 gate |
| D-5: No backoff jitter in writer retry | Open | No committed trigger |
| D-6: No NATS consumer lag visibility | Open | No committed trigger |
| D-9: No pagination beyond 500 rows | Open | No committed trigger |
| PF-5: No CI integration for analytical smoke | Carried | Revisit after Family 03 |
| H-4: Consumer/inserter naming review | Deferred | Not committed |

## Limits and Non-Goals

- **No new family.** Family 03 does not begin until the hardening gate passes.
- **No functional changes.** All three items are behavior-preserving refactors.
- **No schema or migration work.** ClickHouse is untouched.
- **No writer changes.** Write path is stable and unrelated.
- **No broad redesign.** Each item has a defined, narrow scope and known file set.
- **No new dependencies.** No libraries, no new modules, no new binaries.

## Family 03 Gate Status

| Blocker | Status |
|---|---|
| H-1: Struct-based DI | **Defined, not yet implemented** |
| H-2: Smoke extraction | **Defined, not yet implemented** |
| H-3: Helper renaming | **Defined, not yet implemented** |

**Family 03 remains blocked** until all three items pass their verification criteria as documented in `family-03-blockers-and-hardening-success-criteria.md`.

## Preparation for S172

S172 should be the **implementation** of this hardening tranche, following the defined phasing:

1. Phase 1: Execute H-3 (rename helper). Verify. Commit.
2. Phase 2: Execute H-1 (struct DI refactor). Verify. Commit.
3. Phase 3: Execute H-2 (smoke extraction). Verify. Commit.
4. Run composite gate: all tests, smoke behavioral parity, CI green.
5. Produce hardening implementation report.

S172 should be scoped as **implementation only** — the architectural decisions are frozen in S171. No design deliberation should occur during implementation.

## Conclusion

The mandatory hardening tranche is now formally defined. Three items with clear rationale, sequencing, and gate criteria replace three ad-hoc commitments. The tranche is narrow, testable, and behavior-preserving. It directly reduces the marginal cost of Family 03 and every subsequent family expansion.

The pattern has earned the right to scale. This hardening ensures it scales cleanly.
