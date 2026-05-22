# risk — Risk assessments

The `risk` domain models risk evaluation of proposed strategy
directions. Where strategy says "go long with confidence X", risk
says "yes, allowed under current constraints" or "no, blocked
because of constraint Y".

Risk is the last gating layer before an intent reaches execution.

---

## What this domain models

A risk assessment at moment T for partition P takes a strategy
resolution as input and evaluates it against a configured set of
constraints: drawdown limits, position exposure thresholds, recent
loss accumulation, etc. The output is a verdict with rationale.

Two risk evaluator types are implemented today, each scoped to a
specific constraint family.

---

## Core types

- `RiskAssessment` — the evaluation outcome with Source, Symbol,
  Timeframe, Timestamp, Type, Verdict, snapshot of `Constraints`
  applied, and rationale.
- `StrategyInput` — the upstream strategy resolution consumed.
- `Constraints` — the parameters of the risk check (numeric thresholds,
  windows, scale factors).
- `RiskAssessedEvent` — the envelope payload carrying a `RiskAssessment`.

Each follows the canonical `Validate() *problem.Problem` pattern.

---

## Concrete risk types

The system implements 2 risk evaluators in `internal/application/risk/`
(plus a shared `risk_scaling.go` helper):

| Type identifier | Evaluator file | Purpose |
|---|---|---|
| `drawdown_limit` | `drawdown_limit_evaluator.go` | Blocks if cumulative drawdown exceeds configured threshold |
| `position_exposure` | `position_exposure_evaluator.go` | Blocks if exposure would exceed the configured cap |

The `{type}` identifier is the value the `/risk/:type/latest` HTTP
route accepts and the value embedded in the NATS subject.

---

## Event flow

- **Writer:** `derive` binary
- **Stream:** `RISK_EVENTS`
- **Consumers:**
  - `store` — per-type KV projection (`store-risk-position-exposure`,
    `store-risk-drawdown-limit`)
  - `writer` — per-type ClickHouse persistence (`writer-risk-position-exposure`,
    `writer-risk-drawdown-limit`)

### Subject taxonomy

```
risk.events.{type}.assessed
risk.query.{type}.latest
```

For example:
- `risk.events.drawdown_limit.assessed`
- `risk.events.position_exposure.assessed`
- `risk.query.drawdown_limit.latest`

The partition key is encoded in the subject suffix appended by the
publisher. For exact form, see
`internal/adapters/nats/natsrisk/registry.go`.

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | `internal/adapters/nats/natsrisk/` | Stream + publisher + 2 store-side and 2 writer-side consumer specs |
| Application (producer) | `internal/application/risk/` | 2 per-type evaluators + `risk_scaling.go` shared helper |
| Application (reader) | `internal/application/riskclient/` | Gateway-side read client |
| ClickHouse | `internal/adapters/clickhouse/risk_reader.go` | `risk_assessments` table |

---

## KV bucket coverage

Both risk types have `_LATEST` buckets (verified in
`internal/adapters/nats/natsrisk/kv_store.go`):

| Type | KV `_LATEST` bucket |
|---|---|
| `drawdown_limit` | `RISK_DRAWDOWN_LIMIT_LATEST` |
| `position_exposure` | `RISK_POSITION_EXPOSURE_LATEST` |

Full coverage.

---

## HTTP surface

One operational route: `GET /risk/:type/latest` (see
[`../HTTP-API.md`](../HTTP-API.md) → Domain latest group).

Analytical history: `GET /analytical/risk/history` with `type` and
`disposition` query params.

---

## Known anomalies and patterns

Follows canonical FamilyProcessor + Pipeline + FamilyDeps patterns.
Full KV bucket coverage. No domain-specific anomalies observed.

Risk is the **smallest derivation domain by concrete-type count** (2,
vs 6 for signal and 3 each for decision and strategy). This reflects
the deliberately narrow scope of automated risk evaluation in the
system today: two constraints, both numeric, both per-partition.

There is also a `multi_symbol_concurrency_test.go` in the application
package, indicating that multi-symbol concurrent evaluation has been
explicitly verified — relevant when the system runs against multiple
symbols simultaneously.

---

## Reading further

| If you want | Go to |
|---|---|
| The strategies being risk-checked | [strategy.md](strategy.md) |
| How risk verdicts gate execution | [execution.md](execution.md) |
| HTTP endpoints | [`../HTTP-API.md`](../HTTP-API.md) |
