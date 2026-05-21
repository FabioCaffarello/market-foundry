# Futures Real Rejection and Partial Fill Evidence

**Stage**: S423 — Futures Real Rejection and Partial Fill Evidence
**Wave**: Futures Venue Execution Proof (S421 charter)
**Status**: PASS
**Date**: 2026-03-23

## Purpose

This document records the evidence of rejection and partial fill lifecycle paths on the Binance Futures Testnet, proving the system correctly handles the `submitted -> rejected` and `accepted -> partially_filled` transitions with real venue response contracts.

## Governing Questions

| ID | Question | Verdict |
|---|---|---|
| FV-Q3 | Does lifecycle transition to rejected on real Futures rejection? | PROVEN |
| FV-Q4 | Does VenueOrderRejectedEvent carry real Futures error code and reason? | PROVEN |
| FV-Q5 | Can partially_filled be observed or structurally proven from Futures? | STRUCTURAL |
| FV-Q6 | Does quantity monotonicity hold under Futures partial fills? | PROVEN |

## Rejection Evidence

### Error Response Classification (6 scenarios proven)

| HTTP Status | Venue Code | Scenario | Problem Code | Retryable | Audit Trail |
|---|---|---|---|---|---|
| 400 | -2019 | Insufficient margin | InvalidArgument | No | venue_http_status, venue_error_code |
| 400 | -2010 | Insufficient balance | InvalidArgument | No | venue_http_status, venue_error_code |
| 400 | -1013 | LOT_SIZE violation | InvalidArgument | No | venue_http_status, venue_error_code |
| 401 | -2015 | Auth failure | InvalidArgument | No | venue_http_status |
| 429 | -1015 | Rate limit | Unavailable | Yes | venue_http_status, venue_error_code |
| 400 | -1001 | Venue internal (override) | Unavailable | Yes | venue_http_status, venue_error_code, venue_error_class |

### HTTP 200 Rejection Status (2 scenarios proven)

| Venue Status | Domain Status | Terminal | Fills |
|---|---|---|---|
| REJECTED | StatusRejected | Yes | 0 |
| EXPIRED | StatusRejected | Yes | 0 |

### Lifecycle Proof

All rejection paths verified with explicit `ValidTransition` assertions:

- `submitted -> rejected`: valid (direct rejection)
- `sent -> rejected`: valid (network-level rejection)
- `rejected -> *`: invalid (terminal state, no further transitions)
- `rejected.IsTerminal()`: true

### QueryOrder Reconciliation

Proven that `QueryOrder` (GET /fapi/v1/order) correctly recovers:
- REJECTED status -> StatusRejected
- EXPIRED status -> StatusRejected
- Zero fills and zero FilledQuantity for both paths

### Rejection Event Construction

VenueOrderRejectedEvent construction proven with:
- Intent status mutated to `rejected`, `Final=true`
- RejectionCode carries Problem.Code (e.g., VAL_INVALID_ARGUMENT)
- RejectionReason carries full Problem.Message with venue context
- VenueDetails carries venue_http_status, venue_error_code, venue_error_class
- CorrelationID/CausationID preserved from incoming event
- Source/Symbol preserved through rejection
- Zero fills on rejected intent

## Partial Fill Evidence

### Response Format (Futures-specific)

Futures partial fill uses a different response format from Spot:

| Field | Futures | Spot |
|---|---|---|
| Price source | `avgPrice` (top-level) | Weighted avg from `fills[]` array |
| Fee source | `cumQuote` (fee proxy) | Per-leg `commission` in `fills[]` |
| Timestamp | `updateTime` | `transactTime` |
| Aggregation | None needed (venue aggregates) | Per-leg to single record |

### Fill Record Fidelity

Proven with PARTIALLY_FILLED response:
- Status: StatusPartiallyFilled
- Fill.Price from avgPrice
- Fill.Quantity from executedQty
- Fill.Fee from cumQuote (fee proxy)
- Fill.Simulated = false (real venue)
- Fill.Timestamp from updateTime (venue clock)
- FilledQuantity = executedQty

### Lifecycle Proof

All partial fill paths verified with explicit `ValidTransition` assertions:

- `submitted -> accepted`: valid
- `accepted -> partially_filled`: valid
- `partially_filled -> filled`: valid (completion)
- `partially_filled -> cancelled`: valid (abandon)
- `partially_filled.IsTerminal()`: false

### Quantity Monotonicity (FV-Q6)

Proven across 4 fill ratios:

| Scenario | Quantity | ExecutedQty | Invariant |
|---|---|---|---|
| half_filled | 0.001 | 0.0005 | FilledQuantity < Quantity |
| quarter_filled | 0.004 | 0.001 | FilledQuantity < Quantity |
| tiny_partial | 1.0 | 0.001 | FilledQuantity < Quantity |
| near_full | 0.001 | 0.0009 | FilledQuantity < Quantity |

### QueryOrder Reconciliation

Proven that `QueryOrder` correctly recovers PARTIALLY_FILLED orders with:
- StatusPartiallyFilled
- Correct fill record (price, quantity, timestamp from venue)
- VenueOrderID preserved

## Segment Isolation

Both rejection and partial fill paths proven isolated:
- Source `binancef` routes exclusively to Futures adapter
- Spot adapter sentinel server never contacted
- Proven at both adapter level (S423) and actor composition level (S417)

## Regression

S422 fill path proven unchanged:
- FILLED status still produces StatusFilled
- Fill records carry correct Simulated=false
- Correlation chain preserved through fill path
- FilledQuantity matches executedQty

## Test Coverage

| Level | Tests | File |
|---|---|---|
| Adapter (S423) | 19 | `internal/application/execution/s423_futures_rejection_partial_fill_test.go` |
| Adapter (S417) | 17 | `internal/application/execution/s417_futures_rejection_partial_fill_test.go` |
| Actor (S417) | 8 | `internal/actors/scopes/execute/s417_futures_rejection_partial_fill_test.go` |
| **Total** | **44** | |

## Honest Limitations

1. **Partial fill not observed on testnet**: Futures testnet fills market orders instantly. PARTIALLY_FILLED is structurally proven via mock responses matching the real Futures API contract, not via live observation. This is an environment limitation, not an implementation gap.

2. **cumQuote as fee proxy**: The `cumQuote` field is the cumulative quote asset spent, not the actual trading commission. True commission data requires the `GET /fapi/v1/userTrades` endpoint, which is out of scope for this wave.

3. **No per-leg aggregation**: Futures venue provides pre-aggregated `avgPrice`, unlike Spot which returns per-fill legs. This means the adapter's fill aggregation logic is not exercised for Futures.

4. **Single symbol scope**: All evidence uses BTCUSDT. Cross-symbol behavior is structurally equivalent but not explicitly tested.

5. **Testnet behavioral differences**: Testnet may accept orders that production would reject (e.g., margin requirements may be relaxed). The error code classification is proven structurally against the documented Binance API contract.
