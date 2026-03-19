# Execution — Activation and Ownership

> Activation model, actor ownership, and data flow for the `execution` domain in Market Foundry.
> Date: 2026-03-18 | Stage: S69

---

## 1. Activation Model

Execution follows the same two-layer activation model proven by signal, decision, strategy, and risk.

### Layer 1: Family Activation (Structural)

```jsonc
// derive.jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"],
    "strategy_families": ["mean_reversion_entry"],
    "risk_families": ["position_exposure"],
    "execution_families": ["paper_order"]           // NEW
  }
}
```

| Property | Value |
|----------|-------|
| Key | `pipeline.execution_families` |
| Semantics | List of enabled execution families; empty list = none enabled |
| Change | Requires binary restart |
| Validation | Rejects unknown families; enforces `executionDependsOnRisk` |
| Default | `[]` (disabled — execution is never auto-activated) |
| Independence | Each execution family is independent; enabling one does not enable others |

### Layer 2: Binding Activation (Runtime)

| Trigger | Effect |
|---------|--------|
| `IngestionRuntimeChanged` — new binding for source/symbol/timeframe | SourceScopeActor spawns `PaperOrderEvaluatorActor` for the new combination (if `paper_order` family is enabled) |
| `IngestionRuntimeChanged` — binding removed | Execution evaluator actors for that combination are stopped |

### Preconditions

Execution activation cannot proceed unless:
- Risk family `position_exposure` is enabled (config validation blocks startup otherwise)
- The full dependency chain is satisfied: evidence → signal → decision → strategy → risk → execution

---

## 2. Activation Preconditions

| ID | Precondition | Owner |
|----|-------------|-------|
| AP-1 | `pipeline.execution_families` contains `paper_order` | Settings/config |
| AP-2 | `pipeline.risk_families` contains `position_exposure` | Settings/config (transitive) |
| AP-3 | `pipeline.strategy_families` contains `mean_reversion_entry` | Settings/config (transitive) |
| AP-4 | `pipeline.decision_families` contains `rsi_oversold` | Settings/config (transitive) |
| AP-5 | `pipeline.signal_families` contains `rsi` | Settings/config (transitive) |
| AP-6 | `ValidatePipeline()` passes with no errors | Binary startup |
| AP-7 | `EXECUTION_EVENTS` stream created on NATS startup | ExecutionPublisher init |
| AP-8 | `EXECUTION_PAPER_ORDER_LATEST` KV bucket created | ExecutionKVStore init |
| AP-9 | Durable consumer `store-execution-paper-order` created | ExecutionConsumer init |
| AP-10 | Execution drift rules pass in raccoon-cli | Governance (S70+) |

---

## 3. Ownership Matrix

### Event Stream

| Resource | Writer | Readers |
|----------|--------|---------|
| `EXECUTION_EVENTS` stream | `ExecutionPublisherActor` (derive) | `ExecutionConsumerActor` (store) |

### Projection Ownership

| Resource | Writer | Readers |
|----------|--------|---------|
| `EXECUTION_PAPER_ORDER_LATEST` KV bucket | `ExecutionProjectionActor` (store) | `QueryResponderActor` (store) |

### Query Surface

| Resource | Server | Clients |
|----------|--------|---------|
| `execution.query.paper_order.latest` (NATS subject) | `QueryResponderActor` (store) | `ExecutionGateway` (gateway) |

### HTTP Surface

| Resource | Server | Clients |
|----------|--------|---------|
| `GET /execution/:type/latest` | Gateway HTTP handler | External consumers |

---

## 4. Actor Ownership Tree

### Derive Binary

```
derive-supervisor
├── observation-consumer
├── binding-watcher
└── source-{source} (SourceScopeActor)
    ├── publisher (EvidencePublisherActor)
    ├── signal-publisher (SignalPublisherActor)
    ├── decision-publisher (DecisionPublisherActor)
    ├── strategy-publisher (StrategyPublisherActor)
    ├── risk-publisher (RiskPublisherActor)
    ├── execution-publisher (ExecutionPublisherActor)                 ← NEW
    ├── sampler-BTCUSDT-60s (CandleSamplerActor)
    ├── burst-sampler-BTCUSDT-60s (TradeBurstSamplerActor)
    ├── volume-sampler-BTCUSDT-60s (VolumeSamplerActor)
    ├── signal-rsi-BTCUSDT-60s (RSISignalSamplerActor)
    ├── decision-rsi-oversold-BTCUSDT-60s (RSIOversoldEvaluatorActor)
    ├── strategy-mean-reversion-entry-BTCUSDT-60s (MeanReversionEntryResolverActor)
    ├── risk-position-exposure-BTCUSDT-60s (PositionExposureEvaluatorActor)
    └── execution-paper-order-BTCUSDT-60s (PaperOrderEvaluatorActor)  ← NEW
```

### Store Binary

```
store-supervisor
├── candle-projection
├── candle-consumer
├── trade-burst-projection
├── trade-burst-consumer
├── volume-projection
├── volume-consumer
├── signal-rsi-projection (if enabled)
├── signal-rsi-consumer (if enabled)
├── decision-rsi-oversold-projection (if enabled)
├── decision-rsi-oversold-consumer (if enabled)
├── strategy-mean-reversion-entry-projection (if enabled)
├── strategy-mean-reversion-entry-consumer (if enabled)
├── risk-position-exposure-projection (if enabled)
├── risk-position-exposure-consumer (if enabled)
├── execution-paper-order-projection (if enabled)                     ← NEW
├── execution-paper-order-consumer (if enabled)                       ← NEW
└── query-responder
```

### Gateway Binary

```
gateway
├── HTTP listener
├── configctl-gateway
├── evidence-gateway
├── signal-gateway
├── decision-gateway
├── strategy-gateway
├── risk-gateway
└── execution-gateway                                                  ← NEW
```

---

## 5. Data Flow

```
derive                              NATS                         store                    gateway
──────                              ────                         ─────                    ───────

PositionExposureEvaluatorActor
  │
  │ riskAssessedMessage
  │ (primitive types: symbol,
  │  riskType, disposition,
  │  confidence, maxPositionPct,
  │  maxExposurePct, direction,
  │  strategyConfidence,
  │  timeframe, timestamp,
  │  correlationID, causationID)
  ▼
SourceScopeActor
  │ routeRiskToExecution()
  ▼
PaperOrderEvaluatorActor                                         ExecutionConsumerActor
  │ PaperOrderEvaluator                                            │ durable consumer
  │ .Evaluate() (pure)                                             │ store-execution-paper-order
  │                                                                │
  │ publishExecutionMessage      ──► EXECUTION_EVENTS ──►          │
  ▼                                  (JetStream stream)            ▼
ExecutionPublisherActor                                          ExecutionProjectionActor
  │ publish with                                                   │ final gate
  │ dedup key                                                      │ validate gate
  │ + correlationID                                                │ monotonicity guard
  │ + causationID                                                  ▼
                                                                 EXECUTION_PAPER_ORDER_LATEST
                                                                 (KV bucket)
                                                                   │
                                                                 QueryResponderActor
                                                                   │ NATS req/reply         ◄── ExecutionGateway
                                                                   │                            │ NATS request
                                                                   ▼                            ▼
                                                                 Reply to                   HTTP handler
                                                                 gateway                    GET /execution/:type/latest
```

---

## 6. Risk Consumption Pattern

Execution receives risk data via `riskAssessedMessage` — a local actor message carrying primitive types. This is the same pattern used by:
- Decision receiving signal data via `signalGeneratedMessage`
- Strategy receiving decision data via `decisionEvaluatedMessage`
- Risk receiving strategy data via `strategyResolvedMessage`

### Message Structure

The `riskAssessedMessage` will be extended from the existing fan-out messages:

```go
type riskAssessedMessage struct {
    Symbol             string
    RiskType           string    // "position_exposure"
    RiskDisposition    string    // "approved", "modified", "rejected"
    RiskConfidence     string    // decimal string
    MaxPositionPct     string    // from risk constraints
    MaxExposurePct     string    // from risk constraints
    StrategyDirection  string    // "long", "short", "flat"
    StrategyConfidence string    // decimal string
    Timeframe          int
    Timestamp          time.Time
    CorrelationID      string
    CausationID        string    // RiskAssessedEvent.Metadata.ID
}
```

This message contains ONLY primitive types. No risk domain imports. The `CausationID` carries the `Metadata.ID` of the `RiskAssessedEvent` that produced it, maintaining the causal chain.

### Fan-Out Registration

In `SourceScopeActor`, execution evaluators are registered in a new routing method:

```go
func (s *SourceScopeActor) routeRiskToExecution(msg riskAssessedMessage) {
    for _, pid := range s.executionEvaluators[msg.Symbol] {
        s.ctx.Send(pid, msg)
    }
}
```

This mirrors `routeStrategyToRisk()`, `routeDecisionToStrategy()`, and `routeSignalToDecision()`.

---

## 7. Config Schema Extension

### PipelineConfig Addition

```go
type PipelineConfig struct {
    Timeframes       []int    `json:"timeframes"`
    Families         []string `json:"families"`
    SignalFamilies   []string `json:"signal_families"`
    DecisionFamilies []string `json:"decision_families"`
    StrategyFamilies []string `json:"strategy_families"`
    RiskFamilies     []string `json:"risk_families"`
    ExecutionFamilies []string `json:"execution_families"`    // NEW
}
```

### Helper Methods

```go
func (p PipelineConfig) IsExecutionFamilyEnabled(family string) bool {
    for _, f := range p.ExecutionFamilies {
        if f == family {
            return true
        }
    }
    return false
}

func (p PipelineConfig) EnabledExecutionFamilies() []string {
    if len(p.ExecutionFamilies) == 0 {
        return nil
    }
    result := make([]string, len(p.ExecutionFamilies))
    copy(result, p.ExecutionFamilies)
    return result
}
```

### Semantic Note

Empty list = nothing enabled. This is the same opt-in semantic used by signal, decision, strategy, and risk families. Evidence families (`Families`) use the opposite semantic (empty = all enabled) for backward compatibility.

---

## 8. Store Config Extension

```jsonc
// store.jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"],
    "strategy_families": ["mean_reversion_entry"],
    "risk_families": ["position_exposure"],
    "execution_families": ["paper_order"]           // NEW — must match derive config
  }
}
```

When `paper_order` is enabled in store, `StoreSupervisor` creates:
- `ExecutionConsumerActor` with durable consumer `store-execution-paper-order`
- `ExecutionProjectionActor` writing to `EXECUTION_PAPER_ORDER_LATEST` KV bucket
- `QueryResponderActor` registers handler for `execution.query.paper_order.latest`

---

## 9. Full Dependency Chain

```
observation → evidence (candle) → signal (rsi) → decision (rsi_oversold) → strategy (mean_reversion_entry) → risk (position_exposure) → execution (paper_order)
```

### Validation Rules

```go
var knownExecutionFamilies = map[string]bool{
    "paper_order": true,
}

var executionDependsOnRisk = map[string][]string{
    "paper_order": {"position_exposure"},
}
```

At startup, `ValidatePipeline()`:
1. Rejects unknown execution family names
2. Verifies that each enabled execution family's risk dependencies are also enabled
3. Transitively validates the full chain (execution → risk → strategy → decision → signal → evidence)

### No Auto-Activation

Execution families are NEVER activated by default. Enabling `risk_families: ["position_exposure"]` does NOT imply `execution_families: ["paper_order"]`. Each layer requires explicit opt-in.

---

## 10. Ownership Rules

| ID | Rule |
|----|------|
| OR-1 | `EXECUTION_EVENTS` has exactly one writer: `ExecutionPublisherActor` in the `derive` binary. |
| OR-2 | `EXECUTION_PAPER_ORDER_LATEST` has exactly one writer: `ExecutionProjectionActor` in the `store` binary. |
| OR-3 | `execution.query.paper_order.latest` has exactly one server: `QueryResponderActor` in the `store` binary. |
| OR-4 | `GET /execution/:type/latest` has exactly one server: HTTP handler in the `gateway` binary. |
| OR-5 | Gateway MUST NOT access execution KV buckets directly. All reads go through NATS request/reply to store. |
| OR-6 | Execution evaluator actors receive risk data via local messages ONLY. They MUST NOT subscribe to `RISK_EVENTS`. |
| OR-7 | Execution evaluator actor lifecycle is bound to `SourceScopeActor`. When the source scope stops, all execution actors stop. |

---

## 11. Health and Readiness

### Health Trackers

| Tracker Name | Binary | Events Tracked |
|-------------|--------|----------------|
| `execution-paper-order-publisher` | derive | Successful publishes to EXECUTION_EVENTS |
| `execution-paper-order-projection` | store | Successful materializations to KV |
| `execution-paper-order-consumer` | store | Successful event consumptions |

### Readiness

- Execution query availability is NOT required for gateway readiness (consistent with all optional domains)
- Execution publisher health is tracked but does NOT block derive binary readiness
- Execution projection health is tracked but does NOT block store binary readiness

---

## 12. Failure Isolation

| Failure Scenario | Impact | Isolation Mechanism |
|-----------------|--------|---------------------|
| PaperOrderEvaluatorActor crashes | Only that symbol/timeframe execution stops | SourceScopeActor supervisor; other evaluators unaffected |
| ExecutionPublisherActor crashes | All execution publishing for that source stops | SourceScopeActor supervisor; risk/strategy/other publishers unaffected |
| ExecutionProjectionActor crashes | Execution KV not updated | StoreSupervisor restarts projection; risk/strategy projections unaffected |
| ExecutionConsumerActor crashes | Events accumulate in EXECUTION_EVENTS stream | StoreSupervisor restarts consumer; events replayed from last ack |
| NATS connection lost | Publishing paused; store consumption paused | Reconnection logic in adapters; events buffered in JetStream |

---

## 13. References

- [execution-domain-design.md](execution-domain-design.md) — Domain model and boundary invariants
- [execution-stream-families.md](execution-stream-families.md) — Stream family catalog
- [execution-query-surface-guidelines.md](execution-query-surface-guidelines.md) — Query surface specification
- [risk-activation-and-ownership.md](risk-activation-and-ownership.md) — Upstream activation reference
- [actor-ownership.md](actor-ownership.md) — Global actor ownership
- [stream-family-catalog.md](stream-family-catalog.md) — Global stream family catalog
