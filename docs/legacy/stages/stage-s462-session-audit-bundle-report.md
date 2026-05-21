# S462: Session Audit Bundle and Explainability Surface — Stage Report

Stage: S462
Wave: Session Intelligence & Operational Automation (S459–S463)
Predecessor: S461 (PO Automation and Verification Pipeline)
Successor: S463 (Session Intelligence Evidence Gate)

## Objective

Consolidate session metadata, automated checks, lifecycle state, order activity, fees, and consistency assessment into a single canonical audit bundle with a minimal operational explainability surface.

## Context

After S460 established the session entity with persistence and HTTP queryability, and S461 automated the PO verification pipeline, the operational review workflow still required hitting multiple endpoints and manually correlating data. An operator wanting to understand "what happened in session X?" had to:

1. Query session metadata (`GET /session/:id`)
2. Run PO verification (`GET /session/:id/verify`)
3. Query lifecycle list (`GET /execution/lifecycle/list`)
4. Query execution status per partition key
5. Check fee coverage from ClickHouse
6. Mentally correlate all of the above

S462 collapses this into a single endpoint that produces a structured, auditable bundle.

## Deliverables

### Code

| File | Type | Purpose |
|------|------|---------|
| `internal/domain/execution/audit_bundle.go` | Domain model | SessionAuditBundle, AuditLifecycleEntry, AuditOrderActivity, AuditFeeSummary, AuditConsistency types |
| `internal/application/executionclient/audit_session.go` | Use case | AuditSessionUseCase — 8-phase assembly with graceful degradation |
| `internal/application/executionclient/session_contracts.go` | Contracts | SessionAuditQuery/SessionAuditReply added |
| `internal/interfaces/http/handlers/session.go` | Handler | AuditSession handler for `GET /session/:id/audit` |
| `internal/interfaces/http/routes/session.go` | Routes | Route registration for audit endpoint |
| `internal/interfaces/http/routes/core.go` | Deps | SessionFamilyDeps extended with AuditSession |
| `cmd/gateway/compose.go` | Wiring | AuditSessionUseCase wired with session + lifecycle readers |

### Tests

| File | Count | Coverage |
|------|-------|----------|
| `internal/domain/execution/s462_audit_bundle_test.go` | 5 tests | Domain model: counters aggregation, fee summary, edge cases |
| `internal/application/executionclient/s462_audit_session_test.go` | 6 tests | Use case: full bundle, missing ID, not found, degraded, open session, nil reader |

Total: **11 tests**, all passing.

### Documentation

| File | Purpose |
|------|---------|
| `docs/architecture/session-audit-bundle-and-explainability-surface.md` | Architecture: bundle definition, assembly strategy, degradation model, endpoint, wiring |
| `docs/architecture/session-artifacts-orders-lifecycle-fees-checks-and-explainability-semantics.md` | Cross-surface linkage: entity relationships, data flow, read surface map, explainability levels |
| `docs/stages/stage-s462-session-audit-bundle-report.md` | This report |

## Capability Changes

| Capability | Pre-S462 | Post-S462 |
|------------|----------|-----------|
| Session audit review | Manual (5+ endpoints) | Single endpoint (`GET /session/:id/audit`) |
| Operational explainability | Per-partition only (S455A) | Session-level + per-partition |
| Fee auditability | Manual CH query | Structured coverage ratio in bundle |
| Consistency assessment | Manual comparison | Automated verdict (consistent/degraded/inconsistent) |
| Activity summary | Scattered across counters/lifecycle | Unified, source-attributed |

## Audit Bundle Structure

```json
{
  "bundle": {
    "session": { /* Session entity (S460) */ },
    "verification": { /* POVerificationReport (S461) */ },
    "lifecycle": [
      {
        "source": "binance_spot",
        "symbol": "BTCUSDT",
        "timeframe": 60,
        "intent_status": "submitted",
        "fill_status": "filled",
        "rejection_status": "",
        "propagation": "filled",
        "intent_count": 1,
        "fill_count": 1,
        "rejection_count": 0
      }
    ],
    "order_activity": {
      "total_intents": 5,
      "total_fills": 3,
      "total_rejections": 1,
      "total_errors": 0,
      "from_session_counters": true
    },
    "fee_summary": {
      "total_fill_records": 3,
      "fills_with_fee": 2,
      "fills_without_fee": 1,
      "simulated_fills": 1,
      "fee_assets": ["BNB"],
      "fee_coverage_ratio": "2/3"
    },
    "consistency": {
      "session_found": true,
      "verification_ran": true,
      "lifecycle_available": true,
      "counters_match_activity": true,
      "all_checks_passed": true,
      "overall_verdict": "consistent"
    },
    "explanation": "Session session_20260324_120000 (closed) ran from ... Activity: 5 intents, 3 fills, 1 rejections, 0 errors. Fees: 2/3 coverage. Verification: 2/3 passed, 0 failed, 0 warnings. Overall: consistent.",
    "assembled_at": "2026-03-24T12:00:00Z",
    "assembly_ms": 42
  }
}
```

## Degradation Scenarios

| Scenario | Verdict | Explanation |
|----------|---------|-------------|
| All surfaces available, all checks pass | `consistent` | Full audit bundle |
| Verification unavailable | `degraded` | Bundle assembled without PO checks |
| Lifecycle unavailable | `degraded` | Activity derived from session counters only |
| ClickHouse unavailable | `degraded` | Fee summary shows 0/0 |
| PO checks have failures | `inconsistent` | Bundle includes failures with evidence |
| Session not found | Error 404 | Audit cannot proceed |

## Limitations

1. **Verification not fully wired in HTTP** — the gateway compose wires the audit use case without verification (nil); the `scripts/po-verify.sh` script remains the canonical verification path.
2. **Fill reader not wired** — ClickHouse fill reader for fee summary is not connected in the HTTP composition; fee summary returns 0/0 via HTTP.
3. **24h query windows** — lifecycle and fill queries use 24h windows, not exact session time bounds.
4. **No cross-session comparison** — each audit bundle is single-session.
5. **Lifecycle counts are approximate** — KV stores latest state only; counts are 0 or 1 per partition key.

## Test Evidence

```
$ go test ./internal/domain/execution/ -run "TestNewAudit" -v
=== RUN   TestNewAuditOrderActivityFromCounters      PASS
=== RUN   TestNewAuditOrderActivityFromCounters_Empty PASS
=== RUN   TestNewAuditFeeSummary                      PASS
=== RUN   TestNewAuditFeeSummary_Empty                PASS
=== RUN   TestNewAuditFeeSummary_AllSimulated          PASS

$ go test ./internal/application/executionclient/ -run "TestAudit" -v
=== RUN   TestAuditSession_FullBundle                 PASS
=== RUN   TestAuditSession_MissingSessionID           PASS
=== RUN   TestAuditSession_SessionNotFound            PASS
=== RUN   TestAuditSession_DegradedWithoutVerification PASS
=== RUN   TestAuditSession_OpenSession                PASS
=== RUN   TestAuditSession_NilSessionReader           PASS
```

All existing tests continue to pass (execution domain, executionclient, HTTP handlers, routes).

## Wave Readiness for S463

S462 closes the audit bundle gap. The wave state entering S463:

| Capability | Source | Status |
|------------|--------|--------|
| Session entity (Q6) | S460 | COMPLETE |
| PO automation (C7+, Q5/Q8/Q11) | S461 | SUBSTANTIAL |
| Session audit bundle | S462 | COMPLETE |
| Operational explainability | S455A + S462 | SUBSTANTIAL |
| Session-level consistency | S462 | COMPLETE |

The evidence gate in S463 can now evaluate the full wave against the charter (S459).
