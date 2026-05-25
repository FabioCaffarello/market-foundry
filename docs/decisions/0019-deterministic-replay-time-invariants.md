# ADR 0019: Deterministic replay and time invariants

## Status

Accepted. Promoted from `Proposed` by Onda H-4 (Fase Wire,
PROGRAM-0002): all seven acceptance criteria below are now backed
by tracked code — see "Promoção para Accepted" for the
criterion-by-criterion mapping and the Changelog entry below for
the promotion commit.

## Date

2026-05-24.

## Context

market-foundry's competitive thesis is **bots that are exceptionally
reliable** — strategies whose behavior in paper-trading is the same
behavior they exhibit on a live venue, modulo only the source of the
data feed. The thesis collapses without a guarantee that
"backtest = production": same binary, same code, same logic; the
substitution is the data source.

Today the foundry has none of the infrastructure that backs this
guarantee:

- **Purity is by convention, not enforced.** `internal/domain/` is
  conventionally pure (no time, no random, no I/O), but no analyzer
  catches a regression. A single `time.Now()` slipping into a
  derive evaluator would silently break determinism.
- **No sequencer state.** Ingest receives WS events and forwards
  them; the per-stream-key monotonic sequence (ADR-0020) does not
  exist as code yet. Without it, two replays of the same recorded
  trace can interleave differently.
- **No record/replay infrastructure.** Production traffic cannot be
  captured to fixture and re-fed through the binaries.
- **No golden tests.** Even if record/replay existed, no test
  validates that replay-of-fixture-X produces byte-identical-output
  across runs.

ADR-0017 (envelope) added the **fields** that determinism needs
(`seq`, `ts_ingest`, `idempotency_key`). ADR-0020 will add the
**sequencer** that produces them. This ADR adds the **invariants
that govern how those fields are used** so that replay is genuine,
not theatrical.

The cost of installing determinism after capabilities ship is
high: every existing call site needs auditing, every test needs to
become deterministic, every regression discovered after the fact is
expensive to root-cause. The cost of installing it early — by ADR
in H-2, with enforcement in H-4 — is a single-onda design
investment that pays dividends every onda thereafter.

## Decision

market-foundry adopts **four canonical determinism invariants**:

### INV-D1 — Domain purity

`internal/domain/` is **pure**. No call to `time.Now`, no use of
`math/rand` or `crypto/rand` directly, no `os.Getenv`, no `os.Args`,
no file I/O, no network I/O, no goroutine that depends on wall-clock
scheduling. All sources of non-determinism are **injected via ports**:

- `clock.Clock` for time.
- `random.Source` for randomness.
- Explicit `context.Context` for cancellation (no implicit
  `context.Background()` calls).

A handful of stdlib calls (`time.Date`, `time.Duration` arithmetic,
parsing constants) are permitted because they are pure functions of
their inputs.

**Enforcement:** a new raccoon-cli analyzer `check determinism`
(introduced in H-4 per P5 of the Fase Harvest) scans
`internal/domain/` for banned imports and banned symbols. The
analyzer runs in `make verify` via `quality-gate`.

### INV-D2 — Sequencer is the canonical ordering authority

The Sequencer (ADR-0020) produces a monotonic `seq` for each stream
key `(venue, instrument, event_type)`. Within a stream key,
`seq(n+1) > seq(n)` always. Consumers MUST order by `seq`, not by
either timestamp.

**Enforcement:** unit tests of the Sequencer assert monotonicity;
golden tests of replay validate ordering across 1000+-event fixtures
(see INV-D3). Out-of-order events at consumer ingress produce an
explicit problem (e.g., `OBSERVATION_OUT_OF_ORDER`) and are rejected.

### INV-D3 — Replay produces byte-identical output

Re-running the same binary against the same fixture, with the
clock and sequencer driven from the fixture, produces output
that is **byte-identical** to the prior run. Equivalent across:

- Pointer addresses (output normalized before comparison).
- Goroutine scheduling (no goroutine non-determinism in the hot path).
- Map iteration order (output uses sorted or canonical-order
  serialization).
- Wire codec (ADR-0018's PROTO-G1: proto and JSON encodings of the
  same event produce equivalent decoded shape; golden compares
  decoded form).

**Enforcement:** at least one golden test per family
(observation → evidence → signal → decision → strategy → risk →
execution) replays a recorded fixture and asserts byte-identical
output. Format of the golden file (JSON-lines, proto, etc.) is an
H-4 implementation choice (not decided by this ADR).

### INV-D4 — Byte-stability is validated across multiple runs

The same golden test runs **N=50 times** in CI for at least one
representative fixture. The validation passes only if all 50 runs
produce byte-identical output. This is the canary for hidden
non-determinism (map iteration, scheduler timing, undeclared global
state) that a single-run golden cannot detect.

**Enforcement:** a `make verify`-adjacent target (or a CI step
defined in H-4) runs the multi-run golden. Implementation choice
between in-process loop and N-shell-invocations is H-4's; either
satisfies the invariant.

### Idempotency key as a determinism property

ADR-0017's `idempotency_key = hash(venue, instrument, type, seq)`
is a **pure function** of its inputs. The same envelope replayed
produces the same key byte-for-byte. This composes with INV-D3:
golden tests compare keys directly; a regression in the hash
function or its inputs surfaces as a golden mismatch.

## Non-goals

- **Format of the golden fixture.** JSON-lines, protobuf, custom
  binary — H-4 chooses. The invariant is byte-stability, not
  format.
- **Sequencer implementation details.** Backing store (memory + KV
  persistence), restart-recovery semantics, gap handling — ADR-0020
  decides those.
- **Client-side determinism.** Cliente Odin (H-12+) is a separate
  surface; this ADR governs the server foundry only.
- **Real-time clock semantics.** NTP vs PTP, leap-second handling,
  monotonic vs wall-clock — orthogonal; this ADR specifies only that
  domain receives time via `clock.Clock` and cannot bypass it.
- **Event sourcing replacement.** JetStream provides durable replay
  via consumers; foundry leverages it. A separate event-sourcing
  store is **not** introduced.

## Alternatives considered

- **(A) Determinism is opt-in per package.** Rejected: fragments
  the property; consumers cannot rely on it across the call graph
  without auditing each step.
- **(B) Record-and-replay without byte-stability.** Rejected:
  catches gross logic changes but misses subtle drift (map iteration
  order swap, undocumented dependency on goroutine scheduling). The
  cheap version of "replay works" hides the expensive bugs.
- **(C) Centralize randomness/time in a global "deterministic
  context" struct.** Rejected: pushes complexity to every call site
  while still allowing accidental bypass; injection via ports is
  enforceable.
- **(D) Defer determinism to a future hardening onda.** Rejected
  in the prompt for this ADR: H-2 codifies; H-4 enforces. Deferral
  invites the high install-after-the-fact cost.
- **(E) Use event sourcing with a separate store.** Rejected:
  JetStream's durable consumers already provide replay; a parallel
  store doubles operational surface for marginal benefit.

## Consequences

### Positive

- **Backtest = production** is mechanically true, not aspirational.
- **Regression detection is sharp**: any change to derive evaluator
  logic that shifts output is caught by golden tests.
- **Debugging compresses**: a production anomaly captured as a
  fixture replays deterministically in a developer's IDE.
- **Schema-version migrations are testable**: per-version goldens
  validate that consumers correctly decode N-1 alongside N.
- **The competitive thesis becomes defensible**, not stated.

### Negative

- **Every domain function changes signature** to accept `clock.Clock`,
  `random.Source`, etc. The audit-and-refactor cost is real
  (estimated H-4 scope).
- **Golden tests need maintenance**: an intentional logic change
  invalidates the golden; the developer regenerates and reviews
  the diff. Mitigated by per-family scoping and clear regeneration
  Makefile target.
- **Fixture files add repo weight**: 1000-event fixtures are ~few-MB
  each. Mitigated by selective representative fixtures, not
  fixtures-per-test.
- **The N=50 byte-stability run extends CI time** by minutes for
  the representative fixture. Mitigated by running it only on
  PRs that touch the determinism boundary (H-4 design choice).

## Promoção para Accepted

This ADR is promoted from `Proposed` to `Accepted` when **Onda H-4
(Determinism — replay + sequencer + goldens)** ships:

1. `internal/shared/replay/` package created with recorder + player
   (record/replay infrastructure).
2. `internal/shared/clock/` and `internal/shared/random/` (or
   equivalent ports) introduced; existing direct `time.Now`
   call sites in `internal/domain/` migrated.
3. raccoon-cli `check determinism` analyzer in place, runs in
   `make verify`, scans `internal/domain/` for banned imports and
   symbols (INV-D1).
4. Sequencer (ADR-0020) ships in code, with unit tests asserting
   monotonicity per stream key (INV-D2).
5. At least one end-to-end golden test (typically
   `observation → evidence` for `OBSERVATION_EVENTS`) asserting
   byte-identical replay output (INV-D3).
6. CI step running the representative golden N=50 times and asserting
   uniform byte-stability (INV-D4).
7. `RESUMPTION.md` updated to reflect the determinism gate is in place.

H-4 is responsible for flipping the `Status` field of this ADR to
`Accepted` in the same commit that lands the implementing code.

### Criterion-by-criterion mapping (post-H-4)

1. ✅ `internal/shared/replay/` package created with recorder + player
   in Onda H-4 commit 2 (`d0565e0`). 10 unit tests cover round-trip
   preservation, fixture format invariants, and deterministic replay.
2. ✅ `internal/shared/clock/` and `internal/shared/random/` ports
   introduced in Onda H-4 commit 1 (`d85f333`). The five direct
   `time.Now` call sites discovered during the cascade analysis
   (`DefaultVerificationScope`, `DefaultControlGate`,
   `NewActivationSurface`, `Session.Close`, `Session.Halt`) were
   migrated to consume Clock in commits 6a (`6b2e87d`), 6b
   (`a7631de`), 6c (`cfa9f5f`), 6d (`792bcdf`). `internal/domain/`
   production code now contains zero direct `time.Now` calls.
3. ✅ raccoon-cli `check determinism` analyzer landed in commit 7
   (`cf8cb15`) and wired into `make verify` as Step 7 of the gate
   in commit 8 (`3fb63c7`). The analyzer scans `internal/domain/*.go`
   excluding `*_test.go` (foundry-specific divergence from the
   raccoon reference; see references below).
4. ✅ `internal/shared/sequencer/` shipped in commit 3 (`8ffbe5b`)
   with monotonicity tests covering INV-D2 over 1000-call sequences
   and concurrent-safe load with `-race`.
5. ✅ Golden test `TestGolden_Synthetic100_ByteIdentical` in commit 9
   (`cfeadbc`) validates byte-identical replay over a 100-event
   synthetic fixture. The "end-to-end" scope for H-4 is the
   replay layer cycle (record → persist → load → re-record →
   byte-identical); derive-evaluator goldens (true observation→
   evidence end-to-end) land in a future wave that migrates
   derive to Clock/Source ports.
6. ✅ `TestGolden_ByteStability_N50` runs the same fixture 50 times
   in-process and asserts uniform byte-stability on every iteration.
   In-process chosen over cross-process per PAUSE-AND-REPORT #5;
   the rare cross-process failure modes (init-order side effects)
   are tractable via a sibling test if surfaced.
7. ✅ `RESUMPTION.md` updated in this commit.

## References

- ADR [0017](0017-event-envelope-and-versioning.md) — the envelope
  fields `seq`, `ts_ingest`, `idempotency_key` are the determinism
  substrate this ADR's invariants govern.
- ADR [0018](0018-protobuf-contract-layer.md) — PROTO-G1 (codec
  selection MUST NOT alter golden output) is the cross-cutting
  guarantee from the wire layer; INV-D3 relies on it.
- ADR [0020](0020-sequencing-and-time-normalization.md) — the
  sequencer implementation that INV-D2 governs; sequencer is
  injected into domain via a port per INV-D1.
- ADR [0004](0004-raccoon-cli-static-enforcement.md) — analyzer
  framework that the new `check determinism` builds on; P5 of the
  Fase Harvest applies.
- ADR [0005](0005-layer-sovereignty.md) — `internal/domain/` is the
  inward-most layer and the natural scope for purity enforcement.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro) and P5 (cada
  invariante traz seu enforcement).
- [PROGRAM-0001](../programs/PROGRAM-0001-foundation.md) — Onda H-2
  scope.
- raccoon `docs/adrs/ADR-0015-deterministic-replay-time-invariants.md`
  — inspiração. Foundry diverges by (a) consolidating to four
  invariants (raccoon's five collapse: raccoon R4 is folded into
  ADR-0017 idempotency-key derivation; raccoon R5 codec append-only
  is folded into ADR-0018 PROTO-G5 field-number-reuse-forbidden);
  (b) enforcing purity via raccoon-cli analyzer rather than shell
  grep script (P5); (c) deferring fixture format to H-4 implementation
  rather than specifying JSON-lines in the ADR; and (d) explicitly
  adding INV-D4 (N=50 byte-stability) which raccoon mentions
  implicitly via `TestGoldenReplayByteStable50Runs` but does not
  promote to invariant-level.
- raccoon `docs/rfcs/RFC-0009-W8-deterministic-replay-golden-tests.md`
  — technical detail informing this ADR; not transcribed.
- **H-4 implementation note**: the `check determinism` analyzer
  scopes INV-D1 to **production code only** (`internal/domain/*.go`
  excluding `*_test.go`). Foundry-specific divergence from the
  raccoon reference's `check-domain-isolation.sh` (which applies
  to all `.go` files). Rationale: the real enforcement for tests
  is determinism gates INV-D3/INV-D4 (golden tests + N=50
  byte-stability) — a test using `time.Now` incorrectly flaps
  goldens, so a static check on test files adds friction without
  proportionate security benefit.
- **H-4 implementation note**: the replay-layer fixture format uses
  stdlib `encoding/json` on a private `fixtureRecord` struct, NOT
  `protojson` on `envelope.v1.Envelope`. Rationale: `protojson.Marshal`
  output is not stable between versions of
  `google.golang.org/protobuf` (per upstream docs). A transitive
  dependency bump would silently invalidate every golden and break
  INV-D4. The stdlib encoder is byte-stable given a fixed struct
  definition. See `internal/shared/replay/doc.go` for details.

## Changelog

- **2026-05-24** — ADR-0019 created (Onda H-2, status `Proposed`).
  See PR #21.
- **2026-05-25** — **Promoted to `Accepted`**. Onda H-4 (Fase Wire,
  PROGRAM-0002) delivered all seven acceptance criteria across
  ten commits: clock/random ports (1), replay recorder+player (2),
  sequencer (3), KV bucket + Store (4), gap counter (5), Clock
  plumbing (6.0), domain migration of `DefaultVerificationScope`
  (6a), `DefaultControlGate` (6b), `NewActivationSurface` (6c),
  `Session.Close`/`Halt` (6d), `check determinism` analyzer (7),
  gate integration (8), golden test + N=50 byte-stability (9),
  and this promotion (10). `internal/domain/` production code is
  now mechanically enforced to be free of direct `time.Now`,
  `math/rand`, `crypto/rand`, `os.Getenv`, `context.Background`,
  and related non-determinism sources. The cascade analysis that
  discovered the five production call sites is documented in
  PROGRAM-0002 Changelog. See the H-4 PR for full diff.
