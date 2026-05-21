# Stage S242: Strategy Breadth Expansion — Report

## Executive Summary

S242 adds `trend_following_entry` as the second strategy type alongside the existing `mean_reversion_entry`, achieving real breadth (≥2 types/resolvers) in the strategy domain. The implementation follows the same formulaic expansion pattern established in S241 for decision breadth. All existing tests continue to pass; new tests cover the resolver, actor, and full chain integration. The strategy domain is now breadth-complete and ready for S243 (risk breadth expansion).

## Breadth Applied

### Before S242
| Domain   | Types                     | Breadth |
|----------|---------------------------|---------|
| Decision | rsi_oversold, ema_crossover | 2       |
| Strategy | mean_reversion_entry       | 1       |
| Risk     | position_exposure          | 1       |

### After S242
| Domain   | Types                                             | Breadth |
|----------|---------------------------------------------------|---------|
| Decision | rsi_oversold, ema_crossover                       | 2       |
| Strategy | mean_reversion_entry, **trend_following_entry**    | **2**   |
| Risk     | position_exposure                                  | 1       |

### Decision → Strategy Alignment
| Decision Type   | Strategy Type            | Philosophy     |
|-----------------|--------------------------|----------------|
| rsi_oversold    | mean_reversion_entry     | Counter-trend  |
| ema_crossover   | trend_following_entry    | Pro-trend      |

## Files Changed

### New Files
| File | Purpose |
|------|---------|
| `internal/application/strategy/trend_following_entry_resolver.go` | Pure resolver logic |
| `internal/application/strategy/trend_following_entry_resolver_test.go` | 12 test cases |
| `internal/actors/scopes/derive/trend_following_entry_resolver_actor.go` | Actor wrapper |
| `internal/actors/scopes/derive/trend_following_entry_resolver_actor_test.go` | 6 actor test cases |
| `codegen/families/trend_following_entry.yaml` | Codegen family definition |
| `docs/architecture/strategy-breadth-expansion.md` | Breadth expansion architecture |
| `docs/architecture/strategy-type-02-semantics-and-boundaries.md` | Type 02 semantics doc |

### Modified Files
| File | Change |
|------|--------|
| `internal/adapters/nats/natsstrategy/registry.go` | Added TrendFollowingEntry EventSpec, ControlSpec, consumer specs |
| `internal/adapters/nats/natsstrategy/kv_store.go` | Added `TrendFollowingEntryLatestBucket` constant |
| `internal/adapters/nats/natsstrategy/publisher.go` | Added `trend_following_entry` case in `specForType()` |
| `internal/actors/scopes/derive/derive_supervisor.go` | Registered trend_following_entry in strategyProcessors |
| `internal/actors/scopes/store/store_supervisor.go` | Added trend_following_entry projection pipeline |
| `cmd/writer/pipeline.go` | Added trend_following_entry writer pipeline |
| `internal/actors/scopes/derive/actor_chain_integration_test.go` | Added EMA→TrendFollowing→Risk chain test |

## Semantic and Boundary Gains

1. **Strategy now has real variety**: Two fundamentally different trading philosophies (counter-trend vs. pro-trend) coexist in the same domain model.

2. **Decision↔Strategy pairing is complete**: Every decision type now has a corresponding strategy type, creating a clean 1:1 mapping at the current breadth level.

3. **Domain isolation preserved**: `trend_following_entry` follows the same DBI-9 primitive-data contract. No new cross-domain imports were introduced.

4. **Shared infrastructure validated**: Both strategy types share the same NATS stream (`STRATEGY_EVENTS`), ClickHouse table (`strategies`), and projection actor. The polymorphic design works.

5. **End-to-end chain verified**: The integration test `TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk` proves the full pipeline: EMA signal → ema_crossover decision → trend_following_entry strategy → position_exposure risk.

## Limits and Trade-offs

1. **Fixed parameters**: Both strategy types use compile-time parameter defaults. No runtime tuning.
2. **Long-only bias**: Neither strategy type resolves to `short`. Short entries are deferred.
3. **No multi-decision aggregation**: Each strategy resolves from a single decision. Cross-signal combination is out of scope.
4. **No decision-type filtering in resolvers**: Resolvers act on outcome values, not decision types. This is intentionally simple but means a resolver could process decisions from any source.
5. **Shared stream, separate buckets**: Both types publish to `STRATEGY_EVENTS` but materialize to separate KV buckets for query isolation.

## Test Evidence

- **Application layer**: 12 new tests for `TrendFollowingEntryResolver` (all outcomes, validation, partition keys, metadata, severity)
- **Actor layer**: 6 new tests for `TrendFollowingEntryResolverActor` (triggered/not/insufficient/unknown, severity propagation, fan-out)
- **Integration**: 1 new chain test (`EMA→TrendFollowing→Risk`)
- **Regression**: All existing 40+ tests in `internal/actors/scopes/derive` continue to pass

## Preparation for S243 (Risk Breadth)

The risk domain currently has one evaluator (`position_exposure`). With strategy now providing two distinct types, S243 has a richer input surface:
- Both `mean_reversion_entry` and `trend_following_entry` feed into risk evaluation
- Different strategy parameters (fixed offsets vs. trailing stops) may warrant different risk assessment approaches
- The second risk type could focus on volatility-adjusted exposure, trend-specific drawdown limits, or correlation-based portfolio risk

The strategy domain is breadth-complete and stable. No further strategy changes are required for S243.

## Acceptance Criteria Verification

- [x] `strategy` has ≥2 resolvers/types (`mean_reversion_entry` + `trend_following_entry`)
- [x] New breadth is coherent with `decision` (clean 1:1 pairing)
- [x] Domain boundaries remain clear (DBI-9 primitive data, no cross-imports)
- [x] Base is ready for risk breadth expansion in S243
- [x] Breadth achieved without scope explosion (one type added, formulaic pattern)
