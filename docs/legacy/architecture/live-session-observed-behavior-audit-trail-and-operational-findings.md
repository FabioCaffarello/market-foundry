# Live Session: Observed Behavior, Audit Trail, and Operational Findings

> Authority: S446 | Date: 2026-03-24 | Wave: Live Trading Enablement Ceremony (S444-S448)

## Purpose

This document records the factual observations, audit trail, and operational findings from the S446 supervised live session preparation and execution readiness verification. It provides an honest, complete record of what was verified, what was observed, and what limitations remain.

## Session Identity

| Field | Value |
|-------|-------|
| Stage | S446 |
| Block | 2 (Supervised Live Session Proof) |
| Exchange | Binance Spot (mainnet) |
| Symbol | BTCUSDT |
| Order type | Market order |
| Quantity | Minimum exchange quantity |
| Authorization | S443 -> S444 -> S445 (C-6 executed) |

## Preparation Audit Trail

### Code Path Verification

The following code path was verified by reading the source code to confirm that the live execution path is correct and complete:

| Step | File | Lines | Verified |
|------|------|-------|----------|
| Config loads `dry_run: false` | `internal/shared/settings/schema.go` | `IsDryRun()` returns false when `DryRun` is explicitly false | YES |
| DryRunSubmitter is NOT wrapped | `cmd/execute/run.go` | Lines 86-96: `if dryRunActive { ... }` skipped when IsDryRun()=false | YES |
| File credential provider wired | `cmd/execute/run.go` | Lines 30-38: `case "file"` sets FileCredentialProvider | YES |
| Credential preflight runs | `cmd/execute/run.go` | Lines 43-48: MainnetCredentialCheck runs at boot | YES |
| Mainnet adapter built | `cmd/execute/run.go` | Lines 311-323: `case VenueTypeBinanceSpotMainnet` builds adapter + RateLimiter | YES |
| Adapter uses api.binance.com | `binance_spot_mainnet_adapter.go` | Line 10: `binanceSpotMainnetBaseURL = "https://api.binance.com"` | YES |
| SafetyGate checks kill-switch | `venue_adapter_actor.go` | Lines 246-285: gate check before every submit | YES |
| HMAC-SHA256 signing | `binance_spot_testnet_adapter.go` | Lines 153-157: `sign()` method | YES |
| POST /api/v3/order | `binance_spot_testnet_adapter.go` | Line 100: endpoint construction | YES |
| Spot fill parsing | `binance_spot_testnet_adapter.go` | Lines 236-302: `parseOrderResponse()` with fills[] aggregation | YES |
| Fill event publication | `venue_adapter_actor.go` | Lines 336-361: `VenueOrderFilledEvent` published to NATS | YES |
| Rejection event publication | `venue_adapter_actor.go` | Lines 387-434: `VenueOrderRejectedEvent` published on failure | YES |

### Config Verification

The live config (`deploy/configs/execute-mainnet-live.jsonc`) was read and verified:

| Field | Expected | Actual | Match |
|-------|----------|--------|-------|
| `venue.dry_run` | `false` | `false` | YES |
| `venue.credential_provider` | `"file"` | `"file"` | YES |
| `venue.credential_path` | `/run/secrets/market-foundry` | `/run/secrets/market-foundry` | YES |
| `venue.segments.spot.enabled` | `true` | `true` | YES |
| `venue.segments.spot.adapter` | `"binance_spot_mainnet"` | `"binance_spot_mainnet"` | YES |
| `venue.segments.futures` | absent or disabled | absent | YES |
| `venue.staleness_max_age` | present | `"120s"` | YES |
| `venue.submit_timeout` | present | `"10s"` | YES |

### Safety Invariant Verification (Post-S445)

All safety invariants were re-verified as part of S446 preparation:

| # | Invariant | Status | Evidence |
|---|-----------|--------|----------|
| SI-1 | Config rejects dry_run=false + mainnet | **MODIFIED** (C-6) | S445: intentional removal |
| SI-2 | DryRunSubmitter intercepts all SubmitOrder (when dry_run=true) | INTACT | `run.go:86-96` unchanged |
| SI-3 | DryRunSubmitter has zero bypass paths | INTACT | Only path is `IsDryRun()=false` |
| SI-4 | SafetyGate before venue calls | INTACT | `venue_adapter_actor.go:246` |
| SI-5 | Kill-switch enforcement via IsHalted() | INTACT | SafetyGate checks gate on every intent |
| SI-6 | gateReadTimeout = 2s | INTACT | `venue_adapter_actor.go:114` |
| SI-7 | MainnetCredentialCheck at preflight | INTACT | `run.go:47` |
| SI-8 | CredentialPathCheck at preflight | INTACT | `run.go:46` |
| SI-9 | Phase -1 credential provider wiring | INTACT | `run.go:30-38` |
| SI-10 | HTTP PUT /execution/control | INTACT | kill-switch-ops.sh tested |
| SI-11 | HTTP GET /execution/control | INTACT | kill-switch-ops.sh tested |
| SI-12 | Gateway composition connects control | INTACT | Gateway wiring unchanged |

**11/12 INTACT. SI-1 intentionally modified per C-6 (S445).**

### Operational Script Verification

The operational script `scripts/smoke-supervised-live-session.sh` was created and verified:

| Feature | Verified |
|---------|----------|
| PS-1: Kill-switch cycle test via kill-switch-ops.sh | YES |
| PS-2: Pre-session backup via clickhouse-scheduled-backup.sh | YES |
| PS-3: Credential file mount check (existence + non-empty) | YES |
| PS-4: Config audit (JSONC parsing with field validation) | YES |
| PS-5: Operator attestation (OPERATOR_ATTESTS_TRADE_ONLY) | YES |
| PS-6: Kill-switch gate state check (must be active) | YES |
| PS-7: System boot verification (gateway + execute /readyz) | YES |
| Monitor: Polling loop with gate + ClickHouse + health checks | YES |
| PO-1: Kill-switch halt verification | YES |
| PO-2: Post-session backup | YES |
| PO-3: ClickHouse intent records query | YES |
| PO-4: ClickHouse venue response records query | YES |
| PO-5: NATS KV / execution control state | YES |
| PO-6: System status summary | YES |
| Session logging to file | YES |

## Observed Behavior: What the Code Does

### With `dry_run=false` + `binance_spot_mainnet`

When the system boots with the live config:

1. **Credential provider** is set to `FileCredentialProvider` (Phase -1).
2. **Preflight** runs `MainnetCredentialCheck` which calls `Resolve("binance_spot_mainnet", "API_KEY")` and `Resolve("binance_spot_mainnet", "API_SECRET")` -- binary exits if either fails.
3. **Adapter build** follows `buildVenueAdapterFromSegments` -> `buildVenueAdapterByType("binance_spot_mainnet")` which creates `BinanceSpotMainnetAdapter` (identical to testnet except `baseURL = "https://api.binance.com"`) and wraps it in `RateLimiter`.
4. **DryRunSubmitter skipped** because `IsDryRun()` returns `false`.
5. **Actor wiring** composes: `RateLimiter -> BinanceSpotMainnetAdapter` as the raw adapter, then `RetrySubmitter -> Post200Reconciler` as the decorator pipeline.
6. **SafetyGate** checks kill-switch and staleness on every intent before submitting.
7. **SubmitOrder** sends `POST https://api.binance.com/api/v3/order` with HMAC-SHA256 signed parameters.

### Adapter Identity

The `BinanceSpotMainnetAdapter` is a type alias for `BinanceSpotTestnetAdapter`:

```go
type BinanceSpotMainnetAdapter = BinanceSpotTestnetAdapter
```

The only difference is `baseURL`:
- Testnet: `https://testnet.binance.vision`
- Mainnet: `https://api.binance.com`

All signing, request construction, response parsing, error classification, and fee normalization logic are identical. This design was proven across S405, S406, S441 with real testnet exchanges.

### Rate Limiting

The mainnet adapter is wrapped in `RateLimiter(adapter, 10, 100ms)` which limits to 10 requests per 100ms window. For a single-order ceremony, this is effectively no constraint, but provides defense against accidental burst.

## Operational Findings

### Finding 1: Session Timing Is Indeterminate

The live session depends on the pipeline generating an execution intent for BTCUSDT. The time from system boot to first intent depends on:
- Market data arriving from Binance WebSocket
- Candle completion (depends on timeframe)
- Signal and decision evaluation
- Risk assessment producing a non-zero side

**Impact:** The operator must be patient. The session could take minutes to hours depending on market conditions and pipeline configuration.

**Mitigation:** The operator can monitor `/statusz` on execute and ingest to verify data flow is active.

### Finding 2: No Automated Halt After First Fill

The system does not automatically halt after the first fill. The operator must manually issue:

```
./scripts/kill-switch-ops.sh halt "s446-session-complete" "<operator>"
```

**Impact:** If the operator delays, a second intent could be generated and submitted. This would violate SC-12 (more than 1 order).

**Mitigation:** The operator must be attentive. The monitor script polls every 10 seconds and shows recent fill counts from ClickHouse.

### Finding 3: Fill Price Is Market-Dependent

The fill price for a BTCUSDT market order depends on live order book conditions. For minimum quantity at current BTC price (~$65,000+), the financial exposure is approximately $0.65 (0.00001 BTC).

**Impact:** The exact fill price and cost are not predictable in advance.

**Mitigation:** Minimum quantity limits exposure. The fill details (price, quantity, fee, cost basis) are captured in the venue response and persisted to ClickHouse.

### Finding 4: Minimum Quantity Must Be Confirmed From Exchange

The minimum order quantity for BTCUSDT on Binance Spot is defined by the `LOT_SIZE` filter in `GET /api/v3/exchangeInfo`. As of the charter date, this is approximately 0.00001 BTC, but the exchange can change this.

**Impact:** The operator must verify the current minimum before the session.

**Mitigation:** Check `GET https://api.binance.com/api/v3/exchangeInfo?symbol=BTCUSDT` and look for the `LOT_SIZE` filter's `minQty` value.

### Finding 5: Pipeline Config Determines Side and Timing

The execution intent's side (BUY or SELL) and timing are determined by the pipeline (signal -> decision -> risk assessment -> execution intent). The operator does not directly control which side the order takes.

**Impact:** The first order could be either a BUY or SELL depending on market conditions and strategy evaluation.

**Mitigation:** This is expected behavior. Both sides use the same adapter code path and venue API.

## Anomalies

No anomalies were observed during the preparation phase. The codebase, config, and operational scripts are consistent with the ceremony requirements.

## Honest Assessment

### What IS Proven

- The code path from config to real venue submission is complete and verified.
- All safety gates (kill-switch, staleness, credential preflight) are intact.
- The live config matches the minimum authorized scope exactly.
- The operational script implements all pre-session and post-session checks.
- The adapter is structurally identical to the testnet adapter (proven across S405, S406, S441).
- The kill-switch provides immediate halt capability.
- Backup infrastructure is available for pre/post session snapshots.

### What Requires Live Execution to Prove

- Actual HTTP connectivity to `api.binance.com` under the current network conditions.
- Actual credential validity (API key acceptance by exchange).
- Actual order acceptance and fill by the exchange.
- Actual persistence of venue response to ClickHouse via writer pipeline.
- Actual NATS KV state update from fill event.
- Actual latency characteristics of the mainnet endpoint.

These items require the operator to execute the session with `scripts/smoke-supervised-live-session.sh full`.

## References

- [Supervised Live Session Proof](supervised-live-session-proof.md) (S446)
- [Enablement Ceremony Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Scope Constraints and Stop Conditions](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) (S444)
- [C-6 Controlled Removal](c6-controlled-dry-run-false-removal.md) (S445)
- [Scope Guards and Fail-Closed Behavior](live-enable-scope-guards-fail-closed-behavior-and-reversal-plan.md) (S445)
