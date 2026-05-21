# Batch Audit and Session Evidence Usability

**Stage:** S467
**Date:** 2026-03-24
**Predecessor:** S466 (Verification Parameterization and Operator Ergonomics)

---

## 1. Problem

Before S467, the session audit surface supported only single-session queries:

- `GET /session/:id/audit` -- one session at a time
- `GET /session/:id/verify` -- one verification run at a time
- `GET /session/list` -- list all sessions (no audit data)

To review multiple sessions, operators had to:
1. Call `/session/list` to enumerate sessions.
2. Call `/session/:id/audit` for each session individually.
3. Manually compare verdicts across sessions.
4. Parse the flat verification report to find specific check failures.

This friction was acceptable for single-session post-operation review but impractical for batch review of historical sessions.

---

## 2. Solution

### 2.1 Batch Audit Endpoint

New endpoint: `GET /session/batch-audit`

Query parameters:
- `status` (optional): filter sessions by status (e.g., `closed`, `halted`)
- `ids` (optional): comma-separated session IDs for explicit selection

Behavior:
- When `ids` is provided, those specific sessions are audited.
- When `ids` is empty, all terminal sessions (closed or halted) are auto-resolved from the session list.
- When `status` is set, only sessions matching that status are included.
- Each session is audited independently; individual failures are captured per-entry rather than aborting the batch.
- Maximum 50 sessions per batch (capped by `BatchAuditMaxSessions`).

Response includes:
- `entries[]`: per-session audit bundle (or error)
- `summary`: aggregate verdict counts (consistent, degraded, inconsistent, errored)
- `assembled_at`, `assembly_ms`: timing metadata

### 2.2 Check Index

Each `SessionAuditBundle` now includes a `check_index` field:

```json
{
  "check_index": {
    "verdicts": {
      "PO-1": "pass",
      "PO-2": "manual",
      "PO-3": "fail",
      "PO-4": "warn"
    },
    "failed": ["PO-3"],
    "warnings": ["PO-4"]
  }
}
```

This provides at-a-glance triage without parsing the full verification report. The `failed` and `warnings` arrays enable quick filtering.

### 2.3 Improved Audit Explanation

The human-readable `explanation` field now includes specific check IDs when failures or warnings are present:

> Verification: 7/9 passed, 1 failed, 1 warnings. Failed checks: PO-3. Warned checks: PO-4.

### 2.4 Session-Aware Lifecycle Filtering

The audit assembly now uses the session's first segment from `Config.Segments` to filter the lifecycle query, reducing noise from unrelated partitions.

---

## 3. Architecture

### 3.1 Use Case Composition

```
BatchAuditSessionUseCase
  +-- listSessionsExecutor  (resolves session IDs)
  +-- auditSessionExecutor  (audits each session)
        +-- sessionReader
        +-- verifyUseCase
        +-- lifecycleReader
        +-- fillReader
```

The batch use case composes existing use cases without duplication. It delegates to the same `AuditSessionUseCase` used by the single-session endpoint.

### 3.2 Route Registration

`/session/batch-audit` is registered as a fixed-path route before the `/:id` wildcard, following the same pattern as `/session/list`.

### 3.3 Gateway Wiring

The batch audit use case is wired in `cmd/gateway/compose.go` from:
- `ListSessions` use case (already available)
- `AuditSession` use case (already available from S465)

No new gateways, connections, or external dependencies.

---

## 4. Alignment with Existing Surfaces

| Surface | Relationship |
|---------|-------------|
| Session metadata (S460) | Batch audit reads session list |
| PO verification (S461) | Each session's audit includes verification |
| Audit bundle (S462) | Batch delegates to single-session audit |
| Gateway wiring (S465) | Batch inherits all S465 reader wiring |
| Parameterization (S466) | Check index leverages structured verdicts |

---

## 5. Limitations

- Batch audit is sequential (no parallel per-session queries) -- acceptable for current session counts.
- Maximum 50 sessions per batch; no pagination.
- No cross-session trending or comparison (NG10 from S464 charter).
- No real-time push notification; batch audit is on-demand.
- Lifecycle filtering uses only the first segment from config; multi-segment sessions may miss entries from secondary segments.
