# Stage S490 -- Event-Driven Verification Trigger Report

**Stage**: S490
**Type**: Implementation
**Status**: COMPLETE
**Date**: 2026-03-26
**Gap Closed**: G-OA1 (event-driven auto-trigger for verification)
**Predecessor**: S488 (Session Intelligence Evidence Gate)

---

## 1. Executive Summary

S490 delivers an event-driven verification trigger that automatically runs the PO verification pipeline (PO-1 through PO-9) when a session reaches a terminal state (closed or halted). This closes G-OA1, the primary automation gap identified in S488.

The trigger follows the existing NATS JetStream event-driven pattern: the execute binary publishes a `SessionLifecycleEvent` on session close, and the gateway binary consumes it via a durable consumer to run verification. Manual verification paths remain fully functional.

---

## 2. Capabilities Delivered

### C1: Session Lifecycle Event Publishing

- New `SessionLifecycleEvent` domain type carrying session ID, status, operator, config snapshot.
- New `SESSION_LIFECYCLE_EVENTS` JetStream stream (7-day retention, 16 MB).
- `PublishSessionLifecycle` method on the existing execution publisher.
- Execute supervisor publishes the event after persisting session close to KV.
- Publisher failure is non-fatal: log warning, session data intact in KV.

### C2: Gateway Verification Trigger Consumer

- Durable consumer `gateway-verification-trigger` on the session lifecycle stream.
- `TriggerVerifySessionUseCase` validates terminal status, waits 5s for CH settle, runs `VerifySessionUseCase`.
- Results logged with full summary (pass/fail/warn/skip counts).
- Consumer start failure is non-fatal: gateway operates normally, manual verification works.

### C3: Deduplication and Anti-Loop Protection

- JetStream message ID = `session-lifecycle:{session_id}:{status}` for at-most-once delivery.
- One-way trigger direction (session close → verification); no feedback loop.
- Unconditional ack after processing — verification failure does not cause retry.
- `MaxDeliver: 5` as safety net against infinite delivery loops.

### C4: Fail-Closed Semantics

- Publisher failure: non-fatal, manual fallback available.
- Consumer failure: non-fatal, gateway HTTP server unaffected.
- Verification failure: logged, acked, no retry (same inputs would fail again).
- Non-terminal events: silently skipped (only closed/halted trigger verification).

---

## 3. Evidence of G-OA1 Closure

### Before S490

| Trigger | Mechanism | Operator Required |
|---------|-----------|-------------------|
| `make po-verify` | Script invocation | Yes |
| `GET /session/:id/verify` | HTTP request | Yes |
| `scripts/po-verify.sh` | Shell script | Yes |

### After S490

| Trigger | Mechanism | Operator Required |
|---------|-----------|-------------------|
| Session close event | JetStream consumer | **No** |
| Session halt event | JetStream consumer | **No** |
| `make po-verify` | Script invocation | Yes (preserved) |
| `GET /session/:id/verify` | HTTP request | Yes (preserved) |

The verification pipeline can now fire without operator intervention when the right event occurs. G-OA1 is closed.

---

## 4. Files Changed

### New Files

| File | Role |
|------|------|
| `internal/domain/execution/session_event.go` | SessionLifecycleEvent domain type + dedup key |
| `internal/domain/execution/session_event_test.go` | Dedup key tests |
| `internal/adapters/nats/natsexecution/session_lifecycle_consumer.go` | JetStream consumer for lifecycle events |
| `internal/application/executionclient/trigger_verify_session.go` | Trigger use case |
| `internal/application/executionclient/trigger_verify_session_test.go` | Nil-safety and constructor tests |
| `cmd/gateway/verification_trigger.go` | Gateway-side consumer wiring |
| `cmd/gateway/verification_trigger_test.go` | Structural tests |
| `docs/architecture/event-driven-verification-trigger.md` | Architecture reference |
| `docs/architecture/verification-trigger-events-dedup-fail-closed-semantics-and-limitations.md` | Dedup/fail-closed semantics |

### Modified Files

| File | Change |
|------|--------|
| `internal/adapters/nats/natsexecution/registry.go` | SessionLifecycle event spec + stream + consumer spec |
| `internal/adapters/nats/natsexecution/publisher.go` | PublishSessionLifecycle + stream ensure |
| `internal/actors/scopes/execute/execute_supervisor.go` | Lifecycle publisher init + publish on close |
| `cmd/gateway/run.go` | Trigger startup in gateway lifecycle |
| `cmd/gateway/compose.go` | Return concrete VerifySessionUseCase for trigger |

---

## 5. Test Coverage

| Test | Scope |
|------|-------|
| `TestSessionLifecycleEventDeduplicationKey` | Dedup key format for closed |
| `TestSessionLifecycleEventDeduplicationKeyHalted` | Dedup key format for halted |
| `TestSessionLifecycleEventDeduplicationKeyUniqueness` | Closed vs halted keys differ |
| `TestTriggerVerifySessionNilSafe` | Nil use case and nil verify UC safety |
| `TestTriggerVerifySessionSkipsNonTerminal` | Non-terminal events skipped |
| `TestTriggerVerifySessionConstructor` | Basic construction |
| `TestStartVerificationTriggerNilVerifyUC` | Nil verify UC → nil trigger |
| `TestVerificationTriggerCloseNilSafe` | Nil trigger close safety |
| `TestTriggerVerifySessionUseCaseConstructable` | Gateway-compatible construction |

---

## 6. Known Limitations

| ID | Limitation | Severity | Mitigation |
|----|-----------|----------|------------|
| L1 | Event-driven reports not persisted to filesystem | Low | Operator runs `--save` manually when archival needed |
| L2 | Gateway must be running to consume events | Low | JetStream retains events for 7 days |
| L3 | No external alerting integration | Low | Future concern; logs available for monitoring |
| L4 | 5s settle delay is heuristic | Low | Operator can re-run for definitive result |
| L5 | No partial re-verification for skipped checks | Low | Operator re-runs manually |

---

## 7. Guard Rails Assessment

| Guard Rail | Respected |
|------------|-----------|
| No generic automation platform | Yes — trigger is scoped to verification only |
| No alerting platform | Yes — results logged, no external notifications |
| No masking of trigger/loop risks | Yes — dedup, anti-loop, fail-closed documented |
| Manual verification preserved | Yes — all existing paths unchanged |

---

## 8. Readiness for S491

S490 provides the event-driven trigger mechanism. The system is ready for end-to-end proof in S491:

- Execute binary publishes lifecycle event on session close
- Gateway consumer triggers verification automatically
- Verification runs with session-derived scope
- Results are logged with pass/fail summary
- Manual verification remains available as backup

The main gap remaining after S490 is L1 (report persistence) and L3 (alerting), neither of which blocks the end-to-end proof.
