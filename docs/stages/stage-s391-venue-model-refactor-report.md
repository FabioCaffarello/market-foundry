# Stage S391 — Venue Model Refactor: Exchange, Market Segment, Environment, and Execution Mode

**Wave:** Binance Spot/Futures Segmentation Foundation (S390–S395)
**Block:** B1
**Date:** 2026-03-22
**Predecessor:** S390 (charter and scope freeze)
**Successor:** S392 (adapter boundary split)
**Answers:** SEG-Q1 (venue model orthogonality)

---

## 1. Executive Summary

S391 decomposes the monolithic `VenueType` string into a canonical four-dimension
model: **exchange**, **market segment**, **environment**, and **execution mode**.

The current model encodes `binance_futures_testnet` as a single opaque token.
This prevents the system from reasoning about its constituent parts — it cannot
distinguish "this is Binance" from "this is Futures" from "this is testnet"
without string parsing. Adding Spot requires a new monolithic token with the
same encoding problem.

The refactored model defines three explicit identity dimensions (`Exchange`,
`MarketSegment`, `VenueEnvironment`) that compose the venue identity, plus the
existing `ActivationSurface` as the orthogonal execution mode. The flat
`VenueType` string is preserved as a backward-compatible serialization key
derived from the identity dimensions.

**This stage is modeling and documentation only — no code changes.**

---

## 2. Deliverables

| # | Deliverable | Path | Status |
|---|---|---|---|
| D1 | Canonical venue model document | [`../architecture/venue-model-refactor-exchange-market-segment-environment-and-execution-mode.md`](../architecture/venue-model-refactor-exchange-market-segment-environment-and-execution-mode.md) | Complete |
| D2 | Valid/invalid configurations and fail-closed rules | [`../architecture/venue-segmentation-semantics-valid-invalid-configurations-and-fail-closed-rules.md`](../architecture/venue-segmentation-semantics-valid-invalid-configurations-and-fail-closed-rules.md) | Complete |
| D3 | This stage report | (this file) | Complete |

---

## 3. Model Design Summary

### 3.1 Four Dimensions

| Dimension | Type | Active values | Role |
|---|---|---|---|
| Exchange | `Exchange` | `binance` | Selects counterparty platform |
| Market segment | `MarketSegment` | `spot`, `futures` | Selects product class and API endpoints |
| Environment | `VenueEnvironment` | `testnet` (mainnet defined, not activated) | Selects infra tier and risk classification |
| Execution mode | `EffectiveMode` (existing) | `paper`, `venue_halted`, `venue_live`, `venue_degraded` | Controls whether orders reach the venue |

### 3.2 Identity vs Mode Separation

Venue identity (exchange × segment × environment) answers: **which venue adapter
do we instantiate?**

Execution mode (adapter × gate × credentials) answers: **is this adapter allowed
to submit orders right now?**

These are orthogonal. A `binance_futures_testnet` adapter can be in dry-run mode,
halted mode, or live mode — the identity does not change.

### 3.3 Backward Compatibility

The flat `VenueType` string (`"binance_futures_testnet"`) remains in config
files. The model decomposes it internally into dimensions for validation. No
config file changes are required by S391.

---

## 4. Valid/Invalid Configuration Summary

### 4.1 Active Configurations

| VenueType | dry_run | Result |
|---|---|---|
| `paper_simulator` | true (forced) | Paper simulation |
| `binance_futures_testnet` | true | Dry-run on Futures adapter |
| `binance_futures_testnet` | false | Live Futures execution |
| `binance_spot_testnet` | true | Dry-run on Spot adapter (after S392) |
| `binance_spot_testnet` | false | Live Spot execution (after S392) |

### 4.2 Rejected Configurations

| Scenario | Rejection |
|---|---|
| `paper_simulator` + `dry_run=false` | Startup validation error |
| Unknown VenueType | Startup validation error |
| Missing credentials for non-paper venue | Startup error |
| Mainnet types (not registered) | Startup validation error |
| Incomplete dimensions (exchange without segment) | Startup validation error |

### 4.3 Fail-Closed Defaults

| Missing value | Default |
|---|---|
| `venue.type` | `paper_simulator` |
| `venue.dry_run` | `true` |
| Control gate state | `active` (safe because dry_run defaults true) |

---

## 5. Invariants Established

| ID | Invariant |
|---|---|
| INV-1 | Non-paper VenueType decomposes into exactly three non-empty dimensions |
| INV-2 | Each dimension value belongs to its known set |
| INV-3 | Composed VenueType must be in `knownVenueTypes` registry |
| INV-4 | Paper has no exchange, segment, or environment |
| INV-5 | Each binary serves exactly one segment |
| INV-6 | Credentials namespaced by full identity |
| INV-7 | NATS source values distinct per segment (`binances`, `binancef`) |
| INV-8 | Testnet credentials never reach mainnet endpoints |
| INV-9 | `dry_run` defaults to true when omitted |
| INV-10 | `dry_run=false` with paper is rejected |
| INV-11 | Unregistered types fail at startup |
| INV-12 | Missing credentials fail at startup |
| INV-13 | Mainnet types not registered until activation ceremony |

---

## 6. Limitations and Residual Gaps

| # | Limitation | Impact | Resolution path |
|---|---|---|---|
| L1 | Model is documentation-only; no code changes in S391 | Dimensions exist as design spec, not compiled types | S392 introduces domain types alongside Spot adapter |
| L2 | Config still uses flat `type` string | Internal decomposition only; no explicit `exchange`/`segment`/`environment` fields in JSON | S393 evaluates whether explicit config fields add value |
| L3 | Control gate is global, not per-segment | Halting one segment halts both in current KV design | Future stage when per-segment halt is operationally needed |
| L4 | Only Binance exchange defined | Model supports multi-exchange structurally but only proves one | Future wave |
| L5 | Mainnet not activated | Types defined in the model but not in `knownVenueTypes` | Requires fund-safety activation ceremony |
| L6 | Source value convention is segment-only | Sufficient for single-exchange deployments; would need extension for multi-exchange | Acceptable for current scope |

---

## 7. SEG-Q1 Answer: Venue Model Orthogonality

**Question (SEG-Q1):** Does the venue model decompose into orthogonal
dimensions (exchange, segment, environment) that can be validated independently?

**Answer:** Yes. The model defines three identity dimensions (`Exchange`,
`MarketSegment`, `VenueEnvironment`) that are:

1. **Independently typed** — each has its own Go type and known-value set;
2. **Independently validated** — each dimension is checked against its allowed values;
3. **Composable** — the flat `VenueType` is derivable from the three dimensions;
4. **Orthogonal to execution mode** — `ActivationSurface` governs execution
   independently of venue identity.

The paper simulator is the zero-value identity (no dimensions set), which
preserves backward compatibility without special-casing.

**Evidence:**
- Complete dimension taxonomy in D1 (sections 2–3)
- Exhaustive validity matrix in D2 (section 3)
- Fail-closed rules covering all ambiguous cases in D2 (section 5)
- Invariant registry covering identity, isolation, and safety in D2 (section 6)

---

## 8. Preparation for S392

S392 (Adapter Boundary Split) should:

1. **Introduce domain types** — `Exchange`, `MarketSegment`, `VenueEnvironment`,
   `VenueIdentity` in `internal/domain/execution/`;
2. **Implement `BinanceSpotTestnetAdapter`** — targeting `testnet.binance.vision/api/v3/order`;
3. **Extract common Binance HTTP infrastructure** — HMAC signing, timestamp,
   error classification to `binance_common.go`;
4. **Register `binance_spot_testnet`** in `knownVenueTypes`;
5. **Add `buildVenueAdapter` case** for `VenueTypeBinanceSpotTestnet`;
6. **Unit test** the Spot adapter independently (same coverage pattern as Futures);
7. **Verify INV-2 through INV-7** with compilation and test evidence.

The Spot adapter must satisfy the same `VenuePort` and `VenueQueryPort`
contracts as the Futures adapter. The DryRunSubmitter, RetrySubmitter, and
Post200Reconciler decorators apply identically to either adapter.

---

## 9. References

| Reference | Link |
|---|---|
| S390 charter | [`stage-s390-binance-segmentation-charter-report.md`](stage-s390-binance-segmentation-charter-report.md) |
| Wave charter | [`../architecture/binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md`](../architecture/binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md) |
| Governing questions | [`../architecture/binance-segmentation-capabilities-questions-and-non-goals.md`](../architecture/binance-segmentation-capabilities-questions-and-non-goals.md) |
| Current venue schema | `internal/shared/settings/schema.go` (lines 237–340) |
| Current activation model | `internal/domain/execution/activation.go` |
| Current adapter instantiation | `cmd/execute/run.go` (lines 148–174) |
