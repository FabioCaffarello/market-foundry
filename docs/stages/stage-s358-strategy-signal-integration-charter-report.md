# Stage S358 — Strategy/Signal Integration Charter Report

> **Wave**: Strategy/Signal Integration (S358–S363).
> **Block**: Charter and scope freeze.
> **Date**: 2026-03-22.
> **Type**: Charter ceremony — assessment and scoping only, no implementation.

---

## 1. Executive Summary

Stage S358 formally opens the Strategy/Signal Integration Wave. This is the
first domain-advancement wave since the Operational Foundation Wave (S353–S357)
delivered its operational infrastructure (Prometheus metrics, CI smoke, startup
validation). The strategic directive is clear: reconnect the operational
robustness gained in recent waves with real business value by proving that the
analytical pipeline (signal → decision → strategy) can drive execution through
a controlled, traceable, observable path.

The wave is scoped to **one** signal+strategy pair, exercised through the
**paper adapter**, with full provenance, correlation, and runtime controls.
Scope is frozen at 5 blocks, 8 governing questions, 15 non-goals, and 7 freeze
conditions.

---

## 2. Consolidated State Post-S357

### What the system can do

The system has proven capabilities across three macro-fronts:

**Domain correctness** (S233–S346):
- 5 domain layers implemented: signal (6 samplers), decision (evaluators),
  strategy (3 resolvers), execution (paper + venue adapters), ingestion
- Actor topology with binding-level activation, family-driven config
- NATS JetStream contracts for all domain events
- HTTP query endpoints for signal, strategy, and execution state
- Venue adapter with 3-layer safety gates (kill switch, staleness, submit-time)
- Activation model with 4 dimensions (adapter, gate, credentials, effective)

**Production readiness** (S347–S352):
- Activation state queryability via HTTP
- Operational runbook validation
- Live testnet connectivity proof
- Endurance under sustained operation (counter invariants hold)

**Operational foundation** (S353–S357):
- Prometheus `/metrics` for HTTP, consumer lag, latency histograms
- CI pipeline: 7 jobs covering unit, integration, behavioral, smoke, codegen
- Startup fail-fast validation across all 7 binaries
- Zero regressions through 3 consecutive waves

### What the system cannot do

- Connect strategy output to execution input automatically
- Trace an execution decision back to its originating signal
- Halt execution for a specific strategy type without killing all execution
- Demonstrate an end-to-end analytical-to-execution flow
- Distinguish strategy-driven execution from manual execution in metrics

---

## 3. Wave Charter

### Mission

Prove that one complete signal → decision → strategy → execution path works
under controlled conditions with full traceability, observability, and runtime
controls.

### Blocks

| Block | Name | Objective |
|---|---|---|
| SSI-1 | Source Selection and Canonical Contract | Select one pair, formalize field-level contract |
| SSI-2 | Controlled Source-to-Execution Wiring | StrategyConsumerActor → PaperOrderEvaluator → ExecutionIntent |
| SSI-3 | Explainability and Runtime Controls | Correlation ID, per-strategy gate, confidence threshold, metrics |
| SSI-4 | End-to-End Domain-to-Venue Proof | Integration test: signal → strategy → paper execution |
| SSI-5 | Evidence Gate | Formal gate ceremony with verdict |

### Sequencing

All blocks are strictly sequential (SSI-1 → SSI-2 → SSI-3 → SSI-4 → SSI-5).
No parallelization — each block builds on the prior block's wiring.

| Stage | Block | Prerequisite |
|---|---|---|
| S358 | Charter (this document) | S357 |
| S359 | SSI-1: Source selection + contract | S358 |
| S360 | SSI-2: Source-to-execution wiring | S359 |
| S361 | SSI-3: Explainability + controls | S360 |
| S362 | SSI-4: E2E domain-to-venue proof | S361 |
| S363 | SSI-5: Evidence gate | S362 |

---

## 4. Governing Questions

| # | Question | Block |
|---|----------|-------|
| SSIQ-1 | Is there a documented, justified selection of one signal+strategy pair? | SSI-1 |
| SSIQ-2 | Is there a formal field-level contract mapping strategy → ExecutionIntent? | SSI-1 |
| SSIQ-3 | Does a StrategyConsumerActor produce ExecutionIntents from strategy events? | SSI-2 |
| SSIQ-4 | Does the ExecutionIntent carry strategy provenance in parameters? | SSI-2 |
| SSIQ-5 | Does a correlation ID propagate from signal through to execution? | SSI-3 |
| SSIQ-6 | Can strategy-driven execution be halted per-type with confidence gating? | SSI-3 |
| SSIQ-7 | Does an E2E test prove signal → strategy → paper execution? | SSI-4 |
| SSIQ-8 | Do Prometheus metrics reflect strategy-driven execution flow? | SSI-4 |

---

## 5. Non-Goals (Explicit)

| # | Non-Goal |
|---|----------|
| NG-1 | Multiple signal families |
| NG-2 | Multiple strategy families |
| NG-3 | Multi-venue execution |
| NG-4 | Risk domain implementation |
| NG-5 | OMS (Order Management System) |
| NG-6 | Portfolio risk management |
| NG-7 | Mainnet execution |
| NG-8 | Dashboard or UI |
| NG-9 | Push alerting |
| NG-10 | Log aggregation |
| NG-11 | New domain types |
| NG-12 | Strategy parameter optimization |
| NG-13 | Historical replay |
| NG-14 | Multi-timeframe integration |
| NG-15 | Deployment automation |

Full rationale for each non-goal is in the
[companion document](../architecture/strategy-signal-integration-capabilities-questions-and-non-goals.md).

---

## 6. Guard Rails

| # | Guard rail |
|---|-----------|
| GR-1 | Do not open multiple integration paths in parallel |
| GR-2 | Do not add new signal or strategy families as part of this wave |
| GR-3 | Do not introduce multi-venue routing |
| GR-4 | Do not implement OMS or portfolio management |
| GR-5 | Do not submit orders to mainnet |
| GR-6 | Do not restructure existing domain types; extend only if required |
| GR-7 | Do not build risk domain; strategy → execution is direct for this wave |
| GR-8 | Do not create new NATS streams; use existing contracts |
| GR-9 | Do not expand scope beyond the chartered 5 blocks |
| GR-10 | Do not build dashboards or visualization layers |

---

## 7. Preparation for S359

S359 (SSI-1: Source Selection and Canonical Contract) should:

1. **Evaluate signal families** — compare MACD, RSI, EMA crossover, Bollinger,
   VWAP, and ATR for integration suitability. Selection criteria: implementation
   completeness, sampler state complexity, output interpretability for strategy
   consumption.

2. **Evaluate strategy families** — compare mean_reversion_entry,
   trend_following_entry, and squeeze_breakout_entry. Selection criteria:
   resolver maturity, number of decision inputs consumed, output field richness.

3. **Document the contract** — formal field mapping from selected strategy's
   output struct to `ExecutionIntent` input fields. Include: required fields,
   optional fields, default values, type conversions, validation rules.

4. **Validate against code** — confirm the selected pair's sampler, evaluator,
   and resolver are fully implemented and tested. Confirm
   `PaperOrderEvaluator` can accept the strategy output without modification or
   with minimal extension.

**Expected deliverables for S359**:
- `docs/architecture/strategy-signal-source-selection-and-contract.md`
- Stage report: `docs/stages/stage-s359-source-selection-and-contract-report.md`

---

## 8. Deliverables

| # | Deliverable | Path |
|---|-------------|------|
| 1 | Wave Charter and Scope Freeze | [`docs/architecture/strategy-signal-integration-wave-charter-and-scope-freeze.md`](../architecture/strategy-signal-integration-wave-charter-and-scope-freeze.md) |
| 2 | Capabilities, Questions, and Non-Goals | [`docs/architecture/strategy-signal-integration-capabilities-questions-and-non-goals.md`](../architecture/strategy-signal-integration-capabilities-questions-and-non-goals.md) |
| 3 | Stage Report | This document |

---

## References

- [S357 — Operational Foundation Evidence Gate](stage-s357-operational-foundation-evidence-gate-report.md)
- [Operational Foundation Wave Charter](../architecture/operational-foundation-wave-charter-and-scope-freeze.md)
- [Wave Charter](../architecture/strategy-signal-integration-wave-charter-and-scope-freeze.md)
- [Capabilities, Questions, and Non-Goals](../architecture/strategy-signal-integration-capabilities-questions-and-non-goals.md)
- [Signal Domain Design](../architecture/signal-domain-design.md)
- [Strategy Domain Design](../architecture/strategy-domain-design.md)
- [Execution Domain Design](../architecture/execution-domain-design.md)
