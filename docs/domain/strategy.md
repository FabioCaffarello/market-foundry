# strategy — Direction resolutions

The `strategy` domain models the resolution of one or more decisions
into a trading direction. Where decisions are independent verdicts,
strategies are the **integration step**: "given these decisions, what
is the net stance — long, short, or flat — for this partition?"

---

## What this domain models

A strategy resolution at moment T for partition P combines a set of
recent decisions and produces a directional verdict. The verdict
carries direction (Long / Short / Flat), confidence (numeric or
categorical), and the upstream decisions consumed (for traceability).

Strategies are the **last derivation step** before execution. Their
outputs are read by:
- `risk` — for risk assessment against the proposed direction
- `execute` — for actual intent generation (when strategy + risk green-light an action)

Each strategy type has its own resolver in
`internal/application/strategy/` following the FamilyProcessor pattern.

---

## Core types

- `Strategy` — the resolved direction with Source, Symbol, Timeframe,
  Timestamp, Type, Direction, Confidence, plus a severity-scaling
  output (see `severity_scaling.go` in the application package).
- `DecisionInput` — the upstream decisions consumed (for traceability).
- `StrategyResolvedEvent` — the envelope payload carrying a `Strategy`.

Each follows the canonical `Validate() *problem.Problem` pattern.

---

## Concrete strategy types

The system implements 3 strategy resolvers in
`internal/application/strategy/` (plus a shared `severity_scaling.go`
helper used across resolvers):

| Type identifier | Resolver file | Consumes decision(s) | Purpose |
|---|---|---|---|
| `mean_reversion_entry` | `mean_reversion_entry_resolver.go` | `rsi_oversold` | Long when oversold + price below mean |
| `squeeze_breakout_entry` | `squeeze_breakout_entry_resolver.go` | `bollinger_squeeze` | Long/short when Bollinger squeeze releases |
| `trend_following_entry` | `trend_following_entry_resolver.go` | `ema_crossover` | Long/short with EMA crossover direction |

The `{type}` identifier is the value the `/strategy/:type/latest`
HTTP route accepts and the value embedded in the NATS subject.

---

## Event flow

- **Writer:** `derive` binary
- **Stream:** `STRATEGY_EVENTS`
- **Consumers (three binaries):**
  - `store` — per-type KV projection (only for types with a `_LATEST` bucket; 2 of 3 today)
  - `writer` — per-type ClickHouse persistence (`writer-strategy-mean-reversion-entry`,
    `writer-strategy-trend-following-entry`, `writer-strategy-squeeze-breakout-entry`)
  - **`execute`** — for executing on strategy outputs (durable
    `execute-strategy-mean-reversion-entry`; today only the
    mean-reversion-entry strategy has an execute-side consumer)

The presence of `execute` as a third consumer is what distinguishes
strategy from signal/decision/risk: strategy is the layer that drives
execution intent generation.

### Subject taxonomy

```
strategy.events.{type}.resolved
strategy.query.{type}.latest
```

For example:
- `strategy.events.mean_reversion_entry.resolved`
- `strategy.events.trend_following_entry.resolved`
- `strategy.query.squeeze_breakout_entry.latest`

The partition key is encoded in the subject suffix appended by the
publisher. For exact form, see
`internal/adapters/nats/natsstrategy/registry.go`.

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | `internal/adapters/nats/natsstrategy/` | Stream + publisher + per-type consumer specs (store + writer; plus the execute-side mean-reversion-entry consumer) |
| Application (producer) | `internal/application/strategy/` | 3 per-type resolvers + `severity_scaling.go` shared helper |
| Application (reader) | `internal/application/strategyclient/` | Gateway-side read client |
| ClickHouse | `internal/adapters/clickhouse/strategy_reader.go` | `strategies` table |

---

## KV bucket coverage

Not every strategy type has a `_LATEST` KV bucket (verified in
`internal/adapters/nats/natsstrategy/kv_store.go`):

| Type | KV `_LATEST` bucket | Operational read (`/strategy/:type/latest`) |
|---|---|---|
| `mean_reversion_entry` | `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` ✓ | works |
| `trend_following_entry` | `STRATEGY_TREND_FOLLOWING_ENTRY_LATEST` ✓ | works |
| `squeeze_breakout_entry` | — | returns 404 |

The `squeeze_breakout_entry` type is part of the **G2 gap** documented
in [`../RESUMPTION.md`](../RESUMPTION.md). It flows through the stream
and persists in ClickHouse, but has no operational `_LATEST` projection.

---

## HTTP surface

One operational route: `GET /strategy/:type/latest` (see
[`../HTTP-API.md`](../HTTP-API.md) → Domain latest group).

Analytical history: `GET /analytical/strategy/history` with `type` and
`direction` query params.

---

## Known anomalies and patterns

Follows canonical FamilyProcessor + Pipeline + FamilyDeps patterns.

**Unique pattern:** strategy is the only derivation domain whose
events are consumed by **three** binaries (store, writer, execute).
All other derivation domains have two consumers (store and writer
only). This reflects strategy's role as the bridge to execution.

**Asymmetric execute consumption:** today only `mean_reversion_entry`
has an execute-side consumer (`execute-strategy-mean-reversion-entry`).
The other two strategy types resolve and persist but do not yet drive
execution intents.

---

## Reading further

| If you want | Go to |
|---|---|
| The decisions feeding strategies | [decision.md](decision.md) |
| Risk assessment of strategies | [risk.md](risk.md) |
| How strategies become execution intents | [execution.md](execution.md) |
| KV gap context | [`../RESUMPTION.md`](../RESUMPTION.md) → G2 |
