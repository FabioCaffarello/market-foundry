# Session Access & Verification Closure -- Capabilities, Questions, and Non-Goals

**Wave**: Session Access & Verification Closure (S464--S468)
**Date**: 2026-03-24
**Predecessor**: S463 (Session Intelligence & Operational Automation Evidence Gate)

---

## 1. Capabilities

### 1.1 Capability Matrix

This wave does not introduce new capabilities. It closes gaps in existing capabilities delivered by S459--S463.

| ID | Capability | Current Grade | Gap IDs | Target Grade | Description |
|----|-----------|--------------|---------|-------------|-------------|
| C7+.http | Verification via HTTP Audit Path | DEGRADED (nil) | G3 | FULL | VerifySessionUseCase wired into AuditSessionUseCase in gateway compose; audit endpoint includes verification results |
| C9.fees | Audit Bundle Fee Analysis | DEGRADED (0/0) | G4 | FULL | ClickHouse fill reader wired into AuditSessionUseCase; fee summary reflects actual fill data |
| C7+.scope | Verification Scope Parameterization | HARDCODED | G2 | SESSION-AWARE | Scope parameters (symbol, segment, exchange) derived from session ConfigSnapshot rather than constants |
| C9.time | Session-Bounded Queries | APPROXIMATE (24h) | G5 | SESSION-BOUNDED | Lifecycle and fill queries use session started_at/closed_at instead of 24h window |

### 1.2 Capability Details

#### C7+.http: Verification via HTTP Audit Path

**Current state**: `AuditSessionUseCase` is constructed with `nil` for the `verifyUseCase` parameter in `cmd/gateway/compose.go:261`. The audit endpoint skips Phase 2 (PO verification) and marks `VerificationRan = false`, producing a `degraded` consistency verdict.

**Target state**: `VerifySessionUseCase` is fully constructed with its own dependencies and passed to `AuditSessionUseCase`. The audit endpoint executes all 8 programmatic PO checks and includes their results in the bundle.

**Dependencies to resolve**:
- `VerifySessionUseCase` requires: `VerifyGateReader`, `VerifySessionReader`, `VerifyCHSummary`, `VerifyCHLister`, `VerifyConsistencyChecker`.
- `VerifyGateReader` maps to `GetExecutionControlUseCase` (available when `executionControl` gateway is wired).
- `VerifySessionReader` maps to `GetSessionUseCase` (available when `session` gateway is wired).
- `VerifyCHSummary` and `VerifyCHLister` require ClickHouse client (available when `chClient != nil`).
- `VerifyConsistencyChecker` requires the explain use case or direct CH+KV cross-check.

**Evidence required**: Test proving audit endpoint returns non-nil verification section with check results.

#### C9.fees: Audit Bundle Fee Analysis

**Current state**: `AuditSessionUseCase` is constructed with `nil` for the `fillReader` parameter in `cmd/gateway/compose.go:265`. The `computeFeeSummary` method returns `{FeeCoverageRatio: "0/0"}`.

**Target state**: ClickHouse `ExecutionReader` (or a thin adapter) is wired as `AuditCHFillReader`. The fee summary reflects actual fill records with fee coverage analysis.

**Dependencies to resolve**:
- `AuditCHFillReader` interface requires `List24h(ctx, symbol, execType, status, limit)`.
- `clickhouse.ExecutionReader` must expose or be adapted to satisfy this interface.
- The ClickHouse client is already available in `buildRouteDependencies` when `chClient != nil`.

**Evidence required**: Test proving fee summary contains non-zero values when fills exist in ClickHouse.

#### C7+.scope: Verification Scope Parameterization

**Current state**: The `VerifySessionUseCase` uses hardcoded constants: `"binance_spot"` for source, `"BTCUSDT"` for symbol, and `24h` for time window. These are embedded in the use case logic.

**Target state**: Verification scope is derived from the session's `ConfigSnapshot`:
- `exchange` and `segment` from `ConfigSnapshot.Segments` and `ConfigSnapshot.VenueType`.
- `symbol` from session metadata or config (currently single-symbol; the parameterization prepares for future multi-symbol without implementing it).
- Time window from session `started_at`/`closed_at`.

**Evidence required**: Test proving verification runs with session-derived parameters, not hardcoded constants.

#### C9.time: Session-Bounded Queries

**Current state**: Lifecycle and fill queries in the audit and verification use cases use `time.Now().Add(-24*time.Hour)` as the start bound. This is a conservative approximation that works for short sessions but includes extraneous data for systems that run multiple sessions per day.

**Target state**: Queries use the actual session time bounds:
- `started_at` as the lower bound.
- `closed_at` (or `time.Now()` for open sessions) as the upper bound.

**Evidence required**: Test proving queries use session timestamps, not 24h window.

---

## 2. Governing Questions

### 2.1 Questions for This Wave

| ID | Question | Target Stage | Linked Gap |
|----|----------|-------------|------------|
| Q12 | Does the audit endpoint return a non-degraded bundle with verification and fee data via HTTP alone? | S465 | G3, G4 |
| Q13 | Are verification scope parameters derived from the session being verified? | S466 | G2 |
| Q14 | Do time-bounded queries use actual session start/close timestamps? | S466 | G5 |
| Q15 | Is the audit bundle explanation complete and accurate when all surfaces are wired? | S467 | -- |

### 2.2 Question-to-Stage Mapping

```
S465: Q12
S466: Q13, Q14
S467: Q15
S468: All questions graded
```

### 2.3 How Questions Are Answered

| ID | Answered When |
|----|--------------|
| Q12 | Test sends GET /session/:id/audit (or use case test) and receives bundle with `verification != nil`, `fee_summary.total_fill_records > 0`, `consistency.verification_ran = true`, `consistency.overall_verdict != "degraded"` |
| Q13 | Test creates session with specific config snapshot, runs verification, confirms scope parameters match session config (not hardcoded constants) |
| Q14 | Test creates session with specific time bounds, confirms queries use session.started_at and session.closed_at (not time.Now()-24h) |
| Q15 | Test runs full audit with all surfaces wired, confirms explanation text includes verification counts, fee coverage, and accurate session timing |

---

## 3. Non-Goals

### 3.1 Explicit Exclusions

| ID | Non-Goal | Rationale | Related Concern |
|----|----------|-----------|----------------|
| NG1 | New supervised live session | Parallel S457 track; this wave closes existing gaps without live execution | Operator availability |
| NG2 | Broad dashboards or observability platform | This wave is wiring and parameterization, not visualization | Scope discipline |
| NG3 | OMS expansion (new order types, states, lifecycle changes) | OMS Foundation (S382--S388) is stable; this wave is read-side only | Architecture boundary |
| NG4 | Multi-exchange support | Binance-only per existing scope; session model is exchange-agnostic by design | Incremental approach |
| NG5 | Storage or runtime structural redesign | Uses existing KV + CH + NATS + HTTP patterns; changes are wiring-only | Architecture stability |
| NG6 | New HTTP endpoints | No new routes; existing audit, verify, and session endpoints are unchanged | Surface stability |
| NG7 | ClickHouse schema changes | Read-only usage of existing schema | Schema stability |
| NG8 | Config or compose topology changes | Deployment topology preserved; changes are in Go composition code | Deployment stability |
| NG9 | Automated session orchestration | Session remains passive metadata; no auto-start, auto-halt | Scope containment |
| NG10 | Cross-session comparison or trending | Single-session audit scope; comparison is a future analytics concern | Scope freeze |
| NG11 | PO-2 (backup) HTTP automation | Filesystem access is an architectural constraint; script covers it (G1 accepted) | Accepted limitation |
| NG12 | Real-time streaming, push alerting, or WebSocket surfaces | Post-hoc verification and query only | Complexity containment |
| NG13 | Performance optimization, pagination, or caching | Future wave; current data volumes are small | Premature optimization |
| NG14 | ClickHouse persistence for sessions | KV sufficient for bounded session count; CH persistence deferred | Storage simplicity |
| NG15 | Spot scope expansion or futures live execution | Blocked by standing gates (S451, S457) | Strategic gate |
| NG16 | New test infrastructure or testing frameworks | Standard Go test patterns only | Tooling discipline |
| NG17 | Refactoring existing use case signatures or domain types | Additive changes only; no breaking changes to existing interfaces | Interface stability |

### 3.2 Non-Goal Enforcement

Each non-goal is a **binding exclusion**. If during implementation a task appears to require any of the above, the correct response is:

1. Document the finding.
2. Flag it as a residual gap in the evidence gate.
3. Do NOT expand scope.

---

## 4. Constraints

### 4.1 Technical Constraints

| Constraint | Rationale |
|-----------|-----------|
| No new external services | Existing NATS + CH + HTTP are sufficient |
| No ClickHouse schema changes | Existing schema is read-only for this wave |
| No modifications to existing HTTP endpoint paths or contracts | Response shape may include previously-nil fields; no breaking changes |
| No changes to execution path (actors, adapters, pipeline) | This wave is entirely read-side and wiring |
| No new KV buckets | Existing session bucket is sufficient |
| No new domain types | Existing types (Session, POVerificationReport, SessionAuditBundle) are sufficient |

### 4.2 Process Constraints

| Constraint | Rationale |
|-----------|-----------|
| Wave must complete independently of second live session | No blocking on operator availability |
| Each stage must produce tests | Evidence-backed delivery per wave convention |
| Evidence gate must confirm G3 and G4 are closed | These are the MEDIUM-severity gaps driving this wave |
| Scope freeze is absolute | No additions without a new charter |

---

## 5. Prior Art and Reuse

| Existing Asset | How This Wave Reuses It |
|---------------|------------------------|
| `VerifySessionUseCase` (S461) | Wired as dependency into AuditSessionUseCase in compose |
| `AuditSessionUseCase` (S462) | Existing use case; no code changes to core logic |
| `clickhouse.ExecutionReader` | Adapted or used directly as AuditCHFillReader |
| `cmd/gateway/compose.go` | Primary modification point for wiring |
| `Session.ConfigSnapshot` (S460) | Source of scope parameters for verification |
| `Session.StartedAt`/`ClosedAt` (S460) | Source of time bounds for queries |
| `po-verify.sh` (S461) | Reference for correct verification behavior; script remains canonical |

---

## 6. Success Metrics

| Metric | Threshold |
|--------|-----------|
| Q12 fully answered | Audit endpoint returns non-degraded bundle via HTTP |
| Q13 fully answered | Verification scope derived from session config |
| Q14 fully answered | Queries bounded by session timestamps |
| G3 closed | Verification wired in gateway compose |
| G4 closed | Fill reader wired in gateway compose |
| G2 closed | Scope parameters are session-aware |
| G5 closed | Time bounds are session-aware |
| Zero regressions | All existing tests pass |
| Scope compliance | Zero non-goals violated |
