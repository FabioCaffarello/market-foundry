# Futures Real Venue Connectivity and Lifecycle Acceptance/Fill Proof

> Wave: Futures Venue Execution Proof (Post-Simplification, Phase 47)
> Stage: S422 -- Connectivity and Fill Proof
> Date: 2026-03-23
> Predecessor: S421 -- Charter and Scope Freeze
> Canonical surface: `execute-venue-live.jsonc` + `docker-compose.venue-live.yaml`

---

## 1. Purpose

This document proves that the Binance Futures testnet adapter produces correct lifecycle transitions (`submitted -> accepted -> filled`) and accurate fill records when exercised against realistic Futures testnet responses on the unified runtime, using the canonical surface frozen in S421.

---

## 2. Governing Questions Answered

| ID | Question | Verdict | Evidence |
|---|---|---|---|
| **FV-Q1** | Does `venue_live` write-path produce correct lifecycle transitions for Futures? | **ANSWERED** | 19 tests prove BUY/SELL fill, ValidTransition step-by-step, multi-cycle sustainability |
| **FV-Q2** | Do fill records carry real `avgPrice`, `executedQty`, `cumQuote`? | **ANSWERED** | Fill fidelity tests verify price, qty, fee, timestamp, Simulated=false |
| **FV-Q11** | Does correlation chain remain intact through Futures interactions? | **ANSWERED** | CorrelationID and CausationID preservation proven, intent fields verified post-fill |
| **FV-Q12** | Does post-200 reconciliation work under Futures conditions? | **ANSWERED** | QueryOrder recovers fill state via GET /fapi/v1/order with origClientOrderId |

---

## 3. Connectivity Proof

### 3.1 Adapter Under Test

| Component | Value |
|---|---|
| Adapter | `BinanceFuturesTestnetAdapter` |
| API endpoint | `/fapi/v1/order` (POST for submit, GET for query) |
| Response type | `RESULT` (Futures convention; Spot uses `FULL`) |
| Response model | `avgPrice`, `executedQty`, `cumQuote`, `updateTime` |
| HMAC signing | SHA-256 with API secret, 64 hex char signature |
| Base URL | `https://testnet.binancefuture.com` (production), httptest.Server (tests) |

### 3.2 Request Contract

Every Futures order submission sends:

| Parameter | Value | Source |
|---|---|---|
| `symbol` | Uppercase (e.g., `BTCUSDT`) | `mapSymbol(intent.Symbol)` |
| `side` | `BUY` or `SELL` | `intent.Side` |
| `type` | `MARKET` | Hardcoded |
| `quantity` | Decimal string | `intent.Quantity` |
| `newOrderRespType` | `RESULT` | Hardcoded (Futures-specific) |
| `newClientOrderId` | 32 hex chars | `ClientOrderID(intent)` deterministic |
| `timestamp` | Unix milliseconds | `time.Now().UnixMilli()` |
| `recvWindow` | `5000` | Hardcoded |
| `signature` | HMAC-SHA256 | `sign(params)` |

### 3.3 Response Parsing

| Futures Field | Maps To | Notes |
|---|---|---|
| `orderId` | `VenueOrderReceipt.VenueOrderID` | Numeric, stringified |
| `status` | Domain status via `mapBinanceStatus()` | `FILLED` -> `StatusFilled` |
| `avgPrice` | `FillRecord.Price` | Direct (no per-leg aggregation) |
| `executedQty` | `FillRecord.Quantity` + `FilledQuantity` | Same value in both fields |
| `cumQuote` | `FillRecord.Fee` | Fee proxy (G-4 acknowledged) |
| `updateTime` | `FillRecord.Timestamp` | Milliseconds -> `time.Time` UTC |

---

## 4. Lifecycle Proof

### 4.1 Dominant Path: submitted -> accepted -> filled

The Binance Futures API compresses the lifecycle into a single HTTP response: the venue accepts and fills the market order atomically, returning `status: "FILLED"`.

The canonical lifecycle validates this compressed path:

```
submitted -> accepted   (ValidTransition = true)
accepted  -> filled     (ValidTransition = true)
filled    -> *          (ValidTransition = false for all targets — terminal)
```

**Evidence**: `TestS422_FuturesConnectivity_DominantPath_ValidTransitions` proves all three assertions.

### 4.2 Binance Status Mapping

| Binance Status | Domain Status | Terminal? |
|---|---|---|
| `NEW` | `accepted` | No |
| `FILLED` | `filled` | Yes |
| `PARTIALLY_FILLED` | `partially_filled` | No |
| `CANCELED` / `CANCELLED` | `cancelled` | Yes |
| `REJECTED` | `rejected` | Yes |
| `EXPIRED` | `rejected` | Yes |

All mappings validated in `TestS416_FuturesLifecycleAlignment_BinanceStatusMapping` (prior wave, still passing).

### 4.3 Multi-Cycle Sustained Operation

`TestS422_FuturesConnectivity_MultiCycleSustained` proves 5 sequential order submissions with:
- Alternating BUY/SELL sides
- Unique timestamps producing unique ClientOrderIDs
- Unique VenueOrderIDs from the venue (no duplicates)
- CorrelationID preservation across all cycles
- Zero errors across all cycles

---

## 5. Segment Routing

### 5.1 Source-to-Segment Dispatch

| Source Prefix | Segment | Adapter |
|---|---|---|
| `binancef` | `futures` | `BinanceFuturesTestnetAdapter` |
| `binances` | `spot` | `BinanceSpotTestnetAdapter` |

The `SegmentRouter` dispatches based on `intent.Source`:
- `binancef` intents route to Futures adapter
- `binances` intents route to Spot adapter
- Unknown sources produce an error (fail-closed)

**Evidence**:
- `TestS422_SegmentRouter_FuturesRoutedCorrectly_SpotIsolated`: Futures called, Spot NOT called
- `TestS422_SegmentRouter_SourceMapping_Binancef`: Source mapping verified
- `TestS422_SegmentRouter_UnknownSource_FailsClosed`: Unknown source rejected

### 5.2 Segment Isolation

Under the unified runtime, both adapters are registered in a single `SegmentRouter`. A Futures intent (`source=binancef`) triggers ONLY the Futures adapter. The Spot adapter receives zero calls.

---

## 6. Canonical Surface Compliance

### 6.1 Config Surface

| Config | Shape | Validated |
|---|---|---|
| `execute-venue-live.jsonc` | Both segments enabled, `dry_run=false` | `TestS422_CanonicalConfig_VenueLive_FuturesEnabled` |
| `execute-unified.jsonc` | Both segments enabled, `dry_run=true` | `TestS422_CanonicalConfig_Unified_DryRunTrue` |

No new config files created. No per-segment configs. NG-41 and NG-50 respected.

### 6.2 Compose Surface

Proof execution uses: `docker-compose.yaml` + `docker-compose.venue-live.yaml`.
No new compose files created. NG-46, NG-47, NG-49 respected.

### 6.3 Runtime Topology

All frozen components unchanged:
- Execute binary with `SegmentRouter` dispatch
- NATS subjects unchanged
- KV bucket unchanged
- ClickHouse schema unchanged
- Lifecycle state machine (7 states) unchanged

---

## 7. Differences from Spot Proof

| Dimension | Spot (S405) | Futures (S422) |
|---|---|---|
| API path | `/api/v3/order` | `/fapi/v1/order` |
| Response type | `FULL` (per-leg fills array) | `RESULT` (top-level avgPrice) |
| Fill price | Weighted average from `fills[]` | Direct from `avgPrice` |
| Fee source | `commission` per fill leg | `cumQuote` (fee proxy, G-4) |
| Timestamp field | `transactTime` | `updateTime` |
| Source prefix | `binances` | `binancef` |

---

## 8. Test Evidence Summary

| Test | Governs | Result |
|---|---|---|
| `TestS422_FuturesConnectivity_DominantPath_ValidTransitions` | FV-Q1 | PASS |
| `TestS422_FuturesConnectivity_BuySide_FilledWithVenueOrderID` | FV-Q1 | PASS |
| `TestS422_FuturesConnectivity_SellSide_FilledCorrectly` | FV-Q1 | PASS |
| `TestS422_FuturesFillRecord_PriceFromAvgPrice` | FV-Q2 | PASS |
| `TestS422_FuturesFillRecord_TimestampFromUpdateTime` | FV-Q2 | PASS |
| `TestS422_FuturesCorrelation_ChainPreservedThroughVenue` | FV-Q11 | PASS |
| `TestS422_FuturesCorrelation_IntentFieldsPreservedAfterFill` | FV-Q11 | PASS |
| `TestS422_FuturesCorrelation_ClientOrderIDDeterministic` | FV-Q11 | PASS |
| `TestS422_FuturesReconciliation_QueryOrderRecoversFill` | FV-Q12 | PASS |
| `TestS422_FuturesReconciliation_QueryUsesCorrectFuturesPath` | FV-Q12 | PASS |
| `TestS422_FuturesConnectivity_MultiCycleSustained` | FV-Q1 | PASS |
| `TestS422_SegmentRouter_FuturesRoutedCorrectly_SpotIsolated` | FV-Q1 | PASS |
| `TestS422_SegmentRouter_SourceMapping_Binancef` | FV-Q1 | PASS |
| `TestS422_SegmentRouter_UnknownSource_FailsClosed` | FV-Q1 | PASS |
| `TestS422_CanonicalConfig_VenueLive_FuturesEnabled` | Surface | PASS |
| `TestS422_CanonicalConfig_Unified_DryRunTrue` | Surface | PASS |
| `TestS422_FuturesAPI_PathIsFapi` | FV-Q1 | PASS |
| `TestS422_FuturesAPI_RESULTResponseType` | FV-Q1 | PASS |
| `TestS422_FuturesAPI_HMACSigned` | FV-Q1 | PASS |

**19/19 PASS. Zero failures.**

---

## 9. Links

| Reference | Link |
|---|---|
| Alignment controls and limitations | [`futures-accepted-filled-real-response-alignment-controls-and-limitations.md`](futures-accepted-filled-real-response-alignment-controls-and-limitations.md) |
| Wave charter | [`futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md`](futures-venue-execution-proof-wave-charter-and-canonical-surface-contract.md) |
| Stage report | [`../stages/stage-s422-futures-real-venue-acceptance-fill-proof-report.md`](../stages/stage-s422-futures-real-venue-acceptance-fill-proof-report.md) |
| Test file | `internal/application/execution/s422_futures_venue_connectivity_fill_test.go` |
