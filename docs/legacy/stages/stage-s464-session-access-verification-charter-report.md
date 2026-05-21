# Stage S464 -- Session Access & Verification Closure Wave Charter Report

**Stage**: S464
**Type**: Charter and Scope Freeze
**Status**: COMPLETE
**Date**: 2026-03-24
**Wave**: Session Access & Verification Closure (S464--S468)
**Predecessor**: S463 (Session Intelligence & Operational Automation Evidence Gate)

---

## 1. Executive Summary

S464 opens the Session Access & Verification Closure wave -- a short, focused wave that closes the MEDIUM and LOW severity gaps left by the Session Intelligence & Operational Automation wave (S459--S463).

The predecessor wave delivered substantial value (first-class sessions, automated PO verification, consolidated audit bundles, 41 tests) but closed with 2 MEDIUM gaps (G3: verification not wired in gateway, G4: fill reader not wired in gateway) and 6 LOW gaps. The audit endpoint operates in degraded mode via HTTP because verification and fee analysis dependencies were passed as nil during gateway composition.

This wave resolves these gaps through code-only changes that require no API keys, no operator availability, and no live session. It is a closure wave, not a capability wave -- it finishes what was designed but not connected.

Scope is frozen at 4 execution stages (S465--S468) targeting:
1. Gateway wiring closure (G3, G4).
2. Verification parameterization and session-bounded queries (G2, G5).
3. Audit bundle usability hardening.
4. Evidence gate confirming all gaps closed.

---

## 2. What Was Analyzed

### 2.1 Predecessor Wave State

The Session Intelligence & Operational Automation wave (S459--S463) delivered:

| Capability | Grade | Limitation |
|-----------|-------|------------|
| C3+ Session Metadata Persistence | FULL | None |
| C7+ PO Verification Automation | SUBSTANTIAL | Not wired in HTTP audit path; hardcoded scope |
| C8 Batch Consistency Audit | SUBSTANTIAL | Lifecycle counts approximate |
| C9 Session Audit Bundle | SUBSTANTIAL | Degraded without verification and fees |

### 2.2 Residual Gaps

| Gap | Severity | Addressable? | Resolution |
|-----|----------|-------------|------------|
| G3: Verification not wired in gateway audit | MEDIUM | YES (code wiring) | S465 |
| G4: Fill reader not wired in gateway | MEDIUM | YES (code wiring) | S465 |
| G2: Scope parameters hardcoded | LOW | YES (parameterization) | S466 |
| G5: 24h time window approximation | LOW | YES (session-bound queries) | S466 |
| G1: PO-2 not HTTP-automated | LOW | NO (filesystem constraint) | Accepted |
| G6: No CH persistence for sessions | LOW | Deferred | Accepted |
| G7: No cross-session comparison | LOW | Deferred | Accepted |
| G8: Lifecycle counts approximate | LOW | NO (KV fundamental) | Accepted |

### 2.3 Gateway Compose Analysis

The wiring gap is localized to `cmd/gateway/compose.go:248--269`:

```go
sessionDeps.AuditSession = executionclient.NewAuditSessionUseCase(
    getSessionUC,
    nil, // verification wired below if ClickHouse available  <-- G3
    auditLifecycleReader,
    nil, // fill reader wired below if ClickHouse available    <-- G4
)
```

Both `nil` parameters have well-defined interfaces (`verifySessionExecutor`, `AuditCHFillReader`) and their implementations exist. The gap is purely compositional.

---

## 3. Wave Charter

### 3.1 Blocks and Order

| Stage | Scope | Depends On | Closes |
|-------|-------|-----------|--------|
| S464 | Charter and scope freeze (this stage) | S463 | -- |
| S465 | Gateway wiring and session HTTP closure | S464 | G3, G4 |
| S466 | Verification parameterization and operator ergonomics | S464 | G2, G5 |
| S467 | Session evidence usability and audit bundle hardening | S465, S466 | -- |
| S468 | Evidence gate | S467 | Wave closure |

### 3.2 Governing Questions

| ID | Question | Target |
|----|----------|--------|
| Q12 | Does the audit endpoint return a non-degraded bundle with verification and fee data via HTTP alone? | S465 |
| Q13 | Are verification scope parameters derived from the session being verified? | S466 |
| Q14 | Do time-bounded queries use actual session start/close timestamps? | S466 |
| Q15 | Is the audit bundle explanation complete and accurate when all surfaces are wired? | S467 |

### 3.3 Non-Goals (Summary)

17 explicit non-goals documented in the capabilities document. Key exclusions:
- No new supervised live session (NG1).
- No dashboards or observability platform (NG2).
- No OMS expansion (NG3).
- No multi-exchange (NG4).
- No structural redesign (NG5).
- No new HTTP endpoints (NG6).
- No ClickHouse schema changes (NG7).

---

## 4. Deliverables Produced

| Deliverable | Path |
|-------------|------|
| Wave Charter and Scope Freeze | `docs/architecture/session-access-and-verification-closure-wave-charter-and-scope-freeze.md` |
| Capabilities, Questions, and Non-Goals | `docs/architecture/session-access-verification-capabilities-questions-and-non-goals.md` |
| Stage Report (this document) | `docs/stages/stage-s464-session-access-verification-charter-report.md` |

---

## 5. Preparation for S465

### 5.1 Concrete Wiring Plan

**G3 Resolution** (verification into audit):

1. In the `if conns.session != nil` block of `buildRouteDependencies`:
   - Construct `VerifySessionUseCase` with available dependencies.
   - `VerifyGateReader` = `GetExecutionControlUseCase(conns.executionControl)` if available, else nil.
   - `VerifySessionReader` = `getSessionUC` (already constructed).
   - `VerifyCHSummary` and `VerifyCHLister` = adapt from `chExecutionReader` if `chClient != nil`.
   - `VerifyConsistencyChecker` = adapt from explain use case or construct new.
2. Pass the constructed `VerifySessionUseCase` instead of `nil` to `NewAuditSessionUseCase`.

**G4 Resolution** (fill reader into audit):

1. If `chClient != nil`, construct `AuditCHFillReader` adapter from `clickhouse.ExecutionReader`.
2. Pass it instead of `nil` to `NewAuditSessionUseCase`.

### 5.2 Risk Notes

- `VerifySessionUseCase` has 5 dependencies. Some may not be available in gateway context (e.g., `VerifyConsistencyChecker` requires both CH and KV). The use case already handles nil deps gracefully -- partial wiring is acceptable and produces partial (non-nil) verification.
- The audit endpoint should never error when a verification dependency is unavailable -- it should degrade that specific check, not the entire verification section.

### 5.3 Test Strategy

- Unit test: construct `AuditSessionUseCase` with non-nil verification and fill reader; assert bundle.Verification != nil and bundle.FeeSummary.TotalFillRecords > 0.
- Regression: all existing audit tests must continue passing.

---

## 6. Verdict

**S464: COMPLETE. Wave formally opened with scope frozen.**

The Session Access & Verification Closure wave (S464--S468) is chartered to close the 2 MEDIUM and 2 LOW gaps remaining from S463. Scope is frozen at gateway wiring, verification parameterization, and audit bundle hardening. 17 non-goals are explicit and binding. 4 governing questions are formulated. Stage order is fixed.

---

## 7. References

- [S463 Evidence Gate Report](stage-s463-session-intelligence-evidence-gate-report.md)
- [S463 Evidence Matrix](../architecture/session-intelligence-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [S459 Wave Charter](../architecture/session-intelligence-and-operational-automation-wave-charter-and-scope-freeze.md)
- [S459 Capabilities and Non-Goals](../architecture/session-intelligence-capabilities-questions-and-non-goals.md)
- [Wave Charter](../architecture/session-access-and-verification-closure-wave-charter-and-scope-freeze.md) (S464)
- [Capabilities, Questions, and Non-Goals](../architecture/session-access-verification-capabilities-questions-and-non-goals.md) (S464)
