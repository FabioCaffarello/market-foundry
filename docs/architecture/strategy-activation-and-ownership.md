# Strategy Activation and Ownership — Market Foundry

> Canonical mapping of how `strategy` enters the runtime, who owns what, and how activation works.
> Stage: S53 — Design only. Implementation deferred to S54+.
> Date: 2026-03-17

---

## 1. Activation Model

Strategy uses the same two-layer activation model proven by evidence, signal, and decision.

### Layer 1 — Family Activation (structural)

```jsonc
// deploy/configs/derive.jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"],
    "strategy_families": ["mean_reversion_entry"]
  }
}
```

| Property | Behavior |
|---|---|
| Key | `pipeline.strategy_families` |
| Semantics | Explicit opt-in — empty list means no strategy activation |
| Change | Requires derive restart |
| Validation | Each entry must match a known strategy family name |
| Independence | No coupling to `decision_families` — activating a strategy family does NOT auto-activate its input decision |

**Important**: The operator must ensure that required decision families are also activated. If `mean_reversion_entry` is in `strategy_families` but `rsi_oversold` is not in `decision_families`, the resolver will never receive input and will emit `flat` with insufficient confidence. This is by design — no implicit activation chains.

### Layer 2 — Binding Activation (runtime)

| Trigger | Effect |
|---|---|
| BindingWatcherActor detects new binding | DeriveSupervisor spawns SourceScopeActor |
| SourceScopeActor starts | If strategy family is enabled, spawns StrategyResolverActor(s) |
| BindingWatcherActor detects binding removal | SourceScopeActor stops, including strategy actors |

No changes to BindingWatcherActor are needed — it already manages the scope lifecycle. Strategy resolvers are spawned as children of SourceScopeActor, like decision evaluators.

---

## 2. Activation Preconditions

For a strategy family to activate correctly at runtime, all of the following must hold:

| ID | Precondition | Owner |
|---|---|---|
| **AP-1** | `strategy_families` config key recognized by settings schema | settings package |
| **AP-2** | `IsStrategyFamilyEnabled()` method exists on PipelineConfig | settings package |
| **AP-3** | DeriveSupervisor spawns StrategyResolverActor when family is enabled | derive supervisor |
| **AP-4** | StrategyPublisherActor exists and publishes to STRATEGY_EVENTS | derive scope |
| **AP-5** | STRATEGY_EVENTS stream created on startup | derive or store init |
| **AP-6** | Store consumer + projection for the family exist | store supervisor |
| **AP-7** | KV bucket created on startup | store init |
| **AP-8** | QueryResponderActor handles `strategy.query.*` subjects | store |
| **AP-9** | Gateway routes registered under `/strategy/` | gateway |
| **AP-10** | Raccoon-CLI drift rules updated for strategy domain | governance |

---

## 3. Ownership Matrix

### Event Stream

| Stream | Writer | Consumers |
|---|---|---|
| `STRATEGY_EVENTS` | StrategyPublisherActor (derive) | StrategyConsumerActor (store) |

### Projection Ownership

| KV Bucket | Writer | Reader |
|---|---|---|
| `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` | StrategyProjectionActor (store) | QueryResponderActor (store) |

### Query Surface

| NATS Subject | Server |
|---|---|
| `strategy.query.mean_reversion_entry.latest` | QueryResponderActor (store) |

### HTTP Surface

| Endpoint | Server |
|---|---|
| `GET /strategy/mean_reversion_entry/latest` | gateway |

---

## 4. Actor Ownership Tree

### Derive (after strategy activation)

```
DeriveSupervisor
├── BindingWatcherActor
└── SourceScopeActor (per binding: source+symbol+timeframe)
    ├── ConsumerActor               ← reads OBSERVATION_EVENTS
    ├── SamplerActor (candle)       ← evidence
    ├── SamplerActor (tradeburst)   ← evidence
    ├── SamplerActor (volume)       ← evidence
    ├── SignalSamplerActor (rsi)    ← signal
    ├── DecisionEvaluatorActor (rsi_oversold)  ← decision
    ├── StrategyResolverActor (mean_reversion_entry)  ← NEW
    ├── EvidencePublisherActor      ← writes EVIDENCE_EVENTS
    ├── SignalPublisherActor         ← writes SIGNAL_EVENTS
    ├── DecisionPublisherActor       ← writes DECISION_EVENTS
    └── StrategyPublisherActor       ← writes STRATEGY_EVENTS (NEW)
```

### Store (after strategy projection activation)

```
StoreSupervisor
├── CandleProjectionActor
├── CandleConsumerActor
├── TradeBurstProjectionActor
├── TradeBurstConsumerActor
├── VolumeProjectionActor
├── VolumeConsumerActor
├── SignalConsumerActor
├── SignalProjectionActor
├── DecisionConsumerActor (rsi_oversold)
├── DecisionProjectionActor (rsi_oversold)
├── StrategyConsumerActor (mean_reversion_entry)    ← NEW
├── StrategyProjectionActor (mean_reversion_entry)  ← NEW
└── QueryResponderActor (extended with strategy routes)
```

---

## 5. Data Flow

```
                  derive binary                              store binary        gateway
┌──────────────────────────────────────────┐     ┌──────────────────────────┐   ┌──────┐
│ ConsumerActor                            │     │                          │   │      │
│   ↓ observation                          │     │                          │   │      │
│ SamplerActor(s)                          │     │                          │   │      │
│   ↓ evidence (actor msg)                 │     │                          │   │      │
│ SignalSamplerActor(rsi)                  │     │                          │   │      │
│   ↓ signal (actor msg)                   │     │                          │   │      │
│ DecisionEvaluatorActor(rsi_oversold)     │     │                          │   │      │
│   ↓ decision (actor msg)                 │     │                          │   │      │
│ StrategyResolverActor(mean_reversion)    │     │                          │   │      │
│   ↓ strategy (actor msg)                 │     │                          │   │      │
│ StrategyPublisherActor                   │     │                          │   │      │
│   → STRATEGY_EVENTS (JetStream) ─────────┼────→│ StrategyConsumerActor    │   │      │
│                                          │     │   ↓ actor msg            │   │      │
│                                          │     │ StrategyProjectionActor  │   │      │
│                                          │     │   → KV Bucket            │   │      │
│                                          │     │                          │   │      │
│                                          │     │ QueryResponderActor      │   │      │
│                                          │     │   ← KV read ← NATS req ←┼───┤ HTTP │
└──────────────────────────────────────────┘     └──────────────────────────┘   └──────┘
```

---

## 6. Decision Consumption Pattern

Strategy resolvers consume decisions via **local actor messages** within the same SourceScopeActor, not via JetStream. This matches the established pattern:

```
evidence sampler → actor msg → signal sampler → actor msg → decision evaluator → actor msg → strategy resolver
```

The `StrategyResolverActor` receives a message containing:
- Decision type (string)
- Decision outcome (string)
- Decision confidence (string)
- Decision metadata (map[string]string)
- Decision final flag (bool)
- Decision timestamp (time.Time)

It does NOT receive a `decision.Decision` struct — the message is a derive-internal message type that carries the necessary data without creating a domain import.

---

## 7. Config Schema Extension

```go
// internal/shared/settings/schema.go — PipelineConfig extension
type PipelineConfig struct {
    Timeframes        []int    `json:"timeframes"`
    Families          []string `json:"families"`
    SignalFamilies    []string `json:"signal_families"`
    DecisionFamilies  []string `json:"decision_families"`
    StrategyFamilies  []string `json:"strategy_families"`  // NEW
}

func (p PipelineConfig) IsStrategyFamilyEnabled(family string) bool {
    for _, f := range p.StrategyFamilies {
        if f == family {
            return true
        }
    }
    return false
}
```

Semantics identical to `IsDecisionFamilyEnabled`: empty list = nothing enabled.

---

## 8. Store Config Extension

```jsonc
// deploy/configs/store.jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"],
    "strategy_families": ["mean_reversion_entry"]
  }
}
```

Store uses `strategy_families` to determine which projection pipelines to instantiate. Each enabled strategy family creates one consumer + one projection actor pair.

---

## 9. Full Dependency Chain Visualization

```
pipeline.families         → evidence samplers (candle, tradeburst, volume)
pipeline.signal_families  → signal samplers (rsi)      ← depends on evidence
pipeline.decision_families → decision evaluators        ← depends on signals
pipeline.strategy_families → strategy resolvers         ← depends on decisions
```

Each layer is independently configurable. The operator is responsible for ensuring the transitive dependency chain is satisfied. The system does not auto-activate upstream families — this is explicit by design.

Validation rule for raccoon-cli (P-7):
- If `strategy_families` contains `mean_reversion_entry`, warn if `decision_families` does not contain `rsi_oversold`
- If `decision_families` contains `rsi_oversold`, warn if `signal_families` does not contain `rsi`
- If `signal_families` contains `rsi`, warn if `families` does not contain `candle`

These are warnings, not errors — the operator may have valid reasons for partial activation during development or debugging.

---

## 10. Ownership Rules

| Rule | Description |
|---|---|
| **OR-1** | Only derive writes to STRATEGY_EVENTS — single-writer invariant |
| **OR-2** | Only store reads from STRATEGY_EVENTS — single-consumer per family |
| **OR-3** | Only store writes to strategy KV buckets — single-writer per bucket |
| **OR-4** | Only store reads strategy KV for query — no other binary reads KV |
| **OR-5** | Gateway translates HTTP↔NATS — no KV access, no caching |
| **OR-6** | StrategyResolverActor is owned by SourceScopeActor — lifecycle bound to binding |
| **OR-7** | StrategyPublisherActor is owned by SourceScopeActor — one per scope |

---

## References

- [strategy-domain-design.md](strategy-domain-design.md) — Domain design
- [strategy-stream-families.md](strategy-stream-families.md) — Family catalog
- [decision-activation-and-ownership.md](decision-activation-and-ownership.md) — Decision precedent
- [config-driven-activation-hardening.md](config-driven-activation-hardening.md) — Activation model
- [actor-ownership.md](actor-ownership.md) — Ownership rules
