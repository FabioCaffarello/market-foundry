# ADR 0023: Storage tier roadmap

## Status

Proposed. Foundation ADR delivered in Onda H-2 of the Fase Harvest;
promoted to `Accepted` in stages — partially by Ondas H-8/H-9
(insights persistence in the current tier), fully by Onda H-10
(TimescaleDB adoption if and when triggers fire). See "Promoção
para Accepted" below.

## Date

2026-05-24.

## Context

market-foundry's storage today has two tiers and one explicit gap:

- **Cold / analytical tier** — ClickHouse, governed by ADR-0003.
  Forward-only migrations (per ADR-0003 / I3); writer binary
  populates from the event mesh; historical aggregations and
  bulk queries served from here.
- **Hot / operational tier** — NATS KV projections, governed by
  ADR-0008 single-writer per bucket and projected by actors in
  the `store` binary. Latest-state reads (`CANDLE_LATEST`,
  `SIGNAL_RSI_LATEST`, etc.) served from here.
- **Gap** — there is no explicit warm/operational tier between
  in-memory KV and ClickHouse analytical.

The harvest's reference (`market-raccoon`) adopted **both**
TimescaleDB (warm) and ClickHouse (cold) from early in its
evolution (raccoon ADR-0019). The maintainer's planning question
during Harvest scoping was: *does the foundry need TimescaleDB
too, and if so, when?*

The answer, after evaluation, is: **yes, but at the moment driven
by empirical signals — not pre-emptively.** Adopting TimescaleDB
now would multiply operational surface (backups, monitoring,
capacity planning, schema management) without solving a measured
problem. Deferring it permanently would risk the foundry hitting
ClickHouse's operational ceiling for sub-minute queries when
insights (H-8) and cross-venue snapshots (H-9) scale.

The four canonical query patterns of a trading bot have different
storage fit:

| Pattern | Latency budget | Cardinality | Best fit |
|---|---|---|---|
| Real-time decision (signal + risk + execute) | < 5 ms | Single key | NATS KV |
| Historical analysis (backtest, retrospective) | seconds OK | Bulk | ClickHouse |
| Backtesting (replay infrastructure, fixtures) | minutes OK | Bulk | ClickHouse + fixture files |
| Operational query (gateway: "last 24h of X by Y") | < 50 ms p99 | Range query | KV (today, with limits) or TimescaleDB |

The first three are served well by the current ClickHouse + KV
combination. The fourth — operational range queries — is served
**today** by KV projections that the `store` binary precomputes,
but the precomputation cost grows with the number of insight
shapes (H-8) and cross-venue overlays (H-9). When that cost
saturates the `store` binary's memory or KV's bucket-size limits,
the natural overflow is to a row-store with indexed range queries
— precisely TimescaleDB's strength.

This ADR records the decision **and the triggers that move it
from current state to future state**, so the decision is not
revisited from scratch in each onda.

## Decision

market-foundry adopts a **staged storage architecture** with
explicit, empirical promotion triggers.

### Stage 1 — current (Ondas H-3 through H-9)

The foundry continues with the existing two-tier topology:

| Tier | Technology | Role | Latency target |
|---|---|---|---|
| Hot / operational | NATS KV (`store` binary projections) | Latest-state reads, real-time decision support | < 5 ms |
| Cold / analytical | ClickHouse (forward-only migrations per ADR-0003) | Historical aggregation, bulk queries, audit | < 100 ms typical, seconds OK |

- Insights (H-8) persist to ClickHouse via the existing `writer`
  binary; hot projections (e.g., "latest VPVR row per
  `(venue, instrument)`") land in KV buckets newly declared per
  H-8 design.
- Cross-venue snapshots (H-9) follow the same shape: ClickHouse
  for historical, KV for latest-state.
- The `store` binary remains the single writer for all KV
  projections (preserves ADR-0008).
- No TimescaleDB. No additional warm tier. No second analytical
  engine.

This is the **default through H-9** and is sufficient for the
operational query patterns observed up to that point.

### Stage 2 — TimescaleDB adoption (Onda H-10)

The foundry promotes to a three-tier architecture **only when**
one or more of the following empirical triggers fires:

| Trigger | Signal | Threshold | Source |
|---|---|---|---|
| **T1 (latency)** | Gateway operational-query p99 latency against ClickHouse | > 50 ms sustained for 7 days | Prometheus / `make smoke` measurement |
| **T2 (memory)** | `store` binary RSS attributable to KV projections | > 4 GB per instance | Container metrics |
| **T3 (client)** | Cliente Odin (H-12+) requires "last 24h heatmap" or equivalent without precomputed aggregation, and ClickHouse cannot serve it under p99 < 200 ms | Client SLO miss | Cliente Odin operational requirement |

When any trigger fires (and is confirmed not transient by the
maintainer), Onda H-10 opens to:

1. Add TimescaleDB to the deployment stack
   (`deploy/compose/docker-compose.yml`, healthcheck, init).
2. Introduce `internal/adapters/storage/timescale/` (paralleling
   `internal/adapters/clickhouse/`).
3. Migrate **specific** query patterns (the ones that fired the
   trigger) to TimescaleDB; the rest stay where they are.
4. Update `RUNTIME.md`, `RESUMPTION.md`, and the storage
   sections of `ARCHITECTURE.md`.
5. Promote this ADR fully to `Accepted` (partial → full
   promotion).

Stage 2 is a real onda commitment, not a hypothetical. The
ordering rule is **triggers first, onda second** — the foundry
does not "preemptively prepare" TimescaleDB tooling, schemas, or
deployment automation before a trigger fires.

### Forward-only migration of any added schema

When TimescaleDB lands, its schema follows the same forward-only
rule as ClickHouse (ADR-0003 / I3). No retroactive migration of
existing ClickHouse data; new data persists to both as the migration
proceeds. If backfill is needed, it is an explicit operational
project, not implicit in the ADR.

## Non-goals

- **TimescaleDB schema design.** H-10 designs schemas at the time
  of adoption against then-current query patterns; this ADR does
  not pre-commit shapes.
- **Migration of historical ClickHouse data to TimescaleDB.**
  Forward-only by default; explicit project if required.
- **Object storage for replay** (S3-style, raccoon-style fixture
  blob store). Separate concern; the foundry currently keeps
  replay fixtures (when introduced in H-4 per ADR-0019) as files
  alongside the test corpus.
- **Cassandra / ScyllaDB / DynamoDB.** Considered and rejected
  below; not entertained as fallback paths.
- **Removal of NATS KV as the hot tier.** Stage 2 adds
  TimescaleDB as **additional** hot/operational storage; KV
  remains for sub-millisecond latest-state reads.
- **Dual-write semantics across ClickHouse and TimescaleDB.**
  H-10 designs the write topology at adoption time; this ADR does
  not pre-commit dual-write or single-write.
- **Per-onda re-evaluation of triggers.** Triggers are evaluated
  continuously by the maintainer; an onda does not need to
  re-litigate the decision unless its own work pushes a metric
  past a threshold.

## Alternatives considered

- **(A) Adopt TimescaleDB now, in H-2 or H-3.** Rejected: no
  measured problem to solve; multiplies operational surface
  without solving a current bottleneck; defers other onda work
  for speculative benefit. Raccoon adopted both early and bears
  the operational cost; foundry does not need to inherit it.
- **(B) Commit to never adopting TimescaleDB.** Rejected: assumes
  ClickHouse + KV will scale to every future operational query
  pattern; insights (H-8) and cross-venue (H-9) are likely to
  challenge that assumption; closing the door is overconfident.
- **(C) Adopt a different warm-tier store**
  (Cassandra / ScyllaDB / DynamoDB). Rejected: TimescaleDB is the
  best fit for the foundry's workload pattern (time-series-keyed
  range queries) and the raccoon's operational experience
  validates it in the same problem space. Adopting a less-proven
  alternative without a forcing reason would be undisciplined.
- **(D) Move hot tier off NATS KV to Redis / Memcached.**
  Rejected: NATS KV already integrates with the writer-per-bucket
  invariant (ADR-0008) and the actor projection model; a separate
  cache adds operational surface for a marginal win.
- **(E) Use ClickHouse for both hot and cold by leveraging
  in-memory tables / engines.** Rejected: ClickHouse's in-memory
  shapes are not designed for the sub-5-ms hot path the foundry
  needs; KV's value is precisely the latency profile this
  alternative would not deliver.

## Consequences

### Positive

- **Storage tier matches measured query patterns**, not anticipated
  ones; avoids speculative engineering.
- **Operational surface stays bounded** until empirical signals
  justify expanding it.
- **The TimescaleDB decision is pre-made**; when a trigger fires,
  H-10 opens immediately on a known plan rather than re-litigating
  from scratch.
- **Trigger thresholds are concrete**, so the decision is
  reviewable: anyone can check whether a threshold has been
  breached and whether the breach is sustained.
- **Aligns with foundry's "no aspirational claims" principle
  (I9)**: the ADR does not claim TimescaleDB capability the
  foundry has not shipped.

### Negative

- **Deferred decision creates expectation drift.** Contributors
  may design as if TimescaleDB exists, then have to retrofit when
  it does not. Mitigated by explicit Stage 1 / Stage 2 labelling
  in this ADR and in RUNTIME.md.
- **ClickHouse may be a temporary bottleneck.** If T1 fires and
  takes an onda to address, operational query latency may briefly
  exceed the SLO. Accepted: the tradeoff vs preemptive complexity
  is favorable.
- **Triggers depend on observability.** T1 (latency) requires
  per-route SLI measurement (per `docs/operations/slo.md`); T2
  (memory) requires container metrics; T3 (client) requires the
  client to exist. Triggers cannot fire if instrumentation is
  absent. Mitigated by SLO instrumentation being H-5 / H-10
  ondas in the natural sequence.
- **Re-evaluation of triggers carries judgment.** "Sustained 7
  days" and similar thresholds require human assessment. Accepted:
  the maintainer is the decision authority; this ADR provides the
  thresholds, not the trigger automation.

## Promoção para Accepted

This ADR is promoted in two stages:

### Partial promotion (Stage 1 confirmed)

After **Ondas H-8 and H-9** ship insights and cross-venue
snapshots on the existing ClickHouse + KV topology, this ADR's
Stage 1 description is empirically validated. The maintainer may
flip a sub-status (e.g., "Stage 1: Accepted, Stage 2: Proposed
pending triggers") in the same commit that closes H-9.

### Full promotion (Stage 2 triggered and shipped)

The `Status` field flips from `Proposed` to `Accepted` when **Onda
H-10** ships:

1. At least one trigger (T1, T2, T3) recorded as fired with
   evidence in `RESUMPTION.md`.
2. TimescaleDB integrated into the deployment stack with
   healthcheck and init.
3. `internal/adapters/storage/timescale/` package shipped
   paralleling the ClickHouse adapter.
4. At least one query pattern migrated from ClickHouse (or KV) to
   TimescaleDB with measurable latency improvement.
5. `RUNTIME.md`, `RESUMPTION.md`, and `ARCHITECTURE.md` updated
   to reflect the three-tier topology.
6. Forward-only migration policy extended to TimescaleDB (ADR-0003
   analog or new ADR if semantics diverge).

If no trigger fires through H-12+, the ADR may remain `Proposed`
indefinitely; that is a legitimate steady state, not an oversight.

## References

- ADR [0003](0003-clickhouse-analytical.md) — the existing cold /
  analytical tier; this ADR extends rather than replaces.
- ADR [0008](0008-single-writer-invariant.md) — single-writer per
  KV bucket is preserved across both stages; TimescaleDB writes
  follow the same principle when added.
- ADR [0001](0001-nats-not-kafka.md) — JetStream + NATS KV are the
  hot-tier substrate; this ADR is compatible with that choice.
- `docs/operations/runtime-invariants.md` → I3 (forward-only
  migrations) and I9 (no aspirational claims) — the principles
  this ADR follows.
- `docs/operations/slo.md` — SLO instrumentation that makes T1
  (latency trigger) measurable.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro) and "What
  this repository is NOT" → no premature TimescaleDB adoption.
- [PROGRAM-0001](../programs/PROGRAM-0001-foundation.md) — Onda
  H-2 scope; TimescaleDB is explicitly mapped as "likely H-10",
  consistent with this ADR.
- `docs/RESUMPTION.md` → "M-list" entries that capture
  trigger-style deferrals (e.g., M16/M17/M18 style "deferred
  pending counter data") — this ADR adopts that pattern at ADR
  level rather than M-level.
- raccoon `docs/adrs/ADR-0019-dual-database-operational-strategy.md`
  — inspiração. Foundry diverges by (a) staging the decision in
  two explicit phases with empirical promotion triggers, rather
  than adopting both stores upfront (raccoon's ADR-0019 documents
  an already-operational dual-store); (b) preserving NATS KV as
  the hot tier (raccoon's hot tier is in-memory ring buffers
  inside the process; foundry's projection model is NATS KV per
  ADR-0008, which is structurally different); (c) leaving Stage 2
  schema, dual-write semantics, and migration ordering for H-10
  to decide at adoption time rather than pre-committing now; and
  (d) explicitly allowing the ADR to remain `Proposed` if no
  trigger fires — adopting "stay in Stage 1 if it works" as a
  legitimate steady state.
