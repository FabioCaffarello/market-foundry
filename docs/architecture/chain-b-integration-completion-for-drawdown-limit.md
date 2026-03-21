# Chain B Integration Completion for drawdown_limit

## Context

The breadth wave (S241-S244) introduced Chain B as a second parallel analytical pipeline:

```
Chain B: EMA Signal → ema_crossover Decision → trend_following_entry Strategy → Risk → Execution
```

While the derive supervisor fans out strategy results to **all** registered risk evaluators (both `position_exposure` and `drawdown_limit`), the chain integration test only validated the `position_exposure` path. This left `drawdown_limit` without end-to-end chain-level proof from signal origin through risk assessment.

## Gap Identified (D2 from S244)

| Debt | Description | Severity |
|------|-------------|----------|
| D2 | Integration test Chain B does not pass through `drawdown_limit` risk | Low |

The existing test `TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk` wired:

```
EMA bullish → ema_crossover (triggered) → trend_following_entry (long) → position_exposure (approved)
```

The `drawdown_limit` evaluator was proven in isolation (21 unit tests, 4 actor tests) but never as part of a full chain integration originating from an EMA signal.

## Resolution (S247)

Added `TestActorChain_EMACrossover_TrendFollowingEntry_To_DrawdownLimitRisk` in `actor_chain_integration_test.go`.

### Test Wiring

```
EMA bullish → ema_crossover (triggered) → trend_following_entry (long) → drawdown_limit (approved)
```

### Assertions Validated

| Stage | Assertion | Status |
|-------|-----------|--------|
| Decision | outcome = `triggered`, type = `ema_crossover` | PASS |
| Strategy | type = `trend_following_entry`, direction = `long` | PASS |
| Risk | type = `drawdown_limit` | PASS |
| Risk | disposition = `approved`, final = `true` | PASS |
| Risk | strategies[0].type = `trend_following_entry` | PASS |
| Risk | strategies[0].direction = `long` | PASS |
| Risk | strategies[0].decision_severity is non-empty | PASS |
| Risk | constraints.stop_distance is non-empty | PASS |
| Risk | correlation_id preserved end-to-end | PASS |
| Risk | `Validate()` passes | PASS |

### Confidence Scaling

The test confirms `drawdown_limit` applies its own confidence scaling (x0.90) independently from `position_exposure` (x0.95), producing `0.6750` for the same `0.7500` strategy confidence input.

## Before / After

| Aspect | Before (S244-S246) | After (S247) |
|--------|---------------------|--------------|
| Chain B integration with `position_exposure` | Proven | Proven |
| Chain B integration with `drawdown_limit` | Not proven (D2 open) | Proven (D2 closed) |
| Chain integration test count | 6 functions | 7 functions |
| `drawdown_limit` chain-level correlation_id proof | None | Full end-to-end |

## D2 Status

**Closed.** The `drawdown_limit` risk evaluator now participates in a full Chain B integration test with identical rigor to the `position_exposure` path.
