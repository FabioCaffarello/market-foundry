# Risk Domain Deepening and Consistency Checks

## Context

After S234 (decision domain deepening) and S235 (strategy alignment), the risk domain was the third and final domain in the derive pipeline that needed alignment with the richer semantic outputs now produced upstream. Risk was evaluating strategy intents but had no visibility into the decision context that generated those strategies.

## Problem

Three gaps existed in the risk domain:

1. **Decision context lost at boundary**: `strategyResolvedMessage` carried only strategy-level primitives (type, direction, confidence). Decision severity and rationale — added in S234/S235 — were not forwarded to risk.

2. **Shallow rationale**: Risk rationale was a static string ("Position size within exposure limits" or "Position size capped to exposure limits") with no reference to what drove the strategy.

3. **No end-to-end traceability in StrategyInput**: Risk's `StrategyInput` recorded which strategy contributed but not the decision that preceded it. Analytical queries on `risk_assessments` could not trace back to decision severity without joining tables.

## Solution

### 1. StrategyInput Enrichment

`risk.StrategyInput` gains two new fields:

```go
type StrategyInput struct {
    Type              string `json:"type"`
    Direction         string `json:"direction"`
    Confidence        string `json:"confidence"`
    Timeframe         int    `json:"timeframe"`
    DecisionSeverity  string `json:"decision_severity,omitempty"`
    DecisionRationale string `json:"decision_rationale,omitempty"`
}
```

These are **risk-owned strings** — they do not import from `decision.Severity`. The DBI-9 isolation boundary is preserved. The `omitempty` tag ensures backward compatibility: existing JSON without these fields deserializes cleanly to zero values.

### 2. Message Boundary Update

`strategyResolvedMessage` gains:
- `DecisionSeverity string`
- `DecisionRationale string`

The strategy resolver actor extracts these from the strategy's first `DecisionInput` and forwards them as primitives.

`riskAssessedMessage` gains:
- `DecisionSeverity string`

The risk evaluator actor extracts this from the assessment's first `StrategyInput` and forwards it to execution evaluators for downstream traceability.

### 3. Evaluator Signature

`PositionExposureEvaluator.Evaluate()` accepts two additional string parameters:
- `decisionSeverity` — stored in `StrategyInput.DecisionSeverity` and referenced in rationale
- `decisionRationale` — stored in `StrategyInput.DecisionRationale` and in `Metadata["decision_rationale"]`

### 4. Richer Rationale

Risk rationale now includes decision severity context for non-flat, non-none cases:
- Approved: `"Position size 0.0170 within exposure limits; decision severity high"`
- Modified: `"Position size capped to 0.0200 by exposure limits; decision severity moderate"`
- Flat: `"Flat strategy requires no position"` (unchanged — no risk context needed)

### 5. Metadata Enrichment

Risk assessment `Metadata` now carries decision context for observability:
- `decision_severity` — the originating decision's severity classification
- `decision_rationale` — the originating decision's human-readable explanation

This enables analytical queries on `risk_assessments` to filter/aggregate by decision severity without joins.

## What Changed

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/risk/risk.go` | `StrategyInput` gains `DecisionSeverity`, `DecisionRationale` |
| Application | `internal/application/risk/position_exposure_evaluator.go` | New signature with decision context; richer rationale; metadata enrichment |
| Actor (derive) | `internal/actors/scopes/derive/messages.go` | `strategyResolvedMessage` and `riskAssessedMessage` gain decision fields |
| Actor (derive) | `internal/actors/scopes/derive/strategy_resolver_actor.go` | Forwards decision context from DecisionInput to risk |
| Actor (derive) | `internal/actors/scopes/derive/risk_evaluator_actor.go` | Passes decision context to evaluator; includes in fan-out |

## Backward Compatibility

- **ClickHouse**: `strategies` column stores `StrategyInput` as JSON. The new fields use `omitempty`, so old rows without them deserialize with empty strings. No schema migration required.
- **NATS KV**: Same JSON serialization — new fields are additive.
- **Read path**: `ParseStrategyInputsJSON` handles both old and new formats transparently.

## Non-Objectives

- Decision severity does NOT alter position sizing or disposition logic. It is carried for traceability and observability only.
- No new risk families are introduced.
- No rejection path based on decision severity — this remains a future option if risk policy evolves.
- No changes to the KV store, projection actor, or HTTP handler — they already handle the enriched `RiskAssessment` struct.
