# Strategy/Signal Integration Wave — Charter and Scope Freeze

> Opens the Strategy/Signal Integration Wave with frozen scope, governing
> questions, ordered block plan, and binding constraints.
>
> Predecessor: S357 (Operational Foundation Evidence Gate — SUBSTANTIAL delivery).
> Wave type: Implementation (bounded).
> Date: 2026-03-22.

---

## 1. Strategic Context

The Operational Foundation Wave (S353–S357) closed the minimum infrastructure
gaps blocking unattended operation: Prometheus metrics, CI smoke integration,
consumer lag visibility, latency histograms, and startup credential validation.
Verdict: SUBSTANTIAL (23/25 exit criteria passed, zero regressions, 6/15
residual gaps closed).

The system now has:

| Proven capability | Evidence |
|---|---|
| Venue activation model (paper → venue_live) | S337–S346 |
| Safety gates (kill switch, staleness, submit-time verification) | S328 decorator pipeline |
| Counter invariants under sustained load | S349 endurance assessment |
| Prometheus metrics for HTTP, consumer lag, latency | S354 |
| CI smoke pipeline (7 jobs) | S355 |
| Startup fail-fast validation | S356 |

What has **not** been proven:

| Gap | Impact |
|---|---|
| No signal-to-execution path exercised end-to-end | Domain layers are islands |
| No strategy consumer wired to execution | Strategy resolvers produce events nobody consumes for execution |
| No formal source selection for integration proof | Cannot demonstrate controlled integration without a fixed source |
| No explainability of strategy-driven execution | Cannot trace why a specific order was placed |
| No runtime controls for strategy-driven execution | Cannot halt or throttle strategy-specific execution |

The signal domain (6 samplers), decision domain (evaluators), and strategy
domain (3 resolvers) are **fully implemented** with NATS contracts, actors, and
HTTP query endpoints. The bridge exists (`PaperOrderEvaluator`) but has never
been exercised as a connected pipeline. This wave closes that gap.

---

## 2. Wave Identity

| Field | Value |
|---|---|
| **Wave name** | Strategy/Signal Integration Wave |
| **Wave type** | Implementation (bounded) |
| **Phase** | 36 |
| **Predecessor wave** | Operational Foundation (S353–S357) |
| **Predecessor verdict** | SUBSTANTIAL (23/25 exit criteria, zero regressions) |
| **Opening stage** | S358 (this document) |
| **Estimated stages** | S358 (charter) → S359 → S360 → S361 → S362 → S363 (gate) |
| **Strategic goal** | Prove one complete signal → strategy → execution path under controlled conditions |

---

## 3. Scope Blocks

### SSI-1 — Source Selection and Canonical Contract

**Objective**: Select one signal family and one strategy family as the
integration target. Formalize the data contract from strategy output to
execution input.

**Scope**:
- Select a single signal family (candidate: MACD or RSI) and a single strategy
  family (candidate: mean_reversion_entry) based on implementation maturity
- Document the canonical data contract: strategy event fields → ExecutionIntent
  fields mapping
- Define which strategy fields are required vs optional for execution
- Define the envelope and subject patterns for the integration path
- Validate that selected families have complete sampler → evaluator → resolver
  chains in code

**Answers**: SSIQ-1, SSIQ-2.

**Exit criterion**: A single signal+strategy pair is selected with a formal
field-level contract document. The contract is validated against existing domain
types in code.

**Dependencies**: None. Can start immediately.

### SSI-2 — Controlled Source-to-Execution Wiring

**Objective**: Wire the selected strategy family to execution via a strategy
consumer actor that subscribes to `STRATEGY_EVENTS` and produces
`ExecutionIntent` through the existing `PaperOrderEvaluator`.

**Scope**:
- Implement `StrategyConsumerActor` in `internal/actors/scopes/execute/`
  that subscribes to the selected strategy type's NATS subject
- Route consumed strategies through `PaperOrderEvaluator` to produce
  `ExecutionIntent`
- Propagate strategy provenance fields (type, confidence, direction, source)
  into `ExecutionIntent.Parameters`
- Wire the actor into the execute binary's actor topology
- Validate with paper adapter (no venue submission required)

**Answers**: SSIQ-3, SSIQ-4.

**Exit criterion**: A strategy event published to `STRATEGY_EVENTS` results in
an `ExecutionIntent` produced through the paper order evaluator. Provenance
fields are present in the intent.

**Dependencies**: SSI-1 (contract must be defined first).

### SSI-3 — Explainability and Runtime Controls

**Objective**: Add correlation ID propagation and strategy-specific runtime
controls so that strategy-driven execution is traceable and controllable.

**Scope**:
- Propagate a correlation ID from signal → strategy → execution intent → order
- Add strategy-type-specific gate (can halt execution for a specific strategy
  type without killing all execution)
- Add strategy confidence threshold (configurable minimum confidence below
  which execution is skipped)
- Expose strategy provenance in execution health tracker counters
- Add Prometheus metrics for strategy-driven execution (accepted, rejected,
  below-threshold)

**Answers**: SSIQ-5, SSIQ-6.

**Exit criterion**: A strategy-driven execution carries a correlation ID from
signal origin. Strategy-specific gate and confidence threshold are functional.
Prometheus metrics distinguish strategy-driven from non-strategy execution.

**Dependencies**: SSI-2 (wiring must exist before controls are added).

### SSI-4 — End-to-End Domain-to-Venue Proof

**Objective**: Demonstrate the complete path from signal generation through
strategy resolution to paper execution in a single controlled test scenario.

**Scope**:
- Create an integration test that exercises: signal sampler → signal event →
  decision evaluator → decision event → strategy resolver → strategy event →
  strategy consumer → paper order evaluator → execution intent → paper adapter
- Validate that the execution intent carries the correct strategy provenance
- Validate that correlation ID is preserved across all domain boundaries
- Validate that health tracker counters increment correctly
- Validate that strategy-specific gate halts execution when activated
- Run against composed stack (NATS + paper adapter)

**Answers**: SSIQ-7, SSIQ-8.

**Exit criterion**: A single end-to-end test passes demonstrating signal →
strategy → execution with full provenance, correlation ID, and control surface.

**Dependencies**: SSI-2, SSI-3 (wiring and controls must exist).

### SSI-5 — Strategy/Signal Integration Evidence Gate

**Objective**: Compile evidence across SSI-1 through SSI-4, answer all
governing questions, issue wave verdict.

**Scope**:
- Evidence matrix for SSIQ-1 through SSIQ-8
- Regression audit (all Go modules)
- Non-goal compliance verification
- Wave verdict
- Residual gaps catalog
- Next wave recommendation

**Dependencies**: SSI-1, SSI-2, SSI-3, SSI-4 all complete.

---

## 4. Sequencing and Stage Plan

```
S358 — Charter and scope freeze (this document)
  │
  └── S359 — SSI-1: Source selection and canonical contract
        │
        └── S360 — SSI-2: Controlled source-to-execution wiring
              │
              └── S361 — SSI-3: Explainability and runtime controls
                    │
                    └── S362 — SSI-4: End-to-end domain-to-venue proof
                          │
                          └── S363 — SSI-5: Evidence gate
```

**Recommended execution order**:

| Stage | Block | Parallel? | Prerequisite |
|---|---|---|---|
| S358 | Charter | — | S357 |
| S359 | SSI-1: Source selection + contract | — | S358 |
| S360 | SSI-2: Source-to-execution wiring | — | S359 |
| S361 | SSI-3: Explainability + controls | — | S360 |
| S362 | SSI-4: E2E domain-to-venue proof | — | S361 |
| S363 | SSI-5: Evidence gate | — | S362 |

This wave is **strictly sequential**. Each block depends on the prior block's
output. No parallelization is safe because each block builds on the wiring
established by its predecessor.

---

## 5. Freeze Conditions

The following constraints are binding for the duration of this wave:

| # | Condition |
|---|-----------|
| FC-1 | No new blocks may be added after S358 |
| FC-2 | Block scope is frozen as defined in section 3 |
| FC-3 | Non-goals list is frozen as defined in the companion document |
| FC-4 | Only the selected signal+strategy pair may be integrated; no additional families |
| FC-5 | Integration must use paper adapter; no venue submission required |
| FC-6 | Any scope change requires a new charter stage |
| FC-7 | Existing domain types may be extended but not restructured |

---

## 6. Dependencies and Preconditions

| Precondition | Status | Notes |
|---|---|---|
| S357 closed with SUBSTANTIAL verdict | DONE | 23/25 exit criteria, zero regressions |
| Signal domain implemented (6 samplers) | DONE | MACD, RSI, EMA crossover, Bollinger, VWAP, ATR |
| Strategy domain implemented (3 resolvers) | DONE | Mean reversion, trend following, squeeze breakout |
| NATS signal/strategy contracts defined | DONE | SIGNAL_EVENTS, STRATEGY_EVENTS streams |
| PaperOrderEvaluator exists | DONE | Maps direction → side, carries provenance |
| Prometheus /metrics available | DONE | S354 |
| All Go modules passing vet + tests | DONE | Verified at S357 |

---

## 7. Success Criteria for Wave Closure

The wave may close when:

1. All 8 governing questions (SSIQ-1 through SSIQ-8) receive YES/NO/PARTIAL
   answers with evidence
2. Zero regressions across all Go modules
3. All 4 implementation blocks produce working, tested code
4. One complete signal → strategy → execution path is demonstrated
5. Non-goal compliance is verified
6. Evidence gate (SSI-5) issues a formal verdict

---

## References

- [S357 — Operational Foundation Evidence Gate](../stages/stage-s357-operational-foundation-evidence-gate-report.md)
- [Operational Foundation Wave Charter](operational-foundation-wave-charter-and-scope-freeze.md)
- [Signal Domain Design](signal-domain-design.md)
- [Strategy Domain Design](strategy-domain-design.md)
- [Execution Domain Design](execution-domain-design.md)
- [Capabilities, Questions, and Non-Goals](strategy-signal-integration-capabilities-questions-and-non-goals.md)
