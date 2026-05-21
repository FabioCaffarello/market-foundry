# Squeeze Slice Metrics: Semantics and Usage

## Counter Reference

All counters use the `healthz.Tracker.Counter()` mechanism and appear in `/statusz` and `/diagz` JSON responses under the `derive-publisher` tracker.

### Signal Layer

| Counter | Semantics | Incremented When |
|---------|-----------|------------------|
| `signal:bollinger` | Bollinger band signal published to NATS | BollingerSampler produces a finalized signal and it passes validation |
| `signal:rsi` | RSI signal published | RSISampler produces a finalized signal |
| `signal:ema_crossover` | EMA crossover signal published | EMACrossoverSampler produces a finalized signal |

### Decision Layer

| Counter | Semantics | Incremented When |
|---------|-----------|------------------|
| `decision:bollinger_squeeze:triggered` | Squeeze condition detected | BollingerSqueezeEvaluator returns `outcome=triggered` (bandwidth < threshold) |
| `decision:bollinger_squeeze:not_triggered` | No squeeze condition | BollingerSqueezeEvaluator returns `outcome=not_triggered` |
| `decision:rsi_oversold:triggered` | RSI oversold detected | RSIOversoldEvaluator returns triggered |
| `decision:ema_crossover:triggered` | EMA crossover detected | EMACrossoverEvaluator returns triggered |

### Strategy Layer

| Counter | Semantics | Incremented When |
|---------|-----------|------------------|
| `strategy:squeeze_breakout_entry:long` | Long entry signal | SqueezeBreakoutEntryResolver produces direction=long |
| `strategy:squeeze_breakout_entry:flat` | No trade (flat) | SqueezeBreakoutEntryResolver produces direction=flat |
| `strategy:mean_reversion_entry:long` | Mean reversion long | MeanReversionEntryResolver produces direction=long |
| `strategy:trend_following_entry:long` | Trend following long | TrendFollowingEntryResolver produces direction=long |

### Risk Layer

| Counter | Semantics | Incremented When |
|---------|-----------|------------------|
| `risk:position_exposure:approved` | Position exposure within limits | PositionExposureEvaluator disposition=approved |
| `risk:position_exposure:modified` | Position sized down | PositionExposureEvaluator disposition=modified |
| `risk:position_exposure:rejected` | Position exceeds limits | PositionExposureEvaluator disposition=rejected |
| `risk:drawdown_limit:approved` | Drawdown within tolerance | DrawdownLimitEvaluator disposition=approved |
| `risk:drawdown_limit:modified` | Stop distance adjusted | DrawdownLimitEvaluator disposition=modified |
| `risk:drawdown_limit:rejected` | Drawdown limit breached | DrawdownLimitEvaluator disposition=rejected |

### Execution Layer

| Counter | Semantics | Incremented When |
|---------|-----------|------------------|
| `execution:paper_order:buy` | Buy intent submitted | PaperOrderEvaluator produces side=buy |
| `execution:paper_order:sell` | Sell intent submitted | PaperOrderEvaluator produces side=sell |
| `execution:paper_order:none` | No-op intent (risk rejected/flat) | PaperOrderEvaluator produces side=none |
| `execution:paper_order:submitted` | Intent in submitted status | Status recorded at publish time |
| `execution:paper_order:filled` | Intent filled (paper simulation) | PaperFillSimulator promotes to filled |
| `execution:gate_halted` | Publish blocked by control gate | ExecutionPublisherActor gate check returns halted |

### Existing Counters (Unchanged)

| Counter | Semantics |
|---------|-----------|
| `published:<symbol>` | Total events published for a symbol across all types |

## Operational Usage

### Quick Health Check

```bash
curl -s http://localhost:8081/statusz | jq '.trackers[] | select(.name == "derive-publisher") | .counters'
```

### Squeeze Path Flow Verification

Check that signals flow through the full slice:
```bash
# Expect: signal:bollinger >= decision:bollinger_squeeze:* >= strategy:squeeze_breakout_entry:* >= execution:paper_order:*
curl -s http://localhost:8081/statusz | jq '
  .trackers[] | select(.name == "derive-publisher") | .counters |
  {
    signals: .["signal:bollinger"] // 0,
    squeeze_triggered: .["decision:bollinger_squeeze:triggered"] // 0,
    squeeze_not_triggered: .["decision:bollinger_squeeze:not_triggered"] // 0,
    strategy_long: .["strategy:squeeze_breakout_entry:long"] // 0,
    strategy_flat: .["strategy:squeeze_breakout_entry:flat"] // 0,
    risk_approved: (.["risk:position_exposure:approved"] // 0) + (.["risk:drawdown_limit:approved"] // 0),
    risk_rejected: (.["risk:position_exposure:rejected"] // 0) + (.["risk:drawdown_limit:rejected"] // 0),
    exec_buy: .["execution:paper_order:buy"] // 0,
    exec_filled: .["execution:paper_order:filled"] // 0,
    gate_halted: .["execution:gate_halted"] // 0
  }'
```

### Detecting Silent Failures

If `signal:bollinger` grows but `decision:bollinger_squeeze:*` stays flat, the decision evaluator is not receiving signals (possible routing issue).

If `strategy:squeeze_breakout_entry:long` grows but `execution:paper_order:buy` does not, risk is rejecting all entries.

If `execution:paper_order:buy` grows but `execution:paper_order:filled` does not, the paper fill simulator is failing.

### Gate Block Detection

If `execution:gate_halted` is non-zero, the execution control gate is active and blocking paper orders. Check with:
```bash
curl -s http://localhost:8081/statusz | jq '.trackers[] | select(.name == "derive-publisher") | .counters["execution:gate_halted"]'
```
