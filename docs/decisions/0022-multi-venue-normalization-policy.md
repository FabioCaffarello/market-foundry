# ADR 0022: Multi-venue normalization policy

## Status

Proposed. Foundation ADR delivered in Onda H-2 of the Fase Harvest;
promoted to `Accepted` when Onda H-7 ships the implementing code
(see "Promoção para Accepted" below).

## Date

2026-05-24.

## Context

ADR-0021 establishes the **structural** model for multi-venue:
`CanonicalInstrument`, `Venue` enum, `ContractType` enum, adapter
normalization at the boundary. What ADR-0021 does **not** specify
is the **operational policy** for how adapters handle the fact
that venues do not support the same event-type surface:

- Binance Spot publishes trades and order-book deltas, but not
  mark-price updates.
- Binance USDM Futures publishes mark-price updates (`markPrice`).
- Hyperliquid publishes liquidation events with a venue-specific
  payload field that no other venue carries.
- Coinbase Spot publishes trades and depth, but no funding-rate
  stream (perpetuals don't exist there).
- Kraken Spot vs Kraken Futures differ in supported events.

Without a policy, every multi-venue consumer faces a per-venue
case analysis:

- A heatmap consumer wanting mark-price-aware visualizations
  silently breaks when subscribing to a Coinbase mark-price subject
  that does not exist.
- A consumer wanting "all trades, all venues" works today; tomorrow
  a venue adapter is added that publishes trades on a non-standard
  subject and the consumer silently misses them.
- A developer adding venue N+1 has no automated way to verify they
  declared all event types their venue supports, nor a way to
  validate that subscribers know what to expect.

The cost of ad-hoc policy compounds. By H-9 (cross-venue insights),
four to seven venues will be live; without a declarative
capabilities contract, cross-venue logic becomes a maze of
exception handling.

This ADR introduces the **declarative capabilities contract** that
makes the multi-venue surface introspectable, parity-checkable,
and operationally honest about gaps.

## Decision

market-foundry adopts the following **four-rule multi-venue
normalization policy**:

### R1 — Each adapter publishes a `Capabilities()` declaration

Every adapter under `internal/adapters/exchanges/<venue>/` ships a
`Capabilities()` function returning a structured declaration of
**which event types the adapter supports** for **which contract
types**:

```go
// Capabilities returned by every venue adapter.
type Capabilities struct {
    Venue        instrument.Venue
    EventTypes   []EventTypeSupport    // declared event types
    Contracts    []instrument.ContractType // declared contract families
    Notes        map[string]string     // free-form per-venue annotations
}

type EventTypeSupport struct {
    Type      string               // e.g., "observation.trade", "observation.markprice"
    Contracts []ContractType        // contract types for which the type is supported
}
```

The declaration is **explicit and static**. It is not inferred
from runtime traffic; an adapter that publishes events outside
its declared `Capabilities()` is an architectural bug.

### R2 — Gateway exposes `/venues/capabilities` for introspection

The `gateway` binary exposes an HTTP endpoint:

```
GET /venues/capabilities
```

returning the union of all adapters' `Capabilities()` as JSON.
Consumers (operators, the future Odin client, monitoring) can
discover at runtime what each venue declares.

The endpoint registration follows the conditional-registration
pattern of ADR-0010; the gateway's boot test
(`cmd/gateway/boot_test.go`) is updated alongside per the rule in
CLAUDE.md → Core operating protocols #5.

**The endpoint and its boot-test entry are H-7 deliverables**, not
H-2's. This ADR specifies the contract; the wiring lands with the
adapter that first needs it.

### R3 — Gaps are silently-rejected at producer; downstream handles absence

When an adapter receives a venue-native event for an event-type +
contract pair that is **not** declared in its `Capabilities()`:

- The adapter does **not** publish it (silently rejected at
  producer — no NATS publish).
- The adapter increments a counter
  `marketfoundry_adapter_undeclared_event_total{venue,event_type,contract}`
  so the gap is observable.
- A non-zero counter is an architectural signal that
  `Capabilities()` is out of date with adapter parsing reality;
  fix the declaration or fix the parser.

Consumers downstream:

- **MUST tolerate absence** of declared event types. A consumer of
  `observation.markprice` subscribing across all venues does not
  receive events for Coinbase Spot (because Coinbase Spot does
  not declare it); the consumer treats the absence as "no
  signal", not as an error.
- **MAY query `Capabilities()`** at startup to confirm which venues
  it will receive events from for the subscribed event types.

### R4 — `raccoon-cli check venue-parity` enforces declaration coherence

A new raccoon-cli analyzer `check venue-parity` (introduced in
H-7 per P5 of the Fase Harvest) runs in `make verify` via
`quality-gate`. The analyzer enforces:

- Every venue adapter exposes a `Capabilities()` function (no
  silent absence — empty `Capabilities()` is permitted only with
  an explicit comment justifying why).
- `Capabilities()` returns at least one event type per declared
  contract (an adapter that supports `ContractSpot` but declares
  zero event types for it is treated as misconfigured).
- The set of subjects an adapter publishes to (from registry files
  per ADR-0009) is a subset of what `Capabilities()` declares;
  publishing a subject outside the declaration is a `check
  venue-parity` failure.

### Per-venue payload extensions

For events with venue-specific payload fields (Hyperliquid
liquidation's unique field, Bybit position-snapshot specifics),
the foundry's policy is:

- The **envelope** (ADR-0017) stays canonical: `type`,
  `version`, `venue`, `instrument`, etc., are uniform.
- The **payload** may carry venue-specific fields, signaled by
  payload `version` increments per ADR-0017 versioning rules.
- A future ADR may add a `venue_payload_extensions` map at the
  payload level if cross-venue overlay becomes burdensome;
  premature standardization is rejected here.

The canonical model does **not** synthesize events the venue does
not provide (e.g., the foundry does not emulate Coinbase
`markprice` from trade prices to "fill" the parity gap).

## Non-goals

- **Forcing parity completeness.** Some venues will never support
  what others do; declaring parity as an aspiration is dishonest.
  The policy embraces declared gaps, not erases them.
- **HTTP-API authentication on `/venues/capabilities`.** Loopback
  binding (ADR I4 in runtime-invariants) remains the access
  control; auth is a separate decision.
- **WebSocket push of capability changes.** Capabilities change
  rarely (adapter restart, code deploy); polling is sufficient.
- **Cross-venue event synthesis** (emulating one venue's event from
  another's data). Explicitly rejected: synthesized events would
  carry false `venue` fields, undermining cross-venue analytics.
- **Per-client capability subscription contracts.** Cliente Odin
  (H-12+) consumes via the H-11 delivery surface; capability
  semantics on that surface are H-11's concern, not this ADR's.

## Alternatives considered

- **(A) Parity required across all venues.** Rejected: physically
  impossible. Coinbase Spot has no perpetual; Hyperliquid has no
  Spot; no policy compels a venue to publish what it does not
  source.
- **(B) Ignore differences; let consumers discover gaps at runtime.**
  Status quo at the moment of this ADR (since only Binance Spot +
  Futures exist and they are very similar). Rejected: as soon as
  the third venue lands, silent gaps multiply. The cost of
  installing declaration is least at H-2; cost grows monotonically
  with each venue added.
- **(C) Translation layer that emulates absent events** (e.g.,
  synthesize Coinbase mark-price from trade VWAP). Rejected:
  produces events with a `venue` field that lies about the source;
  defeats audit and cross-venue invariants; opens a class of bugs
  that look correct but are not.
- **(D) Capabilities declared in a static config file, not in code.**
  Rejected: capability declarations and parser code drift
  independently; co-locating them in the adapter package makes
  drift detectable by `check venue-parity` (R4).
- **(E) Capabilities inferred from observed runtime traffic.**
  Rejected: inferences require the venue to actually publish all
  its event types at observation time; absence during a quiet
  window would silently retract a capability.

## Consequences

### Positive

- **Multi-venue surface is honest.** Declared capabilities are the
  contract; consumers know what to expect.
- **Adding a new venue is a checklist, not a guessing game.**
  Implement adapter, declare `Capabilities()`, run `check
  venue-parity`, pass.
- **Gap detection is observable.** The
  `adapter_undeclared_event_total` counter surfaces declaration
  drift; a non-zero rate is an architectural alert.
- **Cross-venue analytics have a foundation.** Insights consumers
  can subscribe to declared event types and rely on the declaration
  rather than discovering venue-specific quirks the hard way.
- **Tooling has an inventory to operate on.** Future ondas
  (operations runbooks, observability dashboards) can drive from
  `/venues/capabilities` rather than rediscovering from code.

### Negative

- **Consumers must tolerate absence.** Code that subscribes
  multi-venue cannot assume every venue produces every event type;
  partial coverage is the steady state.
- **Declaration drift is a new failure mode.** A parser change
  without a corresponding `Capabilities()` update is silently
  wrong until `check venue-parity` runs. Mitigated by R4 running
  in `make verify`.
- **No central enforcement of cross-venue semantic equivalence.**
  Two venues both declaring `observation.trade` may differ in
  semantic edge cases (taker side encoding, fee inclusion). This
  ADR scopes only structural parity; semantic equivalence is
  per-event-type concern of the consumer.

## Promoção para Accepted

This ADR is promoted from `Proposed` to `Accepted` when **Onda H-7
(Multi-venue expansion — first non-Binance adapter)** ships:

1. The new venue adapter (typically Bybit) ships under
   `internal/adapters/exchanges/bybit/` with a `Capabilities()`
   function.
2. Existing `binances/` and `binancef/` adapters retrofitted with
   `Capabilities()`.
3. Gateway route `/venues/capabilities` registered, returning the
   union; entry added to `cmd/gateway/boot_test.go` `routes` slice
   per ADR-0010; route documented in `HTTP-API.md`.
4. Counter `marketfoundry_adapter_undeclared_event_total{venue,event_type,contract}`
   exposed in `internal/shared/metrics/`.
5. raccoon-cli `check venue-parity` analyzer ships, runs in `make
   verify` via `quality-gate`, validates R1–R3 statically.
6. `RUNTIME.md` updated with the new venue's stream entries;
   `RESUMPTION.md` updated to reflect the multi-venue surface.

H-7 is responsible for flipping the `Status` field of this ADR to
`Accepted` in the same commit that lands the implementing code.

## References

- ADR [0021](0021-canonical-instrument-and-venue-model.md) — the
  structural model (Venue, CanonicalInstrument, ContractType) this
  ADR's policy operates on; ADR-0021 is the prerequisite.
- ADR [0017](0017-event-envelope-and-versioning.md) — payload
  versioning is the mechanism by which venue-specific payload
  fields are signaled; this ADR explicitly keeps payload extension
  out of the envelope.
- ADR [0008](0008-single-writer-invariant.md) — single-writer per
  stream is preserved: each adapter ingest is the writer for its
  declared event types and contracts; capability declaration does
  not introduce a second writer.
- ADR [0009](0009-subject-taxonomy.md) — subject taxonomy is
  unchanged; `check venue-parity` validates that an adapter
  publishes only to subjects within its declaration.
- ADR [0010](0010-httprouter-trie-constraints.md) — gateway route
  registration discipline; `/venues/capabilities` follows the
  conditional pattern and updates the boot test.
- ADR [0004](0004-raccoon-cli-static-enforcement.md) — analyzer
  framework that `check venue-parity` builds on; P5 of the Fase
  Harvest applies.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro) and P5 (cada
  invariante traz seu enforcement).
- [PROGRAM-0001](../programs/PROGRAM-0001-foundation.md) — Onda H-2
  scope.
- raccoon `docs/adrs/ADR-0017-multi-exchange-normalization.md` —
  inspiração. Foundry diverges by (a) scoping ADR-0022 to
  **operational event-type parity policy** rather than naming
  canonicalization (raccoon's ADR-0017 covers internal-key vs
  display-form, which foundry handles in ADR-0021); (b) introducing
  the `Capabilities()` declaration as the explicit contract surface
  (raccoon has no analog — its parity is implicit in adapter code);
  (c) requiring a `check venue-parity` analyzer to enforce
  declaration coherence in CI (P5 of the Fase Harvest, formalizing
  what raccoon's "MEX-4 planned" left open); and (d) explicitly
  forbidding cross-venue event synthesis (raccoon does not state
  it, and the absence permits future drift).
- raccoon `docs/rfcs/RFC-0010-W9-multi-exchange-readiness.md` —
  technical detail informing this ADR; not transcribed.
