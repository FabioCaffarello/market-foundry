# Risk Stream Families

> Stage S62 — Approved 2026-03-18
> Status: **DESIGN ONLY — no implementation in this stage**

---

## 1. What Is a Risk Family

A risk family is a **named group of risk evaluators** that share:

- A common evaluation boundary (what risk rule is applied).
- A single producing binary (derive).
- Consistent subject encoding within `RISK_EVENTS`.
- A shared JetStream stream with family-scoped subject filtering.
- Common retention policy.
- One projection pipeline per family in store.

Risk families follow the exact same organizational pattern as decision families and strategy families.

---

## 2. Family Catalog

### RF-01: Position Exposure (Phase 1)

| Property             | Value                                                              |
|----------------------|--------------------------------------------------------------------|
| Canonical name       | `position_exposure`                                                |
| Phase                | 1 (first slice)                                                    |
| Input                | `mean_reversion_entry` strategy (via local actor message)          |
| Evaluation           | Position sizing against max exposure percentage                    |
| Dispositions         | `approved` / `modified` / `rejected`                               |
| Parameters           | `max_position_pct`, `max_portfolio_exposure_pct`                   |
| Publisher            | RiskPublisherActor (derive)                                        |
| Consumer             | RiskConsumerActor (store)                                          |
| Projection bucket    | `RISK_POSITION_EXPOSURE_LATEST`                                    |
| Query endpoint       | `GET /risk/position_exposure/latest?source=X&symbol=Y&timeframe=Z` |
| Dependencies         | strategy_families: `["mean_reversion_entry"]`                      |

**Evaluation Logic (Position Exposure):**

1. Receive strategy intent (direction, confidence).
2. Calculate proposed position size based on confidence and `max_position_pct`.
3. Check against `max_portfolio_exposure_pct`.
4. If within limits → `approved` with calculated `max_position_size`.
5. If exceeds limits but direction valid → `modified` with reduced `max_position_size`.
6. If strategy direction is `flat` or confidence below minimum → `rejected` with rationale.

This is a **stateless evaluation** — no external state, no position database, no portfolio aggregation. Phase 1 risk is rule-based, not state-based.

---

### RF-02: Drawdown Guard (Deferred — S65+)

| Property             | Value                                                              |
|----------------------|--------------------------------------------------------------------|
| Canonical name       | `drawdown_guard`                                                   |
| Phase                | Deferred                                                           |
| Input                | Strategy + historical P&L (requires execution/portfolio layers)    |
| Evaluation           | Max drawdown threshold check                                       |
| Blocking dependency  | Execution and portfolio domains must exist                         |
| Rationale for deferral | Requires state that does not yet exist in the system              |

---

### RF-03: Correlation Limit (Deferred — S66+)

| Property             | Value                                                              |
|----------------------|--------------------------------------------------------------------|
| Canonical name       | `correlation_limit`                                                |
| Phase                | Deferred                                                           |
| Input                | Strategy + active positions across symbols                         |
| Evaluation           | Cross-symbol correlation exposure check                            |
| Blocking dependency  | Portfolio-level aggregation                                        |
| Rationale for deferral | Requires multi-symbol portfolio state (not individual partition)  |

---

### RF-04: Volatility Scaler (Deferred — S66+)

| Property             | Value                                                              |
|----------------------|--------------------------------------------------------------------|
| Canonical name       | `volatility_scaler`                                                |
| Phase                | Deferred                                                           |
| Input                | Strategy + real-time volatility evidence                           |
| Evaluation           | Position size adjustment based on current volatility regime        |
| Blocking dependency  | Volatility evidence family (evidence layer expansion)              |
| Rationale for deferral | Requires evidence types not yet implemented                       |

---

## 3. Stream Definition

| Property           | Value                                                              |
|--------------------|--------------------------------------------------------------------|
| Stream name        | `RISK_EVENTS`                                                      |
| Subject pattern    | `risk.events.{type}.assessed.{source}.{symbol}.{timeframe}`       |
| Retention          | Limits                                                             |
| Max age            | 72h                                                                |
| Storage            | File                                                               |
| Discard            | Old                                                                |
| Deduplication      | Enabled (window: 2m)                                               |
| Replicas           | 1 (single-node)                                                    |

### Subject Examples

```
risk.events.position_exposure.assessed.binancef.btcusdt.60
risk.events.position_exposure.assessed.binancef.ethusdt.300
risk.events.drawdown_guard.assessed.binancef.btcusdt.60       # future
```

All families share the `RISK_EVENTS` stream. Subject filtering isolates families for consumer processing.

---

## 4. KV Buckets

### Phase 1

| Bucket                              | Key Format                        | Writer                  | Reader               |
|-------------------------------------|-----------------------------------|-------------------------|----------------------|
| `RISK_POSITION_EXPOSURE_LATEST`     | `{source}.{symbol}.{timeframe}`   | RiskProjectionActor     | QueryResponderActor  |

### Future (Deferred)

| Bucket                              | Phase     | Notes                              |
|-------------------------------------|-----------|-------------------------------------|
| `RISK_POSITION_EXPOSURE_HISTORY`    | S65+      | If history queries needed           |
| `RISK_DRAWDOWN_GUARD_LATEST`        | S65+      | When drawdown family implemented    |
| `RISK_CORRELATION_LIMIT_LATEST`     | S66+      | When correlation family implemented |
| `RISK_VOLATILITY_SCALER_LATEST`     | S66+      | When volatility family implemented  |

---

## 5. Durable Consumers

| Consumer Name                         | Stream        | Filter                                              | Binary |
|---------------------------------------|---------------|------------------------------------------------------|--------|
| `risk-position-exposure-store`        | RISK_EVENTS   | `risk.events.position_exposure.assessed.>`           | store  |

One durable consumer per family in store. Consumer name follows pattern: `{domain}-{family}-{binary}`.

---

## 6. Family Growth Pattern

Adding a new risk family follows the same 9-step checklist as all prior domains:

1. **Design** — Architecture doc with family definition, evaluation logic, boundaries.
2. **Domain** — Entity, event, validation in `internal/domain/risk/`.
3. **Actor** — RiskEvaluatorActor variant in `internal/actors/scopes/derive/`.
4. **Config** — Family registration in `schema.go`, dependency DAG entry.
5. **Store** — Consumer + projection actor in `internal/actors/scopes/store/`.
6. **Query** — QueryResponderActor subject registration in store.
7. **Gateway** — HTTP handler + route + use case in gateway.
8. **Governance** — raccoon-cli rules for new family.
9. **Test** — Domain tests, actor tests, adapter tests, integration tests.

---

## 7. Dependency DAG Extension

```go
// internal/shared/settings/schema.go (design — not implemented yet)

var riskDependsOnStrategy = map[string][]string{
    "position_exposure": {"mean_reversion_entry"},
}
```

Config validation ensures:
- If `risk_families` contains `"position_exposure"`, then `strategy_families` must contain `"mean_reversion_entry"`.
- The full chain is validated: risk → strategy → decision → signal → evidence.
- Unknown risk family names are rejected at startup.

---

## 8. Data Flow

```
                    derive binary
┌─────────────────────────────────────────────────────┐
│  SourceScopeActor                                   │
│    StrategyResolverActor ──(local msg)──►            │
│                             RiskEvaluatorActor       │
│                               │                      │
│                               ▼                      │
│                         RiskPublisherActor            │
└─────────────────────┬───────────────────────────────┘
                      │
                      ▼
              RISK_EVENTS (JetStream)
                      │
                      ▼
┌─────────────────────────────────────────────────────┐
│  store binary                                        │
│    RiskConsumerActor → RiskProjectionActor            │
│                          │                           │
│                          ▼                           │
│              RISK_POSITION_EXPOSURE_LATEST (KV)      │
│                          │                           │
│                          ▼                           │
│              QueryResponderActor                     │
└─────────────────────┬───────────────────────────────┘
                      │
                      ▼ (NATS request/reply)
┌─────────────────────────────────────────────────────┐
│  gateway binary                                      │
│    GET /risk/position_exposure/latest                 │
└─────────────────────────────────────────────────────┘
```
