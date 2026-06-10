# ADR 0009: NATS subject taxonomy with verb-last pattern

## Status

Accepted.

## Context

NATS subjects form a hierarchical namespace. The choice of hierarchy
shape affects:
- Subject filter expressivity (consumers subscribe to wildcards)
- Cognitive clarity (someone reading a subject can locate it)
- Future evolution (adding new event types, new aggregates)

For an event-publishing domain (signal, decision, etc.), the
canonical structure could be:

- `{domain}.events.{verb}.{type}` — verb first, type last
  (e.g., `signal.events.generated.rsi`)
- `{domain}.events.{type}.{verb}` — type first, verb last
  (e.g., `signal.events.rsi.generated`)
- `{domain}.{type}.events.{verb}` — type promoted
  (e.g., `signal.rsi.events.generated`)

Until P1A.6c, this choice was implicit in the code rather than
documented. P1A.6c verified the actual pattern in registry files:
the codebase uses **verb last**.

## Decision

**NATS subject taxonomy is `{domain}.{plane}.{type}.{verb}[.{key}]`**
with verb last.

Concretely:
- `signal.events.{type}.generated.{source}.{symbol}.{timeframe}`
- `decision.events.{type}.evaluated.{source}.{symbol}.{timeframe}`
- `strategy.events.{type}.resolved.{source}.{symbol}.{timeframe}`
- `risk.events.{type}.assessed.{source}.{symbol}.{timeframe}`
- `evidence.events.{type}.sampled.{source}.{symbol}.{timeframe}`
  (or `detected` for trade bursts)
- `execution.events.{type}.intent` / `.fill` / `.rejection`
- `observation.events.trade.received.{source}.{symbol}`
- `configctl.{event|events}.config.{verb}` (transition surface,
  documented in RUNTIME)

Planes are: `events`, `event` (configctl singular legacy), `control`,
`command`, `reply`, `query`, `projection`, plus execution-specific
`fill`, `rejection`, `session`, `activation`.

> **Erratum (2026-06-10 — Onda H-6.e, Decisões #1/#3).** The
> `{symbol}` token is the **canonical subject token**, derived from
> `CanonicalInstrument` exclusively via the single helper
> `SubjectToken()` in `internal/domain/instrument`:
> `{base}_{quote}_{contract}` lowercase (e.g. `btc_usdt_spot`,
> `btc_usdt_perpetual`, `btc_usd_coinfutures`) — subject-safe (no
> `.`, `/`, or uppercase) and carrying a **dormant `[_expiry]`
> slot**: expiry is not yet a field of the canonical model, so
> delivery-futures contracts with different expiries collide in
> canonical identity itself — a registered modeling debt (see
> PROGRAM-0004 → H-6.e.2 scope and RESUMPTION known-gaps), not a
> token-formatting concern. Subject builders MUST NOT format the
> token themselves (enforced by raccoon-cli `check subjects`).
>
> Before H-6.e the token was the venue-native lowercase form
> (e.g. `btcusdt`) derived via the transitory `VenueSymbol()`
> helper — VenueSymbol-derived, **not** `CanonicalInstrument.Symbol()`-
> derived as PROGRAM-0004's H-6.e row originally described
> (imprecision corrected by this erratum). Messages published
> before the cutover keep the legacy token until stream TTL (72h)
> retires them — mixed-state by design, precedent H-6.d. Stream
> subject patterns and consumer filters (wildcard `>`) are
> unaffected. KV partition keys intentionally keep the legacy
> composition until **H-6.e.2** (see ADR-0021 criterion #2
> erratum of the same date).

## Consequences

### Positive

- **Type-first filters are efficient**: consumers wanting "all events
  for type X" subscribe to `{domain}.events.{type}.*` — natural.
- **Adding new types is local**: introducing a new signal type doesn't
  require coordinating with other domains.
- **Verb is the "what happened" leaf**: reading a subject tail-to-head
  gives concrete-event-first context.
- **Partition key after verb**: source/symbol/timeframe always at the
  end of the subject, easy to extract with consistent indexing.

### Negative

- **Inconsistency in configctl**: configctl uses both singular
  (`event`) and plural (`events`) in parallel — known transition
  surface (D3 in RESUMPTION). This taxonomy decision applies to
  the new pattern; legacy configctl subjects remain until migrated.
- **Verb at the leaf can be redundant**: many domains publish only
  one verb (signal only `generated`, decision only `evaluated`).
  The taxonomy still consistently includes it for uniformity.
- **No clear typing for the verb name**: nothing enforces that
  "generated", "evaluated", "resolved" follow any convention. Future
  ADR could refine if needed.

## Alternatives considered

**Verb first** (`signal.events.generated.{type}`): rejected after
P1A.6c verified the existing code uses verb last. Changing the
pattern would require migrating all publishers and consumers — high
cost for no benefit.

**Type as second segment** (`signal.{type}.events.{verb}`): cleaner
in some ways (groups by type rather than by plane), but loses the
plane separation that makes `signal.events.*` vs `signal.control.*`
queries clean.

**Flat naming** (single segment, e.g., `signal.rsi-generated`):
rejected — loses all hierarchy benefits. NATS subject hierarchy is
designed for this.

## References

- All `internal/adapters/nats/nats<domain>/registry.go` show the
  pattern in concrete subjects
- [`../GLOSSARY.md`](../GLOSSARY.md) → Surface (the 10 planes)
- [`../RUNTIME.md`](../RUNTIME.md) → NATS subject taxonomy
- D3 in [`../RESUMPTION.md`](../RESUMPTION.md) — configctl
  singular/plural transition surface
