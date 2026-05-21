# S327 — Production Wiring Tranche Charter Report

> **Stage:** S327
> **Date:** 2026-03-21
> **Type:** Charter and scope freeze
> **Phase:** 31b (Production Wiring Tranche)
> **Predecessor:** S326 (Venue Progression Evidence Gate — CLOSED)
> **Successor:** S328 (RetrySubmitter + Observability Hook Wiring)

---

## Executive Summary

S327 opens a formal, short, disciplined tranche to compose into the production
actor pipeline the capabilities that were implemented and tested in isolation
during the venue progression (S316–S325). The tranche scope is frozen at four
items: RetrySubmitter wiring, Post200Reconciler wiring, observability hook
wiring, and integration verification. Non-goals are explicitly enumerated to
prevent scope inflation into OMS, multi-venue, dashboards, mainnet, supervisor
redesign, or new domain capabilities.

**Tranche verdict: OPEN. Scope: FROZEN.**

---

## Stage Objective

Transform the S326 recommendation into a formal tranche with:

1. A charter that authorizes only mechanical composition of tested code.
2. Frozen scope with explicit item IDs, stage assignments, and exit criteria.
3. Non-goals that prevent the tranche from becoming a design or capability wave.
4. A gate plan (S331) with objective, verifiable closure criteria.

---

## Deliverables

### D1: Tranche Charter and Scope Freeze

**Document:** [`../architecture/production-wiring-tranche-charter-and-scope-freeze.md`](../architecture/production-wiring-tranche-charter-and-scope-freeze.md)

Contents:
- Strategic context linking S326 closure to this tranche.
- Tranche mandate: compose tested decorators into the production path.
- Scope freeze: 4 items (PWT-1 through PWT-4), explicitly bounded.
- Non-goals: OMS, multi-venue, dashboards, mainnet, supervisor redesign, new capabilities.
- Scope inflation guard: no new interfaces, no parameter changes, escalation protocol.
- Stage plan: S328 (retry + hooks) → S329 (reconciler) → S330 (verification) → S331 (gate).
- Invariant preservation requirement (all 9).
- Risk assessment (all risks Low or Very Low).
- Success criteria for tranche closure.

### D2: Exit Criteria and Non-Goals

**Document:** [`../architecture/production-wiring-items-exit-criteria-and-non-goals.md`](../architecture/production-wiring-items-exit-criteria-and-non-goals.md)

Contents:
- Per-item exit criteria for PWT-1 through PWT-4.
- Target composition code for each item.
- Deferred gap closure mapping (R-S323-3, R-S324-1).
- Tranche gate criteria matrix (7 criteria).
- Non-goals organized by category: capability, design, process.
- Escalation protocol for discovered gaps.

---

## Tranche Structure

### Items

| Item | Description | Stage | Closes Gap |
|------|-------------|-------|------------|
| PWT-1 | RetrySubmitter around adapter in bootstrap | S328 | S319 production path |
| PWT-2 | Post200Reconciler around RetrySubmitter in bootstrap | S329 | S322 production path |
| PWT-3 | WithHaltChecker + WithLogger + WithTracker in bootstrap | S328 | R-S323-3, R-S324-1 |
| PWT-4 | Integration test of composed pipeline | S330 | Composition proof |

### Stage Sequence

| Stage | Role | Items | Key Deliverable |
|-------|------|-------|-----------------|
| S327 | Charter | — | This report + architecture docs |
| S328 | Implementation | PWT-1, PWT-3 | RetrySubmitter + hooks wired in `execute_supervisor.go` |
| S329 | Implementation | PWT-2 | Post200Reconciler wired in `execute_supervisor.go` |
| S330 | Verification | PWT-4 | Integration test exercising full composed chain |
| S331 | Gate | — | Tranche closure gate with exit criteria verification |

### Composition Target

```
Current (S326 baseline):
  VenueAdapterActor(config.Venue = rawAdapter)
    └── SafetyGate → rawAdapter.SubmitOrder()

After tranche (S331 target):
  VenueAdapterActor(config.Venue = post200Reconciler)
    └── SafetyGate → post200Reconciler.SubmitOrder()
          └── retrySubmitter.SubmitOrder()
                .WithHaltChecker(controlStore)
                .WithLogger(logger)
                .WithTracker(tracker)
                └── rawAdapter.SubmitOrder()
```

---

## Non-Goals Summary

| Category | Items excluded |
|----------|---------------|
| Capability | OMS, multi-venue, mainnet, new domain families, dashboards |
| Design | Supervisor redesign, new interfaces, new config knobs, retry policy tuning |
| Process | CI/CD changes, test refactoring, new documentation categories |

Full enumeration in the exit criteria document.

---

## Invariant Preservation

All 9 invariants tracked since S308 must remain preserved:

| ID | Invariant | Risk from wiring |
|----|-----------|-----------------|
| EC-1 | Deterministic client order ID | None — ID generation unchanged |
| EC-3 | Per-request deadline | None — deadline enforcement unchanged |
| F-1 | No bare errors / Problem type | None — error wrapping unchanged |
| F-4 | Credential redaction | None — adapter internals unchanged |
| RF-1 | Retryable flag accuracy | None — classification unchanged |
| PGR-08 | Intent immutability | None — intent handling unchanged |
| INV-REC-1 | No duplicate execution | None — reconciler uses QueryOrder (GET) |
| INV-RC-1 | Deadline independence | None — deadlines are per-layer |
| INV-OBS-1 | Zero noise on success | None — hooks are nil-safe and tested |

---

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Interface mismatch at composition | Very Low | All interfaces satisfied in unit tests |
| Test regression | Very Low | Full suite run at each stage |
| Scope creep | Low | Explicit inflation guard + escalation protocol |
| Discovery of blocking design gap | Low | Pause tranche, document, resolve separately |

---

## Predecessor Chain

```
S306 (Venue Readiness Charter)
  → S312 (Adapter Hardening Tranche Charter)
    → S315 (Foundational Tranche Gate — PASS)
      → S321 (Venue Closure Tranche Charter)
        → S326 (Venue Progression Evidence Gate — CLOSED)
          → S327 (Production Wiring Tranche Charter) ← this stage
            → S328 (RetrySubmitter + Hook Wiring)
              → S329 (Post200Reconciler Wiring)
                → S330 (Integration Verification)
                  → S331 (Production Wiring Gate)
```

---

## Preparation for S328

S328 should begin with:

1. **Read** `execute_supervisor.go` to identify the exact bootstrap location where the adapter is constructed and passed to `VenueAdapterActor`.
2. **Verify** that `RetrySubmitter`, `WithHaltChecker`, `WithLogger`, and `WithTracker` interfaces match what the bootstrap has available (control store, logger, tracker).
3. **Compose** the `RetrySubmitter` wrapping the adapter with all three hooks.
4. **Run** the full test suite to verify zero regressions.
5. **Document** any interface friction discovered during wiring.

Expected scope: 1 file changed (`execute_supervisor.go`), ~10 lines added, 0 interfaces modified.

---

## Verdict

**S327 COMPLETE. Production Wiring Tranche OPEN. Scope FROZEN.**

The tranche is authorized to proceed to S328. Four items are chartered. Exit
criteria are objective. Non-goals are explicit. The inflation guard is active.
