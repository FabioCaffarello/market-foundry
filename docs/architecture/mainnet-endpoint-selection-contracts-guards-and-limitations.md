# Mainnet Endpoint Selection, Contracts, Guards, and Limitations

> Stage: S433 | Date: 2026-03-23 | Type: Contract and Guard Specification

---

## 1. Endpoint Selection Model

### 1.1 Selection Hierarchy

Adapter selection is config-driven and resolved at boot time:

```
config.venue.segments.{segment}.adapter  -->  VenueType constant  -->  Adapter constructor  -->  Boot wiring
```

The `buildVenueAdapterByType()` function in `cmd/execute/run.go` is the single dispatch point. It maps each `VenueType` to its concrete adapter constructor.

### 1.2 Canonical Endpoint Map

| VenueType | REST Base URL | WebSocket Base URL | API Path Prefix |
|---|---|---|---|
| `binance_spot_testnet` | `https://testnet.binance.vision` | `wss://testnet.binance.vision` | `/api/v3/` |
| `binance_spot_mainnet` | `https://api.binance.com` | `wss://stream.binance.com:9443` | `/api/v3/` |
| `binance_futures_testnet` | `https://testnet.binancefuture.com` | `wss://stream.binancefuture.com` | `/fapi/v1/` |
| `binance_futures_mainnet` | `https://fapi.binance.com` | `wss://fstream.binance.com` | `/fapi/v1/` |

WebSocket URLs are used by the ingest binary (not the execute binary). The execute binary only uses REST endpoints for order submission and query.

### 1.3 Environment Classification

Each `VenueType` carries an environment classification:

| VenueType | Segment | Environment |
|---|---|---|
| `binance_spot_testnet` | spot | testnet |
| `binance_spot_mainnet` | spot | mainnet |
| `binance_futures_testnet` | futures | testnet |
| `binance_futures_mainnet` | futures | mainnet |
| `paper_simulator` | -- | -- |

The `Environment()` and `IsMainnet()` methods on `VenueType` expose this classification for validation and logging.

---

## 2. Contracts

### 2.1 VenuePort Interface Contract

All adapters (testnet and mainnet) implement the same `ports.VenuePort` interface:

```go
type VenuePort interface {
    SubmitOrder(ctx context.Context, req VenueOrderRequest) (VenueOrderReceipt, *problem.Problem)
}
```

And optionally `ports.VenueQueryPort`:

```go
type VenueQueryPort interface {
    QueryOrder(ctx context.Context, clientOrderID, symbol string) (VenueOrderReceipt, *problem.Problem)
}
```

Mainnet adapters satisfy both interfaces identically to testnet adapters.

### 2.2 Response Contract

All Binance adapters (testnet and mainnet) share the same response parsing:

- **Spot**: `fills[]` array with per-leg `price`, `qty`, `commission`, `commissionAsset` -> aggregated into `FillRecord` with `Fee`, `FeeAsset`, `CostBasis`
- **Futures**: `avgPrice`, `executedQty`, `cumQuote` -> `FillRecord` with `Fee="0"`, `CostBasis=cumQuote`

This contract is unchanged from S428 fee normalization.

### 2.3 Error Classification Contract

All adapters share the same error classification chain:

1. Venue error code override (`classifyByVenueErrorCode`): -1001, -1003, -1015
2. HTTP status classification: 401/403 (auth), 429 (rate limit), 4xx (client), 502/503 (server)
3. Retryability annotation: `MarkRetryable()` for transient errors

This is unchanged between testnet and mainnet.

### 2.4 Credential Contract

| Credential | Env Var Pattern | Required |
|---|---|---|
| API Key | `MF_VENUE_{VENUE_TYPE}_API_KEY` | Yes |
| API Secret | `MF_VENUE_{VENUE_TYPE}_API_SECRET` | Yes |

Fail-closed: missing credentials cause boot failure (the adapter constructor is never reached).

---

## 3. Guards

### 3.1 Config Validation Guards

| Guard | Condition | Enforcement |
|---|---|---|
| Known venue type | `knownVenueTypes[adapter]` must be true | Boot-time rejection |
| Segment compatibility | `adapterSegmentCompatibility[adapter]` must match segment key | Boot-time rejection |
| Mainnet dry_run enforcement | `dry_run=false` rejected when any mainnet adapter is configured | Boot-time rejection |
| Paper simulator exclusion | `paper_simulator` cannot be a segment adapter | Boot-time rejection |
| Credential presence | All required env vars must be non-empty | Boot-time rejection |

### 3.2 Runtime Guards

| Guard | Layer | Behavior |
|---|---|---|
| DryRunSubmitter | Outermost decorator | Intercepts all venue calls; never delegates |
| Kill-switch | Actor layer | Halts all execution on gate close |
| Staleness guard | Actor layer | Rejects intents older than `staleness_max_age` |
| Rate limiter | Adapter decorator (mainnet only) | Blocks on token exhaustion; fails on context expiry |
| Context deadline | Per-request | 10s default deadline enforced if caller omits one |

### 3.3 Safety Invariants

1. **No mainnet adapter can submit real orders without explicit dry_run=false.** Config validation prevents this.
2. **DryRunSubmitter never delegates to the inner pipeline.** The inner adapter chain is fully composed but never called.
3. **Kill-switch is checked before every venue call.** The actor layer enforces this independent of the adapter.
4. **Rate limiter cannot be bypassed.** It wraps the raw adapter before any other decorator.

---

## 4. Testnet vs Mainnet Differences

### 4.1 Known Differences

| Dimension | Testnet | Mainnet |
|---|---|---|
| Base URL | `testnet.binance.vision` / `testnet.binancefuture.com` | `api.binance.com` / `fapi.binance.com` |
| Rate limits | Lenient (rarely enforced) | Strict (1200 weight/min for Spot, 2400 weight/min for Futures) |
| Fill behavior | Market orders fill atomically | Market orders may partially fill under low liquidity |
| Error codes | Same set, but testnet may return different error rates | Production error rates |
| Credential scope | Testnet API keys | Production API keys (separate key management required) |

### 4.2 Identical Behavior

| Dimension | Shared |
|---|---|
| Authentication scheme | HMAC-SHA256 with timestamp and recvWindow |
| API path structure | `/api/v3/order` (Spot), `/fapi/v1/order` (Futures) |
| Response JSON schema | Identical field names and types |
| Order types supported | MARKET (scope of this system) |
| Client order ID format | Same 36-char UUID convention |

### 4.3 Behavioral Risks on Mainnet

| Risk | Severity | Mitigation |
|---|---|---|
| Rate limit violations | Medium | Token-bucket rate limiter decorator (10 burst, ~10 req/s) |
| Different fill latency | Low | Context deadline (10s default) handles slow responses |
| Partial fills more likely | Low | Domain model handles `PARTIALLY_FILLED` status since S406 |
| Different error semantics | Low | Error classification covers all known Binance error codes |

---

## 5. Limitations

- **WebSocket mainnet URLs are documented but not wired.** Ingest binary changes for mainnet market data are a future stage concern.
- **Rate limiter parameters are fixed.** Configurable burst/refill would require new config fields (not needed for current single-symbol scope).
- **No mainnet-specific error handling.** The classification chain is shared. If mainnet introduces new error codes, they fall through to the default `Unavailable` classification (safe default).
- **Credential rotation is not automated.** The current `LoadCredentials()` reads env vars at boot. Mid-execution credential rotation requires a restart until S434 introduces `CredentialProvider`.
- **No mainnet smoke script.** The S436 mainnet dry-run proof stage will produce the first mainnet-targeted compose and smoke script.
