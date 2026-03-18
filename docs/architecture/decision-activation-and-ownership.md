# Decision Activation and Ownership — Market Foundry

> Canonical mapping of how `decision` enters the runtime, who owns what, and how activation works.
> Stage: S42 — Design only. Implementation deferred to S43+.
> Approved: 2026-03-17

---

## 1. Activation Model

Decision uses the same two-layer activation model proven by evidence and signal.

### Layer 1 — Family Activation (structural)

```jsonc
// deploy/configs/derive.jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"]
  }
}
```

| Property | Behavior |
|---|---|
| Key | `pipeline.decision_families` |
| Semantics | Explicit opt-in — empty list means no decision activation |
| Change | Requires derive restart |
| Validation | Each entry must match a known decision family name |
| Independence | No coupling to `signal_families` — activating a decision family does NOT auto-activate its input signal |

**Important**: The operator must ensure that required signal families are also activated.
If `rsi_oversold` is in `decision_families` but `rsi` is not in `signal_families`, the
evaluator will never receive input and will emit `insufficient` outcomes. This is by design —
no implicit activation chains.

### Layer 2 — Binding Activation (runtime)

| Trigger | Effect |
|---|---|
| BindingWatcherActor detects new binding | DeriveSupervisor spawns SourceScopeActor |
| SourceScopeActor starts | If decision family is enabled, spawns DecisionEvaluatorActor(s) |
| BindingWatcherActor detects binding removal | SourceScopeActor stops, including decision actors |

No changes to BindingWatcherActor are needed — it already manages the scope lifecycle.
Decision evaluators are spawned as children of SourceScopeActor, like signal samplers.

---

## 2. Activation Preconditions

For a decision family to activate correctly at runtime, all of the following must hold:

| ID | Precondition | Owner |
|---|---|---|
| **AP-1** | `decision_families` config key recognized by settings schema | settings package |
| **AP-2** | `IsDecisionFamilyEnabled()` method exists on PipelineConfig | settings package |
| **AP-3** | DeriveSupervisor spawns DecisionEvaluatorActor when family is enabled | derive supervisor |
| **AP-4** | DecisionPublisherActor exists and publishes to DECISION_EVENTS | derive scope |
| **AP-5** | DECISION_EVENTS stream created on startup | derive or store init |
| **AP-6** | Store consumer + projection for the family exist | store supervisor |
| **AP-7** | KV bucket created on startup | store init |
| **AP-8** | QueryResponderActor handles `decision.query.*` subjects | store |
| **AP-9** | Gateway routes registered under `/decision/` | gateway |
| **AP-10** | Raccoon-CLI drift rules updated for decision domain | governance |

---

## 3. Ownership Matrix

### Event Stream

| Stream | Writer | Consumers |
|---|---|---|
| `DECISION_EVENTS` | DecisionPublisherActor (derive) | DecisionConsumerActor (store) |

### Projection Ownership

| KV Bucket | Writer | Reader |
|---|---|---|
| `DECISION_RSI_OVERSOLD_LATEST` | DecisionProjectionActor (store) | QueryResponderActor (store) |

### Query Surface

| NATS Subject | Server |
|---|---|
| `decision.query.rsi_oversold.latest` | QueryResponderActor (store) |

### HTTP Surface

| Endpoint | Server |
|---|---|
| `GET /decision/rsi_oversold/latest` | gateway |

---

## 4. Actor Ownership Tree

### Derive (after decision activation)

```
DeriveSupervisor
├── BindingWatcherActor
└── SourceScopeActor (per binding: source+symbol+timeframe)
    ├── ConsumerActor               ← reads OBSERVATION_EVENTS
    ├── SamplerActor (candle)       ← evidence
    ├── SamplerActor (tradeburst)   ← evidence
    ├── SamplerActor (volume)       ← evidence
    ├── SignalSamplerActor (rsi)    ← signal
    ├── DecisionEvaluatorActor (rsi_oversold)  ← NEW
    ├── EvidencePublisherActor      ← writes EVIDENCE_EVENTS
    ├── SignalPublisherActor         ← writes SIGNAL_EVENTS
    └── DecisionPublisherActor       ← writes DECISION_EVENTS (NEW)
```

### Store (after decision projection activation)

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
├── DecisionConsumerActor (rsi_oversold)    ← NEW
├── DecisionProjectionActor (rsi_oversold)  ← NEW
└── QueryResponderActor (extended with decision routes)
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
│ DecisionPublisherActor                   │     │                          │   │      │
│   → DECISION_EVENTS (JetStream)  ────────┼────→│ DecisionConsumerActor    │   │      │
│                                          │     │   ↓ actor msg            │   │      │
│                                          │     │ DecisionProjectionActor  │   │      │
│                                          │     │   → KV Bucket            │   │      │
│                                          │     │                          │   │      │
│                                          │     │ QueryResponderActor      │   │      │
│                                          │     │   ← KV read ← NATS req ←┼───┤ HTTP │
└──────────────────────────────────────────┘     └──────────────────────────┘   └──────┘
```

---

## 6. Signal Consumption Pattern

Decision evaluators consume signals via **local actor messages** within the same SourceScopeActor,
not via JetStream. This matches the signal consumption pattern:

```
evidence sampler → actor msg → signal sampler → actor msg → decision evaluator
```

The `DecisionEvaluatorActor` receives a message containing:
- Signal type (string)
- Signal value (string)
- Signal metadata (map[string]string)
- Signal final flag (bool)
- Signal timestamp (time.Time)

It does NOT receive a `signal.Signal` struct — the message is a derive-internal message type
that carries the necessary data without creating a domain import.

---

## 7. Config Schema Extension

```go
// internal/shared/settings/schema.go — PipelineConfig extension
type PipelineConfig struct {
    Timeframes       []int    `json:"timeframes"`
    Families         []string `json:"families"`
    SignalFamilies   []string `json:"signal_families"`
    DecisionFamilies []string `json:"decision_families"`  // NEW
}

func (p PipelineConfig) IsDecisionFamilyEnabled(family string) bool {
    for _, f := range p.DecisionFamilies {
        if f == family {
            return true
        }
    }
    return false
}
```

Semantics identical to `IsSignalFamilyEnabled`: empty list = nothing enabled.

---

## 8. Store Config Extension

```jsonc
// deploy/configs/store.jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"]
  }
}
```

Store uses `decision_families` to determine which projection pipelines to instantiate.
Each enabled decision family creates one consumer + one projection actor pair.

---

## 9. Ownership Rules

| Rule | Description |
|---|---|
| **OR-1** | Only derive writes to DECISION_EVENTS — single-writer invariant |
| **OR-2** | Only store reads from DECISION_EVENTS — single-consumer per family |
| **OR-3** | Only store writes to decision KV buckets — single-writer per bucket |
| **OR-4** | Only store reads decision KV for query — no other binary reads KV |
| **OR-5** | Gateway translates HTTP↔NATS — no KV access, no caching |
| **OR-6** | DecisionEvaluatorActor is owned by SourceScopeActor — lifecycle bound to binding |
| **OR-7** | DecisionPublisherActor is owned by SourceScopeActor — one per scope |

---

## References

- [decision-domain-design.md](decision-domain-design.md) — Domain design
- [decision-stream-families.md](decision-stream-families.md) — Family catalog
- [signal-activation-and-ownership.md](signal-activation-and-ownership.md) — Signal precedent
- [config-driven-activation-hardening.md](config-driven-activation-hardening.md) — Activation model
- [actor-ownership.md](actor-ownership.md) — Ownership rules
