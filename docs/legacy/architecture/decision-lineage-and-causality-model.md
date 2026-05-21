# Decision Lineage and Causality Model

> S470 — Canonical model for tracing causal relationships across the market-foundry pipeline.

## Overview

The market-foundry pipeline processes market data through a five-stage causal chain:

```
Signal -> Decision -> Strategy -> Risk -> Execution
```

Each stage transforms its input into a domain-specific output, and every transformation must preserve an explicit, auditable causal link to its predecessor. This document defines the canonical lineage model that governs these relationships.

## Causality Primitives

### CorrelationID

A trace-wide identifier assigned once at signal generation and propagated unchanged through all downstream stages. It enables grouping all events belonging to the same causal chain.

- **Set by**: Signal sampler actor (inherited from evidence candle correlation)
- **Propagated through**: `events.Metadata.CorrelationID` at event level; `ExecutionIntent.CorrelationID` at domain level
- **Semantics**: "All events with this CorrelationID originated from the same signal root"

### CausationID

A pointer to the immediately preceding event in the causal chain. It forms a directed parent-child graph of causation.

- **Set by**: Each stage's actor, pointing to the parent event's ID
- **Propagated through**: `events.Metadata.CausationID` at event level; `ExecutionIntent.CausationID` at domain level
- **Semantics**: "This event was directly caused by the event with this ID"

### Input EventID (S470)

Each domain type's "Input" struct now carries an `EventID` field that references the parent event that produced the input data. This makes the causal reference explicit at the domain level, not just at the event metadata level.

| Input Type | Parent Stage | Field |
|---|---|---|
| `decision.SignalInput` | Signal | `EventID` |
| `strategy.DecisionInput` | Decision | `EventID` |
| `risk.StrategyInput` | Strategy | `EventID` |
| `execution.RiskInput` | Risk | `EventID` |

## Chain Model

A complete causal chain consists of five links, one per stage:

```
ChainLink {
    Stage:         "signal" | "decision" | "strategy" | "risk" | "execution"
    EventID:       <this event's unique ID>
    CorrelationID: <chain-wide trace ID>
    CausationID:   <parent event's ID>
}
```

### Invariants

1. **CorrelationID consistency**: All links in a chain share the same CorrelationID.
2. **CausationID linkage**: Each link's CausationID equals the previous link's EventID.
3. **Stage ordering**: Links must appear in canonical order (signal < decision < strategy < risk < execution).
4. **EventID non-empty**: Every link must have a non-empty EventID.
5. **Input EventID alignment**: Each Input type's EventID matches the CausationID of its containing event.

### Partial Chains

Chains may be incomplete if:
- The chain was terminated early (e.g., risk rejected the position)
- Events are still propagating through the pipeline
- Evidence is queried before all stages have fired

Partial chains are valid as long as the present links satisfy the invariants above.

## Implementation Layers

### Event Metadata Layer

`events.Metadata` carries CorrelationID and CausationID at the transport level. This is the foundational layer — all events carry these fields regardless of domain type.

```go
events.Metadata{
    ID:            "evt-abc123",
    OccurredAt:    time.Now(),
    CorrelationID: "corr-xyz",
    CausationID:   "evt-parent",
}
```

### Domain Input Layer (S470)

Each domain type's Input struct carries an `EventID` that explicitly references the parent event. This enrichment happens at the actor layer after the pure evaluator returns, keeping application logic free of event infrastructure concerns.

```go
// Actor layer enrichment pattern:
dec, ok := evaluator.Evaluate(...)
for i := range dec.Signals {
    dec.Signals[i].EventID = msg.CausationID  // signal event ID
}
```

### Lineage Package

`internal/domain/lineage` provides formal validation of chain integrity:
- `Chain` and `ChainLink` types
- `ValidateChain()` — checks all invariants
- `IsComplete()` / `MissingStages()` — completeness assessment

### Composite Read Model

`CompositeExecutionChain` in `analyticalclient` reconstructs chains from ClickHouse by querying all five tables using CorrelationID. Each `*WithTrace` type carries `EventID`, `CorrelationID`, and `CausationID` from the persisted event metadata.

## Data Flow Example

```
1. Evidence candle finalized (CorrelationID = "corr-001")
      |
2. RSI Signal generated
      EventID = "sig-001", CorrelationID = "corr-001"
      |
3. RSI Oversold Decision evaluated
      EventID = "dec-001", CorrelationID = "corr-001", CausationID = "sig-001"
      Decision.Signals[0].EventID = "sig-001"
      |
4. Mean Reversion Strategy resolved
      EventID = "str-001", CorrelationID = "corr-001", CausationID = "dec-001"
      Strategy.Decisions[0].EventID = "dec-001"
      |
5. Position Exposure Risk assessed
      EventID = "rsk-001", CorrelationID = "corr-001", CausationID = "str-001"
      Risk.Strategies[0].EventID = "str-001"
      |
6. Paper Order Execution intent produced
      EventID = "exe-001", CorrelationID = "corr-001", CausationID = "rsk-001"
      Execution.Risk.EventID = "rsk-001"
      Execution.CorrelationID = "corr-001"
      Execution.CausationID = "rsk-001"
```

## Querying the Chain

### By CorrelationID (full reconstruction)
Query all five ClickHouse tables with `WHERE correlation_id = ?` and assemble the `CompositeExecutionChain`.

### By CausationID (parent lookup)
Given any event, its `causation_id` column points to the parent. This enables walking backward through the chain.

### By Input EventID (domain-level reference)
The JSON-serialized Input types in ClickHouse carry the `event_id` field, enabling domain-level joins without relying solely on event metadata.
