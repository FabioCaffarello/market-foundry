# Full Closed-Loop Squeeze Breakout Scenario

> S291 — Validated end-to-end vertical slice proof for the squeeze breakout path.

## Scenario Overview

This document records the validated closed-loop scenario for the squeeze breakout slice, covering every layer from raw candle data to paper order fill.

### Flow Under Test

```
Candle (tight range, 20 bars)
  → BollingerSignalSampler → signal: bollinger (%B, bandwidth, sma)
    → BollingerSqueezeEvaluator → decision: bollinger_squeeze (triggered, severity, confidence)
      → SqueezeBreakoutEntryResolver → strategy: squeeze_breakout_entry (long, target_pct, stop_pct)
        → PositionExposureEvaluator → risk: position_exposure (approved, max_position)
        → DrawdownLimitEvaluator → risk: drawdown_limit (approved, stop_distance)
          → PaperOrderEvaluator → execution: paper_order (buy, filled, simulated)
```

## Validated Scenarios

### S291-1: Full Observability — Squeeze Triggered

**Input**: 20 candles with price range 100.00–100.19 (tight bandwidth → squeeze).

**Observed at each stage**:

| Stage | Type | Key Outputs |
|-------|------|-------------|
| Signal | bollinger | value=0.9119, bandwidth=0.2307, sma=100.0950 |
| Decision | bollinger_squeeze | outcome=triggered, severity=high, confidence=0.9885 |
| Strategy | squeeze_breakout_entry | direction=long, target=0.06, stop=0.01 |
| Risk/Exposure | position_exposure | disposition=approved, max_position=0.0227 |
| Risk/Drawdown | drawdown_limit | disposition=approved, stop_distance=0.0311 |
| Exec/Exposure | paper_order | side=buy, qty=0.0227, status=filled |
| Exec/Drawdown | paper_order | side=buy, qty=0.0575, status=filled |

**Result**: All 5 layers produced coherent, observable, traceable output. Correlation ID preserved end-to-end.

### S291-2: Suppression Path — Wide Bands (Not Triggered)

**Input**: Bollinger signal with bandwidth=50, sma=100 (relative bandwidth 0.50, well above 0.10 threshold).

**Observed**:
- Decision: `not_triggered`, severity=none
- Strategy: `flat`, confidence=0.0000
- Risk: `approved` (flat is safe)
- Execution: side=`none` (no paper order action)

**Result**: The suppression path works correctly — wide bands prevent any execution intent from being generated.

### S291-3: Severity Contrast — High vs Low

**Input**: Two runs with different squeeze intensities:
- High: bandwidth=1.0, sma=100 → relBW=0.01 (extreme compression)
- Low: bandwidth=8.0, sma=100 → relBW=0.08 (mild compression)

**Observed differences**:

| Metric | High Severity | Low Severity |
|--------|---------------|--------------|
| Decision severity | high | low |
| Strategy confidence | 0.9500 | 0.4800 |
| Breakout target | 0.06 (1.50x) | 0.03 (0.75x) |
| Breakout stop | 0.01 (0.75x) | 0.02 (1.50x) |
| Execution quantity | 0.0218 | 0.0077 |

**Result**: Severity produces observably different outputs at every stage, confirming the severity-aware scaling pipeline is live.

### S291-4: Context Preservation

**Input**: Full signal-to-execution chain with tagged correlation ID.

**Verified**: Correlation ID `sq4-trace-preservation` survived all 5 stages:
- Signal event metadata
- Decision event metadata + causation ID set
- Strategy event metadata
- Risk event metadata
- Execution event metadata + intent-level correlation + causation

**Result**: Full causation/correlation chain is preserved, enabling audit trail reconstruction.

## NATS Subjects Exercised

| Stage | Subject |
|-------|---------|
| Signal | `signal.events.bollinger.generated` |
| Decision | `decision.events.bollinger_squeeze.evaluated` |
| Strategy | `strategy.events.squeeze_breakout_entry.resolved` |
| Risk | `risk.events.position_exposure.assessed` |
| Risk | `risk.events.drawdown_limit.assessed` |
| Execution | `execution.events.paper_order.submitted` |

## Test File

`internal/actors/scopes/derive/squeeze_closed_loop_end_to_end_test.go`

All 4 scenarios pass with `go test ./internal/actors/scopes/derive/ -run TestSqueezeClosedLoop -v`.
