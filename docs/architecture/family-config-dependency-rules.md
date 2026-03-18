# Family Config Dependency Rules

This document defines the mandatory dependency rules between families across
the evidence, signal, decision, and strategy layers. These rules are enforced at
two levels: **Go config validation** (startup-time) and **raccoon-cli static
analysis** (pre-deploy / CI).

## Layers and Activation Semantics

| Layer    | Config field              | Default when absent       |
|----------|---------------------------|---------------------------|
| Evidence | `pipeline.families`       | All evidence families enabled (backward compatible) |
| Signal   | `pipeline.signal_families`| No signal families enabled (opt-in) |
| Decision | `pipeline.decision_families`| No decision families enabled (opt-in) |
| Strategy | `pipeline.strategy_families`| No strategy families enabled (opt-in) |

## Known Families

Each layer has a closed set of recognized family names. Unknown names are
rejected at config validation time (typo protection).

| Layer    | Known families                    |
|----------|-----------------------------------|
| Evidence | `candle`, `tradeburst`, `volume`  |
| Signal   | `rsi`                             |
| Decision | `rsi_oversold`                    |
| Strategy | `mean_reversion_entry`            |

When a new family is added to the codebase, its name must also be registered
in `internal/shared/settings/schema.go` (Go) for the validation to accept it.

## Cross-Layer Dependency Rules

### Signal depends on Evidence

Each signal family declares which evidence families it requires as input.
If the required evidence family is disabled, the signal family cannot produce
meaningful output and the config is rejected.

| Signal family | Required evidence families |
|---------------|---------------------------|
| `rsi`         | `candle`                  |

**Rule:** If `pipeline.signal_families` contains `"rsi"`, then
`pipeline.families` must either be empty (all enabled) or include `"candle"`.

### Decision depends on Signal

Each decision family declares which signal families it requires as input.
If the required signal family is disabled, the decision family cannot receive
signal events and the config is rejected.

| Decision family | Required signal families |
|-----------------|-------------------------|
| `rsi_oversold`  | `rsi`                   |

**Rule:** If `pipeline.decision_families` contains `"rsi_oversold"`, then
`pipeline.signal_families` must include `"rsi"`.

### Strategy depends on Decision

Each strategy family declares which decision families it requires as input.
If the required decision family is disabled, the strategy family cannot receive
decision events and the config is rejected.

| Strategy family        | Required decision families |
|------------------------|---------------------------|
| `mean_reversion_entry` | `rsi_oversold`            |

**Rule:** If `pipeline.strategy_families` contains `"mean_reversion_entry"`, then
`pipeline.decision_families` must include `"rsi_oversold"`.

### Transitive Dependencies

Strategy families transitively depend on signal and evidence families through
their decision dependencies. For example:

```
mean_reversion_entry → rsi_oversold (decision) → rsi (signal) → candle (evidence)
```

The validation enforces each hop independently. Enabling `mean_reversion_entry`
requires `rsi_oversold` in decisions, `rsi_oversold` requires `rsi` in signals,
and `rsi` requires `candle` in evidence.

## Cross-Service Consistency

Families must be enabled consistently across **derive** and **store** configs.
A family enabled in derive but missing in store means events are published but
never projected. A family in store but not derive means projections wait for
events that never arrive.

The raccoon-cli `runtime-bindings` analyzer enforces this by comparing
`deploy/configs/derive.jsonc` and `deploy/configs/store.jsonc`:

- Evidence: flagged when both have explicit lists and the lists differ
- Signal: flagged when one service enables a signal family the other does not
- Decision: flagged when one service enables a decision family the other does not
- Strategy: flagged when one service enables a strategy family the other does not

Gateway does not need family config — it discovers available families
dynamically from store's NATS KV buckets at startup.

## Where Validation Happens

| Validation | When | What |
|------------|------|------|
| `PipelineConfig.ValidatePipeline()` | Service startup (`settings.Load`) | Unknown family names, cross-layer dependencies |
| `raccoon-cli runtime-bindings` | CI / pre-deploy | Cross-service consistency (derive vs store) |

## Adding a New Family

To add a new family to the system:

1. Register the family name in `internal/shared/settings/schema.go`:
   - `knownEvidenceFamilies`, `knownSignalFamilies`, `knownDecisionFamilies`, or `knownStrategyFamilies`
2. If the family has cross-layer dependencies, add entries to:
   - `signalDependsOnEvidence`, `decisionDependsOnSignal`, or `strategyDependsOnDecision`
3. Add the family to both `deploy/configs/derive.jsonc` and `deploy/configs/store.jsonc`
4. Run `raccoon-cli runtime-bindings` to verify consistency
5. Run `go test ./internal/shared/settings/` to verify the name is accepted

## Failure Modes Prevented

| Scenario | Without validation | With validation |
|----------|-------------------|-----------------|
| Typo in family name (`"cnadle"`) | Silent no-op: sampler never spawns | Startup error: `unknown evidence family "cnadle"` |
| `rsi_oversold` without `rsi` | Decision evaluator spawns but never fires | Startup error: `decision family "rsi_oversold" requires signal family "rsi"` |
| `rsi` without `candle` | Signal sampler spawns but never receives candles | Startup error: `signal family "rsi" requires evidence family "candle"` |
| `mean_reversion_entry` without `rsi_oversold` | Strategy resolver spawns but never receives decisions | Startup error: `strategy family "mean_reversion_entry" requires decision family "rsi_oversold"` |
| `rsi` in derive but not store | Signal events published, never projected | raccoon-cli error: `signal family 'rsi' enabled in derive but not in store` |
