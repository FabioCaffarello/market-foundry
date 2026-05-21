# Risk Type 02: `drawdown_limit` — Semantics, Consistency, and Boundaries

## Type Identity

- **Type name**: `drawdown_limit`
- **Domain**: risk
- **Layer position**: after strategy resolution, before execution
- **Disposition space**: `approved` | `modified` (never `rejected`)
- **Constraint fields**: `StopDistance`, `MaxExposure`

## Semantic Definition

`drawdown_limit` answers: **"Given this strategy's direction and confidence, what is the maximum acceptable loss before the position should be stopped out?"**

This is distinct from `position_exposure` which answers: **"How large should this position be relative to portfolio limits?"**

### Disposition Semantics

| Disposition | Meaning |
|-------------|---------|
| `approved` | Stop distance is within max drawdown limits; trade may proceed with computed stop |
| `modified` | Stop distance exceeds max drawdown; capped to maximum allowed drawdown |

### Confidence Semantics

- **Input**: strategy confidence (decimal string, 0.0–1.0)
- **Output**: risk confidence = strategy confidence × 0.90
- The 0.90 discount (vs 0.95 for position_exposure) reflects that drawdown estimates carry more uncertainty than position sizing

### Constraint Semantics

| Constraint | Format | Meaning |
|------------|--------|---------|
| `StopDistance` | `"0.0255"` | Maximum distance (as %) before stop-loss triggers |
| `MaxExposure` | `"0.0500"` | Maximum drawdown limit (as %) for this evaluation |

### Parameter Semantics

| Parameter | Default | Meaning |
|-----------|---------|---------|
| `max_drawdown_pct` | `0.0500` (5%) | Maximum acceptable drawdown per position |
| `stop_distance_pct` | `0.0300` (3%) | Base stop distance, scaled by confidence |

## Consistency with Adjacent Domains

### Decision → Strategy → Risk Chain

```
Decision (rsi_oversold / ema_crossover)
  │ outcome, confidence, severity, rationale
  ↓
Strategy (mean_reversion_entry / trend_following_entry)
  │ direction, confidence, decision_severity, decision_rationale
  ↓
Risk (position_exposure / drawdown_limit)    ← both receive same inputs
  │ disposition, constraints, risk_confidence
  ↓
Execution
```

Both risk evaluators receive identical `strategyResolvedMessage` data. Neither imports the strategy domain — all data arrives as primitives per DBI-9.

### Decision Context Traceability

`drawdown_limit` preserves the full decision context chain:
- `StrategyInput.DecisionSeverity` → carried forward from strategy
- `StrategyInput.DecisionRationale` → carried forward from strategy
- `Metadata["decision_severity"]` → queryable in ClickHouse
- `Metadata["decision_rationale"]` → queryable in ClickHouse
- `Rationale` → includes decision severity when present

### Type Coexistence

Both risk types:
- Share `RISK_EVENTS` stream (subjects differentiated by type segment)
- Share `risk_assessments` ClickHouse table (differentiated by `type` column)
- Share `RiskAssessment` domain struct
- Use independent KV buckets for latest materialization
- Use independent durable consumers for writer and store
- Are independently enabled via `pipeline.risk_families`

## Boundaries

### What `drawdown_limit` Does

- Computes stop distance from strategy confidence
- Enforces a minimum stop distance floor (0.50%)
- Caps stop distance to max drawdown limit
- Carries decision context through for observability
- Produces one `RiskAssessedEvent` per strategy resolution

### What `drawdown_limit` Does NOT Do

- Does not consider market volatility or ATR
- Does not aggregate risk across multiple symbols
- Does not implement trailing stops or dynamic stop adjustment
- Does not reject trades (only approves or modifies)
- Does not read state from other risk evaluators
- Does not interact with `position_exposure` results
- Does not tune parameters at runtime

### Future Extension Points

If depth is needed later, these are the natural extension points:
1. **Volatility-aware stops**: feed ATR or historical vol through metadata → widen/tighten stops
2. **Severity-gated rejection**: if decision severity is `critical`, consider `rejected` disposition
3. **Cross-type aggregation**: combine position_exposure + drawdown_limit into composite risk score
4. **Dynamic parameters**: load max_drawdown_pct from configctl for per-symbol tuning

None of these are in scope for S243 (breadth-only).
