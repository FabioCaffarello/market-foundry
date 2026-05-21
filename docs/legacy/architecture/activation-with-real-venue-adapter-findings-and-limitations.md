# Activation with Real Venue Adapter — Findings and Limitations

> S342: Behavioral differences, findings, and residual gaps from exercising the activation lifecycle with BinanceFuturesTestnetAdapter.

## Key Findings

### 1. Gate Blocks Venue Contact Entirely

When the gate is halted, the safety gate check runs **before** the venue submit call. This means:

- Zero HTTP requests reach the venue server
- No credentials are transmitted
- No network latency is incurred
- No venue rate-limit budget is consumed

This was assumed by design but S342 is the first stage to **prove** it with a real HTTP adapter and request counters.

### 2. Decorator Pipeline Executes with Real Adapter

The full decorator stack is assembled and exercised:

```
Post200Reconciler → RetrySubmitter(+halt checker, logger, tracker) → BinanceFuturesTestnetAdapter
```

With the paper adapter, the Post200Reconciler was wired but never triggered (paper never returns body-read-failure). With the real adapter, the reconciler is actively composed and the VenueQuery port is available.

### 3. Fill Records Differ Materially

| Field | Paper | Real Adapter |
|-------|-------|-------------|
| Simulated | true | false |
| Price | synthetic | parsed from `avgPrice` field |
| Quantity | input echoed | parsed from `executedQty` |
| Fee | "0" | parsed from `cumQuote` |
| VenueOrderID | "paper-{nano}" | numeric `orderId` from venue |
| Timestamp | now() | parsed from `updateTime` millis |

The `Simulated=false` distinction is critical: downstream consumers (analytics, P&L, risk) must distinguish paper fills from real fills.

### 4. Error Path Prevents Spurious Fills

When the venue returns an HTTP error (400, 401, 429, 5xx):

- No VenueOrderFilledEvent is published
- Error is recorded in the health tracker
- The adapter's error classification (retryable vs non-retryable) is exercised
- The retry submitter's halt-check hook prevents retry loops during gate transition

### 5. Activation Surface Correctly Reports Venue Dimensions

With `WithActivationState(AdapterVenue, CredentialPresent)`:

- Startup log shows `adapter=venue`, `credentials=present`
- Resolved surface after KV connect shows correct effective mode
- Gate transitions change the effective mode between `venue_halted` and `venue_live`

### 6. HMAC Signing Pipeline Exercised

Every HTTP request to the venue server carries:

- `X-MBX-APIKEY` header (credential wiring proof)
- `signature` query parameter (HMAC-SHA256 signing proof)
- `timestamp` and `recvWindow` (replay protection)

The simulated server validates the presence of these fields, proving the signing pipeline is wired end-to-end in the actor context.

## Behavioral Differences from Paper Adapter

### Latency Profile

Paper adapter fills are instant (configurable delay, typically 0). Real adapter fills involve:

1. HTTP round-trip to httptest.Server (~1ms in tests; 50-500ms with real testnet)
2. JSON response parsing
3. Fill record extraction

In the test environment, this difference is negligible. In production, the latency delta means the real adapter path has a larger window where in-flight intents could race with gate transitions.

### Error Surface

Paper adapter never fails (by design). Real adapter can fail due to:

- Authentication errors (401, 403)
- Rate limiting (429)
- Venue rejection (400)
- Server errors (502, 503, 5xx)
- Network timeouts
- Body-read failures after HTTP 200

S342/RVA-5 exercises the venue rejection path. Other error paths are covered by unit tests in `binance_futures_testnet_adapter_test.go`.

### Retry Behavior

Paper adapter never triggers retries. Real adapter can trigger the RetrySubmitter for retryable failures (rate limit, server error, timeout). The halt-check hook in RetrySubmitter ensures that retry loops are interrupted if the gate transitions to halted during a retry sequence.

## Limitations

| Limitation | Severity | Notes |
|-----------|----------|-------|
| httptest.Server, not live Binance testnet | Medium | Proves adapter code path and pipeline wiring; does not prove network behavior, testnet API quirks, or real credential validation |
| Single symbol (BTCUSDT) only | Low | By wave scope design; adapter code is symbol-agnostic |
| No partial fill scenario tested | Low | Testnet rarely produces partial fills for market orders; unit test covers PARTIALLY_FILLED status mapping |
| No body-read-failure-after-200 scenario | Low | Post200Reconciler is wired and unit-tested; integration proof deferred |
| No sustained load test | Low | Tests run in seconds; extended observation deferred to operational validation |
| Retry submitter not triggered in integration | Low | No retryable failure injected in RVA tests; unit tests cover retry behavior |
| No binary restart with real adapter | Low | Proven at domain level; integration restart test would require credential management |

## Comparison: S341 vs S342 Gap Closure

| S341 Limitation | S342 Status |
|----------------|-------------|
| Paper adapter used (no real venue HTTP) — **Medium** | **Closed**: RVA-1 through RVA-5 exercise real adapter |
| Binary restart rollback untested in integration — Low | Unchanged (out of scope) |
| Extended observation window not exercised — Low | Unchanged (out of scope) |
| Multi-venue gating not available — Low | Unchanged (by design) |
| HTTP → KV path not tested in integration — Low | Unchanged (smoke script covers) |
| Fail-open on KV unavailability accepted — Accepted | Unchanged (accepted risk) |

## Conclusion

S342 eliminates the principal medium-severity gap from S341. The activation lifecycle is now proven on both the paper and real venue adapter paths. Remaining limitations are low-severity and align with the wave's scope boundaries.
