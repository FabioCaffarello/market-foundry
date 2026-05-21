# Adapter Boundary Split for Binance Spot and Binance Futures

**Stage:** S392 — Adapter Boundary Split
**Status:** Canonical
**Authority:** This document defines the adapter-level boundary between Binance Spot and Binance Futures.
**Depends on:** S391 (Venue Model Refactor)

---

## 1. Problem Statement

The current codebase has a single Binance adapter: `BinanceFuturesTestnetAdapter`. Adding Spot support requires a clear architectural decision: should Spot be a branch inside the existing adapter, a separate adapter sharing common infrastructure, or a fully independent implementation?

**Decision: Separate adapters with extracted shared core.**

Spot and Futures are distinct Binance products with different REST endpoints, response schemas, and operational semantics. Collapsing them behind a single adapter with `if segment == spot` branches would degrade readability, testability, and independent evolution. Full duplication would waste the significant shared surface (signing, error classification, status mapping). The split extracts genuinely common Binance infrastructure into a shared module while giving each segment its own adapter file, constructor, and response parser.

---

## 2. Boundary Definition

### 2.1 File Layout (Target State)

```
internal/application/execution/
├── binance_common.go                          # Shared Binance infrastructure
├── binance_common_test.go                     # Tests for shared infrastructure
├── binance_futures_testnet_adapter.go          # Futures-specific adapter (refactored)
├── binance_futures_testnet_adapter_test.go     # Futures adapter tests (existing)
├── binance_spot_testnet_adapter.go             # Spot-specific adapter (new)
├── binance_spot_testnet_adapter_test.go        # Spot adapter tests (new)
├── credentials.go                             # Credential loading (unchanged)
├── paper_venue_adapter.go                     # Paper simulator (unchanged)
└── ...
```

### 2.2 Adapter Identity

| Adapter | VenueType Constant | Base URL | API Path |
|---------|-------------------|----------|----------|
| `BinanceFuturesTestnetAdapter` | `binance_futures_testnet` | `https://testnet.binancefuture.com` | `/fapi/v1/order` |
| `BinanceSpotTestnetAdapter` | `binance_spot_testnet` | `https://testnet.binance.vision` | `/api/v3/order` |

### 2.3 Port Compliance

Both adapters MUST implement:
- `ports.VenuePort` — `SubmitOrder(ctx, req) (VenueOrderReceipt, *problem.Problem)`
- `ports.VenueQueryPort` — `QueryOrder(ctx, clientOrderID, symbol) (VenueOrderReceipt, *problem.Problem)`

Both adapters plug into the same pipeline:
```
rawAdapter → RetrySubmitter → Post200Reconciler → DryRunSubmitter
```

No changes to the pipeline composition, actor layer, or decorator chain.

---

## 3. Adapter Struct Design

### 3.1 Futures Adapter (Refactored)

```go
type BinanceFuturesTestnetAdapter struct {
    client *BinanceClient  // shared infrastructure (extracted)
}
```

After extraction, the Futures adapter delegates signing, HTTP execution, and error classification to `BinanceClient`, keeping only:
- Futures-specific base URL and path constants
- Futures-specific response parsing (`binanceFuturesOrderResponse`)
- Constructor wiring

### 3.2 Spot Adapter (New)

```go
type BinanceSpotTestnetAdapter struct {
    client *BinanceClient  // same shared infrastructure
}
```

Spot adapter owns:
- Spot-specific base URL and path constants
- Spot-specific response parsing (`binanceSpotOrderResponse`)
- Constructor wiring

### 3.3 Shared Client (Extracted)

```go
type BinanceClient struct {
    baseURL    string
    apiKey     string
    apiSecret  string
    httpClient *http.Client
}
```

`BinanceClient` provides:
- `Sign(payload) string` — HMAC-SHA256 signing
- `Do(ctx, method, path, params) (statusCode int, body []byte, *problem.Problem)` — HTTP execution with deadline enforcement
- `HandleErrorResponse(statusCode, body) (VenueOrderReceipt, *problem.Problem)` — error classification
- `ClassifyByVenueErrorCode(statusCode, venueCode, details) (*problem.Problem, bool)` — venue-code overrides

---

## 4. Response Schema Differences

### 4.1 Futures Response (Current)

```json
{
  "orderId": 12345,
  "clientOrderId": "mf-xxx",
  "symbol": "BTCUSDT",
  "status": "FILLED",
  "avgPrice": "50000.00",
  "executedQty": "0.001",
  "cumQuote": "50.00",
  "updateTime": 1700000000000
}
```

Average price and executed quantity are top-level fields. Fee proxy is `cumQuote`.

### 4.2 Spot Response

```json
{
  "orderId": 12345,
  "clientOrderId": "mf-xxx",
  "symbol": "BTCUSDT",
  "status": "FILLED",
  "executedQty": "0.001",
  "cummulativeQuoteQty": "50.00",
  "fills": [
    { "price": "50000.00", "qty": "0.001", "commission": "0.05", "commissionAsset": "USDT" }
  ]
}
```

Key differences:
- **No `avgPrice` field** — price comes from `fills[].price` array
- **`cummulativeQuoteQty`** instead of `cumQuote`
- **Explicit `fills` array** with individual fill legs and commission details
- **No `updateTime`** — Spot uses `transactTime`

Each adapter owns its response struct and `parseOrderResponse` implementation. This is the primary reason the adapters cannot be collapsed into one.

---

## 5. Registration and Factory

### 5.1 Settings Schema Addition

```go
const (
    VenueTypePaperSimulator         VenueType = "paper_simulator"
    VenueTypeBinanceFuturesTestnet  VenueType = "binance_futures_testnet"
    VenueTypeBinanceSpotTestnet     VenueType = "binance_spot_testnet"     // NEW
)

var knownVenueTypes = map[VenueType]bool{
    VenueTypePaperSimulator:        true,
    VenueTypeBinanceFuturesTestnet: true,
    VenueTypeBinanceSpotTestnet:    true,  // NEW
}
```

### 5.2 Factory Branch Addition

```go
func buildVenueAdapter(config settings.AppConfig) (venueAdapterResult, error) {
    switch config.Venue.Type {
    // ... existing cases ...

    case settings.VenueTypeBinanceSpotTestnet:
        creds, prob := appexec.LoadCredentials(string(config.Venue.Type), []string{"API_KEY", "API_SECRET"})
        if prob != nil {
            return venueAdapterResult{}, fmt.Errorf(...)
        }
        adapter := appexec.NewBinanceSpotTestnetAdapter(creds, config.Venue.SubmitTimeoutDuration())
        return venueAdapterResult{submit: adapter, query: adapter, credentialState: domainexec.CredentialPresent}, nil
    }
}
```

### 5.3 Credential Isolation

- Futures: `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY`, `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET`
- Spot: `MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY`, `MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET`

Each adapter loads credentials under its own `VenueType` prefix. No cross-contamination.

### 5.4 NATS Source Values

- Futures: `binancef` (existing)
- Spot: `binances` (new)

Source values partition NATS subject trees for stream isolation between segments.

---

## 6. Invariants Preserved

| Invariant | Mechanism |
|-----------|-----------|
| Fail-closed on unknown type | `knownVenueTypes` gate + factory default branch |
| `dry_run=true` default | `VenueConfig.IsDryRun()` unchanged — applies to all types |
| Credentials never logged | `CredentialSet` contract unchanged |
| Kill switch in actor layer | No adapter changes — safety checks are upstream |
| Per-request deadline (EC-3) | `BinanceClient.Do` enforces deadline |
| Post-200 reconciliation | Both adapters implement `VenueQueryPort` |

---

## 7. Migration Strategy

The refactor is **additive and non-breaking**:

1. **Extract** `binance_common.go` from `binance_futures_testnet_adapter.go` — move `sign`, `handleErrorResponse`, `classifyByVenueErrorCode`, `mapBinanceStatus`, `mapSymbol`, shared types
2. **Refactor** `BinanceFuturesTestnetAdapter` to use `BinanceClient` — behavioral parity verified by existing tests
3. **Add** `BinanceSpotTestnetAdapter` using `BinanceClient` with Spot-specific response parsing
4. **Register** `binance_spot_testnet` in settings schema and factory
5. **Add** unit tests for Spot adapter at same coverage level as Futures

No existing tests, configs, or runtime behavior changes. Futures adapter behavior is verified by its existing test suite passing without modification after the extraction.

---

## 8. Non-Goals

- Multi-exchange support (only Binance in scope)
- Mainnet adapter activation (registered in enum, not implemented)
- Generic adapter plugin framework
- WebSocket or streaming adapters
- Adapter auto-discovery or registry patterns
