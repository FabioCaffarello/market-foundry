# Canonical Session Metadata Model and Persistence

S460 | 2026-03-24

---

## 1. Purpose

This document defines the canonical session metadata model introduced in S460. A session is the unit of operational accountability: it captures who ran what configuration, from when to when, with what outcome.

Before S460, session metadata existed only in:
- Manual documentation (live session execution records)
- Shell script environment variables (SESSION_ID, OPERATOR_NAME)
- Scattered runtime state (ControlGate.UpdatedBy, health tracker counters)

After S460, session metadata is a **first-class domain entity** persisted in NATS KV, queryable via NATS request-reply and HTTP, and linked to the activation surface, segment counters, and configuration snapshots.

---

## 2. Domain Entity

```go
// internal/domain/execution/session.go

type Session struct {
    SessionID  string        `json:"session_id"`
    Operator   string        `json:"operator,omitempty"`
    Status     SessionStatus `json:"status"`         // open, closed, halted
    HaltReason string        `json:"halt_reason,omitempty"`

    StartedAt time.Time  `json:"started_at"`
    ClosedAt  *time.Time `json:"closed_at,omitempty"`

    Config     SessionConfigSnapshot     `json:"config"`
    Activation SessionActivationSnapshot `json:"activation"`

    SegmentCounters []SessionSegmentCounters `json:"segment_counters,omitempty"`
    Artifacts       map[string]string        `json:"artifacts,omitempty"`
}
```

### Session Status Lifecycle

```
open  -->  closed   (graceful shutdown)
open  -->  halted   (kill-switch or error)
```

Terminal statuses (`closed`, `halted`) require `ClosedAt` to be set. `halted` additionally requires `HaltReason`.

### Session ID Format

```
session_{YYYYMMDD}_{HHMMSS}
```

Generated from UTC timestamp at session creation. Example: `session_20260324_144213`.

---

## 3. Persistence

### KV Bucket

| Property | Value |
|----------|-------|
| Bucket | `EXECUTION_SESSION` |
| Storage | FileStorage |
| Max Bytes | 16 MB |
| Key | `session_id` |

### Write Path

1. **Execute binary startup** (`execute_supervisor.go:openSession`): Creates a session record with status `open`, config snapshot, and activation snapshot. Persists to KV.
2. **Execute binary shutdown** (`execute_supervisor.go:closeSession`): Updates session to `closed` or `halted` with segment counters. Persists to KV.

### Read Path

1. **Store binary** (`query_responder_actor.go`): Opens `SessionKVStore` at startup. Serves `SessionGet` and `SessionList` queries via NATS request-reply.
2. **Gateway binary** (`session_gateway.go`): Forwards session queries via NATS to the store.
3. **HTTP surface** (`/session/:id`, `/session/list`): Exposes session queries via the gateway HTTP server.

---

## 4. NATS Contract

| Subject | Type | Purpose |
|---------|------|---------|
| `execution.session.get` | Control | Get session by ID |
| `execution.session.list` | Control | List all sessions |

Both are registered in `natsexecution.Registry` and served by the store query responder.

---

## 5. HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/session/:id` | Retrieve session by ID |
| GET | `/session/list` | List all sessions (newest-first) |

Both follow the existing gateway pattern: gateway binary calls NATS request-reply to the store binary.

---

## 6. Lifecycle Integration

The `ExecuteSupervisor` manages the session lifecycle:

1. **On start**: `openSession()` creates a session record with:
   - Generated session ID
   - Operator identity (from `WithOperator` option)
   - Config snapshot (venue type, dry_run, enabled segments)
   - Activation snapshot (adapter, credentials, gate, effective mode)
   - Status: `open`

2. **On stop**: `closeSession("")` updates the session with:
   - Segment counters from the venue-adapter tracker
   - Status: `closed` (or `halted` if reason provided)
   - Closure timestamp

### Degraded Mode

If the session KV store is unavailable:
- Session metadata is still created in-memory
- Writes are skipped with a warning log
- The execute binary continues to operate normally

---

## 7. What Is NOT in This Model

| Excluded | Reason |
|----------|--------|
| Order-level links (order IDs) | Orders are already queryable via lifecycle endpoints; session-to-order join is done at query time via timestamp window |
| ClickHouse persistence | Session records are small and bounded; KV is sufficient. CH persistence is a future concern if session history exceeds KV retention |
| Automated session orchestration | Session lifecycle is operator-driven; automation is out of scope (NG9 in charter) |
| Multi-binary session correlation | Session is execute-owned; derive/store/writer binaries do not participate in session lifecycle |
| Session workflow engine | This is a metadata record, not a workflow state machine |
