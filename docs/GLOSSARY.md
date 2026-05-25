# Glossary

Terms specific to market-foundry. Generic terms (actor, goroutine,
HTTP, NATS, ClickHouse) are not defined here — see their respective
upstream documentation.

---

## Architecture

**Foundry**
The substrate — the layered, message-driven foundation on which
market-domain capabilities are built. The repository is named for
this concept. It is deliberately not an application.

**Stream mesh**
The architecture of communication between binaries. Every data flow
is a named, typed, ownership-bound message flow on NATS+JetStream.
The mesh has explicit producers, consumers, partitions, and
deduplication keys.

**Family**
A class of events flowing through one stream. Current families:
`configctl`, `observation`, `evidence`, `signal`, `decision`,
`strategy`, `risk`, `execution`.

**Surface**
The plane a message belongs to in NATS subject taxonomy. Common planes:
`events` (published facts), `control` (commands), `query` (request/reply),
`projection` (KV refresh). The execution domain adds `fill`, `rejection`,
`session`, `activation` for lifecycle specifics. The configctl domain
currently uses both `events` and `event` (singular) in parallel —
transitional, see [`RUNTIME.md`](RUNTIME.md).

**Domain**
The business area a piece of code belongs to. Domains live under
`internal/domain/` and map one-to-one with families (plus a few
internal-only domains: `consistency`, `effectiveness`, `lineage`,
`monitoring`, `pairing`, `triage`).

**Layer sovereignty**
The rule that imports flow strictly inward:
`domain → application → adapters → actors → interfaces → cmd`.
Enforced automatically by `raccoon-cli arch-guard`.

**Venue**
An exchange (or exchange-family product) the foundry can source
market data from or route orders to. Per
[ADR-0021](decisions/0021-canonical-instrument-and-venue-model.md),
the `Venue` enum is the canonical identifier
(`binance`, `binancef`, `bybit`, `coinbase`, `hyperliquid`,
`kraken`, `krakenf`). Carried at the envelope level
(ADR-0017) so cross-venue capabilities route without payload
inspection.

**Canonical instrument**
The foundry-internal identity of a tradable instrument, defined in
[ADR-0021](decisions/0021-canonical-instrument-and-venue-model.md)
as `CanonicalInstrument{Base, Quote, Contract}` where `Contract`
is one of `spot`, `usdtfutures`, `coinfutures`, `perpetual`.
Identical structure across every venue; venue-native nuances (lot
sizes, tick sizes) live in adapter-side metadata, not in the
canonical identity. Domain layer (`internal/domain/`) never
handles venue-native symbol formats; normalization happens at the
adapter boundary via `ToCanonical` / `FromCanonical`.

**Storage tier**
The class of persistent store a piece of data lives in. Per
[ADR-0023](decisions/0023-storage-tier-roadmap.md), the foundry's
**Stage 1** topology is two tiers: **hot / operational** (NATS KV
projections, sub-5-ms latest-state reads) and **cold / analytical**
(ClickHouse, time-range queries and aggregations). **Stage 2**
(Onda H-10) adds TimescaleDB as a warm/operational tier when
empirical triggers fire; until then, "storage tier" refers to the
two-tier model.

---

## Binaries

**configctl**
The configuration lifecycle service. Sole authority over configuration
state transitions (Draft → Validated → Compiled → Active → Deactivated).

**gateway**
The HTTP API gateway. Stateless translator between HTTP and NATS
request/reply. Owns no business logic.

**ingest**
The market data capture service. Subscribes to exchange WebSockets
(Binance Spot, Binance Futures) and publishes normalized
`OBSERVATION_EVENTS`.

**derive**
The derivation pipeline. Consumes `OBSERVATION_EVENTS` and produces
downstream domain events: `EVIDENCE_EVENTS`, `SIGNAL_EVENTS`,
`DECISION_EVENTS`, `STRATEGY_EVENTS`, `RISK_EVENTS`, `EXECUTION_EVENTS`.

**store**
The read-model materialization service. Consumes domain events and
builds KV projections served via NATS request/reply.

**execute**
The execution control service. Consumes execution intents and
materializes controlled execution state via venue adapters (paper,
testnet, mainnet).

**writer**
The analytical writer. Consumes selected domain events from JetStream
and persists them into ClickHouse for analytical reads.

**migrate**
The schema migration tool. Forward-only ClickHouse migrations with
checksum validation.

---

## Patterns

**FamilyProcessor**
The declarative registration pattern used in `derive`. Adding a new
evidence type means adding one entry to a processor list, not modifying
a spawning loop.

**ProjectionPipeline**
The declarative registration pattern used in `store`. One pipeline per
family, with exclusive write access to its KV buckets.

**EvidenceFamilyDeps**
The grouped use-case dependency pattern used in `gateway`. Each
evidence family gets its own dep group; the gateway is purely
translation, with no direct KV access.

**Single-writer invariant**
For every JetStream stream and KV bucket, exactly one binary or actor
writes. Multiple readers are fine; multiple writers are forbidden.

**Envelope (transport)**
The generic NATS message wrapper in `internal/shared/envelope/`.
Carries `Kind`, `Type`, `Source`, `Subject`, `CorrelationID`,
`CausationID`, `Timestamp`, and `Payload`. Used for in-cluster RPC
(commands, requests, replies) and for transport-level metadata on
the event flow. Distinct from the **canonical event envelope** (see
below), which adds domain-event fields (`venue`, `instrument`,
`seq`, `idempotency_key`, etc.) for events on the 11 streams.

**Canonical event envelope**
The nine-field domain-event envelope decided in
[ADR-0017](decisions/0017-event-envelope-and-versioning.md): `type`,
`version`, `venue`, `instrument`, `ts_exchange`, `ts_ingest`, `seq`,
`idempotency_key`, `payload`. Lives in
`internal/shared/contracts/envelope/` once delivered by Onda H-3.
The substrate for replay (ADR-0019), sequencing (ADR-0020), and
multi-venue routing (ADR-0021/0022).

**Wire format**
The serialization codec of a payload on the mesh. Per
[ADR-0018](decisions/0018-protobuf-contract-layer.md), protobuf is
the primary wire format for the 11 streams (with JSON as fallback
during per-stream migration); HTTP-API and control plane stay JSON.
Codec choice is signaled at the envelope level via `content_type`
and is **orthogonal** to schema `version` — a codec migration and a
schema migration are independent concerns.

**Deduplication key**
Every published event carries a deterministic message ID derived from
its content. JetStream uses it to discard duplicates within a window.

**Sequencer**
The component that produces a monotonic `seq` per stream key for
events on the JetStream mesh. Per
[ADR-0020](decisions/0020-sequencing-and-time-normalization.md), the
Sequencer is owned by the single writer of each stream (preserving
ADR-0008), persists state in NATS KV bucket `SEQUENCER_STATE_LATEST`,
and guarantees monotonicity always (density best-effort across
restart). `seq` is the canonical ordering source; consumers MUST NOT
order by `ts_exchange` or `ts_ingest`.

**Stream key**
The tuple `(venue, instrument, event_type)` that the Sequencer
(ADR-0020) keys monotonic counters by. Each combination has its
own independent `seq` space; `seq(n+1) > seq(n)` holds within a key,
never across keys. Cross-key ordering (e.g., cross-venue snapshots
in H-9) is consumer-side merge logic, not a Sequencer concern.

---

## Execution

**ExecutionIntent**
The canonical execution request. Carries side, quantity, status, risk
input, fills, correlation/causation IDs, and metadata. Lives in
`internal/domain/execution/`.

**FillRecord**
A single fill event within an execution. Carries price, quantity, fee,
fee asset, cost basis, fee source, and a `Simulated` flag.

**FeeSource**
The provenance of fee data in a fill. One of:
- `venue` — real commission from the exchange.
- `unavailable` — venue API did not return commission (e.g., Binance
  Futures RESULT response). Fee=0 is expected, not a gap.
- `simulated` — paper/dry-run fill, no real fee.
- `fallback` — venue fill where the fills array was unexpectedly empty.

**Paper / dry-run**
Execution mode where orders are accepted and "filled" by the
`PaperVenueAdapter` without contacting any exchange. All fills marked
`Simulated=true`. The default safe mode.

**Testnet**
Exchange test environment (e.g., Binance Testnet). Real WebSocket
data, real order plumbing, but no real money. Configured via
`execute-mainnet-dry-run.jsonc` and equivalents.

**Mainnet / live**
Exchange production environment. Real orders, real money, real risk.
Requires explicit configuration and credentials.

---

## Read paths

**Operational read**
A latest-value query served by `store` over NATS request/reply.
Bounded latency, single source of truth for "what is the current
state of X".

**Analytical read**
A historical query served by `writer`'s read adapter against ClickHouse.
Used for time-range queries, aggregations, and explainability.

**Read model**
A KV projection in NATS, maintained by `store`. One bucket per family
per partition (typically `{source}.{symbol}.{timeframe}`).

---

## Lifecycle

**Effectiveness**
The win/loss/breakeven/unresolved classification of a decision chain,
based on observed P&L from fills. Computed read-side; no new
ClickHouse tables.

**Pairing**
The FIFO matching of entry and exit legs within a session, producing
round-trip P&L attribution.

**Session**
A bounded execution window with a start, a list of intents/fills, and
a close. Sessions exist to scope continuity and pairing.

**Session close**
The deterministic transition that finalizes a session. Includes
in-flight order surfacing, reconciliation, and carryover handling.

---

## Tooling

**raccoon-cli**
The Rust CLI that enforces architecture rules statically. Reads files,
configs, and source; runs subprocesses only for bounded support
checks. Never imported by Go code. Provides `check`, `inspect`, and
`change` command families.

**arch-guard**
A `raccoon-cli` rule set that fails if any import violates layer
sovereignty.

**quality-gate**
The consolidated CLI guard rail behind `make check*`. Three profiles:
`fast` (default), `ci`, `deep`.

**Smoke**
A `make smoke*` target that exercises an end-to-end path with a real
stack (compose up + seed + probes). The canonical operational
proof-of-record.

**Proto schema**
A schema defined using Protocol Buffers v3, living under
`proto/<family>/v<n>/<name>.proto`. Each schema is registered in
`proto/registry.json` and generates Go types under
`internal/shared/contracts/<family>/v<n>/` (the boundary established
by [ADR-0018](decisions/0018-protobuf-contract-layer.md); generated
Go is tracked in Onda H-3.b, gitignored during H-3.a).

**buf**
The Protocol Buffers tooling suite — `buf lint`, `buf generate`,
`buf breaking` — driven by `proto/buf.yaml` and `proto/buf.gen.yaml`.
See [buf.build](https://buf.build) for upstream docs. Foundry pins
buf ≥ 1.50.0 (`make bootstrap` validates); current locally-validated
version is 1.68.4. Introduced in Onda H-3.a (Fase Wire). The three
Makefile entrypoints are `make proto-lint`, `make proto-gen`,
`make proto-breaking`.

**Schema registry**
The canonical inventory of proto schemas, kept at
`proto/registry.json`. Each entry maps `(type, version)` to its
`.proto` file path and target Go message symbol. The registry is the
source-of-truth that links the envelope's `(type, version)` pair
([ADR-0017](decisions/0017-event-envelope-and-versioning.md)) to a
concrete decoder. Static validation that registry ↔ `.proto` ↔
generated Go stays in sync is the raccoon-cli `check proto`
analyzer's job (delivered in Onda H-3.b per
[ADR-0018](decisions/0018-protobuf-contract-layer.md) acceptance
criterion 5).

**Converter**
A function (or pair of functions) that translates between proto-
generated types under `internal/shared/contracts/` and foundry-
native domain types. Lives in the contracts package per
[ADR-0018](decisions/0018-protobuf-contract-layer.md) boundary;
isolates proto-runtime noise from consumer code. The first
foundry converter is
`internal/shared/contracts/envelope/v1/converter.go` shipped in
Onda H-3.b: `ToProto(CanonicalEvent) (*Envelope, error)` and
`FromProto(*Envelope) (CanonicalEvent, error)`, where
`CanonicalEvent` is the foundry-native domain projection of the
canonical event envelope ([ADR-0017](decisions/0017-event-envelope-and-versioning.md)).
Both directions perform explicit required-field validation; the
defence-in-depth FromProto check exists because proto3 does not
enforce required-fields at the wire level.

**Schema status (`registry.json`)**
Classification of the evolutionary state of a proto schema,
**independent of the status of the ADR that governs it**. Values:

- `draft` — schema may change while the governing ADR is `Proposed`
  or while the schema has no runtime consumer.
- `stable` — schema has at least one runtime consumer and breaking
  changes require a version bump per
  [ADR-0018](decisions/0018-protobuf-contract-layer.md) PROTO-G2.
- `deprecated` — schema has been superseded by a newer version;
  consumers should migrate.

The status of a schema and the status of its governing ADR evolve
on independent timelines: a schema can be `draft` even after its
ADR is `Accepted` (no runtime consumer yet), and an ADR can be
`Proposed` while its schemas are `draft` (the H-3.a state for
envelope v1 and marketdata.trade v1).

---

## Other

**Boot panic**
A panic raised at startup, typically during route or actor
registration. Caught by `cmd/gateway/boot_test.go` (added in P0.6)
for the gateway router specifically.

**ADR**
Architecture Decision Record. A short document under
[`decisions/`](decisions/README.md) capturing one durable design
choice, its context, and consequences.
