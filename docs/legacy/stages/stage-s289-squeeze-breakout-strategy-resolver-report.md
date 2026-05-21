# Stage S289: Squeeze Breakout Strategy Resolver

## Status: COMPLETE

## Objective

Design, implement, validate, and document the `squeeze_breakout_entry` strategy resolver, producing a canonical decision-to-strategy path for the Bollinger squeeze breakout use case.

## Executive Summary

S289 delivers the third strategy family in the Foundry: `squeeze_breakout_entry`. This is a volatility-driven strategy that consumes `bollinger_squeeze` decision events and resolves them into actionable positional intent with severity-adjusted parameters. The implementation follows the canonical resolver pattern established by `mean_reversion_entry` and `trend_following_entry`, maintaining clean layer boundaries between decision and strategy.

## Deliverables

### Code Artifacts

| File | Purpose |
|---|---|
| `internal/application/strategy/squeeze_breakout_entry_resolver.go` | Pure application logic resolver |
| `internal/application/strategy/squeeze_breakout_entry_resolver_test.go` | 20 test cases covering all outcomes, severity scaling, parameter adjustment |
| `internal/actors/scopes/derive/squeeze_breakout_entry_resolver_actor.go` | Actor wrapper with NATS publish + scope fan-out |
| `internal/actors/scopes/derive/squeeze_breakout_entry_resolver_actor_test.go` | 6 actor-level integration tests |

### Registration Points Modified

| File | Change |
|---|---|
| `internal/shared/settings/schema.go` | Added `squeeze_breakout_entry` to `knownStrategyFamilies`, `strategyDependsOnDecision`; also added missing `trend_following_entry` entries |
| `internal/adapters/nats/natsstrategy/registry.go` | Added `SqueezeBreakoutEntryResolved`, `SqueezeBreakoutEntryLatest` specs, writer/store consumer specs, `LatestSpecByType` case |
| `internal/adapters/nats/natsstrategy/publisher.go` | Added `squeeze_breakout_entry` case to `specForType` |
| `internal/actors/scopes/derive/derive_supervisor.go` | Added `squeeze_breakout_entry` to `strategyProcessors` |
| `cmd/writer/pipeline.go` | Added writer pipeline entry for `squeeze_breakout_entry` |

### Documentation

| File | Purpose |
|---|---|
| `docs/architecture/squeeze-breakout-strategy-resolver-design.md` | Resolver design, semantics, data flow |
| `docs/architecture/squeeze-breakout-strategy-contracts-and-boundaries.md` | Contracts, ownership, layer boundaries |
| `docs/stages/stage-s289-squeeze-breakout-strategy-resolver-report.md` | This report |

## Test Results

- Application layer: 20/20 tests pass (confidence scaling, parameter adjustment, all outcomes, edge cases)
- Actor layer: 6/6 tests pass (triggered/flat/insufficient, severity propagation, fan-out with decision context)
- Settings validation: existing tests pass (no regression)
- Full build: all packages compile

## Design Decisions

1. **Volatility-driven semantics**: Unlike trend-following (pro-trend) or mean-reversion (counter-trend), squeeze breakout is based on bandwidth compression as a volatility regime signal. Parameters reflect breakout expectations rather than trend-riding or reversion targets.

2. **Parameter naming**: Used `breakout_target_pct` and `breakout_stop_pct` (not `target_offset`/`stop_offset` or `trailing_stop_pct`/`take_profit_pct`) to distinguish the semantic nature of the strategy.

3. **Base parameter values**: Target=0.04 (wider than mean-reversion's 0.02 because breakouts typically produce larger moves) and stop=0.015 (tighter than mean-reversion's 0.01 because squeeze failures invalidate quickly).

4. **trend_following_entry schema gap**: While implementing S289, discovered `trend_following_entry` was missing from `knownStrategyFamilies` and `strategyDependsOnDecision`. Fixed as part of this stage to ensure schema coherence.

## Acceptance Criteria Verification

| Criterion | Status |
|---|---|
| Canonical decision → strategy path exists | YES: `bollinger_squeeze` → `squeeze_breakout_entry` |
| Boundaries between decision and strategy are clean | YES: decision produces outcome/severity, strategy produces direction/parameters |
| Integration is coherent with existing architecture | YES: follows same resolver/actor/publisher/supervisor pattern |
| Slice is not left incomplete | YES: all registration points covered, tests passing |

## Guard Rails Compliance

| Constraint | Status |
|---|---|
| No risk/execution changes | COMPLIANT: no risk evaluator or execution changes |
| No decision-strategy coupling | COMPLIANT: primitive data interface maintained |
| No broad strategy layer redesign | COMPLIANT: additive change only |
| No premature abstractions | COMPLIANT: follows existing concrete patterns |

## Limits and Known Gaps

1. **Long-only**: Squeeze breakout currently resolves to `long` only. Short-side breakout would require a separate decision outcome or strategy family.
2. **No risk scaling factors**: Risk evaluators (`position_exposure`, `drawdown_limit`) do not yet have `squeeze_breakout_entry` in their strategy-type scaling maps. This is appropriate — risk integration is a separate stage.
3. **No codegen family spec**: The codegen yaml for `squeeze_breakout_entry` is not yet created. Writer pipeline entry is manually registered.

## Recommended S290 Preparation

**Option A — Risk Integration for Squeeze Breakout**:
Extend `position_exposure` and `drawdown_limit` risk evaluators to include `squeeze_breakout_entry` in their strategy-type confidence and stop factor maps. This completes the vertical slice from signal to risk.

**Option B — Codegen Spec for Squeeze Breakout**:
Create `codegen/families/squeeze_breakout_entry.yaml` and integrate into `codegen/integrated.yaml` with golden snapshot validation, converting the manual writer pipeline entry to codegen-governed.

**Option C — End-to-End Scenario Test**:
Create a closed-loop integration test that drives `bollinger` signal → `bollinger_squeeze` decision → `squeeze_breakout_entry` strategy through the full actor tree, validating the complete Bollinger squeeze slice.

Recommendation: **Option A** — risk integration is the natural next vertical step that completes the squeeze breakout slice without opening new lateral surface.
