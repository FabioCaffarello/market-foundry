# Operational Automation Closure — Capabilities, Questions, and Non-Goals

**Wave**: Operational Automation Closure (S489–S493)
**Date**: 2026-03-26
**Charter**: [operational-automation-closure-wave-charter-and-scope-freeze.md](operational-automation-closure-wave-charter-and-scope-freeze.md)

---

## 1. Capabilities

### 1.1 Full capability map

| ID | Capability | Stage | Priority | Source Gap | Evidence Required |
|----|-----------|-------|----------|------------|-------------------|
| C-AC1 | Event-driven verification trigger on session halt | S490 | MUST | G-OA1 | Test proving halt event → automatic `VerifySession` invocation without operator action |
| C-AC2 | Unified operational report artifact (JSON) | S491 | MUST | G-OA2 | Test proving single JSON artifact containing verification + effectiveness + pairing data |
| C-AC3 | End-to-end automation proof (halt → verify → report → triage) | S491 | MUST | G-OA5 | Integration test proving full chain without manual steps |
| C-AC4 | Prometheus gauge extensions for operational health | S492 | SHOULD | G-OA3 | Gauge metrics reflecting verification state and triage severity in `/metrics` |
| C-AC5 | Reconciliation/resolved rates in monitoring surface | S492 | SHOULD | G-OA6 | `GET /monitoring/state` includes reconciliation rate and resolved rate |
| C-AC6 | Temporal trend signals in triage | S492 | MAY | G-OA4 | Triage endpoint includes trend direction or delta from prior session |

### 1.2 Dependency graph

```
C-AC1 (trigger) ──→ C-AC3 (proof)
                       ↑
C-AC2 (report) ───────┘
C-AC4 (gauges) ──→ independent
C-AC5 (rates)  ──→ independent
C-AC6 (trends) ──→ independent
```

C-AC3 depends on both C-AC1 and C-AC2. All SHOULD/MAY capabilities are independent
and can be delivered in any order.

### 1.3 Gap closure mapping

| Original Gap | Severity | Closing Capability | Expected Verdict |
|-------------|----------|-------------------|-----------------|
| G-OA1 (no auto-trigger) | MEDIUM | C-AC1 | FULL |
| G-OA2 (no unified report) | LOW | C-AC2 | FULL |
| G-OA5 (no e2e proof) | LOW | C-AC3 | FULL |
| G-OA3 (no Prometheus gauges) | LOW | C-AC4 | FULL or deferred |
| G-OA6 (no reconciliation rates) | LOW | C-AC5 | FULL or deferred |
| G-OA4 (no temporal trends) | LOW | C-AC6 | FULL or deferred |

---

## 2. Governing Questions

| ID | Question | Required Answer | How to Prove |
|----|----------|----------------|-------------|
| Q-AC1 | Does verification trigger automatically on session halt without operator intervention? | **YES** | Test: publish session halt event → verify `VerifySession` was invoked → verify report generated |
| Q-AC2 | Does the system produce a single, archivable operational report per session? | **YES** | Test: invoke report composition → verify JSON artifact contains verification scope, effectiveness summary, pairing metrics, triage flags |
| Q-AC3 | Does the end-to-end chain function without manual steps? | **YES** | Test: session halt → auto-trigger → report → monitoring state reflects session → triage surfaces session anomalies |
| Q-AC4 | Are operational health signals available in Prometheus? | **YES** (for FULL PASS) | Test: after verification, Prometheus gauge reflects state |

### 2.1 Question-to-capability mapping

| Question | Required Capabilities |
|----------|----------------------|
| Q-AC1 | C-AC1 |
| Q-AC2 | C-AC2 |
| Q-AC3 | C-AC1 + C-AC2 + C-AC3 |
| Q-AC4 | C-AC4 |

---

## 3. Non-Goals

These are explicitly excluded from the Operational Automation Closure wave.
They are legitimate future work but are frozen out to prevent scope inflation.

### 3.1 Infrastructure and platform

| # | Non-Goal | Rationale |
|---|----------|-----------|
| NG-1 | Observability platform (Grafana, Loki, Tempo) | Infrastructure dependency. Out of scope for a closure wave. |
| NG-2 | Push alerting (Slack, PagerDuty, email) | Requires external service integration. Not automation closure. |
| NG-3 | New ClickHouse tables or schemas | Write-path change. Existing tables sufficient. |
| NG-4 | New NATS subjects beyond session lifecycle | Scope inflation. Use existing subjects only. |

### 3.2 Domain expansion

| # | Non-Goal | Rationale |
|---|----------|-----------|
| NG-5 | OMS expansion (new order types, new lifecycle states) | Unrelated domain. |
| NG-6 | Multi-exchange support | Major feature. Not closure scope. |
| NG-7 | Cross-session position continuity | Separate structural gap (G-RT4). Requires its own wave. |
| NG-8 | New trading strategies or signal sources | Unrelated domain. |
| NG-9 | Futures fee recovery (write-path) | Separate gap (G-RT1). Requires write-path guard rail change. |

### 3.3 UX and presentation

| # | Non-Goal | Rationale |
|---|----------|-----------|
| NG-10 | Dashboards (web UI, admin panel) | Presentation layer. Not automation infrastructure. |
| NG-11 | Historical trend visualization | Requires storage and rendering. Beyond closure scope. |
| NG-12 | CLI commands for report retrieval | UX improvement, not automation. |

### 3.4 Structural

| # | Non-Goal | Rationale |
|---|----------|-----------|
| NG-13 | Large-scale refactoring of existing actors | Closure wave adds wiring, not redesign. |
| NG-14 | New domain packages beyond operational report composition | Composition uses existing domains. |
| NG-15 | Migration of existing endpoints to new patterns | Existing endpoints work. No migration needed. |

---

## 4. Decision Log

| Decision | Rationale | Date |
|----------|-----------|------|
| Wave limited to 5 stages (S489–S493) | Closure wave must be short. Anything beyond 5 stages signals scope inflation. | 2026-03-26 |
| S492 hardening is optional | MUST capabilities (C-AC1–C-AC3) are sufficient for PASS. SHOULD/MAY capabilities improve grade but do not block closure. | 2026-03-26 |
| NATS subscription preferred over polling for auto-trigger | Event-driven is the stated gap. Polling would be a workaround, not a closure. | 2026-03-26 |
| Unified report is JSON composition, not new storage | Existing surfaces provide data. Report composes them. No new persistence layer. | 2026-03-26 |
