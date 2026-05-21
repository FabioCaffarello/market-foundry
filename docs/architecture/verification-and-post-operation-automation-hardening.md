# Verification and Post-Operation Automation Hardening

**Authority**: S485
**Date**: 2026-03-26
**Status**: Active

---

## 1. Purpose

This document describes the hardening of the verification and post-operation automation pipeline delivered in S485. The focus is on making verification session-aware, reducing operator intervention, improving structured output quality, and enabling cross-session operational triage.

---

## 2. Problem Statement

Prior to S485, the verification pipeline had three structural limitations:

1. **Fixed 24h time window**: All ClickHouse queries used `time.Now().Add(-24h)` regardless of session boundaries. Multiple sessions within 24h polluted each other's verification results, and sessions older than 24h could not be verified.

2. **Hardcoded symbol scope**: PO checks (PO-3, PO-4, PO-7, PO-8, PO-9) hardcoded `"BTCUSDT"` instead of deriving symbols from session configuration. Verification was not self-aware of what it was verifying.

3. **No cross-session check-level triage**: Batch audit (`GET /session/batch-audit`) aggregated session-level verdicts (consistent/degraded/inconsistent) but did not aggregate which specific PO checks failed across sessions. Operators had to inspect each session individually to identify recurring check failures.

---

## 3. Changes Delivered

### 3.1 Session-Scoped Verification (VerificationScope)

**Domain**: `internal/domain/execution/verification.go`

New `VerificationScope` struct captures the session-derived boundaries:
- `Symbols []string` — allowed symbols (from session config)
- `Since time.Time` — start of verification window (session.StartedAt - 5min buffer)
- `Until time.Time` — end of verification window (session.ClosedAt + 5min buffer)
- `Segments []string` — segment identifiers from session config
- `DryRun bool` — execution mode
- `VenueType string` — venue type

The scope is derived automatically from the session entity when available, with `DefaultVerificationScope()` providing the 24h/BTCUSDT fallback when session metadata is unavailable.

**Impact**: `POVerificationReport` now carries a `Scope *VerificationScope` field, making every verification report self-describing and reproducible.

### 3.2 Session-Aware CH Interfaces

**Application**: `internal/application/executionclient/verify_session.go`, `audit_session.go`

Updated interfaces:
- `VerifyCHSummary.Summary(ctx, symbol, since, until)` — replaces `Summary24h`
- `VerifyCHLister.List(ctx, symbol, execType, status, limit, since, until)` — replaces `List24h`
- `AuditCHFillReader.List(ctx, symbol, execType, status, limit, since, until)` — replaces `List24h`

**Gateway**: `cmd/gateway/session_reader.go`

Both adapters (`sessionCHSummaryAdapter`, `sessionCHListerAdapter`) now pass caller-provided `since`/`until` to `QueryExecutionList` instead of computing `time.Now().Add(-24h)`.

### 3.3 Session-Derived Scope in VerifySessionUseCase

**Application**: `internal/application/executionclient/verify_session.go`

The `Execute` method now:
1. Fetches session metadata (existing Phase 1)
2. Calls `deriveVerificationScope(session)` to build scope
3. Passes scope to all check methods that query ClickHouse

`deriveVerificationScope` extracts:
- Time bounds from `Session.StartedAt` / `Session.ClosedAt` (with 5-minute buffers)
- VenueType and Segments from `Session.Config`
- Symbols (currently defaulting to BTCUSDT; multi-symbol mapping deferred to S486+)

### 3.4 Scope-Aware PO Checks

All ClickHouse-dependent checks now use scope:
- **PO-3** (intent records): uses `scope.Symbols[0]` and `scope.Since/Until`
- **PO-4** (venue responses): uses `scope.Symbols[0]` and `scope.Since/Until`
- **PO-7** (fee fields): uses `scope.Symbols[0]` and `scope.Since/Until`
- **PO-8** (lifecycle consistency): uses `scope.VenueType` as source
- **PO-9** (scope containment): uses `scope.Symbols` as allowed set and `scope.Since/Until` for time window

### 3.5 Batch Check Aggregation

**Domain**: `internal/domain/execution/audit_bundle.go`

New `BatchCheckAggregation` struct:
- `CheckID string` — PO check identifier
- `PassCount`, `FailCount`, `WarnCount`, `SkipCount` — verdict distribution

Added to `BatchAuditSummary` as `CheckAggregation []BatchCheckAggregation`.

`ComputeBatchSummary` now aggregates per-check verdicts across all sessions in canonical PO check order (PO-1 through PO-9). Operators can now see "PO-7 failed in 3/5 sessions" at a glance.

---

## 4. Backward Compatibility

All changes are backward compatible:
- `VerificationScope` is an optional field (omitted when nil)
- `BatchCheckAggregation` is `omitempty` (absent when empty)
- When session metadata is unavailable, `DefaultVerificationScope()` reproduces the prior 24h/BTCUSDT behavior
- HTTP API response shapes are additive (new fields only)

---

## 5. Design Decisions

| Decision | Rationale |
|----------|-----------|
| 5-minute buffer on session time bounds | Accounts for inflight events and late ClickHouse writes without over-fetching |
| Symbol defaults to BTCUSDT when not derivable | Maintains backward compatibility; multi-symbol mapping is a separate concern |
| Check aggregation uses canonical PO order | Consistent presentation; operators can compare across reports |
| Scope attached to report, not query | Makes reports self-describing without requiring the caller to know scope details |

---

## 6. Limitations

1. **Multi-symbol scope**: Symbol derivation currently defaults to BTCUSDT. Multi-symbol mapping from segments requires segment→symbol configuration that doesn't exist yet.
2. **PO-2 remains manual**: Backup verification requires filesystem access. No change in S485.
3. **PO-8 consistency checker still nil**: The `VerifyConsistencyChecker` is not wired in gateway composition. S485 does not change this; PO-8 remains `skip` in HTTP surface.
4. **Batch audit sequential**: Not parallelized. Acceptable for current scale (≤50 sessions).
5. **No session-bounded lifecycle queries**: The audit use case's lifecycle reader still uses segment-based filtering, not session time bounds. This is a separate gap.
