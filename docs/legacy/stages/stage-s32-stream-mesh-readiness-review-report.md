# S32 — Stream Mesh Readiness Review

**Stage:** S32
**Type:** Review
**Status:** Complete
**Date:** 2025-03-17
**Depends on:** S26-S31 (mesh canonicalization cycle)

## Objective

Conduct a formal readiness review of the stream mesh after the S26-S31 canonicalization cycle, determining whether the Foundry is ready for the next expansion phase.

## Review Method

Four independent audits were conducted, each covering a major subsystem:

1. **Derive audit** — FamilyProcessor pattern, spawning loop, publisher, registry
2. **Store audit** — ProjectionPipeline pattern, query responder, messages, trackers
3. **Gateway audit** — EvidenceFamilyDeps, routes, handlers, ports, contracts, wiring
4. **Governance audit** — Architecture documents staleness, raccoon-cli coverage gaps

Each audit read the actual source files and evaluated pattern adherence, consistency, and gaps.

## Executive Summary

**The mesh is architecturally ready. The governance is not.**

The S26-S31 cycle successfully transformed the Foundry's stream topology from implicit code into an explicit, canonical mesh model. Three evidence types (candle, tradeburst, volume) are proven end-to-end across all three binaries. The FamilyProcessor (derive), ProjectionPipeline (store), and EvidenceFamilyDeps (gateway) patterns are validated — they handled volume adoption without modifying any spawning loops or core routing.

However, the governance layer has not kept pace:
- `actor-ownership.md` is 5 stages stale (last updated S12)
- raccoon-cli topology rules reference a generic durable name (`store-evidence`) that doesn't exist in code
- Six architecture documents still list volume as "Planned" despite it being implemented
- No premature-entry guards exist for SIGNAL_EVENTS

**Verdict: Fix governance before expanding further.**

## Readiness Assessment by Subsystem

### Derive — READY

| Criterion | Met? | Evidence |
|-----------|------|----------|
| FamilyProcessor pattern works | **Yes** | 3 families registered, spawning loop untouched for volume |
| SourceScopeActor is family-agnostic | **Yes** | Zero hardcoded family references |
| Publisher scales by case addition | **Yes** | 3 identical publish cases |
| Registry specs consistent | **Yes** | EventSpec + ControlSpec + ConsumerSpec per family |
| Dedup keys collision-free | **Yes** | Family-specific prefixes (none, "burst:", "vol:") |

### Store — READY with caveat

| Criterion | Met? | Evidence |
|-----------|------|----------|
| ProjectionPipeline pattern works | **Yes** | 3 pipelines registered, spawning loop untouched |
| Projection actors follow invariants | **Yes** | Final gate, Validate gate, monotonicity guard on all 3 |
| Health trackers dynamic | **Yes** | Map-based, convention-named |
| QueryResponderActor scales cleanly | **No** | Manual field/route/handler accumulation per type |

**Caveat detail:** QueryResponderActor currently has 3 KV stores, 4 routes, 4 handlers. Each new evidence type adds ~15 lines across 5 sections. This is acceptable at 3 types but becomes friction at 5+. Not a blocker — a maintenance risk to monitor.

### Gateway — READY

| Criterion | Met? | Evidence |
|-----------|------|----------|
| EvidenceFamilyDeps groups use cases | **Yes** | 4 fields, HasAny() auto-includes new types |
| Routes organized by family | **Yes** | Family comments, conditional registration |
| Contracts consistent | **Yes** | Query/Reply pairs for all 4 operations |
| Graceful degradation works | **Yes** | Evidence optional, configctl required |
| Port interface complete | **Yes** | 4 methods, 1:1 with query operations |

### raccoon-cli — NOT READY

| Criterion | Met? | Evidence |
|-----------|------|----------|
| Topology rules match code | **No** | `store-evidence` durable doesn't exist; should be `store-candle`, `store-trade-burst`, `store-volume` |
| Subject prefixes complete | **No** | Only candle subjects listed; tradeburst and volume missing |
| Architecture doc inventory current | **No** | 3 of 9+ docs inventoried |
| Premature-entry guards exist | **No** | SIGNAL_EVENTS could be added without detection |
| Contract audit works | **Yes** | Validates all registered specs correctly |
| Layer guard works | **Yes** | AST-based boundary enforcement |

### Documentation — PARTIALLY STALE

| Document | Status | Gap |
|----------|--------|-----|
| `actor-ownership.md` | **STALE (HIGH)** | Missing 6 actors, incorrect matrix entries |
| `stream-ownership-matrix.md` | **STALE (MEDIUM)** | Missing volume consumer, bucket, query |
| `stream-families.md` | **STALE (LOW)** | Volume listed as planned |
| `stream-family-catalog.md` | **STALE (LOW)** | CF-07 volume marked planned |
| `projection-family-matrix.md` | **STALE (LOW)** | P-03 volume marked planned |
| `stream-mesh-model.md` | Current | — |
| `mesh-vs-transport.md` | Current | — |
| `derive-family-processor-pattern.md` | Current | — |
| `projection-families-model.md` | Current | — |
| `latest-history-by-family.md` | Current | — |
| `query-contracts-by-family.md` | Current | — |
| `gateway-read-surface-guidelines.md` | Current | — |

## Gaps Inventory

| ID | Gap | Severity | Blocker? |
|----|-----|----------|----------|
| G1 | actor-ownership.md stale (5 stages behind) | HIGH | Yes (signal) |
| G2 | raccoon-cli topology rules stale | HIGH | Yes (CI) |
| G3 | QueryResponderActor manual scaling | MEDIUM | No (threshold: 5 types) |
| G4 | Catalog documents incomplete for volume | LOW | No |
| G5 | Config-driven activation not designed | MEDIUM | Yes (signal only) |
| G6 | No volume smoke test | LOW | No |
| G7 | Stale publisher comment | COSMETIC | No |

## Recommended Next Moves (Sequenced)

### S33 — Governance Hygiene (recommended next)

Fix G1 + G2 + G4 + G7. No implementation changes. High impact, bounded scope.

- Rewrite actor-ownership.md Phase 3 sections
- Update raccoon-cli topology.rs and drift_detect.rs
- Update catalog documents for volume status

### S34 — evidence.stats (optional)

4th evidence type. Validates patterns at 4 types. Provides distributional metrics.

Prerequisite: S33 complete.

### S35 — Signal domain design (gated)

Design only — no implementation. Signal contracts, activation mechanism, stream spec.

Prerequisites: S33 complete, config-driven activation designed.

### S36 — Signal implementation (hard-gated)

All 7 prerequisites from the readiness review must be met.

## Signal Entry Gate

| Prerequisite | Status |
|-------------|--------|
| actor-ownership.md current | NOT MET |
| raccoon-cli validates topology | NOT MET |
| 3+ evidence types proven | **MET** |
| Config-driven activation designed | NOT MET |
| Config-driven activation implemented | NOT MET |
| Signal domain designed | NOT MET |
| raccoon-cli guards for SIGNAL_EVENTS | NOT MET |

**3 of 7 prerequisites met. Signal is not ready.**

## Deliverables

| File | Purpose |
|------|---------|
| `docs/architecture/stream-mesh-readiness-review.md` | Formal readiness assessment per subsystem |
| `docs/architecture/mesh-gaps-and-next-moves.md` | Actionable gap analysis with sequenced recommendations |
| `docs/stages/stage-s32-stream-mesh-readiness-review-report.md` | This report |

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Review is honest and specific | Met — gaps identified with severity and evidence |
| Review is actionable | Met — 7 gaps with fixes, 4 sequenced moves |
| State of mesh is clear | Met — subsystem-by-subsystem assessment with metrics |
| Next steps are well-founded | Met — prerequisites and gates for signal entry |
| Foundry gains a real gate before next expansion | Met — 7-prerequisite signal entry gate established |
| Gaps not masked | Met — raccoon-cli rated NOT READY, actor-ownership rated STALE |
| Signal not pushed prematurely | Met — 3 of 7 prerequisites met, explicitly blocked |
| No vague documentation | Met — every gap has severity, impact, and fix |
| Blockers and preconditions concrete | Met — G1, G2, G5 identified as blockers with specific remediation |
