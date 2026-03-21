# Stage S308 — Venue Execution Contracts and Invariants

**Status:** DELIVERED
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Predecessor:** S307 — Production Gap Map
**Successor:** S309 — End-to-End Venue Integration

---

## 1. Executive Summary

S308 formalises the minimum contracts and invariants required to bridge paper execution and real venue order lifecycle. The stage delivers two architecture artefacts that define every contract, state, transition, invariant, and ownership boundary for venue execution — without opening implementation scope.

**Key outcome:** The execution domain model, state machine, and fill record structure already in code are **sufficient** for venue operation. No domain model changes are required. What was missing was the explicit documentation of production invariants, terminal failure semantics, transition ownership, and boundary rules that must hold when the pipeline moves from simulated fills to real venue interaction.

**Verdict:** Ready for S309 implementation. All contracts are verifiable through existing test patterns.

---

## 2. Deliverables

| # | Artefact | Path | Content |
|---|---------|------|---------|
| 1 | Contracts & Invariants | `docs/architecture/venue-execution-contracts-and-invariants.md` | 5 contract categories, 5 cross-cutting invariants, verification criteria |
| 2 | State Model & Boundaries | `docs/architecture/venue-order-state-model-transitions-and-boundaries.md` | 7 states, transition matrix, ownership map, terminal semantics, guard rails |
| 3 | Stage Report | `docs/stages/stage-s308-venue-execution-contracts-and-invariants-report.md` | This document |

---

## 3. Contracts Defined

### 3.1 Five Contract Categories

| Contract | ID | Scope |
|---------|----|-------|
| Submit Intent | C-SUB | Pre-conditions for dispatching an intent to venue (safety gates, validation, side filter) |
| Venue Submission | C-VEN | VenuePort interface invariants (request immutability, context deadline, credential safety) |
| Fill Record | C-FILL | Fill field mapping from venue to domain (price, quantity, fee, simulated, timestamp) with consistency rules CR-1 through CR-5 |
| Acceptance / Rejection | C-ACK | Binance status → domain status mapping (6 venue statuses → 5 domain statuses) |
| Terminal Failure | C-FAIL | Error classification taxonomy (8 failure classes with problem category and retryable flag) |

### 3.2 Alignment With S307 Blockers

All 6 S308-assigned blockers from S307 are addressed:

| Blocker | Resolution |
|---------|-----------|
| VA-2 (EXPIRED mapping) | C-ACK: `EXPIRED → rejected` |
| VA-3 (CANCELED mapping) | C-ACK: `CANCELED → cancelled` |
| VA-4 (REJECTED mapping) | C-ACK: `REJECTED → rejected` |
| FM-1 (real price) | C-FILL: `Price = avgPrice` |
| FM-2 (real quantity) | C-FILL: `Quantity = executedQty` |
| FM-3 (real fee) | C-FILL: `Fee = cumQuote` |
| FM-4 (real timestamp) | C-FILL: `Timestamp = updateTime` |
| FM-5 (simulated flag) | C-FILL: `Simulated = false` |

Note: The adapter already implements VA-2/VA-3/VA-4 and FM-1 through FM-5 correctly in `binance_futures_testnet_adapter.go:214-228` and `binance_futures_testnet_adapter.go:196-204`. S308 elevates these from implementation details to documented contracts.

---

## 4. Invariants Explicated

### 4.1 Five Cross-Cutting Invariant Groups

| Invariant | ID | Rules | Enforcement Point |
|-----------|-----|-------|-------------------|
| Idempotency | INV-IDEM | IDEM-1 through IDEM-3 | JetStream dedup, KV monotonicity, future client order ID |
| State Monotonicity | INV-MONO | MONO-1 through MONO-5 | `ValidTransition()`, terminal state absorption, append-only fills |
| Correlation/Causation | INV-TRACE | TRACE-1 through TRACE-5 | ExecutionIntent fields, propagation through pipeline |
| Transition Ownership | INV-OWN | 11 transitions mapped | Derive binary (submitted), execute binary (all others) |
| Temporal | INV-TIME | TIME-1 through TIME-5 | Staleness guard, fill timestamp ordering |

### 4.2 Key Findings

1. **State machine is already correct.** The `validTransitions` map in `domain/execution/execution.go:55-60` exactly matches venue order lifecycle requirements. No new states needed.

2. **Terminal states are well-defined.** `IsTerminal()` correctly identifies `filled`, `rejected`, `cancelled`. The `Final` flag provides an additional explicit marker.

3. **Idempotency has one gap.** JetStream dedup and KV monotonicity are proven. Venue-side idempotency (client order ID) is the S307 EC-1 blocker — S308 defines the derivation rule (IDEM-3) but implementation belongs to S307.

4. **Correlation/causation chain is complete in schema** (`CorrelationID`, `CausationID` on `ExecutionIntent`). Runtime enforcement needs scenario validation in S309.

5. **Ownership boundary is clean.** Derive binary owns only `→ submitted`. Execute binary owns all post-submitted transitions. Store binary owns persistence. No overlap.

---

## 5. State Model Summary

### 5.1 States

| State | Terminal | Paper | Venue |
|-------|----------|-------|-------|
| `submitted` | No | Yes | Yes |
| `sent` | No | No | Optional (async) |
| `accepted` | No | No-action only | Yes |
| `partially_filled` | No | No | Yes (rare for market orders) |
| `filled` | Yes | Yes (instant) | Yes |
| `rejected` | Yes | No | Yes |
| `cancelled` | Yes | No | Yes |

### 5.2 Typical Venue Path (Binance Testnet Market Orders)

```
submitted → accepted → filled       (success: ~95% of market orders)
submitted → rejected                (failure: bad params, no balance)
```

### 5.3 Intent-Fill Consistency

7 rules (IFC-1 through IFC-7) ensure that fills, `FilledQuantity`, and status are mutually consistent at all times.

---

## 6. Boundaries and Ownership

### 6.1 Layer Responsibility Matrix

| Layer | Creates | Transitions | Persists | Queries |
|-------|---------|------------|----------|---------|
| Derive | ExecutionIntent | `→ submitted` | No | No |
| Execute (actor) | Safety check result | `submitted → *` | No | Kill switch |
| Execute (adapter) | Fill records | Maps venue → domain | No | No |
| Store | KV entries, CH rows | None | Yes | No |
| Gateway | HTTP responses | None | No | Yes |

### 6.2 Paper-Venue Discrimination

Single field: `FillRecord.Simulated`. No branching logic — only observability metadata.

---

## 7. Residual Gaps

| Gap | Severity | Owner | Status |
|-----|----------|-------|--------|
| Client order ID | High | S307 (EC-1) | Contract defined (IDEM-3); implementation pending |
| Retry with backoff | Medium | S310 | Deferred to failure envelope stage |
| Per-symbol kill switch | Low | Post-S312 | Global gate sufficient for testnet scope |
| Async fill handling | Low | Post-S312 | Non-goal (NG-5) |
| Partial fill aggregation | Low | Post-S312 | Single fill per market order |
| Position tracking | Low | Post-S312 | Not OMS |

---

## 8. Verification Readiness

All contracts are verifiable through unit and scenario tests:

| Contract | Test Type | Infrastructure Required |
|----------|----------|----------------------|
| C-SUB | Unit | None (pure functions) |
| C-VEN | Unit | `httptest.Server` mock |
| C-FILL | Unit | None (struct assertions) |
| C-ACK | Unit | None (switch coverage) |
| C-FAIL | Unit | `httptest.Server` returning error codes |
| INV-IDEM | Unit + Scenario | JetStream (already in CI) |
| INV-MONO | Unit | None (ValidTransition) |
| INV-TRACE | Scenario | NATS pipeline (existing smoke) |

---

## 9. S309 Preparation

S309 (End-to-End Venue Integration) can now proceed with:

### 9.1 Prerequisites Met

- All contracts documented and traceable to code
- State machine confirmed sufficient — no model changes needed
- Transition ownership clear — no ambiguity in binary responsibilities
- Fill field mapping explicit — adapter already implements correctly
- Safety gate boundary defined — check before VenuePort, not inside

### 9.2 S309 Implementation Focus

| Task | Input From S308 |
|------|----------------|
| Wire execute binary actor to call VenuePort under safety gate | C-SUB, INV-OWN |
| Validate real Binance fills against C-FILL rules | C-FILL, CR-1 through CR-5 |
| Prove lifecycle traversal with real venue responses | State model, transition matrix |
| Enforce kill switch under real venue | Kill switch boundary rules |
| Prove staleness guard under real venue | INV-TIME, TIME-4 |
| Validate composite read model with real data | C-FILL (Simulated=false path) |

### 9.3 Recommended S309 Scope

1. Execute binary actor integration with `BinanceFuturesTestnetAdapter`
2. Safety gate enforcement proof under real venue
3. E2E scenario: intent → venue submit → fill → KV → ClickHouse → HTTP query
4. Kill switch halt/resume proof under real venue
5. Regression: paper pipeline unchanged (zero regressions)

---

## 10. Governing Question Progress

| Question | S308 Contribution |
|----------|------------------|
| VQ1: Adapter submits and receives? | Contracts defined (C-VEN, C-ACK) |
| VQ2: Lifecycle reflects venue states? | State model formalised; mapping documented |
| VQ3: Real fills persist without schema changes? | Confirmed: FillRecord schema accommodates real values |
| VQ4: Composite read works with real data? | Confirmed: Simulated flag is metadata, not branching |
| VQ5: Failures classified and contained? | C-FAIL taxonomy with 8 classes |
| VQ6: Safety gate enforced? | Boundary rules documented |
| VQ7: Multi-symbol isolation maintained? | No state model impact; proven in Phase 29 |

---

## 11. Acceptance Criteria Checklist

| Criterion | Met |
|-----------|-----|
| Minimum venue execution contracts are clear | ✓ — 5 contract categories with explicit rules |
| Critical invariants are explicated | ✓ — 5 invariant groups, 25+ individual rules |
| Boundaries between layers remain clean | ✓ — 5-layer responsibility matrix |
| Stage prepares base for OMS/lifecycle in S309 | ✓ — implementation focus and prerequisites documented |
| No implementation opened | ✓ — design artefacts only |
| No multi-venue scope | ✓ — single adapter (Binance testnet) |
| No infinite spec | ✓ — 2 documents, bounded scope |
| Paper vs. production states mediated | ✓ — Simulated flag discrimination documented |

---

*Delivered: 2026-03-21 — Stage S308, Phase 30*
