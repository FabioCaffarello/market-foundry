# Venue Integration Entry Prerequisites

> Mandatory conditions that must be satisfied before any venue-integrated execution code enters Market Foundry.
> Date: 2026-03-18 | Stage: S74

## Purpose

This document defines the hard prerequisites for crossing the action boundary — transitioning from paper execution (fire-and-forget intents) to venue-integrated execution (real-world order placement). Each prerequisite has a concrete verification method and resolution path.

---

## Category A: Unresolved S68 Prerequisites

These were required by the original S68 readiness review but were never formally resolved.

### A-1: Derive Actor Test Coverage

**Status**: UNRESOLVED since S68

**What**: No unit tests exist for derive actor message routing, fan-out correctness, or error isolation. The execution evaluator actor, publisher actor, and source scope fan-out paths are untested at the actor level.

**Why this blocks venue integration**: A fan-out bug in SourceScopeActor could send a risk assessment to the wrong execution evaluator, producing an order for the wrong symbol. This is acceptable for paper execution (no real consequence) but catastrophic for real orders.

**Verification**: `go test ./internal/actors/scopes/derive/...` must cover execution evaluator message routing and fan-out correctness.

**Resolution**: Dedicated derive actor test stage.

### A-2: Automated Traceability Verification

**Status**: UNRESOLVED since S68

**What**: The causal chain (correlation_id + causation_id) was hardened in S67, but verification is manual. No automated test asserts that the full chain observation → evidence → signal → decision → strategy → risk → execution maintains trace integrity.

**Why this blocks venue integration**: Every order placed at a venue must be traceable to the market data, signal, decision, strategy, and risk assessment that produced it. Without automated verification, a broken chain may go undetected until post-incident analysis.

**Verification**: Integration test with running NATS that publishes synthetic trade → asserts full chain correlation_id integrity.

**Resolution**: Dedicated traceability verification stage.

### A-3: Trace Metadata Persistence in Projections

**Status**: UNRESOLVED since S68

**What**: KV projections store only domain model fields. Querying `GET /execution/paper_order/latest` returns the execution intent without correlation_id or causation_id. Audit trail requires JetStream replay or log aggregation.

**Why this blocks venue integration**: Operational audit queries must return the full provenance chain without requiring infrastructure-level tools. A compliance review or incident investigation needs "show me the chain that produced this order" — not "grep the NATS stream."

**Options** (from S68):
1. Embed trace IDs in KV projection payload
2. Separate audit KV bucket
3. JetStream replay (status quo)
4. Dedicated audit stream

**Verification**: Design decision documented + implemented. `GET /execution/paper_order/latest` response includes trace context.

**Resolution**: Design in S75 (design-only), implement in subsequent stage.

### A-4: Kill Switch Mechanism

**Status**: UNRESOLVED since S68

**What**: No mechanism to halt execution without binary restart. Configuration-driven halt via configctl event is designed but not implemented.

**Why this blocks venue integration**: In a live execution scenario, seconds matter. A restart takes 5-10 seconds including NATS reconnection and consumer rebalancing. A kill switch should take effect within 1 event cycle (< 1 second).

**Verification**: Kill switch activates within 1 event cycle, is auditable, and recoverable without restart.

**Resolution**: Design in S75 (design-only), implement in subsequent stage.

---

## Category B: New Prerequisites for Venue Integration

These are new requirements that were not in S68's scope (which assessed paper execution readiness).

### B-1: Execution Lifecycle State Machine

**What**: The current domain model has only `StatusSubmitted`. Venue-integrated execution requires a complete lifecycle: submitted → sent → accepted → filled → partially_filled → rejected → cancelled → expired.

**Why this blocks venue integration**: Without lifecycle tracking, the system cannot distinguish between "intent generated" and "order placed at venue." This is the minimum state machine for any order-to-cash workflow.

**Verification**: Domain model includes lifecycle enum, transition rules, and validation tests.

### B-2: Fill Tracking Model

**What**: No fields exist for filled quantity, fill price, fill timestamp, or venue order ID. The system can generate intents but cannot record outcomes.

**Why this blocks venue integration**: Without fill tracking, execution is write-only. There is no feedback loop, no reconciliation, no PnL calculation possible.

**Verification**: ExecutionIntent or a new entity carries fill fields with validation.

### B-3: Venue Adapter Port Interface

**What**: No port/interface exists for order placement. `ports.ExecutionGateway` only supports query (read). There is no `ports.VenueAdapter` or equivalent for write operations.

**Why this blocks venue integration**: Without a port interface, venue adapter implementation would bypass the hexagonal architecture boundary. The adapter must be injected, testable, and mockable.

**Verification**: Port interface defined in `internal/application/ports/` with clear contract.

### B-4: Failure Recovery for Publish and Projection

**What**: Two critical failure paths silently lose data:
1. NATS publish failure → event dropped, no retry
2. KV write failure → message ACK'd despite write failure

**Why this blocks venue integration**: A dropped execution event means a missed or untracked order. An ACK'd-but-not-written projection means the system believes an order state that doesn't match reality.

**Verification**: Publish retries on transient failure. Projection NAKs on KV write failure (triggers redelivery).

### B-5: Risk Assessment Staleness Guard

**What**: Execution evaluators accept risk assessments of any age. There is no staleness window.

**Why this blocks venue integration**: A 5-minute-old risk assessment may be based on market conditions that no longer exist. Executing an order based on stale risk data violates the "current market conditions" principle.

**Verification**: Configurable staleness threshold. Assessments older than threshold are logged and rejected.

### B-6: Operational Validation with Live Pipeline

**What**: Execution has never been smoke-tested with a real pipeline (Binance WS → full chain → execution projection). All validation is structural (unit tests).

**Why this blocks venue integration**: Structural correctness does not guarantee operational correctness. Consumer lag, NATS throughput, memory pressure, and timing issues only manifest under real load.

**Verification**: `make up && make seed-multi && make smoke-multi` passes with execution steps 13-14 showing materialized data (not null).

---

## Prerequisite Dependency Graph

```
UNRESOLVED S68:
  A-1 (derive actor tests) ────────────────────────────┐
  A-2 (automated trace verification) ──────────────────├──→ B-6 (operational validation)
  A-3 (trace persistence) ─── design in S75 ───────────┤
  A-4 (kill switch) ─── design in S75 ─────────────────┘
                                                         │
NEW VENUE PREREQUISITES:                                 ↓
  B-1 (lifecycle state machine) ─── design in S75 ──→ Implementation Stage
  B-2 (fill tracking model) ─── design in S75 ──────→ Implementation Stage
  B-3 (venue adapter port) ─── design in S75 ───────→ Implementation Stage
  B-4 (failure recovery) ─── design in S75 ─────────→ Implementation Stage
  B-5 (staleness guard) ─── design in S75 ──────────→ Implementation Stage
```

---

## Verification Checklist

Before any venue-integrated execution code enters the repository:

**S68 carryover (must be implemented, not just designed):**
- [ ] A-1: Derive actor test coverage for execution message routing
- [ ] A-2: Automated traceability verification test passing
- [ ] A-3: Trace metadata persisted in execution projections
- [ ] A-4: Kill switch mechanism operational

**New venue prerequisites (must be designed first, then implemented):**
- [ ] B-1: Execution lifecycle state machine designed and tested
- [ ] B-2: Fill tracking model designed and tested
- [ ] B-3: Venue adapter port interface defined
- [ ] B-4: Publish and projection failure recovery implemented
- [ ] B-5: Risk assessment staleness guard implemented
- [ ] B-6: Operational validation with live pipeline passing

**No prerequisite may be skipped.** The action boundary is the highest-stakes transition in Market Foundry's evolution.
