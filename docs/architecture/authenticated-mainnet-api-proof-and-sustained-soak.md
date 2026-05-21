# Authenticated Mainnet API Proof and Sustained Soak

**Stage:** S441
**Status:** Complete
**Date:** 2026-03-24

## Purpose

This document describes the authenticated mainnet API proof and sustained soak executed in S441. The goal is to demonstrate that the platform can operate with valid Binance mainnet credentials, select correct endpoints, maintain fail-closed controls, and sustain stable authenticated operation over a defined time window — all without submitting any real order.

## Authenticated API Surface

S441 introduces `AccountStatus()` methods on both Spot and Futures adapters. These call authenticated, read-only Binance endpoints:

| Segment | Endpoint | Method | Auth | Effect |
|---------|----------|--------|------|--------|
| Spot | `GET /api/v3/account` | HMAC-SHA256 | API Key + Secret | Read-only: returns balances, permissions |
| Futures | `GET /fapi/v2/account` | HMAC-SHA256 | API Key + Secret | Read-only: returns account, positions |

These endpoints were chosen because:
- They are GET requests with zero write side effects
- They require full HMAC-SHA256 authentication (proves credential validity)
- They return meaningful account metadata (proves the account exists and is accessible)
- They are the minimal authenticated surface — no symbol parameter, no order state

## HMAC-SHA256 Signing

Both adapters use the same signing mechanism inherited from the testnet adapter:

1. Build query parameters: `timestamp` (current ms) + `recvWindow` (5000ms)
2. Compute HMAC-SHA256 of the query string using the API secret
3. Append `signature` parameter to the query string
4. Set `X-MBX-APIKEY` header with the API key
5. Execute GET request

This is identical to the signing used for `SubmitOrder()` and `QueryOrder()`, proving that the same credential resolution and signing pipeline works for mainnet.

## Endpoint Selection

Endpoint selection is determined by the adapter constructor:

| Adapter | Base URL | Set By |
|---------|----------|--------|
| `BinanceSpotMainnetAdapter` | `https://api.binance.com` | `NewBinanceSpotMainnetAdapter()` |
| `BinanceFuturesMainnetAdapter` | `https://fapi.binance.com` | `NewBinanceFuturesMainnetAdapter()` |
| `BinanceSpotTestnetAdapter` | `https://testnet.binance.vision` | `NewBinanceSpotTestnetAdapter()` |
| `BinanceFuturesTestnetAdapter` | `https://testnet.binancefuture.com` | `NewBinanceFuturesTestnetAdapter()` |

Mainnet adapters are type aliases of testnet adapters with the base URL overridden. This ensures identical request construction, signing, response parsing, and error classification — the only variable is the endpoint.

## Credential Resolution

Credentials are resolved via the `CredentialProvider` interface (S434):

1. **Environment provider** (default): `MF_VENUE_{VENUE_TYPE}_{KEY}` environment variables
2. **File provider** (S439): Mounted secret files at `{basePath}/{venue_type}/{KEY}`

For mainnet, format validation is applied (S434):
- Minimum 16 characters (catches truncated pastes)
- No whitespace (catches copy-paste errors)

Preflight check (`MainnetCredentialCheck`) fails fast if any mainnet credential is missing at startup.

## Sustained Soak Design

The soak test (AMP-5) executes repeated authenticated calls to both Spot and Futures endpoints over a configurable time window:

| Parameter | Default | Override |
|-----------|---------|----------|
| Duration | 5 minutes | `MF_SOAK_DURATION` env var |
| Interval | 15 seconds | Fixed (4 calls/min/segment) |
| Rate | ~8 calls/min total | Well within Binance 1200 req/min limit |
| Tolerance | 5% failure rate | Network jitter allowance |

Metrics collected during soak:
- Per-segment success/failure counts
- Maximum latency per segment
- Failure rate calculation

## DryRunSubmitter Invariant

Throughout all tests, `DryRunSubmitter` remains the outermost decorator in the pipeline:

```
rawAdapter → RateLimiter → DryRunSubmitter
```

The soak stability test (AMP-6) interleaves authenticated API calls with DryRunSubmitter submissions to prove that:
- DryRunSubmitter interception is 100% reliable (zero escapes)
- All VenueOrderIDs carry the `dryrun-` prefix
- All fills have `Simulated=true`
- No real order is ever submitted

## Test Matrix

| ID | Test | What It Proves |
|----|------|----------------|
| AMP-1 | Spot AccountStatus | HMAC signing + credential validity + endpoint selection (Spot) |
| AMP-2 | Futures AccountStatus | HMAC signing + credential validity + endpoint selection (Futures) |
| AMP-3 | DryRun after auth | DryRunSubmitter intact after real authenticated call |
| AMP-4 | Pipeline chain | Full pipeline (adapter → RL → DRS) with authenticated adapters |
| AMP-5 | Sustained soak | Stability over 5-minute window, both segments |
| AMP-6 | Soak DRS stability | DryRunSubmitter 100% reliable throughout soak |

## Authorization Conditions Closed

This proof closes two of the six conditions from the S437 evidence gate:

- **C-1: Authenticated mainnet API call proven** — AMP-1 and AMP-2 demonstrate authenticated calls returning HTTP 200 with valid account data
- **C-4: Sustained mainnet soak** — AMP-5 demonstrates stable operation over the defined soak window

## Limitations

1. **AccountStatus() is not in the order path** — it proves credential/endpoint validity but does not exercise SubmitOrder against mainnet (which remains behind DryRunSubmitter)
2. **Read-only API keys sufficient** — the proof does not require trading permissions, so it cannot prove that the same keys would work for order submission
3. **Soak window is configurable** — the default 5-minute window is sufficient for proof but not a production endurance test
4. **Single-host execution** — the soak runs from the test runner, not from the containerized stack
5. **No WebSocket authenticated streams** — only REST authenticated calls are proven
