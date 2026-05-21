# Session Access & Verification Closure Wave -- Charter and Scope Freeze

**Wave**: Session Access & Verification Closure
**Charter Stage**: S464
**Date**: 2026-03-24
**Predecessor**: S463 (Session Intelligence & Operational Automation Evidence Gate -- WAVE CLOSED, SUBSTANTIALLY COMPLETE)
**Parallel Track**: S457 (Second Supervised Live Session -- pending operator availability)

---

## 1. Strategic Context

The Session Intelligence & Operational Automation wave (S459--S463) delivered substantial value: first-class session entities, automated PO verification, consolidated audit bundles, and three levels of operational explainability. The wave closed with verdict SUBSTANTIALLY COMPLETE.

However, two MEDIUM-severity gaps remain in the HTTP/gateway surface, and several LOW gaps affect verification ergonomics and audit usability:

| Gap | Severity | Source | Description |
|-----|----------|--------|-------------|
| G3 | MEDIUM | S463 | Verification use case not wired in HTTP gateway audit path |
| G4 | MEDIUM | S463 | ClickHouse fill reader not wired in HTTP gateway |
| G2 | LOW | S463 | Scope parameters hardcoded (Binance Spot, BTCUSDT, 24h) |
| G5 | LOW | S463 | 24h time window approximation instead of session-bound queries |
| G1 | LOW | S463 | PO-2 (backup) not automated at HTTP level |

These gaps mean:
- The audit endpoint (`GET /session/:id/audit`) operates in **degraded mode**: verification returns nil, fee summary returns 0/0.
- PO verification scope is hardcoded rather than derived from session config.
- Time-bounded queries approximate with 24h windows instead of exact session bounds.

This wave closes these gaps with a short, focused set of stages. No new capabilities are introduced -- only existing capabilities are completed to their intended design.

### 1.1 Why Now

- These gaps are **code-only fixes** -- no API keys, operator availability, or live session required.
- The audit endpoint is the canonical operational review surface; degraded mode reduces its value.
- Completing gateway wiring makes the system ready for the second supervised live session (S457 track) with a fully self-contained audit path.
- The work is small, bounded, and low-risk.

### 1.2 What This Wave Is NOT

This is not a new capability wave. It is a **closure wave** -- finishing what S459--S463 started, wiring what was designed but not connected, and hardening the verification surface for operational use.

---

## 2. Problem Statement

### 2.1 What the System Can Do Today

After S459--S463, the system has:
- First-class session entity with 12 fields, KV persistence, 2 HTTP endpoints (S460).
- Automated PO verification: 8/9 checks programmatic, 9/9 via script, structured JSON output (S461).
- Consolidated audit bundle: single endpoint, 8-phase assembly, degradation model (S462).
- 41 new tests, zero regressions.

### 2.2 What the System Cannot Do Today

| Gap | Impact | Root Cause |
|-----|--------|------------|
| Audit endpoint shows no verification results | Operator must run `po-verify.sh` separately; HTTP-only workflows are incomplete | `VerifySessionUseCase` passed as nil in gateway compose |
| Audit endpoint shows 0/0 fee coverage | Fee analysis unavailable via HTTP; script queries CH directly | `AuditCHFillReader` passed as nil in gateway compose |
| Verification scope is hardcoded | Cannot verify sessions with different symbols or segments without code changes | Scope derived from constants rather than session config snapshot |
| Time windows approximate 24h | Query results may include data outside session bounds | Session `started_at`/`closed_at` not used for time-bounded queries |

### 2.3 Root Cause

The S459--S463 wave prioritized the domain model, use cases, and test coverage over gateway wiring. This was the correct priority -- the model had to be right before the plumbing. The plumbing gaps are now the remaining work.

---

## 3. Wave Objective

Close all MEDIUM-severity HTTP/gateway gaps from S463 and improve verification ergonomics so the audit endpoint operates at full capacity without degradation, with session-aware scope and time bounds.

**Outcome**: `GET /session/:id/audit` returns a complete, non-degraded bundle including verification results, fee analysis, session-scoped queries, and session-derived parameters -- without requiring the script surface.

---

## 4. Wave Blocks

```
S464  Charter and Scope Freeze                    <-- THIS STAGE
  |
  +---> S465  Gateway Wiring and Session HTTP Closure
  |             - Wire VerifySessionUseCase into AuditSessionUseCase in compose.go
  |             - Wire ClickHouse ExecutionReader as AuditCHFillReader in compose.go
  |             - Audit endpoint returns non-degraded bundle
  |             - G3 and G4 formally closed
  |
  +---> S466  Verification Parameterization and Operator Ergonomics
  |             - Derive scope (symbol, segment, exchange) from session config snapshot
  |             - Use session started_at/closed_at for time-bounded queries
  |             - G2 and G5 formally closed
  |
  +---> S467  Session Evidence Usability and Audit Bundle Hardening
  |             - Audit bundle includes verification summary counts
  |             - Consistency verdict reflects full (non-degraded) assembly
  |             - Audit explanation text covers fee and verification details
  |
  +---> S468  Session Access & Verification Closure Evidence Gate
                - All gaps graded
                - Wave closure verdict
                - Audit endpoint operates at full capacity
```

### Dependency Model

- S465 and S466 are independent and can execute in parallel after S464.
- S467 depends on both S465 and S466 (it validates the combined result).
- S468 depends on S467.

```
S464 --+--> S465 --+--> S467 --> S468
       |           |
       +--> S466 --+
```

---

## 5. Capability Definitions

This wave does not introduce new capabilities. It closes gaps in existing ones:

| ID | Capability | Current Grade | Gap | Target Grade |
|----|-----------|--------------|-----|-------------|
| C7+ | PO Verification Automation | SUBSTANTIAL | G3: not wired in gateway audit path | FULL (HTTP path) |
| C9 | Session Audit Bundle | SUBSTANTIAL | G3+G4: degraded without verification and fees | FULL |
| C7+.scope | Verification Scope Awareness | N/A (hardcoded) | G2: parameters not session-derived | Session-aware |
| C9.time | Session-Bounded Queries | N/A (24h approx) | G5: not using session time bounds | Session-bounded |

---

## 6. Governing Questions

| ID | Question | Target Stage |
|----|----------|-------------|
| Q12 | Does the audit endpoint return a non-degraded bundle with verification and fee data via HTTP alone? | S465 |
| Q13 | Are verification scope parameters derived from the session being verified? | S466 |
| Q14 | Do time-bounded queries use actual session start/close timestamps? | S466 |
| Q15 | Is the audit bundle explanation complete and accurate when all surfaces are wired? | S467 |

---

## 7. Alignment with Existing Infrastructure

This wave modifies **only wiring and parameterization** -- no new infrastructure:

| Infrastructure | Usage | Changes |
|---------------|-------|---------|
| NATS KV | Session metadata (existing bucket) | None |
| NATS Request-Reply | Session and execution queries (existing) | None |
| ClickHouse | Fill reader for fee analysis (existing reader) | Wired into gateway compose |
| HTTP Gateway | Existing audit endpoint | Wiring changes only |
| Scripts | `po-verify.sh` (existing) | None |

No new external dependencies. No new services. No new databases. No new KV buckets. No new HTTP endpoints.

---

## 8. Scope Freeze

### 8.1 What Is In Scope

1. Wire `VerifySessionUseCase` into `AuditSessionUseCase` in `cmd/gateway/compose.go`.
2. Wire ClickHouse `ExecutionReader` (or equivalent) as `AuditCHFillReader` in `cmd/gateway/compose.go`.
3. Derive verification scope (symbol, segment, exchange) from session's `ConfigSnapshot`.
4. Use session `started_at`/`closed_at` for time-bounded lifecycle and fill queries.
5. Validate audit bundle completeness when all surfaces are wired.
6. Evidence gate confirming G2, G3, G4, G5 are closed.

### 8.2 What Is NOT In Scope (Non-Goals)

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG1 | New supervised live session | Parallel S457 track; this wave is independent |
| NG2 | Broad dashboards or observability platform | Data wiring and ergonomics only |
| NG3 | OMS expansion (new order types, states, lifecycle) | OMS Foundation is stable and untouched |
| NG4 | Multi-exchange support | Binance-only per existing scope |
| NG5 | Storage or runtime structural redesign | Wiring changes only |
| NG6 | New HTTP endpoints | Existing endpoints only; changes are internal wiring |
| NG7 | ClickHouse schema changes | Read-only usage of existing schema |
| NG8 | Config or compose topology changes | Deployment topology preserved |
| NG9 | Automated session orchestration | Session remains passive metadata |
| NG10 | Cross-session comparison or trending | Single-session audit scope |
| NG11 | PO-2 (backup) HTTP automation | Filesystem access constraint accepted in S463 |
| NG12 | Real-time streaming or push alerting | Post-hoc verification only |
| NG13 | Performance optimization or pagination | Future wave concern |
| NG14 | ClickHouse persistence for sessions | KV sufficient for current needs |
| NG15 | Spot scope expansion or futures live execution | Blocked by standing gates |

**Scope is frozen. No additions permitted without a new charter.**

---

## 9. Gap-to-Stage Mapping

| Gap | Source | Stage | Resolution |
|-----|--------|-------|------------|
| G3 (verification not wired in gateway) | S463 MEDIUM | S465 | Wire VerifySessionUseCase into audit compose |
| G4 (fill reader not wired in gateway) | S463 MEDIUM | S465 | Wire ExecutionReader as AuditCHFillReader |
| G2 (scope parameters hardcoded) | S463 LOW | S466 | Derive from session ConfigSnapshot |
| G5 (24h time window approximation) | S463 LOW | S466 | Use session started_at/closed_at |
| G1 (PO-2 not HTTP-automated) | S463 LOW | -- | Accepted: filesystem constraint (NG11) |
| G6 (no CH persistence for sessions) | S463 LOW | -- | Accepted: KV sufficient (NG14) |
| G7 (no cross-session comparison) | S463 LOW | -- | Accepted: out of scope (NG10) |
| G8 (lifecycle counts approximate) | S463 LOW | -- | Accepted: KV limitation, counters are authoritative |

**5 gaps addressed. 3 gaps accepted as bounded limitations.**

---

## 10. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Wiring verification creates circular dependency | Low | Medium | VerifySessionUseCase is a leaf use case with no compose-back dependency |
| CH fill reader wiring requires new interface adapter | Low | Low | AuditCHFillReader interface already defined; adapter is trivial |
| Session-derived scope breaks hardcoded tests | Medium | Low | Update test fixtures to use session config snapshot |
| Wave scope creeps into endpoint redesign | Low | Medium | Scope freeze: wiring and parameterization only |

---

## 11. Success Criteria

| Criterion | Measure |
|-----------|---------|
| Q12 answered | `GET /session/:id/audit` returns non-degraded bundle with verification and fees |
| Q13 answered | Verification scope derived from session config, not constants |
| Q14 answered | Time-bounded queries use session timestamps |
| G3 and G4 closed | Both MEDIUM gaps resolved with test evidence |
| No regression | All existing tests pass; zero failures |
| Scope discipline | Zero non-goals violated |

---

## 12. Preparation for S465

### Recommended Pre-Work

1. **Read compose wiring**: `cmd/gateway/compose.go` lines 248--269 -- understand current nil-wiring points.
2. **Read AuditSessionUseCase**: `internal/application/executionclient/audit_session.go` -- understand the 4 dependencies.
3. **Read VerifySessionUseCase**: `internal/application/executionclient/verify_session.go` -- understand its own dependencies for composition.
4. **Read ClickHouse ExecutionReader**: `internal/adapters/clickhouse/execution_reader.go` -- identify the method that satisfies `AuditCHFillReader`.
5. **Map dependency chain**: VerifySessionUseCase needs (VerifyGateReader, VerifySessionReader, VerifyCHSummary, VerifyCHLister, VerifyConsistencyChecker) -- identify which are available in gateway compose.

### S465 Entry Criteria

- S464 charter accepted (this document).
- Dependency chain for VerifySessionUseCase mapped.
- CH fill reader adapter identified.

### S465 Exit Criteria

- `cmd/gateway/compose.go` wires VerifySessionUseCase and AuditCHFillReader into AuditSessionUseCase.
- `GET /session/:id/audit` returns non-degraded bundle in tests.
- G3 and G4 formally closed with test evidence.
