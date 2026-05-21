# Stage S325 — Venue Error Code Aware Classification Report

> **Status:** Complete
> **Predecessor:** S322 (Post-200 Reconciliation), S323 (Retry Coordination Hardening), S324 (Retry Observability)
> **Scope:** Surgical enrichment of error classification using Binance venue error codes
> **Gap closed:** R-S320-4 (venue error codes unused for classification)

## 1. Executive Summary

S325 enriches the venue error classification to use Binance-specific error codes where they provide stronger semantic signal than the HTTP status alone. Three error codes (-1001, -1003, -1015) were identified as cases where the HTTP 4xx status misrepresents the actual failure class: all three are transient, retryable conditions incorrectly classified as permanent client errors.

The implementation adds a single method (`classifyByVenueErrorCode`) that intercepts 4xx responses before the HTTP-based switch, overriding classification only for the three mapped codes. All other codes fall through to the existing classification unchanged.

**Result**: 10 new tests (22 total with subtests), zero regressions in the existing 80+ test suite, three false-permanent classifications eliminated.

## 2. Enrichment Delivered

### 2.1 Classification Overrides

| HTTP Status | Venue Code | Before S325 | After S325 |
|------------|-----------|-------------|------------|
| 400 | -1001 (DISCONNECTED) | InvalidArgument, non-retryable | Unavailable, retryable |
| 418 | -1003 (TOO_MANY_REQUESTS) | InvalidArgument, non-retryable | Unavailable, retryable |
| 400 | -1015 (TOO_MANY_ORDERS) | InvalidArgument, non-retryable | Unavailable, retryable |

### 2.2 Safety Guards

- Auth errors (401/403) are immune to override
- HTTP 429 is immune to override (already correct)
- 5xx errors bypass override (already retryable)
- Unmapped codes fall through to existing HTTP-based classification

### 2.3 Observability Enhancement

Overridden classifications include `venue_error_class` in problem details:
- `venue_internal` for code -1001
- `ip_rate_limit` for code -1003
- `order_rate_limit` for code -1015

## 3. Files Changed

| File | Action | Description |
|------|--------|-------------|
| `internal/application/execution/binance_futures_testnet_adapter.go` | Modified | Added `classifyByVenueErrorCode` method and override hook in `handleErrorResponse` |
| `internal/application/execution/venue_error_code_classification_test.go` | New | 10 tests covering overrides, fallthrough, safety guards, and regression |
| `docs/architecture/venue-error-code-aware-classification-enrichment.md` | New | Design decision, override model, and code selection rationale |
| `docs/architecture/error-code-mapping-coverage-benefits-and-limitations.md` | New | Coverage analysis, benefits, and limitations |
| `docs/stages/stage-s325-venue-error-code-aware-classification-report.md` | New | This report |

## 4. Tests and Evidence

### 4.1 New Tests (S325)

```
=== RUN   TestEC_S325_1_HTTP400_Code1001_VenueInternal_Retryable       --- PASS
=== RUN   TestEC_S325_2_HTTP418_Code1003_IPRateLimit_Retryable         --- PASS
=== RUN   TestEC_S325_3_HTTP400_Code1015_OrderRateLimit_Retryable      --- PASS
=== RUN   TestEC_S325_4_HTTP400_Code1121_NoOverride_InvalidArgument    --- PASS
=== RUN   TestEC_S325_5_HTTP401_Code1001_AuthNotOverridden             --- PASS
=== RUN   TestEC_S325_6_HTTP429_Code1015_AlreadyCorrect                --- PASS
=== RUN   TestEC_S325_7_HTTP500_Code1001_5xxNotOverridden              --- PASS
=== RUN   TestEC_S325_8_HTTP400_NoCode_FallsThrough                    --- PASS
=== RUN   TestEC_S325_9_CredentialRedaction_WithOverride               --- PASS (3 sub)
=== RUN   TestEC_S325_10_ExistingClassification_Unchanged              --- PASS (9 sub)
PASS — 10/10 tests (22 total with subtests), 0 regressions
```

### 4.2 Test Coverage Matrix

| Test ID | What It Verifies |
|---------|-----------------|
| EC-S325-1 | Code -1001 override: Unavailable + retryable |
| EC-S325-2 | Code -1003 override: Unavailable + retryable |
| EC-S325-3 | Code -1015 override: Unavailable + retryable |
| EC-S325-4 | Unmapped code (-1121) falls through to HTTP-based classification |
| EC-S325-5 | Auth (401) immune to override even with mapped code |
| EC-S325-6 | HTTP 429 immune to override (already correct) |
| EC-S325-7 | 5xx bypasses override (already retryable) |
| EC-S325-8 | No venue code (code=0) falls through |
| EC-S325-9 | Credential redaction preserved for all 3 override paths |
| EC-S325-10 | Full regression matrix: 9 HTTP+code combinations unchanged |

### 4.3 Regression Evidence

```
ok  internal/application/execution  31.898s  (full suite, zero failures)
```

## 5. Limits Remaining

| ID | Gap | Risk | Disposition |
|----|-----|------|-------------|
| R-S320-6 | No per-error-class differentiated retry policies | Low | Out of scope — standard backoff works for all retryable classes |
| R-S325-1 | No real-world error code corpus | Low | Conservative mapping; unmapped codes default to HTTP classification |
| R-S325-2 | No Retry-After header extraction | Low | Standard exponential backoff sufficient for testnet |
| R-S325-3 | Mapping is Binance-specific | Low | Override scoped to adapter; each venue gets its own mapping |

## 6. Invariants Preserved

| Invariant | Source | Verification |
|-----------|--------|-------------|
| F-1: No bare errors | S308 | All override paths return Problem |
| F-4: No credentials in errors | S314 | EC-S325-9: 3 override paths verified |
| RF-1: Retryable flag accuracy | S314 | EC-S325-10: 9-case regression matrix |
| VA-1: Classification completeness | S314 | All 8 failure classes verified with overrides |

## 7. Gap Closure

| Gap ID | Description | Closed By |
|--------|-------------|-----------|
| R-S320-1 | No reconciliation for body-read-failure-after-200 | S322 |
| R-S320-2 | No global retry deadline | S323 |
| R-S320-3 | Kill switch not checked during retry backoff | S323 |
| R-S320-4 | Venue error codes unused for classification | **S325 (this stage)** |
| R-S320-5 | No structured retry metrics | S324 |
| R-S320-6 | No per-error-class differentiated retry policies | Open (low risk) |

With S325 complete, 5 of 6 S320 residual gaps are closed. The remaining gap (R-S320-6) is low-risk and does not block the evidence gate.

## 8. Preparation for S326

S326 is the venue closure tranche gate. Recommended preparation:

1. **Evidence compilation**: Aggregate all S321–S325 test evidence into a single gate summary.
2. **Gap disposition**: Document final disposition of R-S320-6 (accept vs defer).
3. **Contract freeze validation**: Verify Problem type, VenuePort, and retry interfaces are stable.
4. **Coverage check**: Ensure all 8 failure classes from S314 remain fully covered post-enrichment.
