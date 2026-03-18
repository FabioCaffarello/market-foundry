# Risk Activation and Ownership

> Stage S62 — Approved 2026-03-18
> Status: **DESIGN ONLY — no implementation in this stage**

---

## 1. Activation Model

Risk uses the same **two-layer activation model** proven by evidence, signal, decision, and strategy domains.

### Layer 1 — Family Activation (Structural)

- **Scope**: Which risk families are enabled.
- **Mechanism**: Configuration key in service config file.
- **Granularity**: Per-family opt-in list.
- **Lifecycle**: Requires service restart to change.
- **Default**: Empty list (risk disabled).

```jsonc
// deploy/configs/derive.jsonc
{
  "pipeline": {
    "evidence_families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"],
    "strategy_families": ["mean_reversion_entry"],
    "risk_families": ["position_exposure"]          // ← NEW
  }
}
```

**Behavior when empty**: No risk actors are spawned. No risk events are produced. The derive binary operates exactly as before. This is the default state.

### Layer 2 — Binding Activation (Runtime)

- **Scope**: Which source/symbol/timeframe combinations are active.
- **Mechanism**: BindingWatcherActor monitors configctl lifecycle events.
- **Granularity**: Per-binding (source × symbol × timeframe).
- **Lifecycle**: Dynamic — bindings activate/deactivate without restart.
- **Trigger**: configctl binding state changes (activated/deactivated).

**Activation flow**:
1. BindingWatcherActor detects new binding (e.g., `binancef.btcusdt.60` activated).
2. SourceScopeActor spawns child actors for the binding.
3. If `position_exposure` is in `risk_families`, RiskEvaluatorActor and RiskPublisherActor are spawned.
4. RiskEvaluatorActor receives strategy output via local actor message from StrategyResolverActor.
5. RiskPublisherActor writes risk assessments to `RISK_EVENTS`.

**Deactivation flow**:
1. BindingWatcherActor detects binding deactivation.
2. SourceScopeActor tears down all child actors for that binding.
3. RiskEvaluatorActor and RiskPublisherActor stop gracefully.

---

## 2. Activation Preconditions

| ID    | Precondition                                              | Type      | Enforcement               |
|-------|-----------------------------------------------------------|-----------|---------------------------|
| AP-1  | `strategy_families` contains required dependency          | Config    | Startup validation         |
| AP-2  | Full dependency chain valid (risk→strategy→decision→…)    | Config    | Startup validation         |
| AP-3  | `RISK_EVENTS` JetStream stream exists                     | Infra     | Startup check              |
| AP-4  | Risk family registered in `knownRiskFamilies`             | Code      | Compile-time               |
| AP-5  | Dependency DAG entry in `riskDependsOnStrategy`           | Code      | Compile-time               |
| AP-6  | raccoon-cli governance rules active                       | CI        | Gate (S63)                 |
| AP-7  | Adapter tests passing for risk publisher/consumer         | Test      | CI gate                    |
| AP-8  | Derive actor tests passing for RiskEvaluatorActor         | Test      | CI gate                    |
| AP-9  | Store bucket `RISK_{TYPE}_LATEST` exists                  | Infra     | Store startup              |
| AP-10 | Gateway route registered and tested                       | Code      | Route test                 |

---

## 3. Ownership Matrix

### Event Surface

| Resource                | Owner                     | Binary  | Invariant                    |
|-------------------------|---------------------------|---------|------------------------------|
| `RISK_EVENTS` stream    | RiskPublisherActor        | derive  | Single-writer per stream     |

### Projection Surface

| Resource                              | Owner                   | Binary  | Invariant                      |
|---------------------------------------|-------------------------|---------|--------------------------------|
| `RISK_POSITION_EXPOSURE_LATEST` KV    | RiskProjectionActor     | store   | Single-writer per bucket       |

### Query Surface

| Resource                                  | Owner                 | Binary  | Invariant                     |
|-------------------------------------------|-----------------------|---------|-------------------------------|
| `risk.query.position_exposure.latest`     | QueryResponderActor   | store   | Single-server per subject     |
| `GET /risk/position_exposure/latest`      | HTTP route handler    | gateway | Stateless translator          |

---

## 4. Actor Ownership Tree

### Derive

```
DeriveSupervisor
  └── SourceScopeActor [per source binding]
        ├── ConsumerActor
        ├── SamplerActor (candle)
        ├── PublisherActor (evidence)
        ├── TradeBurstSamplerActor
        ├── VolumeSamplerActor
        ├── SignalSamplerActor (RSI)
        ├── SignalPublisherActor
        ├── DecisionEvaluatorActor (RSI Oversold)
        ├── DecisionPublisherActor
        ├── StrategyResolverActor (Mean Reversion Entry)
        ├── StrategyPublisherActor
        ├── RiskEvaluatorActor (Position Exposure)       ← NEW
        └── RiskPublisherActor                           ← NEW
```

**Conditional spawning**: RiskEvaluatorActor and RiskPublisherActor are only spawned if:
1. `"position_exposure"` is in `risk_families` config.
2. The binding for this source is active.
3. The strategy dependency (`mean_reversion_entry`) is also active.

### Store

```
StoreSupervisor
  ├── EvidenceConsumerActor (candle)
  ├── CandleProjectionActor
  ├── TradeBurstConsumerActor
  ├── TradeBurstProjectionActor
  ├── VolumeConsumerActor
  ├── VolumeProjectionActor
  ├── SignalConsumerActor
  ├── SignalProjectionActor
  ├── DecisionConsumerActor
  ├── DecisionProjectionActor
  ├── StrategyConsumerActor
  ├── StrategyProjectionActor
  ├── RiskConsumerActor                                  ← NEW
  ├── RiskProjectionActor                                ← NEW
  └── QueryResponderActor (serves all query subjects)
```

### Gateway

```
Gateway
  └── HTTP Routes
        ├── /configctl/*
        ├── /evidence/*
        ├── /signal/*
        ├── /decision/*
        ├── /strategy/*
        └── /risk/*                                      ← NEW
```

---

## 5. Ownership Rules

| ID    | Rule                                                                                 |
|-------|--------------------------------------------------------------------------------------|
| OR-1  | Only RiskPublisherActor writes to `RISK_EVENTS` — no other actor, binary, or tool    |
| OR-2  | Only RiskProjectionActor writes to `RISK_{TYPE}_LATEST` — single-writer per bucket   |
| OR-3  | Only QueryResponderActor serves `risk.query.*` subjects — single-server invariant    |
| OR-4  | Gateway never accesses risk KV buckets — always routes through NATS request/reply    |
| OR-5  | Gateway never interprets risk dispositions — passes data transparently               |
| OR-6  | Risk evaluator receives strategy data via local actor messages, never JetStream      |
| OR-7  | Store risk consumers read only `RISK_EVENTS`, never `STRATEGY_EVENTS`                |

---

## 6. Configuration Schema Extension

### New Registry Entry

```go
// internal/shared/settings/schema.go (design)

var knownRiskFamilies = map[string]bool{
    "position_exposure": true,
}
```

### New Dependency DAG

```go
var riskDependsOnStrategy = map[string][]string{
    "position_exposure": {"mean_reversion_entry"},
}
```

### Pipeline Config Extension

```go
type PipelineConfig struct {
    EvidenceFamilies  []string `json:"evidence_families"`
    SignalFamilies    []string `json:"signal_families"`
    DecisionFamilies  []string `json:"decision_families"`
    StrategyFamilies  []string `json:"strategy_families"`
    RiskFamilies      []string `json:"risk_families"`      // ← NEW
}
```

### Validation Rules

1. Every entry in `risk_families` must exist in `knownRiskFamilies`.
2. For each risk family, all entries in `riskDependsOnStrategy[family]` must be present in `strategy_families`.
3. Transitive dependency validation: risk → strategy → decision → signal → evidence.
4. Empty `risk_families` is valid (risk disabled).

---

## 7. Store Configuration Extension

```jsonc
// deploy/configs/store.jsonc (design)
{
  "pipeline": {
    "evidence_families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"],
    "strategy_families": ["mean_reversion_entry"],
    "risk_families": ["position_exposure"]              // ← NEW
  }
}
```

Store uses `risk_families` to determine which consumer/projection actor pairs to spawn.

---

## 8. Health and Readiness

### Health Trackers (Store)

| Tracker Name                          | Owner                   |
|---------------------------------------|-------------------------|
| `risk-position_exposure-consumer`     | RiskConsumerActor       |
| `risk-position_exposure-projection`   | RiskProjectionActor     |

### Readiness (Gateway)

Risk query availability is **non-blocking** for gateway readiness (same as evidence, signal, decision, strategy). Gateway readiness is gated only on configctl availability.

---

## 9. Failure Isolation

- **Derive**: RiskEvaluatorActor failure within a SourceScopeActor does not affect other sources or other domain actors within the same scope. The scope actor supervises restart.
- **Store**: RiskProjectionActor failure does not affect other projection actors. Consumer pauses until projection recovers.
- **Gateway**: Risk endpoint unavailability returns 503; other endpoints unaffected.
