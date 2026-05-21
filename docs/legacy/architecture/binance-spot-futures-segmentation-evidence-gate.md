# Binance Spot/Futures Segmentation Foundation -- Evidence Gate

**Wave:** Binance Spot/Futures Segmentation Foundation (S390--S395)
**Gate stage:** S395
**Date:** 2026-03-22
**Charter:** S390
**Blocks evaluated:** B1 (S391), B2 (S392), B3 (S393), B4 (S394)

---

## 1. Gate Purpose

This document is the formal evidence gate for the Binance Spot/Futures
Segmentation Foundation Wave. It evaluates whether S390--S394 delivered the five
chartered capability targets with sufficient evidence to close the wave and
authorize the next strategic ceremony.

The gate answers one question: **Did the wave produce a robust, segmented
Binance architecture that is ready to underpin the Testnet Venue Execution Proof
Wave?**

---

## 2. Governing Questions Verdict

Each governing question (SEG-Q1 through SEG-Q10) is evaluated against the
evidence produced by its target stage.

### SEG-Q1: Venue Model Orthogonality -- FULL

> Does the venue model cleanly separate exchange, market segment, and
> environment as orthogonal dimensions?

**Evidence:**
- `MarketSegment` type with `spot` and `futures` constants in `schema.go:257-261`.
- `VenueType.Segment()` method maps each type to its implied segment (`schema.go:266-275`).
- `VenueType.RequiresSegmentConfig()` discriminates paper from segment-bearing types (`schema.go:279-281`).
- Flat `VenueType` string preserved for config backward compatibility; decomposition is internal.
- 13 invariants defined (INV-1 through INV-13) in S391 venue model document.

**Judgment: FULL** -- Types exist, are used by config validation and adapter
selection, and are independently testable. The model separates identity
(exchange x segment x environment) from execution mode.

### SEG-Q2: Adapter Extensibility -- FULL

> Can a new market segment (Spot) be added without modifying existing adapter
> code?

**Evidence:**
- `BinanceSpotTestnetAdapter` implemented in a new file (`binance_spot_testnet_adapter.go`, 373 lines).
- `BinanceFuturesTestnetAdapter` has zero diff from this wave.
- Spot adapter implements `VenuePort` and `VenueQueryPort` identically.
- Factory dispatch in `buildVenueAdapter` (`run.go:173-182`) adds a new case without modifying existing cases.

**Judgment: FULL** -- Futures adapter code untouched. Spot added as pure
extension. Shared core extraction was not implemented (deferred), but the
adapter boundary is clean.

### SEG-Q3: Spot Adapter Correctness -- FULL

> Does BinanceSpotTestnetAdapter satisfy VenuePort and VenueQueryPort with
> correct Spot-specific endpoint and response mapping?

**Evidence:**
- 7 unit tests in `binance_spot_testnet_adapter_test.go`:
  - Filled order with venue_order_id, status, fill record.
  - Multi-fill aggregation (3 legs -> 1 aggregated fill, weighted average price, total fee).
  - No-action intent (SideNone -> Accepted, zero HTTP requests).
  - Auth error (HTTP 401 -> InvalidArgument, not retryable).
  - API path verification (`/api/v3/order`).
  - Simulated flag (`false` for real venue fills).
  - Client order ID propagation.
- Base URL: `testnet.binance.vision`.
- Response type: `FULL` with `fills[]` array parsing.
- Weighted average price from per-leg fills.
- Fee: sum of per-leg commission values.

**Judgment: FULL** -- All unit tests pass. Field mapping validated against Spot
API schema.

### SEG-Q4: Config Validation Rigor -- FULL

> Does config validation reject invalid type/segment combinations at startup?

**Evidence:**
- 25 test cases in `s393_segment_enablement_test.go` covering:
  - 3 VenueType.Segment() identity tests.
  - 1 RequiresSegmentConfig() test.
  - 5 SegmentConfig fail-closed tests (nil, absent, true, false, mixed).
  - 9 VenueConfig validation tests (paper, futures, spot, missing, cross-segment).
  - 3 dry_run preservation tests.
- `validateSegmentEnablement()` in `schema.go:426-456` rejects:
  - Missing segments block for segment-requiring types.
  - Disabled segment for the active venue type.
  - Cross-segment mismatch (futures type with only spot_enabled).
  - Paper with any segment enabled.

**Judgment: FULL** -- All negative cases covered. Fail-closed semantics enforced
at startup.

### SEG-Q5: Fail-Closed Preservation -- FULL

> Does dry_run=true remain the default for all new venue types?

**Evidence:**
- `IsDryRun()` returns `true` when `DryRun` pointer is nil (`schema.go:461-463`).
- Both segmented configs (`execute-futures.jsonc`, `execute-spot.jsonc`) set `dry_run: true`.
- DryRunSubmitter wraps both Spot and Futures adapters identically in `run.go`.
- Tests verify dry_run=false on paper is rejected; absent dry_run defaults to true.

**Judgment: FULL** -- Fail-closed dry_run preserved for all venue types.

### SEG-Q6: Stream/KV Isolation -- SUBSTANTIAL

> Can Spot and Futures execute binaries run concurrently in compose without
> stream/KV collision?

**Evidence:**
- NATS source convention: `binancef` (Futures), `binances` (Spot) -- defined in S391, used in adapter identity.
- Compose overrides (`docker-compose.futures.yaml`, `docker-compose.spot.yaml`) isolate config and credentials per segment.
- Smoke script (`smoke-segmented-compose.sh`) defines 6-phase validation including segment isolation check.
- 7 structural tests in `s394_segmented_compose_test.go` prove config isolation.
- Segment identity logged at startup (`segment=futures` / `segment=spot`).

**Gap:** Concurrent execution of both segments in a single stack was not proven
in smoke. The smoke script swaps one segment at a time (sequential, not
concurrent). Multi-instance compose (two execute services) is documented as
L1 limitation.

**Judgment: SUBSTANTIAL** -- Config-level isolation proven; NATS subject
convention defined; concurrent runtime isolation designed but not
smoke-exercised.

### SEG-Q7: Activation Surface Segment Awareness -- SUBSTANTIAL

> Are activation dimensions segment-aware and independently observable?

**Evidence:**
- Segment identity logged at startup in `run.go:79-84`.
- Each compose instance reports its segment in structured logs.
- Activation surface inherits adapter identity (Futures or Spot).

**Gap:** No dedicated HTTP endpoint or KV key exposing segment as an observable
activation dimension. Observability is via logs only.

**Judgment: SUBSTANTIAL** -- Segment logged at startup; not yet queryable via
API or activation surface KV.

### SEG-Q8: Independent Gate Control -- PARTIAL

> Does the control gate operate independently per segment instance?

**Evidence:**
- Each compose instance has its own process and responds to its own config.
- Control gate design is global (single KV key) per L5 in S394.
- Halting one segment halts both in the current KV design.

**Gap:** Per-segment control gate not implemented. Acknowledged as a limitation
in S391, S394.

**Judgment: PARTIAL** -- Process-level isolation exists (separate binaries), but
gate control is global. Per-segment gate deferred to future stage.

### SEG-Q9: Credential Isolation -- FULL

> Is credential isolation between segments enforced?

**Evidence:**
- Futures credentials: `MF_VENUE_BINANCE_FUTURES_TESTNET_{API_KEY,API_SECRET}` (`run.go:166-167`).
- Spot credentials: `MF_VENUE_BINANCE_SPOT_TESTNET_{API_KEY,API_SECRET}` (`run.go:176-177`).
- Compose overrides bind segment-specific env vars only.
- `LoadCredentials()` returns error for missing env vars, caught at startup.

**Judgment: FULL** -- Credential namespaces isolated per segment. Enforced by
convention and startup validation.

### SEG-Q10: Mainnet Extensibility -- FULL (structural)

> Can the segmented architecture be extended to mainnet types without structural
> changes?

**Evidence:**
- Adding `binance_futures_mainnet` requires only:
  - New `VenueType` constant and `knownVenueTypes` entry.
  - New `MarketSegment` mapping (reuses `futures`).
  - New config file with appropriate segments block.
  - New compose override with mainnet credentials.
  - New adapter file targeting production base URL.
  - New case in `buildVenueAdapter` switch.
- No domain model, pipeline, port interface, or event type changes required.
- Invariants INV-8 (testnet creds never reach mainnet) and INV-13 (mainnet not
  registered until ceremony) provide safety rails.

**Judgment: FULL (structural)** -- Extensibility proven by design analysis. No
mainnet code written (correct per NG-1).

---

## 3. Capability Classification

| ID | Capability | Target | Classification | Evidence |
|---|---|---|---|---|
| C1 | Canonical venue model with segment dimension | SEG-Q1 | **FULL** | Types, methods, invariants, tests |
| C2 | Binance Spot testnet adapter | SEG-Q2, Q3 | **FULL** | Adapter impl + 7 tests + zero Futures diff |
| C3 | Config-driven segment enablement | SEG-Q4, Q5, Q9 | **FULL** | 25 validation tests + fail-closed semantics |
| C4 | Compose-level segment isolation | SEG-Q6, Q7, Q8 | **SUBSTANTIAL** | Config isolation + smoke design; concurrent runtime and per-segment gate are gaps |
| C5 | Mainnet extensibility proof | SEG-Q10 | **FULL (structural)** | Design analysis; no structural changes needed |

---

## 4. Regression Verification

| Prior capability | Wave | Regression? | Evidence |
|---|---|---|---|
| Paper simulator adapter | Foundation | No | `paper_venue_adapter.go` zero diff; `run.go` case unchanged |
| BinanceFuturesTestnetAdapter | S308-S310 | No | Zero diff on adapter file; factory case unchanged |
| DryRunSubmitter fail-closed | S379 | No | `dry_run_submitter.go` zero diff; wraps both adapters |
| Lifecycle state machine (7 states) | S382-S388 | No | `events.go` zero diff; event types preserved |
| PriceSource wiring | S387 | No | Integration in `run.go` preserved |
| Rejection event path | S386 | No | Event types, consumer, projection actor intact |
| Multi-binary compose orchestration | S370-S375 | No | Structural tests unchanged |
| Config validation framework | S327-S331 | No | Existing validation logic preserved; segment validation additive |
| Activation surface | S337-S346 | No | No changes to activation model |
| Go build | All | No | `go build ./cmd/execute` compiles without error |

**Regression verdict: CLEAN** -- No regressions detected. All prior wave
capabilities remain intact.

---

## 5. Non-Goal Compliance

| ID | Non-goal | Compliant? |
|---|---|---|
| NG-1 | No mainnet execution | Yes -- testnet only |
| NG-2 | No multi-exchange | Yes -- Binance only |
| NG-3 | No full OMS | Yes -- existing lifecycle preserved |
| NG-4 | No portfolio risk | Yes -- no position tracking |
| NG-5 | No advanced order types | Yes -- market orders only |
| NG-6 | No WebSocket fills | Yes -- REST only |
| NG-7 | No multi-symbol routing | Yes -- single symbol per binary |
| NG-8 | No real trading as focus | Yes -- architecture proof, dry_run default |
| NG-9 | No ClickHouse segment columns | Yes -- no schema migrations |
| NG-10 | No margin/leverage config | Yes -- none added |
| NG-11 | No cross-segment positions | Yes -- no position tracking |
| NG-12 | No fee tier differentiation | Yes -- raw fee values only |
| NG-13 | No platform redesign | Yes -- venue segmentation only |

**Non-goal compliance: 13/13** -- All frozen exclusions respected.

---

## 6. Wave Verdict

### Classification: PASS -- Wave closes with SUBSTANTIAL evidence.

**Rationale:**

- 4 of 5 capabilities classified FULL.
- C4 (compose-level isolation) classified SUBSTANTIAL due to two specific gaps:
  concurrent multi-instance runtime not smoke-proven, and per-segment control
  gate not implemented.
- These gaps are **acceptable for wave closure** because:
  1. Config-level and process-level isolation are proven and sufficient for the
     Testnet Venue Execution Proof Wave.
  2. Per-segment gate is a correctness refinement, not a safety concern (global
     gate is more conservative, not less).
  3. Concurrent multi-instance compose is an operational deployment detail, not
     an architectural gap.
- Zero regressions.
- 13/13 non-goals respected.
- 37 new tests across 3 test files.
- 10 architecture documents internally consistent, aligned with 5 stage reports.

### The Binance Spot/Futures Segmentation Foundation Wave is formally CLOSED.

---

## 7. Recommendations for Next Ceremony

See companion document:
[`binance-segmentation-evidence-matrix-residual-gaps-and-next-ceremony.md`](binance-segmentation-evidence-matrix-residual-gaps-and-next-ceremony.md)

---

## 8. References

| Reference | Link |
|---|---|
| Wave charter | [`binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md`](binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md) |
| Capabilities and questions | [`binance-segmentation-capabilities-questions-and-non-goals.md`](binance-segmentation-capabilities-questions-and-non-goals.md) |
| Venue model refactor | [`venue-model-refactor-exchange-market-segment-environment-and-execution-mode.md`](venue-model-refactor-exchange-market-segment-environment-and-execution-mode.md) |
| Adapter boundary split | [`adapter-boundary-split-for-binance-spot-and-binance-futures.md`](adapter-boundary-split-for-binance-spot-and-binance-futures.md) |
| Config-driven enablement | [`config-driven-enablement-for-binance-spot-and-futures.md`](config-driven-enablement-for-binance-spot-and-futures.md) |
| Compose proof | [`compose-proof-with-live-listening-and-dry-run-on-segmented-binance-paths.md`](compose-proof-with-live-listening-and-dry-run-on-segmented-binance-paths.md) |
| Stage INDEX | [`../stages/INDEX.md`](../stages/INDEX.md) |
