# Stage S235 — Strategy Alignment on Top of Richer Decisions

## Status: Complete

## Objective

Align the strategy domain to the enriched decision semantics introduced in S234 (severity, rationale), preserving domain boundaries and avoiding heuristic inflation.

## Executive Summary

S235 threads decision severity and rationale through the decision-to-strategy boundary, enriching `DecisionInput` and strategy metadata without altering resolution logic. Strategy now records *how extreme* the decision condition was and *why* it was made, making the full pipeline observable from strategy queries alone.

## Changes Applied

### Domain Layer
- **`internal/domain/strategy/strategy.go`**: Added `Severity string` and `Rationale string` to `DecisionInput` struct.

### Message Layer
- **`internal/actors/scopes/derive/messages.go`**: Added `DecisionSeverity` and `DecisionRationale` to `decisionEvaluatedMessage`.

### Actor Layer
- **`internal/actors/scopes/derive/decision_evaluator_actor.go`**: Forwards `dec.Severity` and `dec.Rationale` in fan-out message to strategy resolvers.
- **`internal/actors/scopes/derive/strategy_resolver_actor.go`**: Passes severity and rationale to resolver.

### Application Layer
- **`internal/application/strategy/mean_reversion_entry_resolver.go`**: Resolver accepts `decisionSeverity` and `decisionRationale` as primitive strings. Records them in `DecisionInput` and propagates non-empty rationale to `Metadata["decision_rationale"]`.

### Tests
- **`internal/domain/strategy/strategy_test.go`**: Updated `validStrategy` fixture with severity and rationale.
- **`internal/application/strategy/mean_reversion_entry_resolver_test.go`**: All existing tests updated with new resolver signature. Added 4 new tests:
  - `TestMeanReversionEntryResolver_DecisionRationaleInMetadata`
  - `TestMeanReversionEntryResolver_EmptyRationaleNotInMetadata`
  - `TestMeanReversionEntryResolver_SeverityPreservedForAllOutcomes`
  - `TestMeanReversionEntryResolver_DecisionInputPreserved` (extended with severity/rationale assertions)
- **`internal/adapters/clickhouse/writerpipeline/support_test.go`**: Strategy row fixtures include severity and rationale in DecisionInput.
- **`internal/adapters/clickhouse/strategy_reader_test.go`**: Added `TestParseDecisionInputsJSON_BackwardCompatible` for old format without severity/rationale. Updated `TestParseDecisionInputsJSON_ValidArray` with new fields.

### Architecture Documentation
- **`docs/architecture/strategy-alignment-on-top-of-richer-decisions.md`**: Technical design document covering enrichment, backward compatibility, and non-changes.
- **`docs/architecture/decision-to-strategy-semantics-and-boundaries.md`**: Reference document defining the semantic contract, DBI-9 boundary, data flow, and rules for future evolution.

## Test Evidence

```
ok  internal/domain/strategy
ok  internal/application/strategy
ok  internal/adapters/clickhouse
ok  internal/adapters/clickhouse/writerpipeline
ok  internal/actors/scopes/derive
ok  internal/actors/scopes/store
ok  internal/application/strategyclient
ok  internal/application/analyticalclient
ok  internal/interfaces/http/handlers
ok  internal/interfaces/http/routes
ok  cmd/gateway
```

Full test suite: all project packages pass.

## Semantic Gains

1. **Traceability**: Strategy now records the full decision context — outcome, confidence, severity, and rationale — in a single `DecisionInput` struct. No need to join back to decisions for context.
2. **Observability**: Decision rationale surfaces in strategy metadata (`decision_rationale`), visible in HTTP responses, KV queries, and ClickHouse analytical queries.
3. **Boundary clarity**: DBI-9 isolation preserved — severity and rationale cross as primitive strings. Strategy does not import decision types.
4. **Backward compatibility**: Old DecisionInput JSON (without severity/rationale) deserializes with zero-value defaults. No migration needed.

## Boundary Preservation

| Boundary | Status |
|----------|--------|
| Decision does not import strategy | Preserved |
| Strategy does not import decision | Preserved |
| Data crosses as primitives (DBI-9) | Preserved |
| Resolution logic unchanged | Preserved |
| No new strategy families | Preserved |
| No severity-dependent heuristics | Preserved |

## Limits and Trade-offs

1. **Severity is recorded, not acted upon**: The resolver does not modulate parameters (target_offset, stop_offset) based on severity. This is deliberate — introducing severity-aware resolution without validation data would inflate heuristics without proof of value.
2. **Rationale is forwarded, not parsed**: Strategy stores the rationale string verbatim. It does not extract structured data from the rationale (e.g., parsing distance percentages). This keeps the coupling shallow.
3. **Single-decision strategies only**: The enrichment works for the current 1:1 decision-to-strategy pattern. Multi-decision strategies would need composite severity logic, which is deferred.
4. **No ClickHouse schema change**: The `decisions` column in the `strategies` table is JSON, so new fields serialize automatically. However, old rows in ClickHouse will have DecisionInput without severity/rationale fields.

## Non-Objectives (Guard Rails Honored)

- Did NOT create new strategy families.
- Did NOT merge decision and strategy into a single domain.
- Did NOT introduce severity-dependent parameter modulation.
- Did NOT add new heuristics or evaluation logic.
- Did NOT change NATS streams, consumers, or ClickHouse DDL.

## Preparation for S236 (Risk)

The strategy domain now carries richer decision context that risk evaluators can consume:
- **Severity availability**: Risk could use decision severity (via strategy's DecisionInput) to adjust position sizing or exposure limits. The data is now present in the pipeline.
- **Rationale traceability**: Risk assessments can reference the full decision→strategy chain for audit trails.
- **Pattern established**: The DBI-9 primitive-crossing pattern used here for decision→strategy can be replicated for strategy→risk, forwarding strategy-level severity or confidence modifiers.
- **Suggested focus for S236**: Align risk domain to richer strategy output, potentially introducing severity-aware constraints (e.g., tighter position limits for low-severity triggers, wider limits for high-severity triggers).
