# Session Intelligence & Operational Automation -- Capabilities, Questions, and Non-Goals

**Wave**: Session Intelligence & Operational Automation (S459--S463)
**Date**: 2026-03-24
**Predecessor**: S456A (Operational History & Explainability Evidence Gate)

---

## 1. Capabilities

### 1.1 Capability Matrix

| ID | Capability | Description | Depends On | Target Grade |
|----|-----------|-------------|-----------|-------------|
| C3+ | Session Metadata Persistence | First-class session entity with structured fields, persisted in NATS KV, queryable via HTTP | Existing KV infrastructure | FULL |
| C7+ | Post-Session Verification Automation | All 9 PO checks from S447 codified as executable, structured validations with pass/fail output | Existing HTTP endpoints (S453A--S455A) | FULL |
| C8 | Batch Consistency Audit | Automated sweep of all KV keys in a time window against corresponding ClickHouse records; reports divergences as structured output | Existing KV + CH infrastructure | SUBSTANTIAL |
| C9 | Session Audit Bundle | Single consolidated artifact (JSON or structured report) combining session metadata, all PO check results, and explain output for every execution in the session window | C3+, C7+, existing explain endpoint | SUBSTANTIAL |

### 1.2 Capability Details

#### C3+: Session Metadata Persistence

**Problem**: Sessions are implicit time windows. There is no entity, no ID, no structured metadata, and no way to query "what happened in session X" as a unit.

**Solution**: Define a minimal session entity with the following fields:

| Field | Type | Source |
|-------|------|--------|
| session_id | string (UUID or timestamp-based) | Generated at session start |
| started_at | RFC3339 timestamp | System boot or operator start command |
| ended_at | RFC3339 timestamp (nullable) | Kill-switch halt or operator stop |
| operator | string | Config or environment |
| exchange | string | Config (e.g., "binance") |
| segment | string | Config (e.g., "spot") |
| symbol | string | Config (e.g., "BTCUSDT") |
| execution_mode | string | Config (e.g., "venue_live", "paper", "dry_run") |
| dry_run | bool | Config |
| halt_reason | string (nullable) | Kill-switch or operator halt reason |
| outcome | string | "completed", "halted", "aborted" |
| config_snapshot | JSON blob | Serialized execution config at start time |

**Persistence**: NATS KV bucket `session_metadata`. Key format: `session_id`.

**Query surface**:
- `GET /analytical/session/:id` -- retrieve session by ID.
- `GET /analytical/session/list` -- list sessions with optional time/segment/mode filters.

**Evidence required**: Tests proving persistence round-trip and HTTP query.

#### C7+: Post-Session Verification Automation

**Problem**: S447 defined 9 PO checks but only 2 were executed in S449. Manual execution is slow, error-prone, and inconsistent.

**Solution**: Codify all 9 checks as a structured verification pipeline.

| PO Check | Description | Data Source |
|----------|-------------|-------------|
| PO-1 | ClickHouse record count matches expected | `GET /analytical/execution/summary` |
| PO-2 | No type confusion (all records match expected type) | `GET /analytical/execution/list?type=...` |
| PO-3 | No stuck status (all records reach terminal state) | `GET /analytical/execution/list?status=...` |
| PO-4 | KV-to-CH consistency for all keys | `GET /analytical/execution/explain` (batch) |
| PO-5 | Non-zero fees for live fills | `GET /analytical/execution/list` with fee field check |
| PO-6 | Kill-switch state correct post-session | Kill-switch HTTP endpoint |
| PO-7 | Backup bracket verified (pre + post) | Backup script exit code |
| PO-8 | No out-of-scope executions (segment/symbol containment) | `GET /analytical/execution/list` with segment filter |
| PO-9 | Session halt reason documented | Session metadata (C3+) |

**Output**: Structured JSON report with per-check pass/fail, evidence, and timestamps.

**Execution**: Single command (`make po-verify SESSION_ID=...` or equivalent script).

#### C8: Batch Consistency Audit

**Problem**: The explain endpoint checks one key at a time. There is no automated way to sweep all keys in a session window and report divergences.

**Solution**: Script or test harness that:
1. Lists all KV keys in the session time window (via `GET /execution/lifecycle/list`).
2. Calls explain for each key.
3. Aggregates results into a structured divergence report.

**Output**: JSON report with total keys, consistent count, divergent count, unavailable count, and per-key details for divergences.

#### C9: Session Audit Bundle

**Problem**: Post-session review requires the operator to manually call multiple endpoints and mentally combine results.

**Solution**: Single command or endpoint that produces a consolidated audit bundle:

```json
{
  "session": { ... C3+ metadata ... },
  "verification": { ... C7+ PO results ... },
  "consistency": { ... C8 batch audit ... },
  "executions": [ ... explain output per key ... ]
}
```

**Persistence**: Written to `backups/sessions/<session_id>/audit-bundle.json` or served via HTTP.

---

## 2. Governing Questions

### 2.1 Questions Inherited from S456A (Unanswered)

| ID | Question | Original Source | Target Stage |
|----|----------|----------------|-------------|
| Q5 | Can post-session verification run without manual intervention? | S452A | S461 |
| Q6 | Does session-level metadata exist as queryable state? | S452A | S460 |

### 2.2 New Questions for This Wave

| ID | Question | Target Stage |
|----|----------|-------------|
| Q7 | Can the system produce a single consolidated audit artifact for any session? | S462 |
| Q8 | Does the batch consistency audit detect divergences that per-key checking misses? | S461 |
| Q9 | Can the operator review a session's full operational history without touching multiple endpoints manually? | S462 |
| Q10 | Is the session metadata model stable enough to survive future session types (multi-symbol, futures) without breaking changes? | S460 |
| Q11 | Can PO verification run against historical sessions (not just the most recent)? | S461 |

### 2.3 Question-to-Stage Mapping

```
S460: Q6, Q10
S461: Q5, Q8, Q11
S462: Q7, Q9
S463: All questions graded
```

---

## 3. Non-Goals

### 3.1 Explicit Exclusions

| ID | Non-Goal | Rationale | Related Concern |
|----|----------|-----------|----------------|
| NG1 | New supervised live session | Parallel S457 track; this wave produces value without live execution | Operator availability |
| NG2 | Spot Scope Expansion | Blocked by S451 GO/NO-GO; requires second session evidence | Strategic gate |
| NG3 | Futures live execution | Out of scope per standing freeze | Risk containment |
| NG4 | OMS expansion (new order types, states, lifecycle changes) | OMS Foundation (S382--S388) is stable; session intelligence is read-side | Architecture boundary |
| NG5 | Broad dashboards or visualization UI | Data correctness and automation first; presentation is a future concern | Scope discipline |
| NG6 | Multi-exchange support | Binance-only per existing scope; session model is exchange-agnostic by design | Incremental approach |
| NG7 | Structural redesign of storage or runtime | Uses existing KV + CH + NATS + HTTP patterns | Architecture stability |
| NG8 | Real-time streaming, push alerting, or WebSocket surfaces | Post-hoc verification and query only | Complexity containment |
| NG9 | Automated session orchestration (auto-start, auto-halt, scheduled sessions) | Session metadata is passive observation; operator controls lifecycle | Scope freeze |
| NG10 | Config or compose topology changes | Existing deployment topology preserved; new capabilities are additive | Deployment stability |
| NG11 | Performance optimization, cursor-based pagination, or query caching | Future wave (G6 from S456A); current data volumes are small | Premature optimization |
| NG12 | Cross-domain lifecycle trace (signal-to-fill composite chain) | Execution-domain trace is complete; composite trace is a separate concern | Boundary discipline |
| NG13 | Fee/commission model changes | S428 fee normalization is stable and untouched | Architecture boundary |
| NG14 | External API endpoints or public-facing surfaces | Internal operational review only | Security boundary |
| NG15 | Automated trading decisions based on PO results | PO verification produces reports; it does not make trading decisions | Scope containment |

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
| No ClickHouse schema changes | Session metadata lives in KV; CH is queried read-only for consistency |
| No modifications to existing endpoints | New endpoints only; existing ones preserved |
| No changes to execution path (actors, adapters, pipeline) | This wave is entirely read-side and metadata |
| New KV bucket only | `session_metadata` bucket follows existing patterns |

### 4.2 Process Constraints

| Constraint | Rationale |
|-----------|-----------|
| Wave must complete independently of S458 (second live session) | No blocking on operator availability |
| Each stage must produce tests | Evidence-backed delivery per wave convention |
| Evidence gate must formally answer Q5 and Q6 | These are inherited obligations from S456A |
| Scope freeze is absolute | No additions without a new charter |

---

## 5. Prior Art and Reuse

| Existing Asset | How This Wave Reuses It |
|---------------|------------------------|
| S453A lifecycle history endpoint | PO checks consume it for timeline verification |
| S454A list/summary endpoints | PO checks consume them for count/type/status validation |
| S455A explain endpoint | Batch consistency audit iterates over explain |
| S454A lifecycle/list endpoint | Provides KV key enumeration for batch audit |
| S447 PO protocol | Defines the 9 checks being automated |
| S434 credential provider pattern | Session metadata follows similar KV persistence pattern |
| S429 segment health pattern | Session health follows similar structured-output pattern |

---

## 6. Success Metrics

| Metric | Threshold |
|--------|-----------|
| Q5 fully answered | PO harness executes all 9 checks in < 30 seconds |
| Q6 fully answered | Session entity round-trips through KV and HTTP |
| Q7 fully answered | Audit bundle produced by single command |
| Zero regressions | All 334+ existing tests pass |
| Test coverage | Minimum 15 new tests across the wave |
| Scope compliance | Zero non-goals violated |
