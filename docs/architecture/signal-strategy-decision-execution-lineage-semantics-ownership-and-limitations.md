# Signal-Strategy-Decision-Execution Lineage: Semantics, Ownership, and Limitations

> S470 — Detailed semantics, ownership boundaries, and known limitations of the decision lineage model.

## Stage Semantics

### Signal

**Domain**: `internal/domain/signal`
**Owner binary**: derive
**Produces**: `SignalGeneratedEvent` -> `SIGNAL_EVENTS` stream

A Signal is a derived interpretation of evidence (candle, trade burst, volume). It carries a single numeric value (e.g., RSI = 23.5) and type-specific metadata. Signals are the root of the causal chain — they originate CorrelationIDs.

**Causal inputs**: Evidence candle (via CorrelationID inherited from candle)
**Causal outputs**: Consumed by Decision evaluators

### Decision

**Domain**: `internal/domain/decision`
**Owner binary**: derive
**Produces**: `DecisionEvaluatedEvent` -> `DECISION_EVENTS` stream

A Decision is a categorical judgment derived from one or more signals. It carries outcome (triggered/not_triggered/insufficient), severity, confidence, and rationale.

**Causal inputs**: Signal(s) via `SignalInput[]` — each input now carries `EventID` (S470)
**Causal outputs**: Consumed by Strategy resolvers
**Transformation semantics**: Signal value -> threshold evaluation -> categorical outcome + severity classification

### Strategy

**Domain**: `internal/domain/strategy`
**Owner binary**: derive
**Produces**: `StrategyResolvedEvent` -> `STRATEGY_EVENTS` stream

A Strategy resolves one or more decisions into a directional intent (long/short/flat) with confidence.

**Causal inputs**: Decision(s) via `DecisionInput[]` — each input now carries `EventID` (S470)
**Causal outputs**: Consumed by Risk evaluators
**Transformation semantics**: Decision outcome + severity -> directional position intent

### Risk

**Domain**: `internal/domain/risk`
**Owner binary**: derive
**Produces**: `RiskAssessedEvent` -> `RISK_ASSESSMENT_EVENTS` stream

A RiskAssessment evaluates strategy intents against position limits, exposure constraints, and drawdown tolerance.

**Causal inputs**: Strategy(s) via `StrategyInput[]` — each input now carries `EventID` (S470)
**Causal outputs**: Consumed by Execution evaluators
**Transformation semantics**: Strategy direction + confidence + decision severity -> disposition (approved/modified/rejected) + constraints

### Execution

**Domain**: `internal/domain/execution`
**Owner binary**: derive (paper intent), execute (venue submission)
**Produces**: `PaperOrderSubmittedEvent`, `VenueOrderFilledEvent`, `VenueOrderRejectedEvent`

An ExecutionIntent is a concrete order intent derived from an approved risk assessment. It carries side, quantity, status, fills, and the full causal trace (CorrelationID + CausationID at domain level).

**Causal inputs**: RiskAssessment via `RiskInput` — now carries `EventID` (S470)
**Causal outputs**: Terminal (fills, rejections)
**Transformation semantics**: Risk disposition + constraints -> order side + quantity + lifecycle

## Ownership Boundaries

### Domain Isolation Rule

Each domain package owns its own "Input" types. A Decision uses `decision.SignalInput`, not `signal.Signal`. This prevents import cycles and keeps each domain independently testable.

```
decision.SignalInput    -- owned by decision, not signal
strategy.DecisionInput  -- owned by strategy, not decision
risk.StrategyInput      -- owned by risk, not strategy
execution.RiskInput     -- owned by execution, not risk
```

### Enrichment Layer

Input types are populated by pure application evaluators. Causal metadata (EventID) is enriched at the actor layer, which has access to event context. This separation keeps:
- **Application logic** free of event infrastructure
- **Actor layer** responsible for wiring causal references
- **Domain types** carrying the result

### Event Metadata vs Domain Fields

| Field | Event Metadata | Domain Type | Scope |
|---|---|---|---|
| `CorrelationID` | All events | Only `ExecutionIntent` | Chain-wide trace |
| `CausationID` | All events | Only `ExecutionIntent` | Immediate parent |
| `EventID` (of parent) | N/A | All Input types (S470) | Parent stage reference |

## Semantic Depth Forwarding

The pipeline forwards key semantic context across boundaries for downstream use:

| Forwarded Field | From | To | Purpose |
|---|---|---|---|
| `DecisionSeverity` | Decision -> Strategy -> Risk -> Execution | Risk multipliers, behavioral analysis |
| `DecisionRationale` | Decision -> Strategy -> Risk | Risk rationale composition |
| `StrategyType` | Strategy -> Risk -> Execution | Strategy-aware position sizing |
| `StrategyDirection` | Strategy -> Risk -> Execution | Side determination |

This forwarding is separate from lineage — it carries semantic content, not identity references.

## Limitations

### L1: Evidence-to-Signal gap

The Signal domain type does not carry an explicit reference to the evidence candle(s) that produced it. The linkage exists via CorrelationID at the event metadata level, but not at the domain level. This gap is intentional — signals are generated from in-memory candle state, not from discrete evidence events.

### L2: Single-parent assumption

The current model assumes single-parent causation at each stage (one CausationID per event). Multi-signal decisions carry multiple `SignalInput` entries, but all reference the same parent event (the signal that triggered the evaluation). Cross-symbol or cross-timeframe causation is not modeled.

### L3: No retroactive chain validation

Chains are validated prospectively (at creation time) via the lineage package. There is no background process that validates persisted chains in ClickHouse. Stale or broken chains in historical data are detectable only via composite chain queries.

### L4: KV latest does not carry full Input detail

NATS KV latest buckets store the most recent domain object per partition key. The Input types (including EventID) are serialized in JSON within the KV value. However, the KV layer does not index by EventID — full chain reconstruction requires ClickHouse.

### L5: EventID is best-effort in existing data

The EventID enrichment (S470) applies only to events produced after deployment. Historical events in ClickHouse will have empty `event_id` fields in their serialized Input types. This does not affect CorrelationID/CausationID-based chain reconstruction, which has been in place since the pipeline's inception.

### L6: Rejection and partial fill lineage

VenueOrderRejectedEvent and VenueOrderFilledEvent carry the ExecutionIntent's CorrelationID and CausationID, but these are the execution event's causal references (pointing to risk), not new links in the chain. They are lifecycle extensions of the execution stage, not additional causal stages.

## Trade-offs

### Why Input-level EventID instead of expanding CorrelationID/CausationID to all domain types?

Adding CorrelationID/CausationID to Signal, Decision, Strategy, and Risk domain types was considered but rejected:
- It would duplicate what event metadata already carries
- It would require ClickHouse schema changes (new columns) for existing tables
- The Input-level EventID captures the same information with minimal disruption
- Only ExecutionIntent justifies domain-level trace fields because it crosses binary boundaries (derive -> execute)

### Why not a full lineage event stream?

A dedicated lineage stream that records every chain link was considered but deferred:
- The existing per-stage streams already carry all lineage data via event metadata
- The CompositeExecutionChain read model already reconstructs chains
- A lineage stream would add write amplification without enabling new queries
- This may be revisited if real-time chain monitoring becomes a requirement

## Test Coverage

- `internal/domain/lineage/lineage_test.go` — Chain validation, completeness, ordering, error cases
- `internal/actors/scopes/derive/s470_lineage_causality_test.go` — Per-stage EventID wiring, full-chain integration
