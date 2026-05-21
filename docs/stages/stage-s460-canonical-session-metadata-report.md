# S460 -- Canonical Session Metadata Model and Persistence

**Status**: COMPLETE
**Date**: 2026-03-24
**Wave**: Session Intelligence and Operational Automation (S459)
**Predecessor**: S459 (charter), S456A (session metadata PARTIAL gap)
**Successor**: S461 (PO Automation and Verification Pipeline)

---

## 1. Objective

Design, implement, validate, and document a canonical session metadata model with persistence and explicit links to orders, outcomes, artifacts, and verification results.

**Governing Question Answered**: Q6 -- Does session-level metadata exist as queryable state? **YES**.

---

## 2. What Was Delivered

### Domain Model

- `internal/domain/execution/session.go` -- Canonical `Session` entity with:
  - SessionID, Operator, Status, HaltReason
  - StartedAt, ClosedAt timestamps
  - SessionConfigSnapshot (venue_type, dry_run, segments, config_file)
  - SessionActivationSnapshot (adapter, credentials, gate_status, effective)
  - SessionSegmentCounters (per-segment processed, filled, rejected, errors)
  - Artifacts map (named references to external files)
- Full validation: `Session.Validate()` with field-level constraints
- Lifecycle methods: `Close()`, `Halt()`, `Duration()`
- Session ID generator: `NewSessionID(time.Time)` -> `session_{YYYYMMDD}_{HHMMSS}`

### Persistence

- `internal/adapters/nats/natsexecution/session_kv_store.go` -- NATS KV adapter
  - Bucket: `EXECUTION_SESSION` (FileStorage, 16 MB)
  - Operations: Put, Get, List (newest-first)
  - Validation before write

### Lifecycle Integration

- `internal/actors/scopes/execute/execute_supervisor.go` -- Session lifecycle:
  - `openSession()` on supervisor start (config snapshot, activation snapshot)
  - `closeSession()` on supervisor stop (segment counters from tracker)
  - `WithOperator(string)` option for operator identity
  - Degraded mode: continues normally if KV unavailable

### Query Surface

- `internal/adapters/nats/natsexecution/registry.go` -- SessionGet, SessionList specs
- `internal/adapters/nats/natsexecution/session_gateway.go` -- NATS gateway adapter
- `internal/application/executionclient/session_contracts.go` -- SessionGetQuery/Reply, SessionListQuery/Reply
- `internal/application/executionclient/get_session.go` -- GetSessionUseCase, ListSessionsUseCase
- `internal/actors/scopes/store/query_responder_actor.go` -- handleSessionGet, handleSessionList
- `internal/interfaces/http/handlers/session.go` -- HTTP handler
- `internal/interfaces/http/routes/session.go` -- HTTP routes
- `internal/interfaces/http/routes/core.go` -- SessionFamilyDeps, DefaultRoutes wiring
- `cmd/gateway/compose.go` -- Session gateway connection and route dependency wiring

### HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/session/:id` | Retrieve session by ID |
| GET | `/session/list` | List all sessions |

### Tests

- `internal/domain/execution/session_test.go` -- 11 unit tests covering:
  - Session ID generation
  - Validation (valid, missing fields, terminal constraints)
  - Close and Halt transitions
  - Duration computation
  - Status validation and terminality
- `internal/application/execution/s460_session_metadata_test.go` -- 6 integration tests covering:
  - Required field presence
  - Lifecycle transitions (open -> closed)
  - Halt with reason capture
  - Config snapshot preservation
  - Activation snapshot capture
  - Multi-segment counter aggregation

### Documentation

- `docs/architecture/canonical-session-metadata-model-and-persistence.md`
- `docs/architecture/session-entity-fields-links-ownership-and-limitations.md`

---

## 3. Capability Assessment

| ID | Capability | Before S460 | After S460 | Grade |
|----|-----------|-------------|------------|-------|
| C3+ | Session Metadata Persistence | PARTIAL (no entity, no KV) | First-class entity in KV, queryable via HTTP | FULL |
| Q6 | Session metadata queryable? | NOT YET | YES -- `/session/:id` and `/session/list` | ANSWERED |

---

## 4. Files Changed

### New Files

| File | Purpose |
|------|---------|
| `internal/domain/execution/session.go` | Domain entity |
| `internal/domain/execution/session_test.go` | Domain tests |
| `internal/adapters/nats/natsexecution/session_kv_store.go` | KV persistence |
| `internal/adapters/nats/natsexecution/session_gateway.go` | NATS gateway adapter |
| `internal/application/executionclient/session_contracts.go` | Query contracts |
| `internal/application/executionclient/get_session.go` | Use cases |
| `internal/application/execution/s460_session_metadata_test.go` | Integration tests |
| `internal/interfaces/http/handlers/session.go` | HTTP handler |
| `internal/interfaces/http/routes/session.go` | HTTP routes |
| `docs/architecture/canonical-session-metadata-model-and-persistence.md` | Architecture doc |
| `docs/architecture/session-entity-fields-links-ownership-and-limitations.md` | Fields/links doc |

### Modified Files

| File | Change |
|------|--------|
| `internal/actors/scopes/execute/execute_supervisor.go` | Session lifecycle (open/close), WithOperator option |
| `internal/adapters/nats/natsexecution/registry.go` | SessionGet, SessionList specs |
| `internal/actors/scopes/store/query_responder_actor.go` | Session store + handlers |
| `internal/application/ports/execution.go` | SessionGateway interface |
| `internal/interfaces/http/routes/core.go` | SessionFamilyDeps, Dependencies.Session |
| `cmd/gateway/compose.go` | Session gateway connection + route wiring |

---

## 5. Guard Rails Compliance

| Guard Rail | Compliance |
|------------|------------|
| No workflow engine inflation | Session is a passive metadata record, not a workflow state machine |
| No broad domain redesign | Session is additive; no existing types were modified |
| No coupling to specific ceremony | Session captures generic operational metadata applicable to any execution session |
| No gap masking | Limitations documented explicitly (no CH persistence, no multi-binary correlation, temporal order links) |

---

## 6. Residual Gaps

| Gap | Severity | Addressed By |
|-----|----------|-------------|
| No automated PO checks linked to session | LOW | S461 (PO Automation) |
| No consolidated audit bundle | LOW | S462 (Session Audit Bundle) |
| No ClickHouse persistence for sessions | LOW | Future stage if retention requires it |
| Operator field requires explicit WithOperator call | LOW | Can be wired from env var in binary main() |

---

## 7. Test Evidence

```
=== RUN   TestNewSessionID            PASS
=== RUN   TestSessionValidate_Valid   PASS
=== RUN   TestSessionValidate_MissingFields/missing_session_id     PASS
=== RUN   TestSessionValidate_MissingFields/invalid_status         PASS
=== RUN   TestSessionValidate_MissingFields/missing_started_at     PASS
=== RUN   TestSessionValidate_MissingFields/missing_venue_type     PASS
=== RUN   TestSessionValidate_TerminalRequiresClosedAt             PASS
=== RUN   TestSessionValidate_HaltedRequiresReason                 PASS
=== RUN   TestSessionClose            PASS
=== RUN   TestSessionHalt             PASS
=== RUN   TestSessionDuration         PASS
=== RUN   TestSessionDuration_Open    PASS
=== RUN   TestValidSessionStatus      PASS
=== RUN   TestSessionStatus_IsTerminal PASS
=== RUN   TestS460_SessionEntityHasRequiredFields                  PASS
=== RUN   TestS460_SessionLifecycleTransitions                     PASS
=== RUN   TestS460_SessionHaltCapturesReason                       PASS
=== RUN   TestS460_SessionConfigSnapshotPreservesState             PASS
=== RUN   TestS460_SessionActivationSnapshotCapturesSurface        PASS
=== RUN   TestS460_SessionSegmentCountersPerSegment                PASS
```

All 17 tests pass. No regressions in existing domain tests.

---

## 8. Verdict

S460 is **COMPLETE**. Session metadata is now a canonical, persisted, queryable entity. The C3+ capability gap from S456A is closed. Q6 is formally answered: session-level metadata exists as queryable state via `/session/:id` and `/session/list`.
