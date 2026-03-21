# Strategy Alignment on Top of Richer Decisions

## Context

S234 enriched the decision domain with two new first-class fields:
- **Severity** (`none`, `low`, `moderate`, `high`) — classifies how extreme the evaluated condition is.
- **Rationale** (human-readable string) — explains why the decision reached its outcome.

Before S235, these fields were produced by the decision evaluator but not forwarded to the strategy layer. The strategy resolver received only `outcome` and `confidence` as primitive data across the DBI-9 boundary.

## Problem

Strategy's `DecisionInput` — the strategy-owned record of which decision contributed — lacked the semantic depth that decisions now carry. This created an information gap:
- Strategy had no visibility into *how extreme* the decision condition was.
- Strategy metadata could not trace *why* the decision was made.
- Downstream consumers (risk, execution, analytical queries) saw strategies without decision context.

## Solution

### 1. DecisionInput Enrichment

`strategy.DecisionInput` gains two new fields:

```go
type DecisionInput struct {
    Type       string `json:"type"`
    Outcome    string `json:"outcome"`
    Confidence string `json:"confidence"`
    Severity   string `json:"severity"`
    Rationale  string `json:"rationale"`
    Timeframe  int    `json:"timeframe"`
}
```

These are **strategy-owned strings** — they do not import from `decision.Severity`. The DBI-9 isolation boundary is preserved: values cross as primitives.

### 2. Message Boundary Update

`decisionEvaluatedMessage` (the actor-internal message from decision evaluator to strategy resolver) gains:
- `DecisionSeverity string`
- `DecisionRationale string`

The decision evaluator actor now forwards `string(dec.Severity)` and `dec.Rationale` in the fan-out message.

### 3. Resolver Signature

`MeanReversionEntryResolver.Resolve()` accepts two additional string parameters:
- `decisionSeverity` — forwarded into `DecisionInput.Severity`
- `decisionRationale` — forwarded into `DecisionInput.Rationale` and `Metadata["decision_rationale"]`

### 4. Metadata Propagation

When `decisionRationale` is non-empty, the resolver copies it into `strategy.Metadata["decision_rationale"]`. This makes the decision's reasoning visible in strategy queries and analytical reports without requiring a join back to the decisions table.

### 5. Resolution Logic Unchanged

**Critically**, severity does NOT alter the resolution logic in S235. Direction, confidence, and parameters remain determined solely by outcome and confidence, as before. Severity is recorded for traceability, not for control flow.

This is a deliberate guard rail: severity-aware parameter modulation (e.g., wider targets for high-severity signals) is a candidate for future work, but introducing it here would inflate heuristics without validation.

## Backward Compatibility

- Old `DecisionInput` JSON without `severity`/`rationale` fields deserializes correctly — Go's `json.Unmarshal` defaults missing fields to zero values (`""` for strings).
- No ClickHouse schema migration is needed — `decisions` is a JSON column in the strategies table, and the enriched struct serializes transparently.
- The strategy domain validation does not require severity or rationale, so existing strategies remain valid.

## What Changed

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/strategy/strategy.go` | `DecisionInput` gains `Severity`, `Rationale` |
| Messages | `internal/actors/scopes/derive/messages.go` | `decisionEvaluatedMessage` gains `DecisionSeverity`, `DecisionRationale` |
| Actor | `internal/actors/scopes/derive/decision_evaluator_actor.go` | Forwards severity/rationale in fan-out |
| Actor | `internal/actors/scopes/derive/strategy_resolver_actor.go` | Passes severity/rationale to resolver |
| Application | `internal/application/strategy/mean_reversion_entry_resolver.go` | Accepts, records, and propagates severity/rationale |
| Tests | Multiple test files | Updated fixtures and assertions |

## What Did NOT Change

- Strategy resolution logic (direction, confidence, parameters)
- Strategy domain validation rules
- ClickHouse `strategies` table DDL
- NATS stream/consumer configuration
- Strategy KV materialization in store
- Strategy HTTP handlers and routes
- No new strategy families
- No new heuristics or parameter modulation
