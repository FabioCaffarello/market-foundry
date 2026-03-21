# Venue Order State Model, Transitions, and Boundaries

> Stage S308 — Design artefact.
> Formalises the execution lifecycle state machine for venue orders,
> documenting every state, valid transition, ownership, and boundary condition.

---

## 1. Purpose

The execution domain already defines a state machine in code (`validTransitions` map in `domain/execution/execution.go`). This document **elevates that machine to an architectural artefact**, adding:

- semantics for each state under real venue operation;
- ownership assignment per transition;
- boundary conditions that distinguish paper from venue behaviour;
- terminal failure semantics;
- guard-rail rules that prevent invalid lifecycle traversal.

---

## 2. State Definitions

### 2.1 Complete State Catalog

| Status | Terminal | Paper Mode | Venue Mode | Semantics |
|--------|----------|-----------|------------|-----------|
| `submitted` | No | Produced by evaluator | Produced by evaluator | Intent created; not yet dispatched to venue |
| `sent` | No | Not produced | Optional | Intent dispatched to venue API; awaiting acknowledgement |
| `accepted` | No | Not produced (no-action only) | Produced by adapter | Venue acknowledged order; awaiting fill |
| `partially_filled` | No | Not produced | Produced by adapter | Venue partially executed; remainder pending |
| `filled` | Yes | Instant (paper simulator) | Produced by adapter | Order fully executed; all quantity filled |
| `rejected` | Yes | Not produced | Produced by adapter | Venue refused order (or order expired) |
| `cancelled` | Yes | Not produced | Produced by adapter | Order cancelled (with or without partial fill) |

### 2.2 Paper vs. Venue State Traversal

**Paper mode** (current production path):
```
submitted ──→ filled        (actionable: side = buy/sell)
submitted ──→ [no change]   (no-action: side = none)
```

**Venue mode** (target path):
```
submitted ──→ sent ──→ accepted ──→ filled
                                ──→ partially_filled ──→ filled
                                                     ──→ cancelled
                       ──→ rejected
              ──→ rejected
```

**Synchronous market order shortcut** (Binance testnet typical path):
```
submitted ──→ accepted ──→ filled    (most common)
submitted ──→ rejected               (insufficient balance, invalid symbol)
```

The `sent` state is optional for synchronous adapters. It becomes mandatory when async dispatch is introduced (post-S312).

---

## 3. Transition Matrix

### 3.1 Valid Transitions (Code-Authoritative)

Source: `domain/execution/execution.go:55-60`

| From | To | Valid |
|------|----|-------|
| `submitted` | `sent` | ✓ |
| `submitted` | `accepted` | ✓ |
| `submitted` | `rejected` | ✓ |
| `sent` | `accepted` | ✓ |
| `sent` | `rejected` | ✓ |
| `accepted` | `filled` | ✓ |
| `accepted` | `partially_filled` | ✓ |
| `accepted` | `cancelled` | ✓ |
| `partially_filled` | `filled` | ✓ |
| `partially_filled` | `cancelled` | ✓ |
| Any terminal | Any | ✗ |
| Any | `submitted` | ✗ |

### 3.2 Transition Diagram

```
                    ┌──────────────────────────────────────────────┐
                    │                                              │
                    ▼                                              │
              ┌───────────┐                                       │
              │ submitted │──────────────────────────────┐        │
              └─────┬─────┘                              │        │
                    │                                    │        │
           ┌───────┼────────┐                            │        │
           ▼       │        ▼                            ▼        │
      ┌────────┐   │   ┌──────────┐                ┌──────────┐  │
      │  sent  │   │   │ accepted │                │ rejected │  │
      └───┬────┘   │   └────┬─────┘                │ TERMINAL │  │
          │        │        │                      └──────────┘  │
     ┌────┼────┐   │   ┌────┼──────────┐                         │
     ▼    │    │   │   ▼    │          ▼                         │
accepted  │ rejected  filled│   partially_filled                 │
     │    │    │      TERM  │          │                         │
     │    │    │            │     ┌────┼────┐                    │
     │    │    │            │     ▼         ▼                    │
     │    │    │            │  filled   cancelled                │
     │    │    │            │  TERMINAL TERMINAL                 │
     └────┘    └────────────┘                                    │
           (merged into accepted                                 │
            path above)           ownership boundary ────────────┘
                                  derive │ execute
```

### 3.3 Transition Ownership

| Transition | Binary | Component | Trigger |
|-----------|--------|-----------|---------|
| `→ submitted` | derive | PaperOrderEvaluator | Risk event processed |
| `submitted → sent` | execute | VenueAdapterActor | HTTP request dispatched |
| `submitted → accepted` | execute | VenuePort adapter | Synchronous venue response |
| `submitted → rejected` | execute | VenuePort adapter | Venue rejects on submit |
| `sent → accepted` | execute | VenuePort adapter | Venue acknowledges |
| `sent → rejected` | execute | VenuePort adapter | Venue rejects after dispatch |
| `accepted → filled` | execute | VenuePort adapter | Venue fills completely |
| `accepted → partially_filled` | execute | VenuePort adapter | Venue fills partially |
| `accepted → cancelled` | execute | VenuePort adapter | Venue cancels (or operator request) |
| `partially_filled → filled` | execute | VenuePort adapter | Remaining quantity filled |
| `partially_filled → cancelled` | execute | VenuePort adapter | Cancelled with partial standing |

**Boundary rule:** The derive binary only produces `submitted`. All subsequent transitions belong to the execute binary through the VenuePort adapter.

---

## 4. Terminal State Semantics

### 4.1 Filled (Terminal — Success)

| Property | Value |
|----------|-------|
| Meaning | Order fully executed at venue |
| `FilledQuantity` | Equals `Quantity` |
| `Fills` | One or more `FillRecord` entries |
| `Final` | `true` |
| Downstream | Materialised to KV + ClickHouse; queryable in composite read model |
| Retry | Never — success is permanent |

### 4.2 Rejected (Terminal — Failure)

| Property | Value |
|----------|-------|
| Meaning | Venue refused the order (bad parameters, insufficient balance, expired) |
| `FilledQuantity` | `"0"` or empty |
| `Fills` | Empty |
| `Final` | `true` |
| Downstream | Materialised with rejected status; visible in explainability surface |
| Retry | Not automatic — requires new intent from derive cycle |
| Mapped from | Binance `REJECTED`, `EXPIRED` |

### 4.3 Cancelled (Terminal — Aborted)

| Property | Value |
|----------|-------|
| Meaning | Order cancelled before full execution |
| `FilledQuantity` | May be `> 0` if partial fill occurred before cancel |
| `Fills` | Zero or more — partial fills are preserved |
| `Final` | `true` |
| Downstream | Materialised; partial fills visible in analytics |
| Retry | Not automatic |
| Mapped from | Binance `CANCELED`, `CANCELLED` |

### 4.4 Terminal State Invariants

| Rule | Statement |
|------|-----------|
| TERM-1 | Terminal states are absorbing — `ValidTransition(terminal, any)` returns `false` |
| TERM-2 | `Final` must be `true` for all terminal-state intents |
| TERM-3 | A terminated intent's `Fills` array is frozen — no further appends |
| TERM-4 | A terminated intent's `FilledQuantity` is frozen — no further updates |
| TERM-5 | Terminal intents are still materialised — rejection/cancellation is observable data |

---

## 5. Non-Terminal State Semantics

### 5.1 Submitted (Initial)

| Property | Value |
|----------|-------|
| Meaning | Intent produced by evaluator; not yet dispatched |
| Residence | Derive binary → NATS EXECUTION_EVENTS stream |
| Duration (paper) | Instant — immediately transitions to `filled` |
| Duration (venue) | Until execute binary processes and dispatches |
| Safety gates | Checked before transition to `sent`/`accepted` |

### 5.2 Sent (In-Flight)

| Property | Value |
|----------|-------|
| Meaning | HTTP request dispatched to venue; response pending |
| Residence | Execute binary actor state |
| Duration | Bounded by per-request context deadline |
| Failure mode | Timeout → retryable error; network error → retryable |
| Note | Optional for synchronous adapters; Binance testnet typically skips this |

### 5.3 Accepted (Acknowledged)

| Property | Value |
|----------|-------|
| Meaning | Venue acknowledged order; execution in progress |
| Residence | Execute binary; venue has the order |
| Possible exits | `filled`, `partially_filled`, `cancelled` |
| Binance mapping | `NEW` → `accepted` |

### 5.4 Partially Filled (Intermediate)

| Property | Value |
|----------|-------|
| Meaning | Some quantity executed; remainder pending |
| `FilledQuantity` | `> 0` and `< Quantity` |
| `Fills` | One or more records; total quantity matches `FilledQuantity` |
| Possible exits | `filled` (remaining quantity fills), `cancelled` (remainder abandoned) |
| Note | Rare for synchronous market orders; more common with limit orders (post-S312) |

---

## 6. Boundary Conditions

### 6.1 Paper-to-Venue Boundary

The execution domain model is **shared** between paper and venue modes. Discrimination happens through:

| Discriminator | Paper | Venue |
|--------------|-------|-------|
| `FillRecord.Simulated` | `true` | `false` |
| `FillRecord.Price` | `"0"` | Real venue price |
| `FillRecord.Fee` | `"0"` | Real venue fee |
| `VenueOrderReceipt.VenueOrderID` | `paper-{hex}` | Binance `orderId` |
| State traversal | `submitted → filled` (instant) | `submitted → accepted → filled` (typical) |
| Fill source | `PaperFillSimulator` | `BinanceFuturesTestnetAdapter` |

**Rule:** The composite read model, analytics pipeline, and HTTP surfaces must work identically regardless of `Simulated` flag value. The flag is metadata for observability, not for branching logic.

### 6.2 Derive-Execute Boundary

| Aspect | Derive Binary | Execute Binary |
|--------|--------------|----------------|
| Produces | `ExecutionIntent` with `submitted` | Fill events with `filled`/`rejected`/`cancelled` |
| Consumes | Risk events | Execution events |
| State authority | Only `submitted` | All post-submitted states |
| Venue awareness | None — does not know about venue | Full — owns venue adapter lifecycle |
| Safety gates | None — not responsible | Full — enforces kill switch + staleness |

### 6.3 Execute-Store Boundary

| Aspect | Execute Binary | Store Binary |
|--------|---------------|-------------|
| Produces | Fill events on NATS stream | Materialised KV entries, ClickHouse rows |
| Consumes | Execution events | Fill events |
| Validates | Domain rules, venue response | Schema conformance only |
| Authority | Transition ownership | Persistence authority |

### 6.4 Kill Switch Boundary

| Aspect | Rule |
|--------|------|
| Scope | Global — single `ControlGate` for all symbols/families |
| Check point | Execute binary actor layer, before VenuePort call |
| Not checked by | VenuePort adapter (by design — adapter is pure) |
| Fail-open | If gate checker unavailable, kill switch is skipped (SafetyGate design) |
| Halted → active | Does not replay blocked intents — next derive cycle produces new ones |

---

## 7. State Consistency Rules

### 7.1 Intent-Fill Consistency

| Rule | Assertion |
|------|-----------|
| IFC-1 | `status == filled` ⇒ `len(Fills) ≥ 1` |
| IFC-2 | `status == filled` ⇒ `FilledQuantity == Quantity` |
| IFC-3 | `status == partially_filled` ⇒ `len(Fills) ≥ 1 AND FilledQuantity < Quantity` |
| IFC-4 | `status == rejected` ⇒ `len(Fills) == 0` |
| IFC-5 | `status == cancelled AND FilledQuantity > 0` ⇒ `len(Fills) ≥ 1` (partial fill preserved) |
| IFC-6 | `status == cancelled AND FilledQuantity == 0` ⇒ `len(Fills) == 0` |
| IFC-7 | `status == submitted` ⇒ `len(Fills) == 0 AND FilledQuantity == "" OR FilledQuantity == "0"` |

### 7.2 Quantity Monotonicity

| Rule | Assertion |
|------|-----------|
| QM-1 | For any sequence of updates to the same intent: `FilledQuantity[t] ≥ FilledQuantity[t-1]` |
| QM-2 | `FilledQuantity` never exceeds `Quantity` |
| QM-3 | Fill records are append-only — existing records are never modified or removed |

### 7.3 Status Monotonicity

| Rule | Assertion |
|------|-----------|
| SM-1 | Status only moves forward through `validTransitions` — never backward |
| SM-2 | `submitted` is the only valid initial state |
| SM-3 | Each intent reaches exactly one terminal state (or remains in-flight) |
| SM-4 | `Final` flag must be `true` if and only if status is terminal |

---

## 8. Venue Adapter Mapping Reference

### 8.1 Binance Futures Status Mapping

| Binance Status | Domain Status | Notes |
|---------------|---------------|-------|
| `NEW` | `accepted` | Standard acknowledgement |
| `FILLED` | `filled` | Complete execution |
| `PARTIALLY_FILLED` | `partially_filled` | Partial execution |
| `CANCELED` | `cancelled` | Binance US spelling |
| `CANCELLED` | `cancelled` | Alternate spelling (defensive) |
| `REJECTED` | `rejected` | Venue rejection |
| `EXPIRED` | `rejected` | Treated as rejection (not a separate domain state) |
| Any other | Error (`problem.Internal`) | Unknown status is never silently mapped |

### 8.2 Binance Error HTTP Mapping

| HTTP Status | Problem Category | Retryable | Domain Outcome |
|------------|-----------------|-----------|---------------|
| 200 | — | — | Parse response; map status |
| 400 | `InvalidArgument` | No | Intent effectively rejected |
| 401, 403 | `InvalidArgument` | No | Credential issue; operator intervention needed |
| 429 | `Unavailable` | Yes | Rate limited; back off |
| 500–599 | `Unavailable` | Yes | Server error; retry eligible |
| Timeout | `Unavailable` | Yes | Network timeout |
| Connection error | `Unavailable` | Yes | DNS, TCP, TLS failure |

---

## 9. Guard Rails for S309+ Implementation

When implementing the execute binary's venue adapter actor path, the following guard rails apply:

| Guard Rail | Rule |
|-----------|------|
| GR-1 | Safety gate check must occur **before** every VenuePort call — no exceptions |
| GR-2 | `ValidTransition()` must be called before mutating intent status — reject invalid transitions |
| GR-3 | Fill records from venue must be constructed **only** from venue response data — no synthetic values |
| GR-4 | `Simulated` flag must be `false` for all venue fills — never `true` |
| GR-5 | `VenueOrderID` must be persisted with the intent for future reconciliation |
| GR-6 | Context deadlines must be set on all venue HTTP calls — no unbounded waits |
| GR-7 | Credentials must be loaded once at adapter construction — not per-request |
| GR-8 | Actor must not block on venue response beyond context deadline |
| GR-9 | Failed venue calls must not leave intent in an intermediate state — revert to last valid state |
| GR-10 | One intent = one venue submission attempt (until retry infrastructure in S310) |

---

## 10. Future Evolution Path

| Capability | Stage | Impact on State Model |
|-----------|-------|----------------------|
| Client order ID | S307 | New field on VenueOrderRequest; no state model change |
| E2E integration proof | S309 | Exercises full state traversal; no model change |
| Failure containment | S310 | Retry semantics; may introduce `sent → sent` retry loop |
| Multi-symbol isolation | S311 | No state model change; per-symbol concurrency proof |
| Async fills (WebSocket) | Post-S312 | May introduce `pending_cancel` state; `sent` becomes mandatory |
| Limit orders | Post-S312 | `partially_filled` becomes common; may add `amended` state |
| Multi-venue | Post-S312 | Venue ID on fill records; adapter registry pattern |
| OMS | Post-S312 | Position tracking; order book; significant state model extension |

---

*Created: 2026-03-21 — Stage S308*
