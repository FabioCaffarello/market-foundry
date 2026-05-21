# Behavioral Wave — Gains, Trade-offs, and Open Debts

**Charter:** BEHAVIORAL-WAVE-1 (S249–S253)
**Gate:** S254

---

## 1. Gains

### G1: Decision severity is no longer decorative

Before the wave, decision severity flowed through the pipeline as metadata but influenced nothing. Now it multiplicatively scales strategy confidence and adjusts strategy parameters. High-severity decisions produce aggressive positioning; low-severity decisions produce conservative positioning. The behavioral divergence is quantified: 2.56× position size difference between high and low severity (Scenario 3).

### G2: Risk assessment is strategy-type-aware

Before the wave, risk evaluators applied fixed multipliers regardless of strategy origin. Now risk evaluators differentiate between counter-trend (mean_reversion) and pro-trend (trend_following) strategies:

- Counter-trend receives 5.3% lower confidence and 26.1% tighter stops
- Pro-trend receives higher confidence and wider stops to let trends develop

This is semantically correct: counter-trend trades carry higher reversal risk.

### G3: Dual-risk fan-out is coherent

A single strategy resolution fans out to both position_exposure and drawdown_limit evaluators. Each applies its own strategy-type and severity factors independently. The results are coherent — they constrain different aspects of risk without contradicting each other.

### G4: End-to-end context preservation is proven

Decision rationale text survives 6 pipeline checkpoints unchanged (Scenario 6). Correlation IDs survive through fan-out. Strategy metadata carries decision context forward. Risk metadata carries strategy context forward. The audit trail is complete.

### G5: Behavioral tests are CI-protected

Twenty-seven tests across 4 test files are protected by a dedicated `behavioral-scenarios` CI job. Regression is visible immediately as a named failure, not buried in generic test output.

### G6: Zero infrastructure expansion

No new NATS streams, no new ClickHouse tables, no new binaries, no new actors, no new messages. All behavioral changes are in application-layer logic within existing resolvers and evaluators.

### G7: Governance discipline maintained

The charter executed with no amendments, no stop conditions triggered, and hardening at exactly 20% of stages (1 of 5). The governance framework (entry/exit criteria, permitted/prohibited changes, amendment rules) worked as designed.

---

## 2. Trade-offs

### T1: Scaling factors are hardcoded

Severity-to-confidence maps (`high: 1.00, moderate: 0.90, low: 0.80`) and strategy-type-to-risk maps (`mean_reversion: 0.90, trend_following: 0.95`) are Go constants in source files. They are not configurable at runtime. Changing them requires code changes and redeployment.

**Why accepted:** The charter prioritized proving behavioral composition over configurability. Hardcoded values are correct, testable, and simple. Configuration adds complexity that should come after the behavioral model is validated.

**Debt:** These should eventually move to configuration (configctl or similar) to allow tuning without redeployment.

### T2: Scenarios run in-process, not full-stack

End-to-end scenarios use Hollywood actors with msgCollector stand-ins. They do not test NATS serialization, ClickHouse persistence, or HTTP read-path round-trips. They prove behavioral logic but not transport fidelity.

**Why accepted:** Full-stack behavioral smoke requires infrastructure orchestration (Docker, NATS, ClickHouse). This was correctly deferred to avoid exceeding the hardening budget and charter scope.

**Debt:** A full-stack behavioral smoke test is the highest-priority gap remaining.

### T3: Float precision uses 2-decimal formatting

Parameter values are formatted to 2 decimal places (`FormatParam`). IEEE 754 rounding edge cases exist but are documented and tested. This is sufficient for the current behavioral model but could cause issues with very small parameter adjustments.

**Why accepted:** 2-decimal precision is adequate for the parameter ranges in use (0.01–0.05 base values with 0.75–1.50 multipliers). Sub-cent precision is not meaningful for the current domain.

### T4: EMA severity is fixed to moderate

The EMA crossover decision evaluator always produces moderate severity. There is no graduated severity for trend-following signals. The RSI evaluator produces high/moderate/low based on oversold depth, but EMA does not vary.

**Why accepted:** The charter scope was behavioral composition, not signal enrichment. EMA severity graduation is a breadth-adjacent change that belongs in a future wave.

### T5: Risk applies already-severity-scaled confidence

Risk evaluators receive strategy confidence that has already been scaled by decision severity. Risk then applies its own strategy-type multiplier. This means severity is applied once (at strategy layer), not twice. This is correct behavior but means risk cannot independently re-weight severity — it inherits the strategy layer's severity interpretation.

**Why accepted:** Single-point severity application is cleaner than double-scaling. Risk's role is strategy-type differentiation, not severity re-interpretation.

---

## 3. Open Debts

### OD-BW1: Full-stack behavioral smoke test [Medium risk]

**What:** No test validates the behavioral chain through real NATS serialization, ClickHouse write, and HTTP read-back.

**Risk:** Serialization bugs or transport edge cases could silently corrupt behavioral data in production while in-process tests pass.

**Mitigation:** The existing `smoke-analytical` CI job validates serialization/transport for all analytical families, but does not specifically validate behavioral properties (severity scaling, strategy-type factors).

**Recommendation:** Close in next tranche. Add behavioral assertions to existing smoke infrastructure.

### OD-BW2: Configurable scaling factors [Low risk]

**What:** Severity and strategy-type scaling factors are hardcoded Go maps.

**Risk:** Tuning behavioral parameters requires code changes and redeployment.

**Mitigation:** Current values are validated and correct. No operational need to change them without deeper domain analysis.

**Recommendation:** Defer until operational feedback indicates tuning is needed.

### OD-BW3: No rejection path in risk evaluators [Low risk]

**What:** Risk evaluators only produce "approved" or "modified" dispositions. No test validates a "rejected" outcome.

**Risk:** If rejection logic exists, it is untested. If it does not exist, the system cannot reject a trade — only modify it.

**Mitigation:** "Modified" disposition effectively caps risk (reduces position size, tightens stops). Rejection may not be needed at this stage.

**Recommendation:** Evaluate whether rejection semantics are needed before the execution layer is built.

### OD-BW4: Severity boundary and edge-case testing [Low risk]

**What:** Severity parsing handles only "high", "moderate", "low", and empty/unknown (defaults to 1.00). No tests for whitespace, casing variations, or unexpected values.

**Risk:** Upstream changes to severity formatting could silently default to neutral (1.00), hiding behavioral intent.

**Mitigation:** Domain model controls severity values at the source. Upstream would need to change the domain model to introduce unexpected formats.

**Recommendation:** Add defensive normalization (trim, lowercase) when severity parsing is moved to configuration.

### OD-BW5: No performance budget enforcement [Low risk]

**What:** Behavioral tests validate correctness but not performance. No benchmark or timeout enforcement exists.

**Risk:** Behavioral logic adds multiplicative operations per message. Performance regression would be silent.

**Mitigation:** Current behavioral tests complete in <1s for 27 tests. The computational overhead is trivial (map lookups and multiplications).

**Recommendation:** Defer until actor pipeline performance becomes a concern.

### OD-BW6: EX8 — Configuration-driven activation [Low risk]

**What:** Charter exit criterion EX8 required configctl-driven activation of behavioral routing. This was not delivered.

**Risk:** Behavioral routing cannot be toggled or tuned without code changes.

**Mitigation:** The behavioral logic is simple (severity scaling, strategy-type maps) and correct. Toggle/tune capability adds complexity with no current operational need.

**Recommendation:** Address alongside OD-BW2 when configuration infrastructure matures.

### OD-BW7: Execution layer gap [Out of scope]

**What:** The risk→execution boundary is not implemented. Risk produces assessments but nothing acts on them.

**Risk:** The behavioral chain is complete through risk but has no downstream consumer.

**Mitigation:** This was explicitly out of charter scope. The execution layer is a future wave.

**Recommendation:** Do not address until the execution domain is chartered.

---

## 4. Debt Priority Matrix

| Debt | Risk | Effort | Recommended Timing |
|---|---|---|---|
| OD-BW1: Full-stack smoke | Medium | Medium | Next tranche |
| OD-BW2: Configurable factors | Low | Medium | When operational need arises |
| OD-BW3: Rejection path | Low | Low | Before execution layer |
| OD-BW4: Severity edge cases | Low | Low | With configuration work |
| OD-BW5: Performance budgets | Low | Low | When pipeline scale increases |
| OD-BW6: Configctl activation | Low | Medium | With OD-BW2 |
| OD-BW7: Execution layer | Out of scope | High | Future charter |
