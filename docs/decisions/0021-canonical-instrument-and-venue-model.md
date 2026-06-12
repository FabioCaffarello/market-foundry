# ADR 0021: Canonical instrument and venue model

## Status

Proposed. Foundation ADR delivered in Onda H-2 of the Fase Harvest;
promoted to `Accepted` when Onda H-6 ships the implementing code
(see "Promoção para Accepted" below).

## Date

2026-05-24.

## Context

market-foundry today has two venue adapters:
`internal/adapters/exchanges/binances/` (Binance Spot) and
`internal/adapters/exchanges/binancef/` (Binance USDT-Futures).
Each adapter parses venue-native messages and forwards events
downstream with a `symbol` field carried as a plain Go `string`
(no struct, no enum, no contract).

This is acceptable when there is one venue family. It fails as soon
as the foundry adds a second:

- A symbol string `"BTCUSDT"` is unambiguous **within Binance Spot**,
  but the same string on Bybit, Coinbase, Hyperliquid, or Kraken
  carries different listing rules, lot-size conventions, contract
  types, and trading hours. A consumer that wants "BTC traded
  against USDT, anywhere" must inspect both the `venue` and the
  `symbol` and apply venue-specific parsing.
- Cross-venue capabilities (arbitrage detection, cross-venue
  insights in H-9, multi-venue execution routing) require a
  structural notion of "BTC/USDT perpetual" that holds across
  venues; comparing strings does not give it.
- Adding the third venue would force every cross-venue consumer to
  fork its parsing logic; the cost grows quadratically in the
  worst case.

ADR-0017 added `venue` and `instrument` as canonical envelope
fields. ADR-0020 keyed the Sequencer by `(venue, instrument,
event_type)`. Both depend on this ADR to define **what
`instrument` is structurally**.

This is the cheapest moment to install the canonical model:
before H-6 adds the third venue (Bybit, the prompt suggests) and
before the existing two Binance adapters compound their string-
based contract.

## Decision

market-foundry adopts a **canonical instrument model** rooted in
`internal/domain/instrument/`. Domain layer **never knows venue-
native symbol formats**; normalization happens at the adapter
boundary.

### Domain types

```go
package instrument

// Venue identifies an exchange (or exchange-family product).
type Venue string

const (
    VenueBinance       Venue = "binance"       // Binance Spot
    VenueBinanceFutures Venue = "binancef"     // Binance USDT-margined Futures
    VenueBybit          Venue = "bybit"
    VenueCoinbase       Venue = "coinbase"
    VenueHyperliquid    Venue = "hyperliquid"
    VenueKraken         Venue = "kraken"
    VenueKrakenFutures  Venue = "krakenf"
)

// ContractType classifies how an instrument settles.
type ContractType string

const (
    ContractSpot         ContractType = "spot"
    ContractUSDTFutures  ContractType = "usdtfutures"
    ContractCoinFutures  ContractType = "coinfutures"
    ContractPerpetual    ContractType = "perpetual"
)

// BaseAsset and QuoteAsset are uppercase ticker symbols.
type BaseAsset string
type QuoteAsset string

// CanonicalInstrument is the foundry-internal instrument identity.
// Identical structure across every venue; venue-native nuances
// (lot sizes, tick sizes, listing dates) live in adapter-side
// metadata and are not part of the canonical identity.
type CanonicalInstrument struct {
    Base     BaseAsset
    Quote    QuoteAsset
    Contract ContractType
}

// Symbol returns the canonical string representation:
// "{BASE}/{QUOTE}-{CONTRACT}", e.g., "BTC/USDT-spot",
// "BTC/USDT-perpetual", "BTC/USD-coinfutures".
func (c CanonicalInstrument) Symbol() string { ... }
```

The exact field types and method set are H-6 implementation
choices; the **structure** (Base + Quote + Contract; venue carried
separately at envelope level) is the ADR's commitment.

### Venue enum — initial set

The seven venues listed above are the canonical initial set,
chosen to cover the venues that raccoon supports (so the foundry
absorbs the same operational surface) **in the sequence the
foundry adds them**. Foundry adds them across H-6 (refactor
Binance to canonical) and H-7+ (add Bybit, Coinbase,
Hyperliquid, Kraken). Adding a new venue requires extending the
`Venue` enum and a corresponding adapter; both are normal
per-onda deliverables governed by ADR-0022.

### ContractType enum

Four contract types cover the products the foundry handles in the
near-to-medium term:

- **`spot`** — physical settlement, no leverage by default
  (matches Binance Spot, Coinbase, Kraken).
- **`usdtfutures`** — USDT-margined linear futures with explicit
  expiry (matches Binance USDT quarterlies).
- **`coinfutures`** — coin-margined inverse futures
  (matches Kraken Futures, Binance Coin quarterlies).
- **`perpetual`** — perpetual swap; no expiry, funding rate
  (matches Hyperliquid, Bybit Perp, Binance USDM perp).

Options, structured products, and other exotic contract types are
explicit non-goals (see below).

### Adapter normalization boundary

Every venue adapter ships:

```go
func ToCanonical(native NativeSymbol) (CanonicalInstrument, error)
func FromCanonical(c CanonicalInstrument) (NativeSymbol, error)
```

`NativeSymbol` is the adapter's internal symbol type
(string, struct, etc.). `ToCanonical` is the **only** entry point
through which venue-native strings cross into the foundry's domain
layer. `FromCanonical` is the symmetric exit point (needed when
the foundry submits orders or queries venue-side metadata).

Adapter-side metadata (lot size, tick size, listing date, settlement
currency, etc.) lives in
`internal/adapters/exchanges/<venue>/instrument_metadata.go` (or
equivalent). It is **not** part of `CanonicalInstrument` because it
is venue-specific: the same `(BTC, USDT, perpetual)` instrument
has different lot sizes on Bybit vs Binance USDM.

### Where canonical instrument appears

- **Envelope field `instrument`** (ADR-0017) — every event carries
  the canonical instrument.
- **Subject taxonomy `{key}` segment** (ADR-0009) — subjects
  encode `.../source/{symbol}/...` using `CanonicalInstrument.Symbol()`.
- **Sequencer stream key** (ADR-0020) — `(venue, instrument,
  event_type)`.
- **Domain types** (`internal/domain/`) — receive
  `CanonicalInstrument`, never `string`.
- **ClickHouse schemas** — columns store the canonical form;
  cross-venue queries are joins on `(base, quote, contract)` not
  on opaque symbol strings.

## Non-goals

- **Options.** Strike, expiry per leg, multi-leg structures are
  excluded; foundry does not trade options in the foreseeable
  roadmap.
- **Cross-listed asset alias resolution.** "BTC vs WBTC vs cbBTC"
  are distinct `BaseAsset` values; the canonical model does **not**
  treat them as equivalent. A consumer that wants
  cross-token-version analytics applies its own equivalence policy.
- **USDT vs USDC vs FDUSD vs DAI equivalence.** Different
  `QuoteAsset` values; the canonical model does **not** collapse
  stablecoins. Per-venue stablecoin liquidity differs materially;
  collapsing them would erase information.
- **Expiry encoding for dated futures.** ContractType
  `usdtfutures`/`coinfutures` covers the **contract class**;
  per-expiry differentiation (e.g., BTC-0628 vs BTC-0927) is
  carried in adapter-side metadata and may surface in `payload`
  fields per ADR-0017. Whether the canonical model later grows an
  `Expiry` field is an explicit future decision (not this ADR's).

  > **Erratum (2026-06-12 — Onda H-7.c, Decisão #4 da abertura de
  > H-7).** The future decision was made: `CanonicalInstrument`
  > grows an **optional `Expiry string` field** — canonical format
  > **YYMMDD** (six digits, e.g. `240329`; maps directly from the
  > Binance delivery suffix `_YYMMDD`, sortable, subject-safe),
  > permitted ONLY for the dated contract classes
  > (`usdtfutures`/`coinfutures`; spot/perpetual with expiry is a
  > validation error). Empty expiry remains legal for the dated
  > classes (pre-H-7.c constructions), with the collapsed-identity
  > caveat of G10 applying to those values only. Construction via
  > `NewDelivery(base, quote, contract, expiry)`; `Symbol()` gains
  > an `@{expiry}` suffix when non-empty
  > (`BTC/USDT-usdtfutures@240329`) and `FromSymbol` parses it
  > back; the subject token activates its dormant 4th component
  > (see the ADR-0009 erratum of the same date). Instruments
  > without expiry produce byte-identical `Symbol()` and
  > `SubjectToken()` to the pre-H-7.c forms — zero-impact
  > lock-ins in the instrument tests. ClickHouse persistence of
  > expiry is intentionally deferred to the onda that enables
  > delivery futures at ingest (gap G11 in RESUMPTION).
- **Settlement currency separation for perpetuals.** For now,
  `BTC/USDT-perpetual` on Hyperliquid (USDC-settled) and on Binance
  USDM (USDT-settled) are distinct (`QuoteAsset` differs). If a
  future onda needs a "settled-in-any-stable" abstraction, it adds
  it explicitly.
- **HTTP-API symbol format.** The gateway may expose either the
  canonical form or a venue-friendly alias at the HTTP boundary;
  internal mesh always uses canonical. HTTP shape is HTTP-API.md's
  decision, not this ADR's.

## Alternatives considered

- **(A) String as instrument (status quo).** Rejected: forces
  every consumer to parse; cross-venue logic becomes a tangle of
  per-venue branches; the canonical thesis collapses.
- **(B) Per-venue instrument types** (`BinanceSymbol`,
  `BybitSymbol`, ...). Rejected: every cross-venue function takes
  a union type or a generic; multiplies the surface area without
  the benefit of a single normalized contract.
- **(C) FIX-style instrument identifiers** (SecurityID +
  SecurityIDSource). Rejected: overkill for crypto; FIX semantics
  optimized for traditional securities and registry-backed
  identifiers; crypto has no equivalent global registry.
- **(D) IETF / CoinGecko style asset IDs.** Rejected: introduces
  an external dependency for naming; adds drift surface (asset IDs
  change with token migrations); foundry's needs are satisfied by
  uppercase ticker symbols as a convention.
- **(E) Embed venue inside the instrument** (e.g.,
  `CanonicalInstrument` carries `Venue` as a field). Rejected:
  `venue` lives at the envelope level (ADR-0017) because the same
  instrument identity is shared across venues for cross-venue
  capabilities; coupling them again here defeats that goal.

## Consequences

### Positive

- **Multi-venue is structurally clean.** New venues are an adapter
  + a `Venue` enum entry; no domain-layer changes.
- **Cross-venue capabilities are unlocked.** Insights (H-9),
  arbitrage tracking, multi-venue execution all operate over
  uniform `CanonicalInstrument` values.
- **ClickHouse cross-venue queries are joinable.** Aggregations by
  `(Base, Quote, Contract)` are straightforward; no per-venue
  SQL forks.
- **Schema versioning is simpler.** Domain functions that take
  `CanonicalInstrument` survive venue additions without signature
  changes.
- **The Sequencer's stream key (ADR-0020) is well-defined.** The
  triple `(venue, instrument, event_type)` is structurally
  meaningful and stable.

### Negative

- **Refactor of the two existing Binance adapters required in H-6**
  to introduce `ToCanonical` / `FromCanonical` and migrate
  internal call sites. Scope is bounded but real.
- **Convention drift risk on `BaseAsset` strings.** A misspelled
  ticker (e.g., `"btc"` vs `"BTC"`) silently fragments analytics.
  Mitigated by normalization in the constructor and validation in
  the adapter boundary; an analyzer to verify uppercase-asset
  convention may follow in H-7 per P5.
- **Per-venue metadata divergence is invisible at the domain level.**
  Lot-size-aware execution must read adapter-side metadata, not
  `CanonicalInstrument`. This is correct; the consequence is that
  execution code is necessarily venue-aware.

## Promoção para Accepted

This ADR is promoted from `Proposed` to `Accepted` when **Onda H-6
(Multi-venue foundation — Binance refactor + canonical model)**
ships, in the final sub-onda **H-6.f (cleanup pass)** after all
prior sub-ondas (H-6.a through H-6.e) have landed in `main`.

H-6 is implemented in sub-ondas H-6.a through H-6.f per
[PROGRAM-0004](../programs/PROGRAM-0004-multi-venue.md) sub-onda
sequencing policy (strict serial). Each sub-onda delivers a slice
of the cascade discovered during H-6.a pré-flight (342 `.Symbol`
references across 106 production files in 31 packages). Sub-ondas
deliver code; this ADR is **not promoted** until the final
sub-onda H-6.f satisfies all criteria literally.

Acceptance criteria:

1. `internal/domain/instrument/` package created with `Venue`,
   `BaseAsset`, `QuoteAsset`, `ContractType`, `CanonicalInstrument`
   as specified. *(Deliverable in H-6.a.)*
2. Existing `binances/` and `binancef/` adapters refactored to
   implement `ToCanonical` / `FromCanonical`; all domain-layer call
   sites migrated from `string` to `CanonicalInstrument`.
   *(Deliverable across H-6.a–H-6.e.2: H-6.a migrates `ObservationTrade`;
   H-6.b migrates Evidence/Signal/Decision/Strategy/Risk types; H-6.c
   migrates application-layer callers + samplers + evaluators; H-6.e
   addresses NATS subject composition.)*

   **(Erratum 2026-06-10 — H-6.e Decisão #2.)** KV partition-key
   composition + the HTTP read contract split into sub-onda
   **H-6.e.2**: the keys are parser-free (constructed on both write
   and read paths, never parsed), but the read path constructs them
   from the HTTP `(source, symbol, timeframe)` query contract
   (`parseQueryKeyParams`, `handlers/common.go`) — migrating them in
   H-6.e would require either changing that contract or reintroducing
   venue→canonical string inference at the boundary, the exact
   anti-pattern eliminated in H-6.c (`anti_patterns.toml`).
   **Criterion #2 is satisfied literally only after H-6.e.2.**
   Promotion chain: H-6.e → H-6.e.2 → H-6.f; H-6.f additionally
   blocks on H-6.e.2 because `PartitionKey()` legitimately consumes
   the transitory `VenueSymbol()` until the keys migrate, and the
   helper's deletion is an H-6.f deliverable.
3. Envelope `instrument` field (ADR-0017) carrying canonical form
   for at least the `OBSERVATION_EVENTS` stream. *(Deliverable in
   H-6.a: writers emit `CanonicalInstrument.Symbol()` as the
   envelope's `instrument` string field; proto schema unchanged
   per Decision #2 (a) of H-6.a.)*
4. **(Erratum 2026-05-25 — split into #4a and #4b.)**

   **4a — Writer-side canonical handling.** Writer ingests
   `CanonicalInstrument` from domain events; serializes via
   `Instrument.Symbol()` into legacy `symbol` ClickHouse column.
   **Zero schema change.** *(Deliverable in H-6.a.)*

   **4b — Schema migration + dedicated read path.** New ClickHouse
   migration adds canonical columns (`base`, `quote`, `contract`);
   writer dual-writes; analytical client reads canonical preferred
   with legacy fallback; cutover documented in runbook.
   *(Deliverable in H-6.d, **sequenced after H-6.c**
   (application-layer migrated) **and before H-6.f**
   (final cleanup + ADR promotion).)*

   **Justificativa do split:** schema migration introduces data
   ordering risk, back-compat verification overhead, and rollback
   procedures distinct from domain-type refactor. Bundling
   conflates concerns. Split aligns with ADR-0003 (forward-only
   migrations, each in dedicated wave).
5. `RUNTIME.md` and `RESUMPTION.md` updated; `GLOSSARY.md` entry
   for "Canonical Instrument" pointing here. *(Deliverable
   incrementally across sub-ondas; final consolidation in H-6.f.)*

H-6.f is responsible for flipping the `Status` field of this ADR
to `Accepted` in the same commit that lands the implementing
code closing the final sub-onda. **Sub-ondas H-6.a–H-6.e DO NOT
promote** this ADR; each closes its slice with the ADR still
`Proposed`.

## References

- ADR [0017](0017-event-envelope-and-versioning.md) — envelope
  `venue` and `instrument` fields whose structure this ADR defines.
- ADR [0020](0020-sequencing-and-time-normalization.md) — Sequencer
  stream key `(venue, instrument, event_type)` whose `instrument`
  is the canonical model defined here.
- ADR [0022](0022-multi-venue-normalization-policy.md) — operational
  policy for how adapters handle event-type parity across venues;
  this ADR is its structural prerequisite.
- ADR [0005](0005-layer-sovereignty.md) — `internal/domain/` is the
  natural home for `CanonicalInstrument`; adapters normalize
  inward.
- ADR [0008](0008-single-writer-invariant.md) — single writer per
  stream remains intact; canonicalization happens before
  publication, the writer remains unique.
- ADR [0009](0009-subject-taxonomy.md) — subject `{key}` segment
  encodes `CanonicalInstrument.Symbol()`.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — P3
  (capacidade portada passa por documento primeiro) and the
  "What this repository is NOT" section that excludes raccoon's
  auxiliary binaries while preserving the canonical-instrument
  capability they encoded.
- [PROGRAM-0001](../programs/PROGRAM-0001-foundation.md) — Onda H-2
  scope.
- `internal/adapters/exchanges/binances/`,
  `internal/adapters/exchanges/binancef/` — the two adapters that
  H-6 refactors to comply with this ADR.
- raccoon `docs/adrs/ADR-0011-marketdata-binance-canonical-instrument-and-event-mapping.md`
  — inspiração. Foundry diverges by (a) defining a multi-venue
  model from the start (raccoon's ADR-0011 is single-venue-Binance-
  specific and does not declare a Venue enum); (b) requiring a
  structured `CanonicalInstrument` (Base + Quote + Contract) rather
  than a flat canonical string `"BTCUSDT"`; and (c) carrying the
  `instrument` field at the envelope level (ADR-0017) for cross-
  venue routing rather than as a payload-internal detail.

## Changelog

- **2026-05-24** — ADR-0021 created (Onda H-2, status
  `Proposed`). See PR #21.
- **2026-05-25** — **Erratum**: criterion #4 split into #4a
  (writer-side canonical handling, deliverable in H-6.a, zero
  schema change) and #4b (ClickHouse schema migration + dedicated
  read path, deliverable in H-6.d, sequenced after H-6.c and
  before H-6.f). Reason: schema migration introduces data ordering
  risk, back-compat verification overhead, and rollback procedures
  distinct from domain-type refactor; bundling conflates concerns
  per ADR-0003. Also clarified that H-6 is implemented in
  sub-ondas H-6.a–H-6.f per PROGRAM-0004 sequencing policy and
  that promotion to `Accepted` occurs only in H-6.f when all
  criteria are literally satisfied. Lands as **commit 0** of the
  H-6.a PR. Status remains `Proposed`.
