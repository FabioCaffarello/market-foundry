# S390 — Binance Spot/Futures Segmentation Foundation Charter Report

**Stage:** S390
**Type:** Charter and scope freeze
**Date:** 2026-03-22
**Wave:** Binance Spot/Futures Segmentation Foundation
**Predecessor:** S389 (Testnet Venue Execution Proof — Charter opened)

---

## 1. Executive Summary

Stage S390 formally opens the **Binance Spot/Futures Segmentation Foundation
Wave** with frozen scope, 10 governing questions, 5 capability targets, 13
explicit non-goals, and a 5-block execution plan.

The Testnet Venue Execution Proof Wave (S389) opened with the
`BinanceFuturesTestnetAdapter` as the only real venue adapter. Analysis of the
codebase revealed that the venue model, config schema, credential convention,
NATS subject tree, and compose profile all assume a single Binance product
(Futures). Adding Spot as a flat enum value would create a fragile,
non-extensible binding.

**Decision:** Before executing real venue proofs, the platform must segment
Binance Spot and Binance Futures as architecturally independent capabilities —
each with its own adapter, credentials, config, streams, and compose service
instance.

This wave uses the multi-binary orchestration foundation (S370–S375) to run
segment-specific execute binaries concurrently, proving isolation without
introducing a venue routing layer.

---

## 2. Problem Redefinition

### 2.1 Why Segmentation Before Venue Proof

The S389 charter defines 12 governing questions (TV-Q1–TV-Q12) about real
venue behavior. Answering them on a single-product architecture would:

1. **Lock Spot out.** Stream subjects, KV keys, and credential env vars would
   be Futures-specific with no Spot equivalent.
2. **Force a retrofit.** Adding Spot later would require renaming subjects,
   migrating KV keys, and updating compose profiles — breaking evidence
   collected during the Futures-only wave.
3. **Obscure adapter boundaries.** A single adapter handling both products
   would accumulate conditional branching that violates the port contract's
   simplicity.

Segmentation now avoids all three costs.

### 2.2 Architectural Approach

**Multi-binary per segment** — each segment runs as an independent execute
binary instance. This reuses the proven compose orchestration pattern without
introducing intra-binary routing complexity.

| Property | Value |
|---|---|
| Adapter per segment | 1:1 (adapter ↔ segment) |
| Binary per segment | 1:1 (compose service ↔ segment) |
| Credentials per segment | Isolated env var sets |
| NATS source per segment | `binancef` (Futures), `binances` (Spot) |
| Control gate per segment | Independent per binary instance |
| Activation surface | Segment-tagged per instance |

---

## 3. Wave Governance

### 3.1 Governing Questions

| ID | Question | Target |
|---|---|---|
| SEG-Q1 | Does the venue model cleanly separate exchange, segment, and environment? | S391 |
| SEG-Q2 | Can Spot be added without modifying Futures adapter code? | S392 |
| SEG-Q3 | Does BinanceSpotTestnetAdapter satisfy VenuePort/VenueQueryPort correctly? | S392 |
| SEG-Q4 | Does config validation reject invalid type/segment combinations? | S393 |
| SEG-Q5 | Does dry_run=true remain default for all new venue types? | S393 |
| SEG-Q6 | Can Spot and Futures binaries run concurrently without stream/KV collision? | S394 |
| SEG-Q7 | Are activation dimensions segment-aware and independently observable? | S394 |
| SEG-Q8 | Does the control gate operate independently per segment? | S394 |
| SEG-Q9 | Is credential isolation enforced between segments? | S393 |
| SEG-Q10 | Can the architecture extend to mainnet types without structural changes? | S395 |

### 3.2 Capability Targets

| ID | Capability | Block |
|---|---|---|
| C1 | Canonical venue model with segment dimension | B1 (S391) |
| C2 | Binance Spot testnet adapter | B2 (S392) |
| C3 | Config-driven segment enablement | B3 (S393) |
| C4 | Compose-level segment isolation | B4 (S394) |
| C5 | Mainnet extensibility proof (structural) | B5 (S395) |

Full detail in companion document:
[`../architecture/binance-segmentation-capabilities-questions-and-non-goals.md`](../architecture/binance-segmentation-capabilities-questions-and-non-goals.md)

---

## 4. Non-Goals (13 Frozen Exclusions)

| ID | Exclusion | Why frozen |
|---|---|---|
| NG-1 | Mainnet execution | Fund safety, credential governance out of scope |
| NG-2 | Multi-exchange | Prove segmentation within Binance first |
| NG-3 | Full OMS | Segmentation extends existing lifecycle, not builds new |
| NG-4 | Portfolio risk | Requires position state that doesn't exist |
| NG-5 | Advanced order types | Each type has distinct API surface |
| NG-6 | WebSocket fills | Connection management orthogonal to segmentation |
| NG-7 | Multi-symbol per binary | Single-symbol model is stable |
| NG-8 | Real trading as focus | Architecture proof, not trading proof |
| NG-9 | ClickHouse segment columns | Migration governance deferred |
| NG-10 | Margin/leverage config | Futures-specific, creates risk deps |
| NG-11 | Cross-segment positions | Full OMS feature |
| NG-12 | Fee tier differentiation | Trading concern, not segmentation |
| NG-13 | Platform redesign | Wave scoped to venue segmentation only |

---

## 5. Block Sequence and Stage Order

| Order | Block | Stage | Title | Depends on |
|---|---|---|---|---|
| 1 | B1 | S391 | Venue model refactor | S390 (this charter) |
| 2 | B2 | S392 | Adapter boundary split | S391 (types available) |
| 3 | B3 | S393 | Config-driven enablement | S391 (types), S392 (adapter exists) |
| 4 | B4 | S394 | Compose proof | S392 + S393 (both segments wired) |
| 5 | B5 | S395 | Evidence gate | S394 (all evidence collected) |

**Critical path:** B1 → B2 → B3 → B4 → B5 (strictly sequential — each block
depends on the previous).

---

## 6. Relationship to Testnet Venue Execution Proof Wave

The testnet venue execution proof wave (S389) is **recalibrated**:

- S389 charter remains valid — governing questions TV-Q1–TV-Q12 unchanged.
- S390–S395 execute the segmentation foundation.
- S396+ resume the testnet venue execution proof on the segmented architecture.
- TV-Q1–TV-Q12 will be answered per segment (Spot and Futures independently).

This recalibration adds ~5 stages but prevents a costly single-product lock-in.

---

## 7. Foundation Assessment

### 7.1 Capabilities Already Proven

| Capability | Wave | Status |
|---|---|---|
| Multi-binary compose orchestration | S370–S375 | Proven |
| Venue adapter port contracts | S321–S326 | Stable |
| Decorator pipeline (retry, reconcile, dry-run) | S322, S325, S379 | Stable |
| Activation surface | S337–S346 | Stable |
| Control gate | S337–S346 | Stable |
| Lifecycle state machine (7 states) | S382–S388 | Proven (100% coverage) |
| Config validation framework | S327–S331 | Stable |
| BinanceFuturesTestnetAdapter | S308–S310 | Complete |
| PriceSource wiring | S387 | Complete |
| Rejection event path | S386 | Complete |

### 7.2 Infrastructure Ready

| Component | Reuse pattern |
|---|---|
| `VenuePort` / `VenueQueryPort` | Spot adapter implements same interfaces |
| DryRunSubmitter | Wraps any adapter (no changes needed) |
| Post200Reconciler | Works with any `VenueQueryPort` (no changes needed) |
| RetrySubmitter | Error classification shared (minor extraction) |
| Compose profile | Add new service entries (pattern proven) |
| NATS subject convention | Add `binances` source value (pattern proven) |

---

## 8. Preparation for S391

Before starting B1 (Venue Model Refactor), the following must be ready:

1. **Binance Spot testnet API documentation** reviewed for:
   - Base URL: `testnet.binance.vision/api/v3/order`
   - Authentication scheme (HMAC-SHA256, same as Futures)
   - Response schema differences (field names, nesting)
   - Error code differences (if any)

2. **Source value convention** agreed:
   - `binancef` → Binance Futures (existing)
   - `binances` → Binance Spot (new)

3. **Credential env var convention** agreed:
   - `MF_BINANCE_SPOT_TESTNET_API_KEY`
   - `MF_BINANCE_SPOT_TESTNET_API_SECRET`

4. **Domain type location** decided:
   - Proposed: `internal/domain/execution/venue.go` (new file for venue model types)

5. **Shared HTTP helper extraction** scoped:
   - HMAC signing, timestamp generation, HTTP client setup
   - Proposed: `internal/application/execution/binance_common.go`

---

## 9. Deliverables Produced

| Deliverable | Path | Status |
|---|---|---|
| Wave charter and scope freeze | `docs/architecture/binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md` | Delivered |
| Capabilities, questions, non-goals | `docs/architecture/binance-segmentation-capabilities-questions-and-non-goals.md` | Delivered |
| Stage report | `docs/stages/stage-s390-binance-segmentation-charter-report.md` | This document |

---

## 10. Verdict

**Wave status:** OPEN — scope frozen.

The Binance Spot/Futures Segmentation Foundation Wave is formally authorized.
Scope is frozen per this charter. The next stage (S391) may begin immediately.
