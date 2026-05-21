# S417: Futures Real Rejection and Partial Fill Report

**Stage**: S417
**Wave**: Phase 45 -- Futures Venue Execution Proof (S415--S420)
**Type**: Implementation and proof
**Status**: Complete
**Date**: 2026-03-23

## Executive Summary

S417 proves the Futures rejection and partial fill lifecycle paths through the
BinanceFuturesTestnetAdapter and SegmentRouter under the unified runtime.

25 tests (17 adapter-level + 8 actor-level) validate:

- Rejection path `submitted -> rejected` with 10 distinct error scenarios
- Partial fill path with Futures response format (avgPrice + cumQuote)
- Rejection event construction with complete audit trail
- Error classification: retryable vs non-retryable with venue-code overrides
- Segment isolation (Spot not contacted for Futures paths)
- Correlation/causation chain preservation through rejection and partial fill paths
- Quantity monotonicity invariant for partial fills
- Fill timestamp from venue `updateTime` (not local clock)
- DryRunSubmitter intercept for both rejection and partial fill scenarios
- Full S416 regression suite passes (fill path unchanged)

All 25 tests pass. Zero regressions across the full test suite.
Zero production code changes required -- the existing adapter and actor infrastructure
supported all paths without modification.

## What Was Proved

### Rejection: STRONG Evidence

The Futures adapter correctly handles all known Binance error codes and maps them
to the canonical lifecycle state machine:

```
HTTP 400 / -2019 (insufficient margin)  -> Problem(InvalidArgument), non-retryable
HTTP 400 / -2010 (insufficient balance) -> Problem(InvalidArgument), non-retryable
HTTP 400 / -1013 (LOT_SIZE violation)   -> Problem(InvalidArgument), non-retryable
HTTP 401 / -2015 (auth failure)         -> Problem(InvalidArgument), non-retryable
HTTP 429 (rate limit)                   -> Problem(Unavailable), retryable
HTTP 400 / -1001 (venue internal)       -> Problem(Unavailable), retryable (override)
HTTP 400 / -1015 (order rate limit)     -> Problem(Unavailable), retryable (override)
HTTP 503 (server error)                 -> Problem(Unavailable), retryable
HTTP 200 / REJECTED                     -> StatusRejected (response parsing)
HTTP 200 / EXPIRED                      -> StatusRejected (response parsing)
```

Rejection event construction carries complete audit trail:
- RejectionCode, RejectionReason, VenueDetails (venue_http_status, venue_error_code)
- Intent mutated: Status=rejected, Final=true
- Correlation/causation chain preserved

### Partial Fill: STRUCTURAL Evidence

The Futures adapter correctly parses PARTIALLY_FILLED responses in the Futures
format (avgPrice + cumQuote, no fills[] array):

- StatusPartiallyFilled correctly derived from `mapBinanceStatus("PARTIALLY_FILLED")`
- Fill record: Price from avgPrice, Quantity from executedQty, Fee from cumQuote
- Simulated=false for real venue responses
- Timestamp from venue updateTime
- FilledQuantity <= Quantity (monotonicity invariant)
- partially_filled is NOT terminal (transitions to filled or cancelled allowed)

**Honest gap**: No live partial fill was observed on the Futures testnet because
market orders fill instantly with synthetic liquidity. This mirrors the S406 Spot
limitation and is accepted with the same rationale.

### Parity with Spot (S406)

Futures now has full parity with Spot for rejection and partial fill:

| Capability | Spot (S406) | Futures (S417) |
|---|---|---|
| Rejection via HTTP error | Proven | Proven |
| Rejection via HTTP 200 | Proven | Proven |
| Rejection event construction | Proven | Proven |
| Partial fill parsing | Proven | Proven |
| Live partial fill observed | No | No |
| Segment isolation | Proven | Proven |

## Files Changed

### New Files

| File | Purpose |
|---|---|
| `internal/application/execution/s417_futures_rejection_partial_fill_test.go` | 17 adapter-level tests |
| `internal/actors/scopes/execute/s417_futures_rejection_partial_fill_test.go` | 8 actor-composition tests |
| `scripts/smoke-futures-rejection-partial-fill.sh` | S417 smoke script |
| `docs/architecture/futures-real-rejection-and-partial-fill-evidence-on-unified-runtime.md` | Rejection and partial fill proof |
| `docs/architecture/futures-rejected-partialfill-evidence-strength-auditability-and-limitations.md` | Evidence strength and auditability assessment |

### Modified Files

None. Zero production code changes required.

## Principal Evidence

### Rejection (Adapter Level)

| Test | Error Code | Classification | Retryable |
|---|---|---|---|
| `TestS417_Rejection_InsufficientMargin` | -2019 | InvalidArgument | No |
| `TestS417_Rejection_InsufficientBalance` | -2010 | InvalidArgument | No |
| `TestS417_Rejection_InvalidQuantity` | -1013 | InvalidArgument | No |
| `TestS417_Rejection_AuthFailure` | -2015 | InvalidArgument | No |
| `TestS417_Rejection_RateLimit` | -1015 (429) | Unavailable | Yes |
| `TestS417_Rejection_VenueInternalOverride` | -1001 | Unavailable | Yes |
| `TestS417_Rejection_OrderRateLimitOverride` | -1015 (400) | Unavailable | Yes |
| `TestS417_Rejection_ServerError` | -- (503) | Unavailable | Yes |
| `TestS417_Rejection_VenueRejectedStatus` | REJECTED | StatusRejected | N/A |
| `TestS417_Rejection_VenueExpiredStatus` | EXPIRED | StatusRejected | N/A |
| `TestS417_Rejection_LifecycleTransition` | -- | submitted->rejected valid | -- |
| `TestS417_Rejection_CorrelationPreserved` | -2019 | venue details preserved | -- |

### Partial Fill (Adapter Level)

| Test | Evidence |
|---|---|
| `TestS417_PartialFill_FuturesFormat` | avgPrice/cumQuote correctly parsed |
| `TestS417_PartialFill_LifecycleTransitions` | All valid transitions confirmed |
| `TestS417_PartialFill_QuantityMonotonicity` | 3 boundary cases: half, quarter, tiny |
| `TestS417_PartialFill_FillTimestamp` | Timestamp from venue updateTime |
| `TestS417_Regression_FilledStillWorks` | S416 fill path unchanged |

### Actor Composition

| Test | Evidence |
|---|---|
| `TestS417_ActorComposition_FuturesRejection_InsufficientMargin` | Router + segment isolation |
| `TestS417_ActorComposition_FuturesRejection_LOTSize` | Router + error classification |
| `TestS417_ActorComposition_FuturesRejectionEvent_Construction` | Full event path + audit trail |
| `TestS417_ActorComposition_FuturesRejection_VenueRejectedStatus200` | HTTP 200 REJECTED parsing |
| `TestS417_ActorComposition_FuturesPartialFill_ThroughRouter` | Router + segment isolation |
| `TestS417_ActorComposition_FuturesPartialFill_CorrelationPreserved` | Correlation chain |
| `TestS417_ActorComposition_FuturesPartialFill_DryRunIntercepted` | DryRunSubmitter bypass |
| `TestS417_RejectionEvent_FuturesVenueDetails_AuditTrail` | 4 sub-cases: audit completeness |

## Remaining Limitations

1. **No live partial fill observed on testnet** -- structural proof only (same as S406)
2. **cumQuote as fee proxy** -- not true commission (known since S416)
3. **Single symbol scope** -- BTCUSDT only
4. **No position/leverage context** -- adapter does not manage margin or positions

## Handoff to S418

S417 closes the rejection and partial fill gaps identified in S416. Combined with
S416 (acceptance/fill), the Futures segment now has full lifecycle evidence parity
with Spot.

S418 should consolidate:

1. Read-path alignment across Spot and Futures segments
2. Lifecycle queryability on the unified runtime
3. Any remaining cross-segment isolation hardening
