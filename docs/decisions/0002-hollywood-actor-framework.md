# ADR 0002: Hollywood as sole concurrency primitive

## Status

Accepted.

## Context

market-foundry needed a model for concurrent processing:
- per-symbol parallel evidence sampling
- per-type signal/decision/strategy/risk evaluators
- venue interaction with retry, timeout, backoff
- streaming consumer loops with backpressure
- session lifecycle management with deterministic close

The candidates:

- **Raw goroutines + channels** — Go's idiom, no framework, full control.
- **Hollywood actor framework** — Go-native actor system with
  supervision trees, message passing, lifecycle hooks.
- **Other Go actor frameworks** (e.g., proto.actor) — alternatives to Hollywood.
- **Async iterators / fan-out patterns** — structured concurrency
  helpers built on top of goroutines.

The system's requirements:

- Hundreds of concurrent processing units (one per
  signal-type × source × symbol × timeframe combination).
- Deterministic lifecycle: spawn, restart on panic, supervised shutdown.
- Easy reasoning about ownership: who created this goroutine, who
  controls its lifetime.

Raw goroutines are easy to spawn but hard to manage at this scale.
Unsupervised goroutines become very hard to debug when they leak or
deadlock.

## Decision

**Hollywood is the sole concurrency primitive for long-lived work**.
No unsupervised goroutines in service code. Every long-running
activity has a supervisor and a defined lifecycle.

Goroutines for short-lived, scoped work (e.g., a single HTTP request
handler) are still allowed — Hollywood is for **long-lived** activities,
not for every parallel computation.

## Consequences

### Positive

- **Supervision trees**: panics are isolated to actor scope.
  A misbehaving signal evaluator doesn't crash the whole derive
  binary; it restarts under its supervisor.
- **Declarative spawning**: per-type processors are registered
  declaratively (`FamilyProcessor` pattern) and the supervisor
  spawns instances. No imperative spawning code spread across
  the codebase.
- **Ownership clarity**: every actor has exactly one supervisor.
  Easy to trace "who owns this".
- **Lifecycle hooks**: PreStart, PostStop, OnRestart are standard.
  No bespoke initialization patterns per spawned unit.

### Negative

- **Learning curve**: developers new to actor systems take time to
  internalize the pattern.
- **Message passing overhead**: passing messages through actor
  mailboxes has some overhead vs direct function calls.
- **Type safety with messages**: Hollywood uses `interface{}` for
  messages by default. The system uses typed message structures
  with explicit dispatch to recover safety, but this is convention,
  not enforced by Hollywood itself.
- **Debugging actor state**: actor state is encapsulated; observing
  it requires explicit instrumentation.

## Alternatives considered

**Raw goroutines + channels**: rejected for the scale concern. Managing
hundreds of long-lived goroutines with consistent lifecycle, supervision,
and restart semantics requires building most of an actor framework
from scratch.

**proto.actor**: similar capabilities to Hollywood. Hollywood was
chosen for its Go-native feel and active development. Either would
have worked.

**Async iterators**: do not provide supervision or lifecycle
management; suitable for transformation pipelines but not for
long-lived processing units.

## References

- `internal/actors/` — actor implementations (common, scopes, registry)
- `internal/actors/common/{entrypoint,lifecycle,engine}.go` — Hollywood integration
- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Foundational principles" → "Actors own lifecycle"
- Hollywood: https://github.com/anthdm/hollywood
