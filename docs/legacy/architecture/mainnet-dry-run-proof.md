# Mainnet Dry-Run Proof

> Stage: S436 | Date: 2026-03-24 | Phase: 49 (Mainnet Enablement)

## Objective

Prove that the market-foundry platform can operate against Binance mainnet endpoints (Spot and Futures) in strict dry-run mode, with zero risk of real order submission.

## Context

After S433 (mainnet adapters), S434 (secret manager / credential bootstrap), and S435 (ClickHouse backup/restore), all structural blockers for mainnet enablement are closed. S436 transforms abstract mainnet readiness into operationally verified dry-run proof against real mainnet endpoints.

## What Was Exercised

### Network Connectivity (MDR-1, MDR-2, MDR-3)

| Endpoint | Host | Ping Path | Proven |
|----------|------|-----------|--------|
| Spot Mainnet | `api.binance.com` | `/api/v3/ping` | DNS, TCP:443, TLS, HTTP 200 |
| Futures Mainnet | `fapi.binance.com` | `/fapi/v1/ping` | DNS, TCP:443, TLS, HTTP 200 |

Both endpoints resolve, accept TCP connections on port 443, complete TLS handshake with valid certificate chains (TLS 1.2+), and return HTTP 200 on their public `/ping` paths.

### DryRunSubmitter Interception (MDR-4, MDR-5)

The DryRunSubmitter decorator wraps the real mainnet adapter and short-circuits all SubmitOrder calls. The inner adapter (pointing to `api.binance.com` or `fapi.binance.com`) is **never called**.

Proven for both segments:
- Spot: `BinanceSpotMainnetAdapter` → `DryRunSubmitter` → intercepted
- Futures: `BinanceFuturesMainnetAdapter` → `DryRunSubmitter` → intercepted

### Audit Trail Markers (MDR-6)

Every dry-run receipt carries:
- `VenueOrderID` with `dryrun-` prefix (16-char hex suffix)
- `Simulated: true` on every fill record
- `Fee: "0"` on every fill (no real commission)
- `StatusFilled` for active intents, `StatusAccepted` for noop intents

These markers are deterministic and cannot be confused with real venue fills.

### Credential Format Validation (MDR-7)

Mainnet credential loading enforces:
- Minimum 16 character length (rejects truncated or placeholder values)
- No whitespace (rejects copy-paste errors)
- Format validation applies to mainnet adapters only (testnet is unrestricted)

### Pipeline Chain Composition (MDR-8)

Full mainnet pipeline proven:
```
BinanceSpotMainnetAdapter → RateLimiter(10 tokens, 100ms) → DryRunSubmitter
```

Multiple sequential intents all intercepted by DryRunSubmitter. The RateLimiter and inner adapter are composed but never exercised for real API calls.

## Endpoint Selection Proof

| Adapter | Base URL | API Path | Config Key |
|---------|----------|----------|------------|
| `BinanceSpotMainnetAdapter` | `https://api.binance.com` | `/api/v3/order` | `binance_spot_mainnet` |
| `BinanceFuturesMainnetAdapter` | `https://fapi.binance.com` | `/fapi/v1/order` | `binance_futures_mainnet` |

Endpoint selection is a compile-time constant (`binanceSpotMainnetBaseURL`, `binanceFuturesMainnetBaseURL`). There is no runtime URL resolution or dynamic endpoint switching.

## Config Enforcement

The config validation layer (S433) enforces:

| Rule | Effect |
|------|--------|
| `dry_run=false` + mainnet adapter | **Rejected** at config validation |
| `dry_run=true` + mainnet adapter | Accepted |
| `dry_run` omitted (nil) + mainnet adapter | Accepted (defaults to `true`, fail-closed) |

This means no config combination can reach mainnet with `dry_run=false` without modifying the validation code.

## DryRunSubmitter Interception Chain

```
Intent arrives
  → VenueAdapterActor.onIntent()
    → safetyGate.Check() (kill switch + staleness)
      → DryRunSubmitter.SubmitOrder()
        → generates dryrun-{hex} VenueOrderID
        → returns StatusFilled with Simulated=true fills
        → inner adapter is NEVER called
```

The inner adapter (`BinanceSpotMainnetAdapter` or `BinanceFuturesMainnetAdapter`) is retained in the pipeline for structural completeness but its `SubmitOrder` method is never invoked.

## What Was NOT Proven

| Gap | Reason | Risk |
|-----|--------|------|
| Authenticated API call to mainnet | DryRunSubmitter intercepts before any HTTP call | LOW — adapter code is identical to testnet, proven in S405/S416 |
| Rate limiter under mainnet load | DryRunSubmitter intercepts before RateLimiter is exercised | LOW — RateLimiter proven in unit tests |
| Extended soak against mainnet | Out of scope for S436 | ACCEPTED — soak is a future authorization ceremony |
| Real order fill on mainnet | Explicitly prohibited by design | BY DESIGN — requires future authorization ceremony |
| Multi-exchange beyond Binance | Not in scope | ACCEPTED — single exchange focus |

## Limitations

1. **No authenticated mainnet API exercise**: DryRunSubmitter prevents any HTTP call to mainnet. Credential correctness against mainnet auth is not proven (only format validation).
2. **Rate limiter dormant**: The RateLimiter decorator is composed but never exercises its token bucket against mainnet. It was proven in isolation in S433 unit tests.
3. **No soak or endurance**: This proof is a point-in-time connectivity and interception check, not a sustained run.
4. **Certificate expiry**: TLS certificate validity is checked at proof time. Certificates rotate independently.
