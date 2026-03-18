# Risk Domain Design

> Stage S62 — Approved 2026-03-18
> Status: **DESIGN ONLY — no implementation in this stage**

---

## 1. Executive Summary

Risk is the **sixth domain layer** in the Market Foundry pipeline:

```
observation → evidence → signal → decision → strategy → risk → [execution → portfolio]
```

A **risk assessment** is a typed, position-aware evaluation of a strategy intent that produces an approved, modified, or rejected disposition. Risk transforms raw directional intent (from strategy) into a **gated, bounded intent** by applying position-sizing rules, exposure limits, and drawdown constraints.

Risk is **not** execution. Risk answers: "Given this strategy intent, should we proceed, and with what constraints?" Execution answers: "How do we place and manage the order?"

---

## 2. What Risk Is

Risk is a **domain-specific evaluation layer** that:

1. Consumes strategy outputs (resolved intents with direction and confidence).
2. Applies **risk rules** — position sizing, exposure caps, max drawdown, correlation limits.
3. Produces a **risk assessment** with one of three dispositions:
   - `approved` — strategy intent accepted, possibly with adjusted sizing.
   - `modified` — strategy intent accepted with mandatory constraints (reduced size, tighter stops).
   - `rejected` — strategy intent blocked (exposure limit, drawdown breach, correlation constraint).
4. Publishes risk assessments as domain events to `RISK_EVENTS`.
5. Materializes latest assessments via store projections.
6. Exposes assessments via gateway query surface.

### Risk Produces

- A **RiskAssessment** entity with disposition, constraints, and rationale.
- Domain events on `RISK_EVENTS` stream.
- KV projections in store (`RISK_{TYPE}_LATEST`).
- Query responses via gateway HTTP endpoints.

### Risk Consumes

- Strategy resolved events (via local actor messages in derive, not JetStream round-trip).
- Configuration from configctl (family activation, binding lifecycle).

---

## 3. What Risk Is NOT

| Risk is NOT                          | Why                                                                 |
|--------------------------------------|---------------------------------------------------------------------|
| A second strategy layer              | Strategy resolves intent; risk evaluates whether intent is safe      |
| Execution                            | Risk gates intent; execution places orders                          |
| Portfolio management                 | Risk evaluates individual intents; portfolio aggregates positions    |
| A validator/quality service          | Risk is domain-specific, not cross-cutting validation               |
| A real-time position tracker         | Risk evaluates at decision time; position tracking is execution/portfolio |
| A P&L calculator                     | P&L is portfolio concern, not risk                                  |
| A market data filter                 | Observation/evidence handle data quality                            |

---

## 4. Domain Boundary Invariants

Risk domain isolation follows the same pattern as strategy, decision, and signal.

| ID     | Rule                                                                              |
|--------|-----------------------------------------------------------------------------------|
| RBI-1  | `internal/domain/risk` imports **zero** packages from strategy, decision, signal, evidence, observation |
| RBI-2  | `internal/domain/risk` imports **zero** packages from `internal/adapters`         |
| RBI-3  | `internal/domain/risk` imports **zero** packages from `internal/actors`           |
| RBI-4  | `internal/application/risk` imports only `internal/domain/risk` from domain layer |
| RBI-5  | `internal/adapters/nats` risk files import only `internal/domain/risk` and shared packages |
| RBI-6  | Strategy inputs in RiskAssessment are **domain-owned copies**, not strategy domain imports |
| RBI-7  | Risk actors in derive receive strategy data via **local actor messages**, not JetStream subscription |
| RBI-8  | Store risk projection actors consume `RISK_EVENTS` via JetStream, never strategy events directly |
| RBI-9  | Gateway risk handlers are stateless HTTP-to-NATS translators with zero domain logic |
| RBI-10 | raccoon-cli enforces RBI-1 through RBI-9 via static analysis rules                |

### How Risk Depends on Strategy Without Confusion

```
derive binary (single process):
  SourceScopeActor
    ├── StrategyResolverActor  → produces strategy → local message
    ├── StrategyPublisherActor → publishes to STRATEGY_EVENTS
    ├── RiskEvaluatorActor     → receives strategy via local message → produces risk assessment
    └── RiskPublisherActor     → publishes to RISK_EVENTS
```

- Risk **receives** strategy output as a local actor message (same as decision receives signal).
- Risk **does not import** strategy domain types. It defines its own `StrategyInput` struct.
- Risk **does not subscribe** to `STRATEGY_EVENTS`. The actor tree routes data locally.
- If strategy is disabled, risk actors are not spawned (dependency chain enforced at config level).

---

## 5. Domain Model

### RiskAssessment Entity

```go
// internal/domain/risk/risk.go

type RiskAssessment struct {
    Type        string            // risk family name (e.g., "position_exposure")
    Source      string            // exchange source
    Symbol      string            // trading pair
    Timeframe   string            // sampling window
    Disposition string            // approved | modified | rejected
    Confidence  string            // decimal string [0.0, 1.0]
    Strategies  []StrategyInput   // domain-owned copies of strategy inputs
    Constraints Constraints       // applied risk constraints
    Rationale   string            // human-readable explanation
    Parameters  map[string]string // evaluator-specific parameters
    Metadata    map[string]string // flexible metadata
    Final       bool              // completeness flag
    Timestamp   int64             // unix nanos
}

type StrategyInput struct {
    Type       string
    Direction  string
    Confidence string
}

type Constraints struct {
    MaxPositionSize string // decimal string, max allowed size
    MaxExposure     string // decimal string, max portfolio exposure
    StopDistance    string  // decimal string, mandatory stop distance (if modified)
}
```

### Validation Rules

- `Type` must be non-empty and registered.
- `Source`, `Symbol`, `Timeframe` must be non-empty.
- `Disposition` must be one of: `approved`, `modified`, `rejected`.
- `Confidence` must be valid decimal in `[0.0, 1.0]`.
- `Strategies` must contain at least one entry.
- `Final` must be `true` for materialization.
- `Timestamp` must be positive.
- If `Disposition == "modified"`, `Constraints` must have at least one non-empty field.
- If `Disposition == "rejected"`, `Rationale` must be non-empty.

### Methods

- `Validate() error` — enforces all validation rules.
- `PartitionKey() string` — returns `{source}.{symbol}.{timeframe}` (for KV key).
- `DeduplicationKey() string` — returns deterministic key for JetStream dedup.

---

## 6. Domain Events

```go
// internal/domain/risk/events.go

type RiskAssessedEvent struct {
    Metadata events.Metadata
    Risk     RiskAssessment
}

func (e RiskAssessedEvent) EventName() string {
    return "risk_assessed"
}
```

Single event type. The `Disposition` field within the assessment carries the evaluation outcome.

---

## 7. Application Layer

### Evaluator (Pure Function)

```go
// internal/application/risk/position_exposure_evaluator.go

func EvaluatePositionExposure(strategy StrategyInput, params PositionExposureParams) RiskAssessment
```

- Pure function, no I/O, no NATS, no actor dependencies.
- Receives strategy input and evaluator parameters.
- Returns a RiskAssessment with disposition and constraints.
- Testable in isolation.

### Ports

```go
// internal/application/ports/risk.go

type RiskGateway interface {
    GetLatestRisk(ctx context.Context, query GetLatestRiskQuery) (GetLatestRiskReply, *problem.Problem)
}
```

### Client Use Case

```go
// internal/application/riskclient/get_latest_risk.go

type GetLatestRiskQuery struct {
    Type      string
    Source    string
    Symbol    string
    Timeframe string
}

type GetLatestRiskReply struct {
    Risk *risk.RiskAssessment
}
```

---

## 8. Binary Placement

Risk evaluators live in the **derive** binary.

### Rationale

1. **Latency**: Strategy output feeds risk evaluation immediately via local actor messages. No JetStream round-trip.
2. **Consistency**: Signal, decision, and strategy evaluators all live in derive. Risk follows the same pattern.
3. **Activation**: Risk families activate alongside strategy families in the same scope actor tree.
4. **Simplicity**: No new binary, no new deployment, no new compose service.

### When to Extract

Risk moves to a separate binary **only if**:
- Risk evaluation requires external state (position database, portfolio aggregator).
- Risk processing latency materially impacts derive throughput.
- Risk requires independent scaling from derive.

None of these conditions exist in Phase 1. Extraction is a future concern, not a current design constraint.

---

## 9. Actor Ownership

### Derive Binary

```
DeriveSupervisor
  └── SourceScopeActor [per source]
        ├── ConsumerActor (reads OBSERVATION_EVENTS)
        ├── SamplerActor (candle/tradeburst/volume)
        ├── PublisherActor (writes EVIDENCE_EVENTS)
        ├── SignalSamplerActor (RSI)
        ├── SignalPublisherActor (writes SIGNAL_EVENTS)
        ├── DecisionEvaluatorActor (RSI Oversold)
        ├── DecisionPublisherActor (writes DECISION_EVENTS)
        ├── StrategyResolverActor (Mean Reversion Entry)
        ├── StrategyPublisherActor (writes STRATEGY_EVENTS)
        ├── RiskEvaluatorActor (Position Exposure)       ← NEW
        └── RiskPublisherActor (writes RISK_EVENTS)      ← NEW
```

### Store Binary

```
StoreSupervisor
  ├── ... (existing consumer/projection actors)
  ├── RiskConsumerActor       ← NEW (reads RISK_EVENTS)
  └── RiskProjectionActor     ← NEW (writes RISK_POSITION_EXPOSURE_LATEST)
```

### Gateway Binary

```
Gateway
  └── HTTP Routes
        ├── ... (existing routes)
        └── /risk/{type}/latest  ← NEW
```

### Ownership Matrix

| Resource                                | Owner                    | Binary  |
|-----------------------------------------|--------------------------|---------|
| RISK_EVENTS stream                      | RiskPublisherActor       | derive  |
| RISK_POSITION_EXPOSURE_LATEST bucket    | RiskProjectionActor      | store   |
| risk.query.position_exposure.latest     | QueryResponderActor      | store   |
| GET /risk/{type}/latest                 | HTTP handler             | gateway |

---

## 10. Activation Model

### Two-Layer Activation (consistent with all prior domains)

**Layer 1 — Family Activation (structural, requires restart)**

```jsonc
// deploy/configs/derive.jsonc
{
  "pipeline": {
    "risk_families": ["position_exposure"]  // empty = risk disabled
  }
}
```

**Layer 2 — Binding Activation (runtime, dynamic)**

BindingWatcherActor spawns SourceScopeActor, which spawns RiskEvaluatorActor and RiskPublisherActor if the `position_exposure` family is enabled in config.

### Activation Preconditions

| ID    | Precondition                                          | Enforcement          |
|-------|-------------------------------------------------------|----------------------|
| AP-1  | Strategy family dependency satisfied                  | Config validation    |
| AP-2  | RISK_EVENTS stream exists                             | Startup check        |
| AP-3  | RISK_{TYPE}_LATEST bucket exists                      | Store startup        |
| AP-4  | Risk family registered in schema.go                   | Compile-time         |
| AP-5  | Dependency DAG entry exists (risk → strategy)         | Config validation    |
| AP-6  | raccoon-cli governance rules active                   | CI gate              |

### Configuration Dependency Chain

```
risk_families: ["position_exposure"]
  └── requires strategy_families: ["mean_reversion_entry"]
        └── requires decision_families: ["rsi_oversold"]
              └── requires signal_families: ["rsi"]
                    └── requires evidence_families: ["candle"]
```

This chain is validated at startup. If any dependency is missing, the service refuses to start with a clear error message.

---

## 11. Projection Model

### Phase 1: Latest Only

- **Bucket**: `RISK_POSITION_EXPOSURE_LATEST`
- **Key format**: `{source}.{symbol}.{timeframe}`
- **Writer**: RiskProjectionActor (store) — single writer invariant
- **Reader**: QueryResponderActor (store)

### Three Materialization Gates

1. **Final gate**: Skip if `assessment.Final != true`.
2. **Validate gate**: Skip if `assessment.Validate() != nil`.
3. **Monotonicity guard**: Skip if `assessment.Timestamp <= existing.Timestamp`.

### Observability Counters

Seven counters (consistent with all prior projection actors):
- `risk_projection_received`
- `risk_projection_materialized`
- `risk_projection_skipped_stale`
- `risk_projection_skipped_dedup`
- `risk_projection_skipped_non_final`
- `risk_projection_rejected`
- `risk_projection_errors`

### Health Trackers

- `risk-position_exposure-projection`
- `risk-position_exposure-consumer`

### History Projections

**Explicitly deferred.** History for risk assessments is not required in Phase 1. If needed later:
- Bucket: `RISK_POSITION_EXPOSURE_HISTORY`
- Key format: `{source}.{symbol}.{timeframe}.{timestamp}`
- Query: `GET /risk/position_exposure/history?source=X&symbol=Y&timeframe=Z&limit=N`

---

## 12. Query Surface

### Phase 1 Endpoint

```
GET /risk/{type}/latest?source=X&symbol=Y&timeframe=Z
```

### Four-Layer Query Chain

```
HTTP (gateway) → Use Case (riskclient) → NATS Request/Reply (store) → KV (RISK_{TYPE}_LATEST)
```

### NATS Subjects

| Subject                                 | Purpose        |
|-----------------------------------------|----------------|
| `risk.query.position_exposure.latest`   | Latest query   |
| `risk.query.position_exposure.history`  | Deferred       |

### Envelope Types

| Type                                                | Direction |
|-----------------------------------------------------|-----------|
| `risk.query.v1.position_exposure_latest_request`    | Request   |
| `risk.query.v1.position_exposure_latest_reply`      | Reply     |

### Response (200)

```json
{
  "risk": {
    "type": "position_exposure",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": "60",
    "disposition": "approved",
    "confidence": "0.85",
    "strategies": [
      {
        "type": "mean_reversion_entry",
        "direction": "long",
        "confidence": "0.72"
      }
    ],
    "constraints": {
      "max_position_size": "0.01",
      "max_exposure": "0.05",
      "stop_distance": ""
    },
    "rationale": "Position size within exposure limits",
    "parameters": {
      "max_position_pct": "0.02",
      "max_portfolio_exposure_pct": "0.10"
    },
    "metadata": {},
    "final": true,
    "timestamp": 1710700000000000000
  }
}
```

### Response (404)

No materialized risk assessment for the given key.

### Response (400)

Missing or invalid query parameters.

---

## 13. Stream Definition

| Property           | Value                                                                      |
|--------------------|----------------------------------------------------------------------------|
| Stream name        | `RISK_EVENTS`                                                              |
| Subject pattern    | `risk.events.{type}.assessed.{source}.{symbol}.{timeframe}`               |
| Retention          | Limits (72h)                                                               |
| Storage            | File                                                                       |
| Max age            | 72h                                                                        |
| Discard            | Old                                                                        |
| Deduplication      | Enabled (message ID = deduplication key)                                   |
| Writer             | RiskPublisherActor (derive) — single writer                                |
| Consumers          | RiskConsumerActor (store) — durable, deliver-all                           |

### Subject Encoding

```
risk.events.position_exposure.assessed.binancef.btcusdt.60
```

Components:
- `risk.events` — family prefix
- `position_exposure` — risk type (family name)
- `assessed` — verb (action completed)
- `binancef` — source exchange
- `btcusdt` — trading symbol
- `60` — timeframe in seconds

---

## 14. Differentiation Matrix

| Concern                     | Strategy                          | Risk                              | Execution (future)             | Portfolio (future)              |
|-----------------------------|-----------------------------------|-----------------------------------|--------------------------------|--------------------------------|
| **Input**                   | Decisions                         | Strategies                        | Risk assessments               | Execution results              |
| **Output**                  | Directional intent                | Gated/bounded intent              | Order placement                | Position aggregation           |
| **Question answered**       | "What should we do?"              | "Should we proceed, and how much?"| "How do we place the order?"   | "What is our overall position?"|
| **Disposition**             | long / short / flat               | approved / modified / rejected    | filled / partial / failed      | net_long / net_short / flat    |
| **Statefulness**            | Stateless (per evaluation)        | Stateless (per evaluation)        | Stateful (order lifecycle)     | Stateful (position tracking)   |
| **Binary**                  | derive                            | derive                            | TBD (likely separate)          | TBD (likely separate)          |
| **External dependencies**   | None                              | None (Phase 1)                    | Exchange APIs                  | Database                       |

---

## 15. Invariants Preventing Improper Coupling

| ID    | Invariant                                                                           |
|-------|-------------------------------------------------------------------------------------|
| IC-1  | Risk domain types never appear in strategy, decision, signal, or evidence packages  |
| IC-2  | Strategy domain types never appear in risk domain package (use StrategyInput copy)   |
| IC-3  | Risk evaluator is a pure function — no I/O, no NATS, no actor references            |
| IC-4  | Risk never reads from STRATEGY_EVENTS directly — receives via local actor messages   |
| IC-5  | Store risk projection never reads from STRATEGY_EVENTS — only RISK_EVENTS            |
| IC-6  | Gateway risk handler never accesses KV — only NATS request/reply                     |
| IC-7  | Risk configuration validates strategy dependency at startup — not at runtime          |
| IC-8  | No risk type embeds or wraps a strategy type — only references via StrategyInput      |
| IC-9  | Risk does not produce execution orders — only gated intents                           |
| IC-10 | Risk does not aggregate across symbols — evaluates per partition key                  |

---

## 16. What Is Explicitly Deferred

| Item                                    | Deferred To | Rationale                                           |
|-----------------------------------------|-------------|-----------------------------------------------------|
| Risk history projections                | S65+        | Latest-only sufficient for Phase 1                  |
| Multi-strategy risk evaluation          | S65+        | First family evaluates single strategy input         |
| Portfolio-level exposure aggregation    | Portfolio   | Cross-symbol aggregation is portfolio concern        |
| Real-time position tracking             | Execution   | Risk evaluates at assessment time, not continuously  |
| External risk data feeds                | S66+        | No external dependencies in Phase 1                  |
| ClickHouse risk analytics               | S67+        | Analytical storage follows operational stability     |
| Risk family: correlation_limit          | S65+        | Requires multi-symbol portfolio state                |
| Risk family: drawdown_guard             | S65+        | Requires historical P&L (execution/portfolio)        |
| Separate risk binary extraction         | If needed   | Only if derive throughput becomes bottleneck          |
| raccoon-cli risk governance rules       | S63         | Governance activation is next stage                  |

---

## 17. Preparation for S63 and S64

### S63 — Risk Governance Activation

Prerequisites from this design:
- Domain boundary invariants (RBI-1 through RBI-10) ready for raccoon-cli enforcement.
- Stream family definition ready for catalog registration.
- Ownership matrix ready for static analysis rules.
- Import prohibition rules defined (risk ↛ strategy domain, risk ↛ execution).

### S64 — Risk First Slice

Prerequisites from this design:
- Domain model (`RiskAssessment`, `StrategyInput`, `Constraints`) fully specified.
- Event type (`RiskAssessedEvent`) defined.
- Actor tree placement designed (RiskEvaluatorActor, RiskPublisherActor in derive).
- Store projection pattern defined (RiskConsumerActor, RiskProjectionActor).
- Gateway endpoint specified (`GET /risk/{type}/latest`).
- Configuration schema extension designed (`risk_families`, dependency DAG).
- S60 (adapter tests) and S61 (derive actor tests) must be complete before S64.
