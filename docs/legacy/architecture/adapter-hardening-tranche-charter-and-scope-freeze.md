# Adapter Hardening Tranche — Charter and Scope Freeze

**Stage:** S312 — Adapter Hardening Tranche Charter
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave (continued)
**Predecessor:** S311 — Post-Charter Gate and Strategic Direction
**Successor:** S313 — Adapter Contract Hardening (EC-1, EC-2, EC-3)

---

## 1. Strategic Context

S311 closed the Venue Readiness Charter Wave (S306–S310) with verdict **PARTIALLY READY** and recommended **Option B: Short Foundational Tranche** before the implementation wave. The design foundation is sound and complete — 18 of 36 capabilities exist, state machine validated, 13 of 14 guard rails in code, 23 failure modes classified. One critical gap (EC-1: client order ID) and four supporting adapter items (EC-2, EC-3, VA-1, RF-1) must be resolved before E2E venue integration.

This document opens the foundational tranche formally, freezes its scope, and establishes the governance boundary that prevents inflation into a broader implementation wave.

### 1.1 Why a Tranche, Not a Wave

| Property | Wave | Tranche |
|----------|------|---------|
| Scope | Open-ended capability set | Fixed item list, frozen at charter |
| Duration | 5–10 stages typical | 2–3 stages maximum |
| Gate structure | Multi-gate with intermediate assessments | Single exit gate |
| Design work | Produces new design artifacts | Implements against existing specs only |
| Inflation risk | Medium–high | Low — fixed scope prevents expansion |

This tranche implements **zero new design**. Every item has a complete specification from S308–S310. The tranche is pure code delivery against frozen contracts.

---

## 2. Tranche Definition

### 2.1 Name and Identity

| Property | Value |
|----------|-------|
| Name | Adapter Hardening Tranche |
| Origin | S311 §6.1 recommended direction |
| Item source | S307 gap map, S308 contracts, S310 guard rails |
| Item count | 5 (frozen) |
| Stage budget | 2–3 stages (S313–S315) |
| Design dependency | None — all specs exist |
| Exit gate | Tranche gate at final stage |

### 2.2 Governing Question

**TQ1: Is the VenuePort adapter hardened to the level required by S308 contracts and S310 guard rails, such that E2E venue integration can proceed without adapter-level surprises?**

The tranche answers this single question. It does not answer VQ1–VQ7 — those belong to the implementation wave that follows.

### 2.3 Tranche Items (Frozen)

| # | Item ID | Name | Source Spec | Priority |
|---|---------|------|-------------|----------|
| 1 | **EC-1** | Client order ID derivation | S308 IDEM-3, S307 gap | **Critical** |
| 2 | **EC-2** | Response body size cap | S310 PGR-14, S307 gap | Low |
| 3 | **EC-3** | Per-request context deadline | S310 PGR-03, S307 gap | Medium |
| 4 | **VA-1** | Error classification completeness | S308 C-FAIL, S307 gap | High |
| 5 | **RF-1** | Retryable flag completeness | S310 failure modes, S307 gap | High |

**No additional items may be added.** If new gaps are discovered during implementation, they are logged as residuals for the implementation wave — they do not enter the tranche.

---

## 3. Tranche Structure

### 3.1 Stage Allocation

| Stage | Name | Scope | Items |
|-------|------|-------|-------|
| **S313** | Adapter Contract Hardening | Implement EC-1, EC-2, EC-3 against VenuePort adapter | EC-1, EC-2, EC-3 |
| **S314** | Error Classification Hardening | Complete VA-1 and RF-1 against C-FAIL taxonomy | VA-1, RF-1 |
| **S315** | Tranche Gate | Verify all 5 items in isolation; zero regressions; exit decision | Gate only |

### 3.2 Dependency Chain

```
EC-1 (client order ID)
  │
  ├── EC-2 (body cap) ─── independent, parallel OK
  │
  └── EC-3 (deadline) ─── independent, parallel OK
         │
         └──▶ VA-1 (error classification) ─── depends on adapter error paths being stable
                │
                └──▶ RF-1 (retryable flags) ─── depends on VA-1 classes being complete
                       │
                       └──▶ Tranche Gate (S315)
```

**Critical path:** EC-1 → VA-1 → RF-1 → Gate.

EC-2 and EC-3 are independent of EC-1 and can be implemented in parallel. VA-1 depends on the adapter error paths being stable (which EC-1/EC-2/EC-3 affect). RF-1 depends on VA-1 being complete (retryable flag presupposes correct classification).

### 3.3 Entry Conditions for Tranche

| Condition | Source | Status |
|-----------|--------|--------|
| S311 delivered with Option B recommendation | S311 §6.1 | SATISFIED |
| S312 charter freezes scope | This document | SATISFIED |
| All 5 items have complete design specs | S308, S310 | SATISFIED |
| No design work required | S311 §7.2 | CONFIRMED |
| Paper pipeline green (zero regressions) | CI baseline | VERIFIED at S311 |

---

## 4. Item Specifications (Summary)

Each item below references the authoritative spec. The tranche implements against the spec — it does not modify the spec.

### 4.1 EC-1: Client Order ID

| Property | Value |
|----------|-------|
| Spec | S308 §3.1 IDEM-3 |
| What | Deterministic derivation of `newClientOrderId` from `ExecutionIntent` fields |
| Derivation rule | Hash of `DeduplicationKey()` = `exec:{type}:{source}:{symbol}:{timeframe}:{unix}` |
| Integration point | `VenueOrderRequest` sent to Binance API |
| Why critical | Without EC-1, timeout ambiguity cannot be resolved; retry is permanently blocked (RT-1) |
| Acceptance | Same intent → same ID (deterministic); different intent → different ID (collision-free within practical bounds) |

### 4.2 EC-2: Response Body Size Cap

| Property | Value |
|----------|-------|
| Spec | S310 §3.1 PGR-14 |
| What | `io.LimitReader(body, 64*1024)` on all venue HTTP responses |
| Integration point | Adapter HTTP response reading |
| Why | Prevents unbounded memory allocation from malformed or adversarial responses |
| Acceptance | Oversized response → truncated and classified as parse error (C-FAIL parse class) |

### 4.3 EC-3: Per-Request Context Deadline

| Property | Value |
|----------|-------|
| Spec | S310 §3.1 PGR-03, §7.1 |
| What | `context.WithTimeout(ctx, d)` wrapping every `VenuePort.SubmitOrder` call |
| Default timeout | 10 seconds (configurable) |
| Integration point | Actor layer, pre-`SubmitOrder` |
| Why | Without deadline, hung venue calls block the pipeline indefinitely |
| Acceptance | Slow httptest server → context deadline exceeded → `*problem.Problem` with Unavailable category |

### 4.4 VA-1: Error Classification Completeness

| Property | Value |
|----------|-------|
| Spec | S308 §2.5 C-FAIL |
| What | All 8 failure classes return `*problem.Problem` with correct category |
| Classes | Authentication, Client error, Rate limit, Venue unavailable, Server error, Network failure, Parse failure, Unknown |
| Integration point | Adapter error handling paths |
| Why | Incomplete classification means some errors escape as bare Go errors, violating F-1 invariant |
| Acceptance | Every HTTP status code and connection error condition → correct problem category |

### 4.5 RF-1: Retryable Flag Completeness

| Property | Value |
|----------|-------|
| Spec | S310 §6.2, S308 F-2 |
| What | Every `*problem.Problem` carries correct `Retryable` field |
| Retryable | 429 (rate limit), 503 (unavailable), 5xx (server error), network failure |
| Non-retryable | 400/422 (client error), 401/403 (auth), parse failure, unknown |
| Integration point | Adapter error construction |
| Why | Future retry infrastructure (RT-1–RT-7) depends on correct classification; incorrect flags = retry loops on permanent errors |
| Acceptance | Unit test matrix: each failure class → correct retryable flag |

---

## 5. Scope Freeze Rules

### 5.1 What Is In Scope

Only the 5 items listed in §2.3. Nothing else.

### 5.2 What Is Explicitly Out of Scope

| Item | Why Out | Belongs To |
|------|---------|-----------|
| E2E venue call (real testnet) | Tranche hardens adapter in isolation; E2E is implementation wave | I1 |
| Retry infrastructure (RT-1–RT-7) | Blocked until EC-1 proven; requires circuit breaker, backoff, jitter | Post-tranche |
| Multi-symbol venue concurrency | Requires E2E working first | I3 |
| Failure injection tests | Requires E2E working first | I2 |
| ClickHouse schema changes | S306 non-goal (NG-9) | Non-goal |
| New HTTP endpoints | S306 non-goal (NG-10) | Non-goal |
| New NATS subjects or KV buckets | S310 constraint CN-2 | Non-goal |
| New binaries or services | S310 constraint CN-3 | Non-goal |
| Kill switch changes | Existing mechanism sufficient (S310 §4) | Non-goal |
| Fill model code changes | C-FILL contracts already match existing adapter code | Non-goal |
| VenuePort interface redesign | Interface is correct; implementation is the gap | Non-goal |
| OMS or order management | S309 proved no OMS needed | Non-goal |
| Dashboard or observability infrastructure | Structured logging sufficient for testnet | Non-goal |
| WebSocket or async fill feed | S306 non-goal (NG-5) | Non-goal |
| Multi-venue routing or abstraction | Single venue not yet proven | Non-goal |
| Portfolio risk or position tracking | Not in scope until venue proven | Non-goal |
| Production hardening | System is testnet-only | Non-goal |

### 5.3 Inflation Prevention Rules

| Rule | Statement |
|------|-----------|
| IR-1 | No new items may enter the tranche after this charter is delivered |
| IR-2 | Discovered gaps are logged as residuals, not absorbed into the tranche |
| IR-3 | No stage in the tranche may introduce code changes unrelated to EC-1/EC-2/EC-3/VA-1/RF-1 |
| IR-4 | No design documents are produced by the tranche — all specs exist from S308–S310 |
| IR-5 | The tranche gate (S315) must verify exactly 5 items — not 4, not 6 |
| IR-6 | If a tranche item reveals scope larger than expected, the tranche gate logs it as a residual and the implementation wave absorbs it |

---

## 6. Constraints Inherited from Charter Wave

| ID | Constraint | Source |
|----|-----------|--------|
| CN-1 | No schema changes to ClickHouse | S310 |
| CN-2 | No new NATS subjects or KV buckets | S310 |
| CN-3 | No new binaries or services | S310 |
| CN-4 | No changes to derive or store binaries | S310 |
| CN-5 | Paper pipeline must remain zero-regression | S310 |
| CN-6 | No design documents produced | This charter |
| CN-7 | VenuePort interface unchanged | This charter |

---

## 7. Relationship to Implementation Wave

### 7.1 Tranche → Implementation Wave Boundary

The tranche **does not** begin E2E integration. The implementation wave **does not** begin until the tranche gate passes.

```
S306–S310 (Charter Wave)
    │
    └──▶ S311 (Gate: PARTIALLY READY)
           │
           └──▶ S312 (This: Tranche Charter)
                  │
                  └──▶ S313–S315 (Tranche: Adapter Hardening)
                         │
                         └──▶ Tranche Gate (S315 exit)
                                │
                                └──▶ Implementation Wave (I1–I4)
                                       │
                                       └──▶ Evidence Gate (VQ1–VQ7)
```

### 7.2 Implementation Wave Entry Conditions

The implementation wave may begin only when ALL of the following are true:

| Condition | Verified By |
|-----------|------------|
| EC-1 implemented and unit-tested | S315 gate |
| EC-2 implemented and unit-tested | S315 gate |
| EC-3 implemented and unit-tested | S315 gate |
| VA-1 complete and unit-tested | S315 gate |
| RF-1 consistent and unit-tested | S315 gate |
| Zero regressions against paper pipeline | S315 gate |
| No scope inflation in tranche | S315 gate |

---

## 8. Risk Assessment

### 8.1 Tranche Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|-----------|-----------|
| EC-1 derivation collisions | Medium | Low | Hash-based derivation from unique dedup key; unit test for collision resistance |
| Binance `newClientOrderId` format constraints | Medium | Medium | Consult API docs in S313; truncate/encode if needed |
| VA-1 reveals unmapped error codes | Low | Medium | Map to Unknown class (C-FAIL catch-all); log as residual |
| Tranche expands into mini-wave | Medium | Low | IR-1 through IR-6 prevent inflation; gate enforces 5-item count |
| Paper pipeline regression | High | Low | CN-5 enforced; existing tests must pass at every stage |

### 8.2 Accepted Risks

| Risk | Acceptance Rationale |
|------|---------------------|
| EC-1 not proven against real venue | Tranche proves in isolation; E2E proof is implementation wave (I1) |
| VA-1 may miss edge-case error codes | Unknown class exists as catch-all; real venue testing in I1 will surface gaps |
| RF-1 retryable flags not exercised by retry logic | No retry infrastructure in tranche; flags are data, not behavior, until RT-1–RT-7 |

---

## 9. Governance

### 9.1 Governing Questions for the Tranche

| ID | Question | Answered By |
|----|----------|------------|
| TQ1 | Is the adapter hardened per S308/S310 specs? | S315 gate |
| TQ2 | Can EC-1 be derived deterministically without collision? | S313 unit tests |
| TQ3 | Are all 8 C-FAIL classes implemented with correct problem categories? | S314 unit tests |
| TQ4 | Does the paper pipeline remain zero-regression? | S315 regression check |
| TQ5 | Did the tranche stay within its 5-item scope? | S315 scope audit |

### 9.2 Stage Governance

Each stage in the tranche (S313–S315) must:

1. Reference this charter as its scope authority
2. Implement only the items assigned to it
3. Produce unit tests (not integration tests against real venue)
4. Pass existing tests without regression
5. Log any discovered gaps as residuals, not as tranche work

---

*Delivered: 2026-03-21 — Stage S312, Phase 30*
