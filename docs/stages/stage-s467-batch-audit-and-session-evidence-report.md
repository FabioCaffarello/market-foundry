# S467: Batch Audit and Session Evidence Usability

**Status:** Complete
**Date:** 2026-03-24

## Objective

Improve batch audit and session evidence usability for operational review, reducing friction in multi-session triage and individual check inspection.

## Deliverables

### Code Changes

1. **Batch audit domain types (`audit_bundle.go`):** Added `BatchAuditResult`, `BatchAuditEntry`, `BatchAuditSummary`, `AuditCheckIndex` types. `ComputeBatchSummary` aggregates verdicts. `NewAuditCheckIndex` builds per-check verdict map with failed/warnings arrays.

2. **Batch audit contracts (`session_contracts.go`):** Added `SessionBatchAuditQuery` (supports explicit IDs, status filter) and `SessionBatchAuditReply`.

3. **Batch audit use case (`batch_audit_session.go`):** `BatchAuditSessionUseCase` resolves session IDs (explicit or auto-resolve terminal), audits each independently, captures per-entry errors, computes aggregate summary. Capped at 50 sessions.

4. **Batch audit handler (`session.go` handlers):** `BatchAuditSessions` handler parses `status` and `ids` query params. `splitCommaSeparated` helper for comma-delimited IDs.

5. **Route registration (`session.go` routes):** `GET /session/batch-audit` registered before `/:id` wildcard.

6. **Route dependencies (`core.go`):** `SessionFamilyDeps` extended with `BatchAuditSession` field.

7. **Gateway wiring (`compose.go`):** Batch audit wired from existing `ListSessions` and `AuditSession` use cases.

8. **Audit bundle improvements (`audit_session.go`):** Check index populated from verification report. Lifecycle query filtered by session's first config segment. Explanation text includes specific failed/warned check IDs.

### Tests

- `s467_audit_bundle_test.go` (domain): 3 tests -- check index population, nil report handling, batch summary computation.
- `s467_batch_audit_test.go` (executionclient): 5 tests -- explicit IDs, auto-resolve terminal, status filter, partial failure, nil deps.
- `s467_batch_audit_test.go` (handlers): 3 tests -- 200 response, nil use case 503, IDs param.

**Total: 11 new tests, 0 regressions.**

### Documentation

- `docs/architecture/batch-audit-and-session-evidence-usability.md`
- `docs/architecture/session-evidence-organization-batch-audit-ergonomics-and-limitations.md`

## What Changed

| Area | Before S467 | After S467 |
|------|-------------|------------|
| Batch review | N individual `/session/:id/audit` calls | Single `GET /session/batch-audit` with aggregate summary |
| Check triage | Parse full verification report JSON | `check_index.failed` and `check_index.warnings` arrays |
| Explanation | Generic verification summary line | Includes specific failed/warned check IDs |
| Lifecycle query | Unfiltered (all partitions) | Filtered by session's first config segment |

## Acceptance Criteria Evaluation

| Criterion | Met |
|-----------|-----|
| Batch audit and session evidence more usable | Yes -- single-call batch audit with aggregate summary |
| Operational review gains clarity and less friction | Yes -- check index, improved explanation, filtered lifecycle |
| Closes majority of Session Intelligence residual value | Yes -- batch was the primary missing surface |
| Ready for evidence gate in S468 | Yes -- all endpoints wired and tested |

## Limitations

- Batch audit is sequential, not parallel (acceptable for current scale).
- Maximum 50 sessions per batch; no pagination.
- Lifecycle filter uses first segment only; multi-segment sessions may miss secondary entries.
- Fee summary still uses 24h window approximation (accepted residual from S466).
- Consistency checker remains nil (requires composite CH+KV reader).
- PO-2 backup check remains manual (filesystem constraint).
- No cross-session trending or comparison.

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/execution/audit_bundle.go` | BatchAuditResult/Entry/Summary, AuditCheckIndex types |
| `internal/application/executionclient/session_contracts.go` | BatchAuditQuery/Reply contracts |
| `internal/application/executionclient/batch_audit_session.go` | New: batch audit use case |
| `internal/application/executionclient/audit_session.go` | Check index, lifecycle filter, improved explanation |
| `internal/interfaces/http/handlers/session.go` | BatchAuditSessions handler, batchAuditSessionUseCase interface |
| `internal/interfaces/http/routes/session.go` | `/session/batch-audit` route |
| `internal/interfaces/http/routes/core.go` | SessionFamilyDeps.BatchAuditSession field |
| `cmd/gateway/compose.go` | Batch audit wiring |

## Files Added

| File | Purpose |
|------|---------|
| `internal/application/executionclient/s467_batch_audit_test.go` | 5 batch audit use case tests |
| `internal/domain/execution/s467_audit_bundle_test.go` | 3 check index and batch summary tests |
| `internal/interfaces/http/handlers/s467_batch_audit_test.go` | 3 handler tests |
| `docs/architecture/batch-audit-and-session-evidence-usability.md` | Architecture doc |
| `docs/architecture/session-evidence-organization-batch-audit-ergonomics-and-limitations.md` | Ergonomics and limitations doc |
