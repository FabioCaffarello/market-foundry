# Session HTTP Surface: Readers, Composition, and Limitations

**Stage**: S465
**Date**: 2026-03-24
**Status**: COMPLETE
**Wave**: Session Access & Verification Closure (S464--S468)

---

## 1. Purpose

This document maps the complete reader dependency tree for each session HTTP endpoint, documents which readers are wired, which degrade gracefully, and what limitations remain after S465.

---

## 2. Endpoint-to-Reader Map

### 2.1 GET /session/:id

| Reader | Type | Source | Required? |
|--------|------|--------|-----------|
| SessionGateway | NATS KV | `conns.session` | YES |

Returns session metadata from the NATS KV session bucket. Single reader, no ClickHouse dependency.

### 2.2 GET /session/list

| Reader | Type | Source | Required? |
|--------|------|--------|-----------|
| SessionGateway | NATS KV | `conns.session` | YES |

Lists all sessions from NATS KV. Single reader, no ClickHouse dependency.

### 2.3 GET /session/:id/verify

| Reader | Type | Source | Required? | Degrades? |
|--------|------|--------|-----------|-----------|
| SessionGateway (metadata) | NATS KV | `conns.session` | YES | N/A |
| ExecutionControlGateway (gate) | NATS KV | `conns.executionControl` | No | `skip` verdict |
| VerifyCHSummary (intent count) | ClickHouse | `sessionCHSummaryAdapter` | No | `skip` verdict |
| VerifyCHLister (venue responses) | ClickHouse | `sessionCHListerAdapter` | No | `skip` verdict |
| VerifyCHLister (fee fields) | ClickHouse | `sessionCHListerAdapter` | No | `skip` verdict |
| VerifyCHLister (scope containment) | ClickHouse | `sessionCHListerAdapter` | No | `skip` verdict |
| VerifyConsistencyChecker | Cross-surface | nil (not yet composed) | No | `skip` verdict |

9 PO checks total. 2 are always-pass/manual. 5 depend on ClickHouse. 1 depends on execution control gateway. 1 depends on the unimplemented consistency checker.

**After S465**: All available readers are wired. Only the consistency checker remains nil.

### 2.4 GET /session/:id/audit

| Reader | Type | Source | Required? | Degrades? |
|--------|------|--------|-----------|-----------|
| SessionGateway (metadata) | NATS KV | `conns.session` | YES | Returns 503 |
| VerifySessionUseCase (PO checks) | Composite | verify UC | No | `bundle.Verification = nil` |
| AuditLifecycleReader (lifecycle) | NATS KV | `conns.execution` | No | `bundle.Lifecycle = []` |
| AuditCHFillReader (fee analysis) | ClickHouse | `sessionCHListerAdapter` | No | `bundle.FeeSummary = 0/0` |

**Before S465**: Verification was nil (always skipped), fill reader was nil (fee summary always 0/0). The audit bundle returned with `consistency.overallVerdict = "degraded"`.

**After S465**: Verification runs all 9 PO checks. Fee analysis reads fills from ClickHouse. The audit bundle can now return `consistent` or `inconsistent` verdicts.

---

## 3. Composition Dependency Graph

```
buildRouteDependencies()
  |
  +-- conns.session != nil?
  |     |
  |     +-- GetSessionUseCase (always)
  |     +-- ListSessionsUseCase (always)
  |     |
  |     +-- chClient != nil?
  |     |     +-- sessionCHSummaryAdapter
  |     |     +-- sessionCHListerAdapter (dual: VerifyCHLister + AuditCHFillReader)
  |     |
  |     +-- conns.executionControl != nil?
  |     |     +-- GetExecutionControlUseCase (gate reader)
  |     |
  |     +-- VerifySessionUseCase(session, gate, chSummary, chLister, nil)
  |     |
  |     +-- conns.execution != nil?
  |     |     +-- GetLifecycleListUseCase (audit lifecycle reader)
  |     |
  |     +-- AuditSessionUseCase(session, verifyUC, lifecycle, fillReader)
```

---

## 4. Limitations and Residual Gaps

| Gap | Severity | Description | Resolution |
|-----|----------|-------------|------------|
| Consistency checker nil | LOW | Cross-surface CH-vs-KV check skipped | Requires composite reader composition (future) |
| 24h fixed window | LOW | CH adapters hardcode 24h lookback | S466 verification parameterization |
| Hardcoded BTCUSDT scope | LOW | Several PO checks query single symbol | S466 session-bounded queries |
| No session-time-bounded queries | LOW | Verification queries 24h, not session window | S466 |

---

## 5. References

- [Gateway Wiring Closure](./gateway-wiring-and-session-http-closure.md)
- [S464 Charter](../stages/stage-s464-session-access-verification-charter-report.md)
