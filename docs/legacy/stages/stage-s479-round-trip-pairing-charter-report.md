# Stage S479 -- Round-Trip Pairing Charter Report

**Stage**: S479
**Type**: Charter and Scope Freeze
**Status**: COMPLETE
**Date**: 2026-03-26
**Wave**: Round-Trip Pairing (S479--S483)
**Predecessor**: S478 (Strategy Effectiveness Evidence Gate -- PASS)

---

## 1. Executive Summary

S479 opens the Round-Trip Pairing wave in response to the most material residual gap from the Strategy Effectiveness Measurement wave: **G-SE1 (MEDIUM) -- single-leg fills dominate outcomes, making most effectiveness evaluations return `unresolved`.**

The wave is short (4 stages), tightly scoped (6 capabilities, 5 governing questions, 18 non-goals), and operates entirely on existing data. It transforms unresolved single-leg outcomes into paired round-trips by defining canonical matching rules, implementing automated entry/exit leg pairing from ClickHouse data, integrating paired outcomes into the effectiveness attribution pipeline, and providing reconciliation surfaces.

The `ClassifyPair(entry, exit)` function from S476 already handles round-trip P&L computation. This wave provides the matching infrastructure that feeds it.

**Scope is frozen.** No OMS expansion, no portfolio engine, no position/risk tracking, no multi-exchange, no dashboards, no write-path changes.

---

## 2. Problem Analysis

### 2.1 The single-leg dominance problem

The execution pipeline processes individual orders. Each `ExecutionIntent` goes through the lifecycle independently: submitted → accepted → filled (or rejected/cancelled). The effectiveness classifier (`Classify()`) receives one intent at a time. For a filled buy order with no corresponding sell, the outcome is `unresolved` -- there is no way to compute P&L without an exit.

This is not a bug. It is a structural consequence of how the pipeline was designed for safety and auditability. But it means:

- **Win rate is computed on a small subsample.** Only chains where both entry and exit happen to be in the evaluation window get resolved.
- **P&L attribution is incomplete.** Unresolved chains carry `entry_cost_basis` and `total_fees` but no realized P&L.
- **Comparative analysis is underpowered.** Cohort comparisons on thin resolved samples may be noise.

### 2.2 What already exists

The foundation for solving this is already in place:

| Asset | Location | What it does |
|-------|----------|-------------|
| `ClassifyPair(entry, exit)` | `internal/domain/effectiveness/effectiveness.go:161-214` | Computes gross/net P&L, classifies win/loss/breakeven for a matched pair |
| `FillRecord` with all fields | `internal/domain/execution/` | Price, Quantity, Fee, FeeAsset, CostBasis per fill |
| CorrelationID on all events | Pipeline-wide | Links decisions to their execution outcomes |
| CompositeReader | `internal/application/analyticalclient/` | Reads execution chains from ClickHouse |
| Batch effectiveness endpoints | S476, S477 | 3 HTTP endpoints consuming `Attribution` records |
| `DecisionReviewBundle` | S471, S476 | Extensible review surface with effectiveness section |

### 2.3 What this wave adds

| New capability | How it uses existing assets |
|---------------|---------------------------|
| `RoundTrip` domain type | Wraps two `ExecutionIntent` references with matching metadata |
| FIFO leg-matching | Pairs entries/exits by symbol + segment + correlation scope + temporal order |
| Pairing read model | Uses `CompositeReader` to fetch candidates, applies matching, feeds `ClassifyPair()` |
| Paired batch effectiveness | Extends existing batch pipeline with paired attributions |
| Pairs endpoint | New HTTP surface following existing gateway patterns |
| Reconciliation surface | Lists unmatched legs with reason codes |

---

## 3. Wave Structure

### 3.1 Blocks and stages

| Block | Stage | Name | Scope |
|-------|-------|------|-------|
| 1 | S480 | Canonical round-trip and leg-pairing model | Domain types, matching rules, FIFO strategy, partial-fill handling, invariants, tests |
| 2 | S481 | Pairing read model and attribution integration | ClickHouse read path, automated matching, `ClassifyPair()` wiring, batch integration, resolved rate metric, tests |
| 3 | S482 | Round-trip review and outcome reconciliation | Pairs HTTP endpoint, reconciliation surface, review bundle extension, unmatched reason codes, tests |
| 4 | S483 | Evidence gate | Formal assessment, evidence matrix, residual gaps, wave verdict |

### 3.2 Dependency chain

```
S480 (domain model) → S481 (read model + integration) → S482 (HTTP + reconciliation) → S483 (gate)
```

Strictly sequential. Each block depends on the previous block's domain types or read model.

---

## 4. Governing Questions

| ID | Question | What PASS looks like |
|----|----------|---------------------|
| Q-RT1 | Can the system identify and pair entry/exit legs from existing data? | Canonical matching rules implemented and tested. FIFO produces deterministic pairs. |
| Q-RT2 | Does pairing increase the resolved rate? | Batch evaluation shows fewer `unresolved` outcomes with pairing enabled. |
| Q-RT3 | Are paired outcomes correctly classified with accurate P&L? | `ClassifyPair()` integration tested with fee impact. Domain tests pass for all scenarios. |
| Q-RT4 | Can the system surface paired outcomes and flag unmatched legs? | HTTP endpoint returns pairs and unmatched legs with reason codes. |
| Q-RT5 | Is pairing computable from existing data without new infrastructure? | No new ClickHouse tables, no new exchange connectivity, no OMS changes. |

---

## 5. Non-Goals (Explicit)

The following are **frozen out** of this wave:

| ID | Non-Goal | Why excluded |
|----|----------|-------------|
| NG-RT1 | OMS expansion | Pairing reads outcomes; it does not manage orders |
| NG-RT2 | Position / risk engine | Historical outcomes, not ongoing exposure |
| NG-RT3 | Portfolio P&L aggregation | Per-symbol only; portfolio is a separate wave |
| NG-RT4 | Real-time pairing | Read-path only; no streaming |
| NG-RT5 | Multi-exchange pairing | Single-venue (Binance) only |
| NG-RT6 | Cross-session pairing | Correlation-ID scope only |
| NG-RT7 | Advanced matching (LIFO/HIFO) | FIFO only |
| NG-RT8 | UI / dashboards | HTTP endpoints only |
| NG-RT9 | Risk-adjusted metrics | Raw P&L and win/loss only |
| NG-RT10 | New ClickHouse tables | Computed from existing data |
| NG-RT11 | Write-path changes | Read-path extension only |
| NG-RT12 | Slippage analysis | Fill prices only |
| NG-RT13 | Strategy expansion | Pairs existing executions |
| NG-RT14 | ML / predictive scoring | Deterministic classification |
| NG-RT15 | Advanced derivatives | No funding rates, no liquidation |
| NG-RT16 | Alerting / notification | No threshold triggers |
| NG-RT17 | Benchmark comparison | Absolute outcomes only |
| NG-RT18 | Statistical significance | Raw counts and rates |

---

## 6. Guard Rails

10 guard rails enforced (see charter document for full text):

1. No OMS expansion
2. No new ClickHouse tables
3. No new exchange connectivity
4. No write-path changes
5. No portfolio analytics
6. No real-time streaming
7. No domain type refactoring (additive only)
8. No UI or dashboards
9. No risk/position engine
10. Additive only -- zero changes to existing behavior

---

## 7. Preparation Recommended for S480

Before starting S480 (canonical round-trip model), the following preparation is recommended:

1. **Audit existing execution data for pairing potential.** Query ClickHouse to understand how many sessions have both entry (buy) and exit (sell) fills for the same symbol/segment. This sets the baseline resolved rate expectation.

2. **Review `ClassifyPair()` edge cases.** The function exists and is tested for basic scenarios (long win/loss, short win, breakeven, rejected). Confirm that partial-fill quantity handling is compatible with the planned proportional matching.

3. **Review `CompositeReader` query capabilities.** The pairing read model needs to fetch execution intents grouped by symbol/segment/correlation scope. Confirm that existing query patterns support this grouping without new ClickHouse views.

4. **Identify the correlation-ID scope boundary.** Determine whether correlation-ID reliably connects entry and exit decisions within the same strategy cycle. This defines the matching scope for block 1.

---

## 8. Deliverables Produced

| Artifact | Type | Location |
|----------|------|----------|
| Wave charter and scope freeze | Architecture | [`round-trip-pairing-wave-charter-and-scope-freeze.md`](../architecture/round-trip-pairing-wave-charter-and-scope-freeze.md) |
| Capabilities, questions, non-goals | Architecture | [`round-trip-pairing-capabilities-questions-and-non-goals.md`](../architecture/round-trip-pairing-capabilities-questions-and-non-goals.md) |
| S479 charter report (this document) | Stage report | `docs/stages/stage-s479-round-trip-pairing-charter-report.md` |

---

## 9. References

- [Wave Charter and Scope Freeze](../architecture/round-trip-pairing-wave-charter-and-scope-freeze.md)
- [Capabilities, Questions, and Non-Goals](../architecture/round-trip-pairing-capabilities-questions-and-non-goals.md)
- [S478 Evidence Gate Report](stage-s478-strategy-effectiveness-evidence-gate-report.md)
- [Effectiveness Evidence Matrix and Residual Gaps](../architecture/strategy-effectiveness-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Strategy Effectiveness Wave Charter](../architecture/strategy-effectiveness-measurement-wave-charter-and-scope-freeze.md)
- [Stages Index](INDEX.md)
