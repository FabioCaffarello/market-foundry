# Futures Real Venue Connectivity and Lifecycle Acceptance/Fill Proof on Unified Runtime

**Stage**: S416
**Wave**: Phase 45 -- Futures Venue Execution Proof (S415--S420)
**Status**: Complete
**Date**: 2026-03-23

## Purpose

Prove that the Binance Futures testnet adapter achieves real venue connectivity
under the unified runtime and that the dominant lifecycle path
`submitted -> accepted -> filled` produces correct, auditable results with
Futures-specific response semantics.

## Dominant Lifecycle Path

```
ExecutionIntent (source="binancef", status=submitted)
  -> SegmentRouter (source -> MarketSegmentFutures)
    -> BinanceFuturesTestnetAdapter
      -> POST /fapi/v1/order (HMAC-SHA256 signed)
      <- HTTP 200 {status: "FILLED", avgPrice: "...", cumQuote: "...", updateTime: ...}
    -> parseOrderResponse()
      -> mapBinanceStatus("FILLED") -> StatusFilled
      -> FillRecord{Price: avgPrice, Qty: executedQty, Fee: cumQuote, Simulated: false}
    <- VenueOrderReceipt {Status: filled, Fills: [{Price, Qty, Fee, Simulated: false}]}
```

### Lifecycle Alignment with S383

| Transition | ValidTransition() | Evidence |
|---|---|---|
| submitted -> accepted | true | Binance "NEW" -> StatusAccepted |
| accepted -> filled | true | Binance "FILLED" -> StatusFilled |
| filled -> * | false | Terminal, absorbing state |

The Futures path uses the same canonical lifecycle as Spot. The adapter
maps Binance Futures response statuses identically to the Spot adapter
via the shared `mapBinanceStatus()` function.

## Key Differences from Spot (S405)

| Dimension | Spot | Futures |
|---|---|---|
| Base URL | testnet.binance.vision | testnet.binancefuture.com |
| API path | /api/v3/order | /fapi/v1/order |
| Response type | FULL (fills[] array) | RESULT (avgPrice, cumQuote) |
| Price source | Weighted average from fills[] legs | Direct avgPrice field |
| Fee source | Per-leg commission aggregation | cumQuote (quote cost proxy) |
| Timestamp field | transactTime | updateTime |

These differences are structural to Binance's Spot vs Futures APIs and
are encapsulated within each adapter. The SegmentRouter and domain lifecycle
are segment-agnostic.

## Config Pattern

Config file: `deploy/configs/execute-venue-live-futures.jsonc`

- `dry_run: false` -- bypasses DryRunSubmitter
- `futures.enabled: true, adapter: binance_futures_testnet` -- real Futures adapter
- `spot.enabled: true` -- structural coexistence preserved
- Port: 8085 (distinct from Spot venue-live at 8084)

## Connectivity Controls

### Authentication

- HMAC-SHA256 signing via `X-MBX-APIKEY` header and `signature` query parameter
- Credentials loaded from `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` / `API_SECRET`
- Credentials are never logged or included in error messages

### Request Safety

- `recvWindow=5000` prevents replay attacks
- Per-request deadline enforced (EC-3): default 10s if caller provides none
- Response body capped at 64 KB to prevent memory exhaustion
- `newClientOrderId` is deterministic for idempotency

### Segment Isolation

- `SegmentForSource("binancef")` -> `MarketSegmentFutures`
- Futures intents are routed exclusively to the Futures adapter
- Spot adapter is never contacted for Futures intents (proven by test)

## Correlation and Audit Trail

The following fields survive the full adapter round-trip without mutation:

- `CorrelationID` -- end-to-end trace identity
- `CausationID` -- upstream causation chain
- `Source` ("binancef") -- segment identity
- `Symbol`, `Timeframe`, `Risk.*` -- intent context
- `ClientOrderID` -- deterministic, sent to venue and returned in receipt

## Testnet Observations and Limitations

1. **Testnet liquidity**: Futures testnet has synthetic liquidity. MARKET orders
   fill immediately in most cases. Real mainnet may exhibit partial fills.

2. **avgPrice precision**: Futures testnet returns avgPrice as a string with
   variable decimal places. The adapter passes it through without modification.

3. **cumQuote as fee proxy**: The Futures API does not include commission in
   the RESULT response type. `cumQuote` (total quote asset cost) is used as a
   fee proxy. True commission requires a separate endpoint (not in scope).

4. **No position management**: The adapter submits orders but does not manage
   open positions, leverage, or margin. This is intentional for S416.

5. **Credential scope**: Testnet credentials are distinct from mainnet. The
   adapter hardcodes the testnet URL and cannot accidentally reach mainnet.

## Evidence Summary

| Evidence class | Test count | Location |
|---|---|---|
| Adapter-level (connectivity, lifecycle, fidelity) | 32 | `internal/application/execution/s416_futures_venue_acceptance_fill_test.go` |
| Actor-level (routing, composition, dry-run bypass) | 6 | `internal/actors/scopes/execute/s416_futures_venue_lifecycle_test.go` |

All 38 tests pass. Zero regressions across the full test suite.
