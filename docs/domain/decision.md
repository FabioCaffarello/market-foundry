# decision — Evaluator outputs

The `decision` domain models discrete evaluations of signal outputs.
Where signals are continuous computations, decisions are categorical
verdicts: "this signal currently meets criteria X" or "it does not".

Decisions consume one or more signals and emit a binary or trinary
verdict scoped to a partition.

---

## What this domain models

A decision is the output of an evaluator at moment T for partition P,
based on the most recent signal value(s). Each evaluator type
(decision type) implements specific logic: "RSI is below 30",
"EMA fast crossed above EMA slow", "Bollinger Band squeezing into
mean".

Decisions feed strategies, which combine multiple decisions into a
direction resolution. They also flow to risk evaluation and to
analytical history.

Each decision type has its own evaluator in
`internal/application/decision/` following the FamilyProcessor
pattern, analogous to signals.

---

## Core types

- `Decision` — the categorical verdict, with Source, Symbol, Timeframe,
  Timestamp, Type, plus severity and rationale fields (added in S234
  via migration `007_add_decision_severity_rationale.sql`).
- `SignalInput` — the upstream signal data the decision was built on
  (for traceability and reproducibility).
- `DecisionEvaluatedEvent` — the envelope payload carrying a `Decision`.

Each follows the canonical `Validate() *problem.Problem` pattern.

---

## Concrete decision types

The system implements 3 decision evaluators in
`internal/application/decision/`:

| Type identifier | Evaluator file | Reads signal(s) | Purpose |
|---|---|---|---|
| `rsi_oversold` | `rsi_oversold_evaluator.go` | `rsi` | RSI below configured threshold (typically 30) |
| `ema_crossover` | `ema_crossover_evaluator.go` | `ema_crossover` | EMA fast crossed slow (direction-aware) |
| `bollinger_squeeze` | `bollinger_squeeze_evaluator.go` | `bollinger` | Bollinger Band narrowing toward mean |

The `{type}` identifier is the value the `/decision/:type/latest`
HTTP route accepts and the value embedded in the NATS subject.

---

## Event flow

- **Writer:** `derive` binary
- **Stream:** `DECISION_EVENTS`
- **Consumers:**
  - `store` — per-type KV projection (3 durables: `store-decision-rsi-oversold`,
    `store-decision-ema-crossover`, `store-decision-bollinger-squeeze`)
  - `writer` — per-type ClickHouse persistence (`writer-decision-rsi-oversold`,
    `writer-decision-ema-crossover`, `writer-decision-bollinger-squeeze`)

### Subject taxonomy

```
decision.events.{type}.evaluated
decision.query.{type}.latest
```

For example:
- `decision.events.rsi_oversold.evaluated`
- `decision.events.ema_crossover.evaluated`
- `decision.query.bollinger_squeeze.latest`

The partition key is encoded in the subject suffix appended by the
publisher. For exact form, see
`internal/adapters/nats/natsdecision/registry.go`.

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | `internal/adapters/nats/natsdecision/` | Stream + publisher + 3 store-side and 3 writer-side consumer specs |
| Application (producer) | `internal/application/decision/` | 3 per-type evaluators, FamilyProcessor pattern |
| Application (reader) | `internal/application/decisionclient/` | Gateway-side read client |
| ClickHouse | `internal/adapters/clickhouse/decision_reader.go` | `decisions` table |

---

## KV bucket coverage

All 3 decision types have `_LATEST` buckets (verified in
`internal/adapters/nats/natsdecision/kv_store.go`):

| Type | KV `_LATEST` bucket |
|---|---|
| `rsi_oversold` | `DECISION_RSI_OVERSOLD_LATEST` |
| `ema_crossover` | `DECISION_EMA_CROSSOVER_LATEST` |
| `bollinger_squeeze` | `DECISION_BOLLINGER_SQUEEZE_LATEST` |

Full coverage — no gap here (unlike signal and strategy).

---

## HTTP surface

One operational route: `GET /decision/:type/latest` (see
[`../HTTP-API.md`](../HTTP-API.md) → Domain latest group).

Analytical history: `GET /analytical/decision/history` with `type` and
`outcome` filters.

The composite analytical surface under `/analytical/composite/decision/*`
provides explainability — decision review, batch effectiveness,
effectiveness summary — combining decisions with downstream
effectiveness attribution. See [effectiveness.md](effectiveness.md) for
the read-side classification model behind those endpoints.

---

## Known anomalies and patterns

Follows canonical FamilyProcessor + Pipeline + FamilyDeps patterns.
Full KV bucket coverage. No domain-specific anomalies observed beyond
the one-to-one signal→decision mapping (each evaluator reads exactly
one signal type — no multi-signal decisions today).

---

## Reading further

| If you want | Go to |
|---|---|
| The signals feeding decisions | [signal.md](signal.md) |
| The next layer (strategies combining decisions) | [strategy.md](strategy.md) |
| Read-side P&L classification | [effectiveness.md](effectiveness.md) |
| Composite analytical surface | [`../HTTP-API.md`](../HTTP-API.md) → Analytical composite |
