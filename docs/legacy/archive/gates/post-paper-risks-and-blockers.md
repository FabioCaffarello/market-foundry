# Post-Paper Risks and Blockers

**Stage**: S86
**Date**: 2026-03-18
**Scope**: Residual risks and blockers after paper-integrated execution phase (S74–S85)

---

## 1. Hard Blockers for Real Venue

These must be resolved before any real venue adapter can be implemented or activated.

### HB-POST-1: No Infrastructure-Level Testing

**Severity**: Hard Blocker
**Identified**: S79 (deferred), reconfirmed S86

**Problem**: All tests operate at unit or application level. No embedded NATS tests validate:
- KV monotonicity guard behavior under concurrent writes
- Consumer redelivery under MaxDeliver exhaustion
- Stream creation and deduplication window behavior
- Full actor chain with real JetStream (consumer → adapter → publisher → consumer → projection)

**Risk**: Paper mode masks infrastructure failures that only surface under real NATS operations. A real venue adapter that depends on JetStream guarantees has no test evidence those guarantees hold.

**Resolution path**: Create embedded NATS server test harness (S79 deferred scope).

---

### HB-POST-2: No Docker Compose Integration for Execute

**Severity**: Hard Blocker
**Identified**: S80 (noted), reconfirmed S86

**Problem**: The `execute` binary is not in Docker Compose. Smoke tests for execute-specific steps (fill projection, status propagation through execute) are conditional on manual startup. This means:
- CI cannot validate the full 4-binary pipeline
- Smoke test coverage is incomplete by default
- Operational readiness cannot be asserted without manual intervention

**Risk**: Regression in execute integration goes undetected until manual smoke runs.

**Resolution path**: Add `execute` service to docker-compose.yaml with health checks and dependency ordering.

---

### HB-POST-3: No Fill Reconciliation

**Severity**: Hard Blocker
**Identified**: S86

**Problem**: The system trusts that fills correspond to intents but does not verify this. There is no mechanism to:
- Validate that a `VenueOrderFilledEvent` refers to a known `PaperOrderSubmittedEvent`
- Detect orphan fills (fills without matching intents)
- Detect stuck intents (intents without corresponding fills after timeout)

**Risk**: In paper mode this is invisible (PaperVenueAdapter always produces matching fills). In real venue mode, network failures, partial fills, or exchange-side rejections could produce orphan or stuck states with no detection.

**Resolution path**: Design fill reconciliation model (at minimum: query-time validation in composite status endpoint, correlating intent and result by partition key).

---

### HB-POST-4: No Credential Infrastructure

**Severity**: Hard Blocker
**Identified**: S75 (design), reconfirmed S86

**Problem**: No mechanism exists for:
- Storing exchange API keys securely
- Rotating credentials
- Per-venue authentication configuration
- Credential validation at startup

**Risk**: Cannot implement any real venue adapter without this foundation.

**Resolution path**: Design credential infrastructure (likely: encrypted config or secret manager integration, validated at `buildVenueAdapter()` time).

---

## 2. Structural Risks

These are not blockers but represent architectural risks that increase in severity as the system approaches real venue execution.

### SR-1: Transitional Bridge Coupling

**Severity**: Medium
**Current Impact**: Low (documented, acceptable for paper)
**Future Impact**: High (blocks venue-specific subject introduction)

**Problem**: Execute binary subscribes to `execution.events.paper_order.submitted.>` as a transitional bridge. When `VenueOrderIntentEvent` is introduced with subject `execution.events.venue_market_order.submitted`, the consumer spec must migrate. S85 documents a 5-step migration but it has not been executed.

**Mitigation**: Execute migration before or concurrent with real venue adapter implementation.

---

### SR-2: Global Kill Switch

**Severity**: Medium
**Current Impact**: Low (single venue type, single gate is sufficient)
**Future Impact**: Medium (multi-venue routing requires per-venue or per-symbol gates)

**Problem**: EXECUTION_CONTROL KV has a single "global" key. Halting execution halts all symbols across all venue types. Per-venue or per-symbol granularity is not available.

**Mitigation**: Acceptable for paper mode and first real venue step. Granular gates should be designed before multi-venue activation.

---

### SR-3: Synchronous Fill Model

**Severity**: Medium
**Current Impact**: None (PaperVenueAdapter is instant)
**Future Impact**: High (real venues are async: submit → poll/websocket → fill)

**Problem**: VenuePort.SubmitOrder returns a synchronous result. PaperVenueAdapter fills instantly. Real venues require:
- Asynchronous order submission (submit returns order ID, fill arrives later)
- Status polling or websocket subscription for fill events
- Timeout and cancellation handling
- Partial fill accumulation

**Mitigation**: VenuePort interface may need extension (`SubmitOrder` returns receipt, separate fill channel). This is a design concern for the venue adapter design stage.

---

### SR-4: No Observability Surface

**Severity**: Low-Medium
**Current Impact**: Low (stats logged at shutdown)
**Future Impact**: High (production monitoring requires runtime visibility)

**Problem**: Actor counters (processed, filled, skipped_stale, skipped_halt, errors) are only logged at shutdown. No HTTP endpoint or metrics exporter provides runtime visibility.

**Risk**: In production, operators cannot diagnose stuck pipelines, high skip rates, or error spikes without restarting the service.

**Mitigation**: Expose actor stats via health/metrics HTTP endpoint (execute already runs HTTP on :8084).

---

### SR-5: No Dead Letter Path

**Severity**: Low
**Current Impact**: None (MaxDeliver=5, terminated events disappear)
**Future Impact**: Medium (real venue events that exhaust retries are lost)

**Problem**: Events that fail MaxDeliver attempts are terminated (removed from consumer). No dead letter queue captures them for investigation.

**Mitigation**: Acceptable for paper mode. Design dead letter stream before real venue activation.

---

### SR-6: Latest-Only Projection Semantics

**Severity**: Low
**Current Impact**: None (paper mode, stream provides 72h history)
**Future Impact**: Medium (audit requirements may demand queryable history)

**Problem**: KV buckets store only the latest intent/fill per partition key. Historical states require reading raw JetStream (72h retention). No queryable history API exists.

**Mitigation**: Acceptable for current scope. History bucket can be added when audit requirements formalize.

---

## 3. Test Coverage Gaps

| Area | Gap | Severity | Notes |
|------|-----|----------|-------|
| Consumer actors | No unit tests | Low | Tested via integration; simple delegation pattern |
| Store supervisor | No unit tests | Low | Declarative wiring; tested via integration |
| Query responder | No unit tests | Medium | Request-reply handlers with multiple routes; integration-tested but unit coverage would catch regressions faster |
| VenueAdapterActor | No actor-level unit tests | Medium | Multi-gate logic tested via integration but gate interaction not unit-tested |
| NATS round-trip | No embedded NATS tests | High | See HB-POST-1 |
| Multi-binary coordination | No end-to-end CI test | High | See HB-POST-2 |

---

## 4. Governance Gaps

| Area | Gap | Severity | Notes |
|------|-----|----------|-------|
| Activation gate ceremony | Not initiated | Informational | 17-gate ceremony designed in S75, prerequisites not met |
| Real venue type registration | Not possible | By design | `knownVenueTypes` only allows `paper_simulator` |
| ED-2 rule removal | Completed (S83) | None | Pre-implementation guard correctly removed |
| Drift rule for Docker Compose | Missing | Low | No drift rule validates execute service presence in compose |

---

## 5. Priority Matrix

| Item | Priority | Stage Estimate | Blocks |
|------|----------|---------------|--------|
| HB-POST-2: Docker Compose | P0 | S87 | CI validation, smoke reliability |
| HB-POST-1: NATS integration tests | P0 | S87 | Infrastructure confidence |
| SR-4: Observability surface | P1 | S87 | Production readiness |
| HB-POST-3: Fill reconciliation | P1 | S87-S88 | Venue activation |
| SR-1: Transitional bridge | P1 | S88 | Venue-specific subjects |
| SR-3: Async fill model design | P2 | S88 (design) | Real venue adapter |
| HB-POST-4: Credential infrastructure | P2 | S88 (design) | Real venue adapter |
| SR-2: Granular kill switch | P3 | S89+ (design) | Multi-venue |
| SR-5: Dead letter path | P3 | S89+ | Production resilience |
| SR-6: History bucket | P3 | S89+ | Audit requirements |
