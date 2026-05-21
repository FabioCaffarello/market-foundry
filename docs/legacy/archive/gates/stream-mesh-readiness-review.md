# Stream Mesh Readiness Review

> Formal assessment of whether the Market Foundry stream mesh is ready for the next expansion cycle.
> Conducted after S26-S31 (mesh canonicalization, family patterns, first new family adoption).

## Review Scope

This review evaluates five subsystems against explicit readiness criteria:

1. **Stream mesh** — Is it explicit and canonical?
2. **Derive** — Is it truly family-oriented?
3. **Store** — Is it truly mesh-aware?
4. **Gateway** — Are query contracts aligned?
5. **raccoon-cli** — Can it audit the topology?

## Verdict Summary

| Subsystem | Readiness | Confidence | Blocking Issues |
|-----------|-----------|------------|-----------------|
| Stream mesh model | **READY** | High | None |
| Derive | **READY** | High | None |
| Store | **READY with caveat** | Medium | QueryResponderActor scales manually |
| Gateway | **READY** | High | None |
| raccoon-cli | **NOT READY** | Low | Stale topology rules, missing subject/durable coverage |
| Documentation | **PARTIALLY STALE** | Medium | actor-ownership.md outdated, catalog docs incomplete for volume |

**Overall: CONDITIONALLY READY. Two blockers must be resolved before signal entry.**

---

## 1. Stream Mesh — Is It Explicit and Canonical?

**Verdict: READY**

The stream mesh is now a first-class architectural concept with dedicated documents:

| Document | Status | Quality |
|----------|--------|---------|
| `stream-mesh-model.md` | Current | Excellent — transport-independent, correct layers and dimensions |
| `stream-families.md` | Current (minor gap) | Volume listed as planned despite code existing |
| `stream-family-catalog.md` | Current (minor gap) | Same volume status ambiguity |
| `stream-ownership-matrix.md` | Current | Accurate for candle + tradeburst; volume not yet in matrix |
| `mesh-vs-transport.md` | Current | Clean separation of concerns |

**Mesh inventory:**

| Resource | Count | Single-Writer Verified |
|----------|-------|----------------------|
| JetStream streams | 3 (CONFIGCTL, OBSERVATION, EVIDENCE) | Yes |
| Durable consumers | 6 (2 binding watchers, 1 derive, 3 store) | Yes |
| KV buckets | 4 (CANDLE_LATEST, CANDLE_HISTORY, TRADE_BURST_LATEST, VOLUME_LATEST) | Yes |
| Query subjects | 5 (10 configctl + 5 evidence) | Yes |
| HTTP endpoints | 15 (2 core + 8 configctl + 5 evidence) | N/A |

**Evidence:** The mesh model survived the volume adoption (S31) without structural changes. Subject patterns, stream sharing, and consumer isolation all worked as designed.

---

## 2. Derive — Is It Truly Family-Oriented?

**Verdict: READY**

The FamilyProcessor pattern (S28) is proven with 3 families:

| Component | Family-Agnostic? | Evidence |
|-----------|------------------|----------|
| `SourceScopeActor.onActivateSampler` | **Yes** | Iterates `cfg.Processors`, zero hardcoded references |
| `DeriveSupervisor.start` | **Registration point** | 3 entries, clearly commented |
| Sampler actors | **Per-family, type-safe** | SamplerActor, TradeBurstSamplerActor, VolumeSamplerActor |
| Publisher actor | **Explicit per-type cases** | 3 message cases, identical pattern |
| EvidencePublisher adapter | **Explicit per-type methods** | 3 Publish methods, consistent structure |

**Key metric:** Adding volume (S31) required:
- 0 changes to SourceScopeActor (spawning loop)
- 0 changes to ConsumerActor (trade routing)
- 0 changes to BindingWatcherActor (activation)
- 1 entry in processor list
- 1 sampler actor file
- 1 sampler logic file
- 1 publish message type
- 1 publisher case
- 1 adapter method

**Assessment:** The derive binary is genuinely family-oriented. The FamilyProcessor pattern works.

---

## 3. Store — Is It Truly Mesh-Aware?

**Verdict: READY with caveat**

The ProjectionPipeline pattern (S29) is proven with 3 pipelines:

| Component | Family-Agnostic? | Evidence |
|-----------|------------------|----------|
| `StoreSupervisor` spawning loop | **Yes** | Iterates `s.pipelines`, zero hardcoded references |
| Projection actors | **Per-family, type-safe** | CandleProjectionActor, TradeBurstProjectionActor, VolumeProjectionActor |
| Consumer actors | **Per-family, type-safe** | EvidenceConsumerActor, TradeBurstConsumerActor, VolumeConsumerActor |
| Health trackers | **Dynamic map** | Convention-based lookup by pipeline name |
| **QueryResponderActor** | **NOT family-agnostic** | Manual KV store fields, manual route registration, manual handlers |

**Caveat — QueryResponderActor scales manually:**

The responder currently has:
- 3 KV store fields (`store`, `burstStore`, `volumeStore`)
- 3 KV store initializations in `start()`
- 3 `Close()` calls in `Stopped` handler
- 4 query routes (candle latest, candle history, tradeburst latest, volume latest)
- 4 handler methods

Adding a 4th evidence type requires touching all 5 sections. At 3 types this is manageable. At 5+ it becomes a maintenance risk. This is not a blocker but should be addressed before the 5th evidence type.

**Assessment:** The store is mesh-aware for pipeline spawning. The query responder is the one component that still grows by manual multiplication.

---

## 4. Gateway — Are Query Contracts Aligned?

**Verdict: READY**

The EvidenceFamilyDeps pattern (S30) is proven:

| Component | Family-Grouped? | Evidence |
|-----------|-----------------|----------|
| `EvidenceFamilyDeps` | **Yes** | 4 fields grouped by family with comments |
| `Evidence()` route function | **Yes** | Route blocks separated by family |
| `EvidenceWebHandler` | **Explicit per-type** | 4 handlers, shared param parsing |
| `EvidenceGateway` port | **Explicit per-type** | 4 methods, 1:1 with query operations |
| Use case wiring | **Conditional** | All 4 created only if evGateway != nil |
| Readiness | **Non-blocking** | Evidence probe warns but doesn't fail readiness |

**Key metric:** Adding volume (S31) required:
- 1 field in EvidenceFamilyDeps
- 1 route block in Evidence()
- 1 handler method
- 1 use case
- 1 gateway adapter method
- 1 contract pair (query + reply)
- 0 changes to DefaultRoutes, readiness, core routing

**Assessment:** The gateway is clean, aligned, and ready for new families.

---

## 5. raccoon-cli — Can It Audit the Topology?

**Verdict: NOT READY**

raccoon-cli has significant coverage gaps that prevent it from validating the current mesh:

### What raccoon-cli CAN validate

| Capability | Status |
|------------|--------|
| Layer dependencies (arch-guard) | Working — AST-based, comprehensive |
| Envelope contracts (contract-audit) | Working — validates all registered specs |
| Domain event alignment | Working — scans code for event definitions |
| Identity drift (defunct names) | Working — detects quality-service references |
| Basic topology (stream existence) | Working but incomplete |

### What raccoon-cli CANNOT validate

| Gap | Severity | Impact |
|-----|----------|--------|
| **Durable names stale** — expects `store-evidence` (generic), code uses `store-candle`, `store-trade-burst`, `store-volume` | **HIGH** | Topology checks will reject valid code |
| **Subject prefixes incomplete** — only candle subjects listed, tradeburst and volume missing | **MEDIUM** | Topology checks miss 2 of 3 evidence types |
| **Architecture doc inventory stale** — drift-detect checks 3 docs, 6+ new docs not inventoried | **MEDIUM** | Doc staleness goes undetected |
| **No planned-family guards** — SIGNAL_EVENTS/PROJECTION_EVENTS not guarded against premature entry | **MEDIUM** | Signal code could enter without architecture approval |
| **No single-writer validation** — cannot detect if multiple binaries write to the same stream | **LOW** | Enforced by review only |
| **No partition key validation** — cannot verify subject dimensions match actor tree | **LOW** | Enforced by review only |

### Required Fixes Before Next Expansion

1. **topology.rs** — Update `EXPECTED_DURABLES` to `["derive-observation", "store-candle", "store-trade-burst", "store-volume"]`
2. **topology.rs** — Update `EXPECTED_SUBJECT_PREFIXES` to include tradeburst and volume subjects
3. **drift_detect.rs** — Expand `ARCH_DOCS` to include all new architecture documents
4. **drift_detect.rs** — Add premature-entry guards for SIGNAL_EVENTS and PROJECTION_EVENTS

---

## 6. Documentation — Is It Current?

### Stale Documents

| Document | Issue | Severity |
|----------|-------|----------|
| `actor-ownership.md` | Store section missing TradeBurstProjectionActor, TradeBurstConsumerActor, VolumeProjectionActor, VolumeConsumerActor. Derive section missing TradeBurstSamplerActor, VolumeSamplerActor. Cross-binary matrix lists store as OBSERVATION_EVENTS consumer (incorrect). | **HIGH** |
| `stream-families.md` | Volume listed as "Planned" but code is implemented and running | **LOW** |
| `stream-family-catalog.md` | CF-07 (volume) marked "Planned" — should be "Current" | **LOW** |
| `projection-family-matrix.md` | P-03 (volume) listed as "Planned" — implemented in S31 | **LOW** |
| `stream-ownership-matrix.md` | Missing volume consumer, volume KV bucket, volume query subject | **MEDIUM** |
| `signal-readiness-review.md` | Written before volume (S31); does not account for 3rd evidence type | **LOW** |

### Current Documents

| Document | Status |
|----------|--------|
| `stream-mesh-model.md` | Current — architecture-independent, no type-specific content |
| `mesh-vs-transport.md` | Current |
| `derive-family-processor-pattern.md` | Current |
| `projection-families-model.md` | Current |
| `latest-history-by-family.md` | Current |
| `query-contracts-by-family.md` | Current |
| `gateway-read-surface-guidelines.md` | Current |
| `evidence-derivation-pattern.md` | Current |
| `evidence-read-model-guidelines.md` | Current |

---

## Readiness Gate: Signal Entry Prerequisites

The signal domain is the next major mesh expansion. Based on this review, these prerequisites must be met:

| Prerequisite | Status | Blocker? |
|-------------|--------|----------|
| 3+ evidence types proven end-to-end | **MET** (candle, tradeburst, volume) | No |
| FamilyProcessor pattern validated | **MET** (3 families, spawning loop untouched) | No |
| ProjectionPipeline pattern validated | **MET** (3 pipelines, spawning loop untouched) | No |
| Gateway query alignment proven | **MET** (EvidenceFamilyDeps with 4 endpoints) | No |
| actor-ownership.md current | **NOT MET** (stale since S12) | **Yes** |
| raccoon-cli topology rules current | **NOT MET** (stale durables, missing subjects) | **Yes** |
| Config-driven activation proven | **NOT MET** (identified in S25, unresolved) | **Yes — for signal** |
| Stream mesh docs complete | **PARTIALLY MET** (6 docs stale, 9 current) | No (low severity) |

**Signal entry is blocked by 3 prerequisites.** The first two (actor-ownership, raccoon-cli) are documentation/tooling gaps that can be resolved in one stage. The third (config-driven activation) is a design gap that requires implementation work.

---

## Conclusion

The Market Foundry stream mesh has been successfully canonicalized across S26-S31. The mesh model, family patterns, and ownership boundaries are proven with three evidence types. The architecture is genuinely ready for expansion — the patterns work, the code is clean, and the mesh absorbed a new family without distortion.

However, **the governance layer has not kept pace with the implementation.** actor-ownership.md is 5 stages behind, raccoon-cli topology rules are stale, and several catalog documents don't reflect S31 changes. These are not architectural gaps — they are hygiene debts that will cause confusion and CI failures if not addressed.

**The mesh is ready. The governance is not. Fix governance before expanding.**
