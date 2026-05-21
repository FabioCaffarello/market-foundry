# S487: Batch Review and Operational Triage Surfaces

**Status**: Complete
**Date**: 2026-03-26

## Objective

Reduce the human cost of locating problems, anomalies, and relevant cases across sessions, decisions, and round-trips by introducing severity-ranked triage surfaces that build on existing batch review infrastructure (S467, S471, S482, S485).

## What Was Delivered

### Domain Layer

**New package**: `internal/domain/triage`

- `TriageSeverity` — critical / warning / info severity ranking
- `Finding` — structured triage observation (domain, signal, detail, severity)
- `SessionTriageItem`, `DecisionTriageItem`, `RoundTripTriageItem` — per-domain triage items with severity, findings, anomaly counts
- `TriageOverview` — cross-domain triage summary with per-domain severity counts and top findings
- `TriageDomainSummary` — severity distribution (total, critical, warning, info, clean)
- Classification functions: `ClassifySessionSeverity`, `ClassifyDecisionSeverity`, `ClassifyRoundTripSeverity`
- Sorting functions: `SortSessionItems`, `SortDecisionItems`, `SortRoundTripItems`
- Aggregation: `ComputeDomainSummary`

### Application Layer

**New package**: `internal/application/triageclient`

| Use Case | Wraps | Adds |
|----------|-------|------|
| `GetSessionTriageUseCase` | `BatchAuditSessionUseCase` (S467) | Severity classification, anomaly ranking, check filter, clean-item exclusion |
| `GetDecisionTriageUseCase` | `GetDecisionReviewUseCase` (S471) | Consistency violation ranking, incomplete chain detection, severity filter |
| `GetRoundTripTriageUseCase` | `GetRoundTripReviewUseCase` (S482) | Flag-count ranking, P&L/fee reliability signals, severity filter |
| `GetTriageOverviewUseCase` | All three above | Cross-domain aggregation, top-10 findings, partial-result tolerance |

**Contracts**: `SessionTriageQuery/Reply`, `DecisionTriageQuery/Reply`, `RoundTripTriageQuery/Reply`, `TriageOverviewQuery/Reply`, `TriageMeta`.

### HTTP Surface

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/analytical/triage/sessions` | GET | Sessions ranked by anomaly severity |
| `/analytical/triage/decisions` | GET | Decisions ranked by consistency violations |
| `/analytical/triage/roundtrips` | GET | Round-trips ranked by data quality flags |
| `/analytical/triage/overview` | GET | Cross-domain "what needs attention?" |

All endpoints support `severity` filter (critical, warning) and standard partition key parameters where applicable.

### Wiring

- `cmd/gateway/compose.go` — triage use cases wired from existing dependencies
- `internal/interfaces/http/routes/core.go` — `TriageFamilyDeps` added to `Dependencies`
- `internal/interfaces/http/routes/triage.go` — route registration
- `internal/interfaces/http/handlers/triage.go` — HTTP handler

### Documentation

- `docs/architecture/batch-review-and-operational-triage-surfaces.md` — design, classification rules, guard rails
- `docs/architecture/review-queues-triage-signals-operator-usage-and-limitations.md` — operator workflows, signal reference, limitations

## Test Coverage

| Package | New Tests | Description |
|---------|-----------|-------------|
| `internal/domain/triage` | 6 | Severity classification (session, decision, round-trip), sorting (session, decision, round-trip), domain summary computation |
| `internal/application/triageclient` | 5 | Anomaly ranking, check filter, severity filter, error entry handling, nil dependency |
| **Total** | **11** | |

All existing tests pass — zero regressions across `handlers`, `routes`, `pairing`, `execution`, `analyticalclient`, `executionclient`.

## Design Decisions

### D1: Projection over existing surfaces, not new queries

Triage surfaces are read-path projections over existing batch audit, decision review, and round-trip review endpoints. This avoids:
- New ClickHouse queries or tables
- Duplicate data fetching
- Separate consistency model

The cost is that triage latency includes the underlying surface's latency.

### D2: Default exclusion of clean items

Session and decision triage exclude clean (0 anomaly) items by default. This is intentional: the triage view answers "what needs attention?" not "show me everything." Operators can override with `severity=info`.

### D3: Partial-result overview

The triage overview returns whatever domain data is available rather than failing if one domain's triage is unavailable. This makes the overview useful even when session gateway is down (decision and round-trip triage still work).

### D4: Static severity classification

Severity thresholds are deterministic and hardcoded. Custom operator-defined thresholds were considered and deferred — the current rules cover the operational use cases identified.

## Acceptance Criteria Evaluation

| Criterion | Status |
|-----------|--------|
| Batch review is more practical and less manual | PASS — triage surfaces rank by severity, filter by check/severity, exclude clean items |
| Operational triage gains real utility | PASS — single-call overview, per-domain severity summaries, top findings |
| Stage consolidates value from previous waves | PASS — builds on S467, S471, S472, S476, S482, S485 without duplicating logic |
| Wave ready for gate final in S488 | PASS — all surfaces wired, tested, documented |

## Guard Rail Compliance

| Guard Rail | Compliance |
|------------|-----------|
| No dashboard/BI expansion | PASS — four triage endpoints, no visualization layer |
| No new observability platform | PASS — HTTP endpoints only, reusing existing data |
| No masking of triage/usability limitations | PASS — 8 limitations documented explicitly |
| No analytical layer redesign | PASS — pure projection layer, zero changes to existing surfaces |

## File Manifest

### New Files

| File | Lines | Purpose |
|------|-------|---------|
| `internal/domain/triage/triage.go` | ~140 | Domain types, classification, sorting |
| `internal/domain/triage/triage_test.go` | ~130 | Domain tests |
| `internal/application/triageclient/contracts.go` | ~100 | Query/reply contracts |
| `internal/application/triageclient/get_session_triage.go` | ~185 | Session triage use case |
| `internal/application/triageclient/get_decision_triage.go` | ~140 | Decision triage use case |
| `internal/application/triageclient/get_roundtrip_triage.go` | ~95 | Round-trip triage use case |
| `internal/application/triageclient/get_triage_overview.go` | ~115 | Cross-domain overview use case |
| `internal/application/triageclient/get_session_triage_test.go` | ~180 | Session triage tests |
| `internal/interfaces/http/handlers/triage.go` | ~215 | HTTP handler |
| `internal/interfaces/http/routes/triage.go` | ~95 | Route registration |
| `docs/architecture/batch-review-and-operational-triage-surfaces.md` | — | Architecture doc |
| `docs/architecture/review-queues-triage-signals-operator-usage-and-limitations.md` | — | Operator reference |

### Modified Files

| File | Change |
|------|--------|
| `cmd/gateway/compose.go` | Triage use case wiring (import + 20 lines) |
| `internal/interfaces/http/routes/core.go` | `TriageFamilyDeps` field + `Triage` route registration |
| `docs/stages/INDEX.md` | S486 + S487 entries |

## Residual Gaps

| ID | Gap | Impact | Mitigation |
|----|-----|--------|-----------|
| G-T1 | No trend analysis | Cannot detect "PO-1 fails more this week" | Manual comparison of two triage queries |
| G-T2 | No custom severity thresholds | Operators cannot tune classification | Classification rules cover known operational patterns |
| G-T3 | No triage persistence | Cannot review past triage states | Triage is computed from current data; session audit bundles persist |
| G-T4 | Overview does not signal per-domain degradation | If one domain fails, overview shows partial results without explicit warning | Each individual triage endpoint has proper error handling |
| G-T5 | Session triage bounded by BatchAuditMaxSessions (50) | Large session counts are truncated | Operators can use status filter to narrow scope |
