# Stage S33 — Governance Hygiene and Alignment

> **Status:** Complete
> **Date:** 2026-03-17
> **Objective:** Eliminate drift between docs, ownership, CLI, registries, and runtime real.
> **Scope:** Governance alignment only — no runtime changes, no new domain.

---

## Executive Summary

S33 closed the governance gap identified by the S32 stream mesh readiness review. Six governance documents were updated to reflect the actual runtime state (3 evidence families: candle, tradeburst, volume). raccoon-cli received updated topology rules, query subject coverage, and premature domain entry guards. The signal domain remains explicitly blocked.

---

## Drifts Corrected

### 1. actor-ownership.md — HIGH severity

**Before:** Derive section showed only SamplerActor (candle). Store section showed only EvidenceConsumerActor and CandleProjectionActor with stale `store-evidence` durable. Cross-binary matrix incorrectly listed store as OBSERVATION_EVENTS consumer. Control plane matrix missing tradeburst/volume/history queries.

**After:** Derive lists all 3 sampler families (SamplerActor, TradeBurstSamplerActor, VolumeSamplerActor) plus BindingWatcherActor. Store lists all 3 consumer/projection pairs with correct durable names. Cross-binary matrix corrected. Control plane matrix lists all 4 active query subjects. Open decisions updated (AO-2 resolved).

### 2. stream-ownership-matrix.md — MEDIUM severity

**Before:** Missing VolumeConsumerActor in EVIDENCE_EVENTS consumers, missing VOLUME_LATEST KV bucket, missing volume query subject, missing store-volume durable in consumer cursor summary.

**After:** All volume entries added across all 5 sections of the matrix.

### 3. stream-family-catalog.md — LOW severity

**Before:** CF-07 (evidence.volume) marked as "Planned" with minimal placeholder content.

**After:** CF-07 updated to "Current" with full event types, durable consumers, KV projections, query surface, HTTP endpoints, actor ownership chain, and domain fields — matching the CF-03/CF-04 format exactly.

### 4. projection-family-matrix.md — LOW severity

**Before:** P-03 (Volume) listed under "Planned Projections" with expected values. Supervisor tree missing volume actors. Health model missing volume trackers.

**After:** P-03 moved to current projections with full detail. Supervisor tree, health model, cross-reference, and growth projections all updated.

### 5. stream-families.md — LOW severity

**Before:** Evidence section listed only 2 consumers (`store-candle`, `store-trade-burst`), 2 event types, and 3 KV projections.

**After:** Added `store-volume` consumer, `volume.sampled` event type, and `VOLUME_LATEST` KV projection.

### 6. raccoon-cli — HIGH severity (6 files)

| File | Change |
|------|--------|
| `topology.rs` | `EXPECTED_DURABLES`: `store-evidence` → `store-candle`, `store-trade-burst`, `store-volume` |
| `topology.rs` | `EXPECTED_SUBJECT_PREFIXES`: added tradeburst and volume event/query prefixes |
| `topology.rs` | `check_pipeline_continuity`: updated to check per-type store durables instead of generic `store-evidence` |
| `topology.rs` | Added premature entry guards for SIGNAL_EVENTS and PROJECTION_EVENTS |
| `topology.rs` | All test helpers updated to match new constants |
| `runtime_bindings.rs` | `EXPECTED_DURABLES`: `store-evidence` → 3 per-type durables |
| `runtime_bindings.rs` | `EXPECTED_QUERY_SUBJECTS`: added candle.history, tradeburst.latest, volume.latest |
| `runtime_bindings.rs` | Evidence events description updated to mention all 3 types |
| `runtime_bindings.rs` | All test helpers updated |
| `runtime_bindings/source.rs` | Doc comment updated with current durable and query subject names |
| `drift_detect.rs` | `ARCH_DOCS` expanded from 3 to 8 documents |
| `drift_detect.rs` | Added `PROHIBITED_STREAMS` constant for signal/projection guards |
| `drift_detect.rs` | Added `check_premature_domain_entry` function |

---

## New Deliverables

| File | Purpose |
|------|---------|
| `docs/architecture/governance-hygiene-status.md` | Living governance status dashboard with evidence family inventory, signal prerequisites, and remaining gaps |
| `docs/stages/stage-s33-governance-hygiene-report.md` | This report |

---

## Files Changed

### Governance Documents (6 files)
- `docs/architecture/actor-ownership.md`
- `docs/architecture/stream-ownership-matrix.md`
- `docs/architecture/stream-family-catalog.md`
- `docs/architecture/projection-family-matrix.md`
- `docs/architecture/stream-families.md`
- `docs/architecture/governance-hygiene-status.md` (new)

### raccoon-cli (4 files)
- `tools/raccoon-cli/src/analyzers/topology.rs`
- `tools/raccoon-cli/src/analyzers/runtime_bindings.rs`
- `tools/raccoon-cli/src/analyzers/runtime_bindings/source.rs`
- `tools/raccoon-cli/src/analyzers/drift_detect.rs`

---

## Remaining Gaps

| # | Gap | Severity | Recommendation |
|---|-----|----------|----------------|
| 1 | Config-driven activation incomplete | MEDIUM | Store spawns all projections unconditionally. Wire BindingWatcher for dynamic projection lifecycle before signal entry. |
| 2 | QueryResponderActor scales manually | LOW | Manageable at 3 types. Consider refactoring to per-type responders before the 5th evidence type. |
| 3 | No projection lag metric | LOW | Nice-to-have for operations. Not a governance blocker. |
| 4 | Single exchange adapter | LOW | Only binancef implemented. Pattern supports multi-source but untested. |
| 5 | raccoon-cli single-writer validation | LOW | Currently enforced by review only. Could add static analysis. |

---

## Preparation for S34

S33 unblocks the following S34 candidates (in priority order):

1. **evidence.stats (new evidence family)** — All patterns proven, governance up to date, follows checklist. Pure evidence addition, no new domain.
2. **Config-driven activation hardening** — Wire BindingWatcher lifecycle for store projections. Blocks signal entry.
3. **Historical projections for tradeburst/volume** — Add TRADE_BURST_HISTORY and VOLUME_HISTORY buckets following candle pattern.

**Signal entry remains blocked** until config-driven activation is proven (gap #1 above). This is enforced by:
- `governance-hygiene-status.md` prerequisites table
- `signal-readiness-review.md` gap #1
- raccoon-cli premature-entry guards (will fail CI if SIGNAL_EVENTS appears in code)

---

## Guard Rails Compliance

| Guard Rail | Complied? |
|-----------|----------|
| No signal implementation | Yes — premature entry guards added, no signal code |
| No new domain opened | Yes — only governance/documentation changes |
| No broad runtime refactor | Yes — zero Go code changes |
| No superficial text masking | Yes — every drift traced to specific runtime evidence |
| Remaining limits registered | Yes — 5 gaps documented in governance-hygiene-status.md |
