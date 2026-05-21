# End-to-End Venue Integration Proof

**Stage:** S316
**Status:** Delivered
**Date:** 2026-03-21
**Predecessor:** S315 (Foundational Tranche Gate — PASS WITH RESIDUALS)

---

## 1. Purpose

This document records the first real venue integration proof for market-foundry. After the adapter hardening tranche (S312–S315) delivered a hardened adapter with deterministic client order IDs, error classification, and retryability semantics, S316 proves that the complete venue path works end-to-end against Binance Futures testnet.

The scope is intentionally minimal: one venue, one order type (market), synchronous fills only, testnet only.

## 2. What Was Proved

### 2.1 Submit Path (VQ1)

A market order was successfully submitted to Binance Futures testnet via `BinanceFuturesTestnetAdapter.SubmitOrder`. The adapter:

- Mapped the internal lowercase symbol (`btcusdt`) to Binance convention (`BTCUSDT`).
- Signed the request with HMAC-SHA256 using credentials loaded from environment.
- Sent an HTTP POST to `/fapi/v1/order` with `newOrderRespType=RESULT`.
- Received a venue order ID from the testnet.

Both BUY and SELL sides were exercised.

### 2.2 Fill Path (VQ3)

The testnet returned a `FILLED` status with real market data:

- **Price**: Non-zero, venue-determined average fill price.
- **Quantity**: Matched the requested quantity (`0.001`).
- **Fee proxy**: `cumQuote` field captured (real commission requires separate endpoint).
- **Simulated flag**: `false` — confirming this is a real venue fill, not a paper simulation.
- **Timestamp**: Derived from Binance `updateTime` field (millisecond precision).

### 2.3 Persistence Compatibility (VQ4)

The `VenueOrderReceipt` returned by the adapter was validated for structural compatibility with the persistence layer:

| Field | Requirement | Status |
|-------|-------------|--------|
| `VenueOrderID` | Non-empty string | PASS |
| `ClientOrderID` | Matches `ClientOrderID(intent)` derivation | PASS |
| `CorrelationID` | Preserved from input intent | PASS |
| `CausationID` | Preserved from input intent | PASS |
| `PartitionKey()` | Returns `{source}.{symbol}.{timeframe}` | PASS |
| `DeduplicationKey()` | Returns deterministic key for JetStream | PASS |
| JSON round-trip | Marshal → Unmarshal preserves all fields | PASS |

### 2.4 Composite Read Compatibility

The receipt's `ExecutionIntent` carries all fields required by `ExecutionWithTrace` in the composite read model:

- `Type`, `Source`, `Symbol`, `Timeframe` — present and valid.
- `Side`, `Quantity`, `FilledQuantity`, `Status` — populated by adapter from venue response.
- `Risk` (nested) — preserved from input intent.
- `Fills[]` — populated with real venue data.
- `CorrelationID`, `CausationID` — preserved for composite chain reconstruction.

No schema changes were needed. The existing ClickHouse `executions` table schema accommodates real venue data without modification.

### 2.5 Safety Gate (VQ6)

The safety gate was validated on the venue path:

| Scenario | Expected | Result |
|----------|----------|--------|
| Fresh intent, no kill switch | Allowed → venue submit succeeds | PASS |
| Stale intent (5 min old, 2 min max) | Blocked (reason: `stale`) | PASS |
| Kill switch halted | Blocked (reason: `kill_switch`) | PASS |
| Both stale + halted | Blocked (reason: `kill_switch`, priority) | PASS |
| No-action intent (side=none) | Accepted without venue HTTP call | PASS |

## 3. What Was NOT Proved (Guard Rails)

| Excluded Scope | Reason |
|----------------|--------|
| Async fills / websocket | S316 scope freeze — synchronous only |
| Advanced order types (limit, stop) | Market orders only per charter |
| Mainnet | Testnet only — no real funds |
| Multiple venues | Single venue (Binance Futures testnet) |
| Full persistence round-trip (NATS → ClickHouse → HTTP query) | Requires running stack; structural compatibility proved instead |
| Retry infrastructure | Deferred to post-tranche (RT-1 through RT-7) |

## 4. Test Inventory

| Test | VQ | File |
|------|----|------|
| `TestS316_VQ1_SubmitMarketBuy_RealTestnet` | VQ1 | `venue_integration_e2e_test.go` |
| `TestS316_VQ1_SubmitMarketSell_RealTestnet` | VQ1 | `venue_integration_e2e_test.go` |
| `TestS316_VQ3_RealFill_NotSimulated` | VQ3 | `venue_integration_e2e_test.go` |
| `TestS316_VQ4_ReceiptPersistenceCompatibility` | VQ4 | `venue_integration_e2e_test.go` |
| `TestS316_VQ6_SafetyGate_FreshIntent_AllowsVenueSubmit` | VQ6 | `venue_integration_e2e_test.go` |
| `TestS316_VQ6_SafetyGate_StaleIntent_BlocksVenueSubmit` | VQ6 | `venue_integration_e2e_test.go` |
| `TestS316_VQ6_SafetyGate_KillSwitch_BlocksVenueSubmit` | VQ6 | `venue_integration_e2e_test.go` |
| `TestS316_VQ6_SafetyGate_KillSwitchPriority` | VQ6 | `venue_integration_e2e_test.go` |
| `TestS316_E2E_ActorPath_GateToSubmitToReceipt` | VQ1+VQ3+VQ4+VQ6 | `venue_integration_e2e_test.go` |
| `TestS316_NoAction_NoVenueCall_RealAdapter` | — | `venue_integration_e2e_test.go` |
| `TestS316_ClientOrderID_DeterministicWithRealVenue` | VQ1 | `venue_integration_e2e_test.go` |

## 5. Credential Guard

Tests requiring real venue interaction are guarded by `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` and `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET`. When absent, tests skip with a clear message. Safety gate tests (VQ6 blocking scenarios) run without credentials.

## 6. Smoke Script

`scripts/smoke-venue-integration.sh` provides an operational entrypoint:

```bash
# Full proof (requires credentials):
export MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY=<key>
export MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET=<secret>
./scripts/smoke-venue-integration.sh

# Dry run (safety gate tests only):
./scripts/smoke-venue-integration.sh --dry-run
```

## 7. Residuals Closed

This stage closes two residuals from the tranche gate (S315):

| Residual | Description | Closure |
|----------|-------------|---------|
| R-S313-1 | Real venue acceptance untested | Closed: VQ1 proves real testnet acceptance |
| R-S314-1 | No real Binance error corpus | Partially closed: real testnet interactions produce real error shapes |
