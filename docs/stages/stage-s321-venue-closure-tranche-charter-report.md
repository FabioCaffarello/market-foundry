# Stage S321 — Venue Closure Tranche Charter Report

> **Status:** Complete
> **Predecessor:** S320 (Venue Failure Path Verification and Containment)
> **Scope:** Open closure tranche for S320 residual gaps, freeze scope, define S322–S326

## 1. Executive Summary

S320 verified 19 failure paths with zero regressions and identified 6 residual gaps (R-S320-1 through R-S320-6). S321 transforms these residuals into a formal closure tranche — a short, scope-frozen sequence of 4 implementation stages (S322–S325) plus one exit gate (S326).

The tranche prioritizes R-S320-1 (body-read-failure reconciliation, medium risk) as the highest-priority item. Five of six gaps are included; R-S320-6 (per-error-class differentiated retry policies) is excluded as insufficient value for testnet scope.

**Key result**: The venue closure tranche is formally open with scope frozen, priorities ordered, exit criteria defined, and non-goals explicit. The tranche prepares S322–S326 without inflation.

## 2. Charter Summary

| Property | Value |
|----------|-------|
| Tranche name | Venue Closure Tranche |
| Origin | S320 §7 residual gaps |
| Items | 5 (CT-1 through CT-5), frozen |
| Stages | S322–S326 |
| Gate | S326 — single exit gate |
| Governing question | CQ1: Are S320 residual gaps closed for a credible testnet evidence gate? |

## 3. Tranche Items

| # | Item ID | Name | Source | Priority | Stage |
|---|---------|------|--------|----------|-------|
| 1 | CT-1 | Body-read-failure reconciliation | R-S320-1 | Medium (highest) | S322 |
| 2 | CT-2 | Global deadline in RetryPolicy | R-S320-2 | Low | S323 |
| 3 | CT-3 | Kill switch check during retry backoff | R-S320-3 | Low | S323 |
| 4 | CT-4 | Structured retry metrics and logging | R-S320-5 | Low | S324 |
| 5 | CT-5 | Venue error code classification | R-S320-4 | Low | S325 |

**Excluded**: R-S320-6 (per-error-class differentiated retry policies) — deferred to post-evidence-gate.

## 4. Stage Plan

| Stage | Name | Items | Key Deliverable |
|-------|------|-------|-----------------|
| S322 | Body-Read-Failure Reconciliation | CT-1 | QueryOrderStatus + reconciliation path |
| S323 | Retry Policy Hardening | CT-2, CT-3 | GlobalDeadline + HaltCheck in retry loop |
| S324 | Retry Observability | CT-4 | Structured slog entries from retry submitter |
| S325 | Venue Error Code Classification | CT-5 | Binance error code map in adapter |
| S326 | Closure Tranche Gate | — | 5/5 verification, zero regressions, exit decision |

## 5. Exit Criteria (S326 Gate)

| Criterion | Threshold |
|-----------|-----------|
| Items implemented | 5/5 |
| Regression in execution tests (80+) | 0 |
| Regression in failure path tests (19) | 0 |
| Scope inflation | None (exactly 5 items) |
| Paper pipeline | Green |

Gate verdicts: PASS → evidence gate; PASS WITH RESIDUALS → evidence gate with documented residuals; FAIL → diagnose.

## 6. Non-Goals

- OMS, order management, mainnet, multi-venue, portfolio risk
- Dashboard, alerting, Prometheus infrastructure
- Retry architecture redesign, circuit breaker, async retry
- Per-error-class differentiated retry policies
- New binaries, NATS subjects, ClickHouse schema changes
- WebSocket or async fill feed

## 7. Files Delivered

| File | Action | Description |
|------|--------|-------------|
| `docs/architecture/venue-closure-tranche-charter-and-scope-freeze.md` | New | Charter and scope freeze for closure tranche |
| `docs/architecture/closure-tranche-items-priorities-exit-criteria-and-non-goals.md` | New | Detailed items, priorities, exit criteria, non-goals |
| `docs/stages/stage-s321-venue-closure-tranche-charter-report.md` | New | This report |

## 8. Invariants Preserved

| Invariant | Status |
|-----------|--------|
| No code changes in charter stage | Preserved (documentation only) |
| No new design artifacts | Preserved (solutions known from S320) |
| Scope frozen at delivery | 5 items, IR-1 through IR-6 enforced |
| Paper pipeline green | No code touched |

## 9. Preparation for S322

S322 should:
1. Read FP-11 test to understand body-read-failure scenario
2. Consult Binance `GET /fapi/v1/order` endpoint for `origClientOrderId` query
3. Decide interface placement: extend VenuePort vs separate VenueReconciler port
4. Implement reconciliation with httptest mock
5. Verify zero regression in existing 80+ execution tests

**Recommendation**: Add `QueryOrderStatus` to `VenuePort` — same HTTP client, credentials, and signing logic; interface segregation would add complexity without benefit at single-venue scope.

---

*Delivered: 2026-03-21 — Stage S321, Phase 30*
