# Stage S277 — Live Analytical Execution Proof — Report

> Date: 2026-03-21
> Status: **CLOSED**
> Scope: Prove the live, reproducible analytical round-trip for execution paper orders.

## Executive Summary

S277 delivers a deterministic, self-contained integration test that exercises the **real** ClickHouse write→read cycle for execution paper orders. The test proves that `PaperOrderSubmittedEvent` data written via `InsertBatch` survives the full analytical path and is queryable via `QueryExecutionHistory` with field-level coherence across all 16 SELECT columns, optional filters (side, status, since, until), and JSON columns (risk, fills, parameters, metadata).

This closes the gap between S272 (serialization-only) and the infrastructure-level smoke test, giving the Foundry a **reproducible, auditable proof of analytical persistence** that does not depend on docker-compose orchestration.

## Objective

Validate the path:

```
execution event → writer mapper → ClickHouse INSERT → ClickHouse SELECT → domain objects
```

with real persistence and real queries against a live ClickHouse instance.

## Deliverables

| # | Artifact | Path | Status |
|---|----------|------|--------|
| 1 | Live integration test | `internal/adapters/clickhouse/live_execution_analytical_test.go` | Done |
| 2 | Architecture document | `docs/architecture/live-analytical-execution-proof.md` | Done |
| 3 | Queryability findings | `docs/architecture/analytical-execution-queryability-findings.md` | Done |
| 4 | Stage report | `docs/stages/stage-s277-live-analytical-execution-proof-report.md` | This file |

## Validated Analytical Path

```
┌──────────────────────────────────────────────────────────────────┐
│ Test Fixture (PaperOrderSubmittedEvent-equivalent rows)          │
│ 3 events: btcusdt/60/buy, btcusdt/60/sell, ethusdt/300/buy      │
└──────────────┬───────────────────────────────────────────────────┘
               │ InsertBatch (real ClickHouse native protocol)
               ▼
┌──────────────────────────────────────────────────────────────────┐
│ ClickHouse: executions table                                     │
│ MergeTree, 20 columns, partition by toYYYYMM(timestamp)          │
└──────────────┬───────────────────────────────────────────────────┘
               │ QueryExecutionHistory (real parameterized SELECT)
               ▼
┌──────────────────────────────────────────────────────────────────┐
│ []ExecutionIntent (domain objects)                                │
│ Risk, Fills, Parameters, Metadata deserialized from JSON         │
└──────────────────────────────────────────────────────────────────┘
```

## Scenarios Proven

| ID | Scenario | Result |
|----|----------|--------|
| LAE-1 | Basic queryability (write → read) | Pass |
| LAE-2 | All 16 SELECT columns survive round-trip | Pass |
| LAE-3 | Side and status filter narrowing | Pass |
| LAE-4 | Time-range filter (since/until) | Pass |
| LAE-5 | RiskInput JSON (incl. strategy_type, decision_severity) | Pass |
| LAE-6 | FillRecord array (price, qty, fee, simulated) | Pass |
| LAE-7 | Parameters and metadata map fidelity | Pass |
| LAE-8 | Multi-symbol partition isolation | Pass |
| LAE-9 | Full field-level coherence (emitted vs queried) | Pass |

## Files Changed

| File | Change |
|------|--------|
| `internal/adapters/clickhouse/live_execution_analytical_test.go` | New — live ClickHouse integration test (9 scenarios) |
| `docs/architecture/live-analytical-execution-proof.md` | New — proof architecture document |
| `docs/architecture/analytical-execution-queryability-findings.md` | New — queryability findings |
| `docs/stages/stage-s277-live-analytical-execution-proof-report.md` | New — this report |

## Key Evidence

1. **Real persistence**: Rows are inserted via `InsertBatch` using ClickHouse native protocol, not mocked.
2. **Real query execution**: `QueryExecutionHistory` constructs parameterized SQL and scans real ClickHouse rows.
3. **Field-level coherence**: Every field of the emitted fixture is compared against the queried result with appropriate tolerances (exact for strings, 1e-10 for floats, ≤1s for timestamps).
4. **Filter correctness**: Side, status, and time-range filters are proven to narrow results correctly.
5. **JSON round-trip**: Complex nested structures (RiskInput, FillRecord, maps) survive serialization and deserialization with no data loss.
6. **Partition isolation**: Events for different symbol/timeframe partitions are independently queryable with no cross-contamination.

## Remaining Limits

| Limit | Severity | Notes |
|-------|----------|-------|
| NATS consumer not in test loop | Low | Consumer path proven separately by `smoke-analytical-e2e.sh` |
| No aggregation queries | Low | Raw rows sufficient for current observability needs |
| Writer batch flush not tested | Low | `flush_interval=5s` timing proven by smoke test |
| No concurrent writer proof | Low | Single-writer assumption is operational convention |
| No sub-second time filtering | Low | Acceptable for execution event frequency |
| `ingested_at` not asserted | Info | DEFAULT column not in SELECT path |

## Acceptance Criteria Assessment

| Criterion | Met? |
|-----------|------|
| Live, reproducible proof of execution analytical path | Yes — LAE-1 through LAE-9 |
| Analytical trail has operational evidence beyond local serialization | Yes — real ClickHouse persistence and query |
| Observable envelope is more reliable | Yes — deterministic, self-contained test |
| Micro-wave closes with discipline | Yes — strict scope, no analytics expansion |

## Gate Recommendation

**S277 is CLOSED.** The execution analytical surface has live, reproducible proof of the write→read cycle with field-level coherence.

### Recommended next steps (post-S277)

1. **Transition gate assessment**: Evaluate whether the accumulated proofs (S270–S277) provide sufficient operational confidence for production readiness.
2. **Integration CI with ClickHouse service**: Enable the live test in CI by adding a ClickHouse service container and setting `CLICKHOUSE_DSN`.
3. **Remaining open debts**: KV materialization end-to-end (OD-PE3), control gate cross-binary propagation (OD-PE5), and missing S267 report (OD-PE2) remain open from the post-S273 matrix.
