# ADR 0017: Event envelope and versioning

## Status

Proposed. Foundation ADR delivered in Onda H-2 of the Fase Harvest;
promoted to `Accepted` when Onda H-3 ships the implementing code
(see "Promoção para Accepted" below).

## Date

2026-05-24.

## Context

The 11 JetStream streams of market-foundry today carry payloads
wrapped by `internal/shared/envelope/Envelope[T]`. That envelope is
the **transport envelope**: it carries `Kind` (command / event /
request / reply), `ID`, `CorrelationID`, `CausationID`, `Subject`,
`ContentType`, and a generic `Payload` — a complete and useful
contract for in-cluster RPC and event transport.

What that envelope does **not** carry, and what subsequent ondas of
the Harvest require, is **domain-event context for mesh events**:

- A stable **schema version** field separated from `ContentType`, so
  consumers can negotiate backward compatibility on payload shape
  rather than on serialization codec.
- A **venue** identifier so cross-venue capabilities (insights,
  arbitrage tracking) can route, aggregate, and partition events
  without parsing the payload.
- A **canonical instrument** reference so the same field locates a
  Binance Spot BTC/USDT trade and a Hyperliquid Perpetual BTC/USDT
  trade in identical structural terms (per ADR-0021).
- A **dual timestamp** distinguishing the exchange-provided time
  (`ts_exchange`, often unreliable) from the local ingest time
  (`ts_ingest`, monotonic per binary). The current single `Timestamp`
  field forces consumers to guess which they need.
- A **monotonic sequence** (`seq`) per stream key, produced by a
  sequencer (ADR-0020), establishing canonical ordering independent
  of either timestamp.
- An **idempotency key** that survives transport-level redelivery
  and consumer retries, so deduplication does not depend on
  `ID` (transport-level identifier, regenerated on republication).

These fields are the foundation for capabilities that follow:

- Deterministic replay (H-4, ADR-0019) requires `seq` + `ts_ingest`
  + `idempotency_key` to reproduce ordering byte-stably.
- Protobuf wire layer (H-3, ADR-0018) requires `version` decoupled
  from `content_type` so a schema migration is a separate concern
  from a codec migration.
- Multi-venue normalization (H-6/H-7, ADR-0021/ADR-0022) requires
  `venue` + `instrument` at the envelope level so cross-venue
  consumers do not need to inspect heterogeneous payloads.
- Insights (H-8/H-9) and writer projections (H-3+) require
  `idempotency_key` for at-least-once delivery with deduplication.

Adding these fields to a future envelope **before** any of the above
ondas implement them is the cheapest moment to decide the shape; ADR
[0013](0013-pause-and-report-protocol.md) and Fase Harvest P3
("capacidade portada passa por documento primeiro") apply.

## Decision

market-foundry adopts a **canonical event envelope** for every
domain event published to or consumed from the 11 JetStream streams.
The envelope carries the following **nine canonical fields**:

| # | Field             | Kind     | Required | Purpose                                                                  |
|---|-------------------|----------|----------|--------------------------------------------------------------------------|
| 1 | `type`            | string   | yes      | Stable event type (e.g., `observation.trade`, `evidence.candle`)         |
| 2 | `version`         | int32    | yes      | Schema version of `payload`; breaking changes increment                  |
| 3 | `venue`           | string   | yes      | Canonical venue id (`binance`, `binancef`, `bybit`, ... per ADR-0021)    |
| 4 | `instrument`      | message  | yes      | Canonical instrument reference (per ADR-0021)                            |
| 5 | `ts_exchange`     | int64?   | no       | Exchange-provided timestamp in epoch nanoseconds (absent when unknown)   |
| 6 | `ts_ingest`       | int64    | yes      | Local ingest timestamp in epoch nanoseconds (monotonic per binary)       |
| 7 | `seq`             | int64    | yes      | Monotonic sequence per stream key produced by the Sequencer (ADR-0020)   |
| 8 | `idempotency_key` | string   | yes      | Stable deduplication key, derived from `(venue, instrument, seq, type)`  |
| 9 | `payload`         | bytes    | yes      | Versioned payload, encoded per the wire format of ADR-0018               |

### Where it lives

The canonical event envelope is defined in
`internal/shared/contracts/envelope/` (the new boundary introduced
by ADR-0018). The existing `internal/shared/envelope/` package
remains the **transport envelope** (commands, requests, replies);
this ADR does not retire it.

For a domain event flowing through one of the 11 streams, the
canonical event envelope is the **payload contract**. Whether the
transport envelope wraps the event envelope, or the event envelope
fully supersedes the transport envelope for event-class messages,
is an H-3 implementation decision (the ADR does not over-specify
layering; it specifies the event contract).

### Versioning rules

- **Schema versioning is independent of wire codec.** A consumer that
  understands `(type=observation.trade, version=2)` can decode it
  whether it arrives as protobuf (ADR-0018) or JSON-fallback.
- **Breaking changes increment `version`.** A breaking change is any
  payload shape change that prior consumers cannot decode (field
  removal, field-type change, semantic redefinition).
- **Additive changes do not require version increments.** Optional
  new fields are backward-compatible by protobuf semantics (per
  ADR-0018) and by JSON-fallback semantics (unknown fields ignored).
- **Consumers MUST support N and N-1** during the active migration
  window of an event type. Older versions may be deprecated and
  removed only after the producer has fully cut over and the
  migration window closes.
- **`type` names are stable.** A renamed event type is a new event
  type; the old type is deprecated with a sunset window, never
  silently retargeted.

### Backward compatibility with the legacy envelope

- The legacy `internal/shared/envelope/Envelope[T]` is **not retired
  by this ADR.** It continues to serve as the transport envelope
  for in-cluster RPC (commands, requests, replies, control-plane
  traffic).
- For domain events on the 11 streams, the migration to the
  canonical event envelope is **per-stream and additive**: H-3
  defines the migration shape (e.g., dual-publish during cutover,
  consumer feature flag) on a stream-by-stream basis.
- The legacy envelope is considered retired only after **all 11
  streams** have migrated; the retirement is recorded in a future
  Onda's ADR (likely H-3.X) once the cutover is verified empirically.

### Idempotency key derivation

The recommended derivation is
`hash(venue, instrument, type, seq)`; a producer that owns a stream
key (per ADR-0008 single-writer) controls `seq` monotonically, so
the derived key is stable across redeliveries of the same event.
Concrete hash algorithm and encoding are left to H-3.

## Non-goals

- **Wire format.** Whether the envelope serializes as protobuf or
  JSON is the subject of ADR-0018, not this ADR.
- **Sequencer implementation.** The mechanics of `seq` generation
  (per-stream-key counter, persistence, restart recovery) are the
  subject of ADR-0020.
- **Canonical instrument model.** The structure of the `instrument`
  field is the subject of ADR-0021.
- **Subject taxonomy.** Envelope fields and NATS subject hierarchy
  are orthogonal; ADR-0009 governs the subject side.
- **HTTP-API surface.** The HTTP-API uses its own JSON contract; the
  canonical event envelope governs mesh events only.
- **Client transport.** The Odin client (H-12+) consumes events via
  a delivery surface (H-11) that may transform the envelope; client
  shape is mapped, not decided here.

## Alternatives considered

- **(A) JSON-only envelope without version field.** Status quo at
  the moment of this ADR. Rejected: makes payload migration
  impossible to coordinate; consumers cannot negotiate compatibility
  without parsing payload internals.
- **(B) Version inside payload only, no envelope `version` field.**
  Rejected: forces consumers to fully decode payload to discover
  whether they can decode payload — circular. Also defeats per-event
  routing decisions (e.g., "skip events with version > N until
  upgrade").
- **(C) Avro schema registry instead of explicit `(type, version)`
  envelope fields.** Rejected: adds an external service dependency
  for hot-path decoding; foundry's preference (per ADR-0018) is a
  manifest file (`proto/registry.json`) tracked in-repo, which
  composes better with the in-cluster mesh model.
- **(D) Reuse the existing transport envelope `ContentType` for
  version negotiation.** Rejected: conflates schema version with
  serialization codec; a codec migration (JSON → proto) becomes
  indistinguishable from a payload schema migration.
- **(E) Combine `ts_exchange` and `ts_ingest` into a single
  `timestamp` field.** Status quo at the moment of this ADR. Rejected:
  loses the only signal that distinguishes exchange clock skew from
  local-time ordering (per ADR-0020), which is precisely the signal
  ADR-0019 (replay) needs.

## Consequences

### Positive

- **Replay is grounded.** `seq` + `ts_ingest` + `idempotency_key`
  give H-4 the byte-stable inputs it needs to deterministically
  reproduce a stream.
- **Cross-venue capability is unblocked.** Insights (H-8) can group
  by `(venue, instrument, type)` without payload parsing.
- **Migration is decoupled from codec migration.** A consumer that
  understands version 2 of an event type can receive it via either
  wire codec (per ADR-0018) without ADR-0017 changing.
- **Deduplication is mechanical.** `idempotency_key` survives
  redelivery; downstream writers (ADR-0003 ClickHouse, KV
  projections) get an authoritative key for upsert/dedup.
- **Audit trail is uniform.** Every event carries
  `(type, version, venue, instrument, seq, ts_ingest)` — sufficient
  for forensic reconstruction without joining external context.

### Negative

- **Envelope overhead.** Each event grows by ~8 fields of metadata.
  Mitigated by ADR-0018's protobuf encoding (~40-60% smaller than
  JSON) at the point both ship.
- **Producer discipline required.** Producers must source `seq`
  from the Sequencer (ADR-0020), construct stable
  `idempotency_key`, and not regenerate them on retry. Compliance
  is enforced by future analyzer work (P5 of the Fase Harvest), not
  by this ADR alone.
- **Coexistence period.** Until all 11 streams migrate, the
  codebase carries two envelope shapes. Cognitive overhead is
  bounded by clear per-stream cutover ownership in H-3.
- **Schema-evolution discipline required.** `version` increments
  must be reviewed against the N-1 compatibility rule; a producer
  that increments `version` without a migration window for
  consumers breaks the contract.

## Promoção para Accepted

This ADR reaches `Accepted` when **all** of the following are
delivered as tracked code in the foundry:

1. `proto/envelope/v1/envelope.proto` defines the canonical envelope
   with the nine fields specified above (delivered by **Onda H-3.a**).
2. `proto/registry.json` includes an entry for `envelope.v1`
   (delivered by **Onda H-3.a**).
3. `internal/shared/contracts/envelope/v1/envelope.pb.go` exists,
   is generated from the `.proto`, and has a unit test validating
   round-trip serialize/deserialize (delivered by **Onda H-3.b**).
4. `internal/shared/contracts/envelope/v1/converter.go` (or
   equivalent) translates between the proto envelope and the
   foundry's domain types, with a unit test (delivered by
   **Onda H-3.b**).

Runtime adoption — migrating the 11 streams from the legacy JSON
envelope (`internal/shared/envelope/`) to this canonical envelope
— is **execution of this architectural decision** and occurs in a
future phase (likely a new phase of PROGRAM-0002 or a successor
PROGRAM-0003+). Migration is **not** an acceptance criterion of
this ADR; the canonical-envelope decision is accepted when the
proto contract plus the corresponding Go types exist and are
validated in code.

H-3.b is responsible for flipping the `Status` field of this ADR
to `Accepted` in the same commit that lands criteria 3 and 4
(criteria 1 and 2 are prerequisites delivered earlier in H-3.a).

## References

- ADR [0008](0008-single-writer-invariant.md) — single-writer per
  stream is a precondition: a single owner of `seq` per stream key
  is what makes `idempotency_key` stable.
- ADR [0009](0009-subject-taxonomy.md) — subject taxonomy is
  orthogonal; envelope fields and subject hierarchy coexist without
  contradiction.
- ADR [0018](0018-protobuf-contract-layer.md) — wire format for the
  envelope and its payload; the `version` field decouples schema
  version from `content_type`.
- ADR [0019](0019-deterministic-replay-time-invariants.md) — the
  consumer of `seq` / `ts_ingest` / `idempotency_key` as
  determinism inputs.
- ADR [0020](0020-sequencing-and-time-normalization.md) — producer
  of `seq` and the stream-key definition used to derive
  `idempotency_key`.
- ADR [0021](0021-canonical-instrument-and-venue-model.md) —
  structure of the `instrument` and `venue` fields.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro) and P7 (sem
  perda de disciplina documental).
- [PROGRAM-0001](../programs/PROGRAM-0001-foundation.md) — Onda H-2
  scope.
- `internal/shared/envelope/envelope.go` — the transport envelope
  this ADR coexists with.
- raccoon `docs/adrs/ADR-0002-event-envelope-and-versioning.md` —
  inspiração. Foundry diverges by **separating envelope from wire
  format**: raccoon's ADR-0002 carries a 2026-02-12 Amendment that
  decides protobuf-vs-JSON via `content_type`; foundry splits that
  decision into ADR-0018 (wire) so envelope versioning and codec
  negotiation are orthogonal. Foundry also defers the `meta`
  reserved-field block (raccoon Amendment) to H-3 implementation
  rather than ADR-level standardization.

## Changelog

- **2026-05-24** — ADR-0017 created (Onda H-2, status `Proposed`).
  See PR #21.
- **2026-05-25** — **Erratum**: section "Promoção para Accepted"
  rewritten to separate the architectural decision from rollout
  execution. Stream migration is no longer an acceptance criterion;
  it is now a future execution phase. Acceptance now requires only
  that the proto schema (H-3.a), registry entry (H-3.a), generated
  Go types with round-trip test (H-3.b), and converter with test
  (H-3.b) all exist as tracked code. Reason: layer separation —
  ADRs codify architectural decisions; PRDs (and successor
  programs) codify execution of those decisions. Lands as commit
  0 of the H-3.a PR.
