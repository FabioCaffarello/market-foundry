# Stage S452A — Operational History & Explainability Charter Report

**Stage**: S452A
**Type**: Charter and Scope Freeze
**Status**: COMPLETE
**Date**: 2026-03-24
**Wave**: Operational History & Explainability (S452A–S452E)
**Predecessor**: S451 (GO/NO-GO Decision — Stabilization Authorized)

---

## 1. Executive Summary

S452A opens the Operational History & Explainability wave — a focused, short track that runs parallel to Live Session Stabilization. The wave addresses structural weaknesses exposed by S449/S450 that are independent of live session execution: incomplete persistence, ambiguous read surfaces, absent session metadata, and unautomated post-session verification.

The wave is scoped to **four implementation stages plus one evidence gate**, uses only existing infrastructure (KV, ClickHouse, NATS request-reply), and requires no new API keys, live sessions, or exchange connectivity.

**Key decision**: This wave must complete before or alongside S453 (second live session) to ensure that the next session's operational history is fully captured and explainable.

---

## 2. Problem Analysis (Post-S451 State)

### 2.1 What S449/S450 Proved

The first supervised live session (S449) and post-live review (S450) demonstrated:

- Mainnet real data ingestion via `wss://stream.binance.com` (1500–4000 trades/min).
- Full pipeline processing: candle → signal → decision → strategy → risk → intent.
- Kill-switch operational (PS-1 cycle test PASS, session halt PASS, 4 intents blocked after halt).
- All 9 safety mechanisms active.
- Operator session control functional (start, monitor, halt).

### 2.2 What S449/S450 Exposed

| Finding | Severity | Description |
|---------|----------|-------------|
| F3 | MEDIUM | 12 ClickHouse records vs 24 expected venue fills — 50% persistence gap |
| F4 | MEDIUM | `type=paper_order` in records despite live adapter — type confusion |
| F5 | MEDIUM | `status=submitted` instead of `accepted/filled` — status stuck at derive-side |
| F7 | MEDIUM | Only 2 of 9 PO checks executed — verification protocol not automated |
| F10 | MEDIUM | 11 min manual debugging, 5 infrastructure issues undocumented |

### 2.3 Root Cause Classification

These findings share a common root: **the system can execute but cannot explain what it did**. The execution path works; the read/audit/verification path has gaps.

This is not a live-session problem — it is an **operational memory** problem that degrades confidence in any session's data integrity.

---

## 3. Wave Charter

### 3.1 Objective

Strengthen the operational memory of the system by consolidating historical read models, query ergonomics, session explainability surfaces, and cross-surface consistency.

### 3.2 Capabilities

| ID | Capability | Description |
|----|-----------|-------------|
| C1 | Persistence Completeness Invariant | Every KV key must have a corresponding ClickHouse record |
| C2 | Type/Status Disambiguation | Live vs paper vs dry-run unambiguous in all queries |
| C3 | Session Metadata Persistence | Session entity with ID, timestamps, config, operator, outcome |
| C4 | Order Narrative Query | Full lifecycle trace from signal to fill via correlation chain |
| C5 | List Query Ergonomics | Filtering by time, status, segment, mode; summary aggregations |
| C6 | KV-to-ClickHouse Consistency Audit | Automated divergence detection and reporting |
| C7 | Post-Session Verification Automation | All 9 PO checks executable as automated validations |

### 3.3 Governing Questions

| ID | Question | Target Stage |
|----|----------|-------------|
| Q1 | Why did 50% of execution events fail to reach ClickHouse? | S452B |
| Q2 | Can read surfaces distinguish live from paper execution? | S452B |
| Q3 | Can an operator reconstruct the full lifecycle of any intent? | S452D |
| Q4 | Can the system detect KV/ClickHouse divergence? | S452E |
| Q5 | Can post-session verification run without manual intervention? | S452E |
| Q6 | Does session-level metadata exist as queryable state? | S452D |

---

## 4. Non-Goals (Explicit)

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG1 | Broad dashboards or visualization UI | Data correctness first, presentation later |
| NG2 | New observability platform (Grafana, Prometheus) | Existing KV + ClickHouse sufficient |
| NG3 | OMS expansion (new order types/states) | OMS Foundation (S382–S388) is stable |
| NG4 | Multi-exchange support | Binance-only per existing scope |
| NG5 | Mainnet/live expansion or new sessions | Parallel Live Session Stabilization track |
| NG6 | Structural redesign of storage/runtime | Uses existing architecture |
| NG7 | Real-time streaming dashboards | Post-hoc query surfaces only |
| NG8 | Automated alerting or paging | Data foundation first |
| NG9 | Fee/commission model changes | S428 fee normalization is stable |
| NG10 | External API endpoints | Internal operational review only |

---

## 5. Stage Sequence

```
S452A  Charter and Scope Freeze           ← THIS STAGE (COMPLETE)
  │
  └─► S452B  Historical Execution Read Model
        │      - F3 root cause investigation
        │      - Persistence completeness invariant
        │      - Type/status disambiguation (F4, F5)
        │
        ├─► S452C  Operational List Queries and Retrieval
        │            - Filter/summary queries
        │            - Session-window queries
        │            - ClickHouse query equivalents
        │
        └─► S452D  Session Explainability Surface
                     - Session metadata model + KV persistence
                     - Order narrative query (correlation chain)
                     - Full lifecycle trace
                     │
                     └─► S452E  Evidence Gate
                                  - KV-to-ClickHouse consistency audit
                                  - PO check automation (all 9 checks)
                                  - Wave closure
```

**Parallelism**: S452C and S452D are independent after S452B and can execute concurrently.

---

## 6. Alignment with Existing Waves

| Prior Wave | Stages | Relationship |
|------------|--------|-------------|
| OMS Foundation | S382–S388 | This wave extends the read side; OMS write side untouched |
| Unified Segment Runtime | S398–S403 | Segment isolation preserved; queries add segment filter |
| Testnet Venue Execution (Spot) | S404–S409 | Read path audit (S407) extended with ergonomic queries |
| Production Readiness Hardening | S410–S414 | S413 lifecycle queryability is the primary extension point |
| Futures Venue Execution | S421–S426 | Read path audit (S424) extended in parallel with spot |
| Production Hardening | S427–S431 | Fee normalization (S428) queried but not modified |
| Live Trading Authorization | S438–S443 | Post-session protocol (S447) automated |
| Live Trading Enablement | S444–S448 | S447 PO checks codified as executable validations |

---

## 7. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| F3 root cause reveals writer pipeline bug | Medium | High | Writer pipeline is well-tested (S385); fix is surgical |
| Session metadata scope creeps into session management | Low | Medium | Scope freeze: metadata persistence only, no orchestration |
| Consistency audit reveals widespread divergence | Low | High | Treat as blocking; fix before gate |
| Queries require ClickHouse schema changes | Medium | Low | New columns allowed; new tables discouraged |
| Wave delays second live session | Low | Medium | Wave runs parallel; S453 can proceed if S452B closes F3 |

---

## 8. Deliverables Produced

| Deliverable | Path | Status |
|-------------|------|--------|
| Wave Charter and Scope Freeze | `docs/architecture/operational-history-and-explainability-wave-charter-and-scope-freeze.md` | DELIVERED |
| Capabilities, Questions, and Non-Goals | `docs/architecture/operational-history-explainability-capabilities-questions-and-non-goals.md` | DELIVERED |
| Stage Report (this document) | `docs/stages/stage-s452a-operational-history-explainability-charter-report.md` | DELIVERED |

---

## 9. Preparation for S452B

### Recommended Pre-Work

1. **Read the writer pipeline**: `internal/adapters/clickhouse/writerpipeline/support.go` — understand batching, flush, and error handling to hypothesize F3 root cause.
2. **Read the KV projection actors**: `internal/actors/scopes/store/` — understand which events are projected to which KV buckets.
3. **Read the NATS consumer wiring**: `internal/adapters/nats/natsexecution/consumer.go` — understand which subjects are consumed and how events reach the writer.
4. **Query S449 ClickHouse data**: Count records by type, status, and timestamp to characterize the F3 gap precisely.
5. **Read S447 PO protocol**: `docs/architecture/post-session-operational-verification.md` — understand all 9 checks that need automation.

### S452B Entry Criteria

- S452A charter accepted (this document).
- Access to ClickHouse with S449 data still present.
- Writer pipeline source code reviewed.

### S452B Exit Criteria

- F3 root cause documented with evidence.
- Persistence completeness invariant implemented and passing.
- Type/status disambiguation implemented and verified against S449 data.
- Questions Q1 and Q2 answered.

---

## 10. Verdict

**S452A: COMPLETE**

The Operational History & Explainability wave is formally open with scope frozen. The problem is clearly defined (operational memory deficit, not execution deficit), non-goals are explicit, stages are ordered with dependencies, and preparation for S452B is documented.

The wave bridges the gap from "the system can execute" (proven by S449) to "the system can explain what it executed" (required before S453).
