# Behavioral Scenarios: Protected Surface and Remaining Gaps

## Protected Surface (S253)

The following behavioral properties are now CI-protected via `make test-behavioral` and the `behavioral-scenarios` CI job:

### End-to-End Scenarios (6 tests)

| Test | Behavioral Property |
|------|---------------------|
| `TestScenario_RSIOversold_MeanReversion_DualRisk` | RSI oversold triggers mean reversion; fans out to both risk evaluators with strategy-type-aware scaling |
| `TestScenario_EMACrossover_TrendFollowing_DualRisk` | EMA crossover triggers trend following; fans out to both risk evaluators with pro-trend scaling |
| `TestScenario_SeverityContrast_HighVsLow` | Same chain at high vs low severity produces observably different outputs (2.56x position size, 1.79x confidence) |
| `TestScenario_CrossChain_RiskProfileComparison` | Counter-trend vs pro-trend chains produce asymmetric risk profiles (5.3% lower confidence, 26.1% tighter stop for counter-trend) |
| `TestScenario_NotTriggered_BothChains_FlatApproved` | Non-triggered decisions flow cleanly through both chains producing flat/approved assessments |
| `TestScenario_ContextPreservation_RationaleEndToEnd` | Decision rationale survives 6 checkpoints through full chain |

### Actor Chain Wiring (7 tests)

| Test | Behavioral Property |
|------|---------------------|
| `TestActorChain_Signal_To_Decision_To_Strategy_To_Risk` | Full 4-stage chain wiring with message forwarding |
| `TestActorChain_NotTriggered_FlowsThrough` | Not-triggered signals propagate correctly |
| `TestActorChain_EMACrossover_Bullish_Triggered` | EMA crossover bullish triggers correctly |
| `TestActorChain_EMACrossover_Bearish_NotTriggered` | EMA crossover bearish does not trigger |
| `TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk` | EMA → trend following → position exposure chain |
| `TestActorChain_EMACrossover_TrendFollowingEntry_To_DrawdownLimitRisk` | EMA → trend following → drawdown limit chain |
| `TestActorChain_CorrelationID_PreservedEndToEnd` | Correlation IDs survive actor-to-actor forwarding |

### Risk Scaling Behavior (11 tests)

| Test | Behavioral Property |
|------|---------------------|
| `TestPositionExposure_StrategyTypeConfidence` | Strategy type adjusts confidence factor (mean reversion 0.90, trend following 0.95) |
| `TestPositionExposure_SeverityAdjustsPositionLimit` | Decision severity scales position limit (high 1.20x, low 0.75x) |
| `TestPositionExposure_StrategyTypeInMetadata` | Strategy type recorded in risk metadata |
| `TestPositionExposure_RationaleIncludesStrategyType` | Rationale includes strategy type annotation |
| `TestPositionExposure_CombinedStrategyAndSeverity` | Combined strategy + severity produces multiplicative effect |
| `TestDrawdown_StrategyTypeConfidence` | Strategy type adjusts drawdown confidence (mean reversion 0.85, trend following 0.92) |
| `TestDrawdown_StrategyTypeAdjustsStopBase` | Counter-trend gets tighter stop (0.85x), pro-trend gets wider (1.15x) |
| `TestDrawdown_SeverityAdjustsDrawdownTolerance` | Severity scales drawdown tolerance (high 1.15x, low 0.80x) |
| `TestDrawdown_StrategyTypeInMetadata` | Strategy type recorded in drawdown metadata |
| `TestDrawdown_RationaleIncludesStrategyType` | Rationale includes strategy type annotation |
| `TestDrawdown_CombinedStrategyAndSeverity` | Combined strategy + severity produces multiplicative effect |

### Strategy Scaling Behavior (3 tests)

| Test | Behavioral Property |
|------|---------------------|
| `TestScaleConfidence` | Severity multiplier (high 1.10, moderate 1.00, low 0.85) with clamping |
| `TestAdjustParam` | Parameter adjustment by severity level |
| `TestFormatParam` | Parameter formatting for rationale |

**Total protected surface: 27 behavioral tests across 3 packages.**

## Remaining Gaps

### Not Yet CI-Protected

| Gap | Why | Risk Level |
|-----|-----|------------|
| **Full-stack behavioral smoke** | The behavioral tests run in-process without NATS/ClickHouse. A full-stack behavioral smoke (NATS → derive → writer → ClickHouse → reader → HTTP for behavioral families) would validate serialization and transport. | Medium — serialization is covered by codegen golden tests, but round-trip is not |
| **Multi-symbol behavioral isolation** | Scenarios test single-symbol chains. Multi-symbol behavioral isolation (two symbols running concurrently with independent severity) is not scenario-tested. | Low — unit tests cover multi-symbol ownership bleed |
| **Timeframe-aware behavioral variation** | Scenarios use fixed 60s timeframe. Behavioral variation across timeframes (e.g., severity thresholds that differ by timeframe) is not tested. | Low — no timeframe-dependent behavioral logic exists yet |
| **Execution layer behavioral chain** | The `decision → strategy → risk` chain is protected; `risk → execution` is not, as execution actors are not yet behavioral. | N/A — out of charter scope |
| **Behavioral regression thresholds** | Tests assert exact values (e.g., confidence 0.6075). A future enhancement could add golden-value snapshots with tolerance bands to detect behavioral drift without breaking on floating-point noise. | Low — current exact assertions are stable |
| **Performance regression** | Behavioral tests complete in ~1s total. No performance budget is enforced. A future enhancement could add `-timeout` or benchmark tests to detect latency regression. | Low — in-process tests are inherently fast |

### Explicitly Out of Scope

- **Observability/metrics hardening**: No Prometheus/Grafana assertions. This is infrastructure, not behavioral.
- **Load/stress testing**: Not proportional for this charter.
- **External API contract testing**: No upstream dependency contracts to protect.
- **Database migration behavioral testing**: ClickHouse schema is covered by `codegen-golden` and `smoke-analytical` jobs.

## How to Extend the Protected Surface

1. **New behavioral test**: Name it with one of the protected prefixes (`TestScenario_`, `TestActorChain_`, `TestPositionExposure_`, `TestDrawdown_`). It will automatically join the `make test-behavioral` surface.
2. **New behavioral package**: Add its path to `BEHAVIORAL_PACKAGES` in the Makefile and its test prefix to `BEHAVIORAL_PATTERN`.
3. **Full-stack behavioral smoke**: Add a new script in `scripts/` and a corresponding Makefile target + CI job (similar to `smoke-analytical`).
