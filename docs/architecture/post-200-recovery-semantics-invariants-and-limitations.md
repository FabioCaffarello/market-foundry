# Post-200 Recovery Semantics, Invariants, and Limitations

> **Stage:** S322
> **Scope:** Semantic contract for post-200 body-read-failure reconciliation

## 1. Semantic Model

### 1.1 States

When SubmitOrder receives HTTP 200 but fails to read the body, the system enters an **ambiguous state**: the order is accepted at the venue but its status is unknown locally.

```
[Submit OK, Body Lost]
    │
    ├── QueryOrder succeeds → [Recovered: status + fills known]
    │
    └── QueryOrder fails → [Ambiguous: order exists, status unknown]
```

### 1.2 Intermediate State: "Accepted-But-Unconfirmed"

Between the body-read failure and the recovery query completing, the order is in an implicit "accepted-but-unconfirmed" state. This state is **not persisted** — it exists only within the Post200Reconciler's SubmitOrder call.

If recovery succeeds, the caller receives a normal VenueOrderReceipt as if the original submit had succeeded. The recovery is transparent.

If recovery fails, the caller receives the original error enriched with reconciliation metadata. The caller (actor layer) is responsible for logging and/or escalating the ambiguous state.

## 2. Invariants

### INV-REC-1: No Duplicate Execution

The reconciler NEVER calls SubmitOrder again. Recovery is exclusively via QueryOrder (HTTP GET). The deterministic client order ID ensures the query finds the exact order that was submitted.

### INV-REC-2: Client Order ID Stability

The client order ID used for recovery is the same one computed by `ClientOrderID(intent)` and sent with the original submit. This is guaranteed by EC-1 (deterministic SHA-256 derivation).

### INV-REC-3: Intent Preservation

The recovered receipt carries the **original intent** with updated status and fills. The intent's source, symbol, side, quantity, risk input, and metadata are never modified by the reconciler.

### INV-REC-4: Context Independence

The recovery query uses a fresh context with its own deadline (`queryTimeout`). This ensures:
- The expired submit context does not block recovery.
- The recovery has a bounded execution time.
- The recovery does not inherit cancellation signals from the submit.

### INV-REC-5: Error Enrichment

When recovery fails, the original Problem is enriched with structured metadata:
- `reconciliation_attempted: true`
- `reconciliation_failed: true`
- `reconciliation_error: <message>`

The original `body_read_failure_after_200` and `client_order_id` markers are preserved.

### INV-REC-6: Passthrough for Non-Target Errors

Errors that are NOT body-read-failure-after-200 pass through the reconciler unchanged. The reconciler does not inspect, modify, or delay any other error type.

## 3. Composition Semantics

### 3.1 With RetrySubmitter

The Post200Reconciler wraps the RetrySubmitter. This means:
1. Retryable errors (503, 429, network) are retried by the RetrySubmitter.
2. If a retry attempt results in body-read-failure-after-200, the RetrySubmitter returns it as non-retryable.
3. The Post200Reconciler intercepts it and attempts recovery.

This is the correct composition order because body-read-failure-after-200 is non-retryable — the RetrySubmitter correctly stops retrying, and the reconciler handles recovery.

### 3.2 Ordering

```
Post200Reconciler.SubmitOrder(req)
  → RetrySubmitter.SubmitOrder(req)     [may retry N times]
    → Adapter.SubmitOrder(req)          [HTTP POST]
  ← body-read-failure-after-200
  → Adapter.QueryOrder(clientOrderID)   [HTTP GET, fresh context]
  ← recovered receipt
← receipt to caller
```

## 4. Limitations

### L-1: Single Recovery Attempt

The reconciler makes exactly one QueryOrder attempt. If it fails (network error, venue unavailable, order not yet visible), the original error is returned. There is no retry loop for recovery.

**Rationale**: The recovery path itself could suffer the same transient failures. Adding retry to recovery adds complexity without proportional benefit at this stage. The caller can implement retry-of-reconciliation if needed.

### L-2: Race Condition — Order Not Yet Visible

In theory, the venue could accept the order (HTTP 200) but not yet make it queryable via the order API. If QueryOrder returns "order not found," this is treated as a recovery failure.

**Mitigation**: Binance Futures processes market orders synchronously — by the time HTTP 200 is returned, the order is queryable. This race condition is unlikely in practice but remains a theoretical limitation.

### L-3: No Persistence of Ambiguous State

If recovery fails, the ambiguous state ("order exists at venue, status unknown") is not persisted. It exists only as the enriched Problem returned to the caller. The actor layer is responsible for logging.

**Rationale**: Adding persistence would require OMS-level infrastructure, which is out of scope (S322 guard rail).

### L-4: Fills Recovery Depends on Query Response

The fills returned by QueryOrder may differ in structure from what SubmitOrder would have returned. In particular:
- `newOrderRespType=RESULT` (used by SubmitOrder) returns fills inline.
- `GET /fapi/v1/order` returns aggregate fill data (avgPrice, executedQty), not individual fills.

The reconciler uses the same `parseOrderResponse` for both paths, so the fill structure is consistent, but individual trade breakdowns are not available via the order query endpoint.

### L-5: Testnet Scope

This reconciliation mechanism is validated against the Binance Futures testnet adapter. Other venue adapters would need their own QueryOrder implementation.

## 5. Security

- QueryOrder uses the same credential handling as SubmitOrder (API key in header, HMAC signature).
- No credentials are logged or included in error messages (F-4 invariant preserved).
- The recovery query does not expose any additional attack surface beyond what SubmitOrder already uses.

## 6. Observability

Successful recovery is transparent — the caller receives a normal receipt. Failed recovery is observable through the enriched Problem details:

```json
{
  "code": "SYS_INTERNAL",
  "message": "read venue response failed",
  "details": {
    "body_read_failure_after_200": true,
    "client_order_id": "a1b2c3...",
    "reconciliation_attempted": true,
    "reconciliation_failed": true,
    "reconciliation_error": "venue rejected order (HTTP 400, code -2013): Order does not exist."
  }
}
```
