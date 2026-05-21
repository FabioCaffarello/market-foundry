# Operational Automation and Monitoring Hardening Wave — Charter and Scope Freeze

**Wave**: Operational Automation and Monitoring Hardening (S484–S489)
**Status**: OPEN — Scope Frozen
**Date**: 2026-03-26
**Predecessor**: S483 (Round-Trip Pairing Evidence Gate — PASS)

---

## 1. Strategic Context

Three consecutive measurement waves (S469–S483) built a deep, integrated stack:

| Layer | Wave | Stages | What it delivers |
|-------|------|--------|-----------------|
| Decision lineage | Decision Quality | S469–S473 | Signal → Decision causal chain, 9 cross-domain consistency checks |
| Execution effectiveness | Strategy Effectiveness | S474–S478 | Decision → Execution outcome classification, P&L attribution |
| Round-trip pairing | Round-Trip Pairing | S479–S483 | Entry/exit leg matching, reconciliation flags, resolved rate metric |

Each layer is read-path only, additive, operates on existing data, and closed with PASS.

**The measurement infrastructure is now three waves deep but not yet operationally hardened.** No automated alerting, no scheduled post-session verification, no operational state surfaces that an operator can monitor without querying HTTP endpoints manually, no batch triage workflow that aggregates anomalies across sessions.

This wave consolidates that capital into an operational layer that reduces manual review burden and surfaces problems proactively.

---

## 2. Problem Statement

### 2.1 What exists today

| Capability | Surface | Limitation |
|-----------|---------|-----------|
| PO verification (9 checks) | `po-verify.sh`, `GET /session/:id/verify` | Must be run manually per session |
| Session audit bundle | `GET /session/:id/audit` | Must be queried per session |
| Batch audit | `GET /session/batch-audit` | Sequential, 50-cap, no aggregation beyond summary counts |
| Effectiveness evaluation | `GET .../effectiveness/batch` | Returns raw list, no anomaly detection |
| Pairing review | `GET .../pairing/review` | Returns raw list, no trend or drift signals |
| Reconciliation flags | 8 flags + 3 reliability signals | Per-chain only, no session-level aggregation |
| Prometheus metrics | `/metrics` | Counters and histograms, no derived health signals |

### 2.2 What is missing

1. **Automated post-session verification** — PO verification should run automatically when a session transitions to terminal state, not depend on operator remembering to invoke it.
2. **Operational state surface** — A single endpoint that aggregates session health, reconciliation flag rates, resolved rates, and effectiveness distributions into a monitoring-friendly summary.
3. **Batch triage with anomaly signals** — Batch audit should surface which sessions need attention first, based on failed checks, degraded consistency, or anomalous reconciliation patterns.
4. **Structured operational report** — A machine-readable, session-scoped operational report that combines verification, effectiveness summary, and pairing summary into one artifact for archival and comparison.

### 2.3 What this wave does NOT address

This wave hardens what exists. It does not build new measurement capabilities, new exchange integrations, or new strategy families. See Non-Goals (Section 6).

---

## 3. Wave Structure

### 3.1 Blocks and stages

| Block | Stage | Name | Scope |
|-------|-------|------|-------|
| 1 | S485 | Automated post-session verification and operational report | Auto-trigger PO verification on session halt; structured operational report combining verification + effectiveness summary + pairing summary; persistence to `backups/sessions/` |
| 2 | S486 | Operational state and monitoring surfaces | Aggregated operational state endpoint; session-level reconciliation and resolved-rate summaries; Prometheus gauge extensions for operational health; monitoring-friendly JSON shape |
| 3 | S487 | Batch review and operational triage | Triage-oriented batch audit with anomaly ranking; cross-session reconciliation flag trend; effectiveness drift signals; operator-first query ergonomics |
| 4 | S488 | Integration proof and operational soak | End-to-end proof: session runs → auto-verification → operational state reflects results → batch triage surfaces anomalies; short soak demonstrating automated workflow |
| 5 | S489 | Evidence gate | Formal assessment, evidence matrix, residual gaps, wave verdict |

### 3.2 Dependency chain

```
S485 (auto-verify + report) → S486 (state surfaces) → S487 (batch triage) → S488 (integration proof) → S489 (gate)
```

Strictly sequential. S486 consumes the operational report from S485. S487 aggregates across the surfaces from S486. S488 exercises the full chain.

### 3.3 Estimated wave length

5 stages. Each stage is small and builds on consolidated assets. No new domain types, no write-path changes, no new exchange connectivity.

---

## 4. Governing Questions

| ID | Question | What PASS looks like |
|----|----------|---------------------|
| Q-OA1 | Does post-session verification run automatically without operator intervention? | PO verification triggers on session halt and persists a structured report. |
| Q-OA2 | Can an operator assess overall operational health from a single surface? | Operational state endpoint returns aggregated session health, reconciliation rates, and resolved rates. |
| Q-OA3 | Does batch triage surface which sessions need attention first? | Batch review ranks sessions by anomaly severity, failed checks, or degraded consistency. |
| Q-OA4 | Are operational reports machine-readable and archivable? | Reports follow a stable JSON schema, are persisted per session, and are queryable. |
| Q-OA5 | Does the automated workflow function end-to-end without manual steps? | Integration proof demonstrates session → auto-verify → state update → triage visibility without operator action. |

---

## 5. Capabilities

| ID | Capability | Stage | Depends on |
|----|-----------|-------|-----------|
| C-OA1 | Automated PO verification on session halt | S485 | Existing `VerifySessionUseCase`, session lifecycle events |
| C-OA2 | Structured operational report (verification + effectiveness + pairing) | S485 | Existing audit bundle, effectiveness batch, pairing batch |
| C-OA3 | Operational report persistence | S485 | Existing `backups/sessions/` structure |
| C-OA4 | Aggregated operational state endpoint | S486 | C-OA2 (operational reports as input) |
| C-OA5 | Session-level reconciliation and resolved-rate summaries | S486 | Existing pairing and reconciliation read models |
| C-OA6 | Prometheus gauge extensions for operational health | S486 | Existing `/metrics` infrastructure |
| C-OA7 | Triage-oriented batch audit with anomaly ranking | S487 | C-OA4 (state surface), existing batch audit |
| C-OA8 | Cross-session reconciliation flag trend | S487 | C-OA5 (per-session summaries) |
| C-OA9 | Effectiveness drift signals | S487 | Existing effectiveness batch endpoint |
| C-OA10 | End-to-end integration proof | S488 | C-OA1 through C-OA9 |

---

## 6. Non-Goals (Explicit)

| ID | Non-Goal | Why excluded |
|----|----------|-------------|
| NG-OA1 | Full observability platform (ELK, Loki, Grafana dashboards) | Infrastructure decision; out of scope for application-layer hardening |
| NG-OA2 | Push-based alerting (Alertmanager, PagerDuty, Slack) | Requires external integrations; operational state surface provides the data, but push delivery is infrastructure |
| NG-OA3 | OMS expansion (position tracking, portfolio engine) | Separate domain; not a monitoring concern |
| NG-OA4 | Multi-exchange support | No new exchange connectivity in this wave |
| NG-OA5 | New strategy families or strategy domain expansion | Out of scope; consolidation wave only |
| NG-OA6 | Cross-session position continuity | Separate wave candidate (G-RT4); this wave monitors sessions independently |
| NG-OA7 | Futures fee recovery write-path changes | Separate wave candidate (G-RT1); this wave works with existing data |
| NG-OA8 | ML-based scoring or predictive analytics | No data science infrastructure; premature |
| NG-OA9 | Real-time streaming or WebSocket surfaces | Read-path polling is sufficient for operational monitoring |
| NG-OA10 | Large structural refactoring | This wave adds surfaces and automation, not architectural changes |
| NG-OA11 | BI platform or data warehouse integration | Infrastructure decision; out of scope |
| NG-OA12 | Dashboard or visualization layer | Application provides JSON data; visualization is consumer-side |
| NG-OA13 | Historical trend analysis across weeks/months | Session-scoped and recent-batch-scoped only |
| NG-OA14 | New ClickHouse tables or schemas | Operates on existing read models |
| NG-OA15 | Write-path changes of any kind | Strictly read-path and automation-layer |

---

## 7. Guard Rails

1. **No infrastructure dependencies.** Every capability must work with the existing Go binary, NATS, ClickHouse stack. No new external services.
2. **No write-path changes.** All new code is read-path, automation, or composition of existing surfaces.
3. **No new domain types beyond operational reporting.** If a stage needs a new domain entity, it is out of scope.
4. **No scope creep into analytics.** Operational monitoring answers "is it healthy?" not "which strategy is best?"
5. **Each stage must close independently.** No stage depends on a future stage for its own evidence.

---

## 8. Success Criteria

The wave passes if:

1. All 5 governing questions are answered YES with evidence.
2. All 10 capabilities are delivered at FULL or SUFFICIENT level.
3. No CRITICAL or HIGH residual gaps remain.
4. The automated workflow (session halt → verification → state update → triage visibility) functions without manual steps.
5. No regressions in existing measurement surfaces.

---

## 9. Entry Conditions

All met:

- [x] S483 closed with PASS — round-trip pairing wave complete.
- [x] PO verification pipeline exists and is tested (S461).
- [x] Session audit bundle exists (S462).
- [x] Batch audit exists (S467).
- [x] Effectiveness evaluation surfaces exist (S476–S477).
- [x] Pairing and reconciliation surfaces exist (S480–S482).
- [x] Prometheus metrics infrastructure exists (S354).

---

## 10. Exit Criteria

The wave closes when S489 (evidence gate) delivers a formal verdict. Possible outcomes:

- **PASS**: All governing questions answered YES, all capabilities at FULL or SUFFICIENT, no CRITICAL/HIGH gaps.
- **CONDITIONAL PASS**: Minor gaps documented with mitigations; wave closes but gaps carry forward.
- **FAIL**: Capabilities incomplete or regressions detected; corrective stages required before closing.
