# Real Venue Activation Smoke

> S342: Proves activation lifecycle with the real BinanceFuturesTestnetAdapter.

## Purpose

S341 proved that gate transitions control real event flow through the actor pipeline — but only with the paper simulator. This document defines the verification strategy for exercising the same lifecycle with the real HTTP-based venue adapter.

## Strategy

An `httptest.Server` simulates the Binance Futures testnet API. This exercises the **real** adapter code path:

- HMAC-SHA256 request signing
- Query parameter encoding (symbol mapping, client order ID derivation)
- HTTP request construction with API key headers
- Response parsing (order ID, status, fills, timestamps)
- Fill extraction with `Simulated=false`
- Error classification (auth, rate limit, venue rejection)

The simulated server returns realistic Binance Futures API responses, allowing the tests to validate the complete decorator pipeline (RetrySubmitter + Post200Reconciler + BinanceFuturesTestnetAdapter) without requiring live testnet credentials or network access.

## What Changes vs Paper Adapter

| Dimension | Paper (S341) | Real Adapter (S342) |
|-----------|-------------|---------------------|
| Adapter type | PaperVenueAdapter | BinanceFuturesTestnetAdapter |
| HTTP requests | None | Real HTTP to httptest.Server |
| Fill.Simulated | true | false |
| VenueOrderID format | "paper-{nano}" | Numeric ID from venue JSON |
| Fill record fields | Synthetic price/qty | Parsed from venue response |
| Signing | N/A | HMAC-SHA256 on every request |
| Activation dimensions | AdapterPaper, CredentialAbsent | AdapterVenue, CredentialPresent |
| VenueQuery capability | nil | Available (adapter implements both ports) |
| Decorator pipeline | RetrySubmitter only | RetrySubmitter + Post200Reconciler |

## Verification Scenarios

### RVA-1: Halted Gate Blocks Real Venue Path

**Precondition**: Gate=halted, real adapter wired.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Publish event | Event reaches actor (processed counter) |
| 2 | Verify gate check | skipped_halt incremented, filled=0 |
| 3 | Verify HTTP | **Zero** HTTP requests to venue server |

**Evidence**: halted gate prevents any contact with the real venue HTTP endpoint.

### RVA-2: Gate Open Enables Real Venue Flow

**Precondition**: Gate=active, real adapter wired.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Publish event | Event flows through full pipeline |
| 2 | Verify fill | VenueOrderFilledEvent received with Simulated=false |
| 3 | Verify HTTP | >= 1 HTTP request to venue server |
| 4 | Verify fields | Price, quantity, venue_order_id from venue JSON |
| 5 | Verify correlation | CorrelationID preserved end-to-end |

### RVA-3: Runtime Halt Blocks After Enable

**Precondition**: Gate=active → halted during running supervisor.

| Step | Action | Expected |
|------|--------|----------|
| 1 | Publish event (active) | Fill received with Simulated=false |
| 2 | Halt gate via KV PUT | Gate transitions to halted |
| 3 | Publish event (halted) | Event blocked, filled count unchanged |
| 4 | Verify HTTP | Zero additional venue HTTP requests after halt |

### RVA-4: Full Lifecycle (halted -> enabled -> halted)

Three phases on a single running supervisor with real venue adapter:

1. **Halted**: event blocked, zero venue HTTP requests
2. **Enabled**: real venue fill (Simulated=false, venue HTTP request made)
3. **Re-halted**: event blocked, zero additional venue HTTP requests

### RVA-5: Venue Rejection Does Not Produce Fill

**Precondition**: Gate=active, venue returns HTTP 400 (margin insufficient).

| Step | Action | Expected |
|------|--------|----------|
| 1 | Publish event | Event reaches venue (HTTP request made) |
| 2 | Venue rejects | Error recorded in tracker |
| 3 | Verify no fill | filled=0, error_count >= 1 |

**Evidence**: venue HTTP errors are classified correctly and do not produce spurious fills.

### RVA-6: Activation Surface Dimensions for Venue Adapter

Domain-level verification that the three venue states compute correctly:

- venue + halted + present = `venue_halted` (CanReachVenue=true, IsLive=false)
- venue + active + present = `venue_live` (IsLive=true)
- venue + active + absent = `venue_degraded` (IsLive=false)

## Test Infrastructure

- Reuses S333 NATS helpers (`s333NatsURL`, `s333BuildEvent`, `s333FillSubscriber`)
- Reuses S341 gate helpers (`s341SetGate`, `s341WaitCounter`)
- Adds `s342VenueServer` — simulated Binance Futures API with request counting
- Adds `s342RejectionServer` — simulated venue that rejects all orders
- Adds `s342SpawnSupervisor` — wires real adapter with `WithActivationState(AdapterVenue, CredentialPresent)`

## Integration with Smoke Script

Phase 7 of `scripts/smoke-activation.sh` runs all `TestRealVenueActivation_*` tests when NATS is available. Entry: `make smoke-activation`.

## Guard Rails

- No mainnet adapter or configuration
- No multi-venue wiring
- No OMS integration
- No extended operation (tests run in seconds)
- httptest.Server is ephemeral and isolated per test
