# Reconciliation for Body-Read-Failure-After-200

> **Stage:** S322
> **Predecessor:** S320 (Venue Failure Path Verification), S319 (Retry Infrastructure)
> **Scope:** Surgical reconciliation for the single case where HTTP 200 is received but the response body is lost

## 1. Problem Statement

When the venue adapter receives HTTP 200 headers (order accepted) but fails to read the response body (timeout, connection reset, truncation), the system loses the order status, venue order ID, and fill data. The order exists at the venue but the system has no record of it.

This was identified as **R-S320-1** (medium risk) in S320's residual gaps.

### Why This Is Dangerous

- The order is live at the venue — it may be filled, partially filled, or pending.
- Without the response body, the system cannot update its internal state.
- Retrying the submit would risk **duplicate execution** (the venue already accepted).
- Without reconciliation, the system has a "blind spot" — an order exists that it doesn't know about.

### Why This Is Not Retryable

S320/FP-11 established that body-read-failure-after-200 is classified as `Internal, non-retryable`. This is correct: once HTTP 200 headers arrive, the venue has accepted the order. Re-submitting would send a new order (or be rejected as a duplicate, depending on venue idempotency).

## 2. Reconciliation Mechanism

### 2.1 Detection

The adapter marks body-read-failure-after-200 with a distinguishing detail in the Problem:

```
body_read_failure_after_200: true
client_order_id: <deterministic ID from EC-1>
```

This allows downstream components to detect the specific case without string-matching error messages.

### 2.2 Recovery Path

```
SubmitOrder (POST)
    → HTTP 200 received
    → Body read fails
    → Problem returned with marker
    → Post200Reconciler detects marker
    → QueryOrder (GET /fapi/v1/order?origClientOrderId=...)
    → Recovers: venue order ID, status, fills
    → Returns VenueOrderReceipt with original intent + recovered data
```

### 2.3 Query Mechanism

`QueryOrder` uses the **deterministic client order ID** (EC-1, SHA-256 of intent's deduplication key) to query the venue for the order's current state. This is the same ID that was sent with the original submit, so it uniquely identifies the order.

Binance API: `GET /fapi/v1/order` with `origClientOrderId` parameter.

### 2.4 Context Independence

The recovery query uses a **fresh context** with its own deadline, independent of the original submit context (which may have expired). This ensures the recovery attempt is not blocked by the expired deadline that caused the body-read failure.

## 3. Design Invariants

| Invariant | Mechanism |
|-----------|-----------|
| No duplicate submit | Recovery uses GET, never POST |
| Same client order ID | Deterministic derivation (EC-1) |
| Independent deadline | Fresh context for query |
| Failure enrichment | Original error enriched with reconciliation metadata on query failure |
| Passthrough semantics | Non-body-read errors flow unchanged |
| Composability | Post200Reconciler implements VenuePort, composes with RetrySubmitter |

## 4. Architecture

### 4.1 Interfaces

```
VenuePort (existing)
├── SubmitOrder(ctx, req) → (receipt, problem)

VenueQueryPort (new, S322)
├── QueryOrder(ctx, clientOrderID, symbol) → (receipt, problem)
```

`VenueQueryPort` is intentionally separate from `VenuePort` to avoid forcing query capability on all adapters and to keep the submit path clean.

### 4.2 Composition Stack

```
Actor Layer
  └── Post200Reconciler (implements VenuePort)
        ├── inner submit: RetrySubmitter (implements VenuePort)
        │     └── BinanceFuturesTestnetAdapter (implements VenuePort + VenueQueryPort)
        └── inner query: BinanceFuturesTestnetAdapter (implements VenueQueryPort)
```

### 4.3 Decision: Reconciler vs. In-Adapter Recovery

The reconciliation is implemented as a **wrapper** rather than inside the adapter because:

1. The adapter should remain a thin HTTP client; reconciliation is a policy decision.
2. The wrapper pattern preserves testability — submit and query can be mocked independently.
3. The same adapter instance serves both VenuePort and VenueQueryPort roles.

## 5. Failure Modes During Recovery

| Recovery Failure | Behavior |
|-----------------|----------|
| Query returns HTTP error | Original error returned with `reconciliation_attempted`, `reconciliation_failed`, `reconciliation_error` details |
| Query body read fails | Same as above — query is a best-effort recovery |
| Query returns unknown status | Mapped to Internal error by existing classification |
| Query context expires | Timeout treated as query failure, original error returned |
| Order not yet visible at venue | Venue returns 400 (order not found), treated as query failure |

## 6. Non-Goals

- General OMS reconciliation
- Websocket/streaming real-time recovery
- Periodic polling or background reconciliation
- Multi-order batch reconciliation
- Retry of the query itself (single attempt)
