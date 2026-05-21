# Tranche Items Evidence Matrix, Regressions, and Residual Gaps

Status: **DELIVERED** (2026-03-21)
Gate stage: S315
Tranche: Adapter Hardening Foundational Tranche (S312–S314)

---

## 1. Per-Item Evidence Matrix

### EC-1: Client Order ID Derivation (Critical — S313)

| Exit Criterion | Spec | Evidence | Verdict |
|---|---|---|---|
| EC-1.1 | Same intent → same ID across calls | `TestClientOrderID_Deterministic`: calls `ClientOrderID()` 3× on identical intent, asserts equality | **PASS** |
| EC-1.2 | Different intents → different IDs | `TestClientOrderID_Uniqueness`: varies type, source, symbol, timeframe, timestamp independently; all 5 produce distinct IDs | **PASS** |
| EC-1.3 | Binance `newClientOrderId` format | `TestClientOrderID_BinanceFormat`: asserts ≤36 chars, non-empty, hex-only (32 hex chars from SHA-256 truncation) | **PASS** |
| EC-1.4 | `VenueOrderReceipt.ClientOrderID` populated | `TestBinanceAdapter_ClientOrderID_InReceipt`: submits order, asserts `receipt.ClientOrderID` non-empty and matches derivation | **PASS** |
| EC-1.5 | `newClientOrderId` in HTTP request body | `TestBinanceAdapter_ClientOrderID_InHTTPRequest`: httptest intercepts request, asserts `newClientOrderId` parameter present and matches `ClientOrderID(intent)` | **PASS** |
| EC-1.6 | No random/time-varying inputs | `TestClientOrderID_NoRandomInputs`: 1000 calls with frozen intent produce identical ID; code review confirms no `rand`, no `time.Now()` in `client_order_id.go` | **PASS** |

**EC-1 verdict: 6/6 PASS**

---

### EC-2: Response Body Size Cap (Low — S313)

| Exit Criterion | Spec | Evidence | Verdict |
|---|---|---|---|
| EC-2.1 | All body reads use `io.LimitReader` with 64 KB | Code review: `binance_futures_testnet_adapter.go` line 117 — single body read site uses `io.LimitReader(resp.Body, 64*1024)` | **PASS** |
| EC-2.2 | Body >64 KB truncated at boundary | `TestBinanceAdapter_OversizedBody_Truncated`: httptest serves 128 KB; reader stops at 64 KB; valid JSON at front parses correctly | **PASS** |
| EC-2.3 | Truncated response → `problem.Internal` | `TestBinanceAdapter_OversizedBody_CorruptedJSON`: oversized body with mid-JSON truncation → `problem.Internal` category | **PASS** |
| EC-2.4 | Truncated response not retryable | Same test: asserts `Retryable == false` | **PASS** |
| EC-2.5 | Normal responses (<64 KB) unaffected | `TestBinanceAdapter_SubmitOrder_Filled`: standard response parses correctly with fill record | **PASS** |

**EC-2 verdict: 5/5 PASS**

---

### EC-3: Per-Request Context Deadline (Medium — S313)

| Exit Criterion | Spec | Evidence | Verdict |
|---|---|---|---|
| EC-3.1 | Every `SubmitOrder` call wrapped with `context.WithTimeout` | Code review: `binance_futures_testnet_adapter.go` lines 64-69 — checks `ctx.Deadline()`; if absent, wraps with `context.WithTimeout(ctx, defaultRequestDeadline)` | **PASS** |
| EC-3.2 | Timeout duration configurable | Code review: `NewBinanceFuturesTestnetAdapter` accepts `submitTimeout` parameter; `defaultRequestDeadline = 10 * time.Second` used only as fallback | **PASS** |
| EC-3.3 | Slow venue triggers cancellation | `TestBinanceAdapter_ContextDeadline_Exceeded`: httptest delays 2s, context deadline 200ms → `SubmitOrder` returns error before delay completes | **PASS** |
| EC-3.4 | Timeout → `problem.Unavailable`, `Retryable == true` | `TestRF1_6_ContextDeadline_Retryable`: deadline exceeded → Unavailable category, Retryable=true | **PASS** |
| EC-3.5 | Intent state after timeout remains `submitted` (PGR-08) | `TestBinanceAdapter_ContextDeadline_IntentUnmutated`: captures intent before timeout, asserts unchanged after | **PASS** |
| EC-3.6 | Normal responses within deadline unaffected | `TestBinanceAdapter_SubmitOrder_Filled`: httptest responds immediately; fill parsed correctly | **PASS** |

**EC-3 verdict: 6/6 PASS**

---

### VA-1: Error Classification Completeness (High — S314)

| Exit Criterion | Spec | Evidence | Verdict |
|---|---|---|---|
| VA-1.1 | HTTP 401 → `InvalidArgument`, not retryable | `TestVA1_1_HTTP401_InvalidArgument_NotRetryable` | **PASS** |
| VA-1.2 | HTTP 403 → `InvalidArgument`, not retryable | `TestVA1_2_HTTP403_InvalidArgument_NotRetryable` | **PASS** |
| VA-1.3 | HTTP 400 → `InvalidArgument`, not retryable | `TestVA1_3_HTTP400_InvalidArgument_NotRetryable` | **PASS** |
| VA-1.4 | HTTP 422 → `InvalidArgument`, not retryable | `TestVA1_4_HTTP422_InvalidArgument_NotRetryable` | **PASS** |
| VA-1.5 | HTTP 429 → `Unavailable`, retryable | `TestVA1_5_HTTP429_Unavailable_Retryable` | **PASS** |
| VA-1.6 | HTTP 503 → `Unavailable`, retryable | `TestVA1_6_HTTP503_Unavailable_Retryable` | **PASS** |
| VA-1.7 | HTTP 500 → `Unavailable`, retryable | `TestVA1_7_HTTP500_Unavailable_Retryable` | **PASS** |
| VA-1.8 | HTTP 502 → `Unavailable`, retryable | `TestVA1_8_HTTP502_Unavailable_Retryable` | **PASS** |
| VA-1.9 | DNS/TCP/TLS error → `Unavailable`, retryable | `TestVA1_9_NetworkFailure_Unavailable_Retryable`, `TestVA1_9_DNSFailure_Unavailable_Retryable`, `TestVA1_9_ConnectionRefused_Unavailable_Retryable` | **PASS** |
| VA-1.10 | Malformed JSON → `Internal`, not retryable | `TestVA1_10_MalformedJSON_Internal_NotRetryable`, `TestVA1_10_EmptyBody_Internal_NotRetryable` | **PASS** |
| VA-1.11 | Unknown venue status → `Internal`, not retryable | `TestVA1_11_UnknownVenueStatus_Internal_NotRetryable` | **PASS** |
| VA-1.12 | No bare Go errors escape adapter (F-1) | Code review: all return paths in `SubmitOrder`, `handleErrorResponse`, `parseOrderResponse` produce `*problem.Problem` or nil | **PASS** |
| VA-1.13 | No credentials in error messages (F-4) | `TestVA1_13_NoCredentialsInErrorMessages`: 7 HTTP codes tested with injected credentials; none leak into problem detail or message | **PASS** |

**VA-1 verdict: 13/13 PASS**

---

### RF-1: Retryable Flag Completeness (High — S314)

| Exit Criterion | Spec | Evidence | Verdict |
|---|---|---|---|
| RF-1.1 | Every `*problem.Problem` carries `Retryable` | Code review: all `problem.New()` calls include explicit `WithRetryable()` option; `TestRF1_1_AllErrorPaths_RetryableConsistency` covers 9 status codes | **PASS** |
| RF-1.2 | 429 → retryable | `TestVA1_5_HTTP429_Unavailable_Retryable` + `TestRF1_1` table entry | **PASS** |
| RF-1.3 | 503 → retryable | `TestVA1_6_HTTP503_Unavailable_Retryable` + `TestRF1_1` table entry | **PASS** |
| RF-1.4 | 5xx except 503 → retryable | `TestVA1_7_HTTP500`, `TestVA1_8_HTTP502` + `TestRF1_1` table entries | **PASS** |
| RF-1.5 | Network failure → retryable | `TestVA1_9` (3 variants: non-routable, DNS, connection refused) | **PASS** |
| RF-1.6 | Context deadline → retryable | `TestRF1_6_ContextDeadline_Retryable` | **PASS** |
| RF-1.7 | 401/403 → not retryable | `TestVA1_1`, `TestVA1_2` + `TestRF1_1` table entries | **PASS** |
| RF-1.8 | 400/422 → not retryable | `TestVA1_3`, `TestVA1_4` + `TestRF1_1` table entries | **PASS** |
| RF-1.9 | Parse failure → not retryable | `TestVA1_10` (malformed JSON, empty body) | **PASS** |
| RF-1.10 | Unknown error → not retryable | `TestVA1_11_UnknownVenueStatus_Internal_NotRetryable` | **PASS** |

**RF-1 verdict: 10/10 PASS**

---

## 2. Aggregate Scorecard

| Item | Priority | Exit Criteria | Passed | Verdict |
|---|---|---|---|---|
| EC-1 | Critical | 6 | 6 | **PASS** |
| EC-2 | Low | 5 | 5 | **PASS** |
| EC-3 | Medium | 6 | 6 | **PASS** |
| VA-1 | High | 13 | 13 | **PASS** |
| RF-1 | High | 10 | 10 | **PASS** |
| **Total** | — | **40** | **40** | **ALL PASS** |

---

## 3. Regression Verification

### Test Suites Executed (2026-03-21)

| Package | Result | Notes |
|---|---|---|
| `internal/application/execution` | **PASS** (10.310s) | All adapter, safety gate, staleness guard tests green |
| `internal/application/risk` | **PASS** (0.164s) | Risk evaluation, severity, exposure, drawdown tests green |
| `internal/interfaces/http/handlers` | **PASS** (0.160s) | All HTTP handler tests green |
| `internal/interfaces/http/routes` | **PASS** (0.164s) | All route registration tests green |
| `internal/adapters/clickhouse` | **PASS** (0.360s) | Writer pipeline, row mapping, parse tests green |
| `internal/application/analyticalclient` | **PASS** (0.163s) | Composite, funnel, disposition, multi-symbol tests green |
| `cmd/gateway` | **BUILD OK** | Binary compiles cleanly |
| `go vet ./...` | **CLEAN** | No static analysis warnings |

**Regression verdict: ZERO regressions detected.**

Paper pipeline, analytical surface, HTTP handlers, risk evaluation, and multi-symbol concurrency all unaffected by tranche changes.

---

## 4. Scope Inflation Audit

### Items Delivered

| Item | Charted? | Stage | Status |
|---|---|---|---|
| EC-1 | Yes (S312 charter) | S313 | Delivered |
| EC-2 | Yes (S312 charter) | S313 | Delivered |
| EC-3 | Yes (S312 charter) | S313 | Delivered |
| VA-1 | Yes (S312 charter) | S314 | Delivered |
| RF-1 | Yes (S312 charter) | S314 | Delivered |

No unchartered items were delivered. No additional code paths, endpoints, NATS subjects, ClickHouse schema changes, or new binaries were introduced.

**Scope inflation verdict: NONE. Exactly 5 items delivered per charter.**

---

## 5. Residual Gaps

### From S313

| ID | Gap | Severity | Disposition |
|---|---|---|---|
| R-S313-1 | Binance `newClientOrderId` acceptance not tested against real venue | Low | Closes during E2E implementation wave (I1) |
| R-S313-2 | Retry logic not implemented | Expected | Post-tranche per NG-6; blocked until EC-1 proven |
| R-S313-3 | Paper adapter does not use deterministic IDs | By design | Paper adapter is not a venue adapter |

### From S314

| ID | Gap | Severity | Disposition |
|---|---|---|---|
| R-S314-1 | No real Binance error corpus tested | Low | httptest covers all known patterns; closes in E2E |
| R-S314-2 | HTTP 418 (WAF) untested against real infra | Low | Testnet-only; WAF behavior is environment-specific |
| R-S314-3 | Partial fill + network failure combination | Medium | Out of scope per S306 NG-5; deferred to implementation wave |
| R-S314-4 | Body read failure after HTTP 200 is non-retryable | Design decision | Documented and accepted |

### Residual Classification

| Category | Count |
|---|---|
| Closes in E2E (I1) | 3 (R-S313-1, R-S314-1, R-S314-2) |
| Post-tranche by charter | 1 (R-S313-2) |
| By design | 2 (R-S313-3, R-S314-4) |
| Deferred to implementation wave | 1 (R-S314-3) |
| **Blockers for implementation wave** | **0** |

No residual is a blocker for opening the implementation wave.

---

## 6. Constraint Compliance (CN-1 through CN-7)

| Constraint | Requirement | Status |
|---|---|---|
| CN-1 | No ClickHouse schema changes | **Compliant** |
| CN-2 | No new NATS subjects or KV buckets | **Compliant** |
| CN-3 | No new binaries or services | **Compliant** |
| CN-4 | No changes to derive or store binaries | **Compliant** |
| CN-5 | Paper pipeline zero-regression | **Compliant** (tests green) |
| CN-6 | No design documents produced | **Compliant** (only implementation + exit criteria docs) |
| CN-7 | VenuePort interface unchanged | **Compliant** (`ClientOrderID` field added to `VenueOrderReceipt`, not to interface signature) |

---

## 7. Non-Goal Compliance

All 25 non-goals (NG-1 through NG-25) remain unviolated. No E2E venue calls, no retry infrastructure, no circuit breakers, no OMS, no new endpoints, no VenuePort redesign, no multi-symbol testing beyond existing baseline, no production readiness assessment.
