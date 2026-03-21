# Adapter Hardening Items — Exit Criteria and Non-Goals

**Stage:** S312 — Adapter Hardening Tranche Charter
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Companion:** `adapter-hardening-tranche-charter-and-scope-freeze.md`

---

## 1. Purpose

This document defines the **per-item exit criteria** and the **explicit non-goals** for the Adapter Hardening Tranche (S313–S315). Each of the 5 tranche items has objective, testable acceptance criteria derived from S308 contracts and S310 guard rails. The non-goals section prevents scope inflation by enumerating everything the tranche explicitly does not address.

---

## 2. Item Exit Criteria

### 2.1 EC-1: Client Order ID Derivation

**Source spec:** S308 §3.1 IDEM-3 — "Client order ID must be deterministically derived from `DeduplicationKey()`."

| # | Exit Criterion | Verification Method |
|---|---------------|-------------------|
| EC-1.1 | `ClientOrderID(intent)` returns the same value for the same `ExecutionIntent` across multiple calls | Unit test: construct intent, call twice, assert equal |
| EC-1.2 | `ClientOrderID(intent)` returns different values for intents that differ in any key field (type, source, symbol, timeframe, unix timestamp) | Unit test: vary each field individually, assert all IDs are distinct |
| EC-1.3 | Generated ID conforms to Binance `newClientOrderId` format constraints (alphanumeric + limited special chars, max length) | Unit test: validate format against Binance API specification |
| EC-1.4 | `VenueOrderRequest` includes `newClientOrderId` field populated from `ClientOrderID()` | Unit test: construct request, assert field is non-empty and matches derivation |
| EC-1.5 | `newClientOrderId` is present in the HTTP request body sent to venue | httptest: intercept request, assert `newClientOrderId` parameter present |
| EC-1.6 | Derivation does not use random or time-varying inputs beyond what is in the dedup key | Code review: no `rand`, no `time.Now()` in derivation path |

**Pass threshold:** ALL 6 criteria must pass.

### 2.2 EC-2: Response Body Size Cap

**Source spec:** S310 §3.1 PGR-14 — "`io.LimitReader(body, 64*1024)`; reject if exceeded."

| # | Exit Criterion | Verification Method |
|---|---------------|-------------------|
| EC-2.1 | All HTTP response body reads use `io.LimitReader` with 64 KB limit | Code review: grep for `http.Response.Body` reads; all wrapped |
| EC-2.2 | Response body exceeding 64 KB is truncated at the read boundary | Unit test: httptest returns 128 KB body; reader stops at 64 KB |
| EC-2.3 | Truncated response produces a parse error classified as `problem.Internal` | Unit test: oversized body → `*problem.Problem` with Internal category |
| EC-2.4 | Truncated response is not retryable | Unit test: oversized body → `Retryable == false` |
| EC-2.5 | Normal-sized responses (< 64 KB) are unaffected | Unit test: httptest returns valid response < 64 KB; parsed correctly |

**Pass threshold:** ALL 5 criteria must pass.

### 2.3 EC-3: Per-Request Context Deadline

**Source spec:** S310 §3.1 PGR-03 — "Context deadline on all venue calls."
**Default timeout:** 10 seconds (S310 §7.1).

| # | Exit Criterion | Verification Method |
|---|---------------|-------------------|
| EC-3.1 | Every `VenuePort.SubmitOrder` call is wrapped with `context.WithTimeout` | Code review: no code path reaches venue HTTP without deadline |
| EC-3.2 | Timeout duration is configurable (not hard-coded) | Code review: timeout value comes from config, not a constant |
| EC-3.3 | Slow venue response triggers context cancellation | Unit test: httptest delays > timeout; `SubmitOrder` returns error |
| EC-3.4 | Timeout error is classified as `problem.Unavailable` with `Retryable == true` | Unit test: timeout → correct problem category and retryable flag |
| EC-3.5 | Intent state after timeout remains `submitted` (PGR-08 preserved) | Unit test: timeout does not mutate intent status |
| EC-3.6 | Normal venue responses within deadline are unaffected | Unit test: httptest responds within timeout; fill parsed correctly |

**Pass threshold:** ALL 6 criteria must pass.

### 2.4 VA-1: Error Classification Completeness

**Source spec:** S308 §2.5 C-FAIL — 8 failure classes with problem categories.
**Invariant:** F-1 — "Every error path returns a `*problem.Problem`."

| # | Exit Criterion | Verification Method |
|---|---------------|-------------------|
| VA-1.1 | HTTP 401 → `problem.InvalidArgument`, `Retryable == false` | Unit test |
| VA-1.2 | HTTP 403 → `problem.InvalidArgument`, `Retryable == false` | Unit test |
| VA-1.3 | HTTP 400 → `problem.InvalidArgument`, `Retryable == false` | Unit test |
| VA-1.4 | HTTP 422 → `problem.InvalidArgument`, `Retryable == false` | Unit test |
| VA-1.5 | HTTP 429 → `problem.Unavailable`, `Retryable == true` | Unit test |
| VA-1.6 | HTTP 503 → `problem.Unavailable`, `Retryable == true` | Unit test |
| VA-1.7 | HTTP 500 → `problem.Unavailable`, `Retryable == true` | Unit test |
| VA-1.8 | HTTP 502 → `problem.Unavailable`, `Retryable == true` | Unit test |
| VA-1.9 | DNS/TCP/TLS error → `problem.Unavailable`, `Retryable == true` | Unit test |
| VA-1.10 | Malformed JSON response → `problem.Internal`, `Retryable == false` | Unit test |
| VA-1.11 | Unknown/unmapped venue status → `problem.Internal`, `Retryable == false` | Unit test |
| VA-1.12 | No bare Go errors escape the adapter (F-1 invariant) | Code review: all return paths produce `*problem.Problem` or nil |
| VA-1.13 | Error messages never contain credentials or API keys (F-4 invariant) | Code review: no secret material in problem details |

**Pass threshold:** ALL 13 criteria must pass.

### 2.5 RF-1: Retryable Flag Completeness

**Source spec:** S310 §6.2, S308 F-2 — "Retryable flag is set on all transient failures."

| # | Exit Criterion | Verification Method |
|---|---------------|-------------------|
| RF-1.1 | Every `*problem.Problem` returned by the adapter carries a `Retryable` field | Code review: no problem construction without explicit retryable assignment |
| RF-1.2 | Rate limit (429) → `Retryable == true` | Unit test (may overlap VA-1.5) |
| RF-1.3 | Venue unavailable (503) → `Retryable == true` | Unit test (may overlap VA-1.6) |
| RF-1.4 | Server error (5xx except 503) → `Retryable == true` | Unit test |
| RF-1.5 | Network failure (DNS/TCP/TLS) → `Retryable == true` | Unit test |
| RF-1.6 | Context deadline exceeded → `Retryable == true` | Unit test (may overlap EC-3.4) |
| RF-1.7 | Authentication error (401/403) → `Retryable == false` | Unit test (may overlap VA-1.1/VA-1.2) |
| RF-1.8 | Client error (400/422) → `Retryable == false` | Unit test (may overlap VA-1.3/VA-1.4) |
| RF-1.9 | Parse failure → `Retryable == false` | Unit test (may overlap VA-1.10) |
| RF-1.10 | Unknown error → `Retryable == false` | Unit test (may overlap VA-1.11) |

**Pass threshold:** ALL 10 criteria must pass.

---

## 3. Tranche Exit Criteria (Gate at S315)

The tranche gate verifies the aggregate result of S313–S314.

| # | Gate Criterion | Verification |
|---|---------------|-------------|
| G-1 | EC-1 passes all 6 exit criteria (EC-1.1–EC-1.6) | S313 test results |
| G-2 | EC-2 passes all 5 exit criteria (EC-2.1–EC-2.5) | S313 test results |
| G-3 | EC-3 passes all 6 exit criteria (EC-3.1–EC-3.6) | S313 test results |
| G-4 | VA-1 passes all 13 exit criteria (VA-1.1–VA-1.13) | S314 test results |
| G-5 | RF-1 passes all 10 exit criteria (RF-1.1–RF-1.10) | S314 test results |
| G-6 | All existing tests pass (zero regressions) | CI green |
| G-7 | Paper pipeline unaffected | Smoke test: paper execution path produces same results |
| G-8 | No scope inflation: exactly 5 items delivered | Scope audit: no unchartered code changes |
| G-9 | Residual log published (if any gaps discovered) | S315 report |
| G-10 | TQ1 answered: adapter is hardened per S308/S310 specs | Aggregate of G-1 through G-8 |

**Gate verdict options:**

| Verdict | Meaning | Action |
|---------|---------|--------|
| PASS | All 10 criteria met | Proceed to implementation wave |
| PASS WITH RESIDUALS | G-1 through G-8 met; residuals logged | Proceed; residuals enter implementation wave backlog |
| FAIL | One or more of G-1 through G-8 not met | Remediate within tranche; do not proceed |

---

## 4. Non-Goals

### 4.1 Venue Integration Non-Goals

| # | Non-Goal | Reason |
|---|----------|--------|
| NG-1 | Real venue API call (testnet or mainnet) | Tranche is isolation-only; E2E belongs to implementation wave (I1) |
| NG-2 | Venue fill parsing with real data | No real fills produced in tranche; fill model validated in I1 |
| NG-3 | WebSocket or async fill feed | S306 NG-5; synchronous market orders only |
| NG-4 | Multi-venue adapter abstraction | S306 NG-3; single venue not yet proven |
| NG-5 | Venue-side reconciliation query | Manual reconciliation adequate for testnet (S310 §8.2) |

### 4.2 Infrastructure Non-Goals

| # | Non-Goal | Reason |
|---|----------|--------|
| NG-6 | Retry infrastructure (RT-1–RT-7) | Blocked until EC-1 proven; retry architecture is post-tranche |
| NG-7 | Circuit breaker pattern | S310 §3.3; adds complexity beyond testnet need |
| NG-8 | Rate limiting (self-imposed) | S310 §3.3; Binance testnet has generous limits |
| NG-9 | Dashboard or monitoring infrastructure | Structured logging sufficient; not a tranche concern |
| NG-10 | Alerting or PagerDuty integration | Testnet-only system; no operational alerting needed |

### 4.3 Domain Non-Goals

| # | Non-Goal | Reason |
|---|----------|--------|
| NG-11 | OMS or order management system | S309 proved no OMS module needed |
| NG-12 | Portfolio risk or position tracking | Not in scope until venue proven |
| NG-13 | P&L calculation | No P&L model exists or is needed |
| NG-14 | Balance or margin management | Testnet tolerance; venue rejects insufficient balance |
| NG-15 | Per-symbol kill switch | Global gate sufficient for testnet (S310 §3.3) |

### 4.4 Architecture Non-Goals

| # | Non-Goal | Reason |
|---|----------|--------|
| NG-16 | VenuePort interface redesign | Interface is correct; adapter implementation is the gap |
| NG-17 | New binaries or services | S310 constraint CN-3 |
| NG-18 | New NATS subjects or KV buckets | S310 constraint CN-2 |
| NG-19 | ClickHouse schema changes | S306 NG-9 / S310 constraint CN-1 |
| NG-20 | Changes to derive or store binaries | S310 constraint CN-4 |
| NG-21 | New HTTP endpoints | Existing composite read model sufficient |

### 4.5 Process Non-Goals

| # | Non-Goal | Reason |
|---|----------|--------|
| NG-22 | New design documents | All specs exist from S308–S310; tranche is implementation-only |
| NG-23 | Multi-symbol testing | Requires E2E working first; belongs to I3 |
| NG-24 | Failure injection testing | Requires E2E working first; belongs to I2 |
| NG-25 | Production readiness assessment | System is testnet-only |

---

## 5. Test Strategy

### 5.1 Test Scope

All tranche items are verified through **unit tests** and **httptest-based tests**. No integration tests against real venue. No compose-level tests.

| Tool | Purpose | Items |
|------|---------|-------|
| Standard Go unit tests | Deterministic derivation, format validation, classification correctness | EC-1, VA-1, RF-1 |
| `net/http/httptest` | Simulated HTTP server responses, timeouts, oversized bodies | EC-1, EC-2, EC-3, VA-1 |
| Code review | No bare errors escape, no credentials in errors, no randomness in derivation | EC-1, VA-1, RF-1 |

### 5.2 What Is NOT Tested in the Tranche

| Test Type | Why Not |
|-----------|--------|
| E2E against Binance testnet | Tranche is isolation-only |
| Compose-level smoke tests | Not needed for adapter-level changes |
| Load/stress tests | Testnet-only; not proportional |
| Failure injection | Requires E2E working first |

---

## 6. Residual Handling

If implementation of any tranche item reveals gaps not covered by the existing S308–S310 specifications:

1. The gap is logged in the stage report as a **residual**
2. The residual is categorized: blocker for implementation wave or deferrable
3. Blocker residuals enter the implementation wave as pre-conditions for I1
4. Deferrable residuals enter the implementation wave backlog
5. **No residual enters the tranche scope** — the tranche remains 5 items

---

*Delivered: 2026-03-21 — Stage S312, Phase 30*
