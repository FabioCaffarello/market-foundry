# Operational Automation and Monitoring Hardening — Capabilities, Questions, and Non-Goals

**Wave**: Operational Automation and Monitoring Hardening (S484–S489)
**Date**: 2026-03-26
**Charter**: [operational-automation-and-monitoring-hardening-wave-charter-and-scope-freeze.md](operational-automation-and-monitoring-hardening-wave-charter-and-scope-freeze.md)

---

## 1. Purpose

This document is the capability-level reference for the Operational Automation and Monitoring Hardening wave. It maps each capability to its governing question, defines what evidence is needed, and lists all non-goals with rationale.

---

## 2. Governing Questions

| ID | Question | Evidence required |
|----|----------|------------------|
| Q-OA1 | Does post-session verification run automatically without operator intervention? | Demonstration that session reaching terminal state triggers PO verification and persists a structured report without manual invocation. |
| Q-OA2 | Can an operator assess overall operational health from a single surface? | Operational state endpoint returning aggregated health across sessions with reconciliation rates, resolved rates, and check pass rates. |
| Q-OA3 | Does batch triage surface which sessions need attention first? | Batch triage response with ranked anomaly indicators showing degraded or failed sessions before healthy ones. |
| Q-OA4 | Are operational reports machine-readable and archivable? | JSON schema documentation for operational report; proof of persistence to `backups/sessions/<session_id>/`; proof of retrieval. |
| Q-OA5 | Does the automated workflow function end-to-end without manual steps? | Integration test or proof demonstrating session halt → auto-verification → state surface update → triage visibility. |

---

## 3. Capabilities Matrix

### Block 1 — Automated Post-Session Verification and Operational Report (S485)

| ID | Capability | Description | Evidence |
|----|-----------|------------|---------|
| C-OA1 | Automated PO verification on session halt | When a session transitions to a terminal state (halted, completed), PO verification runs automatically. No operator invocation required. | Test showing verification triggers on session halt event. Persisted report found in `backups/sessions/`. |
| C-OA2 | Structured operational report | Single JSON artifact per session combining: PO verification results, effectiveness summary (win/loss/breakeven/unresolved counts and rates), pairing summary (resolved rate, unmatched count, reconciliation flag counts). | Schema documented. Report contains all three sections. Test validates completeness. |
| C-OA3 | Operational report persistence | Reports persist to `backups/sessions/<session_id>/operational-report.json` and are retrievable via HTTP (`GET /session/:id/operational-report`). | File written on disk after auto-verification. HTTP endpoint returns the persisted report. |

### Block 2 — Operational State and Monitoring Surfaces (S486)

| ID | Capability | Description | Evidence |
|----|-----------|------------|---------|
| C-OA4 | Aggregated operational state endpoint | `GET /operational/state` returns a summary across recent sessions: total sessions, check pass/fail/warn distribution, average resolved rate, reconciliation flag frequency, consistency verdict distribution. | Endpoint returns valid JSON with all fields populated from real session data. |
| C-OA5 | Session-level reconciliation and resolved-rate summaries | Each session's operational report includes per-session resolved rate and reconciliation flag breakdown. The state endpoint aggregates these. | Per-session numbers match individual audit results. Aggregation is mathematically correct. |
| C-OA6 | Prometheus gauge extensions | New gauges: `marketfoundry_sessions_total{status}`, `marketfoundry_verification_checks{verdict}`, `marketfoundry_resolved_rate`, `marketfoundry_reconciliation_flags{flag}`. | `/metrics` exports new gauges. Values update after session completion. |

### Block 3 — Batch Review and Operational Triage (S487)

| ID | Capability | Description | Evidence |
|----|-----------|------------|---------|
| C-OA7 | Triage-oriented batch audit with anomaly ranking | `GET /operational/triage` returns sessions ranked by urgency: failed checks first, degraded consistency second, anomalous reconciliation rates third, healthy last. | Response order verified against known session states. Degraded sessions appear before healthy ones. |
| C-OA8 | Cross-session reconciliation flag trend | Triage response includes a `reconciliation_trend` section showing flag frequency across the batch window. Operator sees if `fee_gap` or `simulated` flags are increasing. | Trend section present. Values computed from actual session data across batch window. |
| C-OA9 | Effectiveness drift signals | Triage response includes an `effectiveness_drift` section comparing recent win/loss/unresolved distribution against the batch average. Sessions with anomalous distributions are flagged. | Drift section present. Anomalous sessions correctly identified by deviation from batch mean. |

### Block 4 — Integration Proof and Operational Soak (S488)

| ID | Capability | Description | Evidence |
|----|-----------|------------|---------|
| C-OA10 | End-to-end integration proof | Full automated workflow demonstrated: session runs → reaches halt → PO verification fires → operational report persisted → operational state endpoint reflects new data → triage endpoint ranks sessions correctly. No manual steps. | Proof log showing complete chain. State and triage endpoints return data reflecting the auto-verified session. |

---

## 4. Non-Goals (Complete Reference)

### 4.1 Infrastructure and platform

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-OA1 | Full observability platform (ELK, Loki, Grafana) | Infrastructure decision requiring separate evaluation. This wave produces JSON data surfaces; visualization is consumer-side. |
| NG-OA2 | Push-based alerting (Alertmanager, PagerDuty, Slack) | Requires external service integration. Operational state surface provides poll-friendly data. Push delivery is an infrastructure concern. |
| NG-OA11 | BI platform or data warehouse integration | Not an application concern. Data surfaces are JSON-over-HTTP; downstream consumption is infrastructure. |
| NG-OA12 | Dashboard or visualization layer | Application layer provides structured data. Dashboards are consumer-side tooling. |

### 4.2 Domain expansion

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-OA3 | OMS expansion (position tracking, portfolio engine) | Separate domain with distinct charter requirements. Not a monitoring concern. |
| NG-OA4 | Multi-exchange support | No new exchange connectivity. Current wave monitors existing Binance Spot/Futures segments. |
| NG-OA5 | New strategy families or strategy domain expansion | Consolidation wave. No new strategy types, indicators, or decision models. |
| NG-OA8 | ML-based scoring or predictive analytics | No machine learning infrastructure. Anomaly signals use statistical deviation, not trained models. |

### 4.3 Measurement expansion

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-OA6 | Cross-session position continuity | Separate wave candidate (G-RT4). This wave monitors sessions independently. |
| NG-OA7 | Futures fee recovery write-path changes | Separate wave candidate (G-RT1). This wave works with existing data, including fee_gap flags. |
| NG-OA13 | Historical trend analysis across weeks/months | Scope is session-level and recent-batch. Long-horizon trend analysis requires data retention policy decisions. |

### 4.4 Architecture

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG-OA9 | Real-time streaming or WebSocket surfaces | Read-path polling is sufficient for operational monitoring at current session frequency. |
| NG-OA10 | Large structural refactoring | This wave adds surfaces and automation. It does not reorganize existing code. |
| NG-OA14 | New ClickHouse tables or schemas | Operates on existing read models and query surfaces. |
| NG-OA15 | Write-path changes of any kind | Strictly read-path and automation-layer additions. |

---

## 5. Dependency Map

```
Existing Assets (pre-wave)
├── VerifySessionUseCase (S461)
├── AuditSessionUseCase (S462)
├── BatchAuditSessionUseCase (S467)
├── GetEffectivenessUseCase (S476)
├── GetPairingUseCase (S481)
├── GetRoundTripReviewUseCase (S482)
├── Prometheus /metrics (S354)
└── backups/sessions/ persistence (S461)

S485: Automated Verification + Operational Report
├── Consumes: VerifySessionUseCase, GetEffectivenessUseCase, GetPairingUseCase
├── Produces: OperationalReport, auto-trigger mechanism
└── Persists: backups/sessions/<id>/operational-report.json

S486: Operational State Surfaces
├── Consumes: OperationalReport (S485), existing session list
├── Produces: /operational/state endpoint, Prometheus gauges
└── Aggregates: cross-session health summary

S487: Batch Triage
├── Consumes: /operational/state (S486), batch audit, effectiveness batch
├── Produces: /operational/triage endpoint
└── Computes: anomaly ranking, reconciliation trend, effectiveness drift

S488: Integration Proof
├── Consumes: All of S485–S487
├── Produces: End-to-end proof log
└── Validates: Full automated workflow

S489: Evidence Gate
├── Consumes: All evidence from S485–S488
├── Produces: Evidence matrix, residual gaps, wave verdict
└── Decides: PASS / CONDITIONAL PASS / FAIL
```

---

## 6. Risk Register

| Risk | Severity | Mitigation |
|------|----------|-----------|
| Auto-verification adds latency to session halt path | LOW | Run verification asynchronously after halt confirmation; do not block halt. |
| Operational report schema changes across stages | LOW | Define schema in S485; subsequent stages extend, not modify. |
| Prometheus cardinality explosion from new gauges | LOW | Bound label values to known enums (session status, check verdict, flag name). |
| Batch triage performance on large session counts | MEDIUM | Maintain 50-session cap from existing batch audit; add time-window filter. |
| Integration proof requires running session infrastructure | LOW | Can use existing dry-run or smoke-test sessions; no live trading required. |

---

## 7. Relationship to Prior Waves

| Prior wave | What it provides to this wave |
|-----------|------------------------------|
| Operational Foundation (S353–S357) | Prometheus metrics, health tracking, CI smoke tests |
| Session Intelligence (S459–S463) | PO verification, session audit bundle, metadata model |
| Session Access & Verification (S464–S468) | Batch audit, verification parameterization |
| Decision Quality (S469–S473) | Lineage, consistency checks, review surfaces |
| Strategy Effectiveness (S474–S478) | Effectiveness classification, P&L attribution |
| Round-Trip Pairing (S479–S483) | Leg matching, reconciliation flags, resolved rate |

This wave is the first to **compose** all prior measurement surfaces into an automated operational layer rather than adding a new measurement dimension.
