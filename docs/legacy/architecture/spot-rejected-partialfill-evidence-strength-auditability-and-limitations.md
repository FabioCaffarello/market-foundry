# Spot Rejected/PartialFill Evidence Strength, Auditability, and Limitations

Stage: **S406**
Status: **complete**
Predecessor: S405 (fill evidence strength), S386 (rejection event auditability)

## 1. Purpose

This document provides a methodologically honest assessment of the evidence strength for rejection and partial fill lifecycle paths on Binance Spot testnet. It distinguishes what was directly proven from what was structurally inferred, and identifies the limits of the testnet environment.

## 2. Evidence Classification

### 2.1 Rejection Evidence

| Claim | Evidence Type | Strength | Rationale |
|---|---|---|---|
| Spot adapter classifies HTTP 400/-2010 as InvalidArgument | Mock HTTP + real adapter code | **Strong** | Same handleErrorResponse/classifyByVenueErrorCode code runs against both mock and real Spot responses. The mock replicates exact Binance Spot error payload structure. |
| Spot adapter classifies HTTP 401/-2015 as non-retryable auth | Mock HTTP + real adapter code | **Strong** | HTTP status code branching is deterministic. |
| Spot adapter classifies HTTP 429 as retryable | Mock HTTP + real adapter code | **Strong** | |
| Spot adapter maps REJECTED/EXPIRED status to StatusRejected | Mock HTTP + mapBinanceStatus | **Strong** | mapBinanceStatus is shared between Spot and Futures, tested in both. |
| VenueOrderRejectedEvent carries venue_http_status, venue_error_code | Event construction from Problem.Details | **Strong** | Details flow through Problem -> event; no transformation. |
| Rejection event correlation chain preserved | Event construction test | **Strong** | Metadata inheritance is deterministic. |
| VenueAdapterActor publishes rejection for ALL prob != nil | Code review + S386 tests | **Strong** | Actor code has single path: prob != nil -> publishRejection. |
| Real Spot testnet returns HTTP 400/-2010 for insufficient balance | **Not directly tested** | **Inferred** | Binance Spot testnet API documentation confirms this behavior. Cannot prove without live account balance manipulation. |

### 2.2 Partial Fill Evidence

| Claim | Evidence Type | Strength | Rationale |
|---|---|---|---|
| Adapter parses PARTIALLY_FILLED to StatusPartiallyFilled | Mock HTTP + mapBinanceStatus | **Strong** | mapBinanceStatus switch case is deterministic. |
| Fill record fidelity under partial fill | Mock HTTP response parsing | **Strong** | Same parseOrderResponse code handles both partial and full fills. |
| Multi-leg aggregation under partial fill | Mock HTTP + computeSpotFillAggregates | **Strong** | Arithmetic is deterministic; weighted average formula tested with exact values. |
| Quantity monotonicity (FilledQuantity <= Quantity) | Structural test, 3 scenarios | **Strong** | Adapter sets FilledQuantity = venue executedQty, preserves original Quantity. No corruption path exists. |
| Fill timestamp from venue transactTime | Mock HTTP with known timestamp | **Strong** | Conditional: if TransactTime > 0, use UnixMilli. Deterministic. |
| Lifecycle: partially_filled -> filled valid | Domain state machine test | **Strong** | ValidTransition is a static map lookup. |
| Lifecycle: partially_filled is not terminal | Domain test | **Strong** | IsTerminal checks against fixed set {filled, rejected, cancelled}. |
| Binance Spot testnet returns PARTIALLY_FILLED for market orders | **Not directly observed** | **Weak/Absent** | Market orders on Spot testnet are typically filled instantly. PARTIALLY_FILLED is documented in Binance API but observed primarily with limit orders or during extreme liquidity events. |
| Quantity monotonicity across successive partial fill updates | **Not tested** (single response) | **Structural** | Each adapter call is stateless — monotonicity across calls would require actor-level state tracking (future S407+ scope). |

## 3. Auditability Assessment

### 3.1 Rejection Audit Trail Completeness

A complete rejection audit trail requires:

| Field | Source | Present | Notes |
|---|---|---|---|
| RejectionCode | Problem.Code | Yes | Maps to VAL_INVALID_ARGUMENT or SYS_UNAVAILABLE |
| RejectionReason | Problem.Message | Yes | Human-readable, includes HTTP status and venue code |
| venue_http_status | Problem.Details | Yes | From handleErrorResponse |
| venue_error_code | Problem.Details | Yes | From Binance error response body, when available |
| venue_error_class | Problem.Details | Conditional | Only for override codes (-1001, -1003, -1015) |
| CorrelationID | Event.Metadata | Yes | Inherited from incoming event |
| CausationID | Event.Metadata | Yes | Set to incoming event ID |
| Intent.Source | Event.ExecutionIntent | Yes | Preserved from original intent |
| Intent.Symbol | Event.ExecutionIntent | Yes | Preserved from original intent |
| Intent.Timestamp | Event.ExecutionIntent | Yes | Original intent timestamp, NOT rejection time |
| Event.OccurredAt | Event.Metadata | Yes | Rejection event creation time |

**Assessment**: The rejection audit trail is **complete at the event level**. The gap is in persistence and queryability — events are published to NATS but not yet materialized to ClickHouse or queryable via gateway. This is explicitly S407 scope.

### 3.2 Partial Fill Audit Trail

| Field | Source | Present | Notes |
|---|---|---|---|
| Status | Receipt.Status | Yes | StatusPartiallyFilled from adapter |
| FilledQuantity | Receipt.Intent.FilledQuantity | Yes | From venue executedQty |
| Fills[].Price | Receipt.Intent.Fills | Yes | Weighted average from fills[] |
| Fills[].Quantity | Receipt.Intent.Fills | Yes | Total executed qty |
| Fills[].Fee | Receipt.Intent.Fills | Yes | Total commission from fills[] |
| Fills[].Simulated | Receipt.Intent.Fills | Yes | false for real venue |
| Fills[].Timestamp | Receipt.Intent.Fills | Yes | From venue transactTime |
| CorrelationID | Receipt.Intent | Yes | Preserved through router |

**Assessment**: Partial fill audit trail is complete for the write path. The same VenueOrderFilledEvent is published for partial and full fills — downstream consumers receive identical event structure.

## 4. Environment Limitations

### 4.1 Testnet Behavioral Constraints

1. **Market orders fill instantly on Spot testnet**: The testnet has no real order book depth constraints. Market orders are filled at a reference price with no partial fills. This makes PARTIALLY_FILLED unreproducible via live testnet interaction for market orders.

2. **Balance manipulation required for rejection**: To trigger a real -2010 (insufficient balance) rejection, the testnet account must have insufficient quote asset. This requires deliberate balance drain, which is not idempotent and creates test environment coupling.

3. **Rate limits are generous on testnet**: HTTP 429 rejections are difficult to reproduce organically on testnet.

### 4.2 Structural vs. Live Evidence

| Dimension | Live testnet evidence (S405) | Structural evidence (S406) |
|---|---|---|
| Fill path | Directly proven | Regression confirmed |
| Rejection path | Not proven live | Error classification proven via mock + shared code |
| Partial fill path | Not observed live | Parser + lifecycle proven via mock + shared code |

The key insight: the adapter code that handles mock responses is **identical** to the code that handles live responses. The HTTP response parsing, status mapping, error classification, and fill aggregation are deterministic functions of the response payload. Mock responses replicate exact Binance Spot JSON structure.

### 4.3 What Would Strengthen Evidence

1. **Live rejection test**: Drain testnet account balance, submit market buy, capture actual -2010 response. Risk: account state management complexity, non-idempotent.

2. **Limit order partial fill**: Submit limit order with price at market boundary, observe PARTIALLY_FILLED. Risk: requires market timing, non-deterministic.

3. **WebSocket fill stream**: Observe partial fill progression via user data stream. Scope: beyond S406 (requires WebSocket ingress).

## 5. Methodological Honesty Summary

### What S406 proves:
- The Spot adapter correctly classifies all known Binance Spot error codes to the appropriate Problem code and retryability.
- The rejection event construction path carries complete venue details for audit.
- The lifecycle state machine correctly handles rejected and partially_filled states.
- Multi-leg fill aggregation is correct under both partial and full fills.
- Quantity monotonicity holds structurally at the adapter level.
- Segment isolation is preserved for rejection and partial fill paths.

### What S406 does NOT prove:
- That a real Spot testnet call has returned PARTIALLY_FILLED for market orders.
- That a specific testnet account state triggers -2010 in practice (though the Binance API guarantees it).
- Cross-call quantity monotonicity (multiple successive partial fill updates for the same order).
- Read-path queryability of rejection/partial fill events from store or gateway.

### Why this is sufficient for S406 scope:
The adapter code is a deterministic function of HTTP response payloads. Proving correct behavior for all known payload shapes is equivalent to proving correct behavior for all live responses that match those shapes. The Binance Spot API specification guarantees the response structure.

The remaining gaps (live rejection, live partial fill, read-path queryability) are appropriately scoped for S407+.
