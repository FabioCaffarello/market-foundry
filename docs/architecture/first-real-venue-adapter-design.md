# First Real Venue Adapter Design

## Venue Selection

**Selected:** Binance Futures Testnet (`binance_futures_testnet`)

### Justification

1. **Natural fit:** the codebase already uses `"binancef"` as the canonical source for Binance Futures data. The execution domain was designed with this venue in mind.
2. **Testnet availability:** Binance Futures provides a full-featured testnet at `testnet.binancefuture.com` that mirrors production API semantics without capital exposure.
3. **Simple REST API:** market orders require a single POST to `/fapi/v1/order` with HMAC-SHA256 authentication. No WebSocket, no multi-leg, no order management state machine.
4. **Synchronous fills:** market orders on Binance Futures return fill results inline (with `newOrderRespType=RESULT`), matching the VenuePort contract that expects a synchronous `VenueOrderReceipt`.
5. **No external dependencies:** the adapter uses `net/http` (stdlib) and `crypto/hmac` — zero new module dependencies.

## Adapter Architecture

### Contract Compliance

`BinanceFuturesTestnetAdapter` implements `ports.VenuePort`:

```go
type VenuePort interface {
    SubmitOrder(ctx context.Context, req VenueOrderRequest) (VenueOrderReceipt, *problem.Problem)
}
```

### Seven Invariants (from minimal-real-venue-adapter-contracts.md)

| # | Invariant | Implementation |
|---|-----------|----------------|
| 1 | Context respect | `http.NewRequestWithContext(ctx, ...)` — deadline propagates to HTTP client |
| 2 | No gate bypass | Kill switch and staleness guards remain in VenueAdapterActor — adapter is gate-free |
| 3 | Credential isolation | Loaded via `LoadCredentials("binance_futures_testnet", ["API_KEY","API_SECRET"])` from env vars |
| 4 | Problem classification | HTTP 401/403 → InvalidArgument; 429/503 → Unavailable+Retryable; 4xx → InvalidArgument; 5xx → Unavailable+Retryable |
| 5 | Fill completeness | Response mapped to Status + FilledQuantity + Fills array with real price/quantity/fee |
| 6 | VenueOrderID uniqueness | Binance's `orderId` (int64) converted to string — globally unique per venue |
| 7 | Side-None handling | No-action intents return StatusAccepted without HTTP call |

### Request Flow

```
VenueAdapterActor.onIntent()
  ├── Gate 1: Kill switch (KV read)
  ├── Gate 2: Staleness guard (timestamp check)
  └── Gate 3: SubmitOrder (via BinanceFuturesTestnetAdapter)
        ├── Side=None → immediate StatusAccepted (no HTTP call)
        └── Side=Buy/Sell →
              ├── Build params (symbol, side, type=MARKET, quantity, timestamp, recvWindow)
              ├── HMAC-SHA256 sign
              ├── POST /fapi/v1/order with X-MBX-APIKEY header
              ├── Parse response → VenueOrderReceipt
              └── Error classification (auth, rate limit, rejection, server error)
```

### Symbol Mapping

Internal lowercase symbols are uppercased for Binance: `btcusdt` → `BTCUSDT`.

### Security

- API key sent via `X-MBX-APIKEY` header (Binance convention).
- API secret used only for HMAC signing — never sent over the wire.
- Credentials loaded from `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` and `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET`.
- Credential values never appear in logs, error messages, or Problem details.

### Error Classification

| HTTP Status | Problem Code | Retryable | Reason |
|-------------|-------------|-----------|--------|
| 200 | — | — | Success |
| 401, 403 | InvalidArgument | No | Auth failure — retrying won't help |
| 400 | InvalidArgument | No | Bad request (invalid symbol, quantity, etc.) |
| 429 | Unavailable | Yes | Rate limited — backoff and retry |
| 503 | Unavailable | Yes | Service unavailable — transient |
| 5xx | Unavailable | Yes | Server error — transient |

### Fill Record Mapping

| Binance Field | Domain Field | Notes |
|--------------|-------------|-------|
| `avgPrice` | `FillRecord.Price` | Average fill price |
| `executedQty` | `FillRecord.Quantity` | Executed quantity |
| `cumQuote` | `FillRecord.Fee` | Cumulative quote (fee proxy — real commissions from separate endpoint) |
| `updateTime` | `FillRecord.Timestamp` | Fill timestamp (milliseconds) |
| — | `FillRecord.Simulated` | Always `false` for real venue |

### Status Mapping

| Binance Status | Domain Status |
|---------------|--------------|
| NEW | StatusAccepted |
| FILLED | StatusFilled |
| PARTIALLY_FILLED | StatusPartiallyFilled |
| CANCELED/CANCELLED | StatusCancelled |
| REJECTED/EXPIRED | StatusRejected |

## Scope Constraints

- Market orders only (no limit, stop, OCO).
- Single exchange (no multi-venue routing).
- Synchronous fills (no async fill tracking).
- Testnet only (base URL hardcoded to `testnet.binancefuture.com`).
- Single symbol at a time (no batch orders).
- API key auth only (no OAuth).

## Testing

11 unit tests using `httptest.Server`:
- Successful fill (buy, sell)
- No-action intent (no HTTP call)
- Auth error (401 — non-retryable)
- Rejected order (400 — non-retryable)
- Server error (503 — retryable)
- Timeout (retryable)
- Rate limit (429 — retryable)
- Symbol mapping (lowercase → uppercase)
- Signature presence and format (64 hex chars)
- Fill not simulated (Simulated=false for real venue)

## Config-Driven Activation

```jsonc
{
  "venue": {
    "type": "binance_futures_testnet",
    "staleness_max_age": "120s",
    "submit_timeout": "10s"
  }
}
```

Requires environment variables:
```
MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY=...
MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET=...
```

## Files

- `internal/application/execution/binance_futures_testnet_adapter.go` — adapter implementation
- `internal/application/execution/binance_futures_testnet_adapter_test.go` — 11 unit tests
- `internal/shared/settings/schema.go` — VenueTypeBinanceFuturesTestnet constant
- `cmd/execute/run.go` — buildVenueAdapter wiring
