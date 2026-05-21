# Operational History & Explainability — Capabilities, Questions, and Non-Goals

**Wave**: Operational History & Explainability (S452A–S452E)
**Date**: 2026-03-24
**Companion**: [Wave Charter and Scope Freeze](operational-history-and-explainability-wave-charter-and-scope-freeze.md)

---

## 1. Capabilities to Deliver

### C1 — Persistence Completeness Invariant

**Statement**: Every execution intent that reaches a KV bucket must have a corresponding ClickHouse record within the writer's flush window.

**Current state**: Violated — S449 produced 24 venue fills but only 12 ClickHouse records (F3).

**Deliverable**:
- Root cause analysis of F3 gap with evidence.
- Fix or documented explanation for the 50% drop.
- Automated persistence completeness check: enumerate KV keys, verify ClickHouse coverage.
- Regression guard: test that ensures writer does not silently drop events.

### C2 — Type and Status Disambiguation

**Statement**: Query results must unambiguously distinguish live execution from paper/dry-run execution, and must reflect the correct lifecycle status (submitted → accepted → filled, not stuck at submitted).

**Current state**: Violated — F4 shows `type=paper_order` for live adapter output; F5 shows `status=submitted` for records that should show `accepted` or `filled`.

**Deliverable**:
- Fix type assignment to reflect actual execution mode (live, paper, dry-run).
- Fix status propagation to reflect full lifecycle (not just derive-side status).
- Verify fixes against S449 historical data.

### C3 — Session Metadata Persistence

**Statement**: Each operational session must be recorded as a first-class entity with queryable metadata.

**Current state**: Absent — S449 session details exist only in human-authored documents.

**Deliverable**:
- Session metadata model: `session_id`, `started_at`, `stopped_at`, `config_hash`, `config_file`, `operator`, `segment`, `environment`, `outcome_summary`.
- KV bucket for session metadata (write at session start, update at session stop).
- Query route: `execution.query.session.list` → returns all sessions.
- Query route: `execution.query.session.detail` → returns session metadata + summary stats.

### C4 — Order Narrative Query (Full Lifecycle Trace)

**Statement**: Given an execution ID or correlation ID, the system must reconstruct the full lifecycle narrative from signal through fill/rejection to persistence.

**Current state**: Partial — correlation_id exists in ClickHouse across tables, but no single query joins the chain. KV stores latest-only with no history.

**Deliverable**:
- ClickHouse query that joins `signals` → `decisions` → `strategies` → `risk_assessments` → `executions` on `correlation_id`.
- Response model: ordered list of lifecycle events with timestamps, status transitions, and persistence checkpoints.
- Edge case handling: intents with no fill (noop), intents with rejection, intents with partial fill.

### C5 — List Query Ergonomics

**Statement**: Operational review queries must support filtering by time range, status, segment, execution mode, and must provide summary aggregations.

**Current state**: `LifecycleListQuery` returns all keys with status per surface and propagation flag, but has no filtering or aggregation.

**Deliverable**:
- Extend `LifecycleListQuery` (or add `LifecycleFilterQuery`) with: time range, status filter, segment filter, execution mode filter.
- Add summary response: count by status, count by segment, total intents vs fills vs rejections.
- Session-window query: "all events between T1 and T2" using session metadata timestamps.
- ClickHouse query equivalents for all filters.

### C6 — KV-to-ClickHouse Consistency Audit

**Statement**: The system must detect and report divergence between KV state and ClickHouse records.

**Current state**: No automated check exists. F3 was discovered manually by counting records.

**Deliverable**:
- Consistency check: enumerate all KV keys across all execution buckets, query ClickHouse for matching records.
- Report: missing records, stale records (KV newer than ClickHouse), status mismatches.
- Integration into post-session verification protocol (PO check).
- Runnable as script or query route.

### C7 — Post-Session Verification Automation

**Statement**: All 9 PO checks from S447 must be executable as automated validations, not manual document checks.

**Current state**: S447 defines PO-1 through PO-9 but S449 only executed 2 of 9 (F7).

**Deliverable**:
- Script or test harness that executes all PO checks against current system state.
- Each check reports PASS/FAIL with evidence.
- PO checks include the new consistency audit (C6) and session metadata verification (C3).
- Runnable via `make` target or dedicated script.

---

## 2. Governing Questions

These questions guide scope decisions during the wave. Each stage must advance at least one question toward a definitive answer.

### Q1 — Why did 50% of execution events fail to reach ClickHouse?

**Context**: S449 produced 24 venue fills but only 12 ClickHouse records. The writer pipeline (S385) was tested and verified. The gap could be: writer batching, consumer lag, event filtering, dual-stream interference (paper_order + venue_order events competing), or ClickHouse write failure.

**Answered by**: S452B (root cause investigation).

### Q2 — Can the read surfaces distinguish live execution from paper execution without ambiguity?

**Context**: F4 shows `type=paper_order` for live adapter output. F5 shows `status=submitted` instead of `accepted/filled`. If queries cannot distinguish execution mode, operational review is unreliable.

**Answered by**: S452B (type/status disambiguation).

### Q3 — Can an operator reconstruct the full lifecycle of any execution intent from queryable data alone?

**Context**: Correlation IDs exist across ClickHouse tables but no single query joins the chain. KV stores latest-only state with no history. The "full story of order X" is currently unqueryable.

**Answered by**: S452D (narrative query).

### Q4 — Can the system detect when KV and ClickHouse diverge?

**Context**: F3 proved divergence happens silently. Without automated detection, every session's data integrity is uncertain.

**Answered by**: S452E (consistency audit).

### Q5 — Can post-session verification run without manual intervention?

**Context**: S447 defines 9 checks. S449 executed 2. The gap is not protocol design but automation — the checks require manual ClickHouse queries and visual inspection.

**Answered by**: S452E (PO automation).

### Q6 — Does session-level metadata exist as queryable system state?

**Context**: S449 session details (start, stop, config, operator, outcome) exist only in markdown documents. No query can enumerate past sessions or retrieve session parameters.

**Answered by**: S452D (session metadata).

---

## 3. Non-Goals

### NG1 — Broad Dashboards or Visualization UI

**What it means**: No Grafana boards, no web UI, no charting. This wave delivers queryable data surfaces (NATS query routes, ClickHouse SQL, scripts). Presentation is a separate concern.

**Why excluded**: Dashboards are a consumer of data, not a producer. The wave must first ensure the data is correct, complete, and queryable. Building dashboards on incomplete data wastes effort.

### NG2 — New Observability Platform

**What it means**: No Prometheus, no OpenTelemetry, no metrics pipeline. Existing NATS request-reply and ClickHouse SQL are the query surfaces.

**Why excluded**: The system already has two persistence layers (KV and ClickHouse) and a request-reply query protocol. Adding a third observability layer increases complexity without solving the core problem (data completeness and explainability).

### NG3 — OMS Expansion

**What it means**: No new order types, no new lifecycle states, no new domain events beyond what exists. The canonical order model (S383) and lifecycle invariants (S384) are stable.

**Why excluded**: The OMS Foundation wave (S382–S388) delivered a complete and tested order model. This wave strengthens the read side of that model, not the write side.

### NG4 — Multi-Exchange Support

**What it means**: All work is Binance-only. No new exchange adapters, no exchange abstraction changes.

**Why excluded**: Multi-exchange is a future wave concern. This wave operates entirely within the existing Binance Spot/Futures scope.

### NG5 — Mainnet/Live Expansion or New Sessions

**What it means**: No new API keys, no new live sessions, no credential rotation, no scope expansion. This wave is fully offline — it analyzes and queries existing data.

**Why excluded**: Live session work belongs to the parallel Live Session Stabilization track (S452–S455). This wave must complete independently of whether the next live session happens.

### NG6 — Structural Redesign of Storage or Runtime

**What it means**: No new NATS subjects for existing events, no ClickHouse table redesign, no KV bucket restructuring, no runtime boot changes. Session metadata may add one new KV bucket — this is the only exception.

**Why excluded**: The storage and runtime architecture (KV + ClickHouse + NATS streams) is proven through S448. This wave improves query surfaces on top of existing storage, not the storage itself.

### NG7 — Real-Time Streaming or Event-Driven Dashboards

**What it means**: No WebSocket feeds, no Server-Sent Events, no live-updating views. All queries are request-reply (pull model).

**Why excluded**: Real-time monitoring is a production operations concern. This wave targets post-hoc analysis and session review.

### NG8 — Automated Alerting or Paging

**What it means**: No PagerDuty, no Slack alerts, no threshold-based notifications.

**Why excluded**: Alerting requires defining thresholds and response procedures. This wave establishes the data foundation that future alerting could consume.

### NG9 — Fee/Commission Model Changes

**What it means**: No changes to fee normalization (S428), commission calculation, or cost basis logic.

**Why excluded**: Fee normalization is stable. The wave may query fee data for verification but does not modify how fees are calculated or stored.

### NG10 — External API Endpoints

**What it means**: No HTTP endpoints, no REST API, no external-facing interfaces. All query surfaces are internal (NATS request-reply, ClickHouse SQL, CLI scripts).

**Why excluded**: External interfaces require authentication, rate limiting, and security review. This wave targets internal operational review.

---

## 4. Capability-to-Stage Mapping

| Capability | S452B | S452C | S452D | S452E |
|------------|-------|-------|-------|-------|
| C1 — Persistence Completeness | **PRIMARY** | | | verify |
| C2 — Type/Status Disambiguation | **PRIMARY** | | | verify |
| C3 — Session Metadata | | | **PRIMARY** | verify |
| C4 — Order Narrative Query | | | **PRIMARY** | verify |
| C5 — List Query Ergonomics | | **PRIMARY** | | verify |
| C6 — Consistency Audit | | | | **PRIMARY** |
| C7 — PO Automation | | | | **PRIMARY** |

---

## 5. Question-to-Stage Mapping

| Question | S452B | S452C | S452D | S452E |
|----------|-------|-------|-------|-------|
| Q1 — Why 50% persistence gap? | **ANSWER** | | | |
| Q2 — Live vs paper distinction? | **ANSWER** | | | |
| Q3 — Full lifecycle narrative? | | | **ANSWER** | |
| Q4 — Divergence detection? | | | | **ANSWER** |
| Q5 — Automated PO checks? | | | | **ANSWER** |
| Q6 — Session metadata queryable? | | | **ANSWER** | |

---

## 6. Acceptance Criteria per Stage

### S452B — Historical Execution Read Model
- [ ] F3 root cause documented with evidence (writer logs, event traces, ClickHouse counts).
- [ ] Persistence completeness check implemented and passing for new data.
- [ ] Type field correctly reflects execution mode (live, paper, dry-run).
- [ ] Status field reflects full lifecycle (not stuck at submitted).
- [ ] Historical S449 data corrected or gap explained with documented limitation.

### S452C — Operational List Queries and Retrieval
- [ ] LifecycleListQuery supports time range, status, segment, and mode filters.
- [ ] Summary aggregation returns correct counts for known test data.
- [ ] Session-window query returns all events for a given time range.
- [ ] ClickHouse equivalents exist for all filter dimensions.

### S452D — Session Explainability Surface
- [ ] Session metadata model defined and persisted to KV.
- [ ] Session list query returns at least the S449 session (retroactively populated).
- [ ] Order narrative query reconstructs full lifecycle for a known execution ID.
- [ ] Correlation chain query joins across ClickHouse tables correctly.

### S452E — Evidence Gate
- [ ] KV-to-ClickHouse consistency check reports zero divergence on clean state.
- [ ] All 9 PO checks from S447 are automated and pass.
- [ ] All capabilities C1–C7 verified with evidence.
- [ ] All questions Q1–Q6 answered with documented findings.
- [ ] Wave declared CLOSED.
