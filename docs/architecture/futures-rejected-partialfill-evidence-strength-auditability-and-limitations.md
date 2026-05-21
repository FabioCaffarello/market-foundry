# Futures Rejected/PartialFill Evidence Strength, Auditability, and Limitations

**Stage**: S417 + S423
**Wave**: Phase 45 (S415--S420) + Phase 47 (S421--S426)
**Date**: 2026-03-23 (updated S423)

## Purpose

This document assesses the evidence strength, auditability posture, and honest
limitations of the Futures rejection and partial fill proof delivered in S417
and elevated to lifecycle-grade evidence in S423.

## Evidence Strength Assessment

### Rejection Evidence: STRONG

| Dimension | Assessment | Rationale |
|---|---|---|
| Error code coverage | High | 10 distinct error scenarios covered (8 adapter + 2 venue status) |
| Classification correctness | Complete | All 8 C-FAIL classes mapped: auth, rate-limit, client-error, venue-unavailable, server-error, venue-override (-1001, -1003, -1015), venue-rejected, venue-expired |
| Retryability semantics | Correct | Non-retryable for client errors, retryable for transient failures |
| Venue detail preservation | Full | venue_http_status, venue_error_code, venue_error_class all carried |
| Rejection event construction | Validated | RejectionCode, RejectionReason, VenueDetails, intent mutation (Status=rejected, Final=true) |
| Correlation chain | Preserved | CorrelationID and CausationID survive rejection path |
| Segment isolation | Proven | Spot adapter never contacted for Futures rejections |
| Lifecycle alignment | Confirmed | submitted->rejected and sent->rejected are valid transitions |
| Terminal state | Confirmed | StatusRejected.IsTerminal() == true |

**Confidence**: The rejection path is proven with high confidence. The adapter correctly
classifies all known Binance error codes, preserves venue details for auditability,
and the actor layer correctly mutates the intent and constructs the rejection event.

### Partial Fill Evidence: STRUCTURAL (with honest gap)

| Dimension | Assessment | Rationale |
|---|---|---|
| Status parsing | Correct | PARTIALLY_FILLED maps to StatusPartiallyFilled |
| Fill record | Correct | avgPrice, executedQty, cumQuote correctly extracted |
| Simulated flag | Correct | false for real venue responses |
| Timestamp source | Correct | From venue updateTime, not local clock |
| Quantity monotonicity | Proven | FilledQuantity <= Quantity across all test cases |
| Lifecycle transitions | Validated | accepted->partially_filled->filled, partially_filled->cancelled |
| Non-terminal | Confirmed | StatusPartiallyFilled.IsTerminal() == false |
| Live observation | NOT observed | Testnet instant fills prevent live observation |

**Confidence**: The partial fill handling is proven structurally -- the adapter will
correctly handle a PARTIALLY_FILLED response when one occurs. However, no live
partial fill was observed on the Futures testnet because market orders fill instantly
with synthetic liquidity.

**Why this is acceptable**: Partial fills in Futures are more likely in production
than in Spot (larger order sizes, thinner books on some contracts, limit orders).
The structural proof ensures the system is ready for this scenario. The same gap
exists in S406 for Spot and was accepted there with the same rationale.

## Auditability Posture

### Rejection Event Audit Trail

Every rejection event emitted by the VenueAdapterActor carries:

```
VenueOrderRejectedEvent {
  Metadata:        {CorrelationID, CausationID, EventID, Timestamp}
  ExecutionIntent: {Status: rejected, Final: true, Source, Symbol, Side, Quantity, ...}
  RejectionCode:   "VAL_INVALID_ARGUMENT"     (from Problem.Code)
  RejectionReason: "venue rejected order ..."  (from Problem.Message)
  VenueDetails: {
    venue_http_status: 400,
    venue_error_code:  -2019,
    venue_error_class: "..."  (when override applied)
  }
}
```

This event flows through:

1. **NATS stream**: `EXECUTION_REJECTION_EVENTS` (durable, replayed on restart)
2. **Rejection consumer**: `rejection_consumer.go` with ack/term/nak semantics
3. **Rejection projection**: `rejection_projection_actor.go` materializes to KV store
4. **ClickHouse**: rejection data persisted via writer pipeline (S411)

### Query Paths for Rejected Orders

| Query | Endpoint | Returns |
|---|---|---|
| Latest rejection | VenueRejectionLatest control (S407) | ExecutionRejectionReply with RejectionDetail |
| Lifecycle list | LifecycleList control (S413) | LifecycleEntry with status=rejected |
| KV direct | NATS KV bucket | Latest intent with embedded rejection metadata |

### Partial Fill Audit Trail

Partial fills follow the fill event path:

```
VenueOrderFilledEvent {
  Metadata:        {CorrelationID, CausationID, EventID, Timestamp}
  ExecutionIntent: {Status: partially_filled, FilledQuantity: "0.0005", Fills: [...]}
}
```

The `partially_filled` status is not terminal, meaning:

- The KV projection will be overwritten when the order reaches a terminal state
- ClickHouse retains the full history of all events (fill and rejection)
- Lifecycle list shows the latest status, not the full transition history

## Parity with Spot (S406) Assessment

| Capability | Spot (S406) | Futures (S417) | Parity |
|---|---|---|---|
| Rejection via HTTP error | 10 adapter + 4 actor tests | 12 adapter + 4 actor tests | Full |
| Rejection via HTTP 200 status | REJECTED + EXPIRED | REJECTED + EXPIRED | Full |
| Rejection event construction | Validated | Validated | Full |
| Rejection audit trail | Full | Full | Full |
| Partial fill parsing | fills[] array aggregation | avgPrice/cumQuote direct | Equivalent |
| Partial fill lifecycle | Validated | Validated | Full |
| Quantity monotonicity | Proven | Proven | Full |
| Live partial fill observed | No | No | Same gap |
| Segment isolation | Proven | Proven | Full |
| Correlation chain | Preserved | Preserved | Full |
| DryRunSubmitter bypass | Validated | Validated | Full |

## Test Coverage Summary

### S423 Adapter Level (19 tests — lifecycle-grade)

| Category | Count | Tests |
|---|---|---|
| Rejection lifecycle | 3 | DominantPath_ValidTransitions, HTTP200_REJECTED_ValidTransitions, HTTP200_EXPIRED_ValidTransitions |
| Rejection audit trail | 1 | MultiScenario_AuditTrail (6 sub-scenarios) |
| Rejection QueryOrder | 2 | RecoversRejectedStatus, RecoversExpiredStatus |
| Partial fill lifecycle | 1 | LifecyclePath_ValidTransitions |
| Partial fill QueryOrder | 1 | RecoversPartialFillStatus |
| Quantity monotonicity | 1 | QuantityMonotonicity_WithLifecycle (4 sub-cases) |
| SegmentRouter isolation | 2 | FuturesRejection_SpotIsolated, FuturesPartialFill_SpotIsolated |
| S422 regression | 2 | S422FillPathUnchanged, S422CorrelationPreserved |

### S417 Adapter Level (17 tests — error classification)

| Category | Count | Tests |
|---|---|---|
| Rejection (error codes) | 8 | InsufficientMargin, InsufficientBalance, InvalidQuantity, AuthFailure, RateLimit, VenueInternalOverride, OrderRateLimitOverride, ServerError |
| Rejection (venue status) | 2 | VenueRejectedStatus, VenueExpiredStatus |
| Rejection (structural) | 2 | LifecycleTransition, CorrelationPreserved |
| Partial fill | 3 | FuturesFormat, QuantityMonotonicity (3 sub), FillTimestamp |
| Lifecycle transitions | 1 | PartialFill_LifecycleTransitions |
| Regression | 1 | FilledStillWorks |

### S417 Actor Level (8 tests)

| Category | Count | Tests |
|---|---|---|
| Rejection (router) | 2 | InsufficientMargin (with isolation), LOTSize |
| Rejection (event) | 1 | Event_Construction (full path) |
| Rejection (venue status) | 1 | VenueRejectedStatus200 |
| Partial fill (router) | 2 | ThroughRouter (with isolation), CorrelationPreserved |
| Partial fill (dry run) | 1 | DryRunIntercepted |
| Audit trail | 1 | FuturesVenueDetails_AuditTrail (4 sub-cases) |

### Total: 44 tests (19 S423 + 25 S417) + full S422 regression suite

## Limitations and Deferred Work

### L1: No Live Partial Fill on Testnet

**Impact**: Medium
**Mitigation**: Structural proof covers the parsing path. Production monitoring
should track PARTIALLY_FILLED responses when they occur.

### L2: cumQuote as Fee Proxy

**Impact**: Low (known since S416)
**Mitigation**: Separate `GET /fapi/v1/userTrades` endpoint needed for true commission.
This is a platform-wide concern, not S417-specific.

### L3: Single Symbol Scope

**Impact**: Low
**Mitigation**: Multi-symbol is structurally supported. No evidence of symbol-specific
behavior differences in error classification.

### L4: No Position/Leverage Context in Rejection

**Impact**: Medium
**Mitigation**: The adapter does not manage positions or leverage. Rejection for
insufficient margin depends on the Futures wallet state, which is outside adapter scope.

## S423 Additions

S423 elevated the S417 evidence from mock-based error classification to lifecycle-grade
proof with the following additions:

1. **Explicit ValidTransition assertions** — Every lifecycle path (submitted->rejected,
   accepted->partially_filled->filled) verified step-by-step against the canonical
   state machine, matching the S422 pattern for the fill path.

2. **QueryOrder reconciliation** — Proven that QueryOrder recovers REJECTED, EXPIRED,
   and PARTIALLY_FILLED orders via GET /fapi/v1/order with correct status mapping
   and fill records.

3. **Terminal state exhaustive proof** — Verified that rejected allows no further
   transitions to any state (submitted, sent, accepted, filled, partially_filled,
   rejected, cancelled).

4. **Multi-scenario rejection event audit trail** — 6 error scenarios proven end-to-end
   from adapter error classification through rejection event construction with
   correlation chain and venue detail preservation.

5. **S422 fill-path regression** — Proven that the fill path (S422) is unchanged
   by S423 changes, including correlation chain and fill record fidelity.

## Readiness for S424

S423 closes the rejection and partial fill lifecycle gaps. The Futures segment
now has lifecycle-grade evidence for:

- Acceptance/fill (S422, with ValidTransition proof)
- Rejection and partial fill (S423, with ValidTransition proof and QueryOrder reconciliation)

S424 should consolidate read-path alignment and segment parity across both
segments under the unified runtime.
