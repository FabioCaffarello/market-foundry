# ADR 0018: Protobuf contract layer

## Status

Proposed. Foundation ADR delivered in Onda H-2 of the Fase Harvest;
promoted to `Accepted` when Onda H-3 ships the implementing code
(see "Promoção para Accepted" below).

## Date

2026-05-24.

## Context

market-foundry today serializes every JetStream payload as JSON. The
existing `internal/shared/envelope/Envelope[T]` is generic over `T`,
encoded with the standard library's `encoding/json`. For the current
scale (paper-trading, single venue family, 11 streams), JSON is
adequate.

Ondas H-3 and beyond will change this calculus:

- **Insights (H-8/H-9)** ship payloads — heatmaps, volume profiles,
  cross-venue snapshots — that grow O(rows × buckets). For a typical
  one-minute heatmap row, JSON encoding is ~3× the size of the
  semantically-equivalent protobuf encoding. At 50k+ events/sec in
  the steady state, payload bytes and serialize CPU become first-order
  cost.
- **Multi-venue (H-6/H-7)** introduces parallel payload shapes per
  venue. Without a typed contract layer, divergence between
  Hyperliquid-trade-shape and Binance-trade-shape is policed by
  convention only. Breaking-change detection requires schema-aware
  tooling, not text diffs.
- **Cliente Odin (H-12+)** consumes events from the foundry via a
  delivery surface (H-11). Generating client code from a single
  source of truth (`.proto`) eliminates the manual-keeping-in-sync
  cost between server and client shape.
- **Versioning** (ADR-0017's `version` field) needs schema-aware
  tooling to verify that an increment is a genuine breaking change
  (and a non-increment is genuinely backward-compatible). `buf
  breaking` is the canonical tool for this; JSON has no equivalent
  with comparable guarantees.

The decision is not whether protobuf is technically appropriate (the
raccoon repository validated this years ago and operates protobuf as
the default wire format in production). The decision is **when and
where** to land it in the foundry's structure such that layer
sovereignty (ADR-0005) and the single-writer invariant (ADR-0008) are
preserved.

## Decision

market-foundry adopts **protobuf as the primary wire format** for
events on the 11 JetStream streams, with **JSON as a fallback** for
the migration window and for non-mesh surfaces. Specifically:

### Wire-format policy

| Surface                   | Primary  | Fallback | Rationale                                                                 |
|---------------------------|----------|----------|---------------------------------------------------------------------------|
| Domain events (11 streams)| Proto    | JSON     | Bandwidth + serialize cost dominate; schema-aware tooling required        |
| HTTP-API (gateway)        | JSON     | —        | External-facing; tooling and human-debuggability favor JSON               |
| Control plane (commands, replies, configctl) | JSON | — | Low volume; human-readable; rarely benefits from proto's compactness    |

Proto vs JSON is signaled at the envelope level via `content_type`
(ADR-0017 keeps `version` orthogonal — codec migration and schema
migration are distinct).

### Boundary

Generated proto types live exclusively in
`internal/shared/contracts/`. This is the **new sovereign boundary**
introduced by this ADR; no other package may import generated
proto code directly.

- **Domain layer (`internal/domain/`) is proto-free.** Domain types
  are native Go structs; they neither import nor return proto types.
- **Converters live in `internal/shared/contracts/<family>/converters.go`.**
  Each family (envelope, observation, evidence, signal, decision,
  strategy, risk, execution, insights, …) ships a Go ↔ proto
  converter that preserves semantic equivalence.
- **Adapters (`internal/adapters/`) import proto only via the
  contracts package**, never via the generated `*_pb.go` directly.

This boundary is enforced statically by a new raccoon-cli analyzer
(`check proto`) introduced in H-3, consistent with P5 of the Fase
Harvest (every architectural invariant ships with its enforcement).

### Inventory authority

`proto/registry.json` is the canonical inventory of schemas. Each
entry maps `(type, version)` to a generated `.proto` file path and
the corresponding Go converter. The registry is the source-of-truth
that links the envelope's `(type, version)` pair (ADR-0017) to a
concrete decoder.

### Tooling

- **buf** is the canonical proto toolchain (`buf lint`,
  `buf generate`, `buf breaking`).
- A new Makefile target `make proto-gate` runs all three; it
  composes into `make verify` via `quality-gate` in H-3.
- `buf breaking` guards backward compatibility against the
  registered baseline; an intentional break requires the schema
  owner to bump `version` in `proto/registry.json` and ship the
  corresponding ADR-0017-compliant migration.

### Migration strategy (per-stream)

H-3 ships the proto layer for **one** stream (likely
`OBSERVATION_EVENTS`, the highest-volume) end-to-end. Subsequent
streams migrate one at a time:

1. **Dual-publish window**: the writer publishes to its stream in
   both proto and JSON encodings, distinguished by `content_type`.
   Consumers opt in to proto by negotiating `content_type` at
   subscription time.
2. **Cutover**: when all consumers of a stream report proto-ready,
   the writer drops the JSON publish.
3. **Cleanup**: JSON-fallback decoders for that stream are removed
   from the contracts package; `proto/registry.json` is updated to
   mark the stream as proto-only.

Per-stream migration is a normal H-N+ deliverable; each onda owns
the stream(s) it touches.

## Non-goals

- **HTTP-API wire format.** The gateway continues to serve JSON;
  this ADR governs mesh events, not external API.
- **CBOR / MessagePack / Avro / FlatBuffers.** Considered and
  rejected below; not entertained as future fallback paths.
- **Schema registry as a separate service.** Foundry uses
  `proto/registry.json` (file in repo) as the inventory authority;
  no Confluent-style runtime registry is introduced.
- **Code generation for languages beyond Go.** Generation for the
  Odin client (TS / WASM bindings) is in scope for H-12+, not this
  ADR.
- **Compression.** Wire-level compression (gzip, zstd) is a separate
  decision; proto's compactness is the first-order win.
- **Replay / determinism semantics.** ADR-0019 governs; this ADR
  guarantees only that **codec selection cannot affect golden output**
  (see PROTO-G1 below).

### Guarantees this ADR makes

| ID         | Guarantee                                                                                  |
|------------|---------------------------------------------------------------------------------------------|
| PROTO-G1   | Codec selection (proto vs JSON) MUST NOT alter golden-test output (ADR-0019 alignment).     |
| PROTO-G2   | `buf breaking` runs against the baseline in CI; a break is a merge blocker unless `version` is incremented and a migration window is declared. |
| PROTO-G3   | `internal/domain/` never imports proto-generated code; enforced by raccoon-cli `check proto`. |
| PROTO-G4   | `proto/registry.json` mirrors live `.proto` files; mismatch is a `make proto-gate` failure. |
| PROTO-G5   | Field-number reuse is forbidden; removed fields become `reserved`.                          |

## Alternatives considered

- **(A) JSON-only permanently.** Rejected: insights and cross-venue
  payloads will saturate bandwidth and serialize CPU at projected
  steady-state; schema-versioning tooling is absent.
- **(B) Avro.** Rejected: ecosystem favors a runtime schema registry
  service; foundry prefers the in-repo manifest model. Avro's
  serialization wins over proto are marginal at this payload shape.
- **(C) MessagePack.** Rejected: compact like proto, but lacks
  schema-evolution tooling at parity with `buf breaking`. The
  evolution guard is the primary value, not the bytes.
- **(D) CBOR.** Rejected for the mesh; raccoon's PRD-0004 considers
  CBOR for WebSocket client delivery. Foundry diverges: ADR-0018
  picks proto everywhere on the mesh (uniformity reduces converter
  surface). The H-11/H-12 delivery decision may revisit CBOR for
  the Odin client wire if proto-on-WS is suboptimal; that is a
  future, surface-specific decision, not this ADR's.
- **(E) FlatBuffers.** Rejected: zero-copy decode is attractive but
  Go tooling is less mature than `buf` and the schema evolution
  story is weaker.
- **(F) Hand-rolled tagged binary format.** Rejected: replicates
  what protobuf already does, without the tooling.

## Consequences

### Positive

- **Smaller payloads on the mesh.** Typical reduction 40–60% versus
  JSON for trade/candle/heatmap shapes (raccoon-validated; foundry
  will re-measure in H-3 to confirm).
- **Schema-aware backward compatibility.** `buf breaking` catches
  breaks at PR time rather than at runtime.
- **Codegen for the client.** Odin (H-12+) gets typed bindings from
  the same source as the server, eliminating manual drift.
- **Versioning is mechanizable.** The `version` field of ADR-0017
  pairs with `buf breaking` to make the rule "increment version on
  break" verifiable.
- **Layer sovereignty is preserved.** The contracts boundary
  isolates proto's noise from the domain layer.

### Negative

- **Tooling overhead.** Contributors need `buf` installed; the
  generated code is committed to keep `go build` self-contained
  (or generated on bootstrap, an H-3 design decision).
- **Debuggability cost.** Proto is binary; live-traffic inspection
  requires `protoc --decode_raw` or proto-aware tooling. Mitigated
  by JSON-fallback during migration and by debug logging that
  decodes payloads.
- **Generated-code churn in diffs.** PRs that touch `.proto` files
  produce noisy diffs in generated code; mitigated by reviewer
  convention (skim `*_pb.go`; focus on `*.proto`).
- **Coexistence period adds surface.** Dual-publish during the
  per-stream migration doubles writer overhead for that stream
  temporarily.

## Promoção para Accepted

This ADR is promoted from `Proposed` to `Accepted` when **Onda H-3
(Wire — proto + envelope skeleton + tooling)** ships:

1. `proto/` tree created with at least `proto/envelope/v1/envelope.proto`
   matching ADR-0017's nine-field shape.
2. `proto/registry.json` populated for the envelope and at least one
   event type (typically `observation.trade`).
3. `buf` toolchain integrated: `make proto-lint`, `make proto-gen`,
   `make proto-breaking` runnable; `make proto-gate` composes them.
4. `internal/shared/contracts/` package shipped with envelope
   converters and at least one event-type converter.
5. raccoon-cli `check proto` analyzer in place; runs in `make verify`
   via `quality-gate`; enforces PROTO-G3 (domain boundary).
6. At least one stream (typically `OBSERVATION_EVENTS`) producing or
   consuming proto end-to-end in the running stack.
7. `RUNTIME.md` and `RESUMPTION.md` updated to reflect the migration
   state.

H-3 is responsible for flipping the `Status` field of this ADR to
`Accepted` in the same commit that lands the implementing code.

## References

- ADR [0017](0017-event-envelope-and-versioning.md) — envelope
  versioning is the prerequisite; this ADR decides the codec.
- ADR [0004](0004-raccoon-cli-static-enforcement.md) — the analyzer
  framework that the new `check proto` builds on; P5 of the Fase
  Harvest applies.
- ADR [0005](0005-layer-sovereignty.md) — the layer model the
  contracts boundary respects.
- ADR [0008](0008-single-writer-invariant.md) — preserved: each
  stream still has one writer, regardless of codec.
- ADR [0019](0019-deterministic-replay-time-invariants.md) — codec
  selection MUST NOT alter replay output (PROTO-G1).
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro) and P5 (cada
  invariante traz seu enforcement).
- [PROGRAM-0001](../programs/PROGRAM-0001-foundation.md) — Onda H-2
  scope.
- `internal/shared/envelope/envelope.go` — the transport envelope
  this ADR's wire-format choice ultimately serializes.
- raccoon `docs/adrs/ADR-0016-protobuf-contract-layer.md` —
  inspiração. Foundry diverges on three counts:
  (a) the proto-free domain boundary is enforced by a raccoon-cli
  analyzer (per P5 of the Fase Harvest), not a shell script;
  (b) ADR-0018 is `Proposed` and is promoted only after H-3 ships
  proto end-to-end on at least one stream (raccoon's ADR-0016
  carries an Implementation Matrix because it documents an already-
  partial implementation; foundry has zero proto today, so the matrix
  is unnecessary);
  (c) raccoon's CBOR consideration (PRD-0004) is acknowledged but
  deferred to the H-11/H-12 client-delivery surface, not folded
  into this ADR.
- raccoon `docs/rfcs/RFC-0007-W6-protobuf-contract-layer.md` —
  technical detail informing this ADR; not transcribed.
