# Stream Families

> Canonical catalog of stream families in the Market Foundry mesh.

## What Is a Stream Family

A **stream family** is a named group of related message flows that share:
- a common domain boundary;
- a single producing binary;
- a consistent subject encoding pattern;
- a shared JetStream stream (for event-surface families);
- a common retention and delivery semantic.

Stream families are the primary organizational unit of the mesh. Adding a new family is an architectural decision, not a configuration change.

## Family Classification

Families are classified by their **flow character**:

| Classification | Description | Delivery | Retention | Examples |
|----------------|-------------|----------|-----------|----------|
| **Continuous** | Unbounded flow of raw events, one per external occurrence | At-least-once, ordered per partition | Short (hours) | observation |
| **Sampled** | Events produced at fixed time intervals from continuous input | At-least-once, ordered per partition | Medium (days) | evidence |
| **Derived** | Events produced by applying domain logic to sampled or other derived input | At-least-once, ordered per partition | Medium (days) | signal (future) |
| **Lifecycle** | State transition events for system-managed entities | At-least-once, deliver-last-per-subject | Medium (days) | configctl |
| **Projection** | Notifications that a materialized view was updated | At-most-once or at-least-once | Short (hours) | projection (future) |
| **Query-only** | No stream — synchronous request/reply only | Request/reply | N/A (stateless) | evidence.query, configctl.control |

## Current Families (Implemented)

### configctl — Configuration Lifecycle

| Property | Value |
|----------|-------|
| **Status** | Active |
| **Classification** | Lifecycle |
| **Stream** | `CONFIGCTL_EVENTS` |
| **Writer** | configctl (embedded) |
| **Consumers** | ingest (binding-watcher), derive (binding-watcher) |
| **Subject pattern** | `configctl.events.config.{lifecycle_verb}` |
| **Retention** | 24h, 256 MB |
| **Partitioning** | None (single config authority) |
| **Query surface** | `configctl.control.{operation}` (request/reply) |

**Event types:**
- `draft_created`, `validated`, `compiled`, `activated`, `deactivated`, `ingestion_runtime_changed`, `archived`, `rejected`

**Role in mesh:** Foundation family. All other families depend on configctl for activation signals. The `ingestion_runtime_changed` event is the trigger that starts ingest and derive pipelines.

---

### observation — Raw Market Data

| Property | Value |
|----------|-------|
| **Status** | Active |
| **Classification** | Continuous |
| **Stream** | `OBSERVATION_EVENTS` |
| **Writer** | ingest |
| **Consumers** | derive (`derive-observation`) |
| **Subject pattern** | `observation.events.market.trade.{source}` |
| **Retention** | 6h, 1 GB |
| **Partitioning** | By `source` |
| **Query surface** | None (future: `observation.query.latest.*`) |

**Event types:**
- `market.trade_received` — normalized trade from external exchange

**Role in mesh:** Entry point for all market data. Every trade from every exchange enters through this family. Partitioning by source (not by symbol) keeps the subject space manageable and aligns with ingest's per-exchange scope isolation.

**Design rationale — source-level partitioning:**
Observation events are partitioned by source only, not by symbol. This reflects the physical reality: one WebSocket connection per exchange delivers all symbols for that exchange. Symbol-level partitioning at the observation layer would create subject explosion without benefit — derive already routes internally by symbol.

---

### evidence — Derived Market Facts

| Property | Value |
|----------|-------|
| **Status** | Active |
| **Classification** | Sampled |
| **Stream** | `EVIDENCE_EVENTS` |
| **Writer** | derive |
| **Consumers** | store (`store-candle`, `store-trade-burst`, `store-volume`) |
| **Subject pattern** | `evidence.events.{type}.sampled.{source}.{symbol}.{timeframe}` |
| **Retention** | 72h, 2 GB |
| **Partitioning** | By `source`, `symbol`, `timeframe` |
| **Query surface** | `evidence.query.{type}.{operation}` (request/reply) |

**Event types:**
- `candle.sampled` — OHLCV candle for a completed or interim window
- `tradeburst.sampled` — trade activity summary with burst detection
- `volume.sampled` — volume profile with buy/sell distribution and VWAP

**KV projections:**
- `CANDLE_LATEST` — last candle per source/symbol/timeframe (64 MB)
- `CANDLE_HISTORY` — time-windowed candle archive (256 MB, 24h TTL)
- `TRADE_BURST_LATEST` — last trade burst per source/symbol/timeframe (64 MB)
- `VOLUME_LATEST` — last volume profile per source/symbol/timeframe (64 MB)

**Role in mesh:** The first domain-specific family. Evidence is the bridge between raw observation and actionable insight. Each evidence type follows the [evidence-read-model-guidelines](evidence-read-model-guidelines.md) checklist and the [evidence-derivation-pattern](evidence-derivation-pattern.md).

**Design rationale — full partitioning:**
Evidence events carry the full partition key (`source.symbol.timeframe`) in the subject. This enables per-type durable consumers with filter subjects and allows store to consume only the evidence types it projects, without receiving irrelevant traffic.

---

## Planned Families (Documented, Not Implemented)

### signal — Trading Signals

| Property | Value |
|----------|-------|
| **Status** | Planned |
| **Classification** | Derived |
| **Stream** | `SIGNAL_EVENTS` |
| **Writer** | derive |
| **Consumers** | store (future) |
| **Subject pattern** | `signal.events.{type}.{verb}.{source}.{symbol}.{timeframe}` |
| **Retention** | 72h (estimated) |
| **Partitioning** | By `source`, `symbol`, `timeframe` |
| **Query surface** | `signal.query.{type}.{operation}` (future) |

**Anticipated event types:**
- Signal activation/deactivation events
- Signal strength/confidence updates

**Prerequisite:** Signal readiness review (S25) identified config-driven activation as a blocking prerequisite. Signal must not enter the mesh until the activation mechanism is proven.

**Design notes:**
- Signals derive from evidence, not from observation. A signal never reads raw trades.
- Signal and evidence share the derive binary but use separate streams. This preserves single-writer per stream.
- Signal families may introduce a new dimension: `strategy` or `model`, depending on domain modeling outcomes.

---

### projection — Materialization Notifications

| Property | Value |
|----------|-------|
| **Status** | Planned |
| **Classification** | Projection |
| **Stream** | `PROJECTION_EVENTS` |
| **Writer** | store |
| **Consumers** | gateway (future, for cache invalidation) |
| **Subject pattern** | `projection.events.{family}.{type}.materialized` |
| **Retention** | Short (1-2h, estimated) |
| **Partitioning** | By `family`, `type` |
| **Query surface** | None |

**Role in mesh:** Enables gateway or other consumers to react to projection updates without polling. This is the only family where store is the writer.

---

## Future Families (Conceptual, Not Documented)

These families are referenced in system-vision.md as part of the seven-domain model. They have no architecture documents, no stream specs, and no implementation timeline. They are listed here for completeness and to establish naming reservations.

| Family | Classification | Writer | Domain |
|--------|---------------|--------|--------|
| `decision` | Derived | TBD | Strategy/Decision |
| `risk` | Derived | TBD | Risk Management |
| `execution` | Lifecycle | TBD | Order Execution |
| `portfolio` | Lifecycle | TBD | Portfolio State |

**Naming reservation:** These family names are reserved. No other family may use them. No code may reference these families until an architecture document exists.

**Influence from Market Raccoon:** The seven-domain progression (observation → evidence → signal → decision → risk → execution → portfolio) is a conceptual inheritance from Market Raccoon. The Foundry will implement these domains with its own boundaries, contracts, and stream semantics — not by copying Raccoon's structure.

---

## Stream Family Invariants

These rules apply to all families, current and future:

1. **One stream per family per surface.** Evidence has one JetStream stream (`EVIDENCE_EVENTS`), not one per type. Types are differentiated by subject filters.

2. **One writer per stream.** No stream accepts events from multiple binaries. This is enforced by architecture review and raccoon-cli audits.

3. **Family names are lowercase, singular nouns.** `evidence`, not `evidences` or `Evidence`. `signal`, not `signals`.

4. **New families require architecture documents.** A family does not exist until it has: a stream-families entry, an ownership declaration in actor-ownership.md, and a subject encoding specification.

5. **Families do not share streams.** Evidence and signal events flow through separate JetStream streams, even though both are produced by derive. This prevents consumer coupling and allows independent retention policies.

6. **Query surfaces are family-scoped.** All evidence queries live under `evidence.query.*`. All signal queries will live under `signal.query.*`. Cross-family queries are prohibited.

7. **KV buckets are type-scoped within a family.** Within evidence, each type (candle, tradeburst) has its own KV buckets. Buckets are never shared across types.

## Family Addition Checklist

Before adding a new stream family:

- [ ] Architecture document defining family purpose, classification, and ownership
- [ ] Subject encoding pattern documented
- [ ] Stream spec (retention, max bytes, storage type) defined
- [ ] Consumer list with durable names and filter subjects
- [ ] Actor ownership entry in actor-ownership.md
- [ ] raccoon-cli contract audit rules updated
- [ ] Stage report documenting the decision and rationale
