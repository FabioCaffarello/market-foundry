# Session Intelligence & Operational Automation Wave -- Charter and Scope Freeze

**Wave**: Session Intelligence & Operational Automation
**Charter Stage**: S459
**Date**: 2026-03-24
**Predecessor**: S456A (Operational History & Explainability Evidence Gate -- WAVE CLOSED, SUBSTANTIALLY COMPLETE)
**Parallel Track**: S457 (Second Supervised Live Session Wave -- charter open, execution pending operator availability)

---

## 1. Strategic Context

The Foundry roadmap maintains the live session track (S457--S460) as a standing priority. However, evolution of the system must not be blocked by API keys, operator availability, or market timing. S456A closed with two clear signals of useful structural work that is **independent of live execution**:

- **C3 (Session Metadata Persistence) = PARTIAL** -- no first-class session entity; Q6 NOT YET answered.
- **C7 (Post-Session Verification Automation) = PARTIAL** -- no automated PO check harness; Q5 NOT YET answered.

These are not correctness gaps -- the system can execute and explain execution. They are **operational maturity gaps** that increase manual burden, reduce auditability, and make session-level reasoning fragile.

This wave transforms them into durable capabilities without depending on a live session, without expanding scope, and without touching the execution path.

---

## 2. Problem Statement

### 2.1 What the System Can Do Today

After S452A--S456A, the system has:

- 5 new HTTP endpoints (lifecycle, list, summary, lifecycle/list, explain).
- 48 new tests covering query builder, use case, and explain/consistency layers.
- Field-level consistency audit across KV and ClickHouse (15 fields audited).
- Type/status disambiguation (F4/F5 CLOSED).
- Per-key divergence detection via explain endpoint.

### 2.2 What the System Cannot Do Today

| Gap | ID | Impact |
|-----|------|--------|
| No first-class session entity | G1 | Cannot query "what happened in session X" as a unit; session scoping is implicit |
| No batch KV-to-CH consistency audit | G2 | Divergence detection is per-key only; no automated sweep |
| No automated PO check harness | G3 | All 9 post-session checks require manual execution against HTTP endpoints |
| No session audit bundle | -- | No single artifact that captures session metadata + PO results + explain output |

### 2.3 Root Cause

The system was built execution-first. Session is an implicit concept -- a time window during which the operator ran the system. There is no entity, no persistence, no structured verification pipeline, and no consolidated audit surface for sessions as a unit.

This is the gap this wave closes.

---

## 3. Wave Objective

Transform the operational session from an implicit time window into a **first-class system entity** with:

1. Explicit metadata (ID, timestamps, config snapshot, operator, outcome, halt reason).
2. Automated verification pipeline (all 9 PO checks as a single executable command).
3. Consolidated audit bundle (session metadata + verification results + explain output).
4. Evidence gate confirming Q5 and Q6 are answered.

---

## 4. Wave Blocks

```
S459  Charter and Scope Freeze                    <-- THIS STAGE
  |
  +---> S460  Canonical Session Metadata Model and Persistence
  |             - Session entity definition
  |             - KV persistence (session bucket)
  |             - HTTP query surface (session by ID, list sessions)
  |             - Q6 answered
  |
  +---> S461  PO Automation and Verification Pipeline
  |             - Codify all 9 PO checks as executable validations
  |             - Batch KV-to-CH consistency audit (G2)
  |             - Single-command verification harness
  |             - Q5 answered
  |
  +---> S462  Session Audit Bundle and Explainability Surface
  |             - Consolidated audit endpoint or script
  |             - Session metadata + PO results + explain output
  |             - Structured bundle for post-session review
  |
  +---> S463  Session Intelligence Evidence Gate
                - All capabilities graded
                - Q5 and Q6 formally answered
                - Wave closure verdict
```

### Dependency Model

- S460 and S461 are independent and can execute in parallel after S459.
- S462 depends on both S460 and S461 (it combines their outputs).
- S463 depends on S462.

```
S459 --+--> S460 --+--> S462 --> S463
       |           |
       +--> S461 --+
```

---

## 5. Capability Definitions

| ID | Capability | Description | Target Grade |
|----|-----------|-------------|-------------|
| C3+ | Session Metadata Persistence | First-class session entity with ID, timestamps, config snapshot, operator, outcome, halt reason; persisted in KV; queryable via HTTP | FULL |
| C7+ | Post-Session Verification Automation | All 9 PO checks codified as executable validations; single-command harness; structured output | FULL |
| C8 | Batch Consistency Audit | Automated KV-to-CH divergence sweep across all keys in a session window | SUBSTANTIAL |
| C9 | Session Audit Bundle | Consolidated artifact combining session metadata, PO results, and explain output | SUBSTANTIAL |

---

## 6. Governing Questions

| ID | Question | Target Stage |
|----|----------|-------------|
| Q5 | Can post-session verification run without manual intervention? | S461 |
| Q6 | Does session-level metadata exist as queryable state? | S460 |
| Q7 | Can the system produce a single consolidated audit artifact for any session? | S462 |
| Q8 | Does the batch consistency audit detect divergences that per-key checking misses? | S461 |
| Q9 | Can the operator review a session's full operational history without touching multiple endpoints manually? | S462 |

---

## 7. Alignment with Existing Infrastructure

This wave uses **only existing infrastructure**:

| Infrastructure | Usage | Changes |
|---------------|-------|---------|
| NATS KV | Session metadata bucket (new bucket, existing mechanism) | New bucket only |
| NATS Request-Reply | Session queries via existing gateway pattern | New handler, existing pattern |
| ClickHouse | Queried for consistency audit; no schema changes | Read-only |
| HTTP Gateway | New endpoints following existing analytical pattern | Additive routes |
| Scripts | PO harness as shell script or Go test harness | New script |

No new external dependencies. No new services. No new databases.

---

## 8. Scope Freeze

### 8.1 What Is In Scope

1. Session entity definition and KV persistence.
2. Session HTTP query endpoints (get by ID, list sessions).
3. Codification of all 9 PO checks as automated validations.
4. Batch KV-to-CH consistency audit.
5. Session audit bundle (consolidated artifact).
6. Evidence gate with formal Q5/Q6 closure.

### 8.2 What Is NOT In Scope (Non-Goals)

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG1 | New supervised live session | Parallel S457 track; this wave is independent |
| NG2 | Spot Scope Expansion | Blocked by S451; requires second session + GO/NO-GO |
| NG3 | Futures live execution | Out of scope per standing freeze |
| NG4 | OMS expansion (new order types, states, or lifecycle changes) | OMS Foundation (S382--S388) is stable and untouched |
| NG5 | Broad dashboards or visualization UI | Data correctness and automation first |
| NG6 | Multi-exchange support | Binance-only per existing scope |
| NG7 | Structural redesign of storage or runtime | Uses existing KV + CH + NATS architecture |
| NG8 | Real-time streaming or push alerting | Post-hoc verification and query surfaces only |
| NG9 | Automated session orchestration (auto-start, auto-halt) | Session metadata is passive; operator controls session lifecycle |
| NG10 | Config or compose changes | Existing deployment topology preserved |
| NG11 | Performance optimization or pagination | Future wave (G6 from S456A) |
| NG12 | Cross-domain lifecycle trace (signal-to-fill) | Out of scope per S452A charter |

**Scope is frozen. No additions permitted without a new charter.**

---

## 9. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Session metadata scope creeps into session orchestration | Low | Medium | Scope freeze: metadata persistence and query only; no lifecycle management |
| PO harness requires changes to existing endpoints | Low | Low | PO checks consume existing HTTP surfaces; no modifications needed |
| Batch consistency audit reveals widespread divergence | Low | High | Document findings; do not block wave on fixing all divergences |
| Session entity design over-engineers for future requirements | Medium | Medium | Minimal entity: only fields derivable from current system state |
| Wave delays second live session | NONE | -- | Wave is explicitly independent; S458 can proceed whenever operator is ready |

---

## 10. Success Criteria

| Criterion | Measure |
|-----------|---------|
| Q5 answered | PO harness runs all 9 checks without manual intervention |
| Q6 answered | Session entity persisted in KV and queryable via HTTP |
| Q7 answered | Single command or endpoint produces consolidated audit bundle |
| No regression | All 334+ existing tests pass; zero failures |
| Scope discipline | No infrastructure additions beyond new KV bucket and HTTP endpoints |

---

## 11. Relationship to S457 (Second Live Session)

These waves are **parallel and independent**:

| Dimension | S457 Wave (Live Session) | S459 Wave (Session Intelligence) |
|-----------|-------------------------|----------------------------------|
| Trigger | Operator availability + API keys + market timing | Code changes only |
| Dependency | Requires live exchange connectivity | Requires nothing external |
| Value | Proves real order submission | Makes any session auditable and verifiable |
| Risk | Market/infrastructure risk | Zero operational risk |
| Can proceed now | Only when operator is ready | Immediately |

When both waves complete, the second live session (S458) will benefit from session metadata persistence and automated PO verification -- a strictly better operational posture.

---

## 12. Preparation for S460

### Recommended Pre-Work

1. **Design session entity**: Define fields, KV bucket name, key format, and serialization.
2. **Identify session lifecycle events**: What triggers session start/end in the current system (compose up/down, kill-switch halt, operator command).
3. **Read existing KV patterns**: `internal/adapters/nats/natsexecution/kv_store.go` -- understand bucket creation and key conventions.
4. **Read existing HTTP handler patterns**: `internal/interfaces/http/handlers/analytical.go` -- understand handler/route/composition patterns.
5. **Read S447 PO protocol**: `docs/architecture/post-session-operational-verification.md` -- enumerate all 9 checks for automation.

### S460 Entry Criteria

- S459 charter accepted (this document).
- Session entity fields agreed.
- KV bucket naming convention decided.

### S460 Exit Criteria

- Session entity persisted in KV with all defined fields.
- HTTP endpoint returns session by ID.
- HTTP endpoint lists sessions.
- Q6 formally answered with test evidence.
