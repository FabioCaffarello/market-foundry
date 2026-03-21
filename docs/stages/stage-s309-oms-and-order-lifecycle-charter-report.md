# Stage S309 — OMS and Order Lifecycle Charter

**Status:** DELIVERED
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Predecessor:** S308 — Venue Execution Contracts and Invariants
**Successor:** S310 — Venue Failure Envelope and Containment

---

## 1. Executive Summary

S309 defines the minimum OMS semantics and order lifecycle the Foundry requires to operate venue execution safely — without building an actual Order Management System.

**Key finding:** The Foundry already possesses all necessary OMS capabilities embedded in its domain model (`ExecutionIntent`, `FillRecord`, `Status` lifecycle, `ControlGate`). No new modules, services, or data stores are required. What was missing was the **explicit charter** that:

1. Names these capabilities as the Foundry's OMS surface;
2. Models the five lifecycle layers (intent, venue order, execution status, fill events, cancel/rejection);
3. Assigns precise semantics to each of the 7 states;
4. Documents 15 explicit non-goals that prevent scope inflation toward broker/EMS territory;
5. Prepares guard rails and failure envelope concepts for S310.

**Verdict:** The Foundry's order lifecycle is sufficient for venue readiness. S310 can proceed to define the failure envelope and containment model on top of the lifecycle formalised here.

---

## 2. Deliverables

| # | Artefact | Path | Content |
|---|---------|------|---------|
| 1 | OMS Charter | `docs/architecture/oms-and-order-lifecycle-charter.md` | Capability assessment, order model, ownership matrix, scope freeze, governing principles |
| 2 | Lifecycle Semantics | `docs/architecture/order-lifecycle-semantics-states-and-non-goals.md` | 5 lifecycle layers, 7 state definitions, transition matrix, fill rules, cancel/rejection semantics, 15 non-goals, guard rails |
| 3 | Stage Report | `docs/stages/stage-s309-oms-and-order-lifecycle-charter-report.md` | This document |

---

## 3. OMS Capability Assessment

### 3.1 Capabilities Present (10)

| ID | Capability | Code Location |
|----|-----------|---------------|
| OMS-C1 | Order intent creation | `ExecutionIntent` struct (`execution.go:98-116`) |
| OMS-C2 | Synchronous order submission | `VenuePort.SubmitOrder()` (`venue.go:27`) |
| OMS-C3 | Status tracking per intent | `Status` field on `ExecutionIntent` |
| OMS-C4 | Fill recording | `FillRecord` struct, `Fills []FillRecord` (`execution.go:76-83`) |
| OMS-C5 | Terminal state recognition | `IsTerminal()` (`execution.go:48-52`) |
| OMS-C6 | State transition validation | `ValidTransition()` + `validTransitions` map (`execution.go:54-74`) |
| OMS-C7 | Safety gate enforcement | `ControlGate` (`control.go:5-39`) |
| OMS-C8 | Correlation tracing | `CorrelationID`, `CausationID` fields |
| OMS-C9 | Paper/venue discrimination | `FillRecord.Simulated` flag |
| OMS-C10 | Analytical persistence | Writer + composite read model (ClickHouse) |

### 3.2 Capabilities Excluded (12)

| ID | Excluded | Risk Blocked |
|----|---------|-------------|
| OMS-X1 | Order book / active ledger | Broker creep |
| OMS-X2 | Order amendment | Limit order creep |
| OMS-X3 | User-initiated cancellation | Cancel flow creep |
| OMS-X4 | Smart order routing | EMS creep |
| OMS-X5 | Portfolio position tracking | Portfolio system creep |
| OMS-X6 | Allocation / splitting | Multi-account creep |
| OMS-X7 | Multi-venue arbitrage | Multi-venue creep |
| OMS-X8 | Async fill reconciliation | Async complexity creep |
| OMS-X9 | Order queuing / throttling | Scheduler creep |
| OMS-X10 | Blotter UI | Dashboard creep |
| OMS-X11 | Compliance audit trail | Regulatory creep |
| OMS-X12 | Multi-account management | Account management creep |

### 3.3 Assessment Verdict

**No new OMS module needed.** The `ExecutionIntent` + `FillRecord` + `Status` lifecycle constitutes the Foundry's minimal OMS. The charter formalises this and locks the boundary.

---

## 4. Lifecycle Model

### 4.1 Five Lifecycle Layers

| Layer | Concept | Owner | Representation |
|-------|---------|-------|----------------|
| 1 | Order Intent | Derive binary | `ExecutionIntent` in `submitted` status |
| 2 | Venue Order | Adapter (HTTP POST) | `VenueOrderReceipt.VenueOrderID` |
| 3 | Execution Status | Execute binary (adapter) | `ExecutionIntent.Status` |
| 4 | Fill Events | Adapter from venue response | `FillRecord` in `ExecutionIntent.Fills` |
| 5 | Cancel / Rejection | Adapter (maps venue) or actor (timeout) | Terminal `Status` |

### 4.2 Seven States

| State | Terminal | Semantics |
|-------|----------|-----------|
| `submitted` | No | Intent created; awaiting execution |
| `sent` | No | HTTP dispatched; awaiting response (async only) |
| `accepted` | No | Venue acknowledged; awaiting fill |
| `partially_filled` | No | Partial quantity filled; remainder pending |
| `filled` | **Yes** | Full quantity executed |
| `rejected` | **Yes** | Venue or system refused; zero fills |
| `cancelled` | **Yes** | Venue-side cancellation; partial fills preserved |

### 4.3 Dominant Path

```
Happy path (~95%):  submitted → accepted → filled
Rejection:          submitted → rejected
```

The `sent`, `partially_filled`, and `cancelled` states exist for correctness but are rarely exercised with synchronous market orders on Binance.

---

## 5. Key Design Decisions

### 5.1 Intent vs. Order

The Foundry uses `ExecutionIntent`, not "Order", as its primary abstraction. This prevents OMS vocabulary from pulling in OMS expectations. The intent represents what the **system** wants to do, not what a **user** requested.

### 5.2 Fire-and-Forget

Submit → receive response → record outcome → done. No in-flight order management, no pending orders queue, no active orders monitor. Each intent is independent.

### 5.3 Correlation Over Identity

The Foundry traces execution through `CorrelationID` / `CausationID` (pipeline lineage), not through `VenueOrderID` (venue-specific). The venue order ID is captured but is not the primary identity.

### 5.4 No System-Initiated Cancellation

The `cancelled` state exists only for venue-side cancellation. The Foundry never sends a cancel request. This eliminates an entire category of state management complexity.

### 5.5 Rejection Finality at Domain Level

`rejected` is terminal and absorbing. The `retryable` flag on `Problem` is an adapter-level hint for future retry infrastructure (S310), not a domain-level retry mechanism.

### 5.6 Safety Gate Is Pre-Submit, Not In-Submit

The kill switch and staleness guard operate in the actor layer, before calling `VenuePort.SubmitOrder()`. The adapter never checks the gate. This keeps the adapter focused on venue translation.

---

## 6. Fill and Cancel Semantics

### 6.1 Fill Rules (9 Rules: FR-1 through FR-9)

- Filled status requires at least one fill record with full quantity match
- Rejected status requires zero fills
- Cancelled status preserves any partial fills that occurred
- Filled quantity is monotonically non-decreasing
- Fills array is append-only
- Fill timestamp must be >= intent timestamp

### 6.2 Cancel Rules

- The Foundry does not initiate cancellations
- `EXPIRED` maps to `rejected` (not cancelled) — expiration means no execution
- Partial fills before cancellation are preserved in the intent
- Unfilled remainder on cancel is lost — no resubmission

### 6.3 Rejection Sources

Eight failure classes documented in S308 (C-FAIL): auth, client error, rate limit, venue unavailable, server error, network, parse, unknown. All map to `rejected` at domain level with `retryable` hint at adapter level.

---

## 7. Non-Goals Summary

15 explicit non-goals (NG-OMS-1 through NG-OMS-15) documented in the lifecycle semantics artefact. Grouped by scope inflation risk:

| Risk Category | Non-Goals |
|--------------|-----------|
| Broker / EMS | Order book, amendment, cancellation, routing, allocation, multi-venue, multi-account |
| Portfolio | Position tracking, compliance |
| Infrastructure | Async reconciliation, queuing, dashboards, alerting |
| Premature generalisation | Limit orders, retry infrastructure |

---

## 8. Guard Rails

### 8.1 Lifecycle Guard Rails (8 Rules: GR-1 through GR-8)

| Rule | Protection |
|------|-----------|
| Terminal states are absorbing | No zombie orders |
| Status only moves forward | No state regression |
| FilledQuantity never decreases | Quantity integrity |
| Fills are append-only | Fill integrity |
| Safety gate before submit | Gate bypass prevention |
| Staleness before submit | Stale intent prevention |
| No domain mutation in adapter | Boundary discipline |
| Credentials never logged | Security |

### 8.2 Failure Envelope Preparation

Five failure concepts prepared for S310:

| Concept | S310 Responsibility |
|---------|---------------------|
| Containment | Prove per-symbol failure isolation |
| Classification | Enumerate failure scenarios against C-FAIL taxonomy |
| Terminal finality | Prove no rejected/cancelled intent reprocessing |
| Gate halt | Prove halt/resume cycle under real venue |
| Diagnostic emission | Define failure event schema |

---

## 9. Residual Gaps

| Gap | Severity | Owner | Status |
|-----|----------|-------|--------|
| Retry with backoff | Medium | S310 | Deferred — failure envelope scope |
| Failure containment proof | Medium | S310/S311 | Deferred — multi-symbol isolation |
| Client order ID | Low | Post-S312 | Venue-side idempotency |
| Async fill handling | Low | Post-S312 | Non-goal (NG-OMS-8) |
| Position tracking | Low | Post-S312 | Non-goal (NG-OMS-4) |
| Limit/stop orders | Low | Post-S312 | Non-goal (NG-OMS-13) |

---

## 10. S310 Preparation

S310 (Venue Failure Envelope and Containment) can now proceed with:

### 10.1 Prerequisites Met

- Lifecycle semantics fully documented — every state has defined meaning
- Terminal state finality established — rejected/cancelled are absorbing
- Failure classification taxonomy inherited from S308 (C-FAIL)
- Guard rails defined — enforcement points are clear
- Non-goals prevent scope inflation into retry/reconciliation territory

### 10.2 S310 Focus Areas

| Task | Input From S309 |
|------|----------------|
| Define failure containment per symbol | Lifecycle guard rails (GR-1 through GR-8) |
| Classify all rejection scenarios | Rejection semantics (Section 7 of lifecycle doc) |
| Prove terminal finality under failure | Terminal state semantics (filled, rejected, cancelled) |
| Prove gate halt/resume under real venue | Safety gate pre-submit rule (GR-5, GR-6) |
| Define diagnostic event schema for failures | Failure envelope preparation concepts |

### 10.3 Recommended S310 Scope

1. Failure scenario enumeration against C-FAIL taxonomy
2. Per-symbol containment proof (symbol A failure does not block symbol B)
3. Terminal finality proof (no zombie orders after rejection/cancellation)
4. Gate halt/resume cycle proof under real venue interaction
5. Diagnostic event schema for failure observability

---

## 11. Governing Question Progress

| Question | S309 Contribution |
|----------|------------------|
| VQ1: Adapter submits and receives? | Order model clarifies intent → venue order → receipt flow |
| VQ2: Lifecycle reflects venue states? | 7 states, transition matrix, and state semantics formalised |
| VQ3: Real fills persist without schema changes? | Fill rules FR-1 through FR-9 confirm schema sufficiency |
| VQ4: Composite read works with real data? | Paper/venue discrimination documented (Simulated flag) |
| VQ5: Failures classified and contained? | Rejection semantics + failure envelope preparation for S310 |
| VQ6: Safety gate enforced? | Pre-submit gate rule (GR-5, GR-6) formalised |
| VQ7: Multi-symbol isolation maintained? | Per-intent model prevents cross-symbol state leakage |

---

## 12. Acceptance Criteria Checklist

| Criterion | Met |
|-----------|-----|
| Minimum OMS semantics are clear | Yes — 10 capabilities named, 12 excluded |
| Order lifecycle is modelled usefully | Yes — 5 layers, 7 states, transition matrix, fill rules |
| Non-goals prevent scope inflation | Yes — 15 non-goals blocking 5 risk categories |
| Stage prepares guard rails and failure envelope for S310 | Yes — 8 guard rails, 5 failure concepts |
| No new OMS module or service introduced | Yes — charter only |
| No domain model changes | Yes — `execution.go` unchanged |
| Intent vs. order distinction is clear | Yes — Section 4.2 of charter |
| Cancel semantics documented | Yes — venue-only cancellation, no system-initiated |
| Rejection semantics documented | Yes — 8 failure classes, domain-terminal with adapter-retryable hint |

---

*Delivered: 2026-03-21 — Stage S309, Phase 30*
