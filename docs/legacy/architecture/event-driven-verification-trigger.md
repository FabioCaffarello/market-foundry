# Event-Driven Verification Trigger

**Stage**: S490
**Status**: Implemented
**Gap Closed**: G-OA1 (event-driven auto-trigger for verification)

## Purpose

The verification pipeline (PO-1 through PO-9) previously required manual invocation via `make po-verify`, the HTTP endpoint `GET /session/:id/verify`, or the `scripts/po-verify.sh` script. S490 adds an event-driven trigger that automatically runs verification when a session reaches a terminal state (closed or halted).

## Architecture

```
Execute Binary                         Gateway Binary
┌──────────────────┐                  ┌──────────────────────────┐
│ closeSession()   │                  │ VerificationTrigger      │
│  1. Update KV    │                  │  1. Consume event        │
│  2. Publish      │──JetStream ───→  │  2. Validate terminal    │
│     lifecycle    │   (durable)      │  3. Wait 5s (CH settle)  │
│     event        │                  │  4. Run VerifySessionUC  │
└──────────────────┘                  │  5. Log result           │
                                      └──────────────────────────┘
```

### Event Flow

1. **Execute binary** closes a session (graceful shutdown or halt)
2. Session state is persisted to NATS KV (`EXECUTION_SESSION` bucket)
3. A `SessionLifecycleEvent` is published to the `SESSION_LIFECYCLE_EVENTS` JetStream stream
4. **Gateway binary** has a durable consumer (`gateway-verification-trigger`) that picks up the event
5. The `TriggerVerifySessionUseCase` validates the event, waits 5 seconds for ClickHouse writes to settle, and runs the existing `VerifySessionUseCase`
6. Results are logged — no persistence of triggered reports (operator can re-run manually for persistence)

### NATS Infrastructure

| Component | Value |
|-----------|-------|
| Stream | `SESSION_LIFECYCLE_EVENTS` |
| Subject pattern | `execution.session.lifecycle.{status}` |
| Event type | `execution.session.v1.lifecycle` |
| Storage | File-based JetStream |
| Max age | 7 days |
| Max bytes | 16 MB |
| Consumer durable | `gateway-verification-trigger` |
| Ack policy | Explicit |
| Max deliver | 5 |

### Domain Event

```go
type SessionLifecycleEvent struct {
    SessionID  string        // "session_YYYYMMDD_HHMMSS"
    Status     SessionStatus // "closed" | "halted"
    Operator   string
    HaltReason string        // only when halted
    ClosedAt   time.Time
    VenueType  string
    DryRun     bool
    Segments   []string
}
```

## Integration Points

### Producer: Execute Binary

- `ExecuteSupervisor.closeSession()` persists session to KV, then calls `publishSessionLifecycle()`
- Publisher is initialized during supervisor start, alongside other NATS infrastructure
- Publisher failure is non-fatal: session is still persisted in KV, manual verification still works

### Consumer: Gateway Binary

- `startVerificationTrigger()` creates a background consumer during gateway startup
- Consumer calls `TriggerVerifySessionUseCase.Handle()` for each event
- Consumer start failure is non-fatal: gateway operates normally without event-driven trigger

### Verification Use Case

The trigger reuses the existing `VerifySessionUseCase` (S461) with no modifications. The same 9 PO checks run with session-derived scope (S485).

## Manual Verification Preserved

The event-driven trigger is additive. All existing manual paths remain functional:

- `make po-verify` / `scripts/po-verify.sh`
- `GET /session/:id/verify`
- `GET /session/:id/audit`
- `GET /session/batch-audit`

## Files

| File | Role |
|------|------|
| `internal/domain/execution/session_event.go` | SessionLifecycleEvent domain type |
| `internal/adapters/nats/natsexecution/registry.go` | Stream + event spec + consumer spec |
| `internal/adapters/nats/natsexecution/publisher.go` | PublishSessionLifecycle method |
| `internal/adapters/nats/natsexecution/session_lifecycle_consumer.go` | JetStream consumer |
| `internal/application/executionclient/trigger_verify_session.go` | Trigger use case |
| `internal/actors/scopes/execute/execute_supervisor.go` | Lifecycle event publishing on close |
| `cmd/gateway/verification_trigger.go` | Gateway-side consumer wiring |
| `cmd/gateway/run.go` | Trigger startup in gateway lifecycle |
| `cmd/gateway/compose.go` | VerifySessionUseCase extraction for trigger |
