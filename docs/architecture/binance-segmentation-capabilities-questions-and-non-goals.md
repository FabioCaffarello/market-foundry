# Binance Spot/Futures Segmentation — Capabilities, Questions, and Non-Goals

**Wave:** Binance Spot/Futures Segmentation Foundation
**Charter stage:** S390
**Date:** 2026-03-22
**Companion:** [`binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md`](binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md)

---

## 1. Capability Targets

These are the specific capabilities this wave must deliver. Each maps to a
governing question and a wave block.

### C1 — Canonical Venue Model with Segment Dimension

**What:** Domain types that decompose a venue into orthogonal dimensions:
exchange, market segment, environment, execution mode.

**Why:** The current `VenueType` enum is flat (`paper_simulator`,
`binance_futures_testnet`). Adding Spot as another flat enum value creates a
combinatorial naming problem and obscures the segment boundary.

**Acceptance:** `Exchange`, `MarketSegment`, `VenueEnvironment` types exist in
the domain. `VenueType` is derivable from their composition. Source value
convention (`binancef`, `binances`) is documented and enforced.

**Governing question:** SEG-Q1

### C2 — Binance Spot Testnet Adapter

**What:** `BinanceSpotTestnetAdapter` implementing `VenuePort` and
`VenueQueryPort`, targeting `testnet.binance.vision/api/v3/order`.

**Why:** Spot and Futures use different API endpoints, different response
schemas, and different base URLs. A single adapter cannot serve both without
introducing conditional branching that violates the adapter boundary contract.

**Acceptance:** Adapter passes unit tests for submit and query. Error
classification matches the 8-class model (S308/S310). Fill records carry
correct Spot-specific fields. Decorator pipeline wraps it identically to
Futures.

**Governing questions:** SEG-Q2, SEG-Q3

### C3 — Config-Driven Segment Enablement

**What:** Extended config schema with `binance_spot_testnet` type, `segment`
field, and fail-closed validation.

**Why:** Each execute binary must know at startup which segment it serves.
Config validation must reject mismatched type/segment combinations and enforce
`dry_run=true` as default for every new type.

**Acceptance:** Config validation rejects invalid combinations. `dry_run`
defaults to `true` for new types. Credential env var naming follows segment
convention.

**Governing questions:** SEG-Q4, SEG-Q5, SEG-Q9

### C4 — Compose-Level Segment Isolation

**What:** Docker Compose profile running `execute-futures` and `execute-spot`
concurrently with isolated streams, KV buckets, credentials, and control gates.

**Why:** Multi-binary orchestration was proven generically (S370–S375) but
never with segment-specific isolation. This block proves that two execute
instances serving different segments do not collide.

**Acceptance:** Smoke script proves concurrent boot, independent activation
reporting, segment-isolated NATS subjects, and independent gate control.

**Governing questions:** SEG-Q6, SEG-Q7, SEG-Q8

### C5 — Mainnet Extensibility Proof (Structural)

**What:** Evidence that the segmented model can be extended to
`binance_futures_mainnet` and `binance_spot_mainnet` without structural
changes — only new enum values, config entries, and credential env vars.

**Why:** The wave must prove that segmentation is a foundation, not a testnet
hack. Mainnet execution is a non-goal of this wave, but mainnet
**extensibility** is a success criterion.

**Acceptance:** Evidence gate evaluates extensibility structurally. No mainnet
code is written.

**Governing question:** SEG-Q10

---

## 2. Governing Questions (Full Detail)

### SEG-Q1: Venue Model Orthogonality

> Does the venue model cleanly separate exchange, market segment, and
> environment as orthogonal dimensions?

**Evidence required:**
- Domain types `Exchange`, `MarketSegment`, `VenueEnvironment` exist.
- `VenueType` can be derived from their combination.
- No conditional branching based on combined type string inside adapter code.

**Judgment:** FULL if types exist and are used by config validation and adapter
selection. SUBSTANTIAL if types exist but some adapter code still uses raw
strings.

### SEG-Q2: Adapter Extensibility

> Can a new market segment (e.g., Spot) be added without modifying existing
> adapter code?

**Evidence required:**
- `BinanceSpotTestnetAdapter` added as a new file.
- `BinanceFuturesTestnetAdapter` unchanged (no diff).
- Shared HTTP/auth helpers extracted or already shared.

**Judgment:** FULL if Futures adapter has zero diff. SUBSTANTIAL if minor
shared-helper extraction required a Futures-side change.

### SEG-Q3: Spot Adapter Correctness

> Does `BinanceSpotTestnetAdapter` satisfy `VenuePort` and `VenueQueryPort`
> with correct Spot-specific endpoint and response mapping?

**Evidence required:**
- Unit tests covering submit, query, error classification, fill extraction.
- Endpoint is `testnet.binance.vision/api/v3/order`.
- Response field mapping matches Spot API schema (not Futures).

**Judgment:** FULL if all unit tests pass and field mapping is validated.

### SEG-Q4: Config Validation Rigor

> Does config validation reject invalid type/segment combinations at startup?

**Evidence required:**
- Test: `type=binance_spot_testnet` + `segment=futures` → rejected.
- Test: `type=binance_futures_testnet` + `segment=spot` → rejected.
- Test: unknown type → rejected.

**Judgment:** FULL if all negative cases covered.

### SEG-Q5: Fail-Closed Preservation

> Does `dry_run=true` remain the default for all new venue types?

**Evidence required:**
- Config with `type=binance_spot_testnet` and no `dry_run` field → defaults to
  `true`.
- DryRunSubmitter wraps Spot adapter identically to Futures.

**Judgment:** FULL if both conditions proven.

### SEG-Q6: Stream/KV Isolation

> Can Spot and Futures execute binaries run concurrently in compose without
> stream/KV collision?

**Evidence required:**
- Smoke script starts both services.
- Fill events from Spot land on `execution.fill.*.binances.*` subjects.
- Fill events from Futures land on `execution.fill.*.binancef.*` subjects.
- KV keys are source-prefixed and non-overlapping.

**Judgment:** FULL if smoke passes. PARTIAL if manual verification needed.

### SEG-Q7: Activation Surface Segment Awareness

> Are activation dimensions segment-aware and independently observable?

**Evidence required:**
- Each execute instance reports its segment in activation dimensions.
- HTTP endpoint shows both instances with distinct segment tags.

**Judgment:** FULL if observable via API.

### SEG-Q8: Independent Gate Control

> Does the control gate operate independently per segment instance?

**Evidence required:**
- Halting the Futures gate does not halt the Spot instance.
- Each instance responds independently to its own gate status.

**Judgment:** FULL if proven in smoke. Design-level if compose wiring supports
it but not smoke-tested.

### SEG-Q9: Credential Isolation

> Is the credential isolation between segments enforced?

**Evidence required:**
- Spot instance uses only `MF_BINANCE_SPOT_TESTNET_*` env vars.
- Futures instance uses only `MF_BINANCE_FUTURES_TESTNET_*` env vars.
- Config validation rejects if wrong credential set is present.

**Judgment:** FULL if enforced at validation. SUBSTANTIAL if enforced by
convention only.

### SEG-Q10: Mainnet Extensibility

> Can the segmented architecture be extended to mainnet types without
> structural changes?

**Evidence required:**
- Adding `binance_futures_mainnet` to `knownVenueTypes` requires only:
  - new enum value;
  - new config file;
  - new credential env vars;
  - new compose service entry.
- No domain, adapter interface, or pipeline changes required.

**Judgment:** FULL if structurally proven at gate. SUBSTANTIAL if minor
interface extensions anticipated.

---

## 3. Non-Goals (Frozen Exclusions)

Each non-goal is a boundary that **must not** be crossed during this wave.
Violation requires a new charter.

### NG-1: Mainnet Execution

No mainnet venue calls. All adapters target testnet endpoints only. Mainnet
extensibility is proven structurally, not operationally.

**Why frozen:** Mainnet requires credential governance, fund safety, and
operational runbooks that are out of scope. Mixing mainnet execution into a
segmentation wave creates unacceptable blast radius.

### NG-2: Multi-Exchange Support

Only Binance. No Coinbase, Kraken, Bybit, or other exchange adapters. The
venue model must be exchange-agnostic in design but exchange-specific in
implementation for this wave.

**Why frozen:** Multi-exchange expands the adapter surface exponentially. The
goal is to prove segmentation within one exchange first.

### NG-3: Full OMS

No order management beyond the existing seven-state lifecycle. No cancel API,
no amend, no order book, no position tracking.

**Why frozen:** OMS foundation was established in S382–S388. This wave adds
segmentation to the existing lifecycle, not new lifecycle capabilities.

### NG-4: Portfolio Risk Management

No portfolio-level risk, exposure tracking, or cross-segment risk aggregation.

**Why frozen:** Risk management requires position state that does not exist
yet. Introducing it in a segmentation wave creates a dependency chain that
blocks segmentation delivery.

### NG-5: Advanced Order Types

Market orders only. No limit, stop-loss, OCO, or conditional orders.

**Why frozen:** Each order type has distinct API parameters, response shapes,
and lifecycle implications. Market-only keeps the adapter surface minimal.

### NG-6: WebSocket Fill Streaming

Synchronous REST-based execution only. No WebSocket connections for real-time
fill updates.

**Why frozen:** WebSocket fills introduce connection management, reconnection
logic, and event ordering concerns that are orthogonal to segmentation.

### NG-7: Multi-Symbol Routing Within a Single Binary

Each execute binary serves one symbol. No intra-binary symbol routing.

**Why frozen:** The single-symbol-per-instance model is proven and stable.
Multi-symbol routing is a future concern.

### NG-8: Real Trading as Wave Focus

This wave proves segmentation architecture, not trading outcomes. Dry-run is
the default. Real venue calls are optional validation, not the primary
deliverable.

**Why frozen:** Conflating architecture proof with trading proof creates
unclear success criteria.

### NG-9: ClickHouse Schema Changes for Segment Dimension

No ClickHouse migration to add segment columns. The analytical storage layer
consumes events as-is.

**Why frozen:** Schema changes require migration governance and downstream
query updates. Defer to a dedicated analytical wave.

### NG-10: Margin Mode or Leverage Configuration

No margin type (cross/isolated) or leverage settings in config or adapters.

**Why frozen:** Margin and leverage are Futures-specific concerns that
introduce risk management dependencies (NG-4).

### NG-11: Cross-Segment Position Management

No position tracking across Spot and Futures. No hedge mode. No
cross-collateral.

**Why frozen:** Position management is a full OMS feature (NG-3) with
cross-segment complexity.

### NG-12: Fee Tier Differentiation

No fee tier logic. Fill records carry raw fee values from venue responses.
No maker/taker distinction.

**Why frozen:** Fee optimization is a trading concern, not a segmentation
concern.

### NG-13: Platform-Wide Redesign

No actor topology changes, no new NATS stream families, no domain model
extensions beyond venue segmentation types.

**Why frozen:** The wave is scoped to venue segmentation. Broader
architectural changes belong to dedicated waves.

---

## 4. Capability × Question × Block Matrix

| Capability | Questions | Block | Stage |
|---|---|---|---|
| C1: Canonical venue model | SEG-Q1 | B1 | S391 |
| C2: Spot adapter | SEG-Q2, SEG-Q3 | B2 | S392 |
| C3: Config enablement | SEG-Q4, SEG-Q5, SEG-Q9 | B3 | S393 |
| C4: Compose isolation | SEG-Q6, SEG-Q7, SEG-Q8 | B4 | S394 |
| C5: Mainnet extensibility | SEG-Q10 | B5 | S395 |
