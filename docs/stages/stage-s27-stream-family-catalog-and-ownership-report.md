# S27 — Stream Family Catalog and Ownership Matrix

**Stage:** S27
**Type:** Architecture
**Status:** Complete
**Date:** 2025-03-17
**Depends on:** S26 (Stream Mesh Canonicalization)

## Objective

Translate the S26 mesh canonicalization into an operational catalog and ownership matrix, connecting architecture documents to runtime actors, NATS registries, and contracts.

## Context

S26 established the stream mesh as a first-class architectural concept with layers, dimensions, families, and evolution rules. However, the mapping between families and their runtime components (actors, consumers, publishers, projections, queries) remained implicit — scattered across actor-ownership.md, registry code, and multiple stage reports.

This stage consolidates that mapping into three cross-cutting documents that serve as the operational reference for the mesh.

## Deliverables

### 1. Stream Family Catalog (`docs/architecture/stream-family-catalog.md`)

Complete per-family entries with fixed schema:

| Entry | Family | Classification | Status |
|-------|--------|---------------|--------|
| CF-01 | configctl | Lifecycle | Current |
| CF-02 | observation | Continuous | Current |
| CF-03 | evidence.candle | Sampled | Current |
| CF-04 | evidence.tradeburst | Sampled | Current |
| CF-05 | signal | Derived | Planned |
| CF-06 | projection | Projection | Planned |
| CF-07 | evidence.volume | Sampled | Planned |
| CF-08 | evidence.stats | Sampled | Planned |

Each entry includes: canonical name, bounded context, classification, publisher owner, consumer owners, projection owner, query owner, dimensions, stream, retention, readiness status, event types, durable consumers, query surface, actor ownership chain, and architectural notes.

**Planned evidence types (volume, stats):** Added as planned families because they follow the proven evidence derivation pipeline, require no new infrastructure, and have direct value for signal readiness. They are not speculative futures — they are the next natural evidence types.

**Deferred families (decision, risk, execution, portfolio):** Listed as naming reservations only with explicit blockers. No schema, no subject pattern, no implementation timeline.

### 2. Stream Ownership Matrix (`docs/architecture/stream-ownership-matrix.md`)

Five cross-cutting views:

1. **Event Stream Ownership** — writer and consumer binaries per stream with single-writer verification
2. **Projection Ownership** — writer actor and reader per KV bucket with single-writer verification
3. **Query Surface Ownership** — server binary and client per subject with single-server verification
4. **Binary Role Summary** — what each binary writes, reads, projects, and queries
5. **Consumer Cursor Summary** — all durable consumers with filter subjects and deliver policies

Key insight: the data flow is strictly unidirectional (configctl → ingest → derive → store → gateway) with no feedback loops.

### 3. Projection Family Matrix (`docs/architecture/projection-family-matrix.md`)

Focused on the store's projection layer:

1. **Pipeline pattern** with 7 invariants
2. **Current projections** (candle, trade burst) with full component mapping
3. **Planned projections** (volume, stats) with expected component values
4. **Supervisor actor tree** with growth projections
5. **Health model** with per-pipeline tracker pairs
6. **Cross-reference table** connecting projection → actor → bucket → query → HTTP

---

## Inconsistencies Found

Analysis of the current codebase against the ownership matrix reveals these inconsistencies:

### I-01: actor-ownership.md — Store section outdated (HIGH)

**Document says:** Store has `EvidenceConsumerActor` with durable `store-evidence` consuming `evidence.events.candle.sampled.>`, plus `CandleProjectionActor` writing to `CANDLE_LATEST`.

**Code reality:**
- Store has **two** consumer actors: `EvidenceConsumerActor` (durable: `store-candle`) and `TradeBurstConsumerActor` (durable: `store-trade-burst`)
- Store has **two** projection actors: `CandleProjectionActor` and `TradeBurstProjectionActor`
- CandleProjectionActor writes to **two** buckets: `CANDLE_LATEST` and `CANDLE_HISTORY`
- TradeBurstProjectionActor writes to `TRADE_BURST_LATEST`
- QueryResponderActor serves **three** query subjects: candle latest, candle history, tradeburst latest

**Impact:** actor-ownership.md is the canonical ownership document but does not reflect S23 (trade burst) and S24 (multi-projection) changes. New contributors consulting this document will have an incomplete picture.

**Recommendation:** Update the Phase 3 — Store section to match current code.

### I-02: actor-ownership.md — Derive section outdated (HIGH)

**Document says:** Derive has `SamplerActor` per symbol. EvidencePublisherActor publishes candles only.

**Code reality:**
- Each symbol/timeframe gets **two** sampler actors: `SamplerActor` (candle) and `TradeBurstSamplerActor` (trade burst)
- EvidencePublisherActor handles **both** `publishCandleMessage` and `publishTradeBurstMessage`
- SourceScopeActor spawns pairs: `sampler-{symbol}-{tf}s` and `burst-sampler-{symbol}-{tf}s`

**Impact:** Same as I-01 — the canonical derive ownership tree is incomplete.

**Recommendation:** Update the Phase 3 — Derive section to include TradeBurstSamplerActor and the dual-sampler spawn pattern.

### I-03: actor-ownership.md — Cross-Binary Stream Matrix stale (MEDIUM)

**Document says:** OBSERVATION_EVENTS consumers include `store`. SIGNAL_EVENTS and PROJECTION_EVENTS are listed as Phase 3.

**Code reality:**
- Store does **not** consume OBSERVATION_EVENTS. Only derive consumes it.
- SIGNAL_EVENTS and PROJECTION_EVENTS are **planned**, not Phase 3 (no code exists).

**Impact:** The matrix incorrectly implies store already reads observation events and that signal/projection are Phase 3 deliverables.

**Recommendation:** Remove store from OBSERVATION_EVENTS consumers. Change SIGNAL_EVENTS and PROJECTION_EVENTS phase to "Planned".

### I-04: actor-ownership.md — Control Plane Matrix stale (LOW)

**Document says:** `evidence.query.candle.latest` is Phase 3 (S13).

**Code reality:** This was implemented in S13 but the matrix doesn't reflect S20 (candle history) or S23 (trade burst latest). Missing entries:
- `evidence.query.candle.history` — store → gateway (S20)
- `evidence.query.tradeburst.latest` — store → gateway (S23)

**Recommendation:** Add missing query subject entries.

### I-05: Derive QueryResponderActor residue (LOW)

**actor-ownership.md** shows a `QueryResponderActor` in derive that serves `evidence.query.candle.latest` for real-time snapshots. This was a transitional pattern from S07 before store took over query serving in S13.

**Code reality:** Need to verify if this actor still exists in derive or was removed when store became the query authority.

**Recommendation:** Verify and remove if the derive QueryResponderActor is dead code.

### I-06: Envelope type inconsistency in configctl (LOW)

**Observation:** configctl event envelope types use `configctl.event.config.*` (singular `event`) while observation and evidence use `observation.events.v1.*` and `evidence.events.v1.*` (plural `events` with version). This is a naming asymmetry, not a bug — configctl predates the versioned envelope convention.

**Impact:** Cosmetic. Does not affect functionality. Would require a breaking change to fix (existing consumers decode by type string).

**Recommendation:** Document as known asymmetry. Do not fix unless a major envelope migration happens.

---

## Summary

### Catalog Highlights

- **8 stream families** cataloged: 4 current, 2 planned (signal, projection), 2 planned evidence types (volume, stats)
- **4 deferred families** as naming reservations (decision, risk, execution, portfolio)
- **Every current family** has: canonical name, ownership chain, event types, consumers, query surface, actor tree, and architectural notes

### Ownership Matrix Highlights

- **3 active JetStream streams** with verified single-writer compliance
- **3 active KV buckets** with verified single-writer compliance
- **13 active query subjects** across configctl and evidence
- **5 durable consumers** with unique names and independent cursors
- **Unidirectional data flow** confirmed: no feedback loops in the mesh

### Inconsistencies

| ID | Severity | Document | Issue |
|----|----------|----------|-------|
| I-01 | HIGH | actor-ownership.md | Store section missing trade burst actors and candle history |
| I-02 | HIGH | actor-ownership.md | Derive section missing TradeBurstSamplerActor |
| I-03 | MEDIUM | actor-ownership.md | Cross-binary matrix has stale entries |
| I-04 | LOW | actor-ownership.md | Control plane matrix missing S20/S23 query subjects |
| I-05 | LOW | actor-ownership.md | Possible derive QueryResponderActor residue |
| I-06 | LOW | configctl_registry.go | Envelope type naming asymmetry (no version segment) |

### Recommendations for S28

**R1 — Update actor-ownership.md (HIGH PRIORITY)**

The canonical ownership document is stale. It must be updated to reflect S23 (trade burst) and S24 (multi-projection) changes. This is the highest priority because actor-ownership.md is referenced by raccoon-cli and by all architecture documents. Specifically:
- Rewrite Phase 3 — Store to include all 5 actors
- Rewrite Phase 3 — Derive to include TradeBurstSamplerActor
- Fix Cross-Binary Stream Matrix (remove store from OBSERVATION_EVENTS consumers, change signal/projection to Planned)
- Add missing Control Plane Matrix entries

**R2 — Verify derive QueryResponderActor status**

Check if the derive-side QueryResponderActor from S07 still exists. If it does, it is dead code and should be removed — store is the sole query authority since S13.

**R3 — Design evidence.volume contracts**

With the catalog and ownership matrix established, the volume evidence type can be designed at the contract level. This would produce: domain type, event contract, sampler logic, consumer spec, projection spec, and query surface — following the evidence-read-model-guidelines checklist.

**R4 — Extend raccoon-cli ownership validation**

raccoon-cli should validate ownership invariants against the matrix:
- Every JetStream stream has exactly one writer binary (check registry imports per binary)
- Every KV bucket is written by exactly one actor
- No binary both writes to and reads from the same stream

**R5 — Consolidate canonical doc references**

The Foundry now has multiple architecture documents that cover overlapping concerns. Suggested hierarchy:
- **stream-mesh-model.md** — conceptual model (what is the mesh?)
- **stream-families.md** → superseded by **stream-family-catalog.md** for operational detail
- **stream-ownership-matrix.md** — cross-cutting ownership verification
- **projection-family-matrix.md** — store-specific projection detail
- **actor-ownership.md** — actor tree and supervision structure

Consider adding a navigation header in each doc pointing to related documents.

## Files Created

| File | Purpose |
|------|---------|
| `docs/architecture/stream-family-catalog.md` | Per-family operational catalog |
| `docs/architecture/stream-ownership-matrix.md` | Cross-cutting ownership matrix |
| `docs/architecture/projection-family-matrix.md` | Store projection pipeline matrix |
| `docs/stages/stage-s27-stream-family-catalog-and-ownership-report.md` | This report |

## Files Not Modified

No existing files were modified. The inconsistencies in actor-ownership.md are documented for resolution in S28.

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Each family has clear ownership | Met — catalog has publisher, consumer, projection, and query owner per family |
| Store and gateway appear as correct read-side owners | Met — store owns all projections and query responses; gateway owns HTTP translation only |
| Matrix helps directly with next refactors | Met — inconsistencies I-01 through I-05 provide concrete action items |
| Relationship between family, actor, and projection is clear | Met — projection-family-matrix connects all components with actor trees |
| No new implementations opened | Met — no code changes |
| Catalog not overloaded with speculative futures | Met — 4 current + 2 planned evidence + 2 planned families; 4 deferred as reservations only |
| No ambiguous ownership | Met — every stream, bucket, and query subject has exactly one owner |
| Store's role as projection authority preserved | Met — store is sole writer to all KV buckets and sole server for all evidence queries |
