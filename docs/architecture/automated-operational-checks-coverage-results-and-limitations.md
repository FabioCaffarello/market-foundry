# Automated Operational Checks: Coverage, Results, and Limitations

**Authority**: S485
**Date**: 2026-03-26
**Status**: Active

---

## 1. Purpose

This document provides the authoritative coverage matrix for the automated operational verification pipeline after S485 hardening. It maps each PO check to its automation level, surfaces, and remaining gaps.

---

## 2. PO Check Coverage Matrix (Post-S485)

| Check | Name | Automation | HTTP | Script | Session-Scoped (S485) | Evidence Quality |
|-------|------|-----------|------|--------|----------------------|-----------------|
| PO-1 | Kill-switch halt verification | Full | Yes | Yes | N/A (no time dependency) | gate_status field |
| PO-2 | Post-session backup | Manual | No (manual verdict) | Yes (filesystem) | N/A | Requires operator |
| PO-3 | ClickHouse intent records | Full | Yes | Yes | **Yes** — uses session time bounds | total_records, symbol |
| PO-4 | ClickHouse venue responses | Full | Yes | Yes | **Yes** — uses session time bounds | count, symbol |
| PO-5 | NATS KV state validation | Passthrough | Yes | Yes | N/A | Via session explain |
| PO-6 | System status summary | Passthrough | Yes | Yes | N/A | Gateway responds |
| PO-7 | Fee/commission fields | Full | Yes | Yes | **Yes** — uses session time bounds | total_fills, fills_with_fee, symbol |
| PO-8 | Lifecycle consistency | Skipped | Yes (skip) | Yes | **Yes** — uses session VenueType | Consistency checker not wired |
| PO-9 | Scope containment | Full | Yes | Yes | **Yes** — uses allowed symbols from scope | total_executions, out_of_scope, allowed_symbols |

### Legend
- **Full**: Automated verdict with structured evidence
- **Manual**: Requires human review (structural constraint)
- **Passthrough**: Always passes (presence of response is the check)
- **Skipped**: Dependency unavailable; documented as residual gap

---

## 3. Coverage Summary

| Metric | Before S485 | After S485 |
|--------|------------|-----------|
| Automated checks (HTTP) | 8/9 | 8/9 (unchanged) |
| Session-scoped checks | 0/9 | 5/9 (PO-3, PO-4, PO-7, PO-8, PO-9) |
| Checks with evidence.symbol | 0/9 | 4/9 (PO-3, PO-4, PO-7, PO-9) |
| Self-describing reports (Scope) | No | Yes |
| Batch check aggregation | No | Yes |
| Reproducible across time | No (24h drift) | Yes (session-bounded) |

---

## 4. Verification Surfaces

### 4.1 HTTP Endpoints (Gateway)

| Endpoint | S461 | S462 | S467 | S485 |
|----------|------|------|------|------|
| `GET /session/:id/verify` | ✓ Created | — | — | ✓ Session-scoped, scope in response |
| `GET /session/:id/audit` | — | ✓ Created | ✓ Check index | ✓ Session-scoped fee queries |
| `GET /session/batch-audit` | — | — | ✓ Created | ✓ Check aggregation in summary |

### 4.2 Script Surface

`scripts/po-verify.sh` remains the canonical operational harness. It is not modified in S485 (script uses its own filesystem-based checks and delegates server-side checks to the HTTP endpoints).

---

## 5. Structured Output Improvements (S485)

### 5.1 POVerificationReport.Scope

Every verification report now carries a `scope` field:

```json
{
  "scope": {
    "symbols": ["BTCUSDT"],
    "since": "2026-03-24T10:55:00Z",
    "until": "2026-03-24T12:05:00Z",
    "segments": ["spot"],
    "dry_run": true,
    "venue_type": "binance_spot"
  }
}
```

This enables:
- Reproducibility: re-running verification with the same scope produces the same results
- Auditability: scope is part of the evidence record
- Debugging: operators can see what was actually verified

### 5.2 BatchAuditSummary.CheckAggregation

Batch audit responses now include per-check verdict distribution:

```json
{
  "check_aggregation": [
    {"check_id": "PO-1", "pass_count": 5, "fail_count": 0, "warn_count": 0, "skip_count": 0},
    {"check_id": "PO-7", "pass_count": 3, "fail_count": 1, "warn_count": 1, "skip_count": 0}
  ]
}
```

This enables:
- Identifying recurring check failures without inspecting each session
- Prioritizing hardening effort on the checks that fail most

### 5.3 Evidence Enrichment

PO checks now include the verified symbol in their evidence maps, making each check result self-contained for audit purposes.

---

## 6. Gaps Still Not Automated

| Gap | Description | Effort | Priority |
|-----|-------------|--------|----------|
| PO-2 filesystem | Backup verification requires local filesystem access | Structural | Accepted |
| PO-8 consistency checker | CH-vs-KV cross-reader not wired | Medium | LOW |
| Multi-symbol scope | Segment→symbol mapping not implemented | Low (~30 lines) | LOW |
| Session-bounded lifecycle | Audit lifecycle query doesn't use session time bounds | Medium | LOW |
| Batch parallelization | Sequential execution; acceptable ≤50 sessions | Medium | LOW |
| 5 unchecked invariants | From S472: G1, G2, G3, G5, G6 | Medium | LOW-MEDIUM |

---

## 7. Test Coverage

### New Tests (S485)

| File | Test | Validates |
|------|------|-----------|
| `s485_verification_scope_test.go` | TestDefaultVerificationScope_Uses24hWindow | Default scope produces 24h window |
| `s485_verification_scope_test.go` | TestVerificationScope_InReport | Scope attached to report correctly |
| `s485_verification_scope_test.go` | TestBatchCheckAggregation_InSummary | Per-check aggregation across sessions |
| `s485_verification_scope_test.go` | TestBatchCheckAggregation_EmptyWhenNoVerification | No aggregation when no reports |
| `s485_verify_session_scoped_test.go` | TestVerifySession_DerivesScope_ClosedSession | Scope derived from closed session |
| `s485_verify_session_scoped_test.go` | TestVerifySession_DerivesScope_NilSession | Default scope when session unavailable |
| `s485_verify_session_scoped_test.go` | TestVerifySession_ScopeContainment_UsesAllowedSymbols | PO-9 uses scope symbols for containment |

### Updated Tests (S485)

All existing test stubs updated for new interface signatures (`Summary` instead of `Summary24h`, `List` instead of `List24h`):
- `s461_verify_session_test.go`: `stubCHSummary`, `stubCHLister`
- `s462_audit_session_test.go`: `stubFillReader`

### Existing Tests (Unchanged, Still Passing)

All S461, S462, S467 tests continue to pass without modification (only stubs updated).
