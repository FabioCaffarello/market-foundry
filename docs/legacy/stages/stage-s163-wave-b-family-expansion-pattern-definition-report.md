# Stage S163 — Wave B Family Expansion Pattern Definition Report

**Stage:** S163
**Status:** COMPLETE
**Predecessor:** S162 (Pre-Wave-B Analytical Readiness Gate)
**Successor:** S164 (First Wave B Family Expansion — Signals)

---

## Executive Summary

S163 freezes the canonical expansion pattern for Wave B before any new family is implemented. The deliverables define what every family must produce, how each artifact is structured, what criteria must be met, and what remains explicitly out of scope. The pattern is derived directly from the candle family precedent established during Wave A and hardened through S150–S162.

No code was written. No families were added. This stage is purely definitional.

---

## Deliverables Produced

| # | Document | Purpose |
|---|---|---|
| 1 | `docs/architecture/wave-b-family-expansion-pattern.md` | Canonical step-by-step template for adding one analytical family |
| 2 | `docs/architecture/wave-b-family-checklist-schema-writer-reader-gateway-tests-runbook.md` | Exhaustive checklist covering all artifacts, tests, and verification gates |
| 3 | `docs/architecture/wave-b-iteration-constraints-and-non-goals.md` | Formal constraints inherited from S162, operational guardrails, and explicit non-goals |
| 4 | This report | Stage completion record |

---

## Pattern Summary

### Expansion Unit (9 Artifacts)

Every Wave B family produces exactly nine artifacts:

1. **Migration DDL** — ClickHouse table with standardized metadata columns, domain columns matching Go structs, MergeTree engine, 90-day TTL.
2. **Writer mapper** — `map<Family>Row` function producing `[]any` in exact DDL column order.
3. **Pipeline entry** — `WriterPipeline` catalog entry binding NATS subject → table → mapper.
4. **Reader adapter** — `Query<Family>History` with parameterized queries, timing, and structured logging.
5. **Application contracts** — Query/reply structs and use case function.
6. **HTTP handler** — Parameter validation, Server-Timing headers, structured JSON response.
7. **HTTP route** — `GET /analytical/<domain>/<family>`, registered only when ClickHouse is configured.
8. **Smoke test section** — Write-path and read-path verification in `smoke-analytical-e2e.sh`.
9. **Family documentation** — Limits, deviations, and operational notes.

### Iteration Flow

```
Schema → Writer → Reader → Gateway → Tests → Smoke → Gate Review → Next Iteration
```

Strict left-to-right dependency. No step begins before its predecessor is verified.

### Schema Coherence Rule

DDL, writer mapper, and reader adapter must be column-aligned. Mismatches are blocking defects. Alignment is verified by unit tests asserting column counts and order.

### Observability Parity

Every family inherits the same observability coverage as candles: inserter counters (6 metrics), reader wall-clock timing, handler Server-Timing headers, and presence in `/statusz` and `/diagz`.

---

## Responsibilities by Track

| Track | Owner | Artifacts | Verification |
|---|---|---|---|
| **Schema** | Migration author | DDL file, reverse DDL | `cmd/migrate` applies cleanly, `_migrations` records checksum |
| **Writer** | Writer author | Mapper function, pipeline entry | Mapper unit tests, `/statusz` shows pipeline |
| **Reader** | Reader author | Adapter method, query builder | Query builder tests, adapter struct mapping tests |
| **Gateway** | Gateway author | Handler, route registration | Handler unit tests (200, 400 cases), route isolation verified |
| **Contracts** | Application author | Query/reply structs, use case | Use case unit test |
| **Integration** | Integration author | Smoke test extension | Full E2E pass, no regressions |
| **Runbook** | Iteration lead | Operational notes, limits | Gate review confirms documentation completeness |

In practice, the same person may fill all roles for a single-family iteration. The separation of responsibilities ensures nothing is skipped, not that different people must do each part.

---

## Constraints Enforced

### From S162 (Non-Negotiable)

| ID | Constraint |
|---|---|
| C-1 | One family per iteration |
| C-2 | Full hardening parity with candle baseline |
| C-3 | CI integration before second family |
| C-4 | No external infrastructure |

### Operational (Derived from Wave A)

| ID | Constraint |
|---|---|
| C-5 | Schema coherence is blocking |
| C-6 | Optionality invariant (R-01 through R-10) preserved |
| C-7 | No horizontal redesign |
| C-8 | No cross-family features |
| C-9 | Additive only (never modify existing artifacts) |

---

## Checklist Coverage

The checklist document covers seven verification areas:

1. **Entry conditions** — Prerequisites before starting a family.
2. **Schema** — 15 items covering DDL conventions, types, engine, and migration verification.
3. **Writer** — 13 items covering mapper, pipeline entry, and writer tests.
4. **Reader** — 10 items covering adapter, query builder, and reader tests.
5. **Gateway** — 10 items covering handler, route, and gateway tests.
6. **Integration/Smoke** — 7 items covering E2E verification and regression.
7. **Gate sign-off** — 4 items covering completeness, deviation documentation, and iteration unblocking.

Total: ~80 checklist items per family. Each must be satisfied before the family ships.

---

## Expansion Validity

The constraints document provides a decision tree for distinguishing valid from premature expansion:

- **Valid:** One family, all nine artifacts, NATS subject exists, event structure stable, tests complete before merge.
- **Premature:** Multiple families, partial artifacts, unstable event structures, deferred tests, or "while we're here" scope additions.

---

## Non-Goals (Explicit)

| ID | Non-Goal |
|---|---|
| NG-1 | No generic expansion framework or plugin system |
| NG-2 | No performance optimization |
| NG-3 | No per-family retention policies |
| NG-4 | No schema evolution tooling |
| NG-5 | No multi-instance or sharding |
| NG-6 | No real-time streaming queries |
| NG-7 | No user-facing API documentation |
| NG-8 | No historical backfill |

---

## Open Debts Entering Wave B

Carried forward from S162 gate review:

| Debt | Priority | Impact on Wave B |
|---|---|---|
| CI integration of smoke-analytical | Medium | Must resolve before second family (C-3) |
| Backoff jitter in inserter retry | Low | Does not block expansion |
| Consumer lag visibility | Medium | Does not block; document as known limit |
| Per-family tuning knobs | Low | All families use same defaults |
| Reader connection pooling | Low | Single instance sufficient for Wave B |

None of these debts block the first family iteration.

---

## S164 Preparation

The next stage should be the first Wave B family expansion. Recommended candidate: **signals**.

Rationale:
- Migration `002_create_signals.sql` already exists and is applied.
- Writer mapper `mapSignalRow` already exists.
- Writer pipeline entry for signals already exists.
- The write path is active and consuming signal events.
- Only the read path (reader adapter, contracts, use case, handler, route, smoke) needs to be built.
- This makes signals the lowest-risk candidate for validating the expansion pattern.

S164 scope:
1. Implement `QuerySignalHistory` reader adapter.
2. Add signal query/reply contracts and use case.
3. Add signal HTTP handler and route.
4. Extend smoke-analytical-e2e.sh with signal section.
5. Complete full checklist and gate review.
6. Begin CI integration planning for smoke-analytical (required before S165).

---

## Acceptance Criteria Verification

| Criterion | Status |
|---|---|
| Clear, repeatable pattern for family expansion | SATISFIED — 9-artifact template with step-by-step flow |
| Responsibilities explicitly separated | SATISFIED — 7 tracks with defined artifacts and verification |
| Minimum hardening criteria clear | SATISFIED — ~80 checklist items, observability parity rule, schema coherence rule |
| Risk of ad hoc expansion reduced | SATISFIED — Decision tree, validity test, 9 constraints, 8 non-goals |
| Base ready for first Wave B iteration | SATISFIED — Pattern frozen, signals identified as candidate, no blocking debts |

**Stage S163: COMPLETE.**
