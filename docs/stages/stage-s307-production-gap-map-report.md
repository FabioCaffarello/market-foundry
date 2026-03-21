# Stage S307 — Production Gap Map Report

> **Phase:** 30 — Venue Readiness
> **Stage:** S307
> **Status:** DELIVERED
> **Date:** 2026-03-21
> **Predecessor:** S306 (Venue Readiness Charter & Scope Freeze)
> **Successor:** S308 (Fill Model Validation & Lifecycle Proof)

---

## 1. Executive Summary

S307 maps the structural gap between market-foundry's paper execution envelope and the venue readiness target defined by S306. The audit covered eight capability layers across the entire execution path — from domain models through adapters, NATS messaging, ClickHouse persistence, and operational controls.

**Verdict:** The architecture is sound. No redesign is required. The gap is concentrated in **adapter hardening** (error classification, idempotency, timeout), **fill model fidelity** (real prices/quantities/fees), and **failure containment** (network/auth/rate/rejection classification). All gaps map cleanly to the S307–S312 stage sequence.

**By the numbers:**
- **18 existing capabilities** reusable as-is (domain model, contracts, dedup, kill switch, NATS pipeline, ClickHouse, composite read model)
- **7 partial capabilities** requiring hardening (adapter, status mapping, error classification, timeout, partial fills)
- **11 absent capabilities** requiring implementation (6 within S307–S311, 5 deferred post-S312)
- **15 blockers** identified and assigned to specific stages
- **0 architectural redesigns** needed

---

## 2. Deliverables

| # | Deliverable | Path | Status |
|---|---|---|---|
| 1 | Production Gap Map | `docs/architecture/production-gap-map-from-paper-execution-to-venue-readiness.md` | DELIVERED |
| 2 | Gap Matrix with Dependencies & Blockers | `docs/architecture/venue-readiness-gap-matrix-dependencies-and-blockers.md` | DELIVERED |
| 3 | Stage Report (this file) | `docs/stages/stage-s307-production-gap-map-report.md` | DELIVERED |

---

## 3. Capability Summary

### Existing & Reusable (18 capabilities)

The paper execution pipeline is architecturally complete. The following require no changes for venue readiness:

- **Domain:** ExecutionIntent model, FillRecord schema, state machine, validation
- **Contracts:** VenuePort interface, VenueOrderRequest, VenueOrderReceipt
- **Dedup:** JetStream message ID, KV monotonicity guard, fill dedup key
- **Control:** Global kill switch (ControlGate), staleness guard, SafetyGate
- **Credentials:** CredentialSet with env var binding, no credential leaks
- **Pipeline:** NATS event publishing, KV materialization, ClickHouse write path
- **Read model:** Composite reader (chain, chains, funnel, disposition endpoints)
- **Observability:** Structured logging, health tracker counters
- **Isolation:** Multi-symbol partition keys, per-actor scoping

### Partial — Requires Hardening (7 capabilities)

| Capability | Exists | Gap |
|---|---|---|
| BinanceFuturesTestnetAdapter | Market order submission, HMAC signing | Edge-case responses, body cap, error classification |
| Binance status mapping | NEW, FILLED, PARTIALLY_FILLED | EXPIRED, CANCELED, REJECTED |
| Error classification | 401, 429, 5xx | Retryable flag inconsistent, no malformed handling |
| Request timeout | HTTP client timeout | No per-request context deadline |
| Partial fill tracking | Fills array | No aggregation to FilledQuantity |
| Retry classification | Problem types exist | Actor has no venue-retry policy (NG-12 accepts) |
| Per-symbol degradation | Global kill switch | No per-symbol halt (3-symbol testnet acceptable) |

### Absent — Must Be Built (11 capabilities)

**Within venue readiness wave (S307–S311):**
- Venue-side client order ID (idempotent retry)
- Response body size cap
- Fill model with real prices/quantities/fees/timestamps
- E2E venue integration test
- Failure injection tests
- Multi-symbol venue isolation test

**Deferred post-S312 (by non-goal):**
- Reconciliation (NG-5)
- Distributed tracing (NG-6)
- Prometheus metrics (NG-6)
- Per-symbol kill switch
- Outbound rate limiting

---

## 4. Blocker Distribution by Stage

| Stage | Blockers | Key Gaps |
|---|---|---|
| **S307** (Adapter Hardening) | 5 | Client order ID, error classification, retryable flags, body cap, timeout |
| **S308** (Fill Model) | 6 | EXPIRED/CANCELED/REJECTED mapping, real price/qty/fee/timestamp, Simulated=false |
| **S309** (E2E Integration) | 2 | Kill switch + staleness under real venue |
| **S310** (Failure Envelope) | 1 | Network failure classification |
| **S311** (Multi-Symbol) | 1 | Concurrent venue isolation (implicit) |
| **Total** | **15** | |

---

## 5. Critical Dependency Chain

```
S306 (Charter) ──DONE──▶ S307 (Adapter) ──▶ S308 (Fill) ──▶ S309 (E2E)
                                                                  │
                                                           ──▶ S310 (Failure)
                                                                  │
                                                           ──▶ S311 (Multi-Symbol)
                                                                  │
                                                           ──▶ S312 (Gate)
```

All dependencies are **hard** — each stage requires the previous stage's outputs. No parallelism is possible in the critical path.

---

## 6. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Testnet returns unexpected statuses | Medium | Blocks S308 | Unit tests with mocked responses |
| Testnet rate limiting during multi-symbol | Medium | Blocks S311 | Sequential submission with delays |
| Fill data doesn't fit ClickHouse schema | Low | Violates NG-10 | Pre-validate in S308 unit tests |
| Actor redelivery storm after venue timeout | Medium | Duplicate submissions | Client order ID prevents double-fill |

---

## 7. Acceptance Criteria — Verdict

| Criterion | Status |
|---|---|
| Gap map is clear, honest, and actionable | **MET** — 8 layers mapped, every gap classified |
| Blockers and dependencies are explicit | **MET** — 15 blockers assigned, dependency graph documented |
| Ambiguity about paper → venue leap is reduced | **MET** — existing/partial/absent classification eliminates guesswork |
| Base is ready for contracts and invariants in S308 | **MET** — fill model gaps (FM-1 through FM-7) are the S308 input |

---

## 8. Guard Rail Compliance

| Guard Rail | Status |
|---|---|
| No venue implementation started | **COMPLIANT** — zero code written |
| No redesign or scope inflation | **COMPLIANT** — architecture validated, no changes proposed |
| No gaps hidden | **COMPLIANT** — reconciliation, tracing, metrics listed as absent |
| S306 non-goals respected | **COMPLIANT** — 12 non-goals mapped to deferred gaps |

---

## 9. S308 Handoff

S308 (Fill Model Validation & Lifecycle Proof) receives from S307:

1. **Gap matrix** with every fill model gap uniquely identified (FM-1 through FM-7)
2. **Adapter hardening prerequisites** (VA-1 through VA-5, EC-1 through EC-3) — these must be resolved in S307 before S308 can validate fill semantics
3. **Dependency chain** confirming S308 depends on S307 completion
4. **Test strategy input**: every Binance status (NEW, FILLED, PARTIALLY_FILLED, EXPIRED, CANCELED, REJECTED) must map to a domain state with unit test proof

**S308 entry conditions:**
- S307 adapter hardening complete (error classification, client order ID, body cap, timeout)
- All Binance error codes classified with retryable flag
- Response body parsing hardened with size limits

---

## 10. Stage Sequence Confirmation

The S306 charter proposed S306–S312. This gap map confirms the sequence is correct and complete:

| Stage | Title | S307 Finding |
|---|---|---|
| S306 | Charter & Scope Freeze | DONE — wave open, scope frozen |
| **S307** | **Production Gap Map** | **THIS STAGE — gap map delivered** |
| S308 | Fill Model Validation & Lifecycle Proof | Blockers identified (FM-1 through FM-7) |
| S309 | Venue Execution E2E Integration | Depends on S307 + S308 |
| S310 | Venue Failure Envelope & Containment | Depends on S309 |
| S311 | Multi-Symbol Venue Isolation Proof | Depends on S310 |
| S312 | Venue Readiness Gate & Wave Closure | Depends on S311 — answers VQ1–VQ7 |

No additional stages are needed. No stage reordering is recommended.
