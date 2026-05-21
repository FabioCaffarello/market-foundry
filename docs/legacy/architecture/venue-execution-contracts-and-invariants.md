# Venue Execution Contracts and Invariants

> Stage S308 — Design artefact.
> Defines the minimum contracts and invariants required for venue execution,
> bridging the gap between paper execution and real venue order lifecycle.

---

## 1. Scope

This document formalises the **contracts** (interfaces, data shapes, behavioural expectations) and **invariants** (rules that must hold across all transitions) for venue execution within market-foundry. It intentionally stops short of OMS design, multi-venue routing, or async fill handling — those belong to future stages.

Target venue: **Binance Futures Testnet** (market orders, synchronous fills).

---

## 2. Contract Taxonomy

Contracts are grouped into five categories aligned with the execution lifecycle.

### 2.1 Submit Intent Contract (C-SUB)

**Owner:** Actor layer (execute binary).
**Consumer:** VenuePort implementation.

| Field | Source | Constraint |
|-------|--------|------------|
| `ExecutionIntent` | Derive binary output | Must pass `Validate()` — no zero fields |
| `Side` | Domain evaluator | `buy`, `sell`, or `none` |
| `Quantity` | Domain evaluator | Positive decimal string, non-empty |
| `Status` | Domain evaluator | Must be `submitted` at submit time |
| `CorrelationID` | Causal chain | Non-empty; traces back to originating observation |
| `CausationID` | Causal chain | Non-empty; identifies the immediate cause (risk event) |
| `Timestamp` | Evaluation time | Non-zero; used for staleness check |

**Pre-conditions (enforced by actor layer before VenuePort call):**

| Gate | Contract | Failure Mode |
|------|----------|-------------|
| Kill switch | `ControlGate.Status == active` | Block; reason `kill_switch` |
| Staleness guard | `now - intent.Timestamp ≤ maxAge` | Block; reason `stale` |
| Validation | `intent.Validate() == nil` | Block; reason `invalid_intent` |
| Side filter | `intent.Side != none` → submit to venue | No-action intents never reach venue API |

**Post-condition:** If all gates pass and side is actionable, exactly one `VenuePort.SubmitOrder` call is made.

### 2.2 Venue Submission Contract (C-VEN)

**Owner:** VenuePort implementation (adapter).
**Consumer:** Actor layer.

```go
type VenuePort interface {
    SubmitOrder(ctx context.Context, req VenueOrderRequest) (VenueOrderReceipt, *problem.Problem)
}
```

**Request invariants:**

| Invariant | Rule |
|-----------|------|
| V-REQ-1 | Request carries the complete `ExecutionIntent` — adapter must not mutate upstream fields |
| V-REQ-2 | Context must carry a deadline (per-request timeout) |
| V-REQ-3 | No-action intents (`Side == none`) return immediately without venue API call |
| V-REQ-4 | Adapter must sign requests using credentials loaded at construction — never at call time |
| V-REQ-5 | Adapter must not log, serialize, or embed credentials in error messages |

**Response invariants:**

| Invariant | Rule |
|-----------|------|
| V-RES-1 | `VenueOrderReceipt.VenueOrderID` is non-empty for all successful submissions |
| V-RES-2 | `VenueOrderReceipt.Status` is a valid `domain/execution.Status` |
| V-RES-3 | `VenueOrderReceipt.Intent` contains the enriched intent with fills populated |
| V-RES-4 | On error, a `*problem.Problem` is returned with correct category and retryable flag |
| V-RES-5 | Response body is capped at 64 KB (`io.LimitReader`) |

### 2.3 Fill Record Contract (C-FILL)

**Owner:** VenuePort implementation.
**Consumer:** Store binary, analytical pipeline, composite read model.

| Field | Paper Value | Venue Value | Constraint |
|-------|------------|-------------|------------|
| `Price` | `"0"` | Venue `avgPrice` | Non-empty decimal string |
| `Quantity` | Intent quantity | Venue `executedQty` | Non-empty decimal string; `≤ intent.Quantity` |
| `Fee` | `"0"` | Venue `cumQuote` | Non-empty decimal string; `≥ 0` |
| `Simulated` | `true` | `false` | Discriminator for paper vs. real fills |
| `Timestamp` | Evaluation time | Venue `updateTime` | Non-zero; monotonically ≥ intent timestamp |

**Fill consistency rules (CR-1 through CR-5, carried from S77):**

| Rule | Statement |
|------|-----------|
| CR-1 | `sum(fill.Quantity) ≤ intent.Quantity` — cannot overfill |
| CR-2 | If `status == filled`, then `sum(fill.Quantity) == intent.FilledQuantity` |
| CR-3 | If `status == partially_filled`, then `0 < sum(fill.Quantity) < intent.Quantity` |
| CR-4 | `fill.Timestamp ≥ intent.Timestamp` — fills cannot precede intent creation |
| CR-5 | All fills within a single intent share the same `Simulated` value |

### 2.4 Acceptance / Rejection Contract (C-ACK)

**Owner:** Venue (external) → mapped by adapter.
**Consumer:** Execution domain, store, analytics.

| Venue Status | Domain Status | Terminal | Semantics |
|-------------|---------------|----------|-----------|
| `NEW` | `accepted` | No | Venue acknowledged; awaiting fill |
| `FILLED` | `filled` | Yes | Complete execution |
| `PARTIALLY_FILLED` | `partially_filled` | No | Partial execution; may reach `filled` or `cancelled` |
| `CANCELED` / `CANCELLED` | `cancelled` | Yes | Venue cancelled (user or system) |
| `REJECTED` | `rejected` | Yes | Venue refused the order |
| `EXPIRED` | `rejected` | Yes | Venue expired the order (treated as rejection) |
| Unknown | Error | — | `problem.Internal`; never silently mapped |

### 2.5 Terminal Failure Contract (C-FAIL)

**Owner:** VenuePort implementation.
**Consumer:** Actor layer, observability.

| Failure Class | Problem Category | Retryable | Examples |
|--------------|-----------------|-----------|----------|
| Authentication | `InvalidArgument` | No | HTTP 401, 403 |
| Client error | `InvalidArgument` | No | HTTP 400, 422 (bad quantity, symbol) |
| Rate limit | `Unavailable` | Yes | HTTP 429 |
| Venue unavailable | `Unavailable` | Yes | HTTP 503, connection timeout |
| Server error | `Unavailable` | Yes | HTTP 5xx (except 503) |
| Network failure | `Unavailable` | Yes | DNS, TCP, TLS errors |
| Parse failure | `Internal` | No | Malformed JSON, unknown status |
| Unknown | `Internal` | No | Catch-all for unmapped conditions |

**Invariants:**

| Invariant | Rule |
|-----------|------|
| F-1 | Every error path returns a `*problem.Problem` — no bare Go errors escape the adapter |
| F-2 | Retryable flag is set on all transient failures (rate limit, network, 5xx) |
| F-3 | Non-retryable errors must never be retried by the actor layer |
| F-4 | Error messages never contain credentials, API keys, or secrets |
| F-5 | Adapter failures do not corrupt the `ExecutionIntent` — the original intent is preserved |

---

## 3. Cross-Cutting Invariants

These invariants apply across all contracts and all execution paths.

### 3.1 Idempotency (INV-IDEM)

| Layer | Mechanism | Guarantee |
|-------|-----------|-----------|
| NATS JetStream | `DeduplicationKey()` = `exec:{type}:{source}:{symbol}:{timeframe}:{unix}` | Same intent published twice → stored once |
| KV Store | `PartitionKey()` = `{source}.{symbol}.{timeframe}` | Last-writer-wins per partition; revision monotonicity |
| Venue (future) | Client order ID (`newClientOrderId`) | Same order submitted twice → venue deduplicates (S307 blocker EC-1) |

**Rules:**

| Rule | Statement |
|------|-----------|
| IDEM-1 | The actor layer must not submit the same intent to the venue more than once unless the first attempt returned a retryable error |
| IDEM-2 | JetStream dedup window must exceed the maximum latency of a venue round-trip |
| IDEM-3 | Client order ID (when implemented) must be deterministically derived from `DeduplicationKey()` |

### 3.2 State Monotonicity (INV-MONO)

| Rule | Statement |
|------|-----------|
| MONO-1 | Status transitions must follow the `validTransitions` map — no backward or lateral moves |
| MONO-2 | Terminal states (`filled`, `rejected`, `cancelled`) are absorbing — no further transitions |
| MONO-3 | `FilledQuantity` can only increase or remain unchanged — never decrease |
| MONO-4 | `len(Fills)` can only increase — fill records are append-only |
| MONO-5 | `Final` flag, once set to `true`, must never revert to `false` |

### 3.3 Correlation / Causation Preservation (INV-TRACE)

| Rule | Statement |
|------|-----------|
| TRACE-1 | `CorrelationID` must propagate unchanged from observation through execution fill |
| TRACE-2 | `CausationID` on execution intent must reference the risk event that caused it |
| TRACE-3 | Venue fill events must carry the same `CorrelationID` as the originating intent |
| TRACE-4 | `CausationID` on a fill event must reference the execution intent that submitted it |
| TRACE-5 | No execution event may exist without both `CorrelationID` and `CausationID` populated |

### 3.4 Ownership of Transitions (INV-OWN)

| Transition | Owner | Authority |
|-----------|-------|-----------|
| `→ submitted` | Derive binary (evaluator) | Domain evaluator creates intent with `submitted` |
| `submitted → sent` | Execute binary (actor) | Actor marks `sent` after dispatching to venue API |
| `sent → accepted` | Venue adapter | Adapter maps venue `NEW` to `accepted` |
| `submitted → accepted` | Venue adapter | Shortcut when venue responds synchronously |
| `accepted → filled` | Venue adapter | Adapter maps venue `FILLED` |
| `accepted → partially_filled` | Venue adapter | Adapter maps venue `PARTIALLY_FILLED` |
| `accepted → cancelled` | Venue adapter | Adapter maps venue `CANCELED` |
| `partially_filled → filled` | Venue adapter | Subsequent fill completes the order |
| `partially_filled → cancelled` | Venue adapter | Order cancelled with partial fill standing |
| `submitted → rejected` | Venue adapter | Adapter maps venue `REJECTED` or `EXPIRED` |
| `sent → rejected` | Venue adapter | Venue rejects after acknowledgement |
| Kill switch halt | Operator (via gateway) | Sets `ControlGate.Status = halted` |
| Kill switch resume | Operator (via gateway) | Sets `ControlGate.Status = active` |

### 3.5 Temporal Invariants (INV-TIME)

| Rule | Statement |
|------|-----------|
| TIME-1 | `intent.Timestamp` reflects evaluation time — not submission or fill time |
| TIME-2 | `fill.Timestamp` reflects venue fill time (or `time.Now()` if venue omits it) |
| TIME-3 | `fill.Timestamp ≥ intent.Timestamp` — causality preserved |
| TIME-4 | Staleness guard operates on `intent.Timestamp` vs `now` — not on fill time |
| TIME-5 | `ControlGate.UpdatedAt` is set at mutation time — used for audit, not for logic |

---

## 4. Contract Verification Criteria

Each contract must be verifiable through unit tests or scenario tests without a live venue.

| Contract | Verification Method |
|----------|-------------------|
| C-SUB | Unit test: `Validate()` rejects invalid intents; SafetyGate blocks stale/halted |
| C-VEN | Unit test: mock HTTP server returns known responses; adapter maps correctly |
| C-FILL | Unit test: fill records satisfy CR-1 through CR-5 for all terminal states |
| C-ACK | Unit test: every Binance status string maps to expected domain status |
| C-FAIL | Unit test: every HTTP status code / error condition maps to correct problem category and retryable flag |
| INV-IDEM | Unit test: DeduplicationKey uniqueness; scenario test: duplicate publish → single store |
| INV-MONO | Unit test: ValidTransition rejects backward moves; FilledQuantity never decreases |
| INV-TRACE | Unit test: CorrelationID/CausationID propagation through mock pipeline |
| INV-OWN | Scenario test: each transition triggered by correct binary/layer |
| INV-TIME | Unit test: fill timestamp ≥ intent timestamp; staleness uses intent timestamp |

---

## 5. Boundary Summary

| Layer | Responsibility | Does NOT Do |
|-------|---------------|-------------|
| Derive binary | Creates `ExecutionIntent` with `submitted` status | Does not call venue; does not know about fills |
| Actor layer (execute) | Safety gates → VenuePort dispatch → publish fill event | Does not evaluate risk; does not mutate intent beyond status/fills |
| VenuePort adapter | HTTP call → response parsing → status mapping → fill construction | Does not check kill switch; does not persist anything |
| Store binary | Materialises fill events to KV + ClickHouse | Does not validate transitions (trusts upstream) |
| Gateway (read) | Serves composite read model via HTTP | Does not write; does not submit orders |

---

## 6. Residual Gaps (Deferred Beyond S308)

| Gap | Deferred To | Reason |
|-----|------------|--------|
| Client order ID (venue-side idempotency) | S307 (EC-1) | Requires adapter implementation change |
| Retry loop with backoff | S310 | Failure envelope stage |
| Per-symbol kill switch | Post-S312 | Global gate sufficient for testnet |
| Partial fill aggregation across multiple events | Post-S312 | Synchronous market orders produce single fill |
| Position tracking / net exposure | Post-S312 | Not OMS |
| Async fill via WebSocket | Post-S312 | Non-goal (NG-5) |
| Multi-venue routing | Post-S312 | Non-goal (NG-3) |

---

## 7. Alignment With S307 Blockers

| S307 Blocker | S308 Contract Addressed |
|-------------|------------------------|
| EC-1 (client order ID) | INV-IDEM: IDEM-3 defines derivation rule |
| VA-2 (EXPIRED mapping) | C-ACK: EXPIRED → rejected |
| VA-3 (CANCELED mapping) | C-ACK: CANCELED → cancelled |
| VA-4 (REJECTED mapping) | C-ACK: REJECTED → rejected |
| FM-1 (real price) | C-FILL: Price = avgPrice |
| FM-2 (real quantity) | C-FILL: Quantity = executedQty |
| FM-3 (real fee) | C-FILL: Fee = cumQuote |
| FM-4 (real timestamp) | C-FILL: Timestamp = updateTime |
| FM-5 (simulated flag) | C-FILL: Simulated = false |

---

*Created: 2026-03-21 — Stage S308*
