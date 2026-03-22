# S348 — Live Testnet Connectivity and Credential Handling Assessment Report

> Phase 34: Production Readiness Assessment Wave
> Predecessor: S347 (production readiness assessment charter and scope freeze)
> Next: S349 (endurance assessment)

## Executive Summary

S348 assessed the real connectivity path between the Foundry execute binary and the Binance Futures testnet, including credential loading, authentication error handling, timeout classification, and operational ergonomics. The assessment validates that the venue adapter's network, signing, and error classification pipelines work correctly against the live testnet endpoint. It documents the current credential model's strengths and limitations, providing a clear operational baseline for endurance testing.

## Objective

Reduce uncertainty about live venue connectivity and credential handling before endurance and deployment automation stages. This is a connectivity and credential assessment, not production deployment.

## Connectivity Assessment

### Network Path Validated

| Layer | Test | Status |
|-------|------|--------|
| DNS resolution | LTC-1 | `testnet.binancefuture.com` resolves to Binance CDN |
| TCP connectivity | LTC-1 | Port 443 accepts connections |
| TLS handshake | LTC-2 | TLS 1.2+ with valid certificate chain |
| Public endpoint | LTC-3 | GET /fapi/v1/time → 200 (HTTP-level reachability) |
| Auth rejection | LTC-3 | GET /fapi/v1/account without auth → 4xx (structured) |
| Invalid credentials | LTC-4 | SubmitOrder with garbage keys → structured error, no credential leakage |
| Valid credentials | LTC-5 | Conditional (requires real testnet API keys) |
| Timeout handling | LTC-8 | 1ns timeout → Unavailable+Retryable, no credential leakage |

### Error Classification Under Live Conditions

The S325 venue-error-code classification overrides were validated against the adapter's error handling:

- HTTP 401/403 → InvalidArgument (non-retryable) — authentication failure
- HTTP 429 → Unavailable (retryable) — rate limit
- HTTP 400 + code -1001 → Unavailable (retryable) — venue internal (S325 override)
- HTTP 418 + code -1003 → Unavailable (retryable) — IP rate limit (S325 override)
- HTTP timeout → Unavailable (retryable) — network timeout
- DNS failure → Unavailable (retryable) — resolution failure

## Credential Handling Assessment

### Model Summary

- **Source**: Environment variables (`MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY`, `_API_SECRET`)
- **Loading**: Once at binary startup via `LoadCredentials()`, fail-fast on missing
- **Lifetime**: Immutable per process — rotation requires restart
- **Security**: Values never logged, printed, or included in error messages
- **Signing**: HMAC-SHA256 per request, secret never sent over wire

### Fail-Fast Behavior (LTC-6)

| Scenario | Credentials Count | Validation Issues | Binary Behavior |
|----------|------------------|-------------------|-----------------|
| Both missing | 0 | 2 (one per env var) | Exits at startup |
| API_KEY only | 1 | 1 (API_SECRET) | Exits at startup |
| Both present | 2 | 0 | Starts normally |
| Nil CredentialSet | — | — | Safe: Get→"", HasKey→false |

### Activation Surface Under Credential Variations (LTC-7)

| Adapter | Gate | Credentials | Effective Mode | Real Orders |
|---------|------|-------------|----------------|-------------|
| paper | * | * | `paper` | No |
| venue | halted | present | `venue_halted` | No |
| venue | active | absent | `venue_degraded` | No |
| venue | active | present | `venue_live` | Yes |

Key finding: `venue_degraded` is a defensive domain state that should never occur in practice because `buildVenueAdapter` exits the binary when credentials are missing for a venue adapter.

## Files Changed

### New Files

| File | Purpose |
|------|---------|
| `internal/application/execution/live_testnet_connectivity_test.go` | 8 test cases (LTC-1 through LTC-8) under `livenet` build tag |
| `docs/architecture/live-testnet-connectivity-and-credential-handling-assessment.md` | Connectivity and credential assessment document |
| `docs/architecture/credential-gated-operation-risks-ergonomics-and-limitations.md` | Credential risks, ergonomics, and limitations document |
| `docs/stages/stage-s348-live-testnet-connectivity-assessment-report.md` | This report |

### Updated Files

| File | Change |
|------|--------|
| `docs/architecture/README.md` | Added S348 section |
| `docs/stages/INDEX.md` | Added S348 entry |
| `scripts/smoke-activation.sh` | Added Phase 10: S348 connectivity assessment |

## Test Harness

### Build Tag Strategy

| Tag | Scope | Requirements |
|-----|-------|-------------|
| (none) | Unit tests | None |
| `integration` | NATS + actor pipeline | NATS at localhost:4222 |
| `livenet` | Real testnet connectivity | Outbound HTTPS to testnet.binancefuture.com |

### Running S348 Tests

```bash
# Connectivity assessment (no credentials needed for LTC-1 through LTC-4, LTC-6 through LTC-8)
go test -tags=livenet -count=1 -v -run "TestLiveTestnet_" ./internal/application/execution/...

# With real testnet credentials (enables LTC-5)
MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY=xxx \
MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET=yyy \
go test -tags=livenet -count=1 -v -run "TestLiveTestnet_" ./internal/application/execution/...
```

## Key Evidence

1. **DNS + TCP + TLS**: Validated against live `testnet.binancefuture.com` endpoint
2. **Auth error classification**: Invalid credentials produce structured `InvalidArgument` (non-retryable), no credential values in error messages
3. **Timeout classification**: Network timeouts produce `Unavailable` (retryable), correct for retry loop semantics
4. **Credential loading**: Fail-fast with structured `Problem` naming each missing env var
5. **Nil safety**: CredentialSet accessors are safe on nil receivers
6. **Activation surface**: All 4 effective modes computed correctly under credential variations

## Identified Risks

| Risk | Severity | Status |
|------|----------|--------|
| R1: No startup credential validation | Medium | Documented, proportional mitigation proposed |
| R2: No credential rotation without restart | Low | Acceptable for testnet |
| R3: Credential scope not enforced by adapter | Low | Base URL hardcoded to testnet |
| R4: Environment variable exposure surface | Low | Standard for container deployments |
| R5: No credential expiration awareness | Medium | Documented, proportional mitigation proposed |

## Remaining Limits

1. **LTC-5 conditional**: Full authenticated order flow requires real testnet API keys — not exercised in automated CI
2. **No venue health probe**: Binary does not verify venue reachability at startup
3. **No auth failure detection**: Consecutive authentication failures are not specifically tracked or alerted
4. **Single credential source**: Environment variables only — no file-based or vault integration
5. **Testnet != mainnet**: Testnet behavior may differ (rate limits, fills, availability)

## Acceptance Criteria

| Criterion | Status |
|-----------|--------|
| Live testnet connectivity assessed with real evidence | PASS (LTC-1 through LTC-4) |
| Credential model understood and documented | PASS |
| Operational uncertainty reduced without scope inflation | PASS |
| Base ready for endurance assessment | PASS |

## Preparation for S349 (Endurance Assessment)

### Recommended Before S349

1. **Startup venue ping**: Add an optional lightweight authenticated request at startup to validate credentials (warn-only, non-blocking)
2. **Auth failure counter**: Track consecutive authentication failures in health tracker for operational visibility

### S349 Should Assess

1. Sustained order flow over extended time window (minutes to hours)
2. Rate limit behavior under sustained load
3. Connection stability and reconnection behavior
4. Fill latency distribution under testnet conditions
5. Gate transition behavior during sustained operation

### Not Needed for S349

- Secret management platform
- Multi-account credential pools
- Credential rotation automation
- Mainnet connectivity
