# Credential-Gated Operation: Risks, Ergonomics, and Limitations

> S348 — Venue Activation Wave

## Purpose

Document the operational risks, ergonomic friction, and architectural limitations of the current credential-gated venue operation model. This assessment is scoped to the testnet phase and does not prescribe a full secret management platform.

## Current Model Summary

The Foundry uses a minimal, environment-variable-based credential model:

```
MF_VENUE_{VENUE_TYPE}_{CREDENTIAL_NAME} → CredentialSet → BinanceFuturesTestnetAdapter
```

Credentials are:
- Loaded once at binary startup via `LoadCredentials()`
- Required to be non-empty strings
- Never logged, printed, or included in error messages
- Used to construct HMAC-SHA256 signatures per request
- Immutable for the process lifetime

## Risk Assessment

### R1: No Startup Credential Validation

**Risk**: The binary starts successfully with present-but-invalid credentials. The first `SubmitOrder` call fails with a 401/400 from the venue.

**Impact**: Delayed failure detection. An operator may believe the system is ready (activation surface shows `venue_live`) when it cannot actually execute orders.

**Mitigation options**:
- Add a lightweight venue ping at startup (GET /fapi/v1/time with signed request)
- Log a warning if first SubmitOrder fails with auth error
- Surface auth failure in health check counters

**Current state**: Not mitigated. Acceptable for testnet assessment; should be addressed for sustained operation.

### R2: No Credential Rotation Without Restart

**Risk**: If credentials are compromised or expired, the only way to rotate them is to restart the execute binary with new environment variables.

**Impact**: Downtime during rotation. Any in-flight intents are lost (re-delivered by NATS after AckWait expires).

**Mitigation options**:
- Implement hot-reload of credentials from a watched file or signal
- Accept restart-based rotation (simple, explicit, auditable)

**Current state**: Restart-based rotation is the current model. This is appropriate for testnet and early operation where the execute binary is stateless (NATS handles replay).

### R3: Credential Scope Not Enforced by Adapter

**Risk**: The adapter does not verify that credentials have the correct permissions (e.g., futures trading enabled, testnet-only). A mainnet credential accidentally used on testnet would silently work (and vice versa, a testnet credential on mainnet would fail at the venue).

**Impact**: Operational confusion. The `binanceTestnetBaseURL` constant prevents testnet credentials from reaching mainnet, but there is no adapter-level validation.

**Mitigation options**:
- Verify base URL matches expected testnet URL in adapter constructor
- Add a startup check that queries the venue's exchange info endpoint

**Current state**: The base URL is hardcoded to `https://testnet.binancefuture.com`. The `WithBaseURL` override is only used in tests. This is sufficient for the current scope.

### R4: Environment Variable Exposure Surface

**Risk**: Environment variables are visible to any process running as the same user, in /proc/self/environ (Linux), and in container inspection output.

**Impact**: Credential exposure in shared environments, container orchestration logs, or debugging sessions.

**Mitigation options**:
- Use file-based secrets (e.g., `/run/secrets/mf_venue_api_key`)
- Use a secrets manager with SDK integration
- Set restrictive file permissions on .env files

**Current state**: Environment variables are the only supported source. This is standard practice for container-based deployments and acceptable for testnet. Production should evaluate file-based or vault-based alternatives.

### R5: No Credential Expiration Awareness

**Risk**: If the venue revokes or expires credentials, the adapter discovers this only when a SubmitOrder call fails.

**Impact**: Silent degradation. The activation surface still shows `venue_live` (credentials are `present` because env vars are set), but orders fail with authentication errors.

**Mitigation options**:
- Periodic health probe that exercises the signing pipeline
- Monitor adapter error counters for auth failure patterns
- Alert on consecutive auth failures

**Current state**: Not mitigated. The health tracker records errors, but there is no specific auth-failure detection or alerting.

## Ergonomic Assessment

### Credential Setup Workflow

```
1. Obtain testnet API key pair from testnet.binancefuture.com
2. Set environment variables:
   export MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY="your_key"
   export MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET="your_secret"
3. Configure venue type in settings:
   venue:
     type: binance_futures_testnet
4. Start execute binary
5. Verify activation surface shows venue_live
```

### Friction Points

| Friction | Severity | Description |
|----------|----------|-------------|
| No `.env` file support | Low | Operator must export vars manually or use shell profile |
| No credential validation feedback | Medium | Binary starts, but auth failure only surfaces on first order |
| No credential status in health endpoint | Medium | Health check does not distinguish "credentials loaded" from "credentials valid" |
| Restart required for rotation | Low | Acceptable for testnet; becomes Medium for sustained operation |
| No per-venue credential isolation | Low | Prefix convention (`MF_VENUE_{TYPE}_`) provides namespace isolation |
| Error messages are opaque | Low | By design (security), but makes debugging harder for operators |

### Positive Ergonomic Properties

1. **Convention-based naming**: `MF_VENUE_{TYPE}_{KEY}` is predictable and discoverable
2. **Fail-fast on missing credentials**: Binary exits immediately if required credentials are absent, preventing a partially-configured deployment
3. **Structured validation errors**: Missing credentials produce a `Problem` with `ValidationIssue` list naming each missing env var
4. **No credential in logs**: Security invariant is consistently enforced across all code paths
5. **Activation surface visibility**: `GET /activation/surface` shows `credentials: present|absent` for operational awareness
6. **Kill switch independence**: Gate control works regardless of credential state, allowing halt even if credentials are broken

## Architectural Limitations

### L1: Single Credential Set Per Venue Type

The current model supports one set of credentials per venue type. There is no concept of multiple accounts, sub-accounts, or credential pools for a single venue.

**Impact**: Sufficient for testnet. Production may require multi-account support for risk isolation.

### L2: No Credential Lifecycle Events

Credential loading is a one-shot operation. There are no events emitted when credentials are loaded, validated, rotated, or found invalid.

**Impact**: No audit trail for credential operations. The binary's startup log is the only record of credential state.

### L3: No Cross-Venue Credential Federation

Each venue adapter loads its own credentials independently. There is no shared credential service or cross-venue authentication.

**Impact**: Acceptable for single-venue operation. Would need re-evaluation for multi-venue deployments.

### L4: CredentialSet is Opaque to Domain

The domain layer (`internal/domain/execution`) only knows `CredentialPresent` or `CredentialAbsent`. It has no visibility into credential validity, expiration, or scope.

**Impact**: The activation surface accurately reflects credential presence but not credential validity. This is a deliberate design choice that keeps the domain clean, but it means the surface can show `venue_live` when credentials are actually invalid.

## Recommendations for Next Stages

### Proportional (Before Endurance — S349)

1. **Startup venue ping**: Add a lightweight authenticated request at startup to validate credentials are accepted by the venue. Log result but do not block startup (warn-only).
2. **Auth failure counter**: Track consecutive authentication failures in the health tracker. Surface in health endpoint.

### Deferred (After Endurance)

3. **File-based credential source**: Support reading credentials from a file path (e.g., `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY_FILE`).
4. **Credential rotation signal**: Support SIGHUP or similar to trigger credential reload without full restart.
5. **Credential health probe**: Periodic (e.g., every 5 minutes) lightweight venue request to verify credentials remain valid.

### Out of Scope (Not Recommended for Current Wave)

- Full secret management platform (Vault, AWS Secrets Manager, etc.)
- Multi-account credential pools
- Credential rotation automation
- Compliance-grade audit logging for credential access

## Conclusion

The current credential model is minimal, secure, and appropriate for testnet assessment. Its primary strengths are fail-fast loading, no-leak error handling, and activation surface visibility. Its primary weaknesses are no startup validation and no rotation without restart. Both weaknesses have proportional mitigations available for the next stage.
