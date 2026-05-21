# Closed-Loop Scenarios: Results and Operational Findings

## Executive Summary

Five closed-loop scenarios were designed and validated, covering both strategy families (mean reversion, trend following), both risk evaluators (position exposure, drawdown limit), three severity levels (high, low, none), and both action/no-action paths. All 5 scenarios pass, proving the Market Foundry domain pipeline produces coherent, auditable, operationally meaningful paper execution from signal intelligence.

## Scenario Results

### Closed Loop A: Mean Reversion Full Observability — PASS

**Pipeline trace** (all values from actual test output):

```
Signal:     RSI 10.0000 (extreme oversold)
Decision:   outcome=triggered  severity=high  confidence=0.8333
            rationale="RSI 10.0000 below oversold threshold 30.0 (distance 66.7%); severity high"
Strategy:   type=mean_reversion_entry  direction=long  confidence=0.8333
            target_offset=0.03 (base 0.02 × 1.50)  stop_offset=0.01
Risk/exp:   disposition=approved  confidence=0.7500  max_position=0.0192
Risk/dd:    disposition=approved  confidence=0.7083  stop_distance=0.0212
Exec/exp:   side=buy  qty=0.0192  status=filled  fills=1 (simulated)
Exec/dd:    side=buy  qty=0.0575  status=filled  fills=1 (simulated)
```

**Findings**:
- Dual risk fan-out produces two independent paper orders with different quantities (0.0192 vs 0.0575).
- Position exposure is more conservative than drawdown limit for counter-trend strategies.
- CorrelationID `cla-mr-full-obs` preserved across all 4 stages.

### Closed Loop B: Trend Following Full Observability — PASS

```
Signal:     EMA crossover bullish
Decision:   outcome=triggered  severity=moderate  confidence=0.7500
Strategy:   type=trend_following_entry  direction=long  confidence=0.6750
            trailing_stop_pct=0.03  take_profit_pct=0.05
Risk/exp:   disposition=approved  confidence=0.6412  max_position=0.0135
Risk/dd:    disposition=approved  confidence=0.6210  stop_distance=0.0233
Exec/exp:   side=buy  qty=0.0135  status=filled  fills=1 (simulated)
Exec/dd:    side=buy  qty=0.0500  status=filled  fills=1 (simulated)
```

**Findings**:
- Trend following produces different parameter families (trailing_stop, take_profit) vs mean reversion (target_offset, stop_offset).
- Pro-trend confidence factor (0.95) produces slightly lower position than counter-trend (0.90) when combined with moderate severity.
- Strategy-type distinction preserved through all stages.

### Closed Loop C: Severity Contrast at Every Stage — PASS

```
High (RSI 10):  decision=high  strategy_conf=0.8333  risk_pos=0.0192  exec_qty=0.0192
Low  (RSI 25):  decision=low   strategy_conf=0.4666  risk_pos=0.0075  exec_qty=0.0075
Ratio:          —              1.79×                  2.56×            2.56×
```

**Findings**:
- Severity influence is observable and monotonically increasing at every stage.
- The 2.56× quantity ratio is a compound effect of severity scaling at strategy (confidence), risk (position limit factor), and execution (quantity = position size).
- This is not a trivial pass-through — each stage amplifies or attenuates the severity signal.

### Closed Loop D: No-Signal Suppression — PASS

```
Signal:     RSI 75.0000 (not oversold)
Decision:   outcome=not_triggered  severity=none  confidence=0.8214
Strategy:   direction=flat  confidence=0.0000
Risk:       disposition=approved  confidence=1.0000 (trivially safe)
Execution:  side=none  quantity=0  status=submitted  fills=0
```

**Findings**:
- Non-triggered signals correctly suppress at every stage: no direction, no position, no fills.
- The system still produces observable events (all 4 stages fire), enabling audit and monitoring.
- Risk trivially approves (confidence=1.0) because no position is requested.
- Execution produces a no-action intent (side=none, qty=0) rather than silently dropping the message.

### Closed Loop E: Cross-Chain Behavioral Distinction — PASS

```
Mean Reversion: qty=0.0192 (strategy=mean_reversion_entry sev=high)
Trend Following: qty=0.0135 (strategy=trend_following_entry sev=moderate)
```

**Findings**:
- Different decision types (rsi_oversold vs ema_crossover) feed different strategy families.
- Strategy parameters are semantically distinct: target_offset vs trailing_stop_pct.
- Risk confidence factors differ by strategy type (0.90 vs 0.95).
- Execution preserves strategy type and decision severity for full auditability.

## Operational Findings

### 1. The Pipeline is Operationally Complete in Paper Mode

The Foundry can transform signal intelligence into paper execution across both strategy families, both risk evaluators, and all severity levels. This is the first time the full loop has been proven as a single coherent unit.

### 2. Observability is Structural, Not Bolted-On

Every intermediate stage produces a fully typed domain event with:
- Typed fields (not generic maps) for all behavioral parameters
- CorrelationID and CausationID for trace reconstruction
- Metadata carrying upstream context (decision_severity, strategy_type, confidence_factors)
- Domain validation (`Validate()`) ensuring structural integrity

### 3. Severity is an Active Behavioral Driver

Decision severity is not metadata — it actively influences:
- Strategy confidence (×1.0, ×0.9, ×0.8)
- Strategy parameters (target, stop offsets)
- Risk position limits (×1.15, ×1.00, ×0.80)
- Final execution quantities (2.56× ratio between high and low)

### 4. Negative Paths Close Correctly

The no-signal suppression test proves:
- The system never silently drops messages
- Non-triggered signals produce auditable no-action events
- Every stage fires even when there is nothing to do
- This enables monitoring and alerting on pipeline health

### 5. Dual Risk Fan-Out Produces Independent Outputs

A single strategy fans to both position_exposure and drawdown_limit evaluators, each producing independent paper orders with different quantities based on their risk model. This is correct behavior — in production, order aggregation would happen downstream.

## Limitations and Simplifications

1. **No venue adapter**: Paper fills are instant and simulated. Real venue latency, partial fills, and rejections are not tested.
2. **No SafetyGate integration**: Kill switch and staleness guard are not wired in these scenarios. They exist in code but are not proven end-to-end.
3. **No KV materialization**: Published events are captured by test collectors, not persisted to NATS KV stores.
4. **No ClickHouse round-trip**: Events are not written to ClickHouse or queried via the reader/gateway path.
5. **Single symbol**: All scenarios use btcusdt@60s. Multi-symbol isolation is not tested here.
6. **No concurrent scenarios**: Scenarios run sequentially, not concurrently. Race conditions are not tested.
7. **Static signal values**: Signals are injected with known values, not computed from candle data by samplers.

## Before/After Value Assessment

### Before S268
- Domain intelligence existed at every stage but had never been proven as a single coherent loop.
- Paper order generation was validated (S266) but only the final output was asserted.
- Intermediate stages were assumed correct based on unit tests; no cross-stage behavioral proof existed.
- The pipeline had "no operational exit" — domain enrichment without proven operational closure.

### After S268
- First complete closed-loop proof from signal to paper execution.
- Every intermediate stage is observable and asserted in behavioral scenarios.
- Severity, strategy type, and causal context are proven to survive every stage boundary.
- Negative paths close correctly with auditable no-action events.
- The pipeline is ready for gate evaluation (S269) with concrete evidence.
