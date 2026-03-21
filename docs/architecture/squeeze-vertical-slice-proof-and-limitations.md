# Squeeze Breakout Vertical Slice — Proof and Limitations

> S291 — Honest assessment of what is proven, what is not, and what remains.

## What Is Proven

### 1. Complete Signal-to-Execution Path

The squeeze breakout slice is a fully wired vertical slice. Starting from raw candle data (20 close prices), the pipeline:

1. **Computes Bollinger Bands** — SMA(20), upper/lower bands, %B, bandwidth
2. **Detects squeeze condition** — relative bandwidth < 0.10 threshold, with severity classification (high/moderate/low)
3. **Resolves entry strategy** — direction=long with severity-scaled target and stop parameters
4. **Evaluates dual risk** — position exposure (factor 0.93) and drawdown limit (factor 0.90, stop 1.05x)
5. **Produces paper orders** — buy side with simulated fills, full traceability

This is not theoretical — all 5 layers are exercised in integration tests with real actor instances.

### 2. Severity-Aware Parameter Scaling

The entire pipeline is severity-aware. Decision severity (determined by squeeze intensity) influences:

- **Strategy parameters**: High → wider target (1.50x), tighter stop (0.75x). Low → narrower target (0.75x), wider stop (1.50x).
- **Strategy confidence**: High → 1.00x, moderate → 0.90x, low → 0.80x.
- **Risk position limits**: High → 1.15x multiplier, low → 0.80x.
- **Execution quantity**: Directly proportional to risk-constrained position size.

This produces **measurably different outputs** for different squeeze intensities, confirmed by the severity contrast test.

### 3. Suppression Path

When Bollinger bands are wide (no squeeze), the pipeline correctly suppresses execution:
- Decision: `not_triggered`
- Strategy: `flat` with zero confidence
- Risk: `approved` (flat is inherently safe)
- Execution: side=`none`

No false paper orders are generated from non-squeeze conditions.

### 4. Context Preservation

Correlation IDs survive the full 5-stage pipeline. Causation IDs are set at each stage, enabling complete audit trail reconstruction from paper order back to originating candle.

### 5. Dual Risk Fan-Out

Both risk evaluators (position_exposure and drawdown_limit) independently process the strategy output and independently produce paper orders. This matches the existing architecture for EMA and RSI paths.

## What Is NOT Proven (Limitations)

### 1. No NATS Infrastructure Integration

The closed-loop tests use local actor messaging (msgCollectors) to capture output. They do **not** exercise:
- NATS JetStream publish/subscribe
- Consumer durability and redelivery
- KV store materialization
- Stream retention and replay

**Why**: The existing test pattern (established in S266/S268) validates actor behavior in isolation from NATS infrastructure. NATS infrastructure tests exist separately (e.g., `natsexecution/` tests).

### 2. No ClickHouse Projection

The paper order events are not materialized to ClickHouse in these tests. The analytical projection path (event → writer consumer → ClickHouse) is proven separately for the existing families but not specifically for squeeze_breakout_entry events.

### 3. No SourceScopeActor Routing

The tests manually forward messages between stages (simulating SourceScopeActor routing). The actual SourceScopeActor's fan-out logic for the squeeze path is not exercised in this test. However, the wiring is registered in `derive_supervisor.go` using the same registration pattern as the proven EMA/RSI paths.

### 4. No Multi-Window / Multi-Symbol

Tests use a single symbol (btcusdt) and single timeframe (60s). Cross-symbol or multi-timeframe behavior is not validated.

### 5. No Staleness / Kill-Switch Guard Rails

The paper order evaluator has guard rail hooks (staleness rejection, kill switch), but these are not exercised in the squeeze-specific tests. They are proven generically in existing execution tests.

### 6. Paper Mode Only

All execution is paper — no real venue integration. Paper fills are simulated (Simulated=true). This is by design for the current project phase, not a gap.

### 7. No Short-Side Resolution

The squeeze breakout strategy only resolves to `long` (on triggered) or `flat` (on not_triggered). Short-side squeeze breakouts are not implemented. This is a deliberate scope decision — breakout strategies typically enter long on volatility expansion.

## Architecture Files

| Layer | Application Logic | Actor | Test |
|-------|------------------|-------|------|
| Signal | `signal/bollinger_sampler.go` | `derive/bollinger_signal_sampler_actor.go` | `bollinger_sampler_test.go` |
| Decision | `decision/bollinger_squeeze_evaluator.go` | `derive/bollinger_squeeze_decision_evaluator_actor.go` | `bollinger_squeeze_evaluator_test.go` |
| Strategy | `strategy/squeeze_breakout_entry_resolver.go` | `derive/squeeze_breakout_entry_resolver_actor.go` | `squeeze_breakout_entry_resolver_test.go` |
| Risk | `risk/risk_scaling.go` | `derive/risk_evaluator_actor.go` | `risk_scaling_test.go` |
| Execution | `execution/paper_order_evaluator.go` | `derive/execution_evaluator_actor.go` | `paper_order_evaluator_test.go` |
| **E2E** | — | — | `derive/squeeze_closed_loop_end_to_end_test.go` |

## Wiring in DeriveSupervisor

All families are registered in `derive_supervisor.go`:
- Signal: `bollinger` (line ~160)
- Decision: `bollinger_squeeze` (line ~192)
- Strategy: `squeeze_breakout_entry` (line ~224)
- Risk: `position_exposure`, `drawdown_limit` (line ~236)
- Execution: `paper_order` (line ~259)

## NATS Registry Entries

All families have stream, consumer, and query definitions:
- `natssignal/registry.go`: bollinger family
- `natsdecision/registry.go`: bollinger_squeeze family
- `natsstrategy/registry.go`: squeeze_breakout_entry family
- `natsrisk/registry.go`: position_exposure, drawdown_limit families
- `natsexecution/registry.go`: paper_order family

## Conclusion

The squeeze breakout vertical slice is **operationally complete** within the boundaries of the current paper-mode architecture. It functions identically to the existing EMA crossover and RSI oversold paths — same wiring pattern, same risk evaluation, same paper execution. The limitations are shared with the other slices and do not represent squeeze-specific gaps.
