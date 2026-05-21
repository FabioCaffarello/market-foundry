# Futures Real Rejection and Partial Fill Evidence on Unified Runtime

**Stage**: S417
**Wave**: Phase 45 -- Futures Venue Execution Proof (S415--S420)
**Scope**: Rejection and partial fill lifecycle paths for Futures segment
**Date**: 2026-03-23

## Purpose

This document proves the rejection and partial fill lifecycle paths through the
BinanceFuturesTestnetAdapter and SegmentRouter under the unified runtime.

S416 proved the dominant path (`submitted -> filled`). S417 closes the two
remaining lifecycle gaps identified in S416's limitations:

1. **Rejection**: `submitted -> rejected` with real Futures error codes
2. **Partial fill**: `submitted -> partially_filled` with Futures response format

## Rejection Path (Proven)

### Error Response Path

```
ExecutionIntent (source="binancef", status=submitted)
  -> SegmentRouter (source -> MarketSegmentFutures)
    -> BinanceFuturesTestnetAdapter
      -> POST /fapi/v1/order (HMAC-SHA256 signed)
      <- HTTP 400 {code: -2019, msg: "Margin is insufficient."}
    -> handleErrorResponse()
      -> classifyByVenueErrorCode() -- check for override codes (-1001, -1003, -1015)
      -> HTTP 400 generic -> Problem(InvalidArgument, non-retryable)
      -> Problem.Details: {venue_http_status: 400, venue_error_code: -2019}
    <- Problem returned to actor layer
  -> VenueAdapterActor.publishRejection()
    -> intent.Status = StatusRejected, intent.Final = true
    -> VenueOrderRejectedEvent {
         RejectionCode: "VAL_INVALID_ARGUMENT",
         RejectionReason: "venue rejected order (HTTP 400, code -2019): Margin is insufficient.",
         VenueDetails: {venue_http_status: 400, venue_error_code: -2019}
       }
    -> Publish to EXECUTION_REJECTION_EVENTS stream
```

### Venue Rejected/Expired Status Path

```
ExecutionIntent (source="binancef", status=submitted)
  -> BinanceFuturesTestnetAdapter
    -> POST /fapi/v1/order
    <- HTTP 200 {status: "REJECTED", avgPrice: "0", executedQty: "0"}
  -> parseOrderResponse()
    -> mapBinanceStatus("REJECTED") -> StatusRejected
    -> No fills built (executedQty=0)
  <- VenueOrderReceipt {Status: rejected, FilledQuantity: "0", Fills: []}
```

### Error Classification Matrix (Futures)

| HTTP Status | Venue Code | Classification | Retryable | Venue Error Class |
|---|---|---|---|---|
| 400 | -2019 | InvalidArgument | No | -- |
| 400 | -2010 | InvalidArgument | No | -- |
| 400 | -1013 | InvalidArgument | No | -- |
| 401 | -2015 | InvalidArgument | No | -- |
| 429 | -1015 | Unavailable | Yes | -- |
| 400 | -1001 | Unavailable | Yes | venue_internal |
| 400 | -1015 | Unavailable | Yes | order_rate_limit |
| 503 | -- | Unavailable | Yes | -- |
| 200 | -- (REJECTED) | StatusRejected | N/A | (response parsing) |
| 200 | -- (EXPIRED) | StatusRejected | N/A | (response parsing) |

### Rejection Audit Trail Fields

Every rejection event carries:

- `RejectionCode`: maps to `Problem.Code` (e.g., `VAL_INVALID_ARGUMENT`)
- `RejectionReason`: from `Problem.Message` (includes HTTP status and venue code)
- `VenueDetails.venue_http_status`: HTTP status code from venue
- `VenueDetails.venue_error_code`: Binance error code (e.g., -2019)
- `VenueDetails.venue_error_class`: present for override codes (e.g., `venue_internal`)
- `CorrelationID` / `CausationID`: preserved from incoming event metadata
- `ExecutionIntent.Source`: `binancef` (Futures segment)
- `ExecutionIntent.Status`: `rejected`, `Final`: `true`

## Partial Fill Path (Proven Structurally)

### Futures Partial Fill Response Format

```
ExecutionIntent (source="binancef", status=submitted)
  -> BinanceFuturesTestnetAdapter
    -> POST /fapi/v1/order
    <- HTTP 200 {
         status: "PARTIALLY_FILLED",
         avgPrice: "65000.50",
         executedQty: "0.0005",
         cumQuote: "32.50025",
         updateTime: 1711184100000
       }
  -> parseOrderResponse()
    -> mapBinanceStatus("PARTIALLY_FILLED") -> StatusPartiallyFilled
    -> FillRecord{Price: "65000.50", Qty: "0.0005", Fee: "32.50025", Simulated: false}
    -> FilledQuantity: "0.0005"
  <- VenueOrderReceipt {Status: partially_filled}
```

### Key Structural Differences from Spot Partial Fill

| Aspect | Spot (S406) | Futures (S417) |
|---|---|---|
| Response format | `fills[]` array with per-leg data | `avgPrice` + `cumQuote` (single record) |
| Price source | Weighted average from per-leg fills | Direct `avgPrice` field |
| Fee source | Per-leg `commission` aggregated | `cumQuote` as fee proxy |
| Timestamp | `transactTime` | `updateTime` |
| Multi-leg aggregation | Required (computeSpotFillAggregates) | Not needed (venue provides aggregated) |

### Partial Fill Lifecycle Invariants

| Transition | Valid | Evidence |
|---|---|---|
| accepted -> partially_filled | Yes | `ValidTransition()` returns true |
| partially_filled -> filled | Yes | `ValidTransition()` returns true |
| partially_filled -> cancelled | Yes | `ValidTransition()` returns true |
| partially_filled is terminal | No | `IsTerminal()` returns false |

### Quantity Monotonicity

The Futures adapter preserves the structural invariant `FilledQuantity <= Quantity`:

- `FilledQuantity` is set from venue `executedQty` (never synthesized)
- `Quantity` is preserved from the original intent (adapter never modifies it)
- The adapter does not produce fills where `Quantity` < `FilledQuantity`

This was validated with three boundary cases: half-filled, quarter-filled, and tiny-partial.

## Segment Isolation

Futures rejection and partial fill paths are isolated from Spot:

- Source `binancef` routes exclusively to `MarketSegmentFutures` via `SegmentRouter`
- Spot adapter is never called for Futures intents (validated with sentinel servers)
- Rejection events carry `Source: binancef` for downstream segment-aware filtering

## Honest Limitations

1. **Partial fill not observed on Futures testnet** -- Futures testnet market orders
   fill instantly with synthetic liquidity. PARTIALLY_FILLED is proven structurally
   (adapter correctly parses the status and builds fill records) but was not observed
   as a live testnet response. This mirrors the S406 Spot limitation.

2. **cumQuote as fee proxy** -- Futures RESULT response type does not include per-trade
   commission. The `cumQuote` field (cumulative quote quantity) is used as a fee proxy.
   True commission requires the separate `GET /fapi/v1/userTrades` endpoint.

3. **No multi-leg aggregation needed** -- Unlike Spot, Futures RESULT response provides
   `avgPrice` directly. There is no per-leg fill array to aggregate. This is structurally
   simpler but means the adapter cannot distinguish individual execution legs.

4. **Rejection via insufficient margin is the most reproducible testnet scenario** --
   Other rejection codes (-1013, -2015) are proven via httptest mock servers replicating
   real venue error payloads. Live testnet validation of these specific codes would
   require deliberate misconfiguration.

5. **Single symbol scope** -- All evidence is for BTCUSDT. Multi-symbol is structurally
   supported but not exercised.
