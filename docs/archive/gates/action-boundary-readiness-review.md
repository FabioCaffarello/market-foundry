# Action Boundary Readiness Review

> Formal readiness assessment for crossing the action boundary — venue-integrated execution.
> Date: 2026-03-18 | Stage: S74

## 1. Review Scope

This document evaluates whether Market Foundry is ready to cross the **action boundary** — the transition from paper execution (fire-and-forget intents) to venue-integrated execution (real-world order placement with external system interaction).

**Guiding principle**: Readiness by proof, not by design confidence. Every dimension is evaluated against concrete evidence, not architectural intent.

**Critical distinction**: The S68 readiness review assessed readiness to *design* execution. This review assesses readiness to *integrate with external venues*. The bar is categorically higher.

---

## 2. S68 Prerequisite Checklist — Resolution Status

The S68 readiness review defined 9 hard prerequisites. Their resolution status:

| ID | Prerequisite | Required By | Resolved? | Where |
|----|-------------|-------------|-----------|-------|
| A-1 | Adapter test coverage sweep | S68 | YES | S60 — 55 adapter tests across 6 domains |
| A-2 | Derive actor test coverage | S68 | NO | Never addressed in any stage |
| B-1 | Automated traceability verification test | S68 | NO | S67 hardened propagation but explicit: "no automated verification test" |
| B-2 | Trace metadata persistence decision | S68 | NO | S67/S72 both carry this as known limitation |
| C-1 | Risk drift rules verified | S68 | YES | S63 risk governance activation |
| C-2 | Execution governance rules active | S68 | YES | S70 execution governance activation |
| D-1 | Execution domain boundary definition | S68 | YES | S69 execution domain design (7 documents) |
| D-2 | Venue adapter architecture decision | S68 | YES | S69 — paper-only first, venue deferred |
| E-1 | Kill switch design | S68 | NO | Documented as non-blocking for paper, never designed |

**Resolved: 5/9 | Unresolved: 4/9**

### Unresolved Prerequisites Assessment

**A-2 (Derive actor tests)**: No unit tests exist for derive actor message routing, fan-out correctness, or error isolation. The execution evaluator actor, publisher actor, and source scope fan-out are untested at the actor level. Risk assessment: MEDIUM for paper execution, HIGH for venue integration.

**B-1 (Automated trace verification)**: The causal chain (correlation_id + causation_id) flows correctly through code (verified by manual inspection in S67), but no automated test asserts chain integrity end-to-end. Risk assessment: MEDIUM — structural integrity is high but verification is manual.

**B-2 (Trace metadata in projections)**: KV projections store only domain model fields. `GET /execution/paper_order/latest` returns the execution intent without correlation_id or causation_id. Post-trade audit requires JetStream replay or log aggregation. Risk assessment: HIGH for venue integration — every order must be traceable to its full causal chain.

**E-1 (Kill switch)**: No mechanism exists to halt execution without binary restart. `pipeline.execution_families` removal requires config change + restart. Risk assessment: LOW for paper execution, CRITICAL for venue integration.

---

## 3. Execution Domain Maturity

### 3.1 What Exists and Works

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Domain model | COMPLETE | ExecutionIntent with 12 fields, 3 enums, Validate(), PartitionKey(), DeduplicationKey() |
| Application logic | COMPLETE | PaperOrderEvaluator — pure, deterministic, tested (8 tests) |
| Client layer | COMPLETE | GetLatestExecutionUseCase with validation |
| Port interface | COMPLETE | ExecutionGateway interface |
| NATS adapters | COMPLETE | Registry, publisher, consumer, KV store, gateway (5 files) |
| Derive actors | COMPLETE | Evaluator + publisher actors |
| Store actors | COMPLETE | Consumer + projection actors with 3-gate pattern |
| HTTP surface | COMPLETE | GET /execution/:type/latest |
| Config activation | COMPLETE | pipeline.execution_families with transitive validation |
| Governance | COMPLETE | 5 active drift checks in raccoon-cli |
| Domain tests | STRONG | 20 tests — validation, partition keys, multi-symbol isolation |
| Application tests | STRONG | 8 tests — evaluator logic, multi-symbol independence |
| Projection tests | STRONG | 15 tests — gates, stats invariant, multi-symbol materialization |
| Multi-symbol proof | VERIFIED | S73 — 43 tests, zero cross-symbol bleed |

### 3.2 What's Missing for Venue Integration

| Gap | Severity | Impact |
|-----|----------|--------|
| Status enum: only "submitted" | HIGH | No way to track submitted → accepted → filled → rejected lifecycle |
| No fill tracking fields | HIGH | No FilledQuantity, FillPrice, FillTimestamp |
| No venue order ID field | HIGH | No link between intent and venue confirmation |
| No venue adapter interface/port | HIGH | No abstraction for order placement |
| No order lifecycle state machine | HIGH | No transition rules, no invalid state protection |
| No failure recovery on publish | HIGH | NATS publish failures silently drop events |
| No DLQ for consumer exhaustion | MEDIUM | After MaxDeliver=5, messages are effectively dead-lettered without visibility |
| KV write failures ACK message | HIGH | Projection gap: message ACK'd despite write failure = data loss |
| No rate limiting | MEDIUM | Venue APIs have rate limits; no simulation infrastructure |
| No staleness window on risk input | MEDIUM | Execution may act on arbitrarily old risk assessments |
| Trace metadata not in projections | HIGH | Cannot audit order → risk chain from query surface |

### 3.3 Domain Maturity Verdict

**Paper execution: 9/10** — Complete, tested, proven multi-symbol. Architecturally sound.

**Venue-integrated execution: 3/10** — Domain model lacks lifecycle, fill tracking, venue identity, and failure recovery. The current model is an *intent generator*, not an *order manager*.

---

## 4. Projection Authority

| Aspect | Status |
|--------|--------|
| Single-writer invariant | ENFORCED — ExecutionProjectionActor is sole writer |
| Store as sole authority | ENFORCED — Gateway reads via NATS request/reply, not direct KV |
| Monotonicity guard | ENFORCED — Timestamp-based stale/duplicate rejection |
| Three-gate projection | ENFORCED — Final → validate → monotonicity |
| Stats invariant | ENFORCED — received == sum(outcomes), checked at shutdown |

**Verdict: PASS** — Projection authority is sound for latest-only paper execution.

**Concern for venue integration**: Projection becomes safety-critical. A stale or missing projection entry could cause duplicate order submission. The monotonicity guard is necessary but not sufficient — venue-integrated execution needs at-least-once delivery with idempotent submission, not fire-and-forget.

---

## 5. Query Surface Quality

| Endpoint | Status | Tests |
|----------|--------|-------|
| GET /execution/paper_order/latest | OPERATIONAL | Handler + route + smoke |
| Error: missing timeframe | 400 | Smoke validated |
| Error: unknown type | 400 | Smoke validated |

**Verdict: PASS for paper execution.**

**Concern for venue integration**: The query surface only supports latest-only by type/source/symbol/timeframe. Venue integration requires:
- Query by order ID (venue reference)
- Query by status (active orders, pending fills)
- Query execution history (last N intents per symbol)
- Real-time status updates (WebSocket or polling)

None of these exist.

---

## 6. Traceability and Auditability

| Aspect | Status |
|--------|--------|
| CorrelationID propagation | ENFORCED — Flows observation → execution |
| CausationID propagation | ENFORCED — Each hop sets predecessor's Metadata.ID |
| Structured log trace context | ENFORCED — Both IDs in all actor logs |
| NATS envelope trace fields | ENFORCED — Both IDs in transport |
| KV projection trace metadata | NOT PERSISTED — Only in streams and logs |
| Automated verification test | NOT IMPLEMENTED — Manual/visual only |

**Verdict: PARTIAL PASS**

For paper execution, log-based traceability is acceptable. For venue integration, it is not:
- Every order placed at a venue must be traceable to the full chain (observation → execution)
- Trace metadata must be queryable, not just log-searchable
- Automated verification is mandatory before any real money is at risk

---

## 7. Governance and CLI

| Check | Status |
|-------|--------|
| ED-1: execution-docs-drift | ACTIVE — 7 docs verified |
| ED-2: execution-adapter-drift | ACTIVE — 5 adapter files verified |
| ED-3: execution-domain-drift | ACTIVE — Domain entity, events, evaluator verified |
| ED-4: execution-config-drift | ACTIVE — Symmetric execution_families |
| ED-5: execution-contracts-drift | ACTIVE — Subjects, durables, buckets verified |

**Verdict: PASS** — Execution is under full governance. raccoon-cli enforces all architectural contracts.

---

## 8. Activation Model

| Mechanism | Status |
|-----------|--------|
| Family activation (pipeline.execution_families) | ENFORCED |
| Dependency validation (paper_order → position_exposure) | ENFORCED |
| Known family registry | ENFORCED |
| Config symmetry (derive ↔ store) | ENFORCED by CLI |

**Verdict: PASS for paper execution.**

**Concern for venue integration**: No kill switch, no separate paper/live activation, no activation ceremony, no activation audit trail. The current model treats execution activation the same as signal activation — this is inappropriate for a domain that can place real orders.

---

## 9. Failure Semantics

### Current State

| Scenario | Behavior | Acceptable for Venue? |
|----------|----------|----------------------|
| NATS down during publish | Log error, drop event | NO — order intent lost |
| Consumer decode error (InvalidArgument) | Term() — permanent removal | PARTIAL — needs DLQ visibility |
| Consumer decode error (transient) | Nak() — up to MaxDeliver=5 | PARTIAL — needs DLQ after exhaustion |
| Projection KV write failure | Log error, ACK message | NO — message ACK'd despite data loss |
| Risk assessment stale | No check | NO — may act on old data |

**Verdict: FAIL for venue integration.**

Paper execution tolerates these failures because no real money is at risk. Venue-integrated execution cannot:
- A dropped execution event means a missed trade or, worse, an untracked order
- An ACK'd-but-not-written projection means the system believes an order exists when it doesn't (or vice versa)
- Acting on stale risk assessment means the order is based on outdated market conditions

---

## 10. Gateway Cleanliness

| Aspect | Status |
|--------|--------|
| No business logic | PASS — Pure request/reply translator |
| No direct KV access | PASS — NATS gateway only |
| Optional activation | PASS — Degrades gracefully if execution unavailable |
| No execution-specific state | PASS — Stateless |

**Verdict: PASS** — Gateway remains clean.

---

## 11. Operational Validation

| Dimension | Status |
|-----------|--------|
| Unit tests | 43 tests passing |
| Smoke script coverage | Steps 13-14 validate execution multi-symbol + isolation |
| Live pipeline validation | UNKNOWN — No evidence of smoke-multi running with live data |
| Performance | UNKNOWN — No latency, throughput, or memory data |
| Stability | UNKNOWN — No long-running operation evidence |
| Consumer lag | UNKNOWN — No monitoring data |

**Verdict: INCOMPLETE** — Structural validation is strong. Operational validation is absent.

---

## 12. Overall Readiness Verdict

### Is Market Foundry ready to cross the action boundary?

**Verdict: CONDITIONALLY READY FOR DESIGN — NOT READY FOR IMPLEMENTATION**

| Gate | Target | Actual | Pass? |
|------|--------|--------|-------|
| Domain model complete | YES | YES (paper only) | PASS |
| Multi-symbol isolation proven | YES | YES (43 tests) | PASS |
| Projection authority enforced | YES | YES | PASS |
| Governance active | YES | YES (5 checks) | PASS |
| Traceability chain complete | YES | PARTIAL (not in KV, not automated) | FAIL |
| Failure semantics adequate | YES | NO (silent drops, ACK-without-write) | FAIL |
| Operational validation | YES | NO (never smoke-tested with live pipeline) | FAIL |
| Kill switch available | YES | NO | FAIL |
| Domain model supports lifecycle | YES | NO (submitted only) | FAIL |

**Pass: 4/9 | Fail: 5/9**

### What This Means for S75

The system is ready to **design** venue-integrated execution but NOT ready to **implement** it. The gaps are concrete and resolvable, but they represent real structural deficiencies that would make a venue integration fragile and unauditable.

**Maximum safe step for S75: DESIGN-ONLY**

---

## 13. Smallest Acceptable Design for Venue Integration

If S75 proceeds as design-only, it should produce:

1. **Venue adapter port definition** — Interface contract for order placement and status polling
2. **Execution lifecycle extension** — Status enum: submitted → sent → accepted → filled → partially_filled → rejected → cancelled → expired
3. **Fill tracking model** — Fields: filled_quantity, average_fill_price, fill_timestamp, venue_order_id
4. **Failure recovery pattern** — How publish failures, consumer exhaustion, and KV write failures are handled
5. **Staleness guard design** — Configurable threshold for risk assessment age
6. **Kill switch mechanism** — configctl-driven halt without binary restart
7. **Trace persistence decision** — How correlation_id/causation_id reach the KV projection

**Explicitly NOT in S75:**
- No venue adapter implementation
- No order placement code
- No WebSocket/REST integration with exchanges
- No portfolio/position tracking
- No OMS logic
- No multi-venue routing
