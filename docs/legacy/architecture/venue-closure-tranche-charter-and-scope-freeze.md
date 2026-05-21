# Venue Closure Tranche — Charter and Scope Freeze

**Stage:** S321 — Venue Closure Tranche Charter
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave (closure)
**Predecessor:** S320 — Venue Failure Path Verification and Containment
**Successor:** S322 — Body-Read-Failure Reconciliation

---

## 1. Strategic Context

S320 verified 19 failure paths across the venue execution chain with zero regressions.
The failure path is correctly classified, retry behavior is bounded, and containment is clean.
However, S320 identified six residual gaps (R-S320-1 through R-S320-6) that remain open before an evidence gate can credibly declare the venue path production-grade for testnet.

The most important gap is **R-S320-1**: no reconciliation for body-read-failure-after-200, classified as medium risk. The remaining five are low-risk hardening items with high value-to-cost ratio.

This document opens a **closure tranche** — a short, scope-frozen sequence of stages to close these gaps before the evidence gate.

### 1.1 Why a Closure Tranche, Not a New Wave

| Property | New Wave | Closure Tranche |
|----------|----------|-----------------|
| Scope | Open-ended, exploratory | Fixed residual list from predecessor gate |
| Origin | New charter questions | Known gaps from S320 §7 |
| Duration | 5–10 stages | 4–5 stages (S322–S326) |
| Design work | May produce new design | Zero new design — all gaps have known solutions |
| Inflation risk | Medium–high | Low — items derive from concrete, verified gaps |
| Gate structure | Multi-gate | Single exit gate (S326) |

The S312 Adapter Hardening Tranche established the tranche pattern for venue work. This closure tranche follows the same discipline: fixed item list, frozen scope, single exit gate.

### 1.2 Relationship to Prior Tranches

```
S306–S310 (Charter Wave)
    └──▶ S311 (Gate: PARTIALLY READY)
           └──▶ S312–S315 (Adapter Hardening Tranche)
                  └──▶ S316–S317 (E2E Venue Integration + Persistence)
                         └──▶ S318–S319 (Live Stack Smoke + Retry Infrastructure)
                                └──▶ S320 (Failure Path Verification — 19/19 PASS)
                                       └──▶ S321 (This: Closure Tranche Charter)
                                              └──▶ S322–S325 (Closure Items)
                                                     └──▶ S326 (Closure Gate)
                                                            └──▶ Evidence Gate (final)
```

---

## 2. Tranche Definition

### 2.1 Name and Identity

| Property | Value |
|----------|-------|
| Name | Venue Closure Tranche |
| Origin | S320 §7 residual gaps |
| Item source | R-S320-1 through R-S320-6, filtered and prioritized |
| Item count | 5 (frozen) |
| Stage budget | 4–5 stages (S322–S326) |
| Design dependency | None — all solutions identified in S320 findings |
| Exit gate | S326 — Closure Tranche Gate |

### 2.2 Governing Question

**CQ1: Are the residual venue-path gaps from S320 closed to the level required for a credible testnet evidence gate, without inflating into a broader venue platform redesign?**

The tranche answers this single question. It does not reopen adapter design, retry architecture, or execution semantics — those are settled by S312–S320.

### 2.3 Tranche Items (Frozen)

| # | Item ID | Name | Source | Priority |
|---|---------|------|--------|----------|
| 1 | **CT-1** | Body-read-failure reconciliation | R-S320-1 | **Medium (highest in tranche)** |
| 2 | **CT-2** | Global deadline in RetryPolicy | R-S320-2 | Low |
| 3 | **CT-3** | Kill switch check during retry backoff | R-S320-3 | Low |
| 4 | **CT-4** | Structured retry metrics and logging | R-S320-5 | Low |
| 5 | **CT-5** | Venue error code classification | R-S320-4 | Low |

**Excluded from tranche**: R-S320-6 (per-error-class differentiated retry policies). Rationale: the current uniform backoff is sufficient for single-venue testnet. Differentiated policies (e.g., Retry-After for 429) add complexity without proportional safety benefit at this stage. Logged as a post-evidence-gate item.

**No additional items may be added.** Discovered gaps during implementation are logged as residuals for post-evidence-gate work.

---

## 3. Tranche Structure

### 3.1 Stage Allocation

| Stage | Name | Scope | Items |
|-------|------|-------|-------|
| **S322** | Body-Read-Failure Reconciliation | Order status polling by client order ID when body read fails after 200 | CT-1 |
| **S323** | Retry Policy Hardening | Global deadline in RetryPolicy + kill switch check during backoff | CT-2, CT-3 |
| **S324** | Retry Observability | Structured metrics and logging from retry submitter | CT-4 |
| **S325** | Venue Error Code Classification | Map Binance error codes to failure classes for higher-fidelity routing | CT-5 |
| **S326** | Closure Tranche Gate | Verify all 5 items; zero regressions; exit decision | Gate only |

### 3.2 Dependency Chain

```
CT-1 (reconciliation)
  │
  └── independent of CT-2..CT-5

CT-2 (global deadline) ──┐
                         ├── CT-3 depends on CT-2 (kill switch check lives in retry loop)
CT-3 (kill switch)  ─────┘

CT-4 (metrics) ── independent

CT-5 (venue codes) ── independent

All ──▶ S326 (Gate)
```

**Critical path:** CT-1 (highest risk) should execute first. CT-2 and CT-3 are co-located in S323. CT-4 and CT-5 are independent and can run in any order.

### 3.3 Entry Conditions

| Condition | Source | Status |
|-----------|--------|--------|
| S320 delivered with 19/19 tests passing | S320 §5 | SATISFIED |
| Residual gaps catalogued with risk and mitigation | S320 §7 | SATISFIED |
| Zero regressions in execution test suite (80+ tests) | S320 §1 | SATISFIED |
| Retry infrastructure proven and documented | S319 | SATISFIED |
| This charter freezes scope | This document | SATISFIED |

---

## 4. Item Specifications (Summary)

### 4.1 CT-1: Body-Read-Failure Reconciliation

| Property | Value |
|----------|-------|
| Source | R-S320-1, FP-11 finding |
| What | When body read fails after HTTP 200, query venue order status by client order ID to recover fill details |
| Integration point | RetrySubmitter or adapter post-failure reconciliation path |
| Why medium risk | Order accepted but fill unknown; no way to resume persistence pipeline |
| Solution sketch | `QueryOrderStatus(ctx, clientOrderID)` on VenuePort; called when body-read-failure detected; populates receipt from status response |
| Acceptance | Test: body read fails after 200 → reconciliation queries status → fill recovered → receipt returned |

### 4.2 CT-2: Global Deadline in RetryPolicy

| Property | Value |
|----------|-------|
| Source | R-S320-2 |
| What | Add optional `GlobalDeadline time.Duration` to RetryPolicy; if set, creates a scoped context wrapping the entire retry loop |
| Why | Without it, worst-case wall clock = MaxAttempts × per-request timeout (30.3s). Global deadline caps total time |
| Acceptance | Test: global deadline shorter than total retry budget → loop aborts after deadline with retry metadata |

### 4.3 CT-3: Kill Switch Check During Retry Backoff

| Property | Value |
|----------|-------|
| Source | R-S320-3 |
| What | Check `IsHalted()` before each retry attempt (not just before the first) |
| Why | Kill switch activated mid-retry should abort, not wait for remaining attempts |
| Integration point | RetrySubmitter loop, before each `inner.SubmitOrder` call |
| Acceptance | Test: kill switch activates after first attempt → retry loop aborts immediately |

### 4.4 CT-4: Structured Retry Metrics and Logging

| Property | Value |
|----------|-------|
| Source | R-S320-5 |
| What | Emit structured log entries from RetrySubmitter: attempt start, backoff, exhaustion, recovery |
| Why | Retry behavior invisible without logging; production troubleshooting requires visibility |
| Acceptance | Test: retries produce structured log entries with attempt number, delay, outcome |

### 4.5 CT-5: Venue Error Code Classification

| Property | Value |
|----------|-------|
| Source | R-S320-4 |
| What | Use Binance-specific error codes (e.g., -1015, -2015) in classification decisions alongside HTTP status |
| Why | HTTP status alone may misclassify edge cases; venue codes provide higher fidelity |
| Acceptance | Test: known Binance error codes → correct failure class; unknown codes → fallback to HTTP status classification |

---

## 5. Scope Freeze Rules

### 5.1 What Is In Scope

Only the 5 items listed in §2.3. Nothing else.

### 5.2 What Is Explicitly Out of Scope

| Item | Why Out | Belongs To |
|------|---------|-----------|
| OMS or order management system | S309 proved no OMS needed for testnet | Non-goal |
| Dashboard or monitoring infrastructure | Structured logging sufficient; dashboards are operational maturity | Post-evidence-gate |
| Mainnet venue calls | System is testnet-only | Non-goal |
| Multi-venue routing or abstraction | Single venue not yet fully proven | Non-goal |
| Portfolio risk or position tracking | Not in scope until venue proven in production | Non-goal |
| Per-error-class differentiated retry policies (R-S320-6) | Uniform backoff sufficient for testnet | Post-evidence-gate |
| Circuit breaker pattern | Not needed for single-venue testnet with bounded retries | Non-goal |
| Async/queue-based retry | Synchronous retry proven sufficient | Non-goal |
| WebSocket or async fill feed | S306 non-goal (NG-5) | Non-goal |
| New binaries or services | S310 constraint CN-3 | Non-goal |
| New NATS subjects or KV buckets | S310 constraint CN-2 | Non-goal |
| ClickHouse schema changes | S306 non-goal (NG-9) | Non-goal |
| VenuePort interface redesign | Interface is correct; only add optional QueryOrderStatus | Minimal |
| Retry architecture redesign | RetrySubmitter pattern is proven; only enhance | Minimal |

### 5.3 Inflation Prevention Rules

| Rule | Statement |
|------|-----------|
| IR-1 | No new items may enter the tranche after this charter is delivered |
| IR-2 | Discovered gaps are logged as residuals for post-evidence-gate work |
| IR-3 | No stage may introduce code changes unrelated to CT-1 through CT-5 |
| IR-4 | No new design documents are produced — solutions are known from S320 findings |
| IR-5 | The closure gate (S326) must verify exactly 5 items — not 4, not 6 |
| IR-6 | If an item reveals scope larger than expected, the gate logs it as a residual |

---

## 6. Constraints Inherited

| ID | Constraint | Source |
|----|-----------|--------|
| CN-1 | No schema changes to ClickHouse | S310 |
| CN-2 | No new NATS subjects or KV buckets | S310 |
| CN-3 | No new binaries or services | S310 |
| CN-5 | Paper pipeline must remain zero-regression | S310 |
| CN-7 | VenuePort interface changes minimal (only add QueryOrderStatus if needed for CT-1) | This charter |
| CN-8 | RetrySubmitter remains a decorator; no architectural change | This charter |

---

## 7. Risk Assessment

### 7.1 Tranche Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|-----------|-----------|
| CT-1 reconciliation requires VenuePort interface change | Low | Medium | Minimal addition (one method); does not break existing callers |
| CT-3 kill switch injection into retry loop couples concerns | Low | Low | Pass `IsHalted` as a function, not a concrete type |
| CT-5 Binance error codes incomplete or undocumented | Low | Medium | Fallback to HTTP-status-only classification for unknown codes |
| Tranche inflates beyond 5 items | Medium | Low | IR-1 through IR-6 prevent inflation; gate enforces 5-item count |
| Paper pipeline regression | High | Low | CN-5 enforced; existing 80+ tests must pass at every stage |

### 7.2 Accepted Risks

| Risk | Acceptance Rationale |
|------|---------------------|
| CT-1 not proven against real venue body-read failure | Simulated with httptest; real occurrence is rare on testnet |
| CT-5 code map may miss rare Binance error codes | Unknown codes fall back to HTTP-status classification; no regression |
| CT-4 logging format not standardized across services | Only one service (gateway) uses venue calls; standardization deferred |

---

## 8. Governance

### 8.1 Governing Questions

| ID | Question | Answered By |
|----|----------|------------|
| CQ1 | Are the S320 residual gaps closed for a credible evidence gate? | S326 gate |
| CQ2 | Does body-read-failure reconciliation recover fill details? | S322 tests |
| CQ3 | Is the retry loop bounded by both per-attempt and global deadlines? | S323 tests |
| CQ4 | Is retry behavior observable through structured logging? | S324 tests |
| CQ5 | Does venue-code-aware classification improve fidelity without regression? | S325 tests |
| CQ6 | Zero regressions in the existing 80+ execution test suite? | S326 gate |

### 8.2 Stage Governance

Each stage in the tranche (S322–S325) must:

1. Reference this charter as its scope authority
2. Implement only the items assigned to it
3. Produce unit tests with test evidence in the stage report
4. Pass all existing tests without regression
5. Log any discovered gaps as residuals, not as tranche work

---

*Delivered: 2026-03-21 — Stage S321, Phase 30*
