# Futures Accepted/Filled Real Response Alignment, Controls, and Limitations

**Stage**: S422 (post-simplification wave, Phase 47)
**Wave**: Futures Venue Execution Proof (S421--S426)
**Predecessor**: S416 (Phase 45), S421 (charter)
**Status**: Complete
**Date**: 2026-03-23

---

## Purpose

Document the alignment between Binance Futures testnet real responses and the
canonical order lifecycle model (S383), the controls applied at each boundary,
and the known limitations discovered during S422 proof execution.

This document supersedes the S416 version with post-simplification canonical
surface alignment and multi-cycle sustained connectivity evidence.

---

## Response Alignment Matrix

### Binance Futures Status -> Domain Status

| Binance Status | Domain Status | ValidTransition from submitted? | Terminal? |
|---|---|---|---|
| NEW | accepted | yes (submitted -> accepted) | no |
| FILLED | filled | yes (accepted -> filled) | yes |
| PARTIALLY_FILLED | partially_filled | yes (accepted -> partially_filled) | no |
| CANCELED | cancelled | yes (accepted -> cancelled) | yes |
| CANCELLED | cancelled | yes (accepted -> cancelled) | yes |
| REJECTED | rejected | yes (submitted -> rejected) | yes |
| EXPIRED | rejected | yes (submitted -> rejected) | yes |

### Dominant Path Proof

```
submitted  --(SubmitOrder)--> accepted  --(venue FILLED)--> filled
    |                                                          |
    +-- ValidTransition(submitted, accepted) = true            |
                    ValidTransition(accepted, filled) = true --+
                                           IsTerminal(filled) = true
```

This path is exercised with BUY and SELL sides. Both produce identical lifecycle
transitions. The `SideNone` case short-circuits before venue contact and returns
`StatusAccepted` without an HTTP call (no-op path).

**S422 addition**: Each ValidTransition step is explicitly asserted in
`TestS422_FuturesConnectivity_DominantPath_ValidTransitions`, including
terminal state enforcement (no transitions from `filled`).

---

## Controls

### C1: Request Signing (HMAC-SHA256)

Every request to `/fapi/v1/order` includes:
- `X-MBX-APIKEY` header with the API key
- `signature` query parameter: HMAC-SHA256 of the full query string (64 hex chars)
- `timestamp` and `recvWindow=5000` for replay protection

**Evidence**: `TestS422_FuturesAPI_HMACSigned` (S422), `TestS416_FuturesVenueLive_RequestSigned` (S416).

### C2: Client Order ID (Idempotency)

- `newClientOrderId` is derived deterministically from `ExecutionIntent.DeduplicationKey()`
- SHA-256 hash truncated to 32 hex chars (fits Binance 36-char limit)
- Sent to venue in the request, returned in the receipt
- Enables post-200 reconciliation (QueryOrder with `origClientOrderId`)

**Evidence**: `TestS422_FuturesCorrelation_ClientOrderIDDeterministic` (S422).

### C3: Segment Routing Isolation

- `SegmentForSource("binancef")` -> `MarketSegmentFutures`
- SegmentRouter dispatches Futures intents only to the Futures adapter
- Spot adapter receives zero calls for Futures intents (proven with sentinel server)
- Unknown sources produce an error (fail-closed)

**Evidence**: `TestS422_SegmentRouter_FuturesRoutedCorrectly_SpotIsolated`, `TestS422_SegmentRouter_UnknownSource_FailsClosed` (S422).

### C4: Error Classification

| HTTP Status | Venue Code | Classification | Retryable? |
|---|---|---|---|
| 400 | -2019 | Insufficient margin | no |
| 401/403 | any | Authentication failure | no |
| 429 | -1015 | Rate limit | yes |
| 400 | -1001 | Venue internal (code override) | yes |
| 400 | -1003 | IP rate limit (code override) | yes |
| 400 | -1015 | Order rate limit (code override) | yes |
| 503 | any | Venue unavailable | yes |
| 502 | any | Bad gateway | yes |

**Evidence**: S416 error classification tests (retained from prior wave, still passing).

### C5: DryRunSubmitter Bypass

When `dry_run=false` (canonical `execute-venue-live.jsonc`), the DryRunSubmitter
is not composed into the pipeline. The SegmentRouter calls the real Futures
adapter directly.

When `dry_run=true` (canonical `execute-unified.jsonc`), the DryRunSubmitter
intercepts all calls. Fills are marked `Simulated=true` with `dryrun-` prefix.

**Evidence**: `TestS422_CanonicalConfig_VenueLive_FuturesEnabled` validates `dry_run=false`.

### C6: Fill Record Fidelity

| Field | Source | Example |
|---|---|---|
| Price | `avgPrice` from Futures response (direct) | "67891.23" |
| Quantity | `executedQty` from response | "0.002" |
| Fee | `cumQuote` from response (quote cost proxy) | "135.78246" |
| Simulated | hardcoded `false` for real venue | false |
| Timestamp | `updateTime` from response (milliseconds -> UTC) | 2026-03-23T14:30:00Z |

**Evidence**: `TestS422_FuturesFillRecord_PriceFromAvgPrice`, `TestS422_FuturesFillRecord_TimestampFromUpdateTime` (S422).

### C7: Correlation Chain Preservation

| Field | Preserved? | Evidence |
|---|---|---|
| CorrelationID | Yes | `TestS422_FuturesCorrelation_ChainPreservedThroughVenue` |
| CausationID | Yes | Same test |
| Type, Source, Symbol, Timeframe | Yes | `TestS422_FuturesCorrelation_IntentFieldsPreservedAfterFill` |
| Risk (Type, Disposition) | Yes | Same test |
| Final flag | Yes | Same test |

### C8: Multi-Cycle Sustained Connectivity (NEW in S422)

5 sequential order submissions with alternating BUY/SELL sides:
- Unique VenueOrderIDs across all cycles (no duplicates)
- CorrelationID preserved in every cycle
- Zero errors across all cycles

**Evidence**: `TestS422_FuturesConnectivity_MultiCycleSustained`.

### C9: Post-200 Reconciliation

QueryOrder via GET `/fapi/v1/order` with `origClientOrderId` recovers full
fill state including VenueOrderID, status, and fill record.

**Evidence**: `TestS422_FuturesReconciliation_QueryOrderRecoversFill`, `TestS422_FuturesReconciliation_QueryUsesCorrectFuturesPath` (S422).

---

## Canonical Surface Compliance (NEW in S422)

| Surface | Canonical | Validated |
|---|---|---|
| Config for proof | `execute-venue-live.jsonc` | `TestS422_CanonicalConfig_VenueLive_FuturesEnabled` |
| Config for dry-run | `execute-unified.jsonc` | `TestS422_CanonicalConfig_Unified_DryRunTrue` |
| Compose for proof | `docker-compose.yaml` + `docker-compose.venue-live.yaml` | By contract (S421) |
| New configs created | None | NG-41, NG-50 respected |
| New compose files created | None | NG-46, NG-47, NG-49 respected |
| API path | `/fapi/v1/order` | `TestS422_FuturesAPI_PathIsFapi` |
| Response type | `RESULT` | `TestS422_FuturesAPI_RESULTResponseType` |

---

## Limitations

### L1: cumQuote Is Not True Commission (G-4)

The Futures RESULT response does not include per-fill commission. The adapter
uses `cumQuote` (total quote asset cost = qty * avgPrice) as a fee proxy.
True commission data requires GET `/fapi/v1/userTrades`, which is excluded
by NG-40. This asymmetry with Spot (which returns actual commission) is
tracked as G-4 (Medium severity, not blocking).

### L2: No Partial Fill Exercise

S422 proves the dominant path only. `PARTIALLY_FILLED` is mapped correctly
in `mapBinanceStatus()` and produces a valid `partially_filled` domain status,
but no test exercises the venue returning `PARTIALLY_FILLED` with fill record
verification. Deferred to S423.

### L3: No Rejection Exercise

Rejection paths are tested at the adapter level with mock servers (S416).
Real venue rejection under testnet conditions is deferred to S423.

### L4: No Position/Leverage Management

The adapter submits market orders but does not query or manage open positions,
leverage settings, or margin mode. These are excluded by NG-11, NG-33, NG-35.

### L5: Testnet Behavioral Differences

Futures testnet may behave differently from mainnet in:
- Fill latency (testnet fills typically instant)
- Price realism (testnet prices may diverge from mainnet)
- Rate limit thresholds (testnet may be more permissive)
- Available symbols and contract specifications

### L6: Single Symbol Scope

S422 exercises only BTCUSDT. Multi-symbol execution is structurally supported
but not explicitly proven. NG-3 prohibits multi-symbol execution.

### L7: Compressed Lifecycle

The Binance Futures API returns `FILLED` directly. The `accepted` intermediate
state is never materialized in persistence. This is consistent with Spot behavior
and accepted by design.

### L8: No Live Testnet in Unit Tests

All S422 tests use httptest.Server. Live testnet connectivity is validated at
compose E2E level (S425) via smoke scripts. This separation is by design.

---

## Preparation for S423

S423 (rejection and partial fill) should:
1. Trigger real rejection scenarios: insufficient margin (`-2019`), invalid quantity, invalid symbol.
2. Verify `VenueOrderRejectedEvent` carries real Futures error details.
3. Attempt partial fill observation on Futures testnet (more likely than Spot).
4. Verify quantity monotonicity under partial fills.
5. Validate `submitted -> rejected` transition via `ValidTransition()`.
6. Reuse the same canonical config/compose surface (no deviations).

---

## Links

| Reference | Link |
|---|---|
| Proof document | [`futures-real-venue-connectivity-and-lifecycle-acceptance-fill-proof.md`](futures-real-venue-connectivity-and-lifecycle-acceptance-fill-proof.md) |
| Wave charter | [`futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md`](futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md) |
| Stage report | [`../stages/stage-s422-futures-real-venue-acceptance-fill-proof-report.md`](../stages/stage-s422-futures-real-venue-acceptance-fill-proof-report.md) |
