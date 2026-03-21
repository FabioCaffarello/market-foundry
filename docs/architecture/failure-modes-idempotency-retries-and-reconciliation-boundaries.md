# Failure Modes, Idempotency, Retries, and Reconciliation Boundaries

**Stage:** S310 — Production Guard Rails and Failure Envelope
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Companion:** `production-guard-rails-and-failure-envelope.md`

---

## 1. Purpose

This document provides the **detailed failure mode matrix** and **operational response taxonomy** for every classified failure path in the venue execution pipeline. It extends the C-FAIL taxonomy from S308 with concrete system responses, operator actions, and boundary conditions.

---

## 2. Failure Mode Matrix

### 2.1 Pre-Submission Failures (Before Venue Call)

These failures are fully within system control and produce no venue-side effects.

| ID | Failure Mode | Detection | System Response | Intent State | Operator Action |
|----|-------------|-----------|-----------------|-------------|-----------------|
| F-PRE-01 | Kill switch halted | ControlGate query returns `halted` | Block submission; structured log | `submitted` (unchanged) | Resume via `configctl` when ready |
| F-PRE-02 | Intent stale | `time.Since(intent.Timestamp) > MaxStaleness` | Block submission; structured log | `submitted` (unchanged) | None — new evaluation cycle generates fresh intent |
| F-PRE-03 | Side filter rejection | Intent side not in allowed set | Block submission; structured log | `submitted` (unchanged) | None — by design |
| F-PRE-04 | Validation failure | Intent fields fail structural validation | Block submission; structured log | `submitted` (unchanged) | None — indicates upstream bug if persistent |
| F-PRE-05 | JetStream dedup | Duplicate dedup key within 24h window | Message silently deduplicated | N/A (no intent created) | None |
| F-PRE-06 | KV sequence conflict | Stale sequence on KV write attempt | Write rejected; intent discarded | N/A (write rejected) | None — monotonicity enforced |

**Classification:** All F-PRE failures are **acceptable** — they are guard rails working as designed.

### 2.2 Venue Submission Failures (During Venue Call)

These failures occur after the HTTP request is dispatched but before a successful response is processed.

| ID | Failure Mode | HTTP Signal | Retryable (C-FAIL) | System Response | Intent State | Operator Action |
|----|-------------|------------|---------------------|-----------------|-------------|-----------------|
| F-VEN-01 | Authentication failure | 401, 403 | No | Log `*problem.Problem`; do not retry | `submitted` | Check credentials; reconfigure adapter |
| F-VEN-02 | Client error | 400, 422 | No | Log `*problem.Problem`; do not retry | `submitted` | Investigate request construction; likely upstream bug |
| F-VEN-03 | Rate limit | 429 | Yes | Log `*problem.Problem`; **do not retry** (S310) | `submitted` | Monitor frequency; reduce pipeline rate if persistent |
| F-VEN-04 | Venue unavailable | 503, connection refused | Yes | Log `*problem.Problem`; **do not retry** (S310) | `submitted` | Check venue status; wait for recovery |
| F-VEN-05 | Server error | 500, 502, 504 | Yes | Log `*problem.Problem`; **do not retry** (S310) | `submitted` | Check venue status |
| F-VEN-06 | Network failure | DNS, TCP, TLS error | Yes | Log `*problem.Problem`; **do not retry** (S310) | `submitted` | Check network connectivity |
| F-VEN-07 | Context timeout | No response within deadline | Yes | Context cancelled; log timeout | `submitted` | **Audit for phantom order** — venue may have accepted |
| F-VEN-08 | Response parse failure | 200 but malformed JSON | No | Log `*problem.Problem` with raw body (truncated) | `submitted` | Investigate venue API change |
| F-VEN-09 | Unknown status in response | 200 but unmapped order status | No | Log `*problem.Problem`; reject | `submitted` | Update adapter status mapping |
| F-VEN-10 | Response body too large | Body exceeds 64KB limit | No | `LimitReader` truncates; parse fails | `submitted` | Investigate unexpected response |
| F-VEN-11 | Missing venue order ID | 200 but empty/missing order ID | No | Reject fill; log violation | `submitted` | Investigate venue API behavior |

**Classification:**
- F-VEN-01, F-VEN-02: **Unacceptable if persistent** — indicate configuration or code bugs
- F-VEN-03 through F-VEN-06: **Acceptable** — transient venue issues; self-recovering
- F-VEN-07: **Dangerous** — requires operator audit for phantom orders
- F-VEN-08 through F-VEN-11: **Unacceptable** — indicate venue API contract violations

### 2.3 Post-Submission Failures (After Successful Venue Response)

These failures occur after the venue has accepted/filled/rejected the order but before the result is fully materialized.

| ID | Failure Mode | Detection | System Response | Data State | Operator Action |
|----|-------------|-----------|-----------------|-----------|-----------------|
| F-POST-01 | Transition validation failure | `ValidTransition()` returns false | Reject state mutation; log invariant violation | Intent state unchanged | Investigate; indicates adapter or venue mapping bug |
| F-POST-02 | Fill consistency violation | CR-1 through CR-5 checks | Reject fill; log violation | Intent fills unchanged | Investigate overfill or causality violation |
| F-POST-03 | KV write failure | NATS KV unavailable | Retry KV write (NATS reconnection) | Fill processed but not persisted | Monitor NATS health; fills re-delivered on reconnect |
| F-POST-04 | ClickHouse write failure | ClickHouse insert error | Log error; do not block pipeline | KV written; CH missing | CH writer catches up from KV/stream |
| F-POST-05 | NATS publish failure | Stream unavailable | Log error; message lost | Intent processed; downstream unaware | Restart triggers re-evaluation; no silent data loss |
| F-POST-06 | Structured log failure | Logger error | Swallow; do not crash pipeline | Data intact; observability degraded | Check log infrastructure |

**Classification:**
- F-POST-01, F-POST-02: **Unacceptable** — indicate invariant violation bugs
- F-POST-03, F-POST-04: **Acceptable** — eventually consistent; self-recovering
- F-POST-05: **Acceptable for testnet** — regenerated on next evaluation cycle
- F-POST-06: **Acceptable** — observability degradation, not data loss

---

## 3. Acceptable vs. Unacceptable Failures

### 3.1 Acceptable Failures

Failures that the system tolerates without operator intervention and without violating invariants.

| Category | Examples | Why Acceptable |
|----------|---------|---------------|
| Transient venue errors | 429, 503, 5xx, network failures | Self-recovering; no state corruption; testnet tolerance |
| Kill switch blocks | PGR-01 triggering | Guard rail working as designed |
| Staleness rejections | PGR-09 triggering | Guard rail working as designed |
| Dedup rejections | JetStream or KV dedup | Idempotency working as designed |
| Side filter blocks | PGR-10 triggering | Configuration working as designed |
| ClickHouse write lag | Temporary CH unavailability | Eventually consistent; no data loss |
| Single order failure | Any non-persistent venue rejection | Expected behavior for invalid market conditions |

### 3.2 Unacceptable Failures

Failures that violate invariants, risk data corruption, or require immediate investigation.

| Category | Examples | Why Unacceptable | Required Response |
|----------|---------|-----------------|-------------------|
| Invariant violation | ValidTransition() failure in production path | State machine corruption | Kill switch + investigate |
| Duplicate venue order | Same intent produces two venue orders | Financial risk (even on testnet) | Kill switch + manual reconciliation |
| Phantom order (undetected) | Timeout with no audit trail | Silent venue-side state divergence | Impossible if logging works; log failure is escalation trigger |
| Fill data corruption | Synthetic prices, wrong quantities, wrong simulated flag | Analytical data pollution | Kill switch + data cleanup |
| Credential exposure | Credentials in logs, error messages, or HTTP responses | Security violation | Immediate incident response |
| Terminal state mutation | `filled` → any other state | Invariant violation | Kill switch + investigate |
| Silent message loss | NATS message lost with no log | Undetectable data gap | Impossible with JetStream ack; consumer restart recovers |

### 3.3 Decision Matrix

```
                    ┌──────────────────────────────────┐
                    │       Is an invariant violated?   │
                    └──────────┬───────────────────────┘
                          Yes  │               No
                               ▼                ▼
                    ┌──────────────┐   ┌──────────────────┐
                    │ UNACCEPTABLE │   │ Is data at risk?  │
                    │ Kill switch  │   └────┬─────────────┘
                    │ + investigate│    Yes │          No
                    └──────────────┘       ▼           ▼
                              ┌──────────────┐  ┌───────────────┐
                              │ UNACCEPTABLE │  │ Self-recovering│
                              │ Kill switch  │  │ or by design?  │
                              │ + remediate  │  └──┬────────────┘
                              └──────────────┘ Yes │        No
                                                   ▼         ▼
                                         ┌──────────┐ ┌──────────────┐
                                         │ACCEPTABLE│ │ Log + monitor│
                                         │ No action│ │ Investigate  │
                                         └──────────┘ │ if persistent│
                                                      └──────────────┘
```

---

## 4. Idempotency Detail

### 4.1 Idempotency Guarantee Surface

| Boundary | Guarantee | Mechanism | Proven |
|----------|-----------|-----------|--------|
| Message → Intent | Same pipeline event produces at most one intent | JetStream dedup key (24h) | Yes (S271) |
| Intent → KV | Same intent written at most once to KV | KV optimistic concurrency (expected seq) | Yes (S271) |
| Intent → Venue | Same intent produces at most one venue order | Client order ID (IDEM-3) | **No — EC-1 gap** |
| Venue → Fill | Same venue fill recorded at most once | Append-only fills + monotonic FilledQuantity | Yes (S308) |
| Fill → ClickHouse | Same fill written at most once | Idempotent insert (ReplacingMergeTree + dedup key) | Yes (S159) |

### 4.2 Idempotency Failure Scenarios

| Scenario | Current Risk | Mitigation |
|----------|-------------|------------|
| Network partition during NATS publish | JetStream retries with dedup key; safe | None needed |
| Adapter crash after venue accepts but before response | Venue order exists; system unaware | Manual audit via VenueOrderID; timeout logging |
| Consumer restart processes same message | JetStream dedup catches within 24h | None needed |
| KV write after intent already materialized | Expected sequence mismatch; write rejected | None needed |
| Same signal triggers two evaluation cycles | Different unix timestamps → different dedup keys → two intents | Staleness guard rejects the delayed one |

### 4.3 Idempotency Gap Risk Assessment

The missing client order ID (EC-1) creates the following risk:

```
Timeline:
  T0: Execute actor dispatches HTTP POST to venue
  T1: Network timeout (context deadline exceeded)
  T2: Adapter returns error; intent stays `submitted`
  T3: (No retry in S310 — intent is abandoned)

  BUT: At T0.5, venue received and executed the order.
  Result: Phantom order at venue; system unaware.
```

**Risk for testnet:** LOW — no financial consequence; phantom testnet orders are harmless.
**Risk for production:** HIGH — would require client order ID + reconciliation.

---

## 5. Timeout Semantics Detail

### 5.1 Timeout Hierarchy

| Timeout | Default | Configurable | Effect |
|---------|---------|-------------|--------|
| Venue HTTP call | 10s | Yes (via context) | Request cancelled; adapter returns timeout error |
| Staleness guard | 60s | Yes (via config) | Intent rejected before venue call |
| NATS consumer ack | 30s | Yes (JetStream config) | Message redelivered (dedup key prevents duplicate processing) |
| Kill switch KV read | 5s | Yes (via context) | Fail-open: submission proceeds |
| ClickHouse insert | 30s | Yes (via adapter config) | Insert retried by writer; no data loss |

### 5.2 Timeout Interaction Matrix

| Scenario | Venue Call | Staleness | Consumer Ack | Behavior |
|----------|-----------|-----------|-------------|----------|
| Normal operation | < 1s | Fresh | < 5s | Happy path |
| Slow venue | 5-10s | Fresh | Within ack window | Succeeds but slow |
| Venue timeout | > 10s | Fresh | Within ack window | Timeout; phantom risk |
| Stale intent | N/A | > 60s | N/A | Rejected by staleness guard |
| Slow consumer | < 1s | Fresh | Near ack limit | Succeeds; ack may extend |
| Everything slow | 5-10s | Near limit | Near ack limit | Race condition: may timeout + redeliver + dedup |

---

## 6. Reconciliation Procedure (Manual)

Since S310 does not implement automated reconciliation, the following manual procedure documents how an operator would investigate phantom orders or data inconsistencies.

### 6.1 Phantom Order Investigation

```
1. Query structured logs for timeout events:
   - Filter: level=WARN, category=Unavailable, "context deadline exceeded"
   - Extract: correlation_id, symbol, timestamp, venue_request_details

2. For each timeout event, query venue API:
   - GET /fapi/v1/allOrders with symbol and time range
   - Match by timestamp and quantity (no client order ID yet)

3. If venue order found:
   - Record venue order ID
   - Determine fill status
   - Manually reconcile with system state

4. If no venue order found:
   - Timeout occurred before venue received request
   - No action needed
```

### 6.2 KV-ClickHouse Consistency Check

```
1. Query KV for all execution intents in time range
2. Query ClickHouse for all execution rows in same time range
3. Compare counts and correlation IDs
4. Missing from CH: writer lag or failure — check writer logs
5. Missing from KV: should not happen (KV is upstream) — investigate
```

### 6.3 Fill Consistency Audit

```
1. For each intent with status=filled:
   - Verify sum(fill.Quantity) == intent.FilledQuantity
   - Verify sum(fill.Quantity) == intent.Quantity
   - Verify all fill.Timestamp >= intent.Timestamp
   - Verify all fill.Simulated == false (for venue fills)

2. For each intent with status=rejected:
   - Verify len(fills) == 0

3. For each intent with status=cancelled:
   - Verify fills are preserved (may be 0 or partial)
   - Verify FilledQuantity matches sum of fills
```

---

## 7. Failure Propagation Rules

### 7.1 Error Containment Principle

Failures in one pipeline stage must not corrupt or silently affect other stages.

| Rule | Description |
|------|------------|
| FP-1 | Venue call failure does not mutate intent state (PGR-08) |
| FP-2 | KV write failure does not block or corrupt NATS message processing |
| FP-3 | ClickHouse write failure does not block KV writes |
| FP-4 | Log failure does not block data processing |
| FP-5 | One symbol's failure does not affect another symbol's processing |
| FP-6 | Adapter errors are always wrapped in `*problem.Problem` — no bare Go errors escape |

### 7.2 Error Boundary Map

```
┌─────────────────────────────────────────────────────────────────────┐
│ Execute Actor                                                       │
│  ┌──────────────────┐                                               │
│  │ Pre-Submit Checks │ F-PRE failures terminate here                │
│  │ (kill switch,     │ No venue side-effects                        │
│  │  staleness, side) │                                              │
│  └────────┬─────────┘                                               │
│           │ pass                                                    │
│  ┌────────▼──────────────────────────────────────────┐              │
│  │ Venue Adapter                                      │              │
│  │  ┌──────────┐    ┌───────────┐    ┌─────────────┐ │              │
│  │  │ HTTP Call │───▶│ Parse     │───▶│ Map to      │ │              │
│  │  │          │    │ Response  │    │ Domain      │ │              │
│  │  └──────────┘    └───────────┘    └─────────────┘ │              │
│  │  F-VEN failures return *problem.Problem           │              │
│  └───────────────────────────────────────────────────┘              │
│           │ success                                                 │
│  ┌────────▼─────────┐                                               │
│  │ Post-Submit       │ F-POST failures here                         │
│  │ (state transition,│ Intent already at venue                      │
│  │  fill recording,  │ Must preserve venue result                   │
│  │  KV write, pub)   │                                              │
│  └──────────────────┘                                               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 8. Guard Rail Verification Matrix

Each guard rail and failure mode must be verifiable through tests.

| Guard Rail / Failure | Test Type | Infrastructure | Verification |
|---------------------|-----------|---------------|-------------|
| PGR-01 (kill switch) | Unit + Scenario | NATS KV | ControlGate halted → submission blocked |
| PGR-02 (monotonicity) | Unit | None | Invalid transitions rejected |
| PGR-03 (timeout) | Unit | httptest.Server with delay | Context cancellation fires |
| PGR-08 (no intermediate state) | Unit | httptest.Server returning errors | Intent stays `submitted` after all error types |
| PGR-09 (staleness) | Unit | None (time manipulation) | Old intents rejected |
| PGR-11 (dedup) | Scenario | JetStream | Duplicate messages produce single intent |
| PGR-12 (terminal absorption) | Unit | None | Terminal states reject all transitions |
| PGR-13 (fill consistency) | Unit | None | CR-1 through CR-5 enforced |
| F-VEN-07 (timeout phantom) | Unit | httptest.Server with timeout | Timeout logged; intent stays `submitted` |
| F-POST-01 (transition failure) | Unit | None | Invalid transition rejected; state preserved |

---

## 9. Summary of Boundaries

| Dimension | Boundary | Rationale |
|-----------|----------|-----------|
| Retry | None (no automatic retry) | Idempotency gap (EC-1); testnet tolerance |
| Reconciliation | Manual only | No position model; testnet scope |
| Kill switch | Global, fail-open | Testnet scope; KV unavailability is itself an alert |
| Idempotency | 2 of 3 layers proven | Layer 3 (venue client order ID) is EC-1 blocker |
| Timeout | 10s venue call; 60s staleness | Conservative defaults; configurable |
| Stop mechanism | 3 levels (kill switch, process, infra) | Graduated response |
| Failure tolerance | Transient venue errors absorbed | No financial risk on testnet |
| Failure escalation | Invariant violations, credential exposure | Unacceptable regardless of environment |

---

*Delivered: 2026-03-21 — Stage S310, Phase 30*
