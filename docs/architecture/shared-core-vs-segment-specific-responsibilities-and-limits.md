# Shared Core vs Segment-Specific Responsibilities and Limits

**Stage:** S392 — Adapter Boundary Split
**Status:** Canonical
**Authority:** This document defines what is shared between Binance adapters and what is segment-specific.
**Depends on:** [Adapter Boundary Split](adapter-boundary-split-for-binance-spot-and-binance-futures.md)

---

## 1. Guiding Principle

**Share infrastructure, not behavior.** The shared core contains Binance exchange-level mechanics that are identical across all market segments. Segment-specific code owns everything that differs or could diverge between Spot and Futures. When in doubt, keep it segment-specific — extraction is cheaper than disentanglement.

---

## 2. Shared Core (`binance_common.go`)

### 2.1 Components

| Component | Responsibility | Why Shared |
|-----------|---------------|------------|
| `BinanceClient` struct | Holds base URL, API key, API secret, HTTP client | All Binance segments use the same credential shape and HTTP transport |
| `Sign(payload) string` | HMAC-SHA256 request signing | Signing algorithm is exchange-level, identical across all Binance endpoints |
| `Do(ctx, method, path, params)` | HTTP request execution with deadline enforcement (EC-3) | Transport mechanics, header injection (`X-MBX-APIKEY`), body reading, and size limits are exchange-wide |
| `HandleErrorResponse(statusCode, body)` | HTTP status → `problem.Problem` classification (C-FAIL taxonomy) | Error code semantics are exchange-level (401=auth, 429=rate-limit, 5xx=server) |
| `ClassifyByVenueErrorCode(statusCode, venueCode, details)` | Venue error code overrides (-1001, -1003, -1015) | Binance error codes are exchange-wide, not segment-specific |
| `MapBinanceStatus(status) (Status, *Problem)` | Binance order status → domain `Status` mapping | Order lifecycle statuses (FILLED, REJECTED, etc.) are identical across segments |
| `MapSymbol(symbol) string` | Lowercase → uppercase normalization | Symbol convention is exchange-wide |
| `binanceErrorResponse` struct | `{code, msg}` error shape | Same JSON error envelope across all Binance APIs |
| `defaultRequestDeadline` const | 10s fallback deadline | Uniform deadline policy for all Binance calls |

### 2.2 Limits on Shared Core

The shared core MUST NOT contain:
- Response parsing logic (response shapes differ between Spot and Futures)
- Order submission parameter construction (paths, parameter names may diverge)
- Base URL constants (each segment has its own endpoint family)
- API path constants (`/fapi/v1/` vs `/api/v3/`)
- Fill record construction (different source fields)
- Segment-specific response structs

### 2.3 Extension Rules

To add a function or type to the shared core, ALL of these must hold:
1. The behavior is **proven identical** across Spot and Futures (not just similar)
2. The function operates at **exchange level**, not segment level
3. Adding it reduces real duplication, not just superficial similarity
4. The shared surface does not need segment-aware branching (`if spot then...`)

If rule 4 would be violated, the function stays segment-specific even if 90% of the logic is the same. Clean duplication beats conditional sharing.

---

## 3. Segment-Specific: Binance Futures

### 3.1 Components

| Component | Responsibility |
|-----------|---------------|
| `BinanceFuturesTestnetAdapter` struct | Adapter entry point, holds `*BinanceClient` |
| `NewBinanceFuturesTestnetAdapter(creds, timeout)` | Constructor with Futures-specific base URL |
| `SubmitOrder(ctx, req)` | Builds Futures-specific request params, calls `client.Do`, parses Futures response |
| `QueryOrder(ctx, clientOrderID, symbol)` | Builds Futures-specific query params, calls `client.Do`, parses Futures response |
| `parseOrderResponse(body, intent)` | Futures-specific response → `VenueOrderReceipt` mapping |
| `binanceFuturesOrderResponse` struct | Futures JSON response shape (`avgPrice`, `cumQuote`, `updateTime`) |
| Base URL: `https://testnet.binancefuture.com` | Futures testnet endpoint |
| API path: `/fapi/v1/order` | Futures order API |

### 3.2 Futures-Specific Behaviors

- **Average price** is a top-level response field (`avgPrice`)
- **Fee proxy** is `cumQuote` (cumulative quote quantity)
- **Fill timestamp** comes from `updateTime` (millisecond epoch)
- **Single fill record** per response (Futures market orders fill atomically on testnet)

---

## 4. Segment-Specific: Binance Spot

### 4.1 Components

| Component | Responsibility |
|-----------|---------------|
| `BinanceSpotTestnetAdapter` struct | Adapter entry point, holds `*BinanceClient` |
| `NewBinanceSpotTestnetAdapter(creds, timeout)` | Constructor with Spot-specific base URL |
| `SubmitOrder(ctx, req)` | Builds Spot-specific request params, calls `client.Do`, parses Spot response |
| `QueryOrder(ctx, clientOrderID, symbol)` | Builds Spot-specific query params, calls `client.Do`, parses Spot response |
| `parseOrderResponse(body, intent)` | Spot-specific response → `VenueOrderReceipt` mapping |
| `binanceSpotOrderResponse` struct | Spot JSON response shape (`fills[]`, `cummulativeQuoteQty`, `transactTime`) |
| Base URL: `https://testnet.binance.vision` | Spot testnet endpoint |
| API path: `/api/v3/order` | Spot order API |

### 4.2 Spot-Specific Behaviors

- **No `avgPrice` field** — price is derived from `fills[].price` (weighted average if multiple legs)
- **Explicit `fills` array** with per-leg price, quantity, commission, and commission asset
- **Commission** is explicit per fill (`commission` + `commissionAsset`), not a proxy
- **Fill timestamp** comes from `transactTime` (millisecond epoch)
- **Multiple fill records** possible per response (Spot market orders may fill across multiple price levels)

---

## 5. Responsibility Matrix

| Concern | Owner | Rationale |
|---------|-------|-----------|
| HMAC-SHA256 signing | **Shared** | Exchange-wide authentication protocol |
| HTTP transport + deadline | **Shared** | Uniform transport mechanics |
| Error classification (HTTP) | **Shared** | Exchange-wide HTTP status semantics |
| Error classification (venue code) | **Shared** | Exchange-wide error code space |
| Status mapping | **Shared** | Same order lifecycle across segments |
| Symbol normalization | **Shared** | Same convention across segments |
| Credential loading | **Shared** (via `CredentialSet`) | Already segment-agnostic by design |
| Base URL | **Segment** | Different endpoint families |
| API paths | **Segment** | `/fapi/v1/` vs `/api/v3/` |
| Request param construction | **Segment** | Same today, may diverge (e.g., Spot margin params) |
| Response struct | **Segment** | Structurally different JSON shapes |
| Response parsing | **Segment** | Different field extraction logic |
| Fill record construction | **Segment** | Different source fields, multiplicity |
| No-action intent handling | **Segment** | Identical today, but owned per-adapter for independence |

---

## 6. Trade-offs

### 6.1 Accepted Duplication

The following code appears in both adapters with identical logic:
- No-action intent early return (`Side == none → noop receipt`)
- Request parameter building skeleton (`symbol`, `side`, `type`, `quantity`, `newOrderRespType`, `newClientOrderId`, `timestamp`, `recvWindow`)
- `WithBaseURL` test helper

**Why accepted:** These are 5–10 lines each. Extracting them would require passing segment-specific constants through a shared function, creating coupling without meaningful deduplication. The duplication is stable (these patterns haven't changed since S308) and trivially verifiable.

### 6.2 Rejected Alternatives

| Alternative | Why Rejected |
|-------------|--------------|
| **Single adapter with segment flag** | Conditional branches in response parsing reduce readability; testing requires segment-aware mocking; Spot-specific features (multi-fill, explicit commission) would complicate Futures code paths |
| **Interface-based strategy pattern** | Over-engineering for two concrete implementations; the dispatch is already handled by the factory and VenueType config; adding an interface layer adds indirection without testability gain |
| **Full duplication (no shared core)** | ~120 lines of identical signing + error classification code would diverge over time; bug fixes would need to be applied twice; the shared surface is genuinely exchange-level |
| **Generic `BinanceAdapter` with config struct** | Config struct would encode segment differences that are better expressed as separate types; response parsing cannot be cleanly parameterized without losing type safety |

### 6.3 Evolution Considerations

- **Mainnet adapters** (future): Same split pattern. `BinanceFuturesMainnetAdapter` and `BinanceSpotMainnetAdapter` would share the same `BinanceClient`, differing only in base URL (mainnet endpoints). No shared core changes needed.
- **New order types** (e.g., limit orders): Would be added per-segment since order parameters and response handling may differ.
- **Second exchange** (out of scope): Would get its own shared core (`kraken_common.go`) — the Binance shared core is NOT a generic exchange abstraction.

---

## 7. Shared Core Boundary Rules

1. **No segment constants in shared core** — base URLs, API paths, and source values belong to segment files
2. **No response types in shared core** — only error response types (which are exchange-wide)
3. **No adapter constructors in shared core** — each segment owns its constructor
4. **Shared functions must be stateless or client-scoped** — no global state, no segment awareness
5. **Tests for shared core test infrastructure only** — adapter integration tests belong to segment test files
6. **Shared core changes require both adapter test suites to pass** — this is the regression gate
