# ADR 0020: Sequencing and time normalization

## Status

Proposed. Foundation ADR delivered in Onda H-2 of the Fase Harvest;
promoted to `Accepted` when Onda H-4 ships the implementing code
(see "Promoção para Accepted" below).

## Date

2026-05-24.

## Context

Exchange-provided timestamps on market-data feeds are systematically
unreliable:

- **Clock skew**: an exchange's clock is allowed to drift relative to
  ours; the drift is unbounded in the contract.
- **Out-of-order delivery**: Binance Spot WebSocket has documented
  reorderings of trade events under load.
- **Lost trailing precision**: some venues advertise millisecond
  timestamps that are actually second-precision padded with zeros;
  others advertise microseconds with stable nanosecond bits.
- **Reconnect duplicates**: WebSocket reconnect windows can replay
  trailing events with new arrival times but stale exchange times.

If aggregation, deduplication, or ordering downstream depends on
`ts_exchange`, every venue contributes a different non-determinism
class to the pipeline. The result is that two replays of the same
fixture produce subtly different orderings (defeating ADR-0019's
INV-D3) and that gap detection becomes guesswork.

The ingest binary today captures `ts_exchange` when the venue
provides it but has no native sequencer of its own. The transport
envelope's `Timestamp` field (the existing
`internal/shared/envelope/`) is set from the producer's wall clock
at envelope-construction time, which conflates two distinct
concepts: when the event happened at the venue (`ts_exchange`) and
when it crossed the foundry's process boundary (`ts_ingest`).

ADR-0017 (envelope) added `seq`, `ts_exchange`, and `ts_ingest` as
separate fields. ADR-0019 (replay) established that consumers MUST
order by `seq`. This ADR specifies **what produces `seq`, how
`ts_*` are sourced, and how gaps are detected** so the two prior
ADRs have a concrete producer to depend on.

## Decision

market-foundry adopts the following **sequencing and time-normalization
contract**:

### Two timestamps, one sequence

Every event carries:

- **`ts_exchange`** — the venue's reported timestamp in epoch
  nanoseconds, or absent (`nil`) when the venue did not provide one.
  **Advisory only.** Consumers MUST NOT use `ts_exchange` for
  ordering, dedup, or windowing.
- **`ts_ingest`** — the local timestamp at the moment the event
  crossed the ingest binary's process boundary, in epoch
  nanoseconds, sourced from the `clock.Clock` port (ADR-0019
  INV-D1). Monotonic per binary process; resets on restart but
  recovers continuity via the sequencer (see "Persistence" below).
  **Authoritative for TTL, telemetry, and operational reasoning.**
- **`seq`** — a monotonic integer per stream key, produced by the
  Sequencer described below. **Authoritative for ordering and
  dedup. The only source of canonical ordering.**

### Stream key definition

The Sequencer maintains one independent monotonic counter per
**stream key**, defined as the triple:

```
stream_key = (venue, instrument, event_type)
```

Examples:
- `(binance, BTC/USDT-spot, observation.trade)`
- `(binancef, BTC/USDT-perp, observation.trade)`
- `(binance, ETH/USDT-spot, observation.book.snapshot)`

Each combination has its own sequence space; `seq(n+1) > seq(n)`
holds within a key, never across keys. Cross-key ordering
(needed for cross-venue snapshots in H-9) is a separate concern
handled at the consumer (typically by `ts_ingest`-merge over
`seq`-ordered per-key streams).

### Producer ownership (single-writer alignment)

By ADR-0008, each NATS stream has exactly one writer binary. The
**Sequencer is owned by that writer for the keys it produces**.
Concretely:

- `ingest` owns `seq` for all keys publishing to `OBSERVATION_EVENTS`.
- `derive` owns `seq` for all keys publishing to evidence / signal
  / decision / strategy / risk / execution streams.
- `execute` owns `seq` for fill / rejection / session streams.
- `configctl` owns `seq` for `CONFIGCTL_EVENTS`.

A second writer producing `seq` for the same stream key would
double-allocate sequence numbers; the single-writer invariant
prevents this by construction.

### Persistence

Sequencer state persists in NATS KV bucket
`SEQUENCER_STATE_LATEST`. Keys follow:

```
seq.{owner_binary}.{venue}.{instrument}.{event_type}
```

The bucket carries the highest issued `seq` per stream key. On
binary start, the Sequencer reads the bucket to recover the
last-issued value and resumes from `last + 1`. On a clean
shutdown the writer flushes pending state before exiting.

KV writes are bounded-batched (frequency and batch size are H-4
implementation choices) so the hot path does not synchronously
write per event.

### Gap detection at consumers

Consumers track the highest `seq` they have observed per stream
key. On receiving an event with `seq > last_observed + 1`, the
consumer:

1. Records the gap (counter:
   `marketfoundry_consumer_seq_gap_total{stream_key}`).
2. Optionally requests a JetStream replay covering the gap range
   (durable consumer semantics; ADR-0001).
3. Continues processing forward; gap-filling is recovery, not a
   blocking error.

Out-of-order delivery (`seq <= last_observed`) is dedup at the
consumer (the `idempotency_key` from ADR-0017 is the canonical
dedup key, but a `seq` regression is also a sufficient signal).

### Recovery semantics

Sequencer state restored from KV after a crash MAY have lost the
last few `seq` values that were not yet flushed to KV. The
recovery contract:

- On restart, the Sequencer reads `last_issued` from KV.
- It resumes at `max(last_issued, observed_consumer_last) + 1`,
  conservatively forward. (Observed consumer high-water marks are
  out of scope for this ADR; H-4 may add them or accept a small
  redundant-emit window.)
- Downstream dedup (via `idempotency_key`) absorbs the redundancy.

The contract guarantees **monotonicity always**, never **density
always** — a sequence can have gaps created intentionally by
recovery, and consumers tolerate them by design.

## Non-goals

- **Clock source.** NTP vs PTP, leap-second handling, monotonic-vs-
  wall-clock — orthogonal to this ADR. The Sequencer reads time
  via `clock.Clock` (ADR-0019 INV-D1). If `clock.Clock` is wrong,
  this ADR's invariants still hold (sequence is monotonic from the
  perspective of `clock.Clock`).
- **Cross-venue cross-stream ordering.** Cross-venue snapshots
  (H-9) require ordering across `(venue_A, instrument_X)` and
  `(venue_B, instrument_X)`. That is consumer-side merge logic;
  this ADR provides per-key monotonic streams as the building
  block.
- **Per-event KV flush.** Performance-critical; H-4 designs the
  batching strategy. ADR contract is "state recoverable", not
  "state persistent per event".
- **Reordering at the producer.** Ingest publishes events in WS
  arrival order; no buffer-and-sort. The Sequencer assigns `seq`
  in publish order, which is the order downstream consumers see.
- **Multiple Sequencer instances per owner binary.** Single
  instance per binary (consistent with single-writer per stream).

## Alternatives considered

- **(A) Order by `ts_exchange`.** Status quo at the moment of this
  ADR. Rejected: unreliable per the failure modes catalogued in
  Context.
- **(B) Centralized Sequencer service.** Rejected: introduces a
  single-point-of-failure in the hot path of every writer; the
  single-writer invariant (ADR-0008) already gives per-stream
  ownership without a central service.
- **(C) Lamport / vector clocks.** Rejected: overkill for the
  consistency model the foundry needs. The single-writer
  invariant per stream means a per-stream monotonic counter is
  sufficient; no causal-history reasoning is required.
- **(D) Use the JetStream message sequence as `seq`.** Rejected:
  JetStream sequences are global per stream, not per stream key.
  Two writers publishing to the same stream (e.g., `derive`
  publishing two evidence types) would interleave their JetStream
  sequences, defeating per-key reasoning. Also, JetStream sequence
  is transport-level; using it for domain semantics couples them.
- **(E) Hash-based stable `seq`** (deterministic from payload
  contents). Rejected: defeats monotonicity (hash space is
  unordered); also enables replay-attack-style ambiguity for
  identical-content events.
- **(F) No persistence; restart from 0.** Rejected: every restart
  invalidates downstream dedup (idempotency keys derive from
  `seq` per ADR-0017); a restart would replay events as "new".

## Consequences

### Positive

- **Ordering is reliable.** Every consumer reasons about `seq`
  without consulting unreliable timestamps.
- **Dedup is mechanical.** ADR-0017's `idempotency_key` derives
  from `seq` and is stable across redelivery.
- **Replay is grounded.** ADR-0019 INV-D3 can hold because `seq`
  is the source of order, and `seq` is deterministic for a given
  fixture.
- **Per-key parallelism is preserved.** Consumers can process
  different stream keys concurrently without coordinating; only
  intra-key order matters.
- **Gap detection is explicit.** Operators see
  `consumer_seq_gap_total` rising and can intervene; gaps are not
  silent.

### Negative

- **New failure mode**: KV outage during sequencer flush →
  potential redundant-emit window on next restart. Mitigated by
  consumer-side dedup via `idempotency_key`.
- **KV restore is sensitive**: a corrupted or rolled-back KV
  bucket would invalidate sequence continuity. Operational
  runbook (H-10 onwards) must document KV backup/restore
  alignment with `SEQUENCER_STATE_LATEST`.
- **Slight ingest hot-path cost**: per-event sequence allocation
  (memory operation) plus batched KV write. Cost dominated by
  network I/O, not by the sequencer itself.
- **Stream-key cardinality scales the KV bucket**: O(venues ×
  instruments × event_types). Bounded in practice (low thousands
  even at H-12+ scale); fits comfortably in KV.

## Promoção para Accepted

This ADR is promoted from `Proposed` to `Accepted` when **Onda H-4
(Determinism — replay + sequencer + goldens)** ships:

1. `internal/shared/sequencer/` package created with the per-key
   monotonic counter and the `clock.Clock`-driven `ts_ingest`
   assignment.
2. NATS KV bucket `SEQUENCER_STATE_LATEST` declared in
   `internal/adapters/nats/` registries with `ingest` as the
   single writer for its owned keys (per ADR-0008).
3. Unit tests asserting INV-D2 (monotonicity within stream key)
   over fixtures with intentional out-of-order WS arrival.
4. Counter `marketfoundry_consumer_seq_gap_total{stream_key}`
   exposed in `internal/shared/metrics/` and incremented at
   consumer ingress when a gap is detected.
5. At least one writer binary (typically `ingest`) using the
   Sequencer in the running stack; goldens (per ADR-0019 INV-D3)
   pass for that writer's downstream chain.
6. `RUNTIME.md` updated with the `SEQUENCER_STATE_LATEST` bucket
   entry; `RESUMPTION.md` updated to reflect the sequencer is in
   place.

H-4 is responsible for flipping the `Status` field of this ADR to
`Accepted` in the same commit that lands the implementing code.

## References

- ADR [0017](0017-event-envelope-and-versioning.md) — defines the
  `seq`, `ts_exchange`, `ts_ingest`, `idempotency_key` envelope
  fields this ADR populates.
- ADR [0019](0019-deterministic-replay-time-invariants.md) —
  INV-D2 (canonical ordering authority) is the consumer-side
  contract this ADR backs; the Sequencer is injected into domain
  via a port per INV-D1.
- ADR [0008](0008-single-writer-invariant.md) — each stream's
  single writer is the natural owner of the Sequencer for its
  keys; double-allocation is prevented by construction.
- ADR [0001](0001-nats-not-kafka.md) — JetStream durable consumers
  back the replay capability that gap detection optionally invokes.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro).
- [PROGRAM-0001](../programs/PROGRAM-0001-foundation.md) — Onda H-2
  scope.
- raccoon `docs/adrs/ADR-0005-sequencing-and-time-normalization.md`
  — inspiração. Foundry diverges by (a) defining `stream_key` as
  the **triple** `(venue, instrument, event_type)` rather than the
  raccoon's pair `(venue, instrument)` — event-type independence
  matters once a single venue+instrument carries trades, books,
  and snapshots in parallel (anticipating H-3+ event-type growth);
  (b) declaring the canonical persistence target
  (`SEQUENCER_STATE_LATEST` NATS KV bucket) at the ADR level rather
  than deferring (raccoon Amendment 2026-02-12 leaves persistence
  open); (c) specifying explicit gap-detection semantics with a
  named Prometheus counter; and (d) explicitly forbidding the
  alternative of using JetStream message sequence as `seq` (raccoon
  Amendment 2026-02-12 mentions two sequence domains but does not
  exclude conflation).
