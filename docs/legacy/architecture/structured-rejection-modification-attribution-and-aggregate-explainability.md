# Structured Rejection/Modification Attribution and Aggregate Explainability

> Stage: S298 — Structured Rejection/Modification Attribution
> Status: Complete
> Predecessor: S297 (HTTP Explainability Query Surface)

## Purpose

This document defines the structured attribution model for risk rejections, modifications, and approvals, and the aggregation queries that complete the wave's governing questions Q2, Q5, Q6, and Q7.

## Problem Statement

After S297, the composite chain already carried the full risk stage data (disposition, constraints, rationale, strategies). However:

1. **Q2 was only partially answerable** — operators had to manually traverse the nested risk stage to understand why an execution was rejected or modified.
2. **Q6 was unanswerable** — no aggregation existed to count blocked vs approved executions in a time period.
3. **Q7 was unanswerable** — no funnel existed to show conversion rates across pipeline stages per family.
4. **Q5 was partially limited** — the funnel was missing, making it impossible to see where pipelines break at scale.

## Design Decisions

### 1. Read-Side Attribution (No Write-Side Changes)

The risk domain already stores all attribution data:
- `Disposition`: approved/modified/rejected
- `Rationale`: human-readable explanation of the assessment
- `Constraints`: active position limits (MaxPositionSize, MaxExposure, StopDistance)
- `Strategies[]`: contributing strategies with DecisionSeverity and DecisionRationale

Rather than adding new write-side fields, S298 computes a **RiskAttribution projection** at read time from existing data. This projection is added to every `CompositeExecutionChain` that has a risk stage.

### 2. Attribution Shape

```go
type RiskAttribution struct {
    Disposition       string                      // approved/modified/rejected
    Rationale         string                      // human-readable explanation
    ActiveConstraints risk.Constraints            // constraints active at assessment time
    StrategyContext   []AttributionStrategyContext // contributing strategies with decision context
}

type AttributionStrategyContext struct {
    Type              string // strategy type (e.g., "mean_reversion_entry")
    Direction         string // long/short/flat
    Confidence        string // strategy confidence
    DecisionSeverity  string // from originating decision
    DecisionRationale string // from originating decision
}
```

This surfaces the complete causal chain from decision through risk at the chain level, without requiring traversal of the nested risk stage.

### 3. Aggregation Queries

Two new ClickHouse queries provide the aggregate view:

**Pipeline Funnel** (Q7, Q5): Counts events per stage across all 5 domain tables for a given type/source/symbol/timeframe. Each table is queried independently with `SELECT count() FROM <table> WHERE type=? AND source=? AND symbol=? AND timeframe=?`.

**Disposition Breakdown** (Q6): Groups risk assessments by disposition with `SELECT disposition, count() FROM risk_assessments WHERE ... GROUP BY disposition`.

Both queries support optional time-range filtering (since/until).

### 4. No Batch Lookup Extension for Pre-Execution Chains

The S297 batch endpoint starts from the executions table, meaning chains that broke before execution are not discoverable. After analysis, we chose NOT to extend batch lookup to start from other tables because:

1. The **pipeline funnel** endpoint already reveals where pipelines break at scale (100 signals → 80 decisions → 0 executions tells you more than individual broken chains).
2. Individual broken chains are discoverable via the **single-chain endpoint** if the correlation_id is known.
3. Extending batch to start from signals would require a fundamentally different query pattern and increase scope beyond the surgical intent of S298.

## Attribution Computation

Attribution is computed in the use case layer (`get_composite_chain.go`) after chain assembly, not in the ClickHouse adapter. This keeps the reader as pure data assembly and the use case as business enrichment.

```
Reader assembles chain → computeChainCompleteness() → [returned to use case]
Use case receives chain → computeAttribution() → [returned in reply]
```

Attribution is nil when the risk stage is absent from the chain.

## What This Does NOT Do

- No write-side schema changes
- No new ClickHouse columns or tables
- No risk evaluator redesign
- No per-constraint triggering analysis (the write-side does not store which constraint caused rejection)
- No dashboard or UI
- No streaming
