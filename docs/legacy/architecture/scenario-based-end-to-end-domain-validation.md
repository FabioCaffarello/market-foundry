# Scenario-Based End-to-End Domain Validation

## Purpose

This document defines the scenario-based validation approach used in S252 to prove that the `decision -> strategy -> risk` chain produces coherent, observable, and auditable behavioral output — not merely that individual types exist or respond in isolation.

## Motivation

Stages S241–S244 delivered breadth: new domain types, evaluators, and resolvers across decision, strategy, and risk. Stages S250–S251 enriched cross-domain behavior: severity-aware scaling, strategy-type-aware risk profiles, and rationale propagation.

Without end-to-end scenario validation, these enrichments remain locally verified but globally unproven. A system where each unit test passes but the chain produces incoherent output is not valuable.

## Scenario Selection Criteria

Scenarios were selected to maximize behavioral coverage with minimum combinatorial explosion:

1. **Representative chains only** — two canonical chains cover the full decision × strategy × risk matrix:
   - Chain A: `rsi_oversold -> mean_reversion_entry -> [position_exposure, drawdown_limit]`
   - Chain B: `ema_crossover -> trend_following_entry -> [position_exposure, drawdown_limit]`

2. **Severity contrast** — same chain exercised at different severity levels (high vs low) to prove behavioral divergence.

3. **Cross-chain comparison** — both chains evaluated by the same risk evaluator to prove strategy-type-aware differentiation.

4. **Negative path** — non-triggered decisions flow cleanly through both chains to produce approved flat assessments.

5. **Context preservation** — decision rationale survives the full pipeline for auditability.

## Validation Architecture

### Test Infrastructure

Tests use the existing Hollywood actor engine with `msgCollector` actors as stand-ins for publishers and scope fan-out. This mirrors the production `SourceScopeActor` routing behavior without requiring NATS or the full scope supervisor.

```
signalGeneratedMessage
    ↓
DecisionEvaluatorActor (RSIOversold or EMACrossover)
    ├→ publishDecisionMessage → decisionPub collector
    └→ decisionEvaluatedMessage → decFanout collector
                                      ↓ (manual forward)
StrategyResolverActor (MeanReversion or TrendFollowing)
    ├→ publishStrategyMessage → strategyPub collector
    └→ strategyResolvedMessage → stratFanout collector
                                      ↓ (manual forward to BOTH)
RiskEvaluatorActor (PositionExposure)  ← strategyResolvedMessage
RiskEvaluatorActor (DrawdownLimit)     ← strategyResolvedMessage
    ↓                                      ↓
riskPubExposure collector          riskPubDrawdown collector
```

### Dual-Risk Fan-Out

In production, `SourceScopeActor` fans out `strategyResolvedMessage` to all risk evaluators for a symbol. In tests, this is simulated by sending the same message to both risk evaluator PIDs — proving that a single strategy resolution produces two independent, coherent risk assessments.

## Scenarios Defined

### Scenario 1: RSI Oversold → Mean Reversion → Dual Risk

| Stage | Input | Output |
|-------|-------|--------|
| Decision | RSI 10.0 | triggered, severity=high, confidence=0.8333 |
| Strategy | triggered, high severity | long, confidence=0.8333 (×1.00), target=0.03, stop=0.01 |
| Risk (exposure) | mean_reversion, high | approved, confidence=0.7500 (×0.90), position=0.0192 |
| Risk (drawdown) | mean_reversion, high | approved, confidence=0.7083 (×0.85), stop=0.0213 |

**Validates:** Full chain coherence, dual-risk assessment, severity-adjusted parameters, strategy-type-aware risk factors.

### Scenario 2: EMA Crossover → Trend Following → Dual Risk

| Stage | Input | Output |
|-------|-------|--------|
| Decision | bullish | triggered, severity=moderate, confidence=0.7500 |
| Strategy | triggered, moderate | long, confidence=0.6750 (×0.90), trailing_stop=0.03, take_profit=0.05 |
| Risk (exposure) | trend_following, moderate | approved, confidence=0.6412 (×0.95), position=0.0135 |
| Risk (drawdown) | trend_following, moderate | approved, confidence=0.6210 (×0.92), stop=0.0233 |

**Validates:** Pro-trend chain through both evaluators, moderate severity defaults, strategy-type factors.

### Scenario 3: Severity Contrast (High vs Low)

| Metric | RSI 10.0 (high) | RSI 25.0 (low) |
|--------|-----------------|-----------------|
| Decision severity | high | low |
| Strategy confidence | 0.8333 (×1.00) | 0.4666 (×0.80) |
| Risk confidence | 0.7500 | 0.4199 |
| Position size | 0.0192 | 0.0075 |
| Severity limit factor | 1.15 | 0.80 |

**Validates:** Severity produces observably different numerical outcomes — high severity gets ~2.6× larger position and ~1.8× higher confidence than low severity.

### Scenario 4: Cross-Chain Risk Profile

| Metric | Mean Reversion (counter-trend) | Trend Following (pro-trend) |
|--------|-------------------------------|----------------------------|
| Risk confidence factor | 0.90 | 0.95 |
| Strategy type in metadata | mean_reversion_entry | trend_following_entry |

**Validates:** Counter-trend strategies receive more conservative risk treatment than pro-trend strategies from the same risk evaluator.

### Scenario 5: Not-Triggered (Both Chains)

| Chain | Decision | Strategy | Risk |
|-------|----------|----------|------|
| A (RSI 75.0) | not_triggered | flat, 0.0000 | approved, 1.0000 |
| B (EMA bearish) | not_triggered | flat, 0.0000 | approved, 1.0000 |

**Validates:** Non-triggered decisions flow cleanly through both chains without errors or unexpected state.

### Scenario 6: Context Preservation

**Validates:** Decision rationale text (e.g., "RSI 15.0000 below oversold threshold 30.0 (distance 50.0%); severity moderate") survives unchanged in:
- `decisionEvaluatedMessage.DecisionRationale` (fan-out)
- `Strategy.Decisions[0].Rationale` (DecisionInput)
- `Strategy.Metadata["decision_rationale"]` (metadata)
- `strategyResolvedMessage.DecisionRationale` (strategy fan-out)
- `RiskAssessment.Strategies[0].DecisionRationale` (StrategyInput)
- `RiskAssessment.Metadata["decision_rationale"]` (risk metadata)

## File Location

All scenario tests live in:
```
internal/actors/scopes/derive/scenario_end_to_end_test.go
```

## Relationship to Existing Tests

| File | Purpose | S252 Gap |
|------|---------|----------|
| `actor_chain_integration_test.go` | Individual chain wiring | No dual-risk, no severity contrast, no cross-chain comparison |
| `*_evaluator_test.go`, `*_resolver_test.go` | Unit-level per-type | No chain integration |
| `scenario_end_to_end_test.go` (S252) | Behavioral coherence across chains | Fills all gaps above |

## Limits

- Scenarios do not include real NATS or ClickHouse infrastructure — they validate behavioral logic via actor messages.
- The `SourceScopeActor` fan-out is simulated manually; full scope-level integration is deferred to S253.
- Execution evaluators (paper_order) are not included in these scenarios — the chain ends at risk assessment.
- Only two chains are validated; additional decision types (if added later) will need their own scenarios.
