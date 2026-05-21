# Closure Tranche Items, Priorities, Exit Criteria, and Non-Goals

**Stage:** S321 — Venue Closure Tranche Charter
**Date:** 2026-03-21
**Charter authority:** [venue-closure-tranche-charter-and-scope-freeze.md](venue-closure-tranche-charter-and-scope-freeze.md)

---

## 1. Items and Priority Order

### 1.1 Priority Matrix

| Priority | Item ID | Name | Risk (S320) | Cost | Value | Stage |
|----------|---------|------|-------------|------|-------|-------|
| **1** | CT-1 | Body-read-failure reconciliation | Medium | Medium | High | S322 |
| **2** | CT-2 | Global deadline in RetryPolicy | Low | Low | Medium | S323 |
| **3** | CT-3 | Kill switch check during retry backoff | Low | Low | Medium | S323 |
| **4** | CT-4 | Structured retry metrics and logging | Low | Low | Medium | S324 |
| **5** | CT-5 | Venue error code classification | Low | Low | Low–Medium | S325 |

### 1.2 Priority Rationale

**CT-1 first**: Only medium-risk gap. Body-read-failure-after-200 is the one scenario where an order is accepted but the system has no fill data. Without reconciliation, the persistence pipeline stalls. All other gaps are low-risk operational enhancements.

**CT-2 and CT-3 together (S323)**: Both modify the RetrySubmitter's loop behavior. CT-2 adds the global deadline; CT-3 adds the kill switch check. Co-locating them avoids touching the retry loop twice and keeps the diff focused.

**CT-4 after retry hardening**: Logging should capture the final retry semantics (including global deadline and kill switch), so it runs after S323 stabilizes the loop.

**CT-5 last**: Venue error code classification is a refinement layer on top of working HTTP-status-based classification. It improves fidelity but the existing classification is already correct for all verified scenarios.

---

## 2. Detailed Item Specifications

### 2.1 CT-1: Body-Read-Failure Reconciliation

**Source**: R-S320-1, S320 §3.1, FP-11

**Problem**: When the venue returns HTTP 200 headers but body read fails (timeout, connection reset during streaming), the adapter correctly classifies this as `Internal, non-retryable`. However, the order was accepted by the venue. The system has no fill details (price, quantity, fee) and cannot resume the persistence pipeline.

**Solution**:

1. Add `QueryOrderStatus(ctx context.Context, clientOrderID string) (VenueOrderReceipt, *problem.Problem)` to VenuePort (or as a separate reconciliation port if interface separation is preferred).
2. In the adapter, implement `QueryOrderStatus` using Binance's `GET /fapi/v1/order` endpoint with `origClientOrderId` parameter.
3. In the retry submitter or a wrapping reconciliation layer: when `SubmitOrder` returns an `Internal` error with body-read characteristics, call `QueryOrderStatus` to recover the fill.
4. If reconciliation succeeds, return the recovered receipt. If it fails, return the original error enriched with `reconciliation_attempted: true`.

**Constraints**:
- Reconciliation is a best-effort recovery, not a guaranteed mechanism
- Only triggered for body-read-failure-after-200, not for other Internal errors (parse failure, unknown status)
- Single reconciliation attempt, no retry loop on the reconciliation call itself
- Per-request deadline applies to the reconciliation call

**Exit criteria**:
- [ ] `QueryOrderStatus` implemented and unit-tested against httptest mock
- [ ] Reconciliation path triggered only on body-read-failure-after-200
- [ ] Successful reconciliation returns valid receipt with fill details
- [ ] Failed reconciliation returns original error with reconciliation metadata
- [ ] Existing 19 failure path tests pass without regression
- [ ] Client order ID used for reconciliation matches the original submission

### 2.2 CT-2: Global Deadline in RetryPolicy

**Source**: R-S320-2, S320 findings §2.2

**Problem**: The retry loop has no global deadline. Worst-case wall clock = MaxAttempts × per-request timeout + backoff = ~30.3s with defaults. Callers must provide their own context deadline. A global deadline in RetryPolicy would make this explicit and self-documenting.

**Solution**:

1. Add `GlobalDeadline time.Duration` to `RetryPolicy` struct (zero value = no global deadline, preserving backward compatibility).
2. In `RetrySubmitter.SubmitOrder`, if `GlobalDeadline > 0`, wrap `ctx` with `context.WithTimeout(ctx, policy.GlobalDeadline)` before entering the retry loop.
3. The shorter of caller's context deadline and global deadline wins (standard Go context behavior).

**Constraints**:
- Zero value means no global deadline (backward compatible)
- `DefaultRetryPolicy()` sets a sensible global deadline (e.g., 30s)
- Does not change per-attempt deadline semantics (EC-3)

**Exit criteria**:
- [ ] `GlobalDeadline` field added to RetryPolicy
- [ ] Global deadline enforced in RetrySubmitter when set
- [ ] Zero value preserves current behavior (no deadline)
- [ ] Test: global deadline < total retry budget → loop aborts with retry metadata
- [ ] Test: global deadline > total retry budget → all attempts execute normally
- [ ] Existing retry tests pass without regression

### 2.3 CT-3: Kill Switch Check During Retry Backoff

**Source**: R-S320-3, S319 §4 design note

**Problem**: The safety gate is checked once before the retry loop (S319 §4 design decision). If the kill switch activates during retries, the loop continues until MaxAttempts or context deadline. This means a kill switch signal can be delayed by up to 30s.

**Solution**:

1. Add an optional `HaltCheck func() bool` to `RetrySubmitter` (nil = no check, backward compatible).
2. Before each retry attempt (not the first attempt), call `HaltCheck()`. If it returns true, abort with a dedicated Problem: `Unavailable, non-retryable, "kill switch activated during retry"`.
3. The check is lightweight (KV read, already cached in memory by the safety gate).

**Constraints**:
- First attempt is not gated by HaltCheck (the safety gate already cleared it)
- HaltCheck is a function, not a concrete type (decoupled from safety gate implementation)
- Nil HaltCheck preserves current behavior

**Exit criteria**:
- [ ] `HaltCheck` field added to RetrySubmitter
- [ ] Kill switch check executes before each retry (attempts 2+)
- [ ] Nil HaltCheck preserves current behavior
- [ ] Test: halt activates after first attempt → immediate abort with descriptive Problem
- [ ] Test: halt never activates → all attempts execute normally
- [ ] Existing retry tests pass without regression

### 2.4 CT-4: Structured Retry Metrics and Logging

**Source**: R-S320-5

**Problem**: Retry behavior is only observable through Problem.Details on the final error. There is no logging during the retry loop. Production troubleshooting requires visibility into attempt timing, backoff delays, and recovery patterns.

**Solution**:

1. Add an optional `Logger` (e.g., `*slog.Logger`) to `RetrySubmitter`.
2. Emit structured log entries at key points:
   - `retry.attempt_start`: attempt number, client order ID
   - `retry.attempt_failed`: attempt number, error code, retryable, will_retry
   - `retry.backoff`: attempt number, delay duration
   - `retry.exhausted`: total attempts, final error code
   - `retry.recovered`: attempt number where success occurred
   - `retry.halt_aborted`: attempt number where kill switch fired
3. All log entries use `slog.Logger` with structured key-value pairs.

**Constraints**:
- Nil logger = no logging (backward compatible)
- Log level: Info for recovery, Warn for exhaustion/abort, Debug for per-attempt details
- No metrics counters or Prometheus integration — structured logging only
- No external dependencies added

**Exit criteria**:
- [ ] Logger field added to RetrySubmitter
- [ ] Structured log entries emitted for all retry events
- [ ] Nil logger produces no output
- [ ] Test: capture log output and verify structured fields present
- [ ] Log entries include client order ID, attempt number, and timing
- [ ] Existing retry tests pass without regression

### 2.5 CT-5: Venue Error Code Classification

**Source**: R-S320-4, S320 findings §1.2

**Problem**: The adapter classifies errors using HTTP status codes only. Venue-specific error codes (e.g., Binance code -1015 for rate limit, -2022 for duplicate order) are captured in Problem.Details but do not influence classification. This may misclassify edge cases where the HTTP status is ambiguous but the venue code is specific.

**Solution**:

1. After HTTP-status-based classification, check if the venue error code provides a more specific classification.
2. Define a Binance error code map for known codes:
   - `-1015` → Rate limit (confirm Unavailable, retryable)
   - `-2022` → Duplicate order (confirm InvalidArgument, non-retryable)
   - `-1021` → Timestamp out of window (classify as Unavailable, retryable — clock drift is transient)
   - `-1003` → Too many requests (classify as Unavailable, retryable)
3. Unknown codes fall back to HTTP-status-based classification (no regression).

**Constraints**:
- HTTP status remains the primary classification signal
- Venue codes only override when they provide strictly more specific information
- Unknown venue codes do not change classification
- The code map is a simple switch/map, not a plugin architecture

**Exit criteria**:
- [ ] Binance error code map implemented in adapter
- [ ] Known codes produce correct classification
- [ ] Unknown codes fall back to HTTP-status classification
- [ ] Test: each mapped code → correct failure class
- [ ] Test: unmapped code → same classification as HTTP-status-only
- [ ] Existing 19 failure path tests pass without regression

---

## 3. Exit Criteria for Closure Tranche (S326 Gate)

### 3.1 Per-Item Verification

| Item | Exit Criterion | Evidence |
|------|---------------|----------|
| CT-1 | Reconciliation recovers fill after body-read-failure | Unit test with httptest mock |
| CT-2 | Global deadline bounds total retry wall clock | Unit test with short deadline |
| CT-3 | Kill switch aborts retry loop between attempts | Unit test with mock HaltCheck |
| CT-4 | Structured log entries emitted for all retry events | Log capture test |
| CT-5 | Venue codes improve classification without regression | Unit test matrix |

### 3.2 Aggregate Gate Criteria

| Criterion | Threshold | Verification Method |
|-----------|-----------|-------------------|
| All 5 items implemented | 5/5 | Stage reports |
| Zero regression in existing execution tests | 0 failures in 80+ tests | `go test ./internal/application/execution/...` |
| Zero regression in failure path tests | 0 failures in 19 FP tests | `go test -run TestFP` |
| No scope inflation | Exactly 5 items, no additions | This charter audit |
| All exit criteria checked | All boxes checked per §3.1 | Stage reports |
| Paper pipeline green | All existing tests pass | Full test suite |

### 3.3 Gate Verdict Options

| Verdict | Meaning | Next Step |
|---------|---------|-----------|
| **PASS** | All 5 items closed, zero regressions | Proceed to evidence gate |
| **PASS WITH RESIDUALS** | 4–5 items closed, minor residuals logged | Proceed to evidence gate with residuals documented |
| **FAIL** | <4 items closed or regressions found | Diagnose; do not open evidence gate |

---

## 4. Non-Goals (Explicit Exclusions)

### 4.1 Architecture and Design

| Non-Goal | Rationale |
|----------|-----------|
| OMS or order management system | S309 proved no OMS needed for testnet scope |
| VenuePort interface redesign | Interface is correct; CT-1 adds one optional method only |
| Retry architecture redesign | RetrySubmitter decorator pattern is proven; only enhance |
| Circuit breaker pattern | Not needed for single-venue testnet with bounded retries |
| Async/queue-based retry | Synchronous retry proven sufficient |
| Per-error-class differentiated retry policies (R-S320-6) | Uniform backoff sufficient; deferred to post-evidence-gate |

### 4.2 Operations and Infrastructure

| Non-Goal | Rationale |
|----------|-----------|
| Prometheus/Grafana metrics pipeline | Structured logging sufficient; dashboards are operational maturity work |
| Alerting infrastructure | Out of scope for testnet |
| Distributed tracing integration | No tracing infrastructure exists; not needed for single-service |
| Dashboard creation | Operational maturity, not closure |

### 4.3 Venue and Execution Scope

| Non-Goal | Rationale |
|----------|-----------|
| Mainnet venue calls | System is testnet-only |
| Multi-venue routing or abstraction | Single venue not yet fully proven |
| WebSocket or async fill feed | S306 non-goal (NG-5) |
| Multi-symbol venue concurrency hardening | Verified in S302–S304 |
| Portfolio risk or position tracking | Not in scope until venue proven in production |
| Fill model code changes | C-FILL contracts already match existing adapter code |

### 4.4 Infrastructure

| Non-Goal | Rationale |
|----------|-----------|
| New binaries or services | S310 constraint CN-3 |
| New NATS subjects or KV buckets | S310 constraint CN-2 |
| ClickHouse schema changes | S306 non-goal (NG-9) |
| New HTTP endpoints | S306 non-goal (NG-10) |

---

## 5. Preparation for S322

S322 (Body-Read-Failure Reconciliation) should begin with:

1. **Read**: S320 FP-11 test (`failure_path_verification_test.go`) to understand the body-read-failure scenario
2. **Read**: Binance Futures API docs for `GET /fapi/v1/order` endpoint and `origClientOrderId` parameter
3. **Read**: Current `VenuePort` interface (`internal/application/ports/venue.go`) to plan the minimal interface addition
4. **Read**: `RetrySubmitter` (`internal/application/execution/retry_submitter.go`) to determine where reconciliation logic fits

**Key design decision for S322**: Whether `QueryOrderStatus` belongs on `VenuePort` (extending the existing interface) or on a separate `VenueReconciler` port (keeping interfaces segregated). Recommendation: add to `VenuePort` since the reconciliation is venue-specific and uses the same HTTP client, credentials, and signing logic.

---

*Delivered: 2026-03-21 — Stage S321, Phase 30*
