# Spot Accepted/Filled Real Response Alignment, Controls, and Limitations

**Stage**: S405
**Wave**: Phase 43 -- Testnet Venue Execution Proof on Unified Runtime (S404--S409)
**Type**: Alignment analysis and controls documentation
**Status**: Complete

## Objective

Document how the canonical ExecutionIntent lifecycle model (S383) aligns with
real Binance Spot testnet responses, identifying controls that enforce correctness
and limitations discovered during proof.

## Lifecycle Alignment

### Canonical Model (S383)

```
submitted -> sent -> accepted -> filled          (dominant path)
submitted -> sent -> accepted -> partially_filled -> filled
submitted -> rejected                             (rejection path)
```

### Binance Spot Testnet Observed Behavior

For market orders with `newOrderRespType=FULL`, the Spot testnet typically returns
`status: "FILLED"` in a single synchronous response. The intermediate states
`sent` and `accepted` are not observed as separate HTTP responses because market
orders fill atomically on Spot.

**Effective observed path**: `submitted -> (accepted ->) filled`

The adapter does not emit `StatusSent` or `StatusAccepted` as intermediate states
for a filled order. Instead, the venue compresses the lifecycle to a single
transition. This is a valid compression because:

1. `ValidTransition(submitted, accepted) = true`
2. `ValidTransition(accepted, filled) = true`
3. The composite path `submitted -> accepted -> filled` consists of individually
   valid transitions

### Status Mapping

| Binance Status | Domain Status | Terminal | Notes |
|---|---|---|---|
| `FILLED` | `StatusFilled` | Yes | Dominant for market orders |
| `NEW` | `StatusAccepted` | No | Rare for market orders (limit orders) |
| `PARTIALLY_FILLED` | `StatusPartiallyFilled` | No | Rare on Spot testnet for market orders |
| `CANCELED` / `CANCELLED` | `StatusCancelled` | Yes | Manual cancellation |
| `REJECTED` / `EXPIRED` | `StatusRejected` | Yes | Insufficient balance, expired |

### Spot-Specific Fill Model

Binance Spot returns a `fills[]` array in the order response. Each fill leg
contains:

```json
{
  "price": "65430.00",
  "qty": "0.001",
  "commission": "0.00006543",
  "commissionAsset": "BNB"
}
```

The adapter aggregates these into a single `FillRecord`:

- **Price**: Weighted average `sum(price * qty) / sum(qty)`
- **Quantity**: Total `executedQty` from response
- **Fee**: Sum of all `commission` values
- **Simulated**: `false` (real venue)
- **Timestamp**: From `transactTime` (millisecond epoch, converted to UTC)

This differs from Futures which returns a single `avgPrice` field.

## Controls

### C1: Lifecycle Transition Validation

Every status returned by the venue is mapped through `mapBinanceStatus()` and
the resulting domain Status is a valid terminal or intermediate state per
`ValidTransition()`. Unknown statuses return a Problem (Internal error).

**Evidence**: `TestS405_SpotLifecycleAlignment_BinanceStatusMapping`

### C2: Fill Record Integrity

Fill records are produced ONLY when `status == StatusFilled || status == StatusPartiallyFilled`.
No fills are produced for `StatusAccepted`, `StatusRejected`, or `StatusCancelled`.

**Evidence**: `TestS405_SpotVenueLive_None_NoVenueContact` (0 fills for no-action)

### C3: Simulated Flag

Real venue fills carry `Simulated: false`. This is enforced by:

- `BinanceSpotTestnetAdapter`: hardcoded `Simulated: false` in `parseOrderResponse()`
- `PaperVenueAdapter`: hardcoded `Simulated: true`
- `DryRunSubmitter`: hardcoded `Simulated: true`

**Evidence**: `TestS405_SpotVenueLive_FillRecordFidelity_SingleLeg`

### C4: VenueOrderID Convention

| Mode | Prefix | Source |
|---|---|---|
| venue_live | Numeric (e.g., "67890") | Venue-assigned `orderId` |
| dry_run | `dryrun-` | Generated hex |
| paper | `paper-` | Generated hex |

**Evidence**: `TestS405_ActorComposition_DryRunDisabled_RealAdapterCalled`, `DryRunEnabled_InterceptsSpotAdapter`

### C5: Correlation/Causation Chain

`CorrelationID` and `CausationID` fields on ExecutionIntent survive the full
write-path through the adapter. The adapter copies the intent struct by value
and modifies only lifecycle fields (Status, Fills, FilledQuantity).

**Evidence**: `TestS405_SpotVenueLive_CorrelationChainPreserved`

### C6: ClientOrderID Determinism

`ClientOrderID = SHA-256(DeduplicationKey)[0:32]` (32 hex chars). The same intent
always produces the same ClientOrderID, enabling:

- Safe retries (idempotent venue submission)
- Post-200 reconciliation (QueryOrder with `origClientOrderId`)

**Evidence**: `TestS405_SpotVenueLive_ClientOrderIDDeterministic`

### C7: Request Signing

All Spot testnet requests include HMAC-SHA256 signature over query parameters,
`X-MBX-APIKEY` header, and `timestamp` parameter with 5000ms `recvWindow`.

**Evidence**: `TestS405_SpotVenueLive_RequestSigned`

### C8: Segment Routing Isolation

`SegmentRouter.SubmitOrder()` resolves `SegmentForSource(intent.Source)` and
dispatches ONLY to the matching adapter. Spot intents (source="binances") never
reach the Futures adapter.

**Evidence**: `TestS405_SegmentRouter_SpotSourceDispatchesToSpotAdapter`

### C9: Post-200 Reconciliation

`BinanceSpotTestnetAdapter.QueryOrder()` uses GET `/api/v3/order` with
`origClientOrderId` parameter. Returns a VenueOrderReceipt with the same
parsing logic as SubmitOrder, enabling body-read-failure recovery.

**Evidence**: `TestS405_SpotVenueLive_QueryOrder_ReconcilesFill`, `QueryOrder_UsesCorrectAPIPath`

### C10: Error Classification

| Error Category | HTTP Status | Retryable | Rationale |
|---|---|---|---|
| Auth failure | 401, 403 | No | Credential issue, retry won't help |
| Insufficient balance | 400, -2010 | No | Client error, retry won't help |
| Rate limit | 429, -1003, -1015 | Yes | Transient, backoff and retry |
| Server error | 5xx | Yes | Venue-side issue, retry may succeed |
| Network error | N/A | Yes | Connectivity issue, retry may succeed |

## Limitations

### L1: Lifecycle Compression

Market orders on Spot testnet typically return `FILLED` in a single response.
The intermediate `accepted` state is not observed as a separate event. This means
the lifecycle path is effectively `submitted -> filled` at the adapter level,
though the canonical model treats it as `submitted -> accepted -> filled`.

**Impact**: Low. The adapter produces the correct terminal state. The `VenueAdapterActor`
publishes `VenueOrderFilledEvent` with `Status=filled`, which is the expected
terminal outcome.

### L2: Fill Array Aggregation

Spot fills are aggregated into a single `FillRecord` with weighted average price.
Per-leg fill details (individual prices, quantities, commissions per leg) are
lost after aggregation.

**Impact**: Low. The aggregated record preserves total quantity, weighted average
price, and total commission. Per-leg details can be recovered from venue logs
if needed for forensics.

### L3: Commission Asset Not Captured

The `commissionAsset` field (e.g., "BNB") from Spot fill legs is not captured
in the domain `FillRecord`. Commission is recorded as a numeric amount only.

**Impact**: Low. Commission asset diversity is not needed for the current OMS
foundation. Can be added if multi-asset fee accounting is needed.

### L4: No Streaming Updates

Spot testnet market orders fill synchronously. There are no WebSocket fill
updates to reconcile against. The adapter relies entirely on the synchronous
POST response.

**Impact**: None for market orders. Limit order support (future) would require
WebSocket integration.

### L5: Global Dry-Run Toggle

When `dry_run=false`, ALL enabled segment adapters become live. There is no
per-segment dry_run toggle. Spot-only live execution relies on controlling
which intents are submitted, not on per-segment gating.

**Impact**: Medium. Documented as frozen non-goal NG-30 in S404. Acceptable for
Spot-first testnet proof.

### L6: Testnet Behavioral Differences

Binance Spot testnet may differ from production in:

- Fill latency (testnet is faster)
- Rejection behavior (testnet may reject different symbols)
- Balance mechanics (testnet balance is manually provisioned)
- Rate limits (testnet may have different thresholds)

**Impact**: Medium. S405 proves the integration contract, not production behavior.
Production proof is explicitly out of scope (no mainnet).

## Preparation for S406

S406 targets the rejection and partial fill paths:

1. **TV-Q3**: Real rejection lifecycle (`submitted -> rejected`)
2. **TV-Q4**: Rejection event fidelity (code, reason, HTTP status)
3. **TV-Q5**: Partial fill observation or structural proof
4. **TV-Q6**: Quantity monotonicity under partial fills

S405 establishes the adapter, routing, and config foundation that S406 will
extend with:

- Rejection response handling tests
- Partial fill response handling tests
- `VenueOrderRejectedEvent` publication verification
- Insufficient balance and invalid symbol scenarios
