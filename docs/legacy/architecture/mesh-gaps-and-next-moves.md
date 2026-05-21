# Mesh Gaps and Next Moves

> Actionable gap analysis and sequenced recommendations derived from the S32 readiness review.

## Gap Inventory

### G1 — actor-ownership.md is 5 stages stale (HIGH)

**Current state:** The canonical ownership document reflects S12 state. It does not include:
- TradeBurstSamplerActor, VolumeSamplerActor in derive actor tree
- TradeBurstProjectionActor, TradeBurstConsumerActor, VolumeProjectionActor, VolumeConsumerActor in store actor tree
- FamilyProcessor pattern in derive
- ProjectionPipeline pattern in store
- Volume KV bucket (VOLUME_LATEST)
- Volume query subject and HTTP endpoint
- Corrected cross-binary matrix (store is NOT an OBSERVATION_EVENTS consumer)
- Corrected control plane matrix (missing candle history, tradeburst latest, volume latest subjects)

**Impact:** New contributors consulting this document will have an incomplete and misleading picture of the system. This is the most-referenced architecture document.

**Fix:** Full rewrite of Phase 3 sections (derive, store) and matrices. One stage of focused work.

---

### G2 — raccoon-cli topology rules stale (HIGH)

**Current state:**
- `EXPECTED_DURABLES` lists `store-evidence` (generic name that doesn't exist in code) instead of `store-candle`, `store-trade-burst`, `store-volume`
- `EXPECTED_SUBJECT_PREFIXES` only lists candle subjects; tradeburst and volume missing
- `ARCH_DOCS` lists 3 documents; 6+ new architecture documents not inventoried
- No premature-entry guards for SIGNAL_EVENTS or PROJECTION_EVENTS

**Impact:** Topology checks may reject valid code (false negatives on new durables) and miss invalid code (missing subjects go undetected). Drift detection won't catch deleted architecture documents.

**Fix:** Targeted updates to topology.rs and drift_detect.rs constants. Small scope, high impact.

---

### G3 — QueryResponderActor scales manually (MEDIUM)

**Current state:** Every new evidence type requires adding:
- 1 KV store field to the struct
- 1 KV initialization block in start()
- 1 Close() call in Stopped handler
- 1 ControlRoute entry
- 1 handler method

At 3 types (4 routes) this is manageable. At 5+ it becomes a maintenance risk.

**Impact:** No immediate impact. Becomes friction when adding the 4th and 5th evidence types.

**Fix:** Consider a data-driven approach where query routes are registered from a configuration list, similar to how ProjectionPipeline works for spawning. Not urgent — address before the 5th evidence type.

---

### G4 — Catalog documents incomplete for volume (LOW)

**Current state:** Volume was implemented in S31 but these documents still list it as "Planned":
- `stream-families.md` — CF-03 evidence section lists only candle and tradeburst
- `stream-family-catalog.md` — CF-07 marked "Planned"
- `projection-family-matrix.md` — P-03 marked "Planned"
- `stream-ownership-matrix.md` — missing volume consumer, bucket, query

**Impact:** Low — the implementation is correct regardless of doc status. But creates ambiguity about what is implemented vs. planned.

**Fix:** Update status fields and add volume entries. Mechanical change.

---

### G5 — Config-driven activation not proven (MEDIUM — signal blocker)

**Current state:** S25 identified config-driven activation as a prerequisite for signal. The current activation mechanism works for evidence (binding watcher triggers sampler creation), but signal would need a different activation path (evidence → signal, not observation → signal).

**Impact:** Does not affect current evidence types. Blocks signal domain entry.

**Fix:** Design and implement config-driven pipeline activation that supports evidence-to-signal derivation chains. This is the largest remaining gap and requires a dedicated stage.

---

### G6 — No end-to-end smoke test for volume (LOW)

**Current state:** Candle has a smoke test (`scripts/smoke-first-slice.sh`). Volume has unit tests but no integration or smoke test.

**Impact:** Volume pipeline correctness is verified only by unit tests and manual testing.

**Fix:** Add `scripts/smoke-volume.sh` or extend existing smoke test to include volume assertions.

---

### G7 — EvidencePublisher has no volume import for evidence package (COSMETIC)

**Current state:** `evidence_publisher.go` comment still says "publishes candle sampled events" in the struct doc. It now publishes 3 types.

**Impact:** Cosmetic. No functional impact.

**Fix:** Update struct comment to reflect all 3 evidence types.

---

## Sequenced Next Moves

### Move 1 — Governance Hygiene (S33, recommended next)

**Scope:** Fix G1, G2, G4, G7. No implementation changes.

**Deliverables:**
- Rewrite actor-ownership.md Phase 3 sections
- Update raccoon-cli topology.rs and drift_detect.rs
- Update catalog documents for volume status
- Fix stale comments

**Why first:** These are the two HIGH-severity gaps. Fixing them before any expansion ensures the governance layer is trustworthy.

**Effort:** One focused stage. No code changes beyond raccoon-cli Rust updates.

---

### Move 2 — evidence.stats adoption (S34, optional)

**Scope:** Implement the 4th evidence type using the proven FamilyProcessor + ProjectionPipeline pattern.

**Deliverables:**
- `EvidenceStats` domain type (volatility, spread, tick frequency)
- StatsSampler application logic
- Full pipeline: derive → store → gateway

**Why:** Further validates the patterns at 4 types and provides distributional metrics useful for signal. Also tests whether QueryResponderActor's manual growth is still acceptable at 5 routes.

**Prerequisite:** Move 1 (governance hygiene).

---

### Move 3 — Signal domain design (S35, gated)

**Scope:** Design signal domain contracts and activation mechanism. NO implementation.

**Deliverables:**
- Signal domain type definitions
- Signal stream spec (SIGNAL_EVENTS)
- Signal activation mechanism design (config-driven, evidence→signal derivation)
- Signal query surface design
- Updated raccoon-cli rules for signal guards

**Why:** Signal is the next domain boundary. It requires a different derivation model (reads evidence, not observation) and a new activation mechanism. Design before implementation.

**Prerequisites:** Move 1 (governance hygiene), G5 resolution (config-driven activation design).

---

### Move 4 — Signal implementation (S36+, gated)

**Scope:** Implement signal domain end-to-end.

**Prerequisites:**
- Move 3 complete (design approved)
- Config-driven activation implemented and tested
- raccoon-cli guards for SIGNAL_EVENTS active
- actor-ownership.md current

**This move should NOT begin until all prerequisites are met.**

---

## Decision Framework: When to Enter Signal

| Question | Required Answer |
|----------|----------------|
| Is actor-ownership.md current? | Yes |
| Can raccoon-cli validate the current topology? | Yes |
| Are 3+ evidence types proven end-to-end? | Yes (candle, tradeburst, volume) |
| Is config-driven activation designed? | Yes |
| Is config-driven activation implemented? | Yes |
| Is the signal domain designed (contracts, streams, queries)? | Yes |
| Are raccoon-cli guards for SIGNAL_EVENTS active? | Yes |

**All answers must be "Yes" before signal implementation begins.** This is a hard gate, not a soft recommendation.

## What Is Explicitly NOT Recommended

1. **Do not enter signal before governance is fixed.** The mesh patterns are ready but the governance layer cannot validate them. Expanding without governance creates silent drift.

2. **Do not abstract QueryResponderActor yet.** At 3 types (4 routes) it is manageable. Premature abstraction would add complexity without reducing maintenance. Address at 5+ types.

3. **Do not add evidence.stats just to "prove the pattern again."** The pattern is already proven with 3 types. Stats should enter only if it provides concrete analytical value for the next phase.

4. **Do not implement config-driven activation as a generic framework.** Design it for the specific signal use case (evidence → signal derivation). Generalize later if needed.
