# Batch Review and Operational Triage Surfaces

> S487: Minimum viable triage surfaces for sessions, decisions, and round-trips.

## Problem Statement

After S485-S486, the system has deep batch query surfaces (session audit, decision review, effectiveness, round-trip pairing) but operators still face friction when triaging across sessions, decisions, and round-trips:

1. **Flat lists**: Batch audit returns all sessions equally weighted — operators must scan every entry to find problems.
2. **No severity ranking**: Decision reviews and round-trip reviews don't prioritize anomalies.
3. **No cross-domain view**: Finding "what needs attention?" requires querying 3+ endpoints and mentally correlating results.

## Design

S487 introduces four triage surfaces built as read-path projections over existing batch surfaces. No new ClickHouse tables, no new write-path changes, no new domain entities.

### Architecture

```
Existing Surfaces                    S487 Triage Surfaces
─────────────────                    ────────────────────

/session/batch-audit ──────────────► /analytical/triage/sessions
  (S467/S485)                          Ranked by anomaly severity
                                       Filtered by check/severity

/analytical/composite/               /analytical/triage/decisions
  decision/reviews ────────────────►   Ranked by consistency violations
  (S471/S472)                          Filtered by severity

/analytical/composite/               /analytical/triage/roundtrips
  pairing/review ──────────────────►   Ranked by data quality flags
  (S482)                               Filtered by severity

Session + Decision + RoundTrip ────► /analytical/triage/overview
  triage combined                      Cross-domain "what needs attention?"
```

### Domain Model

**Package**: `internal/domain/triage`

Core types:
- `TriageSeverity`: critical / warning / info — ranks how urgently an item needs attention
- `Finding`: single triage observation with domain, signal, detail, severity
- `SessionTriageItem`: session ranked by anomaly count and severity
- `DecisionTriageItem`: decision ranked by consistency violations
- `RoundTripTriageItem`: round-trip ranked by data quality flags
- `TriageOverview`: cross-domain summary with per-domain severity counts and top findings

Classification functions:
- `ClassifySessionSeverity(verdict, failedCount, warningCount)` — inconsistent/failed = critical; degraded/warnings = warning
- `ClassifyDecisionSeverity(violations, incomplete)` — violations = critical; incomplete = warning
- `ClassifyRoundTripSeverity(flagCount, pnlReliable, feeReliable)` — many flags or unreliable P&L = critical; some flags = warning

Sorting: all triage item types are sorted by severity first (critical → warning → info), then by anomaly count descending within each severity tier.

### Use Cases

**Package**: `internal/application/triageclient`

| Use Case | Input | Output | Wraps |
|----------|-------|--------|-------|
| `GetSessionTriageUseCase` | status, check, severity, limit | Ranked session items + domain summary | `BatchAuditSessionUseCase` (S467) |
| `GetDecisionTriageUseCase` | source/symbol/timeframe, severity, limit | Ranked decision items + domain summary | `GetDecisionReviewUseCase` (S471) |
| `GetRoundTripTriageUseCase` | source/symbol/timeframe, severity, limit | Ranked round-trip items + domain summary | `GetRoundTripReviewUseCase` (S482) |
| `GetTriageOverviewUseCase` | session_status, source/symbol/timeframe | Cross-domain overview with top findings | All three above |

### HTTP Endpoints

| Endpoint | Method | Query Parameters |
|----------|--------|-----------------|
| `/analytical/triage/sessions` | GET | `status`, `check`, `severity`, `limit` |
| `/analytical/triage/decisions` | GET | `source`, `symbol`, `timeframe`, `since`, `until`, `severity`, `limit` |
| `/analytical/triage/roundtrips` | GET | `source`, `symbol`, `timeframe`, `since`, `until`, `severity`, `limit` |
| `/analytical/triage/overview` | GET | `session_status`, `source`, `symbol`, `timeframe`, `since`, `until` |

### Wiring

Triage use cases are wired in `cmd/gateway/compose.go` from existing dependencies:
- Session triage ← `BatchAuditSessionUseCase` (requires session + ClickHouse)
- Decision triage ← `GetDecisionReviewUseCase` (requires ClickHouse)
- Round-trip triage ← `GetRoundTripReviewUseCase` (requires ClickHouse)
- Triage overview ← all three above

Graceful degradation: each triage surface is nil-safe. If session gateway is unavailable, session triage returns 503 but decision and round-trip triage still work. Overview reports partial results.

## Severity Classification Rules

### Session Triage

| Condition | Severity |
|-----------|----------|
| Audit error or nil bundle | Critical |
| Verdict = "inconsistent" or any check failed | Critical |
| Verdict = "degraded" or any check warned | Warning |
| Counter mismatch | Warning (additional finding) |
| All clean | Info (excluded from default triage view) |

### Decision Triage

| Condition | Severity |
|-----------|----------|
| Consistency violations > 0 | Critical |
| Incomplete chain (missing stages) | Warning |
| Clean chain | Info (excluded from default triage view) |

### Round-Trip Triage

| Condition | Severity |
|-----------|----------|
| Flag count > 2 or unreliable P&L with flags | Critical |
| Any flags or unreliable fees | Warning |
| Clean round-trip | Info (pre-filtered by flagged=true) |

## Default Behavior

- Session triage excludes clean sessions (0 anomalies) by default; use `severity=info` to see all.
- Decision triage excludes clean decisions (no violations, chain complete) by default.
- Round-trip triage uses `flagged=true` filter upstream — only flagged items appear.
- Overview always returns partial results (no single source failure blocks the response).

## Guard Rails

- No new ClickHouse tables — pure read-path projection over existing data.
- No changes to existing batch audit, decision review, or round-trip review surfaces.
- No write-path modifications.
- No dashboard or BI platform expansion.
- Triage severity classification is deterministic and auditable.
- Overview is bounded: top 10 findings, capped fetch limits per domain.
