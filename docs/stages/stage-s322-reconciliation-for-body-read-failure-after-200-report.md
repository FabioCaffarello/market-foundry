# Stage S322 — Reconciliation for Body-Read-Failure-After-200 Report

> **Status:** Complete
> **Predecessor:** S320 (Venue Failure Path Verification), S319 (Retry Infrastructure)
> **Scope:** Surgical reconciliation for the R-S320-1 gap: body-read-failure-after-200

## 1. Executive Summary

S320 identified R-S320-1 as the principal medium-risk residual gap: when the venue accepts an order (HTTP 200) but the response body is lost, the system has no mechanism to recover the order status or fills. S322 closes this gap with a minimal, focused reconciliation mechanism.

The solution adds:
- A `VenueQueryPort` interface for querying existing orders by client order ID.
- A `QueryOrder` method on `BinanceFuturesTestnetAdapter` (Binance `GET /fapi/v1/order`).
- A `Post200Reconciler` wrapper that detects body-read-failure-after-200 and automatically queries the venue to recover status/fills.
- Body-read-failure detection markers in the adapter's Problem output.
- 9 reconciliation tests covering recovery, failure, passthrough, composition, and invariant preservation.

**Key result**: R-S320-1 is closed. The system recovers order status and fills after body-read-failure-after-200 without re-submitting or risking duplicate execution. Zero regressions in the existing test suite.

## 2. Reconciliation Mechanism

### 2.1 Detection

The adapter now marks body-read-failure-after-200 with structured details:
```go
problem.Wrap(err, problem.Internal, "read venue response failed").
    WithDetail("body_read_failure_after_200", true).
    WithDetail("client_order_id", ClientOrderID(intent))
```

### 2.2 Recovery Path

```
SubmitOrder (POST /fapi/v1/order)
  → HTTP 200 headers received
  → Body read fails (timeout/reset)
  → Problem with body_read_failure_after_200 marker
  → Post200Reconciler detects marker
  → QueryOrder (GET /fapi/v1/order?origClientOrderId=...)
  → Returns VenueOrderReceipt with recovered status + fills
```

### 2.3 Composition

```
Post200Reconciler (implements VenuePort)
  ├── submit: RetrySubmitter → BinanceFuturesTestnetAdapter
  └── query: BinanceFuturesTestnetAdapter (VenueQueryPort)
```

## 3. Files Changed

| File | Action | Description |
|------|--------|-------------|
| `internal/application/ports/venue.go` | Modified | Added `VenueQueryPort` interface |
| `internal/application/execution/binance_futures_testnet_adapter.go` | Modified | Added `QueryOrder` method; body-read-failure marker with client order ID |
| `internal/application/execution/post200_reconciler.go` | New | Post200Reconciler: detects body-read-failure-after-200, queries venue for recovery |
| `internal/application/execution/post200_reconciler_test.go` | New | 9 reconciliation tests (RC-01 through RC-09) |
| `docs/architecture/reconciliation-for-body-read-failure-after-200.md` | New | Reconciliation mechanism design |
| `docs/architecture/post-200-recovery-semantics-invariants-and-limitations.md` | New | Semantic contract, invariants, limitations |
| `docs/stages/stage-s322-reconciliation-for-body-read-failure-after-200-report.md` | New | This report |

## 4. Test Evidence

### 4.1 Reconciliation Tests (RC-01 through RC-09)

| Test ID | Scenario | Outcome |
|---------|----------|---------|
| RC-01 | Body read failure → recovery via QueryOrder | PASS (3.00s) |
| RC-02 | Body read failure → query also fails → enriched error | PASS (3.00s) |
| RC-03 | Non-body-read error → passes through unchanged | PASS (0.00s) |
| RC-04 | Successful submit → no reconciliation triggered | PASS (0.00s) |
| RC-05 | No duplicate submit — exactly 1 POST, then 1 GET | PASS (2.00s) |
| RC-06 | Recovered receipt has correct client order ID | PASS (2.00s) |
| RC-07 | Recovered intent preserves original fields | PASS (2.00s) |
| RC-08 | Retryable error → passes through without reconciliation | PASS (0.00s) |
| RC-09 | Reconciler composes with RetrySubmitter (503 → body-read-failure → recovery) | PASS (2.00s) |

### 4.2 Regression Check

Full execution test suite: **all tests pass**, zero regressions. The existing FP-11 test (body-read-failure classification) continues to pass, confirming the adapter marker is backward-compatible.

## 5. Invariants Preserved

| Invariant | Source | Verification |
|-----------|--------|-------------|
| INV-REC-1: No duplicate execution | S322 | RC-05: exactly 1 POST |
| INV-REC-2: Client order ID stability | EC-1/S313 | RC-06: correct ID in recovery query |
| INV-REC-3: Intent preservation | S322 | RC-07: original fields preserved |
| INV-REC-4: Context independence | S322 | RC-01: recovery works after submit context expires |
| INV-REC-5: Error enrichment | S322 | RC-02: reconciliation metadata in details |
| INV-REC-6: Passthrough | S322 | RC-03, RC-08: non-target errors unchanged |
| EC-1: Deterministic client order ID | S313 | RC-06: same ID used for submit and query |
| EC-3: Per-request deadline | S308 | QueryOrder enforces deadline |
| F-4: Credential redaction | S314 | QueryOrder same credential handling |

## 6. R-S320-1 Closure Assessment

| Criterion | Before S322 | After S322 |
|-----------|------------|-----------|
| Detection of body-read-failure-after-200 | Classified as Internal (FP-11) | Classified + marked with details |
| Recovery of order status | None | QueryOrder by client order ID |
| Recovery of fills | None | QueryOrder returns fill data |
| Duplicate execution risk | N/A (no recovery attempted) | Zero (GET only, no re-submit) |
| Recovery failure handling | N/A | Enriched error with reconciliation metadata |

**Verdict**: R-S320-1 is **closed**. The body-read-failure-after-200 case now has a recovery mechanism that restores order status and fills without duplicate execution risk.

## 7. Residual Gaps

| ID | Gap | Risk Level | Note |
|----|-----|-----------|------|
| R-S322-1 | Single recovery attempt (no retry on query failure) | Low | Acceptable for testnet; query retry can be added if needed |
| R-S322-2 | No persistence of ambiguous state when recovery fails | Low | Would require OMS infrastructure; out of scope |
| R-S322-3 | Fill detail granularity differs between submit and query responses | Low | Both use same parseOrderResponse; aggregate data available |
| R-S322-4 | Theoretical race: order not yet queryable after 200 | Very Low | Binance processes market orders synchronously |

All residual gaps are low or very low risk and do not block the evidence gate.

## 8. Preparation for S323

With R-S320-1 closed, the venue execution layer has:
- Complete error classification (S314)
- Bounded retry with idempotency (S319)
- Verified failure paths (S320)
- Post-200 reconciliation (S322)

Recommended next directions:
1. **Evidence gate closure** — Aggregate all venue-readiness evidence and close the final gate.
2. **Actor-layer wiring** — Integrate Post200Reconciler + RetrySubmitter into the execution actor pipeline.
3. **Operational observability** — Structured logging/metrics for reconciliation events.
4. **Kill switch coordination** — Check kill switch between retry attempts (R-S320-3).
