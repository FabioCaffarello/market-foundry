# S364 — Derive Integration Wave Charter and Scope Freeze Report

> Stage type: Charter / scope freeze.
> Wave: Derive Integration Wave (Phase 37).
> Predecessor: S363 (Strategy/Signal Integration Evidence Gate — ALL OBJECTIVES MET).
> Date: 2026-03-22.

---

## 1. Executive Summary

S364 formally opens the Derive Integration Wave by chartering its scope,
freezing its boundaries, defining governing questions, and ordering the
execution stages.

The Strategy/Signal Integration Wave (S358–S363) proved the **consumer side**
of the strategy-to-execution path with full evidence: 8/8 governing questions
at HIGH confidence, 31 tests, 11/11 invariants, zero regressions. Its primary
deferred gap (DG-11) was derive-side strategy event production — the
**producer side** of the same path.

This wave closes that gap. The derive binary already has strategy resolvers
and a publisher. What it lacks is **contract compliance verification** against
S359 and **end-to-end proof** that derive-produced events flow correctly
through store, gateway, and execute.

The wave is scoped to **one strategy family** (mean_reversion_entry), **one
signal family** (RSI), and **paper execution**. No new families, no
multi-venue, no OMS, no derive runtime redesign.

---

## 2. What Was Done

### 2.1 State Analysis

Analyzed the consolidated post-S363 state:

- **S363 verdict**: WAVE CLOSED — ALL OBJECTIVES MET
- **Cumulative wave progress**: 4 waves closed (Venue Activation, Production Readiness, Operational Foundation, Strategy/Signal Integration)
- **Key deferred gap**: DG-11 (no derive-side strategy event production)
- **Primary recommendation from S363**: Derive Integration Wave (NW-1)
- **Existing derive infrastructure**: resolvers (3 families), publisher actor, NATS adapter, store projection, gateway query — all implemented but unverified against S359 contract

### 2.2 Charter Produced

Created [`derive-integration-wave-charter-and-scope-freeze.md`](../architecture/derive-integration-wave-charter-and-scope-freeze.md):

- Strategic context linking S363 consumer proof to producer gap
- Wave identity (Phase 37, S364–S369)
- 5 ordered blocks (DI-1 through DI-5)
- 7 binding constraints
- Prerequisites verification (all met)
- Risk assessment (4 risks, all mitigated)
- 7 success criteria

### 2.3 Capabilities, Questions, and Non-Goals Produced

Created [`derive-integration-capabilities-questions-and-non-goals.md`](../architecture/derive-integration-capabilities-questions-and-non-goals.md):

- 6 capabilities under assessment (DC-1 through DC-6)
- 8 governing questions (DIQ-1 through DIQ-8)
- 15 explicit non-goals (NG-1 through NG-15)
- Relationship to S359 contract (4 contract elements mapped)
- Relationship to existing derive implementation (7 components audited)
- Relationship to S363 residual gaps (4 gaps mapped)

---

## 3. Charter Summary

### 3.1 Wave Thesis

The derive binary is a correct, contract-compliant producer of
`StrategyResolvedEvent` and the full analytical-to-execution pipeline works
as a connected system.

### 3.2 Ordered Block Plan

| Block | Stage | Objective | Scope |
|---|---|---|---|
| DI-1 | S365 | Producer spec and derive ownership | Audit derive output against S359 contract; document compliance matrix |
| DI-2 | S366 | Canonical derive producer wiring | Fix contract mismatches; unit tests for resolver + publisher |
| DI-3 | S367 | Store/gateway/read-path verification | Integration tests for materialization + HTTP query |
| DI-4 | S368 | Analytical-to-execution end-to-end proof | Full pipeline test: signal → derive → strategy → execute → fill |
| DI-5 | S369 | Evidence gate final | Formal wave closure; capability audit; regression check |

### 3.3 Governing Questions

| # | Question | Block |
|---|----------|-------|
| DIQ-1 | Does derive resolver satisfy all 11 S359 invariants? | DI-1 |
| DIQ-2 | Is there a field-level compliance mapping? | DI-1 |
| DIQ-3 | Do unit tests prove each invariant on the producer side? | DI-2 |
| DIQ-4 | Does the publisher produce correct NATS messages? | DI-2 |
| DIQ-5 | Does store correctly materialize derive-produced events? | DI-3 |
| DIQ-6 | Does gateway HTTP return derive-produced state? | DI-3 |
| DIQ-7 | Does a full end-to-end test prove the connected pipeline? | DI-4 |
| DIQ-8 | Does correlation chain propagate from signal through execution? | DI-4 |

### 3.4 Non-Goals (Summary)

| Category | What's excluded |
|---|---|
| **Breadth** | Batch family onboarding (NG-1), multiple signal families (NG-2), new domain types (NG-15) |
| **Execution** | Multi-venue (NG-3), mainnet (NG-6), OMS (NG-4), portfolio risk (NG-5) |
| **Infrastructure** | Docker Compose (NG-9), dashboards (NG-7), alerting (NG-10), logs (NG-11) |
| **Architecture** | Derive runtime redesign (NG-8), risk domain (NG-14) |
| **Data** | Parameter optimization (NG-12), historical replay (NG-13) |

---

## 4. Capability-Alvo

The capability this wave must prove:

> **Derive is the canonical producer of `StrategyResolvedEvent` for the
> mean_reversion_entry family, and its output drives the full
> analytical-to-execution pipeline through store, gateway, and execute.**

This decomposes into 6 measurable capabilities (DC-1 through DC-6) defined
in the companion document.

---

## 5. Stage Ordering Rationale

The blocks are ordered by dependency:

1. **DI-1 (audit)** must come first because code changes in DI-2 depend on
   knowing what's wrong.
2. **DI-2 (producer wiring)** must come before DI-3 because store/gateway
   verification needs correctly-shaped events.
3. **DI-3 (store/gateway)** must come before DI-4 because the end-to-end
   proof needs the materialization path verified independently.
4. **DI-4 (E2E proof)** is the capstone that composes all prior blocks.
5. **DI-5 (gate)** evaluates everything.

No parallelism. Each block produces artifacts consumed by the next.

---

## 6. Preparation Recommended for S365

S365 (DI-1: Producer Spec and Derive Ownership) should:

1. Read `strategy_resolver_actor.go` (MeanReversionEntryResolverActor) and
   map each output field against S359 INV-1 through INV-11.
2. Read `strategy_publisher_actor.go` and `natsstrategy/publisher.go` to
   verify subject format matches execute consumer's subscription filter.
3. Read `natsstrategy/registry.go` to verify the mean_reversion_entry spec
   matches what the execute consumer expects.
4. Produce a compliance matrix (invariant × derive output field × status).
5. Document any mismatches with severity and fix plan.

No code changes expected in S365 unless a mismatch is trivially fixable.

---

## 7. Promoted Documents

| Document | Location | Purpose |
|---|---|---|
| Derive Integration Wave Charter | [`docs/architecture/derive-integration-wave-charter-and-scope-freeze.md`](../architecture/derive-integration-wave-charter-and-scope-freeze.md) | Wave scope, blocks, constraints |
| Derive Integration Capabilities, Questions, and Non-Goals | [`docs/architecture/derive-integration-capabilities-questions-and-non-goals.md`](../architecture/derive-integration-capabilities-questions-and-non-goals.md) | Capabilities, governing questions, non-goals |

---

## 8. Verification

| Check | Result |
|---|---|
| Wave formally opened | YES — charter frozen |
| Scope frozen | YES — 5 blocks, 7 constraints, 15 non-goals |
| Capability-alvo defined | YES — 6 capabilities (DC-1 through DC-6) |
| Governing questions defined | YES — 8 questions (DIQ-1 through DIQ-8) |
| Non-goals explicit | YES — 15 non-goals documented |
| Next stages ordered | YES — S365 through S369 with dependency rationale |
| S365 preparation documented | YES — 5-step preparation plan |

---

## References

- [Derive Integration Wave Charter](../architecture/derive-integration-wave-charter-and-scope-freeze.md)
- [Derive Integration Capabilities, Questions, and Non-Goals](../architecture/derive-integration-capabilities-questions-and-non-goals.md)
- [S363 — Strategy/Signal Integration Evidence Gate](stage-s363-strategy-signal-integration-evidence-gate-report.md)
- [Strategy/Signal Integration Evidence Matrix](../architecture/strategy-signal-integration-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Source Selection and Canonical Integration Contract (S359)](../architecture/source-selection-and-canonical-integration-contract.md)
