# Strategy/Signal Integration — Capabilities, Questions, and Non-Goals

> Companion to the [Strategy/Signal Integration Wave Charter](strategy-signal-integration-wave-charter-and-scope-freeze.md).
> Defines the capabilities under assessment, governing questions, evaluation
> criteria, and explicit non-goals for the wave.
>
> Date: 2026-03-22.

---

## 1. Context

"Strategy/signal integration" means proving that the analytical pipeline
(signal → decision → strategy) can drive execution through a controlled,
traceable, controllable path. The system already has all domain layers
implemented independently. This wave proves they compose into a functioning
whole for **one** signal+strategy pair under **paper** execution.

---

## 2. Capabilities Under Assessment

| Capability | Current State | Target State |
|---|---|---|
| Source selection contract | Signal and strategy families exist independently; no formal integration contract | One pair selected; field-level contract documented and validated against types |
| Strategy-to-execution wiring | PaperOrderEvaluator accepts strategy fields manually; no actor consumption | StrategyConsumerActor subscribes to STRATEGY_EVENTS, routes through evaluator |
| Correlation ID propagation | No cross-domain correlation | Signal → strategy → execution carries a single correlation ID |
| Strategy-specific controls | Only global kill switch exists | Per-strategy-type gate + confidence threshold |
| Strategy execution observability | Health tracker counters exist per-domain but not cross-domain | Prometheus metrics for strategy-driven execution flow |
| End-to-end domain proof | Never exercised as connected pipeline | Integration test proving signal → strategy → paper execution |

---

## 3. Governing Questions

### SSI-1: Source Selection and Canonical Contract

| # | Question | Block | Evidence type |
|---|----------|-------|---------------|
| SSIQ-1 | Is there a documented, justified selection of one signal family and one strategy family for integration? | SSI-1 | Document + code validation |
| SSIQ-2 | Is there a formal field-level contract mapping strategy output fields to ExecutionIntent input fields? | SSI-1 | Contract document + type alignment check |

### SSI-2: Controlled Source-to-Execution Wiring

| # | Question | Block | Evidence type |
|---|----------|-------|---------------|
| SSIQ-3 | Does a StrategyConsumerActor exist that subscribes to the selected strategy type's NATS subject and produces ExecutionIntents? | SSI-2 | Code + unit tests |
| SSIQ-4 | Does the produced ExecutionIntent carry strategy provenance (type, confidence, direction, source) in its parameters? | SSI-2 | Unit test assertion |

### SSI-3: Explainability and Runtime Controls

| # | Question | Block | Evidence type |
|---|----------|-------|---------------|
| SSIQ-5 | Does a correlation ID propagate from signal origin through strategy to execution intent? | SSI-3 | Integration test assertion |
| SSIQ-6 | Can strategy-driven execution be halted per-strategy-type without killing global execution, and does a configurable confidence threshold gate execution? | SSI-3 | Unit test + integration test |

### SSI-4: End-to-End Domain-to-Venue Proof

| # | Question | Block | Evidence type |
|---|----------|-------|---------------|
| SSIQ-7 | Does a single end-to-end test demonstrate signal → decision → strategy → execution under composed stack? | SSI-4 | Integration test (NATS + paper adapter) |
| SSIQ-8 | Do Prometheus metrics correctly reflect strategy-driven execution flow (accepted, rejected, below-threshold, correlation)? | SSI-4 | Metrics scrape + test assertion |

---

## 4. Evaluation Criteria

Each capability and question is classified using the standard scale:

| Rating | Meaning |
|---|---|
| **FULL** | Capability demonstrated with complete evidence; all related questions answered YES |
| **SUBSTANTIAL** | Most evidence present; minor gaps documented but non-blocking |
| **PARTIAL** | Capability demonstrated in limited scope; material gaps remain |
| **PENDING** | Not demonstrated in this wave (out-of-scope per charter) |

---

## 5. Non-Goals

The following items are explicitly excluded from this wave. Each non-goal
includes a rationale.

| # | Non-Goal | Rationale |
|---|----------|-----------|
| NG-1 | **Multiple signal families** — integrating more than one signal family | Depth over breadth: prove the pattern with one family first |
| NG-2 | **Multiple strategy families** — integrating more than one strategy family | Same rationale as NG-1; additional families follow the proven pattern |
| NG-3 | **Multi-venue execution** — routing strategy-driven orders to multiple venues | Venue expansion is a separate wave; paper adapter is sufficient proof |
| NG-4 | **Risk domain implementation** — formal risk evaluation between strategy and execution | Risk is critical but separate; this wave proves the wiring, risk adds policy |
| NG-5 | **OMS (Order Management System)** — order lifecycle tracking, fills reconciliation, position management | OMS is a distinct domain; this wave proves strategy → intent, not intent → position |
| NG-6 | **Portfolio risk management** — portfolio-level exposure, correlation analysis, drawdown limits | Requires OMS and risk domain; out of scope |
| NG-7 | **Mainnet execution** — any real-money order submission | All proof uses paper adapter or testnet at most |
| NG-8 | **Dashboard or UI** — Grafana dashboards, web UI, or any visualization layer | Prometheus metrics are exposed; consumption is infrastructure, not domain |
| NG-9 | **Push alerting** — Alertmanager, PagerDuty, or any notification system | Deferred from S352; remains infrastructure concern |
| NG-10 | **Log aggregation** — centralized log collection (ELK, Loki, etc.) | Deferred from S352; remains infrastructure concern |
| NG-11 | **New domain types** — creating new domains (e.g., portfolio, risk, allocation) | This wave wires existing domains; no new domains |
| NG-12 | **Strategy parameter optimization** — backtesting, parameter sweeps, ML-based tuning | Optimization requires data infrastructure not yet built |
| NG-13 | **Historical replay** — replaying historical market data through the pipeline | Replay requires ingest backfill and time-simulation capabilities |
| NG-14 | **Multi-timeframe integration** — combining signals across timeframes in a single strategy | Adds combinatorial complexity; one timeframe per integration proof |
| NG-15 | **Deployment automation** — K8s, Helm, rolling updates, blue-green | Deferred from S352; infrastructure concern |

---

## 6. Relationship to Prior Residual Gaps

The S357 evidence gate cataloged 9 remaining residual gaps. This wave's
relationship to each:

| Gap | Severity | This wave |
|---|---|---|
| RG-2: No push alerting | HIGH | **Not addressed** (NG-9) |
| RG-4: Endurance limited to minutes | MEDIUM | **Not addressed** — separate wave |
| RG-8: No log aggregation | MEDIUM | **Not addressed** (NG-10) |
| RG-10: No port pre-check | LOW | **Not addressed** — deferred |
| RG-11: ClickHouse DSN docs | LOW | **Not addressed** — deferred |
| RG-12: Venue credentials not automated | LOW | **Not addressed** — deferred |
| RG-13: No resource profiling alerts | LOW | **Not addressed** — requires Alertmanager |
| RG-14: No credential rotation | LOW | **Not addressed** — by design |
| RG-15: No credential expiration | LOW | **Not addressed** — by design |

**None of these gaps block strategy/signal integration.** The operational
foundation (metrics, CI, preflight) is sufficient to support the domain work
in this wave.

---

## 7. Relationship to Existing Domain Implementation

This wave builds on proven implementations:

| Domain | Key artifact | Status |
|---|---|---|
| Signal | `internal/domain/signal/signal.go`, 6 samplers in `internal/application/signal/` | Complete |
| Decision | `internal/domain/decision/`, evaluators in `internal/application/decision/` | Complete |
| Strategy | `internal/domain/strategy/strategy.go`, 3 resolvers in `internal/application/strategy/` | Complete |
| Execution | `internal/domain/execution/execution.go`, `PaperOrderEvaluator` | Complete but not wired to strategy consumer |
| NATS contracts | `natssignal/`, `natsdecision/`, `natsstrategy/`, `natsexecution/` registries | Complete |
| Actors | Signal publisher, sampler, strategy resolver, publisher, venue adapter | Complete but disconnected |

The wave's implementation risk is **low** because all building blocks exist.
The work is primarily **wiring and proving composition**, not building new
domain capabilities.

---

## References

- [Strategy/Signal Integration Wave Charter](strategy-signal-integration-wave-charter-and-scope-freeze.md)
- [S357 — Operational Foundation Evidence Gate](../stages/stage-s357-operational-foundation-evidence-gate-report.md)
- [Signal Domain Design](signal-domain-design.md)
- [Strategy Domain Design](strategy-domain-design.md)
- [Execution Domain Design](execution-domain-design.md)
