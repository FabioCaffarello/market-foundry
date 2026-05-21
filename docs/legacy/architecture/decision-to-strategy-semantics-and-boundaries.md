# Decision-to-Strategy Semantics and Boundaries

## Purpose

This document defines the semantic contract between the decision and strategy domains, the isolation boundary that separates them, and the rules for how information flows across that boundary.

## Domain Responsibilities

### Decision Domain

**Responsibility**: Evaluate whether a condition is met, how strongly, and why.

**Owns**:
- `Outcome` — categorical result: `triggered`, `not_triggered`, `insufficient`
- `Severity` — extremeness classification: `none`, `low`, `moderate`, `high`
- `Confidence` — continuous scalar (0.0–1.0) representing evaluation certainty
- `Rationale` — human-readable explanation of the evaluation
- `Signals` — which signals contributed (via `SignalInput`)
- `Metadata` — evaluator-specific context (threshold, rsi_zone, distance_pct)

**Does NOT own**: directional intent, entry parameters, position sizing, risk constraints.

### Strategy Domain

**Responsibility**: Resolve decisions into directional intent with actionable parameters.

**Owns**:
- `Direction` — positional intent: `long`, `short`, `flat`
- `Confidence` — strategy-level confidence (may differ from decision confidence)
- `Parameters` — execution parameters: entry type, target offset, stop offset
- `Decisions` — which decisions contributed (via `DecisionInput`)
- `Metadata` — strategy-specific context (reason, decision_rationale)

**Does NOT own**: signal values, evaluation thresholds, severity classification logic.

## Isolation Boundary (DBI-9)

The decision and strategy domains are **import-isolated**:
- `strategy` package does NOT import `decision` package.
- `decision` package does NOT import `strategy` package.
- Data crosses the boundary as **primitive values** (strings, ints), never as domain structs.

### Boundary Enforcement Points

1. **Actor message** (`decisionEvaluatedMessage`): carries decision data as flat primitives.
2. **Resolver signature**: accepts `string` parameters, not `decision.Decision`.
3. **DecisionInput struct**: strategy's own type that mirrors (but does not reference) decision fields.

## Data Flow

```
Decision Domain                     Boundary (primitives)              Strategy Domain
─────────────────                   ─────────────────────              ───────────────
Decision {                          decisionEvaluatedMessage {         Resolver receives:
  Type: "rsi_oversold"       →       DecisionType                  →   decisionType string
  Outcome: "triggered"        →       DecisionOutcome               →   decisionOutcome string
  Confidence: "0.85"          →       DecisionConfidence            →   decisionConfidence string
  Severity: "low"             →       DecisionSeverity              →   decisionSeverity string
  Rationale: "RSI 28.5..."   →       DecisionRationale             →   decisionRationale string
  Timeframe: 60               →       Timeframe                     →   decisionTimeframe int
  Timestamp: t                →       Timestamp                     →   ts time.Time
}                                   }
```

## Semantic Contract

### What Strategy May Consume from Decision

| Field | Usage in Strategy | Purpose |
|-------|------------------|---------|
| `outcome` | Controls direction (triggered→long, not_triggered→flat, insufficient→flat) | Primary resolution input |
| `confidence` | Forwarded as strategy confidence for triggered outcomes | Continuous strength signal |
| `severity` | Recorded in `DecisionInput.Severity` | Traceability; future parameter modulation candidate |
| `rationale` | Recorded in `DecisionInput.Rationale` and `Metadata["decision_rationale"]` | Observability and auditability |

### What Strategy Must NOT Do with Decision Data

1. **Must NOT import decision types** — use primitive strings, not `decision.Severity`.
2. **Must NOT re-evaluate conditions** — strategy resolves intent from outcomes, not from raw signal values.
3. **Must NOT override decision semantics** — if decision says `not_triggered`, strategy must not infer a trigger.
4. **Must NOT couple resolution logic to severity** (in S235) — severity is recorded, not acted upon. This preserves the option to introduce severity-aware resolution as a validated, deliberate change.

## Contracts Summary

### DecisionInput (strategy-owned)

```go
type DecisionInput struct {
    Type       string `json:"type"`       // decision family (e.g., "rsi_oversold")
    Outcome    string `json:"outcome"`    // categorical result
    Confidence string `json:"confidence"` // evaluation confidence
    Severity   string `json:"severity"`   // extremeness classification
    Rationale  string `json:"rationale"`  // human-readable explanation
    Timeframe  int    `json:"timeframe"`  // evaluation timeframe in seconds
}
```

### Strategy Metadata Keys

| Key | Source | Description |
|-----|--------|-------------|
| `reason` | Resolver | Why strategy resolved to flat (e.g., `insufficient_data`) |
| `decision_rationale` | Decision (forwarded) | The decision's rationale string, for observability |

## Boundary Rules for Future Work

1. **Adding new decision fields**: Add to `decisionEvaluatedMessage`, resolver signature, and `DecisionInput`. Do NOT add decision imports to strategy.
2. **Severity-aware resolution**: Acceptable as a future change if validated. Would modulate `Parameters` (e.g., target_offset, stop_offset) based on severity. Must be introduced with explicit tests and documented thresholds.
3. **Multi-decision strategies**: Future strategies may consume multiple decisions. The `Decisions []DecisionInput` slice already supports this. Each decision's severity and rationale would be preserved independently.
4. **New strategy families**: Must follow the same DBI-9 boundary pattern. No strategy family may import decision domain types.

## Non-Goals

- Merging decision and strategy into a single domain.
- Strategy re-evaluating raw signals.
- Exposing decision internal state (thresholds, zones) to strategy beyond what rationale and severity carry.
- Building composite severity across multiple decisions in this stage.
