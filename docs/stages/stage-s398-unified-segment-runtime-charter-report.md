# S398 -- Unified Segment Runtime Foundation Wave Charter Report

**Stage:** S398
**Type:** Charter and scope freeze (wave opening)
**Date:** 2026-03-22
**Wave:** Unified Segment Runtime Foundation (S398--S403)
**Predecessor:** S397 (Spot ingest binding seed -- complete)

---

## 1. Executive Summary

S398 opens the Unified Segment Runtime Foundation Wave. The wave addresses
four structural debts inherited from the Binance Segmentation Foundation
Wave (S390--S395), which proved segment isolation through a multi-binary,
multi-config, multi-compose approach. That approach was correct for proving
isolation but creates compounding maintenance cost:

1. **Sequential seed semantics** -- one binding set per activation.
2. **One config per segment** -- duplicated pipeline/log/HTTP/NATS settings.
3. **One compose overlay per segment** -- operator must choose one segment.
4. **Distributed source routing** -- leakage risk grows with each consumer.

This wave resolves all four by establishing a unified runtime where a single
execute binary, single config, and single compose stack support multiple
Binance market segments concurrently with segment isolation preserved at
runtime through source-based intent routing and NATS subject filtering.

The wave is organized into five blocks (S399--S403) with strict ordering,
12 governing questions, 10 capability targets, and 15 frozen non-goals.
Scope is frozen. The direction is explicit. The Testnet Venue Execution
Proof stages (originally S399--S401 in S396's plan) are renumbered to
S404+ and will resume on the unified runtime.

---

## 2. Deliverables

| # | Deliverable | Path | Status |
|---|---|---|---|
| D1 | Wave charter and scope freeze | [`../architecture/unified-segment-runtime-wave-charter-and-scope-freeze.md`](../architecture/unified-segment-runtime-wave-charter-and-scope-freeze.md) | Complete |
| D2 | Capabilities, questions, and non-goals | [`../architecture/unified-segment-runtime-capabilities-questions-and-non-goals.md`](../architecture/unified-segment-runtime-capabilities-questions-and-non-goals.md) | Complete |
| D3 | Stage report (this document) | (this file) | Complete |

---

## 3. Problem Analysis

### 3.1 Structural Debts Addressed

| Debt | Origin | Current cost | Wave resolution |
|---|---|---|---|
| D1: Sequential seed | `seed-configctl.sh` one source per run | Two manual seed commands for dual-segment | Merged seed in B2 (S400) |
| D2: Config duplication | `execute-spot.jsonc` + `execute-futures.jsonc` | Identical pipeline/log/HTTP/NATS in both | Unified segment-indexed config in B1 (S399) |
| D3: Compose overlay per segment | `docker-compose.spot.yaml` + `docker-compose.futures.yaml` | Operator chooses one; no single-command dual | Single compose in B4 (S402) |
| D4: Distributed source routing | `websocket_actor.go` Source switch | No centralized dispatch; leakage scales linearly | `VenueAdapterRouter` in B2 (S400), hardened in B3 (S401) |

### 3.2 What Is NOT a Problem

- **Ingest routing**: Already source-aware after S397. No change needed.
- **Adapter implementations**: `binancef` and `binances` adapters are stable. No functional change.
- **DryRunSubmitter**: Fail-closed semantics are preserved. No change needed.
- **Activation surface**: Will be extended (per-segment reporting) but not redesigned.
- **NATS subject convention**: Source values (`binancef`, `binances`) are stable. Subjects unchanged.

### 3.3 Architectural Decision: Segment Array, Not Multi-Binary

The S390 charter chose multi-binary per segment to avoid coupling. This wave
reverses that decision for the execute binary only, because:

1. **Config duplication exceeds isolation benefit.** 90% of config is
   segment-agnostic (pipeline, log, HTTP, NATS).
2. **Compose complexity grows linearly.** Each new segment requires a new
   overlay, new service entry, new port mapping.
3. **Intent routing is already source-indexed.** The intent's `Source` field
   naturally selects the adapter. Multi-binary adds no safety.
4. **Single-binary multi-adapter is proven at concept.** The decorator
   pipeline is per-adapter. DryRunSubmitter wraps each independently.

The ingest binary is NOT unified in this wave (NG-9) because its
source-aware routing is already functional and structurally different.

---

## 4. Wave Blocks (Ordered)

| Block | Stage | Title | Depends on |
|---|---|---|---|
| B1 | S399 | Unified config model and segment enablement | S398 |
| B2 | S400 | Binding merge and multi-segment runtime projection | S399 |
| B3 | S401 | Segment-safe routing and leakage hardening | S400 |
| B4 | S402 | Single-compose coexistence proof | S401 |
| B5 | S403 | Evidence gate | S402 |

---

## 5. Governing Questions Summary

| ID | Question | Block |
|---|---|---|
| USR-Q1 | Single config for multiple segments? | B1 |
| USR-Q2 | Validation rejects bad multi-segment configs? | B1 |
| USR-Q3 | Legacy single-segment configs still work? | B1 |
| USR-Q4 | Single seed for all segments? | B2 |
| USR-Q5 | Multi-adapter boot from one config? | B2 |
| USR-Q6 | Intent routes to correct adapter by source? | B2 |
| USR-Q7 | Unknown source rejected fail-closed? | B2, B3 |
| USR-Q8 | No cross-segment intent delivery? | B3 |
| USR-Q9 | NATS subject filtering per segment? | B3 |
| USR-Q10 | Both segments in single compose? | B4 |
| USR-Q11 | Global dry_run preserved? | B1, B4 |
| USR-Q12 | Per-segment overrides still valid? | B4 |

Full detail in D2 (capabilities and questions document).

---

## 6. Non-Goals Summary

15 frozen exclusions. Key items:

- **NG-1:** No separate compose per segment as permanent model.
- **NG-2:** No separate config per segment as permanent model.
- **NG-3:** No multi-exchange.
- **NG-6:** No mainnet.
- **NG-7:** No per-segment dry_run toggle.
- **NG-13:** No platform-wide actor topology redesign.
- **NG-15:** No real trading.

Full enumeration in D2.

---

## 7. S395 Gap Disposition

| S395 Gap | Before S398 | S398 disposition |
|---|---|---|
| G1: Concurrent multi-instance compose | Open | Resolved by B4 (S402) -- single-compose coexistence |
| G2: Per-segment control gate | Open | Non-goal (NG-7 scope; operational refinement) |
| G3: Spot ingest not seeded | **CLOSED (S397)** | -- |
| G4: Activation surface segment query | Open | Partial: health reports both segments (B4); full query is future |
| G5: Shared core extraction | Open | Non-goal for this wave |

---

## 8. S397 Limitation Disposition

| S397 Limitation | Before S398 | S398 disposition |
|---|---|---|
| Sequential seed semantics | Open | Resolved by B2 (S400) -- merged seed |
| Cross-segment intent leakage risk | Open | Resolved by B3 (S401) -- subject filtering + invariant tests |
| Spot WS URL hardcoded | Open | Non-goal (ingest concern, NG-9) |
| binancef/binances duplication | Open | Non-goal for this wave (NG in S390 preserved) |

---

## 9. Impact on Testnet Venue Execution Wave

The S396 charter planned:

| S396 plan | S398 disposition |
|---|---|
| S398: Dual-instance compose proof | **Replaced by this charter** |
| S399: Acceptance/fill/rejection proof | Renumbered to **S404** (post-wave) |
| S400: OMS read-path + E2E | Renumbered to **S405** (post-wave) |
| S401: Evidence gate | Renumbered to **S406** (post-wave) |

The 12 testnet venue governing questions (TV-Q1--TV-Q12) remain valid and
will be answered on the unified runtime. The Spot-first strategy is preserved.

---

## 10. Preparation Recommended for S399

Before S399 begins:

1. **Read current `VenueConfig` and `SegmentConfig`** in
   `internal/shared/settings/schema.go`. The refactor point is the
   `Segments *SegmentConfig` field and the `venue.type` single-value model.

2. **Design the `VenueSegmentEntry` type.** Fields: `enabled bool`,
   `type VenueType`, `source string`. The `Segments` field becomes
   `map[MarketSegment]VenueSegmentEntry`.

3. **Plan backward compatibility.** Legacy configs with `venue.type` (no
   `segments` map) must parse into a single-segment entry. Emit deprecation
   log. Do not break existing configs.

4. **Draft unified `execute.jsonc`.** Both segments enabled, single file,
   no overlay required.

5. **Inventory all validation rules** in `VenueConfig.Validate()` and
   `validateSegmentEnablement()` that reference the current model. These
   need migration.

---

## 11. Verdict

**S398 COMPLETE.** The Unified Segment Runtime Foundation Wave is formally
opened with frozen scope. Five blocks (S399--S403) are ordered. Twelve
governing questions are registered. Fifteen non-goals are frozen. The
architectural direction -- single binary, single config, single compose,
segment isolation via source-based routing -- is explicit and locked.

---

## 12. References

| Reference | Link |
|---|---|
| Wave charter | [`../architecture/unified-segment-runtime-wave-charter-and-scope-freeze.md`](../architecture/unified-segment-runtime-wave-charter-and-scope-freeze.md) |
| Capabilities and non-goals | [`../architecture/unified-segment-runtime-capabilities-questions-and-non-goals.md`](../architecture/unified-segment-runtime-capabilities-questions-and-non-goals.md) |
| S397 report | [`stage-s397-spot-ingest-binding-seed-report.md`](stage-s397-spot-ingest-binding-seed-report.md) |
| S396 charter refresh | [`stage-s396-testnet-venue-execution-charter-refresh-report.md`](stage-s396-testnet-venue-execution-charter-refresh-report.md) |
| S395 evidence gate | [`stage-s395-binance-segmentation-evidence-gate-report.md`](stage-s395-binance-segmentation-evidence-gate-report.md) |
| Segmentation wave charter | [`../architecture/binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md`](../architecture/binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md) |
| Stage INDEX | [`INDEX.md`](INDEX.md) |
