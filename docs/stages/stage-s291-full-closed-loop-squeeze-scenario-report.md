# Stage S291 — Full Closed-Loop Squeeze Scenario Report

**Status**: COMPLETE
**Date**: 2025-03-21
**Scope**: End-to-end vertical slice validation for squeeze breakout path

## Objective

Validate that the squeeze breakout slice (bollinger signal → bollinger_squeeze decision → squeeze_breakout_entry strategy → risk → paper execution) operates as a coherent, complete vertical slice — not a set of disconnected components.

## Deliverables

| Deliverable | Status | Path |
|-------------|--------|------|
| Closed-loop e2e test | Done | `internal/actors/scopes/derive/squeeze_closed_loop_end_to_end_test.go` |
| Scenario documentation | Done | `docs/architecture/full-closed-loop-squeeze-scenario.md` |
| Proof and limitations | Done | `docs/architecture/squeeze-vertical-slice-proof-and-limitations.md` |
| Stage report | Done | `docs/stages/stage-s291-full-closed-loop-squeeze-scenario-report.md` |

## Test Results

4 scenarios, all passing:

| Test | What It Proves |
|------|---------------|
| `TestSqueezeClosedLoop_Triggered_FullObservability` | Full 5-layer path from candles to paper buy order with stage-by-stage observability |
| `TestSqueezeClosedLoop_NotTriggered_Suppression` | Wide bands correctly suppress execution (not_triggered → flat → none) |
| `TestSqueezeClosedLoop_SeverityContrast_HighVsLow` | Different squeeze intensities produce measurably different outputs at every stage |
| `TestSqueezeClosedLoop_ContextPreservation` | Correlation ID survives all 5 stages; causation IDs set at each boundary |

## Evidence Summary

### Signal Layer
- Bollinger sampler produces valid %B, bandwidth, SMA, upper/lower after 20-candle warmup
- Signal metadata is complete and correct

### Decision Layer
- Tight bands (relBW=0.0023) → triggered, severity=high, confidence=0.9885
- Wide bands (relBW=0.50) → not_triggered, severity=none
- Squeeze threshold, relative bandwidth, %B zone enriched in metadata

### Strategy Layer
- Triggered → direction=long, entry=market, breakout_target_pct and breakout_stop_pct set
- Not triggered → direction=flat, confidence=0.0000
- High severity → target=0.06, stop=0.01; Low severity → target=0.03, stop=0.02
- Decision input (type, outcome, severity, rationale) preserved in strategy output

### Risk Layer
- Position exposure: factor 0.93 applied, severity multiplier active (1.15x for high)
- Drawdown limit: factor 0.90, stop factor 1.05x, severity multiplier active
- Both risk types approve squeeze_breakout_entry strategy
- Strategy type and decision severity preserved in risk output

### Execution Layer
- Approved long → side=buy, status=filled, 1 simulated fill
- Flat strategy → side=none (no action)
- Quantity scales with severity (high=0.0218, low=0.0077 for position_exposure path)
- Risk input carries strategy_type=squeeze_breakout_entry and decision_severity

## Acceptance Criteria

| Criterion | Met? | Evidence |
|-----------|------|----------|
| End-to-end proof exists | Yes | 4 passing tests covering triggered, suppression, severity contrast, context preservation |
| Slice not interrupted mid-architecture | Yes | All 5 layers produce output; no dead-end between decision and execution |
| Domain converts to operational flow | Yes | Raw candle prices → bollinger signal → squeeze decision → strategy → risk → paper order |
| Remaining limits clearly stated | Yes | `squeeze-vertical-slice-proof-and-limitations.md` documents 7 specific limitations |

## Guard Rails Compliance

| Guard Rail | Compliance |
|------------|-----------|
| No new family opened | Compliant — tests use existing bollinger, bollinger_squeeze, squeeze_breakout_entry families |
| No secondary scenario inflation | Compliant — 4 focused scenarios, each testing a distinct aspect |
| No artificial asserts | Compliant — all assertions verify real domain invariants |
| Focus on recém-delivered slice | Compliant — tests only exercise the squeeze breakout path |

## Files Changed

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/squeeze_closed_loop_end_to_end_test.go` | New — 4 closed-loop scenarios |
| `docs/architecture/full-closed-loop-squeeze-scenario.md` | New — scenario documentation |
| `docs/architecture/squeeze-vertical-slice-proof-and-limitations.md` | New — proof and limitations |
| `docs/stages/stage-s291-full-closed-loop-squeeze-scenario-report.md` | New — this report |

## Remaining Limitations

1. No NATS infrastructure integration (local actor messaging only)
2. No ClickHouse analytical projection for squeeze events
3. No SourceScopeActor routing validation (manual forwarding in tests)
4. Single symbol/timeframe only
5. No staleness/kill-switch guard rail exercise
6. Paper mode only (by design)
7. Long-side only (no short-side squeeze breakout)

## Preparation for S292

Recommended next focus areas:

1. **Cross-chain behavioral comparison**: Validate that squeeze breakout outputs are semantically distinct from existing EMA/RSI paths when running in parallel (distinct risk profiles, position sizes, parameter shapes).

2. **SourceScopeActor routing validation**: Prove that the SourceScopeActor correctly fans out bollinger signals to the squeeze decision evaluator and squeeze strategies to risk evaluators in the actual supervisor wiring.

3. **NATS infrastructure proof for squeeze families**: Validate stream creation, consumer registration, KV materialization, and query responses for bollinger_squeeze and squeeze_breakout_entry families.

4. **Writer pipeline integration**: Confirm squeeze-path events flow through the writer pipeline to ClickHouse with correct schema mapping.

5. **Multi-strategy coexistence**: Validate that squeeze_breakout_entry can run alongside mean_reversion_entry and trend_following_entry in the same derive scope without interference.
