# Strategy Domain — First Slice

## Summary

The strategy domain translates categorical decision outcomes into positional intents (long/short/flat). It sits between the decision layer and future action layers (risk, execution, portfolio).

## Family: mean_reversion_entry (STF-01)

### Input
- Decision type: `rsi_oversold`
- Decision outcomes: `triggered`, `not_triggered`, `insufficient`

### Output
- Direction: `long` (when triggered), `flat` (otherwise)
- Confidence: inherited from decision (triggered), `0.0000` (flat)
- Parameters (when long): `entry=market`, `target_offset=0.02`, `stop_offset=0.01`

### Resolution Rules

| Decision Outcome | Strategy Direction | Confidence | Parameters |
|---|---|---|---|
| `triggered` | `long` | inherited | entry, target_offset, stop_offset |
| `not_triggered` | `flat` | `0.0000` | (none) |
| `insufficient` | `flat` | `0.0000` | (none) + reason metadata |

## Stream Contracts

- **Stream**: `STRATEGY_EVENTS` (72h retention, 2GB, file-backed)
- **Event subject**: `strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}`
- **Query subject**: `strategy.query.mean_reversion_entry.latest`
- **KV bucket**: `STRATEGY_MEAN_REVERSION_ENTRY_LATEST`
- **Durable consumer**: `store-strategy-mean-reversion-entry`

## HTTP Surface

```
GET /strategy/:type/latest?source=X&symbol=Y&timeframe=Z
```

## Projection Pattern

Three-gate (final, validate, monotonicity) latest-only. No history in Phase 1.

## Activation

- Config key: `pipeline.strategy_families` (opt-in, explicit)
- Dependency: `mean_reversion_entry → rsi_oversold → rsi → candle → observation`

## Boundaries

- `strategy` does not import from `decision`, `signal`, `evidence`, or `observation` domains
- Decision data enters strategy as actor messages with primitive types
- `flat` is a valid strategy output, not an error
- `Direction` is the defining characteristic (position-aware vs position-agnostic outcome)

## Deferred to S57+

- Strategy history queries
- Additional strategy families (macd_momentum_entry, confluence_entry)
- Multi-decision strategy patterns
- Risk, execution, portfolio layers
