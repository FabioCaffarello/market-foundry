# Order Lifecycle Semantics, States, and Non-Goals

**Stage:** S309
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Companion to:** [OMS and Order Lifecycle Charter](oms-and-order-lifecycle-charter.md)

---

## 1. Purpose

This document models the **minimum order lifecycle** the Foundry uses to track execution intents from creation to terminal state. It defines what each state means, who owns each transition, how fills relate to states, and — critically — what is explicitly excluded.

The lifecycle documented here is already implemented in code (`internal/domain/execution/execution.go`). This document elevates implementation to architecture by assigning semantics, invariants, and guard rails.

---

## 2. Lifecycle Concepts

### 2.1 Five Lifecycle Layers

The Foundry's order lifecycle operates across five distinct conceptual layers. Understanding these layers prevents conflation of concerns.

```
┌─────────────────────────────────────────────────────┐
│  Layer 1: ORDER INTENT                              │
│  What the system wants to do.                       │
│  Created by: derive binary                          │
│  Represented by: ExecutionIntent (submitted)        │
├─────────────────────────────────────────────────────┤
│  Layer 2: VENUE ORDER                               │
│  The actual order placed at the venue.              │
│  Created by: adapter via HTTP POST                  │
│  Represented by: VenueOrderReceipt.VenueOrderID     │
├─────────────────────────────────────────────────────┤
│  Layer 3: EXECUTION STATUS                          │
│  The venue's response mapped to domain status.      │
│  Owned by: execute binary (adapter layer)           │
│  Represented by: ExecutionIntent.Status             │
├─────────────────────────────────────────────────────┤
│  Layer 4: FILL EVENTS                               │
│  Price, quantity, fee, and timestamp of execution.  │
│  Created by: adapter from venue response            │
│  Represented by: FillRecord in ExecutionIntent.Fills│
├─────────────────────────────────────────────────────┤
│  Layer 5: CANCEL / REJECTION SEMANTICS              │
│  Terminal failure states and their meaning.          │
│  Owned by: adapter (maps venue) or actor (timeout)  │
│  Represented by: Status = rejected | cancelled      │
└─────────────────────────────────────────────────────┘
```

### 2.2 Layer Interactions

```
Derive ──→ [Layer 1: Intent Created]
               │
               ▼
Execute ──→ [Safety Gate Check]
               │ pass
               ▼
Adapter ──→ [Layer 2: Venue Order Placed]
               │
               ▼
Venue   ──→ [Response Received]
               │
               ▼
Adapter ──→ [Layer 3: Status Mapped] + [Layer 4: Fill Recorded]
               │                         or
               ▼                     [Layer 5: Rejection/Cancel]
Store   ──→ [KV + ClickHouse Persistence]
               │
               ▼
Gateway ──→ [HTTP Query Surface]
```

---

## 3. State Definitions

### 3.1 Complete State Table

| Status | Code | Terminal | Meaning | Owner | Paper Path | Venue Path |
|--------|------|----------|---------|-------|------------|------------|
| `submitted` | `StatusSubmitted` | No | Intent created and published; awaiting execution | Derive binary | Yes | Yes |
| `sent` | `StatusSent` | No | HTTP request dispatched to venue; awaiting response | Execute binary | No | Optional (async only) |
| `accepted` | `StatusAccepted` | No | Venue acknowledged the order; awaiting fill | Execute binary (adapter) | No | Yes |
| `partially_filled` | `StatusPartiallyFilled` | No | Venue filled part of the quantity; remainder pending | Execute binary (adapter) | No | Yes (rare for market) |
| `filled` | `StatusFilled` | **Yes** | Order fully executed; all quantity filled | Execute binary (adapter) | Yes (instant) | Yes |
| `rejected` | `StatusRejected` | **Yes** | Venue or system refused the order; zero fills | Execute binary (adapter) | No | Yes |
| `cancelled` | `StatusCancelled` | **Yes** | Order cancelled; partial fills may be preserved | Execute binary (adapter) | No | Yes |

### 3.2 State Semantics

#### `submitted` — The Origin State
- **Created by:** Derive binary's `PaperOrderEvaluator` after risk evaluation passes.
- **Meaning:** The pipeline has decided to execute. The intent is now in the execution queue.
- **Invariant:** `Fills` is empty. `FilledQuantity` is "0" or empty. `Final` is false.
- **Next states:** `sent` (async), `accepted` (sync), `rejected` (pre-submit failure).

#### `sent` — In-Flight (Async Path Only)
- **Created by:** Execute binary actor after dispatching HTTP request.
- **Meaning:** The venue request is in flight. The system is awaiting a response.
- **Invariant:** `Fills` is empty. `FilledQuantity` is "0".
- **Current usage:** Not used in synchronous Binance adapter. Reserved for future async adapters.
- **Next states:** `accepted`, `rejected`.

#### `accepted` — Venue Acknowledged
- **Created by:** Adapter mapping Binance `NEW` status.
- **Meaning:** The venue accepted the order and will attempt to fill it.
- **Invariant:** `Fills` may be empty (awaiting fill) or populated (partial).
- **For market orders:** This state is transient — Binance typically responds with `FILLED` directly.
- **Next states:** `filled`, `partially_filled`, `cancelled`.

#### `partially_filled` — Partial Execution
- **Created by:** Adapter mapping Binance `PARTIALLY_FILLED` status.
- **Meaning:** Some quantity was filled, but the order is not complete.
- **Invariant:** `len(Fills) >= 1`. `FilledQuantity < Quantity`. `FilledQuantity > 0`.
- **For market orders:** Extremely rare. Occurs only under extreme liquidity conditions.
- **Next states:** `filled`, `cancelled`.

#### `filled` — Terminal Success
- **Created by:** Adapter mapping Binance `FILLED` status.
- **Meaning:** The full requested quantity was executed at the venue.
- **Invariant:** `len(Fills) >= 1`. `FilledQuantity == Quantity`. `Final` is true.
- **Absorbing:** No further transitions allowed.

#### `rejected` — Terminal Refusal
- **Created by:** Adapter mapping Binance `REJECTED` or `EXPIRED`, or actor on pre-submit failure.
- **Meaning:** The order was refused. No execution occurred.
- **Invariant:** `len(Fills) == 0`. `FilledQuantity` is "0". `Final` is true.
- **Absorbing:** No further transitions allowed.
- **Causes:** Bad parameters, insufficient balance, rate limit, authentication failure, venue down.

#### `cancelled` — Terminal Cancellation
- **Created by:** Adapter mapping Binance `CANCELED` / `CANCELLED`.
- **Meaning:** The order was cancelled. If partial fills occurred before cancellation, they are preserved.
- **Invariant:** If `FilledQuantity > 0`, `len(Fills) >= 1`. `Final` is true.
- **Absorbing:** No further transitions allowed.
- **Note:** The Foundry does not initiate cancellations. This state reflects venue-side cancellation only.

---

## 4. Transition Matrix

### 4.1 Valid Transitions

```
                    ┌──────────────┐
                    │  submitted   │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         ┌────────┐  ┌──────────┐  ┌──────────┐
         │  sent  │  │ accepted │  │ rejected │ ■
         └───┬────┘  └────┬─────┘  └──────────┘
             │            │
        ┌────┼────┐  ┌────┼─────────────┐
        ▼    ▼    │  ▼    ▼             ▼
   ┌────────┐ ┌───┘ ┌────────┐  ┌────────────────┐  ┌───────────┐
   │accepted│ │     │ filled │■ │partially_filled │  │ cancelled │■
   └────┬───┘ │     └────────┘  └───────┬─────────┘  └───────────┘
        │     │                         │
        └─────┘                    ┌────┼────┐
                                   ▼         ▼
                              ┌────────┐ ┌───────────┐
                              │ filled │■│ cancelled │■
                              └────────┘ └───────────┘

■ = terminal state (absorbing)
```

### 4.2 Transition Table (Code-Aligned)

| From | To | Owner | Trigger | Contract |
|------|----|-------|---------|----------|
| (none) | `submitted` | Derive | Risk evaluation produces execution intent | C-SUB |
| `submitted` | `sent` | Execute (actor) | HTTP request dispatched to venue | C-VEN |
| `submitted` | `accepted` | Execute (adapter) | Synchronous venue response: `NEW` | C-ACK |
| `submitted` | `rejected` | Execute (adapter) | Venue rejects on submit or pre-submit failure | C-ACK, C-FAIL |
| `sent` | `accepted` | Execute (adapter) | Async venue acknowledgement | C-ACK |
| `sent` | `rejected` | Execute (adapter) | Async venue rejection | C-ACK, C-FAIL |
| `accepted` | `filled` | Execute (adapter) | Venue fills full quantity | C-ACK, C-FILL |
| `accepted` | `partially_filled` | Execute (adapter) | Venue fills partial quantity | C-ACK, C-FILL |
| `accepted` | `cancelled` | Execute (adapter) | Venue cancels (venue-initiated) | C-ACK |
| `partially_filled` | `filled` | Execute (adapter) | Remaining quantity filled | C-ACK, C-FILL |
| `partially_filled` | `cancelled` | Execute (adapter) | Venue cancels remainder | C-ACK |

### 4.3 Dominant Path (Binance Testnet Market Orders)

For the current scope (synchronous market orders on Binance Futures testnet), the overwhelmingly common paths are:

```
Happy path (~95%):   submitted → accepted → filled
Rejection path:      submitted → rejected
Direct fill:         submitted → filled  (some venues return FILLED directly)
```

The `sent`, `partially_filled`, and `cancelled` states exist for correctness but are rarely exercised with synchronous market orders.

---

## 5. Fill Event Semantics

### 5.1 Fill Record Structure

```go
type FillRecord struct {
    Price     string    // Average execution price (venue: avgPrice)
    Quantity  string    // Executed quantity (venue: executedQty)
    Fee       string    // Cumulative fee/commission (venue: cumQuote)
    Simulated bool      // false for real venue, true for paper
    Timestamp time.Time // Fill time (venue: updateTime)
}
```

### 5.2 Fill Rules

| Rule | ID | Description |
|------|----|-------------|
| Fill presence on filled | FR-1 | `status == filled` ⇒ `len(Fills) >= 1` |
| Fill presence on partial | FR-2 | `status == partially_filled` ⇒ `len(Fills) >= 1` |
| No fills on rejected | FR-3 | `status == rejected` ⇒ `len(Fills) == 0` |
| Partial fills on cancel | FR-4 | `status == cancelled AND FilledQuantity > 0` ⇒ `len(Fills) >= 1` |
| Quantity consistency | FR-5 | `status == filled` ⇒ `FilledQuantity == Quantity` |
| Partial consistency | FR-6 | `status == partially_filled` ⇒ `0 < FilledQuantity < Quantity` |
| Quantity monotonicity | FR-7 | `FilledQuantity[t] >= FilledQuantity[t-1]` (never decreases) |
| Fills append-only | FR-8 | Fills array only grows; existing entries are never modified |
| Temporal ordering | FR-9 | `fill.Timestamp >= intent.Timestamp` |

### 5.3 Paper vs. Venue Fill Discrimination

| Aspect | Paper Fill | Venue Fill |
|--------|-----------|------------|
| `Simulated` | `true` | `false` |
| `Price` | Derived from last known price | Venue `avgPrice` |
| `Quantity` | Equals `intent.Quantity` | Venue `executedQty` |
| `Fee` | "0" | Venue `cumQuote` |
| `Timestamp` | `time.Now()` | Venue `updateTime` |
| Lifecycle | `submitted → filled` (instant) | Full state machine |

---

## 6. Cancel Semantics

### 6.1 What "Cancel" Means in the Foundry

The Foundry **does not initiate order cancellations**. The `cancelled` state exists solely to represent venue-side cancellation of orders. This is a critical distinction:

| Aspect | Traditional OMS | Foundry |
|--------|----------------|---------|
| Who cancels | User, system, or venue | Venue only |
| Cancel request API | Yes | No |
| Cancel confirmation flow | Request → Pending → Confirmed | Venue response maps directly |
| Partial cancel | Supported | Fills preserved, remainder lost |

### 6.2 When Cancellation Occurs

For Binance Futures testnet market orders, venue-side cancellation is rare but can happen:

| Scenario | Binance Status | Domain Status |
|----------|---------------|---------------|
| Self-trade prevention | `CANCELED` | `cancelled` |
| Expired (FOK/IOC not filled in time) | `EXPIRED` | `rejected` (not cancelled) |
| Market order no liquidity | `CANCELED` | `cancelled` |

**Design decision:** `EXPIRED` maps to `rejected`, not `cancelled`, because expiration means the venue never executed — semantically equivalent to rejection.

### 6.3 Reconciliation of Partial Fills on Cancel

If an order receives partial fills before cancellation:

```
accepted → partially_filled → cancelled
```

The `cancelled` intent preserves:
- `FilledQuantity` > 0 (the portion that was filled)
- `Fills` array contains the partial fill records
- `Final` is true
- The unfilled remainder is **lost** — no resubmission, no retry

This is acceptable because:
1. Partial fills on market orders are extremely rare;
2. The Foundry does not manage positions — each intent is independent;
3. Resubmission logic belongs to a future stage (if ever needed).

---

## 7. Rejection Semantics

### 7.1 Rejection Sources

| Source | Examples | Retryable | Domain Status |
|--------|----------|-----------|---------------|
| Venue client error | Bad symbol, insufficient balance | No | `rejected` |
| Venue auth failure | Invalid API key, expired signature | No | `rejected` |
| Venue rate limit | 429 Too Many Requests | Yes (at adapter level) | `rejected` |
| Venue unavailable | 503 Service Unavailable | Yes (at adapter level) | `rejected` |
| Venue server error | 500 Internal Server Error | Yes (at adapter level) | `rejected` |
| Network failure | Timeout, DNS, connection refused | Yes (at adapter level) | `rejected` |
| Pre-submit gate | Kill switch halted, staleness exceeded | No | Intent never transitions from `submitted` |

### 7.2 Retryable vs. Terminal Rejection

The `rejected` status is always **terminal at the domain level**. The `retryable` flag on `Problem` is an adapter-level hint for future retry infrastructure (S310). The domain model does not retry — a rejected intent stays rejected.

```
Domain level:     rejected = terminal, absorbing, no retry
Adapter level:    Problem.Retryable() = hint for S310 failure envelope
```

### 7.3 Pre-Submit Rejection (Safety Gate)

If the safety gate (kill switch or staleness guard) blocks execution:
- The intent **remains in `submitted` status** — it is not transitioned to `rejected`.
- The actor publishes a diagnostic event but does not modify the intent.
- The intent will be retried on the next evaluation cycle (if the gate reopens).

This is different from venue rejection: a gate block is temporary and does not consume the intent.

---

## 8. Non-Goals

### 8.1 Explicit Non-Goals for This Stage

| ID | Non-Goal | Rationale |
|----|---------|-----------|
| NG-OMS-1 | **Order book / active orders ledger** | Fire-and-forget; no in-flight management |
| NG-OMS-2 | **Order amendment / modification** | Market orders only; nothing to amend |
| NG-OMS-3 | **User-initiated cancellation** | System does not cancel; venue cancels |
| NG-OMS-4 | **Position tracking / net exposure** | Per-intent, not per-symbol aggregate |
| NG-OMS-5 | **Smart order routing** | Single venue; no routing decision |
| NG-OMS-6 | **Allocation / order splitting** | Single account; no splitting needed |
| NG-OMS-7 | **Multi-venue management** | One adapter (Binance testnet) |
| NG-OMS-8 | **Async fill reconciliation** | Synchronous response model |
| NG-OMS-9 | **Order queuing / throttling / scheduling** | Actor processes sequentially per symbol |
| NG-OMS-10 | **Historical order search / blotter UI** | Composite HTTP read model is sufficient |
| NG-OMS-11 | **Compliance / audit trail** | Not applicable at testnet |
| NG-OMS-12 | **Multi-account management** | Single credential set |
| NG-OMS-13 | **Limit / stop / conditional orders** | Market orders only (S306 scope freeze) |
| NG-OMS-14 | **Retry infrastructure** | Deferred to S310 failure envelope |
| NG-OMS-15 | **Operational dashboards / alerting** | Composite HTTP read model only |

### 8.2 Why These Are Non-Goals

Each non-goal maps to a specific **scope inflation risk**:

| Risk | Non-Goals That Block It |
|------|------------------------|
| Building a broker | NG-OMS-1, 2, 3, 5, 6, 7, 12 |
| Building an EMS | NG-OMS-5, 6, 7, 9 |
| Building a portfolio system | NG-OMS-4, 11 |
| Building ops infrastructure | NG-OMS-10, 15 |
| Premature generalisation | NG-OMS-8, 13, 14 |

### 8.3 Future Evolution Path

Some non-goals may become goals in future waves. The expected evolution:

| Capability | Earliest Stage | Prerequisite |
|-----------|----------------|-------------|
| Retry with backoff | S310 | Failure envelope classification |
| Async fill handling | Post-S312 | WebSocket adapter |
| Client order ID | Post-S312 | Venue-side idempotency |
| Per-symbol kill switch | Post-S312 | Kill switch granularity |
| Position tracking | Post-S312 | OMS module (new wave) |
| Limit orders | Post-S312 | Order type extension |
| Multi-venue | Post-S312 | Adapter registry |

---

## 9. Guard Rails for Production

### 9.1 Lifecycle Guard Rails

| ID | Guard Rail | Enforcement |
|----|-----------|-------------|
| GR-1 | **Terminal states are absorbing** | `validTransitions` map has no entries for terminal states |
| GR-2 | **Status only moves forward** | `ValidTransition()` rejects backward transitions |
| GR-3 | **FilledQuantity never decreases** | QM-1 invariant (S308) |
| GR-4 | **Fills are append-only** | No delete or modify on `Fills` slice |
| GR-5 | **Safety gate checked before submit** | Actor layer checks `ControlGate.IsHalted()` before `VenuePort.SubmitOrder()` |
| GR-6 | **Staleness checked before submit** | Actor layer verifies intent timestamp freshness |
| GR-7 | **No domain model mutation in adapter** | Adapter returns `VenueOrderReceipt`; actor applies state change |
| GR-8 | **Credentials never logged** | `CredentialSet` design prevents credential leakage |

### 9.2 Failure Envelope Preparation (for S310)

This lifecycle model prepares the following failure concepts for S310:

| Concept | Definition | S310 Responsibility |
|---------|-----------|---------------------|
| **Containment** | A venue failure for symbol A does not affect symbol B | Prove isolation under failure |
| **Classification** | Each failure maps to a `Problem` category with retryable flag | Enumerate failure scenarios |
| **Terminal finality** | Rejected/cancelled intents are never reprocessed | Prove no zombie orders |
| **Gate halt** | Kill switch stops all submissions; staleness blocks stale intents | Prove halt/resume cycle |
| **Diagnostic emission** | Failures produce observable events without blocking pipeline | Define event schema |

---

## 10. Verification Criteria

### 10.1 How to Verify This Lifecycle

| Criterion | Verification Method |
|-----------|-------------------|
| All 7 states are reachable | Unit tests exercising each transition |
| Terminal states are absorbing | `ValidTransition(terminal, any)` returns false |
| Fill rules FR-1 through FR-9 hold | Property-based test on fill/status combinations |
| Paper path is `submitted → filled` | Paper adapter unit test |
| Venue path traverses full machine | Binance adapter unit test with mock responses |
| Safety gate blocks submission | Actor test with halted gate |
| Correlation IDs propagate | E2E scenario tracing CorrelationID through pipeline |
| No domain model changes needed | Diff check: `execution.go` unchanged after S309 |

### 10.2 What S310 Must Verify (Not S309)

| Criterion | Deferred To |
|-----------|-------------|
| Retry with backoff under retryable failure | S310 |
| Containment: symbol A failure does not affect B | S310/S311 |
| Network failure classification completeness | S310 |
| Kill switch halt/resume under real venue | S310/S311 |

---

*Delivered: 2026-03-21 — Stage S309, Phase 30*
