# ADR 0008: Single-writer invariant per stream and KV bucket

## Status

Accepted.

## Context

In distributed systems, the most pernicious bugs come from concurrent
writers to the same state. Two binaries writing to the same NATS
stream, or two actors updating the same KV key, produce race
conditions that are very hard to reproduce and very hard to debug.

market-foundry has 11 JetStream streams and 16 KV buckets. Each could
in principle be written by multiple binaries. The question is
whether to allow it.

## Decision

**Every JetStream stream has exactly one writer binary. Every NATS
KV bucket has exactly one writer actor. Every NATS query subject has
exactly one server binary.**

Multiple **readers/consumers** are fine and encouraged. Multiple
**writers** are forbidden, with no exception.

Concretely:
- `OBSERVATION_EVENTS` is written only by `ingest`.
- `EVIDENCE_EVENTS`, `SIGNAL_EVENTS`, `DECISION_EVENTS`,
  `STRATEGY_EVENTS`, `RISK_EVENTS`, `EXECUTION_EVENTS` are written
  only by `derive`.
- `EXECUTION_FILL_EVENTS`, `EXECUTION_REJECTION_EVENTS`,
  `SESSION_LIFECYCLE_EVENTS` are written only by `execute`.
- `CONFIGCTL_EVENTS` is written only by `configctl`.
- KV buckets (`CANDLE_LATEST`, `SIGNAL_RSI_LATEST`, etc.) are written
  only by the `store` binary's projection actors, one bucket per actor.

This is enforced by:
- Code structure: only one adapter publishes to each stream.
- Code review: PRs introducing a second writer are rejected.
- Static checks: raccoon-cli flags suspicious patterns where possible.

## Consequences

### Positive

- **Reasoning simplicity**: when looking at a stream or bucket, the
  source of writes is unambiguous. No "who else might be writing this".
- **Race-free updates**: KV monotonicity guards work because only one
  writer is racing with itself, which is solvable, not with arbitrary
  other writers.
- **Auditability**: writes can be traced to a single binary in logs.
- **Fault isolation**: if a writer behaves wrongly, the blast radius
  is bounded to its owned streams/buckets.

### Negative

- **Coordination required for new flows**: introducing a new event
  type requires choosing the right owner binary. Mistakes here create
  awkward refactoring later.
- **No "convenient" cross-binary updates**: if binary X needs to
  update state that binary Y owns, X must publish a message that Y
  consumes. No direct write. This adds indirection.
- **Single point of write means single point of failure**: if the
  writer binary is down, no writes to its streams/buckets.
  Mitigated by the writer being durable and restart-safe.

## Alternatives considered

**Allow multiple writers with optimistic locking**: rejected because
KV in NATS doesn't natively support compare-and-swap (CAS) without
care, and even with CAS, the cognitive load of reasoning about
concurrent writers in a busy system is high.

**Allow multiple writers with last-writer-wins**: rejected because it
silently loses data. Latest-value KV reads would become unreliable.

**Sharding writes across writers by key**: workable but adds complexity
(a key router, partition consistency hashing). Not worth it for the
current scale; the single-writer pattern handles the load.

## References

- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Foundational principles"
  → "Single-writer invariant"
- [`../RUNTIME.md`](../RUNTIME.md) → stream catalog with writer/consumer
  columns
- All `internal/adapters/nats/nats<domain>/registry.go` files show
  the writer per stream
- ADR [0005](0005-layer-sovereignty.md) — layer enforcement that
  prevents accidental cross-binary writes
