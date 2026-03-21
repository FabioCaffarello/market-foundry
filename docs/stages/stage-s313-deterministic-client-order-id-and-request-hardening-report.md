# Stage S313 — Deterministic Client Order ID and Request Hardening

**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave / Adapter Hardening Tranche
**Predecessor:** S312 (Adapter Hardening Tranche Charter)
**Successor:** S314 (Error Classification and Retryable Flag Completeness)

---

## 1. Executive Summary

S313 delivers the first three items of the adapter hardening tranche: deterministic client order ID derivation (EC-1), response body size cap validation (EC-2), and per-request context deadline enforcement (EC-3). EC-1 was the root blocker from S311 — without deterministic client order IDs, retry infrastructure and timeout resolution are permanently blocked. All three items are now implemented, unit-tested, and documented. The adapter is hardened for the request/response envelope without inflating scope into retry logic or venue integration.

---

## 2. Hardening Implemented

### 2.1 EC-1: Deterministic Client Order ID

**Implementation:** `ClientOrderID(intent)` function derives a 32-character hex string from `SHA-256(intent.DeduplicationKey())`.

**Changes:**
- New function `ClientOrderID()` in `internal/application/execution/client_order_id.go`
- `BinanceFuturesTestnetAdapter.SubmitOrder()` now includes `newClientOrderId` in HTTP params
- `VenueOrderReceipt` gains `ClientOrderID` field for traceability
- `binanceOrderResponse` gains `ClientOrderID` field for Binance response echo

**Properties proven:**
- Deterministic: same intent → same ID (1000 iterations)
- Unique: varying any key field → different ID
- Binance-compatible: 32 hex chars, within 36-char limit
- No random inputs: no `rand`, no `time.Now()` in derivation

### 2.2 EC-2: Response Body Size Cap

**Implementation:** Already present via `io.LimitReader(resp.Body, 64*1024)`.

**New validation:**
- Oversized body truncation tested (128 KB body → truncated at 64 KB)
- Corrupted JSON from truncation produces `problem.Internal`, non-retryable
- Normal responses unaffected (covered by existing happy-path tests)

### 2.3 EC-3: Per-Request Context Deadline

**Implementation:** Dual-layer enforcement:
1. Actor layer wraps with `context.WithTimeout(ctx, cfg.SubmitTimeout)`
2. Adapter layer adds defensive fallback: if caller omits deadline, enforces 10s default

**New validation:**
- Context deadline exceeded → error returned, retryable
- Intent not mutated after timeout (PGR-08 preserved)
- Default deadline enforced when caller provides `context.Background()`
- Normal responses within deadline unaffected

---

## 3. Files Changed

### New Files

| File | Purpose |
|------|---------|
| `internal/application/execution/client_order_id.go` | `ClientOrderID()` derivation function |
| `internal/application/execution/client_order_id_test.go` | EC-1.1, EC-1.2, EC-1.3, EC-1.6 tests |
| `docs/architecture/deterministic-client-order-id-and-request-hardening.md` | Design and invariants |
| `docs/architecture/client-order-id-body-cap-deadline-invariants-and-limits.md` | Limits and assumptions |
| `docs/stages/stage-s313-deterministic-client-order-id-and-request-hardening-report.md` | This report |

### Modified Files

| File | Change |
|------|--------|
| `internal/application/execution/binance_futures_testnet_adapter.go` | Added `newClientOrderId` to params, `ClientOrderID` to receipt, defensive deadline, `binanceOrderResponse.ClientOrderID` field |
| `internal/application/execution/binance_futures_testnet_adapter_test.go` | Added EC-1.4, EC-1.5, EC-2.2, EC-2.3, EC-2.4, EC-3.1, EC-3.3, EC-3.5 tests |
| `internal/application/ports/venue.go` | Added `ClientOrderID` field to `VenueOrderReceipt` |

### Unmodified Files

| File | Reason |
|------|--------|
| `internal/application/execution/paper_venue_adapter.go` | Paper adapter uses random IDs by design (no venue interaction) |
| `internal/domain/execution/execution.go` | `DeduplicationKey()` already correct |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Context wrapping already correct; compiles cleanly with new receipt field |

---

## 4. Exit Criteria Verification

### 4.1 EC-1: Client Order ID Derivation — ALL 6 PASS

| # | Criterion | Test | Result |
|---|-----------|------|--------|
| EC-1.1 | Same intent → same ID | `TestClientOrderID_Deterministic` | PASS |
| EC-1.2 | Different intents → different IDs | `TestClientOrderID_Uniqueness` | PASS |
| EC-1.3 | Conforms to Binance format | `TestClientOrderID_BinanceFormat` | PASS |
| EC-1.4 | Receipt includes `ClientOrderID` | `TestBinanceAdapter_ClientOrderID_InReceipt` | PASS |
| EC-1.5 | `newClientOrderId` in HTTP request | `TestBinanceAdapter_ClientOrderID_InHTTPRequest` | PASS |
| EC-1.6 | No random/time-varying inputs | `TestClientOrderID_NoRandomInputs` (1000 iterations) | PASS |

### 4.2 EC-2: Response Body Size Cap — ALL 5 PASS

| # | Criterion | Test | Result |
|---|-----------|------|--------|
| EC-2.1 | All reads use `io.LimitReader` | Code review: single read site, wrapped | PASS |
| EC-2.2 | Oversized body truncated | `TestBinanceAdapter_OversizedBody_Truncated` | PASS |
| EC-2.3 | Truncated → `problem.Internal` | `TestBinanceAdapter_OversizedBody_CorruptedJSON` | PASS |
| EC-2.4 | Truncated → non-retryable | `TestBinanceAdapter_OversizedBody_CorruptedJSON` | PASS |
| EC-2.5 | Normal responses unaffected | All existing happy-path tests | PASS |

### 4.3 EC-3: Per-Request Context Deadline — ALL 6 PASS

| # | Criterion | Test | Result |
|---|-----------|------|--------|
| EC-3.1 | All venue calls have deadline | Code review: adapter enforces fallback if missing | PASS |
| EC-3.2 | Timeout configurable | Code review: `VenueAdapterConfig.SubmitTimeout` | PASS |
| EC-3.3 | Slow response → cancellation | `TestBinanceAdapter_ContextDeadline_Exceeded` | PASS |
| EC-3.4 | Timeout → `Unavailable`, retryable | `TestBinanceAdapter_SubmitOrder_Timeout` | PASS |
| EC-3.5 | Intent unmutated after timeout | `TestBinanceAdapter_ContextDeadline_IntentUnmutated` | PASS |
| EC-3.6 | Normal responses unaffected | `TestBinanceAdapter_DefaultDeadline_Enforced` | PASS |

### 4.4 Regression Check

All 80+ tests in `internal/application/execution` pass. Actor layer compiles cleanly. No regressions.

---

## 5. Residual Limits

| # | Residual | Severity | Impact |
|---|----------|----------|--------|
| R-1 | Binance `newClientOrderId` acceptance not tested against real venue | Low | Verified against API spec; real validation in E2E (I1) |
| R-2 | Retry logic not implemented (NG-6) | Expected | EC-1 unblocks retry; implementation is post-tranche |
| R-3 | Paper adapter does not use deterministic IDs | By design | Paper fills are instant; no retry/reconciliation concern |

**Assessment:** No blocker residuals. All residuals are expected non-goals per S312 charter.

---

## 6. Preparation for S314

S314 addresses the remaining two tranche items:
- **VA-1**: Error classification completeness (13 exit criteria)
- **RF-1**: Retryable flag completeness (10 exit criteria)

### Recommended Focus

1. Complete HTTP status code coverage (VA-1.1–VA-1.8): expand `handleErrorResponse` to cover all 8 failure classes explicitly
2. Add DNS/TCP/TLS error classification (VA-1.9): test with unreachable host
3. Verify malformed JSON handling (VA-1.10): already partially covered
4. Ensure every `*problem.Problem` has explicit retryable assignment (RF-1.1)
5. Cross-reference VA-1 and RF-1 tests to avoid duplication (many criteria overlap)

### Pre-conditions Met

- EC-1 is implemented → retry infrastructure can be designed (post-tranche)
- EC-2 is validated → oversized response handling is proven
- EC-3 is enforced → timeout errors are classified and retryable

The adapter's request/response envelope is hardened. S314 completes the error classification layer.

---

## 7. Guard Rails Compliance

| Guard Rail | Status |
|------------|--------|
| No retry infrastructure opened | COMPLIANT |
| No real venue calls | COMPLIANT |
| VenuePort interface unchanged | COMPLIANT (only `VenueOrderReceipt` struct gained a field) |
| No adapter redesign | COMPLIANT — surgical additions only |
| Scope: exactly 3 items (EC-1, EC-2, EC-3) | COMPLIANT |

---

*Delivered: 2026-03-21 — Stage S313, Phase 30*
