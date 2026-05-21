# Stage S486 -- Monitoring and Operational State Surfaces Report

**Stage**: S486
**Type**: Implementation
**Status**: COMPLETE
**Date**: 2026-03-26
**Predecessor**: S485 (Verification Scope Hardening)

---

## 1. Executive Summary

S486 delivers minimal monitoring and operational state surfaces that make the system's current state legible in a single query. Before this stage, an operator needed to call `/session/list`, `/execution/control`, and mentally track which endpoint families were available. Now, `GET /monitoring/state` returns a consolidated snapshot of the latest session, gate status, and surface availability with graceful degradation when any source is unavailable.

This stage is intentionally narrow: it adds one endpoint and its supporting domain/application/interface layers, without inflating toward dashboards, alerting, or observability platforms.

---

## 2. Capabilities Delivered

### C-MON1: Consolidated Operational State Endpoint

New endpoint: `GET /monitoring/state`

Returns:
- Latest session summary (ID, status, operator, config, segment counters)
- Gate status (active/halted, reason, last update)
- Surface availability (which endpoint families are wired)
- Observation timestamp

Implemented across:
- Domain: `internal/domain/monitoring/state.go`
- Use case: `internal/application/monitoringclient/get_operational_state.go`
- Contracts: `internal/application/monitoringclient/contracts.go`
- Handler: `internal/interfaces/http/handlers/monitoring.go`
- Routes: `internal/interfaces/http/routes/monitoring.go`
- Wiring: `cmd/gateway/compose.go`
- Route registration: `internal/interfaces/http/routes/core.go`

### C-MON2: Surface Availability Registry

`SurfaceAvailability` struct captures which of 9 endpoint families (evidence, signal, decision, strategy, risk, execution, session, analytical, activation) are wired at gateway composition time.

- `DegradedFamilies()` method returns names of unavailable families.
- Static capture at startup — no per-query probing.
- Available in monitoring response and usable programmatically.

### C-MON3: Graceful Degradation

The monitoring endpoint never fails with 5xx:
- Session gateway unavailable → `session: null`
- Execution control gateway unavailable → `gate: null`
- Both unavailable → still returns `surfaces` and `observed_at`

This ensures monitoring itself is never a point of failure.

### C-MON4: Session Summary Projection

`NewSessionSummary()` creates a lightweight monitoring view from a full `execution.Session`:
- Strips audit-only fields (artifacts, activation snapshot)
- Computes human-readable duration for terminal sessions
- Preserves per-segment counters for throughput visibility

---

## 3. Test Evidence

### Domain Tests (`internal/domain/monitoring/state_test.go`)

| Test | What it validates |
|---|---|
| `TestNewSessionSummary` | Closed session maps correctly with duration and counters |
| `TestNewSessionSummary_OpenSession` | Open session has no duration or closed_at |
| `TestSurfaceAvailability_DegradedFamilies/all_available` | Zero degraded when all surfaces wired |
| `TestSurfaceAvailability_DegradedFamilies/none_available` | All 9 families reported degraded |
| `TestSurfaceAvailability_DegradedFamilies/partial` | Correct count with mixed availability |

### Use Case Tests (`internal/application/monitoringclient/get_operational_state_test.go`)

| Test | What it validates |
|---|---|
| `TestGetOperationalState_FullWiring` | Session, gate, and surfaces all populated correctly |
| `TestGetOperationalState_NoSessions` | Empty session list → null session |
| `TestGetOperationalState_NilDependencies` | Nil lister and reader → null session and gate, no error |
| `TestGetOperationalState_SessionListerError` | Lister error → null session, gate still populated |
| `TestGetOperationalState_GateHalted` | Halted gate correctly surfaces status and reason |

All 8 tests pass.

---

## 4. Acceptance Criteria Assessment

| Criterion | Met? | Evidence |
|---|---|---|
| System is more monitorable without inflating scope | YES | One endpoint, one use case, KV-only reads |
| Relevant operational states are more explicit | YES | Session, gate, and surface availability consolidated |
| Legibility and operational accompaniment improved | YES | Single call replaces 3+ previous calls |
| Base ready for S487 batch review and triage | YES | Surface availability registry enables programmatic triage |

---

## 5. Guard Rail Compliance

| Guard rail | Compliance |
|---|---|
| No dashboards | No UI, visualization, or dashboard infrastructure added |
| No observability platform | No tracing, distributed logging, or APM integration |
| No alerting platform | No thresholds, rules, or push notifications |
| No redundant surfaces | Monitoring endpoint aggregates existing data, does not duplicate storage |
| No masking of limitations | Limitations documented in architecture doc |

---

## 6. Files Changed

### New Files

| File | Purpose |
|---|---|
| `internal/domain/monitoring/state.go` | Domain types: OperationalState, SessionSummary, GateSummary, SurfaceAvailability |
| `internal/domain/monitoring/state_test.go` | Domain tests (3 test functions) |
| `internal/application/monitoringclient/contracts.go` | Query/reply contracts |
| `internal/application/monitoringclient/get_operational_state.go` | Use case implementation |
| `internal/application/monitoringclient/get_operational_state_test.go` | Use case tests (5 test functions) |
| `internal/interfaces/http/handlers/monitoring.go` | HTTP handler |
| `internal/interfaces/http/routes/monitoring.go` | Route registration and MonitoringFamilyDeps |
| `docs/architecture/monitoring-and-operational-state-surfaces.md` | Surface definition and semantics |
| `docs/architecture/operational-states-monitoring-semantics-coverage-and-limitations.md` | Coverage matrix and limitations |

### Modified Files

| File | Change |
|---|---|
| `internal/interfaces/http/routes/core.go` | Added `Monitoring` field to Dependencies; registered monitoring routes in DefaultRoutes |
| `cmd/gateway/compose.go` | Wired monitoring use case with surface availability, session lister, and gate reader |

---

## 7. Limitations and Known Gaps

1. **Static surface availability** does not reflect runtime connectivity changes.
2. **Session counters** for open sessions may be zero (populated at close).
3. **No cross-binary health aggregation** — each binary has its own `/statusz`.
4. **No ClickHouse probing** in monitoring endpoint — analytical surface availability is startup-only.
5. **No effectiveness/pairing** in monitoring snapshot — different latency profile.

These are documented design decisions, not oversights.

---

## 8. Promoted Architecture Documents

- [`docs/architecture/monitoring-and-operational-state-surfaces.md`](../architecture/monitoring-and-operational-state-surfaces.md) — Surface definition
- [`docs/architecture/operational-states-monitoring-semantics-coverage-and-limitations.md`](../architecture/operational-states-monitoring-semantics-coverage-and-limitations.md) — Coverage and limitations

---

## 9. Next Stage Readiness

S486 prepares the system for:
- **S487**: Batch review and triage surfaces — can leverage `SurfaceAvailability` to understand which analytical surfaces are available for batch queries.
- Future monitoring extensions can add fields to `OperationalState` without changing the endpoint contract.
