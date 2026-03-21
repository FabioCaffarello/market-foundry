# Paper Execution Wave: Gains, Trade-offs, and Open Debts

**Stage:** S269
**Wave:** Paper Execution (S264–S268)
**Date:** 2026-03-21

---

## 1. Gains

### 1.1 First Operational Loop Closed

The Foundry now has a proven closed loop from signal intelligence to paper execution:

```
Signal → Decision → Strategy → Risk → Execution (paper)
```

This is the first time the system produces an actionable output — a paper order — from raw market intelligence. Before this wave, the pipeline terminated at risk assessment.

| Metric | Before (S263) | After (S268) | Delta |
|--------|---------------|--------------|-------|
| Pipeline terminus | Risk assessment | Paper order (filled) | +1 layer |
| End-to-end scenarios | 0 (chain stopped at risk) | 12 (S266: 7 + S268: 5) | +12 |
| Closed-loop scenarios | 0 | 5 | +5 |
| Boundary gaps fixed | N/A | 3 (S265) | +3 |
| Production code changes in execution | N/A | 6 files (S265) | Foundational |
| Execution domain tests | Unit only | Unit + integration + end-to-end + closed-loop | Full stack |

### 1.2 Behavioral Proof Through the Full Pipeline

Severity is no longer a label — it is a behavioral driver that shapes every stage:

| Stage | High Severity | Low Severity | Effect |
|-------|--------------|-------------|--------|
| Decision | severity=high | severity=low | Classification |
| Strategy | confidence=0.8333, target×1.50, stop×0.75 | confidence=0.4666, target×0.75, stop×1.50 | Parameter scaling |
| Risk (position) | qty=0.0192, factor=0.90×1.15 | qty=0.0075, factor=0.90×0.80 | Position sizing |
| Risk (drawdown) | qty=0.0575, tolerance×1.15 | qty=0.0225, tolerance×0.80 | Drawdown limits |
| Execution | filled paper buy, qty=0.0192 | filled paper buy, qty=0.0075 | Order generation |

The 2.56× quantity ratio between high and low severity is the single strongest evidence that the Foundry's behavioral infrastructure is not decorative.

### 1.3 Dual Risk Fan-Out

A single strategy assessment fans out to both risk evaluators independently:

- **Position exposure**: conservative sizing with confidence factor 0.90 (mean reversion) or 0.95 (trend following).
- **Drawdown limit**: wider tolerance with strategy-specific stop distance factors.

Each produces an independent paper order. This proves the risk layer handles strategy context without coupling evaluators to each other.

### 1.4 Cross-Chain Behavioral Distinction

Mean reversion and trend following produce observably different outputs at every stage:

| Aspect | Mean Reversion | Trend Following |
|--------|---------------|-----------------|
| Strategy family | mean_reversion_entry | trend_following_entry |
| Parameters | target_offset, stop_offset | trailing_stop_pct, take_profit_pct |
| Risk confidence factor | 0.90 (conservative) | 0.95 (trend-aligned) |
| Drawdown stop factor | 0.85 (tighter) | 1.15 (wider) |
| Paper order quantity | 0.0192 | 0.0135 |

### 1.5 Contract Alignment

S265 fixed three concrete boundary gaps:

1. **StrategyType** was dropped at risk → execution boundary — now preserved in `RiskInput`.
2. **DecisionSeverity** was dropped at risk → execution boundary — now preserved in `RiskInput`.
3. **Drawdown StopDistance** was semantically mismatched as `MaxPositionPct` — now correctly maps to `MaxExposure`.

These fixes are small in code but significant in correctness: without them, execution would have operated with incomplete or wrong context.

### 1.6 Negative Path Closure

Non-triggered signals produce auditable no-action events through the entire pipeline:

```
RSI 75 → not_triggered → flat → approved (trivially safe) → side=none, qty=0, fills=0
```

This proves the system doesn't just generate orders — it correctly suppresses non-signals with full observability.

### 1.7 Causal Traceability

CorrelationID and CausationID survive all 4 stage boundaries in every scenario. This is structural traceability — built into domain types, not bolted-on logging.

---

## 2. Trade-offs

### 2.1 Paper Fills Are Instant and Deterministic

**What we gained:** Simplicity and determinism in testing. Paper fills transition immediately from submitted to filled with a single `FillRecord`.

**What we gave up:** Realism. Real venues introduce latency, partial fills, rejections, requotes, and status transitions that the paper simulator doesn't model. The 5-status lifecycle (`submitted → sent → accepted → filled/rejected`) exists in the domain but only `submitted → filled` is exercised.

**Severity:** Low for paper execution proof. Medium if the Foundry attempts to infer production behavior from paper results.

### 2.2 SafetyGate Not Wired Into End-to-End Path

**What we gained:** Faster delivery of the core domain loop. The closed-loop scenarios focus on behavioral proof without safety-check plumbing.

**What we gave up:** End-to-end proof that the kill switch and staleness guard actually prevent order submission. SafetyGate is unit-tested (16 tests) and integration-tested (pipeline), but the actor chain bypasses it.

**Severity:** High. This is the most consequential trade-off. Safety mechanisms that aren't proven in the actual execution path are not trustworthy by definition.

### 2.3 Single Symbol, Static Signals

**What we gained:** Focused scenarios that isolate behavioral proof from signal computation complexity.

**What we gave up:** Confidence that the pipeline handles multi-symbol isolation under real conditions, and that quantities scale correctly with different price ranges.

**Severity:** Low. Multi-symbol isolation is proven structurally (partition keys, dedup keys) even though only one symbol is used in scenarios.

### 2.4 Ten-Parameter Evaluate() Signature

**What we gained:** Full context propagation from risk to execution — all 6 `RiskInput` fields, plus disposition, direction, and sizing.

**What we gave up:** API ergonomics. The `Evaluate()` function takes 10 positional parameters: `disposition, direction, riskType, confidence, maxPositionPct, strategyType, decisionSeverity, symbol, timeframe, timestamp`.

**Severity:** Low. Only 2 call sites exist (actor + tests). A parameter object refactor is available if call sites grow.

### 2.5 No KV/ClickHouse Proof for Execution Events

**What we gained:** Clean separation between domain correctness proof and infrastructure integration proof.

**What we gave up:** Confidence that paper orders are observable outside the actor chain. ClickHouse behavioral round-trip tests exist for decision/strategy/risk (S255) but do not cover execution events.

**Severity:** Medium. Observability is not optional for production paper trading.

---

## 3. Open Debts

### 3.1 Paper Execution Wave Debts

| ID | Description | Severity | Impact | Recommended Action |
|----|-------------|----------|--------|-------------------|
| OD-PE1 | SafetyGate not wired into actor/closed-loop path | High | Kill switch and staleness guard not enforced in actual execution flow | Wire into PaperOrderEvaluatorActor before next wave |
| OD-PE2 | S267 formal stage report missing | Medium | Governance gap — work was done but not formally documented | Produce retroactive report or fold into gate record |
| OD-PE3 | KV materialization unproven for execution events | Medium | Paper orders published as events but not proven in KV buckets | Add integration test for KV materialization |
| OD-PE4 | ClickHouse round-trip unproven for paper execution | Medium | Execution events not included in behavioral round-trip validation | Extend behavioral_roundtrip_test.go to cover execution |
| OD-PE5 | ControlGate runtime kill switch not proven end-to-end | Medium | Kill switch exists but toggle behavior not demonstrated in running system | Add end-to-end kill switch test |
| OD-PE6 | Single symbol coverage | Low | Only btcusdt@60s exercised | Add at least 2 symbols to scenario matrix |
| OD-PE7 | Static signal values | Low | Signals are hardcoded, not computed from candle data | Acceptable for paper proof; needed for live paper |
| OD-PE8 | No concurrent scenario testing | Low | All scenarios run sequentially; no actor contention proof | Add concurrent scenario if moving toward live paper |

### 3.2 Inherited Debts Still Open

| ID | Description | Severity | Origin | Still Relevant? |
|----|-------------|----------|--------|----------------|
| OD-CG1 | Column-opaque spec: no type validation | Medium | Codegen wave | Yes — blocks typed mapper generation |
| OD-CG2 | Store consumer specs ungoverned | Low | Codegen wave | Yes |
| OD-CG3 | Manual marker placement | Low | Codegen wave | Yes |
| OD-CG4 | No registry codegen | Low | Codegen wave | Yes |
| OD-CG5 | No config codegen | Low | Codegen wave | Yes |
| OD-CG6 | AckWait/MaxDeliver hardcoded | Medium | Codegen wave | Yes — blocked by OD-BW2 |
| OD-BW2 | Configurable scaling absent | Medium | Behavioral wave | Yes |
| OD-BW5 | Performance budgets undefined | Low | Behavioral wave | Yes |
| OD-BW6 | configctl tooling absent | Low | Behavioral wave | Yes |

### 3.3 Items Explicitly Deferred by Charter

- **Real venue integration** — not in scope, not attempted.
- **OMS (order management system)** — not in scope, not attempted.
- **Portfolio tracking and PnL** — not in scope, not attempted.
- **Multi-venue routing** — not in scope, not attempted.
- **Live candle-driven signals** — not in scope, not attempted.

These were charter-prohibited and remain deferred by design.

---

## 4. Net Assessment

The paper execution wave invested 5 stages and returned:

- **First closed operational loop** — the Foundry produces paper orders from market intelligence.
- **12 end-to-end scenarios** (7 in S266, 5 in S268) proving behavioral coherence.
- **3 boundary gaps fixed** with zero regressions across 47 behavioral tests.
- **Severity proven as behavioral driver** with 2.56× quantity ratio.
- **Dual risk and cross-chain distinction** working as designed.

The investment was proportional to the return. The Foundry moved from "infrastructure that processes domain events" to "system that generates operational outputs." This is a qualitative shift.

However, the wave left safety integration (OD-PE1, OD-PE5), observability (OD-PE3, OD-PE4), and governance (OD-PE2) as open debts. The most consequential is OD-PE1: SafetyGate not wired into the execution path. A paper execution system without proven safety controls is a proof of concept, not an operational system.

The honest assessment: the core domain loop works and is behaviorally rich. The operational envelope around it — safety, observability, persistence — needs one more pass before the Foundry can claim operational readiness even in paper mode.
