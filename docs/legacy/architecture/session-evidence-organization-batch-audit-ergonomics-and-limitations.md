# Session Evidence Organization, Batch Audit Ergonomics, and Limitations

**Stage:** S467
**Date:** 2026-03-24

---

## 1. Session Evidence Organization

### 1.1 Evidence Surfaces After S467

| Surface | Endpoint | Scope | Added/Improved |
|---------|----------|-------|----------------|
| Session metadata | `GET /session/:id` | Single session | S460 |
| Session list | `GET /session/list` | All sessions | S460 |
| PO verification | `GET /session/:id/verify` | Single session, 9 checks | S461 |
| Audit bundle | `GET /session/:id/audit` | Single session, consolidated | S462 |
| **Batch audit** | **`GET /session/batch-audit`** | **Multiple sessions** | **S467** |
| Session explain | `GET /analytical/execution/explain` | Per-partition | S455A |
| Lifecycle list | `GET /execution/lifecycle/list` | All partitions | S413 |

### 1.2 Audit Bundle Contents After S467

The `SessionAuditBundle` now contains:

```
session               -- metadata, config, activation, counters
verification          -- 9 PO check results (optional, degrades)
lifecycle[]           -- per-partition lifecycle state
order_activity        -- aggregated intent/fill/rejection/error counts
fee_summary           -- fill coverage and fee asset breakdown
consistency           -- session-level cross-surface assessment
check_index           -- [NEW] per-check verdict map with failed/warned lists
explanation           -- [IMPROVED] includes specific failed/warned check IDs
assembled_at          -- timestamp
assembly_ms           -- latency
```

### 1.3 How Evidence Connects

```
Session Metadata (KV)
  |
  +-- PO Verification --> Check Index (quick scan)
  |     |
  |     +-- PO-1: Gate status
  |     +-- PO-2: Backup (manual)
  |     +-- PO-3: Intent records (CH)
  |     +-- PO-4: Venue responses (CH)
  |     +-- PO-5: KV state
  |     +-- PO-6: System status
  |     +-- PO-7: Fee fields (CH)
  |     +-- PO-8: Lifecycle consistency
  |     +-- PO-9: Scope containment (CH)
  |
  +-- Lifecycle Entries (KV via NATS request-reply)
  |     filtered by session config segment
  |
  +-- Order Activity (session counters or lifecycle-derived)
  |
  +-- Fee Summary (CH fill reader)
  |
  +-- Consistency Assessment
        verdict: consistent | degraded | inconsistent
```

---

## 2. Batch Audit Ergonomics

### 2.1 Operator Workflow Before S467

```
1. GET /session/list                    --> session IDs
2. for each ID:
     GET /session/{id}/audit            --> audit bundle
3. manually compare verdicts
4. manually find failed checks in each report
```

### 2.2 Operator Workflow After S467

```
1. GET /session/batch-audit             --> all terminal sessions audited
   or
   GET /session/batch-audit?status=halted  --> only halted sessions
   or
   GET /session/batch-audit?ids=s1,s2      --> specific sessions

2. Read summary.consistent / summary.degraded / summary.inconsistent
3. For flagged sessions: read entry.bundle.check_index.failed
```

### 2.3 Triage Path

1. **Quick health**: check `summary` -- if all consistent, done.
2. **Identify problems**: filter entries where `bundle.consistency.overall_verdict != "consistent"`.
3. **Drill down**: read `bundle.check_index.failed` for specific check IDs.
4. **Deep dive**: read `bundle.verification.checks[]` for evidence details on specific checks.
5. **Correlate**: use `bundle.lifecycle[]` and `bundle.fee_summary` for operational context.

---

## 3. Ergonomics Decisions

| Decision | Rationale |
|----------|-----------|
| Sequential per-session audit | Simplicity; current session count (~5-20) doesn't warrant parallel |
| Max 50 sessions per batch | Prevents unbounded resource use; explicit IDs bypass auto-resolve |
| Terminal-only default | Open sessions have incomplete data; terminal sessions are review-ready |
| Check index as flat map | Enables O(1) lookup by check ID; failed/warnings arrays for quick filter |
| Lifecycle filter by first segment | Pragmatic for current single-segment configs; multi-segment noted as limitation |

---

## 4. What Remains Outside Scope

| Item | Rationale | Where It Belongs |
|------|-----------|-----------------|
| Cross-session trending | NG10 from S464 charter; requires time-series analysis | Future analytics wave |
| Parallel batch audit | Current session count doesn't justify complexity | Scaling wave if needed |
| Batch pagination | 50-session cap sufficient; no known use case for >50 | Scaling wave if needed |
| Multi-segment lifecycle coverage | Would require N lifecycle queries per session | S468 or future |
| PO-2 backup automation | Filesystem access constraint accepted in S463 | Accepted limitation |
| Consistency checker wiring | Requires composite CH+KV reader not yet built | S468+ |
| Session-bounded time windows | Fee summary still uses 24h window, not session bounds | S468+ |
| Real-time alerting | Post-hoc review only; no push notifications | Out of wave scope |

---

## 5. Coverage Summary

| Metric | Value |
|--------|-------|
| New use cases | 1 (BatchAuditSessionUseCase) |
| New domain types | 4 (BatchAuditResult, BatchAuditEntry, BatchAuditSummary, AuditCheckIndex) |
| New HTTP endpoints | 1 (`GET /session/batch-audit`) |
| New tests | 11 (5 use case, 3 domain, 3 handler) |
| Modified files | 7 (audit_bundle.go, session_contracts.go, session.go handlers, session.go routes, core.go, audit_session.go, compose.go) |
| New files | 4 (batch_audit_session.go, s467_batch_audit_test.go, s467_audit_bundle_test.go, s467_batch_audit_test.go handlers) |
| Regressions | 0 (all existing tests pass) |
