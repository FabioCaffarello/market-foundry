# Decision Review Surface and Evidence Bundle

**Stage:** S471
**Status:** Implemented
**Date:** 2026-03-25

## Purpose

This document describes the decision review surface introduced in S471. The surface provides a decision-centric view of the causal pipeline, enabling operators and developers to review how a decision was formed, what constraints were applied, and what execution resulted — all from a single endpoint.

## Problem Statement

Before S471, decision review required correlating data across multiple endpoints:
- `/analytical/composite/chain` (execution-centric, walks backward from execution)
- `/analytical/decision/history` (flat decision records, no upstream/downstream context)
- `/analytical/risk/history` (risk assessments, disconnected from decision inputs)

An operator asking "why did this decision trigger, and what happened next?" had to manually reconstruct the chain by querying 3-5 endpoints and correlating by timestamp or correlation_id. This is error-prone and slow for operational review.

## Solution: Decision Review Bundle

The Decision Review Bundle is a read-side projection that takes the existing `CompositeExecutionChain` (S296/S298) and re-projects it through a decision-centric lens. No new write-side changes are required.

### Bundle Structure

A `DecisionReviewBundle` contains five sections:

| Section | Description | Source |
|---|---|---|
| **Inputs** | Signal evidence that fed the decision | `decision.Signals` + chain signal event |
| **Transform** | The decision evaluation itself (type, outcome, severity, confidence, rationale) | Chain decision event |
| **Resolution** | Strategy resolved from the decision (type, direction, confidence, parameters) | Chain strategy event |
| **Constraints** | Risk assessment and limits applied (disposition, rationale, limits) | Chain risk event |
| **Output** | Execution intent and its lifecycle outcome (side, quantity, status) | Chain execution event |

Each section is optional — a `not_triggered` decision will typically have no Resolution, Constraints, or Output.

### Human-Readable Explanation

Each bundle includes a structured `explanation` field that synthesizes all sections into a narrative:

> Decision "rsi_oversold" evaluated binance/btcusdt/60: outcome=triggered, severity=high, confidence=0.85. Rationale: RSI crossed below 30 threshold. Signal inputs: rsi. Strategy resolved: mean_reversion_entry direction=long confidence=0.80. Risk gate: disposition=approved confidence=0.75. Execution: paper_order side=buy status=submitted quantity=0.1 filled=.

## HTTP Endpoints

### Single Decision Review
```
GET /analytical/composite/decision/review?correlation_id=...&symbol=...
```
Returns the evidence bundle for one specific decision chain.

**Required parameters:**
- `correlation_id` — the correlation ID linking the causal chain
- `symbol` — required for cross-symbol isolation (S301)

**Optional parameters:**
- `outcome` — filter by decision outcome (triggered, not_triggered, insufficient)

### Batch Decision Reviews
```
GET /analytical/composite/decision/reviews?source=...&symbol=...&timeframe=...&outcome=...&since=...&until=...&limit=...
```
Returns recent decision review bundles for a given partition key.

**Required parameters:**
- `source`, `symbol`, `timeframe` — standard partition key

**Optional parameters:**
- `outcome` — filter by decision outcome
- `since`, `until` — unix timestamp range
- `limit` — max results (default 20, max 100)

## Architecture

### Data Flow

```
ClickHouse tables (5 domain tables)
  |
  v
CompositeReader.QueryChainByCorrelationID / QueryChainsBatch
  |
  v
GetDecisionReviewUseCase (projects chain -> DecisionReviewBundle)
  |
  v
CompositeWebHandler.GetDecisionReview / GetDecisionReviews
  |
  v
HTTP JSON response
```

### Key Design Decisions

1. **Reuses existing CompositeReader** — no new ClickHouse queries or tables. The review surface is a pure read-side projection over the existing composite chain infrastructure.

2. **Decision-anchored** — chains without a decision stage produce no review bundle. The decision is the mandatory anchor for this surface.

3. **Outcome filtering** — batch queries over-fetch from the composite reader and post-filter by outcome. This avoids adding decision-specific queries to the composite reader while keeping the surface responsive.

4. **Attribution preserved** — the existing `computeAttribution` (S298) runs before projection, so risk attribution context is available in the Constraints section.

## Limitations

1. **Execution-rooted batch fetching**: Batch mode starts from the executions table (via `QueryChainsBatch`), which means decisions that never reached execution (e.g., `not_triggered` decisions) are not returned in batch mode unless they happen to share a correlation_id with an execution event. This is a known limitation inherited from the composite reader design.

2. **No decision-first batch query**: A decision-first batch (starting from the decisions table) would require a new ClickHouse query path. This is deferred — the current surface covers the most common review case (decisions that produced execution).

3. **Signal input granularity**: The Inputs section carries the decision's own `SignalInput` records, which are a decision-owned summary of what signals contributed. The raw signal values (e.g., exact RSI value at decision time) are available through the chain signal event but are not duplicated into the review — use the signal event_id for drill-down.

4. **No temporal comparison**: The surface shows one decision at a time or a list. It does not compute diffs between decisions or trend analysis — that is a dashboard concern (out of scope per guard rails).

## Alignment

- **S296/S298**: Reuses composite chain and attribution infrastructure.
- **S301**: Symbol isolation enforced on all lookup paths.
- **S470**: EventID fields on all domain inputs enable precise causal tracing within the bundle.
- **S455A**: The review surface complements (does not replace) the session explain surface, which is execution/lifecycle-centric.
