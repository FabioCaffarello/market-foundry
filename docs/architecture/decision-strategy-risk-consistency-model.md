# Decision-Strategy-Risk Consistency Model

## Purpose

This document defines the end-to-end consistency model across the three derive domains — decision, strategy, and risk — after S234/S235/S236. It establishes how semantic information flows, what each domain owns, and how traceability is maintained without coupling.

## Domain Responsibilities (Post-S236)

### Decision Domain
**Evaluates whether a condition is met, how strongly, and why.**

| Field | Semantics |
|-------|-----------|
| Outcome | `triggered`, `not_triggered`, `insufficient` |
| Severity | `none`, `low`, `moderate`, `high` — extremeness of the condition |
| Confidence | 0.0–1.0 scalar — certainty of the evaluation |
| Rationale | Human-readable explanation (e.g., "RSI 28.50 below threshold 30.0") |
| Signals | Which signals contributed (via `SignalInput`) |

### Strategy Domain
**Resolves decisions into directional intent with actionable parameters.**

| Field | Semantics |
|-------|-----------|
| Direction | `long`, `short`, `flat` — positional intent |
| Confidence | Strategy-level confidence (currently inherited from decision) |
| Parameters | Execution parameters: entry type, target/stop offsets |
| Decisions | Which decisions contributed (via `DecisionInput` with severity/rationale) |

### Risk Domain
**Assesses strategy intent against exposure limits and produces disposition.**

| Field | Semantics |
|-------|-----------|
| Disposition | `approved`, `modified`, `rejected` — risk gate outcome |
| Confidence | Risk-adjusted confidence (strategy confidence * 0.95) |
| Constraints | Risk-imposed limits: max position size, max exposure |
| Rationale | Context-rich explanation referencing decision severity |
| Strategies | Which strategies contributed (via `StrategyInput` with decision context) |
| Metadata | Observability: `decision_severity`, `decision_rationale` |

## Information Flow

```
Signal → Decision → Strategy → Risk → Execution
         ↓           ↓           ↓
      severity     DecisionInput StrategyInput
      rationale    (severity,    (decision_severity,
      confidence    rationale)    decision_rationale)
```

### Boundary Crossings

Each boundary uses **primitive data only** (strings, ints). No domain struct crosses a boundary.

| Boundary | Carrier | Fields Forwarded |
|----------|---------|-----------------|
| Decision → Strategy | `decisionEvaluatedMessage` | type, outcome, confidence, severity, rationale |
| Strategy → Risk | `strategyResolvedMessage` | type, direction, confidence, decision_severity, decision_rationale |
| Risk → Execution | `riskAssessedMessage` | type, disposition, confidence, max_position, strategy_direction, decision_severity |

### Input Types (Domain-Owned Copies)

Each domain owns a copy of upstream data rather than importing upstream structs:

| Domain | Input Type | Upstream Fields Carried |
|--------|-----------|------------------------|
| Decision | `SignalInput` | type, value, timeframe |
| Strategy | `DecisionInput` | type, outcome, confidence, severity, rationale, timeframe |
| Risk | `StrategyInput` | type, direction, confidence, timeframe, decision_severity, decision_rationale |

## Consistency Guarantees

### 1. Semantic Coherence
- A `flat` direction always produces `approved` disposition with `1.0000` confidence
- A `long`/`short` direction produces position sizing scaled by confidence
- Decision severity flows through unchanged — risk records it, does not transform it

### 2. Traceability Chain
Every risk assessment can trace back to its originating decision:
- `RiskAssessment.Strategies[0].DecisionSeverity` → the decision's severity
- `RiskAssessment.Strategies[0].DecisionRationale` → the decision's reasoning
- `RiskAssessment.Metadata["decision_severity"]` → quick-access for analytical queries

### 3. Monotonicity
- KV stores enforce timestamp monotonicity: newer assessments always win
- Deduplication keys prevent duplicate processing at each layer

### 4. Validation Gates
Each domain validates its output before publishing:
- `Decision.Validate()` — outcome, confidence, severity constants
- `Strategy.Validate()` — direction, confidence, at least one decision input
- `RiskAssessment.Validate()` — disposition, confidence, rationale, at least one strategy input

### 5. Materialization Gate
Only `Final=true` assessments materialize to KV stores. Non-final assessments are silently skipped.

## Analytical Query Patterns

### Risk assessments with decision context (no joins needed)
```sql
SELECT
    type, symbol, timeframe, disposition, confidence, rationale,
    JSONExtractString(strategies, 1, 'decision_severity') AS decision_severity,
    JSONExtractString(strategies, 1, 'decision_rationale') AS decision_rationale
FROM risk_assessments
WHERE type = 'position_exposure'
  AND symbol = 'btcusdt'
ORDER BY timestamp DESC
LIMIT 50
```

### Filter by decision severity in metadata
```sql
SELECT *
FROM risk_assessments
WHERE JSONExtractString(metadata, 'decision_severity') = 'high'
ORDER BY timestamp DESC
```

## Non-Objectives

- Decision severity does NOT influence position sizing or disposition logic today
- No cross-domain validation (e.g., risk does not reject based on severity)
- No aggregate risk across multiple symbols
- No temporal risk (e.g., "too many high-severity decisions in 5 minutes")
- These remain viable future extensions if risk policy evolves

## Trade-offs

| Decision | Rationale |
|----------|-----------|
| Severity as traceability, not logic | Keeps risk evaluation deterministic and testable; avoids coupling risk rules to decision heuristics |
| `omitempty` JSON tags | Backward-compatible with existing serialized data; no migration needed |
| Metadata duplication | `decision_severity` in both `StrategyInput` and `Metadata` enables both structured and query-optimized access |
| No rejection path | Rejection based on severity is a policy decision that should be deliberate, not an implicit S236 side-effect |
