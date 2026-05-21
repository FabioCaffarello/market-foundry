# Stage S35 — Signal Domain Design

> **Status:** Complete
> **Date:** 2026-03-17
> **Objective:** Design the signal domain layer — boundaries, families, activation, ownership, and query surface — without implementation.
> **Scope:** Design-only. No code, no schema changes, no actor implementation.

---

## Executive Summary

S35 produced the canonical design for the **signal domain** — the third layer in Market Foundry's progression (`observation → evidence → signal`). Signal transforms evidence into actionable interpretations (e.g., MACD crossover, RSI overbought) without deciding or acting on them.

The design resolves five concerns:

1. **Domain boundaries** — signal is a separate bounded context with its own types, stream, projections, and query namespace. Zero imports from `internal/domain/evidence/` in `internal/domain/signal/`.
2. **Stream families** — two Phase 1 families (MACD, RSI), both single-evidence (candle-only), with three deferred candidates cataloged.
3. **Activation model** — config-driven via `pipeline.signal_families` in derive and store configs, extending the S34 `pipeline.families` pattern. Opt-in, not backward-compatible default.
4. **Ownership** — derive produces signal events; store projects and serves them; gateway translates HTTP. Single-writer invariant at every layer.
5. **Query surface** — latest-only in Phase 1, under `/signal/{type}/latest`, following evidence query patterns exactly.

Five design documents were produced:
- `signal-domain-design.md` — core design, types, invariants, deferred items
- `signal-stream-families.md` — family catalog with event contracts and projections
- `signal-activation-and-ownership.md` — activation flow, ownership matrices, preconditions
- `signal-query-surface-guidelines.md` — query chain, naming conventions, response format
- `signal-readiness-review.md` — pre-existing readiness assessment (S25), whose highest-severity gap (config-driven activation) was resolved in S34

---

## Design Decisions and Rationale

### 1. Signal lives in the derive binary, not a separate binary

Signal processing runs in the same `derive` binary that produces evidence. Rationale:
- Derive is already the single writer for derived events.
- Signal consumes evidence that derive itself just produced — a separate binary would add cross-process latency for data already in memory.
- The `FamilyProcessor` pattern already supports multiple families within derive.
- Stream-level isolation is maintained: `SIGNAL_EVENTS` is a separate JetStream stream from `EVIDENCE_EVENTS`, with a separate publisher actor.

A separate binary remains architecturally possible but offers no benefit at current scale.

### 2. Local fan-out from evidence samplers to signal samplers

Signal samplers in derive receive finalized evidence via **local actor messages**, not by subscribing to `EVIDENCE_EVENTS` JetStream.

When `CandleSamplerActor` finalizes a candle, it sends a local message to `SourceScopeActor`, which fans out to registered signal samplers. This avoids re-consuming from JetStream data that is already in-memory.

If future multi-evidence signals (candle + volume) prove complex to wire locally, JetStream-based evidence consumption within derive can be introduced. The design supports both.

### 3. Generic Signal struct with Metadata map

A single `Signal` struct with `Value string` (primary output) and `Metadata map[string]string` (type-specific fields) was chosen over per-type structs. Rationale:
- Different signal types produce different auxiliary values (MACD: signal_line + histogram; RSI: avg_gain + avg_loss).
- Per-type structs would force domain-level proliferation before knowing which signals survive.
- Store serializes the full struct; gateway passes it through; clients parse metadata by type.
- If types stabilize and metadata parsing becomes error-prone, per-type structs can be introduced in S37+.

### 4. Latest-only projections in Phase 1

Signal projections start with one KV entry per partition key (`source.symbol.timeframe`), overwritten on each finalized signal. No history projections. Rationale:
- No downstream consumer requires signal history yet.
- The decision domain (signal's primary future consumer) does not exist.
- Evidence proved that latest-first → history-later is correct sequencing (candle history entered in S19/S20).

### 5. Separate `signal_families` config key (opt-in, not default-on)

Unlike evidence `families` (where absent = all families enabled for backward compatibility), `signal_families` absent = **no signal activation**. Signal is a new domain and must be explicitly opted into.

### 6. No signal-to-signal derivation in Phase 1

All signal families derive exclusively from evidence. No signal family may consume another signal family's events. This keeps the dependency graph flat and avoids unbounded derivation chains.

---

## What Is Defined

### Boundaries

| Boundary | Rule |
|----------|------|
| Signal ↔ Evidence | Signal reads evidence events. Never writes to evidence streams. `internal/domain/signal/` has zero imports from `internal/domain/evidence/`. Evidence has no knowledge of signal. |
| Signal ↔ Store | Store materializes signal projections into `SIGNAL_*_LATEST` buckets. Signal domain does not access KV directly. Same three gates: Final, Validate, Monotonicity. |
| Signal ↔ Gateway | Gateway exposes `/signal/{type}/{operation}`. Stateless translator, no domain logic, no KV access, no metadata interpretation. |
| Signal ↔ Decision | Decision domain does not exist. Signal prepares for it but does not anticipate its shape. No "what to do next" encoding in signal events. |
| No backflow | Signal never modifies, enriches, or annotates evidence. Adding or removing a signal family has zero effect on the evidence domain. |

### Stream Families

| ID | Family | Evidence Input | Status |
|----|--------|---------------|--------|
| SF-01 | signal.macd | evidence.candle (Final=true) | Phase 1 — S36 |
| SF-02 | signal.rsi | evidence.candle (Final=true) | Phase 1 — S36 |
| SF-03 | signal.vwmom | evidence.candle + evidence.volume | Deferred — S37+ |
| SF-04 | signal.bbands | evidence.candle | Deferred — S37+ |
| SF-05 | signal.vwapdev | evidence.volume | Deferred — S38+ |

Phase 1 families are both candle-only consumers, minimizing the initial wiring to one evidence input path.

### Activation

- **Family activation**: `pipeline.signal_families` in JSONC configs for derive and store. Not runtime-dynamic (requires restart).
- **Binding activation**: `BindingWatcherActor` (existing, hardened in S34) controls which source/symbol pairs get signal samplers. Runtime-dynamic via configctl events.
- **Both conditions required**: A signal sampler exists only when the family is in `signal_families` AND a binding is active for the source/symbol pair.

### Ownership

| Concern | Owner Binary | Owner Actor |
|---------|-------------|-------------|
| Signal event production | derive | `SignalPublisherActor` (one per SourceScopeActor) |
| Signal sampling logic | derive | `SignalSamplerActor` (one per type x symbol x timeframe) |
| Signal event consumption | store | `SignalConsumerActor` (one per signal type) |
| Signal projection (KV) | store | `SignalProjectionActor` (one per signal type) |
| Signal query serving | store | `QueryResponderActor` (extended) |
| Signal HTTP translation | gateway | `SignalHandler.GetLatest` (stateless) |

Single-writer invariant: only derive writes `SIGNAL_EVENTS`; only store writes `SIGNAL_*_LATEST`.

### Query Surface

| Signal Type | HTTP Route | NATS Subject | KV Bucket |
|-------------|-----------|-------------|-----------|
| MACD | `GET /signal/macd/latest?source=X&symbol=Y&timeframe=Z` | `signal.query.macd.latest` | `SIGNAL_MACD_LATEST` |
| RSI | `GET /signal/rsi/latest?source=X&symbol=Y&timeframe=Z` | `signal.query.rsi.latest` | `SIGNAL_RSI_LATEST` |

Response returns `"signal": { ... }` with full Signal struct including opaque metadata. `null` if not found (HTTP 200).

---

## Intentional Limits

1. **No implementation in S35.** All five documents are design-only. Code changes, schema additions, stream creation, and actor implementation are S36 scope.
2. **No history projections.** Latest-only until a concrete consumer demands historical lookback.
3. **No multi-evidence signals.** Phase 1 is candle-only. Multi-evidence (candle + volume) requires validated local fan-out and enters in S37+.
4. **No signal-to-signal chains.** Flat dependency graph: evidence → signal only.
5. **No composite/aggregated query endpoints.** No "all signals for symbol" API. Clients query per-type.
6. **No WebSocket/streaming for signal.** HTTP request/reply only.
7. **No decision domain anticipation.** Signal events are self-contained. Decision domain design is S38+.

---

## Risks and Gaps

| # | Risk/Gap | Severity | Mitigation |
|---|----------|----------|------------|
| R-1 | `Metadata map[string]string` may become error-prone as signal types grow | LOW | Controlled trade-off for Phase 1. Per-type structs can be introduced in S37+ if parsing errors surface. |
| R-2 | Local fan-out for multi-evidence signals (candle + volume) is unvalidated | MEDIUM | Deferred to S37+. Phase 1 uses single-evidence only. JetStream fallback path is documented. |
| R-3 | Signal warm-up periods (26 candles for MACD, 14 for RSI) mean delayed first output | LOW | Expected behavior. No mitigation needed — warm-up is inherent to EMA/RSI math. |
| R-4 | Raccoon-CLI drift rules for signal contracts do not exist yet | MEDIUM | Enters alongside S36 implementation. Risk is that S36 code ships without governance checks if CLI rules lag. |
| R-5 | Signal readiness review (S25) predates S34 activation hardening | LOW | The highest-severity gap (P-9: BindingWatcher) was resolved in S34. Remaining preconditions (P-1 through P-8) are S36 implementation tasks, not blockers. |
| R-6 | QueryResponderActor serves both evidence and signal — could become a bottleneck | LOW | Separate queue groups (`signal.query` vs `evidence.query`) enable independent scaling. Split to separate actors only if latency proves it necessary. |

---

## Deferred to S36 (Implementation)

S36 will implement the signal domain as designed in S35:

- `internal/domain/signal/` — `Signal` type, `Validate()`, event names
- `internal/application/signal/` — MACD and RSI samplers (pure logic, table-driven tests)
- `SignalPublisherActor` in derive's `SourceScopeActor`
- `SignalFamilyProcessor` registration for MACD and RSI
- `SignalConsumerActor` + `SignalProjectionActor` in store
- `SIGNAL_EVENTS` stream definition in NATS registry
- `SIGNAL_MACD_LATEST` and `SIGNAL_RSI_LATEST` KV buckets
- `pipeline.signal_families` in settings schema
- Signal query handler in gateway under `/signal/{type}/latest`
- Raccoon-CLI drift rules for signal contracts

## Deferred to S37+ (Hardening)

- Signal history projections (`SIGNAL_MACD_HISTORY`, etc.)
- Multi-evidence signal support (SF-03 vwmom: candle + volume → momentum)
- Per-type domain structs (if Metadata proves insufficient)
- Signal expiration lifecycle (`signal_expired` event)
- Bollinger Bands (SF-04 bbands)

## Deferred to S38+ (Extension)

- Signal-to-signal composition (composite signals)
- VWAP deviation signal (SF-05 vwapdev)
- Decision domain design
- Cross-timeframe and cross-source signal aggregation

---

## Preparation for First Signal Slice (S36)

Before S36 implementation begins, these structural preconditions must hold:

| # | Precondition | Status after S34/S35 |
|---|-------------|---------------------|
| P-1 | `pipeline.signal_families` key in settings schema | Not implemented — S36 task |
| P-2 | `SIGNAL_EVENTS` JetStream stream created on startup | Not implemented — S36 task |
| P-3 | `IsFamilyEnabled` extended for signal families | Not implemented — S36 task |
| P-4 | SignalFamilyProcessor registration in DeriveSupervisor | Not implemented — S36 task |
| P-5 | Signal projection pipelines in StoreSupervisor | Not implemented — S36 task |
| P-6 | Signal KV buckets created on store startup | Not implemented — S36 task |
| P-7 | Signal query subjects in QueryResponderActor | Not implemented — S36 task |
| P-8 | Signal HTTP routes in gateway | Not implemented — S36 task |
| P-9 | BindingWatcherActor fully wired in derive | **Active** — resolved in S34 |
| P-10 | Evidence families operational (candle at minimum) | **Active** — operational since S06 |

All P-1 through P-8 are implementation tasks, not design gaps. The design documents provide complete specifications for each.

---

## Source Documents

| Document | Path | Purpose |
|----------|------|---------|
| Signal Domain Design | `docs/architecture/signal-domain-design.md` | Core design, types, invariants, actor hierarchy |
| Signal Stream Families | `docs/architecture/signal-stream-families.md` | Family catalog, event contracts, projections |
| Signal Activation and Ownership | `docs/architecture/signal-activation-and-ownership.md` | Activation flow, ownership matrices, configuration |
| Signal Query Surface Guidelines | `docs/architecture/signal-query-surface-guidelines.md` | Query chain, naming, response format, gateway rules |
| Signal Readiness Review | `docs/architecture/signal-readiness-review.md` | Pre-signal readiness assessment (S25 origin) |
