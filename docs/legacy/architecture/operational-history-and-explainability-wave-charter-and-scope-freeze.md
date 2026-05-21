# Operational History & Explainability Wave — Charter and Scope Freeze

**Wave**: Operational History & Explainability
**Stages**: S452A–S452E
**Status**: OPEN — Scope Frozen
**Date**: 2026-03-24
**Predecessor**: S451 (GO/NO-GO Decision — Stabilization Authorized)
**Parallel to**: Live Session Stabilization track (S452–S455)

---

## 1. Problem Statement

The S449 supervised live session and S450 post-live review exposed a structural weakness that is independent of whether the next live session succeeds:

- **Persistence completeness gap (F3)**: 12 ClickHouse records vs 24 expected venue fills — 50% of execution history silently missing.
- **Status/type confusion (F4, F5)**: Records show `type=paper_order` and `status=submitted` despite live adapter operation — the read surface cannot distinguish live from paper execution.
- **Post-session verification incompleteness (F7)**: Only 2 of 9 PO checks executed — the verification protocol exists but the system cannot self-verify.
- **Infrastructure friction undocumented (F10)**: 11 min of manual debugging across 5 issues with no recorded runbook.

These are not live-session bugs. They are **explainability and operational history deficits** that degrade confidence in any session — past, present, or future.

### Why This Wave Exists Separately

The Live Session Stabilization track (S452–S455) targets the next real order submission. This wave targets the ability to **understand, audit, and explain** what any session did, regardless of whether real orders were placed.

No new API keys, no new live sessions, no new exchange connectivity required.

---

## 2. Wave Objective

Strengthen the operational memory of the system by consolidating historical read models, query ergonomics, session explainability surfaces, and cross-surface consistency — using only capabilities already deployed.

### Success Definition

After this wave closes, an operator can:

1. List all past sessions with their configuration, duration, and outcome.
2. Trace any execution intent from signal through fill (or rejection) to persistence in a single query.
3. Identify which intents reached KV but not ClickHouse (persistence audit).
4. Review all safety events (kill-switch activations, halts) with timestamps and context.
5. Run a pre-flight validation that catches the 5 infrastructure issues from S449 before they block a session.

---

## 3. Wave Blocks

### Block 1 — Historical Execution/Lifecycle Read Model (S452B)

**Problem**: No unified historical view of execution lifecycle. KV stores latest-only state. ClickHouse has records but with unexplained gaps (F3) and type/status confusion (F4, F5).

**Scope**:
- Root-cause investigation of F3 persistence gap (12 vs 24 records).
- Audit ClickHouse write path for venue fill events — identify where events are dropped or filtered.
- Establish persistence completeness invariant: every KV key must have a corresponding ClickHouse record.
- Fix type/status disambiguation so live execution records are distinguishable from paper/dry-run.

**Builds on**: S385 (write path by mode), S387 (lifecycle persistence), S411 (rejection persistence), S413 (lifecycle queryability).

### Block 2 — Operational List Queries and Retrieval Ergonomics (S452C)

**Problem**: Existing queries work for individual lookups but lack ergonomic list/filter/summary capabilities for operational review.

**Scope**:
- Extend `LifecycleListQuery` with filtering by time range, status, segment, and execution mode.
- Add summary aggregation: count by status, count by segment, total fills vs total intents.
- Ensure ClickHouse queries support the same filter dimensions as KV list queries.
- Provide a "what happened in session X" composite query that returns all intents, fills, rejections, and safety events for a time window.

**Builds on**: S413 (lifecycle list queries), S407/S418 (read path audit per segment).

### Block 3 — Session Explainability Surface (S452D)

**Problem**: No session-level audit trail exists. S449 session metadata (start time, stop time, config, operator, outcome) lives only in human-written documents, not in queryable system state.

**Scope**:
- Define session metadata model: session ID, start/stop timestamps, config hash, operator identifier, outcome summary.
- Persist session metadata to a dedicated KV bucket at session start/stop.
- Provide "full story of order X" narrative query: signal → decision → strategy → risk → intent → fill/rejection → ClickHouse record.
- Provide correlation chain query using existing `correlation_id` linking across ClickHouse tables.

**Builds on**: S383 (canonical order model), S384 (lifecycle invariants), S387 (price source wiring).

### Block 4 — Cross-Surface Consistency Audit (S452E-pre)

**Problem**: KV and ClickHouse may diverge silently. F3 proved this happens. No automated check exists to detect or report divergence.

**Scope**:
- Implement KV-to-ClickHouse consistency check: enumerate all KV keys, verify each has a ClickHouse record.
- Report missing records, stale records, and status mismatches.
- Integrate consistency check into post-session verification protocol (PO checks).
- Codify the 9 PO checks from S447 as executable validations, not just documentation.

**Builds on**: S447 (post-session verification), S413 (lifecycle read surfaces).

### Block 5 — Evidence Gate (S452E)

**Problem**: The wave must close with a formal gate proving all four blocks delivered their stated capabilities.

**Scope**:
- Verify persistence completeness invariant holds for historical S449 data.
- Verify list/filter queries return correct results for known session data.
- Verify session metadata is queryable.
- Verify KV-to-ClickHouse consistency check finds zero divergence on clean data.
- Verify all 9 PO checks are executable and pass on known-good state.

---

## 4. Scope Freeze

### What Is IN Scope

| Item | Rationale |
|------|-----------|
| F3 root cause investigation | Blocking — cannot trust persistence without this |
| F4/F5 type/status disambiguation | Blocking — read surfaces are misleading |
| F7 PO check automation | Blocking — verification protocol is manual-only |
| F10 infrastructure friction documentation | Blocking — operator cannot self-serve setup |
| KV-to-ClickHouse consistency audit | Core deliverable — catches silent divergence |
| Session metadata model and persistence | Core deliverable — enables session-level queries |
| Correlation chain query | Core deliverable — enables order narrative |
| List query ergonomics (filter, summary) | Core deliverable — enables operational review |

### What Is NOT In Scope

| Item | Rationale |
|------|-----------|
| New dashboards or visualization UI | Out of scope — this wave targets queryable data, not presentation |
| New observability platform (Grafana, Prometheus) | Out of scope — existing NATS/ClickHouse surfaces sufficient |
| OMS expansion (new order types, new lifecycle states) | Out of scope — OMS Foundation (S382–S388) is stable |
| Multi-exchange support | Out of scope — Binance-only per existing scope |
| Mainnet/live expansion or new sessions | Out of scope — parallel Live Session Stabilization track |
| Structural redesign of storage or runtime | Out of scope — uses existing KV + ClickHouse + NATS |
| Real-time streaming dashboards | Out of scope — post-hoc query surfaces only |
| Fee/commission model changes | Out of scope — S428 fee normalization is stable |
| New API endpoints or external interfaces | Out of scope — internal query surfaces only |
| Automated alerting or paging | Out of scope — operational, not monitoring |

### Scope Inflation Guard

Any proposal that requires:
- New exchange credentials or API keys → REJECTED
- New compose services or infrastructure → REJECTED
- Changes to the execution/submission path → REJECTED
- New domain events or NATS subjects → evaluated case-by-case (session metadata may need one)
- Changes to ClickHouse schema → evaluated case-by-case (new columns allowed, new tables discouraged)

---

## 5. Stage Sequence

| Stage | Name | Dependency | Focus |
|-------|------|------------|-------|
| **S452A** | Charter and Scope Freeze | S451 | This document — wave opened |
| **S452B** | Historical Execution Read Model | S452A | F3 root cause, persistence invariant, type/status fix |
| **S452C** | Operational List Queries and Retrieval | S452B | Filter/summary queries, session-window queries |
| **S452D** | Session Explainability Surface | S452B | Session metadata, narrative query, correlation chain |
| **S452E** | Evidence Gate | S452C + S452D | Consistency audit, PO automation, wave closure |

### Dependency Graph

```
S452A (charter)
  └─► S452B (read model + F3 fix)
        ├─► S452C (list queries) ──┐
        └─► S452D (explainability) ─┤
                                    └─► S452E (gate)
```

S452C and S452D are independent after S452B completes and can execute in parallel.

---

## 6. Alignment with Existing Capabilities

| Capability | Wave | Status | How This Wave Uses It |
|------------|------|--------|----------------------|
| Canonical Order Model | S383 | Stable | Session metadata extends, does not modify |
| Lifecycle Invariants | S384 | Stable | Persistence invariant adds to existing set |
| Write Path by Mode | S385 | Stable | F3 investigation reads this path, does not change it |
| Rejection Event Path | S386 | Stable | Consistency audit covers rejection records |
| Lifecycle Persistence | S387 | Stable | Read model builds on existing KV/ClickHouse wiring |
| Read Path Audit (Spot) | S407 | Stable | Query ergonomics extend existing routes |
| Rejection Persistence | S411 | Stable | Consistency audit covers rejection persistence |
| Lifecycle Queryability | S413 | Stable | List queries extend S413 LifecycleListQuery |
| Fee Normalization | S428 | Stable | Fee queries use existing normalized model |
| Post-Session Verification | S447 | Stable | PO automation codifies S447 protocol |

---

## 7. Exit Criteria

This wave is complete when:

1. F3 root cause is documented and persistence gap is closed or explained.
2. Type/status disambiguation is implemented — live vs paper is unambiguous in queries.
3. Session metadata is persisted and queryable for S449 (retroactively) and future sessions.
4. Correlation chain query reconstructs full order narrative from ClickHouse.
5. KV-to-ClickHouse consistency check reports zero divergence on clean state.
6. All 9 PO checks from S447 are executable as automated validations.
7. Evidence gate (S452E) passes with all checks green.

---

## 8. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| F3 root cause is in the writer pipeline, requiring code changes | Medium | Writer pipeline is well-tested (S385); changes are surgical |
| Session metadata model scope creeps into full session management | Medium | Scope freeze: metadata only, no session orchestration |
| Consistency audit reveals widespread KV/ClickHouse divergence | High | Treat as blocking finding; document and fix before gate |
| Query ergonomics pull toward building a UI | Low | Guard rail: queryable data only, no presentation layer |

---

## 9. Timeline Expectation

This is a short wave. Each block is scoped to 1–2 stages of focused work on existing surfaces.

- S452B: Investigation + fix (highest risk, highest value)
- S452C + S452D: Parallel query/explainability work (moderate effort)
- S452E: Gate (verification only)

The wave should complete before or alongside the Live Session Stabilization track reaching S453 (second live session), ensuring that the next session has proper operational history and explainability infrastructure.
