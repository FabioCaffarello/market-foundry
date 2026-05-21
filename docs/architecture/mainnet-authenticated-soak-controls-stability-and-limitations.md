# Mainnet Authenticated Soak: Controls, Stability, and Limitations

**Stage:** S441
**Status:** Complete
**Date:** 2026-03-24

## Purpose

This document describes the safety controls, stability characteristics, and known limitations of the authenticated mainnet soak executed in S441. It is the companion document to the proof methodology described in `authenticated-mainnet-api-proof-and-sustained-soak.md`.

## Safety Controls

### 1. DryRunSubmitter (Outermost Decorator)

The `DryRunSubmitter` is the outermost layer in the venue pipeline. When active:

- `SubmitOrder()` is intercepted before any call reaches the inner adapter
- All receipts carry `VenueOrderID` prefix `"dryrun-"`
- All fills are marked `Simulated=true`
- The inner adapter's `SubmitOrder()` is **never called**
- A structured log line is emitted for every intercepted intent

**Invariant verified:** AMP-3 and AMP-6 confirm DryRunSubmitter interception at 100% reliability throughout the soak, including after authenticated API calls.

### 2. Config Validation (Fail-Closed)

`VenueConfig.Validate()` rejects `dry_run=false` when any mainnet adapter is configured:

```
mainnet adapter + dry_run=false → validation error (config rejected)
mainnet adapter + dry_run=true  → accepted
mainnet adapter + dry_run=nil   → accepted (IsDryRun() defaults to true)
```

This means there is no configuration that can both select a mainnet adapter and disable dry-run mode. Removing this guard requires a code change and a new authorization ceremony.

### 3. Preflight Credential Check

`MainnetCredentialCheck()` runs at startup before any adapter is constructed:

- Iterates enabled segments
- For each mainnet adapter, resolves API_KEY and API_SECRET
- Fails fast (process exit) if any credential is missing
- Format validation rejects truncated (<16 chars) or whitespace-containing values

### 4. Rate Limiter

The `RateLimiter` decorator enforces a token-bucket rate limit:

- **Capacity:** 10 tokens (burst)
- **Refill:** 1 token per 100ms (~10 req/s sustained)
- **Blocking:** waits for token or returns `Unavailable` on context expiry

This prevents accidental rate-limit violations against Binance mainnet's strict limits.

### 5. AccountStatus() Read-Only Design

The `AccountStatus()` method introduced in S441:

- Uses `GET` method only (no write side effects)
- Calls `/api/v3/account` (Spot) or `/fapi/v2/account` (Futures)
- Returns account metadata (balances, permissions) without modifying state
- Uses the same HMAC-SHA256 signing as SubmitOrder (proves credential validity)
- Response body is limited to 256KB (Spot) / 512KB (Futures) to prevent memory exhaustion

### 6. No Order Submission Path in Tests

No test in the S441 suite calls `SubmitOrder()` on a raw mainnet adapter. All order submission goes through `DryRunSubmitter`, which intercepts before the inner adapter is reached.

## Stability Characteristics

### Soak Window

| Parameter | Value |
|-----------|-------|
| Default duration | 5 minutes |
| Call interval | 15 seconds per segment |
| Total call rate | ~8 authenticated calls/min |
| Failure tolerance | 5% (network jitter allowance) |

### Expected Behavior

During a healthy soak:
- Both Spot and Futures endpoints return HTTP 200 consistently
- Latency is typically 100-500ms per call
- No rate limiting triggers (well below Binance limits)
- DryRunSubmitter interception is 100% (zero escapes)

### Failure Modes

| Failure | Cause | Impact | Recovery |
|---------|-------|--------|----------|
| DNS resolution failure | Network outage | Soak records failure | Retries on next interval |
| TLS handshake failure | Certificate rotation | Soak records failure | Retries on next interval |
| HTTP 401/403 | Invalid/expired credentials | Soak records failure | Credential refresh needed |
| HTTP 429 | Rate limit exceeded | Soak records failure | Automatic backoff via interval |
| HTTP 5xx | Binance server error | Soak records failure | Retries on next interval |
| Context timeout | Network latency | Soak records failure | 15s timeout per call |

All failure modes are non-destructive (read-only calls). The soak tolerates up to 5% failure rate before marking the test as failed.

## Audit Trail

### Authenticated Call Audit

Each `AccountStatus()` call produces:
- HTTP request with `X-MBX-APIKEY` header (key visible to venue)
- HMAC-SHA256 signature in query string (proves key+secret pair)
- HTTP response status code (200 on success)
- Parsed account metadata (canTrade, balanceCount, etc.)

### DryRunSubmitter Audit

Each intercepted `SubmitOrder()` call produces:
- `VenueOrderID` with `dryrun-` prefix (filterable in all downstream systems)
- `Simulated=true` on all fill records
- Structured log line (when logger attached)
- `Fee="0"` on all simulated fills

## Limitations

### L-1: Read-Only Proof Does Not Exercise Order Path

`AccountStatus()` proves credential validity and endpoint selection but does not exercise the `POST /api/v3/order` or `POST /fapi/v1/order` paths. Those paths remain behind `DryRunSubmitter` and are proven to work via the testnet adapters (which share identical request construction).

### L-2: Soak Duration Is Configurable

The default 5-minute soak is sufficient for proof but is not a production endurance test. For extended validation, set `MF_SOAK_DURATION=1h` (or longer).

### L-3: Single-Host Execution

The soak runs from the test runner process, not from the containerized Docker Compose stack. This validates the adapter + credential path but not the full runtime boot sequence with NATS, ClickHouse, and actor supervision.

### L-4: No WebSocket Authenticated Streams

Only REST authenticated calls are proven. WebSocket authenticated streams (user data stream) are not covered by this soak.

### L-5: Credential Rotation Not Tested

The soak uses a single credential set throughout. Key rotation during an active soak window is not tested.

### L-6: Binance Spot and Futures Only

This proof covers Binance Spot and Binance Futures mainnet only. No other exchanges or venues are in scope.

## Relationship to Authorization Conditions

| Condition | Status | Evidence |
|-----------|--------|----------|
| C-1: Authenticated mainnet API call | **CLOSED** | AMP-1, AMP-2: HTTP 200 with valid account data |
| C-2: External secret manager deployed | CLOSED (S439) | FileCredentialProvider proven |
| C-3: Automated off-host backup | CLOSED (S440) | Backup pipeline proven |
| C-4: Sustained mainnet soak | **CLOSED** | AMP-5: 5-minute soak within tolerance |
| C-5: Kill-switch operational runbook | OPEN | Deferred to S442 |
| C-6: Explicit removal of dry_run=false rejection | OPEN | Requires authorization ceremony |
