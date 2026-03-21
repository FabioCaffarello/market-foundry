# Problem Mapping, Retryability Semantics, and Coverage

**Stage:** S314 — Error Classification and Retryability Completion
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave (Adapter Hardening Tranche)
**Companion:** `error-classification-and-retryability-completion.md`

---

## 1. Purpose

This document provides the test evidence matrix for VA-1 (Error Classification Completeness, 13 criteria) and RF-1 (Retryable Flag Completeness, 10 criteria). Each exit criterion maps to one or more unit tests with pass/fail status.

---

## 2. VA-1 Evidence Matrix

### 2.1 HTTP Error Classification

| # | Criterion | Test | Code | Retryable | Status |
|---|----------|------|------|-----------|--------|
| VA-1.1 | HTTP 401 → InvalidArgument, non-retryable | `TestVA1_1_HTTP401_InvalidArgument_NotRetryable` | `VAL_INVALID_ARGUMENT` | `false` | PASS |
| VA-1.2 | HTTP 403 → InvalidArgument, non-retryable | `TestVA1_2_HTTP403_InvalidArgument_NotRetryable` | `VAL_INVALID_ARGUMENT` | `false` | PASS |
| VA-1.3 | HTTP 400 → InvalidArgument, non-retryable | `TestVA1_3_HTTP400_InvalidArgument_NotRetryable` | `VAL_INVALID_ARGUMENT` | `false` | PASS |
| VA-1.4 | HTTP 422 → InvalidArgument, non-retryable | `TestVA1_4_HTTP422_InvalidArgument_NotRetryable` | `VAL_INVALID_ARGUMENT` | `false` | PASS |
| VA-1.5 | HTTP 429 → Unavailable, retryable | `TestVA1_5_HTTP429_Unavailable_Retryable` | `SYS_UNAVAILABLE` | `true` | PASS |
| VA-1.6 | HTTP 503 → Unavailable, retryable | `TestVA1_6_HTTP503_Unavailable_Retryable` | `SYS_UNAVAILABLE` | `true` | PASS |
| VA-1.7 | HTTP 500 → Unavailable, retryable | `TestVA1_7_HTTP500_Unavailable_Retryable` | `SYS_UNAVAILABLE` | `true` | PASS |
| VA-1.8 | HTTP 502 → Unavailable, retryable | `TestVA1_8_HTTP502_Unavailable_Retryable` | `SYS_UNAVAILABLE` | `true` | PASS |

### 2.2 Network and Parse Failures

| # | Criterion | Test(s) | Code | Retryable | Status |
|---|----------|---------|------|-----------|--------|
| VA-1.9 | DNS/TCP/TLS → Unavailable, retryable | `TestVA1_9_NetworkFailure_Unavailable_Retryable`, `TestVA1_9_DNSFailure_Unavailable_Retryable`, `TestVA1_9_ConnectionRefused_Unavailable_Retryable` | `SYS_UNAVAILABLE` | `true` | PASS |
| VA-1.10 | Malformed JSON → Internal, non-retryable | `TestVA1_10_MalformedJSON_Internal_NotRetryable`, `TestVA1_10_EmptyBody_Internal_NotRetryable` | `SYS_INTERNAL` | `false` | PASS |
| VA-1.11 | Unknown venue status → Internal, non-retryable | `TestVA1_11_UnknownVenueStatus_Internal_NotRetryable` | `SYS_INTERNAL` | `false` | PASS |

### 2.3 Code Review Criteria

| # | Criterion | Evidence | Status |
|---|----------|---------|--------|
| VA-1.12 | No bare Go errors escape (F-1) | All 9 return paths in `SubmitOrder`, `handleErrorResponse`, `parseOrderResponse`, and `mapBinanceStatus` return `*problem.Problem` or nil. Interface signature `(VenueOrderReceipt, *problem.Problem)` enforces at compile time. | PASS |
| VA-1.13 | No credentials in error messages (F-4) | `TestVA1_13_NoCredentialsInErrorMessages` — tests all 7 HTTP error codes with known API key/secret, asserts neither appears in `prob.Error()`. Code review: no credential field in any format string or details map. | PASS |

**VA-1 Result: 13/13 PASS**

---

## 3. RF-1 Evidence Matrix

| # | Criterion | Test(s) | Retryable | Status |
|---|----------|---------|-----------|--------|
| RF-1.1 | Every problem carries correct Retryable | `TestRF1_1_AllErrorPaths_RetryableConsistency` (9 sub-tests: 401, 403, 400, 422, 429, 500, 502, 503, 504) | Per-class | PASS |
| RF-1.2 | 429 → retryable | `TestVA1_5_HTTP429_Unavailable_Retryable` | `true` | PASS |
| RF-1.3 | 503 → retryable | `TestVA1_6_HTTP503_Unavailable_Retryable` | `true` | PASS |
| RF-1.4 | 5xx except 503 → retryable | `TestVA1_7_HTTP500_Unavailable_Retryable`, `TestVA1_8_HTTP502_Unavailable_Retryable`, `TestRF1_1.../504_timeout` | `true` | PASS |
| RF-1.5 | Network failure → retryable | `TestVA1_9_NetworkFailure_Unavailable_Retryable`, `TestVA1_9_DNSFailure_Unavailable_Retryable`, `TestVA1_9_ConnectionRefused_Unavailable_Retryable` | `true` | PASS |
| RF-1.6 | Context deadline → retryable | `TestRF1_6_ContextDeadline_Retryable` | `true` | PASS |
| RF-1.7 | 401/403 → non-retryable | `TestVA1_1_HTTP401_...`, `TestVA1_2_HTTP403_...` | `false` | PASS |
| RF-1.8 | 400/422 → non-retryable | `TestVA1_3_HTTP400_...`, `TestVA1_4_HTTP422_...` | `false` | PASS |
| RF-1.9 | Parse failure → non-retryable | `TestVA1_10_MalformedJSON_...`, `TestVA1_10_EmptyBody_...` | `false` | PASS |
| RF-1.10 | Unknown error → non-retryable | `TestVA1_11_UnknownVenueStatus_...` | `false` | PASS |

**RF-1 Result: 10/10 PASS**

---

## 4. Supplemental Coverage

Beyond the exit criteria, S314 delivers additional coverage:

| Test | Purpose |
|------|---------|
| `TestVA1_ErrorDetails_VenueHTTPStatus` | Verifies structured details (`venue_http_status`, `venue_error_code`) in problem objects |
| `TestVA1_StatusMapping_AllKnownStatuses` | Verifies all 7 Binance status values map correctly to domain statuses |

---

## 5. Retryability Decision Model

```
                      ┌──────────────┐
                      │ Error Source  │
                      └──────┬───────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
         │ Network │   │  HTTP   │   │  Parse  │
         │ Layer   │   │  Layer  │   │  Layer  │
         └────┬────┘   └────┬────┘   └────┬────┘
              │              │              │
              │         ┌────┼────┐         │
              │         │    │    │         │
         retryable   ┌──▼─┐ │ ┌──▼──┐   not
              │       │4xx │ │ │5xx  │  retryable
              │       └──┬─┘ │ └──┬──┘     │
              │          │   │    │        │
              │       not │ 429  retryable │
              │     retry │ retry   │      │
              │          │   │      │      │
              ▼          ▼   ▼      ▼      ▼
         Unavailable  Invalid  Unavail  Internal
         retry=true   retry=F  retry=T  retry=F
```

---

## 6. Residual Gaps

| # | Gap | Impact | Resolution Path |
|---|-----|--------|----------------|
| R-1 | No real Binance error corpus tested | Low — httptest covers all known HTTP patterns | Closes naturally in E2E (I1) |
| R-2 | Binance WAF (HTTP 418) untested against real infrastructure | Low — falls into 4xx catch-all correctly | Testable only with real venue |
| R-3 | Partial fill + network failure combination | Medium — requires async fill feed | Out of scope (S306 NG-5) |
| R-4 | Body read failure after HTTP 200 is non-retryable | Design decision, not gap | Documented in companion doc §2.3 |

None of these residuals block the tranche gate (S315).

---

*Delivered: 2026-03-21 — Stage S314, Phase 30*
