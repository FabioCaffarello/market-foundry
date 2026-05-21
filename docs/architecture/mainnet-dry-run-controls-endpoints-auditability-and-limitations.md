# Mainnet Dry-Run Controls, Endpoints, Auditability, and Limitations

> Stage: S436 | Date: 2026-03-24 | Phase: 49 (Mainnet Enablement)

## Fail-Closed Semantics Chain

The platform enforces dry-run mode for mainnet through five independent layers. Any single layer is sufficient to prevent real order submission; all five are active simultaneously.

### Layer 1: Config Validation (compile-time equivalent)

```
VenueConfig.Validate() rejects dry_run=false when any adapter IsMainnet()
```

- **Guard**: `VenueConfig.hasMainnetAdapter()` check in `Validate()`
- **Error**: `"dry_run=false is not authorized for mainnet adapters"`
- **Bypass**: Requires modifying Go source code in `schema.go`

### Layer 2: Fail-Closed Default

```
VenueConfig.IsDryRun() returns true when DryRun is nil
```

- **Guard**: Nil-pointer check on `*bool` field
- **Effect**: Omitting `dry_run` from config always activates dry-run
- **Bypass**: Requires explicitly setting `"dry_run": false` in config (caught by Layer 1)

### Layer 3: DryRunSubmitter Wrapper (runtime)

```
DryRunSubmitter.SubmitOrder() intercepts all calls, inner adapter never invoked
```

- **Guard**: The function body generates synthetic receipts without calling `inner.SubmitOrder()`
- **Markers**: `dryrun-` prefix on VenueOrderID, `Simulated: true` on all fills
- **Bypass**: Requires removing DryRunSubmitter from the decorator chain in `cmd/execute/run.go`

### Layer 4: Preflight Credential Check (startup)

```
bootstrap.MainnetCredentialCheck() fails-fast if credentials are missing or malformed
```

- **Guard**: Checks all mainnet-configured segments for `API_KEY` and `API_SECRET`
- **Format**: Minimum 16 chars, no whitespace (mainnet only)
- **Effect**: Process exits before any actor starts if credentials are absent

### Layer 5: Safety Gate (runtime)

```
VenueAdapterActor.safetyGate.Check() enforces kill switch and staleness
```

- **Guard**: Independent of dry-run — additional runtime safety layer
- **Effect**: Rejects intents even if all other layers failed (defense in depth)

## Endpoint Map

| Adapter | Base URL | Order Path | Query Path | Auth |
|---------|----------|------------|------------|------|
| `binance_spot_mainnet` | `https://api.binance.com` | `POST /api/v3/order` | `GET /api/v3/order` | HMAC-SHA256 |
| `binance_futures_mainnet` | `https://fapi.binance.com` | `POST /fapi/v1/order` | `GET /fapi/v1/order` | HMAC-SHA256 |
| `binance_spot_testnet` | `https://testnet.binance.vision` | `POST /api/v3/order` | `GET /api/v3/order` | HMAC-SHA256 |
| `binance_futures_testnet` | `https://testnet.binancefuture.com` | `POST /fapi/v1/order` | `GET /fapi/v1/order` | HMAC-SHA256 |

Public health endpoints (no auth required):

| Segment | Ping | Time |
|---------|------|------|
| Spot Mainnet | `GET /api/v3/ping` | `GET /api/v3/time` |
| Futures Mainnet | `GET /fapi/v1/ping` | `GET /fapi/v1/time` |

## Credential Flow

```
Environment / Secret Manager
  → CredentialProvider.Resolve("binance_spot_mainnet", "API_KEY")
    → EnvCredentialProvider reads MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY
      → LoadCredentials() validates: non-empty, min-length 16, no whitespace (mainnet only)
        → CredentialSet stored in adapter (apiKey, apiSecret fields)
          → DryRunSubmitter intercepts before adapter uses credentials
```

Credentials are loaded and validated at startup but never used in HTTP calls during dry-run. This validates the credential resolution path without exposing credentials to the network.

## Audit Trail Markers

### VenueOrderID Format

| Mode | Prefix | Example |
|------|--------|---------|
| Dry-run | `dryrun-` | `dryrun-a1b2c3d4e5f67890a1b2c3d4e5f67890` |
| Paper | `paper-` | `paper-a1b2c3d4e5f67890a1b2c3d4e5f67890` |
| Real venue | Numeric | `123456789` (Binance order ID) |

### Fill Record Markers

| Field | Dry-Run Value | Real Venue Value |
|-------|--------------|-----------------|
| `Simulated` | `true` | `false` |
| `Fee` | `"0"` | Actual commission |
| `Price` | PriceSource lookup or `"0"` | Venue fill price |
| `Quantity` | Intent quantity (full fill) | Actual fill quantity |

### Structured Logging

DryRunSubmitter emits a structured log entry for every intercepted intent:

```json
{
  "level": "INFO",
  "component": "dry-run-submitter",
  "msg": "dry-run intercepted",
  "source": "binances",
  "symbol": "btcusdt",
  "timeframe": 60,
  "side": "buy",
  "quantity": "0.001",
  "correlation_id": "...",
  "venue_order_id": "dryrun-..."
}
```

### Health Counters

| Counter | Description |
|---------|-------------|
| `dryrun_intercepted` | Total intents intercepted by DryRunSubmitter |
| `dryrun_filled` | Active intents given synthetic fills |
| `dryrun_noop` | No-action intents (Side=none) passed through |

## Pipeline Decorator Chain

```
                        Mainnet Dry-Run Pipeline
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  BinanceSpotMainnetAdapter (https://api.binance.com)        │
│       │                                                     │
│       ▼                                                     │
│  RateLimiter (10 tokens, 100ms refill) ← mainnet only      │
│       │                                                     │
│       ▼                                                     │
│  RetrySubmitter (backoff, deadline, halt check)             │
│       │                                                     │
│       ▼                                                     │
│  Post200Reconciler (body-read recovery via QueryOrder)      │
│       │                                                     │
│       ▼                                                     │
│  ██████████████████████████████████████████████████████████  │
│  █  DryRunSubmitter ← ALL CALLS INTERCEPTED HERE        █  │
│  █  Inner pipeline NEVER executes                       █  │
│  ██████████████████████████████████████████████████████████  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

The DryRunSubmitter is the **outermost** decorator. It receives the intent first and returns a synthetic receipt without delegating to any inner decorator.

## Compose Configuration

File: `deploy/configs/execute-mainnet-dry-run.jsonc`

```jsonc
{
  "venue": {
    "dry_run": true,          // ← explicit enforcement
    "segments": {
      "spot":    { "enabled": true, "adapter": "binance_spot_mainnet" },
      "futures": { "enabled": true, "adapter": "binance_futures_mainnet" }
    }
  }
}
```

File: `deploy/compose/docker-compose.mainnet-dry-run.yaml`

Overrides the execute service to mount `execute-mainnet-dry-run.jsonc` and pass mainnet credentials via environment variables (resolved from host or `.env`).

## Limitations

1. **Credentials validated but not network-tested**: Format validation confirms credential shape (length, no whitespace) but does not prove they are accepted by Binance mainnet auth. This gap closes when dry-run enforcement is lifted in a future authorization ceremony.

2. **Rate limiter dormant**: The RateLimiter is composed in the pipeline but never exercises its token bucket against mainnet because DryRunSubmitter intercepts first. Rate limit behavior was proven in S433 unit tests.

3. **No real fill data**: All fills are synthetic (`Simulated: true`, `Fee: "0"`). Price realism depends on PriceSource availability (NATS last-price lookup). Without PriceSource, fill price defaults to `"0"`.

4. **TLS certificate rotation**: Certificate validity is proven at test time. Binance may rotate certificates independently; this is not monitored by the platform.

5. **No soak or endurance**: This proof validates a point-in-time snapshot. Sustained mainnet dry-run operation is not proven in S436.

6. **Single exchange**: Only Binance Spot and Futures mainnet endpoints are exercised. No other exchanges are in scope.
