# Mainnet Adapter Readiness -- Spot and Futures

> Stage: S433 | Date: 2026-03-23 | Type: Architecture and Implementation Record

---

## 1. Summary

S433 introduces structural readiness for Binance mainnet adapters (Spot and Futures) into the Foundry. Both adapters are now registered, config-validated, boot-wired, and tested. They are always deployed behind `DryRunSubmitter` -- mainnet live trading requires a separate authorization ceremony.

This stage resolves blocker B-1 (no mainnet adapter implementation) identified in the S430 mainnet readiness audit.

---

## 2. Architecture

### 2.1 Adapter Model

The mainnet adapters follow the exact same pattern as the proven testnet adapters:

| Adapter | Base URL | API Path (Spot) | API Path (Futures) | Auth |
|---|---|---|---|---|
| `binance_spot_testnet` | `testnet.binance.vision` | `/api/v3/order` | -- | HMAC-SHA256 |
| `binance_spot_mainnet` | `api.binance.com` | `/api/v3/order` | -- | HMAC-SHA256 |
| `binance_futures_testnet` | `testnet.binancefuture.com` | -- | `/fapi/v1/order` | HMAC-SHA256 |
| `binance_futures_mainnet` | `fapi.binance.com` | -- | `/fapi/v1/order` | HMAC-SHA256 |

### 2.2 Implementation Strategy

Mainnet adapters are **type aliases** of their testnet counterparts with only the base URL overridden:

```go
type BinanceSpotMainnetAdapter = BinanceSpotTestnetAdapter
type BinanceFuturesMainnetAdapter = BinanceFuturesTestnetAdapter
```

This ensures:
- Zero code duplication -- all request construction, signing, response parsing, error classification, and fee normalization logic is shared.
- Testnet proof directly transfers to mainnet -- same contract, same behavior.
- Bug fixes apply to both environments simultaneously.

The only runtime difference is the base URL constant injected at construction time.

### 2.3 Rate Limiter

A token-bucket `RateLimiter` decorator is inserted between the raw adapter and higher-level decorators for mainnet adapters:

```
rawAdapter -> RateLimiter -> RetrySubmitter -> Post200Reconciler -> DryRunSubmitter
```

Configuration:
- Bucket capacity: 10 tokens (burst)
- Refill rate: 1 token per 100ms (~10 req/s steady state)
- Context-aware: blocks until token available or context expires
- Fail-safe: returns `Unavailable` problem on context expiry

The rate limiter is applied only to mainnet adapters in `cmd/execute/run.go`. Testnet adapters continue without rate limiting (Binance testnet has lenient limits).

### 2.4 Safety Controls

The 4-layer safety defense is preserved and extended for mainnet:

| Layer | Control | Mainnet Behavior |
|---|---|---|
| 1 | `dry_run` config flag | **Enforced true** for mainnet adapters via config validation (S433) |
| 2 | `DryRunSubmitter` decorator | Intercepts all venue calls; never delegates to inner adapter |
| 3 | Kill-switch (`EXECUTION_CONTROL` KV) | Global halt gate -- unchanged |
| 4 | Staleness guard | Rejects stale intents -- unchanged |

**New enforcement (S433):** Config validation rejects `dry_run=false` when any mainnet adapter is configured. This is a compile-time-equivalent guard -- the binary refuses to start with an invalid config.

---

## 3. Config Integration

### 3.1 New VenueType Constants

```go
VenueTypeBinanceSpotMainnet    VenueType = "binance_spot_mainnet"
VenueTypeBinanceFuturesMainnet VenueType = "binance_futures_mainnet"
```

Both are registered in `knownVenueTypes` and `adapterSegmentCompatibility`.

### 3.2 VenueType Helpers

New methods on `VenueType`:

| Method | Returns | Purpose |
|---|---|---|
| `Environment()` | `"testnet"`, `"mainnet"`, or `""` | Classify adapter by environment |
| `IsMainnet()` | `bool` | Quick mainnet check for validation |

### 3.3 Config Examples

Single-segment mainnet (Spot only):
```jsonc
{
  "venue": {
    "dry_run": true,  // or omit (defaults to true)
    "segments": {
      "spot": { "enabled": true, "adapter": "binance_spot_mainnet" }
    }
  }
}
```

Dual-segment mainnet:
```jsonc
{
  "venue": {
    "segments": {
      "spot": { "enabled": true, "adapter": "binance_spot_mainnet" },
      "futures": { "enabled": true, "adapter": "binance_futures_mainnet" }
    }
  }
}
```

Mixed environment (testnet Spot, mainnet Futures):
```jsonc
{
  "venue": {
    "segments": {
      "spot": { "enabled": true, "adapter": "binance_spot_testnet" },
      "futures": { "enabled": true, "adapter": "binance_futures_mainnet" }
    }
  }
}
```

### 3.4 Credential Convention

Mainnet credentials follow the existing `MF_VENUE_` prefix convention:

| Adapter | API_KEY env var | API_SECRET env var |
|---|---|---|
| `binance_spot_mainnet` | `MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY` | `MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET` |
| `binance_futures_mainnet` | `MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY` | `MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET` |

Credentials are loaded via `LoadCredentials()` using the standard env-var pattern. The S434 secret manager integration will introduce a `CredentialProvider` interface as an alternative to raw env vars.

---

## 4. Files Changed

| File | Change |
|---|---|
| `internal/shared/settings/schema.go` | Added mainnet VenueType constants, Environment/IsMainnet methods, hasMainnetAdapter validation, dry_run enforcement |
| `internal/application/execution/binance_spot_mainnet_adapter.go` | New: Spot mainnet adapter (type alias + base URL override) |
| `internal/application/execution/binance_futures_mainnet_adapter.go` | New: Futures mainnet adapter (type alias + base URL override) |
| `internal/application/execution/rate_limiter.go` | New: Token-bucket rate limiter VenuePort decorator |
| `cmd/execute/run.go` | Added mainnet adapter cases in buildVenueAdapterByType with rate limiter wiring |
| `internal/shared/settings/s433_mainnet_adapter_config_test.go` | New: 10 config validation tests |
| `internal/application/execution/s433_mainnet_adapter_readiness_test.go` | New: 10 adapter + rate limiter + credential tests |

---

## 5. Limitations

- Mainnet adapters are structurally ready but have not been tested against real mainnet endpoints. The S436 mainnet dry-run proof will validate actual connectivity.
- The rate limiter uses fixed parameters (10 burst, 100ms refill). Configurable rate limits are a future enhancement if operational needs diverge from defaults.
- Credential loading still uses environment variables. The S434 secret manager integration will provide a secure alternative.
- The `dry_run=false` enforcement for mainnet is a config validation guard, not a runtime guard. A future live-trading authorization ceremony must explicitly remove this enforcement.
