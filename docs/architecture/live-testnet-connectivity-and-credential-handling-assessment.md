# Live Testnet Connectivity and Credential Handling Assessment

> S348 — Venue Activation Wave

## Purpose

Assess and document the real connectivity path between the Foundry execute binary and the Binance Futures testnet, including credential loading, authentication, error classification, and operational ergonomics.

This assessment reduces uncertainty about the live venue path before endurance and deployment automation stages.

## Scope

- DNS resolution and TCP/TLS connectivity to `testnet.binancefuture.com`
- Credential loading via environment variables (`MF_VENUE_*`)
- HMAC-SHA256 request signing pipeline
- Authentication error classification (valid, invalid, missing credentials)
- Adapter timeout behavior under real network conditions
- Activation surface behavior under credential variations

## Connectivity Path

### Network Layer

```
execute binary
  → BinanceFuturesTestnetAdapter
    → HTTP POST https://testnet.binancefuture.com/fapi/v1/order
      → DNS resolution (testnet.binancefuture.com)
      → TLS 1.2+ handshake (system CA trust store)
      → HMAC-SHA256 signed query parameters
      → X-MBX-APIKEY header
```

### Assessment Results

| Layer | Status | Evidence |
|-------|--------|----------|
| DNS resolution | Validated | LTC-1: resolves to Binance CDN IPs |
| TCP port 443 | Validated | LTC-1: connection accepted |
| TLS handshake | Validated | LTC-2: valid certificate chain, TLS 1.2+ |
| Public endpoint | Validated | LTC-3: GET /fapi/v1/time returns 200 |
| Auth rejection | Validated | LTC-3: GET /fapi/v1/account without auth → 4xx |
| Invalid credentials | Validated | LTC-4: structured error, no credential leakage |
| Valid credentials | Conditional | LTC-5: requires real testnet API key/secret |
| Timeout handling | Validated | LTC-8: Unavailable+Retryable classification |

## Credential Handling Model

### Loading Convention

```
MF_VENUE_{VENUE_TYPE}_{CREDENTIAL_NAME}
```

For Binance Futures testnet:
- `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY`
- `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET`

### Loading Behavior

| Scenario | Outcome | Binary Behavior |
|----------|---------|-----------------|
| Both present | `CredentialSet` created | Binary starts with `CredentialPresent` |
| API_KEY missing | `Problem` with validation issues | Binary exits (fail-fast) |
| API_SECRET missing | `Problem` with validation issues | Binary exits (fail-fast) |
| Both missing | `Problem` with 2 validation issues | Binary exits (fail-fast) |
| Nil CredentialSet | Safe: Get→"", HasKey→false | Defensive nil-safety in accessor methods |

### Security Invariants (Verified)

1. Credential values are never logged, printed, or included in error messages
2. Credential values are never stored in config files
3. Load fails fast on missing required credentials
4. HMAC-SHA256 signature computed per-request, secret never sent over wire
5. API key sent only in `X-MBX-APIKEY` header (HTTPS-encrypted)
6. Error messages from venue adapter contain HTTP status codes and venue error codes but never credential fragments

### Authentication Flow

```
1. LoadCredentials("binance_futures_testnet", ["API_KEY", "API_SECRET"])
   → validates presence of both env vars
   → returns CredentialSet or Problem

2. NewBinanceFuturesTestnetAdapter(creds, timeout)
   → extracts API_KEY and API_SECRET from CredentialSet
   → configures HTTP client with timeout

3. SubmitOrder(ctx, request)
   → builds query params (symbol, side, type, quantity, timestamp, recvWindow)
   → signs payload: HMAC-SHA256(query_string, api_secret)
   → sets X-MBX-APIKEY header
   → POST /fapi/v1/order?{signed_params}
```

## Activation Surface Under Credential Variations

| Adapter | Gate | Credentials | Effective Mode | Can Submit Orders |
|---------|------|-------------|----------------|-------------------|
| paper | * | * | `paper` | No (simulated only) |
| venue | halted | present | `venue_halted` | No (gate blocks) |
| venue | halted | absent | `venue_halted` | No (gate blocks) |
| venue | active | absent | `venue_degraded` | No (no credentials) |
| venue | active | present | `venue_live` | Yes |

Key observation: the `venue_degraded` state should never occur in practice because `buildVenueAdapter` in `cmd/execute/run.go` calls `LoadCredentials` before creating the adapter. If credentials are missing, the binary exits before reaching the supervisor. The `venue_degraded` mode exists as a defensive state for the domain model.

## Error Classification Under Live Conditions

### Observed Error Patterns

| Condition | HTTP Status | Venue Code | Adapter Classification | Retryable |
|-----------|-------------|------------|----------------------|-----------|
| No API key | 401 | -2015 | InvalidArgument | No |
| Invalid API key | 400/401 | -2015 | InvalidArgument | No |
| Invalid signature | 400 | -1022 | InvalidArgument | No |
| Missing timestamp | 400 | -1021 | InvalidArgument | No |
| Venue internal error | 400 | -1001 | Unavailable (override) | Yes |
| IP rate limit | 418 | -1003 | Unavailable (override) | Yes |
| Order rate limit | 400 | -1015 | Unavailable (override) | Yes |
| HTTP timeout | — | — | Unavailable | Yes |
| DNS failure | — | — | Unavailable | Yes |
| TLS failure | — | — | Unavailable | Yes |

### S325 Classification Override

Three Binance error codes override HTTP-based classification:
- `-1001` (internal error): HTTP 400 → reclassified as Unavailable+Retryable
- `-1003` (IP rate limit): HTTP 418 → reclassified as Unavailable+Retryable
- `-1015` (order rate limit): HTTP 400 → reclassified as Unavailable+Retryable

This override is critical for correct retry behavior: without it, these transient failures would be classified as non-retryable client errors.

## Test Harness

### Build Tag Strategy

| Tag | Scope | Network Required | Credentials Required |
|-----|-------|-----------------|---------------------|
| (none) | Unit tests | No | No |
| `integration` | NATS + actor tests | NATS at localhost:4222 | Test values via t.Setenv |
| `livenet` | Real testnet connectivity | Outbound HTTPS | Optional (LTC-5 skips without) |

### Running S348 Tests

```bash
# Network connectivity tests (no credentials needed for LTC-1 through LTC-4)
go test -tags=livenet -count=1 -v -run "TestLiveTestnet_" ./internal/application/execution/...

# With real testnet credentials (enables LTC-5)
MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY=your_key \
MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET=your_secret \
go test -tags=livenet -count=1 -v -run "TestLiveTestnet_" ./internal/application/execution/...
```

## Operational Observations

### Testnet Availability

The Binance Futures testnet (`testnet.binancefuture.com`) is a shared public resource:
- Subject to periodic maintenance windows (typically unannounced)
- May have different rate limits than production
- Order book liquidity is synthetic/limited
- Fills may behave differently than mainnet (wider spreads, partial fills rare)

### Credential Acquisition

Testnet API keys are obtained from https://testnet.binancefuture.com:
1. Create testnet account (separate from mainnet)
2. Generate API key pair
3. Keys have no expiration by default
4. IP whitelist is optional on testnet (recommended on mainnet)

### Operational Friction Points

1. **No credential rotation support**: credentials are loaded once at binary startup; rotation requires restart
2. **No credential validation at startup**: the binary starts successfully with invalid (but present) credentials; the first SubmitOrder call reveals invalid credentials
3. **No health check against venue**: no periodic ping to verify venue reachability before intents arrive
4. **Environment variable only**: no support for file-based secrets, vault integration, or mounted secrets

These are documented limitations, not bugs. They are appropriate for the current testnet assessment phase.

## Conclusion

The connectivity path from the Foundry execute binary to the Binance Futures testnet is functional and well-classified. The credential model is simple but secure (values never logged). The error classification handles all observed testnet failure modes correctly, including the S325 venue-error-code overrides for transient failures.

The primary gap is operational: credentials are validated only on first use, not at startup. This is acceptable for testnet assessment but should be addressed before sustained live operation.
