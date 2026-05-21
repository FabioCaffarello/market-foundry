# Stage S441: Authenticated Mainnet Proof — Report

**Wave:** Live Trading Authorization
**Status:** Complete
**Date:** 2026-03-24
**Depends on:** S439 (External Secret Manager), S440 (Automated Backup)
**Enables:** S442 (Kill-Switch Operational Runbook)

## Objective

Execute and validate an authenticated proof against the Binance mainnet API (Spot + Futures), with sustained soak, demonstrating that the stack operates with valid credentials, correct endpoint selection, fail-closed controls, and stability — without submitting any real order.

## Capabilities Delivered

| # | Capability | Rating | Evidence |
|---|-----------|--------|----------|
| 1 | Authenticated Spot mainnet API call | FULL | AMP-1: GET /api/v3/account returns HTTP 200 with valid account data |
| 2 | Authenticated Futures mainnet API call | FULL | AMP-2: GET /fapi/v2/account returns HTTP 200 with valid account data |
| 3 | HMAC-SHA256 signing against mainnet | FULL | AMP-1, AMP-2: signatures accepted by production Binance |
| 4 | Credential resolution for mainnet | FULL | AMP-1, AMP-2: env-based credential provider resolves valid keys |
| 5 | Endpoint selection correctness | FULL | AMP-1 routes to api.binance.com, AMP-2 routes to fapi.binance.com |
| 6 | DryRunSubmitter integrity after auth | FULL | AMP-3: interception 100% after real authenticated call |
| 7 | Pipeline chain with auth adapters | FULL | AMP-4: adapter -> RateLimiter -> DryRunSubmitter, both segments |
| 8 | Sustained authenticated soak (Spot) | FULL | AMP-5: 5-minute soak within 5% failure tolerance |
| 9 | Sustained authenticated soak (Futures) | FULL | AMP-5: 5-minute soak within 5% failure tolerance |
| 10 | DryRunSubmitter soak stability | FULL | AMP-6: 100% interception reliability throughout soak |
| 11 | Zero order submission | FULL | No test calls SubmitOrder on raw mainnet adapter |

**Rating:** 11/11 capabilities at FULL.

## What Was Built

### Code Changes

1. **`BinanceSpotTestnetAdapter.AccountStatus()`** — Authenticated read-only call to `GET /api/v3/account`. Returns `AccountInfo` with canTrade, accountType, balanceCount. HMAC-SHA256 signed.

2. **`BinanceFuturesTestnetAdapter.AccountStatus()`** — Authenticated read-only call to `GET /fapi/v2/account`. Returns `FuturesAccountInfo` with canTrade, feeTier, assetCount, positionCount. HMAC-SHA256 signed.

3. **Types:** `AccountInfo`, `FuturesAccountInfo` — Minimal response structs for authenticated proofs.

### Tests

- **`s441_authenticated_mainnet_proof_test.go`** — 6 test functions (AMP-1 through AMP-6) under `livemainnet` build tag. Requires real mainnet credentials and network access.

### Scripts

- **`scripts/smoke-authenticated-mainnet-soak.sh`** — 4-phase smoke script. Supports `--quick` (30s soak), `--skip-soak` (auth proof only). Validates credentials before starting.

### Architecture Documents

- **`authenticated-mainnet-api-proof-and-sustained-soak.md`** — Proof methodology, API surface, signing, soak design, authorization conditions.
- **`mainnet-authenticated-soak-controls-stability-and-limitations.md`** — Safety controls, stability characteristics, failure modes, audit trail, limitations.

## Files Changed

| File | Change |
|------|--------|
| `internal/application/execution/binance_spot_testnet_adapter.go` | Added `AccountStatus()`, `AccountInfo` type |
| `internal/application/execution/binance_futures_testnet_adapter.go` | Added `AccountStatus()`, `FuturesAccountInfo` type |
| `internal/application/execution/s441_authenticated_mainnet_proof_test.go` | New: 6 test functions (AMP-1 through AMP-6) |
| `scripts/smoke-authenticated-mainnet-soak.sh` | New: 4-phase smoke script |
| `docs/architecture/authenticated-mainnet-api-proof-and-sustained-soak.md` | New: proof methodology doc |
| `docs/architecture/mainnet-authenticated-soak-controls-stability-and-limitations.md` | New: controls and limitations doc |
| `docs/stages/stage-s441-authenticated-mainnet-proof-report.md` | New: this report |

## Authorization Conditions

S437 established six conditions for live trading authorization. S441 closes two:

| Condition | Status | Closed By |
|-----------|--------|-----------|
| C-1: Authenticated mainnet API call proven | **CLOSED** | S441 (AMP-1, AMP-2) |
| C-2: External secret manager deployed | CLOSED | S439 |
| C-3: Automated off-host backup | CLOSED | S440 |
| C-4: Sustained mainnet soak | **CLOSED** | S441 (AMP-5) |
| C-5: Kill-switch operational runbook | OPEN | Deferred to S442 |
| C-6: Explicit removal of dry_run=false rejection | OPEN | Requires authorization ceremony |

After S441: **4/6 conditions closed**, 2 remaining.

## Safety Invariants Verified

1. **DryRunSubmitter intercepts 100% of order submissions** — AMP-3, AMP-6
2. **Config validation rejects dry_run=false + mainnet** — unchanged from S433/S436
3. **Preflight fails fast on missing credentials** — unchanged from S434
4. **Rate limiter prevents burst violations** — AMP-4 pipeline composition
5. **AccountStatus() is GET-only** — no write side effects possible

## Limitations

1. AccountStatus() proves credential/endpoint validity but does not exercise SubmitOrder against mainnet
2. Read-only API keys are sufficient — cannot prove trading permission
3. Soak window is configurable (default 5m, not a production endurance test)
4. Single-host execution (not containerized stack)
5. No WebSocket authenticated streams covered
6. Credential rotation during soak not tested

## Next Step

**S442: Kill-Switch Operational Runbook** — the remaining authorization condition before the explicit dry_run=false removal ceremony.

## Verdict

**COMPLETE.** Authenticated mainnet proof delivered with sustained soak. Authorization conditions C-1 and C-4 are closable. The wave is ready for the kill-switch operational runbook in S442.
