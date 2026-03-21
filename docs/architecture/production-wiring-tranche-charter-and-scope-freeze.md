# Production Wiring Tranche — Charter and Scope Freeze

> **Tranche ID:** PWT-1
> **Charter date:** 2026-03-21
> **Authorization:** S326 venue progression closure verdict
> **Scope status:** FROZEN
> **Stages covered:** S328–S331

---

## 1. Strategic Context

The venue progression (S316–S325) closed with FULL evidence on all seven
evaluation criteria. The closure tranche (S321–S325) delivered five chartered
items with zero scope inflation. The test suite stands at 186 passing, 0
failing. All 9 tracked invariants are preserved.

The single remaining gap is not one of design, capability, or evidence. It is a
**mechanical composition gap**: the capabilities that were implemented and tested
in isolation — RetrySubmitter, Post200Reconciler, and observability hooks — are
not yet composed into the production actor pipeline. The actor currently calls
the raw venue adapter directly.

This tranche exists to close that gap, and nothing else.

---

## 2. Tranche Mandate

Compose the tested execution decorators into the production pipeline path so
that the actor's submit flow benefits from:

1. Automatic retry with exponential backoff, deadline, and halt-check
2. Post-200 body-read-failure reconciliation
3. Structured observability (logs and counters) for retry events

The composition target is a single chain:

```
VenueAdapterActor
  └── SafetyGate (existing, unchanged)
        └── Post200Reconciler(retrySubmitter, queryPort)
              └── RetrySubmitter(adapter)
                    .WithHaltChecker(controlStore)
                    .WithLogger(logger)
                    .WithTracker(tracker)
                    └── BinanceFuturesTestnetAdapter
```

---

## 3. Scope Freeze

### 3.1 In-Scope Items

| Item | ID | Description | Touches |
|------|----|-------------|---------|
| RetrySubmitter wiring | PWT-1 | Compose RetrySubmitter around BinanceFuturesTestnetAdapter in bootstrap | `execute_supervisor.go` |
| Post200Reconciler wiring | PWT-2 | Compose Post200Reconciler around RetrySubmitter in bootstrap | `execute_supervisor.go` |
| Observability hook wiring | PWT-3 | Wire WithHaltChecker, WithLogger, WithTracker in bootstrap | `execute_supervisor.go` |
| Integration verification | PWT-4 | Verify composed pipeline in actor-level test with production-like wiring | `venue_adapter_actor_test.go` or equivalent |

### 3.2 Explicitly Out of Scope (Non-Goals)

| Non-Goal | Rationale |
|----------|-----------|
| OMS infrastructure | Requires dedicated charter; not a wiring task |
| Multi-venue expansion | Second venue adapter is a separate capability wave |
| Dashboards or monitoring UI | Observability hooks produce counters; dashboards are a consumer concern |
| Mainnet activation | Testnet-only scope; mainnet requires independent risk review |
| New breadth or domain capability | No new signals, strategies, families, or domain deepening |
| Redesign of execute supervisor | Wiring uses existing interfaces; supervisor shape does not change |
| Per-error-class differentiated retry policies | Deferred gap R-S320-6; requires production evidence before design |
| Retry-After header extraction | Accepted gap R-S325-2; exponential backoff sufficient for testnet |
| Persistence of ambiguous state | Accepted gap R-S322-2; requires OMS infrastructure |
| New failure classification codes | Accepted gap R-S325-1; current mapping is conservative and safe |

### 3.3 Scope Inflation Guard

Any change that does not directly serve composition of existing tested code into
the existing actor pipeline is **out of scope**. Specifically:

- No new interfaces may be introduced.
- No existing interfaces may be modified.
- No new actor types or supervisor changes.
- No new configuration knobs beyond what existing options already expose.
- No changes to retry policy parameters (these are already set and tested).

If a wiring step reveals a design gap that prevents composition, the item is
**escalated**, not solved inline. The tranche pauses and the gap is documented
for a separate stage.

---

## 4. Stage Plan

| Stage | Role | Item(s) | Description |
|-------|------|---------|-------------|
| S327 | Charter | — | This document: tranche charter, scope freeze, exit criteria (current stage) |
| S328 | Implementation | PWT-1, PWT-3 | Wire RetrySubmitter + WithHaltChecker + WithLogger + WithTracker around adapter in bootstrap |
| S329 | Implementation | PWT-2 | Wire Post200Reconciler around RetrySubmitter in bootstrap |
| S330 | Verification | PWT-4 | Integration test: composed pipeline exercised through actor with mock venue |
| S331 | Gate | — | Production wiring tranche gate: all exit criteria verified, invariants preserved |

### 4.1 Stage Dependencies

```
S327 (charter) → S328 (retry + hooks) → S329 (reconciler) → S330 (verification) → S331 (gate)
```

S328 and S329 are sequential because the reconciler wraps the retry submitter;
the inner layer must be wired first.

---

## 5. Invariants

All 9 existing invariants must remain preserved across the tranche:

| ID | Invariant | Source |
|----|-----------|--------|
| EC-1 | Deterministic client order ID | S313 |
| EC-3 | Per-request deadline | S308 |
| F-1 | No bare errors / Problem type | S308 |
| F-4 | Credential redaction | S314 |
| RF-1 | Retryable flag accuracy | S314 |
| PGR-08 | Intent immutability | S310 |
| INV-REC-1 | No duplicate execution | S322 |
| INV-RC-1 | Deadline independence | S323 |
| INV-OBS-1 | Zero noise on success | S324 |

No new invariants are introduced by this tranche. Wiring is composition of
tested code; it does not alter behavioral contracts.

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Interface mismatch at composition boundary | Very Low | Low | All interfaces already satisfied in unit tests |
| Regression in existing tests | Very Low | Medium | Full test suite run at each stage |
| Scope creep into supervisor redesign | Low | High | Explicit non-goals and inflation guard above |
| Discovery of design gap during wiring | Low | Medium | Escalation protocol: pause tranche, document gap, separate stage |

---

## 7. Success Criteria for Tranche Closure (S331)

1. All 4 PWT items completed and verified.
2. Test suite passes with 0 failures (baseline: 186 tests).
3. All 9 invariants preserved.
4. Composed pipeline exercised in at least one integration-level test.
5. No new interfaces introduced.
6. No scope inflation beyond PWT-1 through PWT-4.
7. Retry metadata flows through actor-level structured logs (INV-OBS-5 compliance).

---

## 8. Predecessor Chain

```
S306 (Venue Readiness Charter)
  → S312 (Adapter Hardening Tranche Charter)
    → S315 (Foundational Tranche Gate — PASS)
      → S321 (Venue Closure Tranche Charter)
        → S326 (Venue Progression Evidence Gate — CLOSED)
          → S327 (Production Wiring Tranche Charter) ← this document
```
