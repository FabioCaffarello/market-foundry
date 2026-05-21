# Automated Operational Flow — Report Contents, Coverage, and Limitations

**Stage**: S491
**Status**: COMPLETE
**Date**: 2026-03-26
**Companion**: [end-to-end-automation-proof-and-unified-operational-report-artifact.md](end-to-end-automation-proof-and-unified-operational-report-artifact.md)

---

## 1. Purpose

This document specifies what the unified operational report contains, what
it covers, what it does not cover, and the known limitations of the
automated operational flow.

---

## 2. Report Contents

### 2.1 Section: Verification

| Field | Source | Description |
|-------|--------|-------------|
| `all_passed` | Derived | True when no PO check has `fail` verdict |
| `summary` | `POVerificationReport.Summary` | Pass/fail/warn/skip/manual counts |
| `duration_ms` | `POVerificationReport.DurationMs` | Time to run all 9 checks |
| `checks[]` | `POVerificationReport.Checks` | Per-check ID, name, verdict, detail, evidence |

**Data source**: `VerifySessionUseCase` → NATS KV (gate, session), ClickHouse (intents, fills, lifecycle)

### 2.2 Section: Audit

| Field | Source | Description |
|-------|--------|-------------|
| `session_status` | `SessionAuditBundle.Session.Status` | Terminal status (closed/halted) |
| `operator` | `SessionAuditBundle.Session.Operator` | Who ran the session |
| `order_activity` | `SessionAuditBundle.OrderActivity` | Intent/fill/rejection/error counts |
| `fee_summary` | `SessionAuditBundle.FeeSummary` | Fill fee coverage ratio |
| `consistency` | `SessionAuditBundle.Consistency` | Cross-surface consistency verdict |
| `check_index` | `SessionAuditBundle.CheckIndex` | Per-check verdict map for quick scanning |

**Data source**: `AuditSessionUseCase` → NATS KV (session, lifecycle), ClickHouse (fills)

### 2.3 Section: Operational State

| Field | Source | Description |
|-------|--------|-------------|
| `gate_status` | `OperationalState.Gate.Status` | Current execution gate (halted/active) |
| `gate_reason` | `OperationalState.Gate.Reason` | Why the gate is in its current state |
| `available_surfaces` | `OperationalState.Surfaces` | Which HTTP endpoint families are wired |

**Data source**: `GetOperationalStateUseCase` → NATS KV (gate), static composition

### 2.4 Section: Triage

| Field | Source | Description |
|-------|--------|-------------|
| `total_anomalies` | `TriageOverview.TotalAnomalies` | Sum of anomalies across all domains |
| `session_critical` | `TriageOverview.SessionSummary.Critical` | Sessions needing immediate attention |
| `session_warning` | `TriageOverview.SessionSummary.Warning` | Sessions with minor issues |
| `decision_critical` | `TriageOverview.DecisionSummary.Critical` | Decisions with violations |
| `decision_warning` | `TriageOverview.DecisionSummary.Warning` | Decisions with warnings |
| `round_trip_critical` | `TriageOverview.RoundTripSummary.Critical` | Round-trips with data quality issues |
| `round_trip_warning` | `TriageOverview.RoundTripSummary.Warning` | Round-trips with minor flags |
| `top_findings` | `TriageOverview.TopFindings[].Detail` | Human-readable top anomaly descriptions |

**Data source**: `GetTriageOverviewUseCase` → batch audit (NATS KV + CH), decision review (CH), round-trip review (CH)

---

## 3. Coverage Matrix

### 3.1 What the Report Covers

| Dimension | Covered | How |
|-----------|---------|-----|
| PO verification (9 checks) | Yes | Verification section |
| Session metadata | Yes | Audit section (status, operator) |
| Order activity counters | Yes | Audit section |
| Fee coverage | Yes | Audit section |
| Cross-surface consistency | Yes | Audit section |
| Execution gate state | Yes | Operational state section |
| Surface availability | Yes | Operational state section |
| Session-level anomalies | Yes | Triage section |
| Decision-level anomalies | Yes | Triage section |
| Round-trip anomalies | Yes | Triage section |
| Top findings (human-readable) | Yes | Triage section |

### 3.2 What the Report Does NOT Cover

| Dimension | Why | Where to Find It |
|-----------|-----|-------------------|
| Full lifecycle history per partition | Too verbose for summary artifact | `GET /session/:id/audit` |
| Individual decision review bundles | Per-decision detail, not session-level | `GET /analytical/composite/decision/review` |
| Individual round-trip pairings | Per-round-trip detail | `GET /analytical/composite/pairing/review` |
| Effectiveness cohort summaries | Aggregation beyond single session | `GET /analytical/composite/decision/effectiveness/summary` |
| Filesystem backup verification (PO-2) | Requires local filesystem access | `scripts/po-verify.sh --save` |
| Prometheus metrics | Out of scope (S492 candidate) | Future gauge extensions |
| Temporal trend analysis | Out of scope (S492 candidate) | Not yet implemented |
| Historical report comparison | Not yet implemented | Future consideration |
| External alerting/notification | Out of scope (guard rail GR-5) | Not planned |

---

## 4. Automation Flow Coverage

### 4.1 What Is Automated

| Flow | Trigger | Automated |
|------|---------|-----------|
| Session close → verification | JetStream lifecycle event | Yes (S490) |
| Session halt → verification | JetStream lifecycle event | Yes (S490) |
| Verification → unified report | TriggerVerifySessionUseCase | Yes (S491) |
| Report → structured log | TriggerVerifySessionUseCase | Yes (S491) |
| On-demand report | HTTP `GET /session/:id/report` | Yes (S491) |

### 4.2 What Requires Operator Action

| Flow | Why | Mitigation |
|------|-----|------------|
| Report archival to filesystem | JetStream trigger has no filesystem access | `scripts/po-verify.sh --save` or `curl ... > file` |
| Backup verification (PO-2) | Requires local path check | `scripts/po-verify.sh` |
| Remediation of failures | Operational judgment required | Report verdict guides priority |
| Report comparison across sessions | No historical report store | Operator saves and compares manually |

---

## 5. Limitations

| ID | Limitation | Severity | Mitigation |
|----|-----------|----------|------------|
| L1 | Unified report not persisted to filesystem by auto-trigger | Low | Operator uses `--save` or `curl` for archival |
| L2 | Triage section requires ClickHouse for decision/round-trip data | Low | Section becomes a gap when CH unavailable |
| L3 | Audit section requires both NATS KV and ClickHouse | Low | Section becomes a gap; verification still runs |
| L4 | 5s ClickHouse settle delay is heuristic | Low | Operator can re-run via HTTP for definitive result |
| L5 | Report does not include effectiveness cohort aggregation | Low | Available via separate analytical endpoint |
| L6 | No external notification on report generation | Low | Logs are the notification surface; future alerting is S492+ |
| L7 | Triage overview uses default query scope (not session-derived) | Low | Triage captures system-wide state at report time |

---

## 6. Consistency with Existing Surfaces

The unified report composes data from existing use cases without duplicating
logic. Each section delegates to its authoritative source:

| Section | Authoritative Use Case | HTTP Equivalent |
|---------|----------------------|-----------------|
| Verification | `VerifySessionUseCase` | `GET /session/:id/verify` |
| Audit | `AuditSessionUseCase` | `GET /session/:id/audit` |
| Operational State | `GetOperationalStateUseCase` | `GET /monitoring/state` |
| Triage | `GetTriageOverviewUseCase` | `GET /analytical/triage/overview` |

The report adds no new data sources or transformations. It is purely a
composition layer that makes the operator experience more ergonomic.
