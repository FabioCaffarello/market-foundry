# Error Classification and Retryability Completion

**Stage:** S314 — Error Classification and Retryability Completion
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave (Adapter Hardening Tranche)
**Companion:** `problem-mapping-retryability-semantics-and-coverage.md`

---

## 1. Purpose

This document defines the complete error taxonomy for `BinanceFuturesTestnetAdapter`, mapping every failure path to a `*problem.Problem` with correct classification and retryability. It closes **VA-1** (Error Classification Completeness) and **RF-1** (Retryable Flag Completeness) from the adapter hardening tranche (S312).

---

## 2. Error Taxonomy

The adapter produces errors in 8 failure classes, derived from S308 §2.5 C-FAIL taxonomy and governed by S310 §6.2 retryability semantics.

### 2.1 Failure Classes

| # | Class | Trigger | Problem Code | Retryable | Rationale |
|---|-------|---------|-------------|-----------|-----------|
| 1 | Authentication | HTTP 401, 403 | `VAL_INVALID_ARGUMENT` | `false` | Credentials are wrong; retrying with same creds is pointless |
| 2 | Client Error | HTTP 400, 422, other 4xx | `VAL_INVALID_ARGUMENT` | `false` | Request is malformed; venue will reject again |
| 3 | Rate Limit | HTTP 429 | `SYS_UNAVAILABLE` | `true` | Transient; venue will accept after backoff |
| 4 | Venue Unavailable | HTTP 503 | `SYS_UNAVAILABLE` | `true` | Transient venue outage |
| 5 | Server Error | HTTP 500, 502, 5xx | `SYS_UNAVAILABLE` | `true` | Transient server failure |
| 6 | Network Failure | DNS, TCP, TLS, connection refused | `SYS_UNAVAILABLE` | `true` | Infrastructure-level transient failure |
| 7 | Parse Failure | Malformed JSON, empty body, truncated body | `SYS_INTERNAL` | `false` | Deterministic parse failure; same input → same failure |
| 8 | Unknown Status | Unmapped venue order status | `SYS_INTERNAL` | `false` | Requires code update, not retry |

### 2.2 Pre-Request Failures

| Class | Trigger | Problem Code | Retryable | Notes |
|-------|---------|-------------|-----------|-------|
| Request Build | `http.NewRequestWithContext` fails | `SYS_INTERNAL` | `false` | Programming error or invalid URL; not transient |
| Body Read | `io.ReadAll` on response body fails | `SYS_INTERNAL` | `false` | Post-200 failure; order may have been accepted |

### 2.3 Design Decision: Body Read Failures Are Non-Retryable

When a body read fails after HTTP 200, the venue has already accepted the order. Retrying would risk duplicate execution even with EC-1 dedup (the venue may have already filled). The correct recovery path is status reconciliation, not retry.

---

## 3. Structured Error Details

Every HTTP error response includes structured details for observability:

```json
{
  "code": "SYS_UNAVAILABLE",
  "message": "venue server error (HTTP 500)",
  "retryable": true,
  "details": {
    "venue_http_status": 500,
    "venue_error_code": -1001
  }
}
```

| Detail Key | Type | Present When |
|-----------|------|-------------|
| `venue_http_status` | `int` | Always (on HTTP error responses) |
| `venue_error_code` | `int` | When Binance returns a non-zero error code |

These details are never included for network-level failures (no HTTP response) or parse failures (HTTP 200 received).

---

## 4. Binance Status Mapping

The adapter maps Binance Futures order statuses to domain statuses:

| Binance Status | Domain Status | Terminal |
|---------------|--------------|---------|
| `NEW` | `accepted` | No |
| `FILLED` | `filled` | Yes |
| `PARTIALLY_FILLED` | `partially_filled` | No |
| `CANCELED` / `CANCELLED` | `cancelled` | Yes |
| `REJECTED` / `EXPIRED` | `rejected` | Yes |
| (anything else) | Error: `SYS_INTERNAL` | N/A |

Any unmapped status produces a non-retryable `SYS_INTERNAL` error (class 8).

---

## 5. Credential Safety (F-4 Invariant)

No error message or problem detail contains API keys, secrets, or any credential material. This is enforced by:

1. **Message construction**: All error messages use format strings with only HTTP status codes, Binance error codes, and Binance error messages (which are public API responses).
2. **No credential fields in details**: The `details` map contains only `venue_http_status` and `venue_error_code`.
3. **Cause wrapping**: The `Cause` field wraps Go `net/http` errors, which never contain our credentials (they are sent in headers, not reflected in error messages).

---

## 6. F-1 Invariant: No Bare Errors Escape

Every return path in `SubmitOrder` produces either:
- `(receipt, nil)` on success
- `(zero, *problem.Problem)` on failure

No bare `error` value is returned. The `VenuePort` interface signature enforces this at compile time.

---

## 7. Limits and Non-Goals

| Item | Status | Notes |
|------|--------|-------|
| Retry loops | Not implemented | NG-6; blocked until EC-1 proven in E2E |
| Circuit breaker | Not implemented | NG-7; testnet only |
| Rate limiter (self-imposed) | Not implemented | NG-8; Binance testnet has generous limits |
| Binance WAF/IP ban (HTTP 418) | Falls into class 2 (4xx) | Non-retryable; manual IP resolution needed |
| Failure injection testing | Not done | NG-24; requires E2E first |
| Real venue error corpus | Not available | Testnet error codes verified against Binance API docs |

---

*Delivered: 2026-03-21 — Stage S314, Phase 30*
