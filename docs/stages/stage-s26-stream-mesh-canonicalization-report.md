# S26 — Stream Mesh Canonicalization

**Stage:** S26
**Type:** Architecture
**Status:** Complete
**Date:** 2025-03-17

## Objective

Transform the stream mesh from an implicit consequence of NATS adapter code into a first-class architectural concept with explicit vocabulary, classification, ownership rules, and evolution guidelines.

## Context

The Foundry reached S25 with a functional stream topology: three active JetStream streams, five durable consumers, three KV buckets, and two evidence types flowing end-to-end. The topology worked, but it existed primarily as implementation detail — scattered across registry files, actor-ownership.md, and stream-taxonomy.md.

The MarketMonkey codebase demonstrated the value of treating stream topology as an explicit architectural artifact. Its natural mesh of source/symbol/timeframe-scoped streams enabled rapid domain evolution. The Foundry needed to absorb this strength without copying its structure.

## Deliverables

### 1. Stream Mesh Model (`docs/architecture/stream-mesh-model.md`)

Defines the stream mesh as an architectural concept with:
- **Four conceptual layers**: Control Surface, Event Surface, Projection Surface, Query Surface
- **Seven mesh dimensions**: family, surface, aggregate, verb, source, symbol, timeframe
- **Five mesh properties**: single-writer streams, fan-out consumption, partition-aligned isolation, deduplication by design, envelope uniformity
- **Stream lifecycle states**: Planned → Defined → Implemented → Active → Deprecated → Removed
- **Five evolution rules** governing how the mesh grows
- **Bounded context mapping** clarifying the relationship between families and binaries

### 2. Stream Families (`docs/architecture/stream-families.md`)

Catalogs every stream family with consistent metadata:
- **Three active families**: configctl (lifecycle), observation (continuous), evidence (sampled)
- **Two planned families**: signal (derived), projection (notification)
- **Four future families**: decision, risk, execution, portfolio (naming reservations only)
- **Six classification types**: continuous, sampled, derived, lifecycle, projection, query-only
- **Seven invariants** that apply to all families
- **Addition checklist** for new families

### 3. Mesh vs. Transport (`docs/architecture/mesh-vs-transport.md`)

Separates the logical mesh from its NATS encoding:
- **Mesh vocabulary** (family, surface, aggregate, partition key, writer, consumer, projection)
- **Transport encoding** (subject patterns, JetStream configs, KV buckets, consumers, envelopes)
- **Complete mapping tables** from mesh concepts to NATS artifacts
- **Decision framework** for classifying changes as mesh-level or transport-level

## Stream Families — Summary

| Family | Classification | Stream | Writer | Status |
|--------|---------------|--------|--------|--------|
| configctl | Lifecycle | CONFIGCTL_EVENTS | configctl | Active |
| observation | Continuous | OBSERVATION_EVENTS | ingest | Active |
| evidence | Sampled | EVIDENCE_EVENTS | derive | Active |
| signal | Derived | SIGNAL_EVENTS | derive | Planned |
| projection | Projection | PROJECTION_EVENTS | store | Planned |
| decision | Derived | — | TBD | Future (reserved) |
| risk | Derived | — | TBD | Future (reserved) |
| execution | Lifecycle | — | TBD | Future (reserved) |
| portfolio | Lifecycle | — | TBD | Future (reserved) |

## Codebase Gaps

Analysis of the current codebase against the canonical mesh model reveals these gaps:

### G1 — No mesh-level test assertions

The mesh model defines invariants (single-writer, family isolation, deduplication), but there are no tests that validate these invariants at the mesh level. Integration tests validate individual pipelines but do not assert cross-family properties.

**Severity:** Low (invariants are enforced by architecture review and raccoon-cli, but automated validation would add confidence).

### G2 — stream-taxonomy.md partially overlaps with new documents

The existing `stream-taxonomy.md` covers subject naming conventions and stream definitions. Parts of this content now overlap with `stream-mesh-model.md` (conceptual model) and `mesh-vs-transport.md` (encoding rules). The taxonomy document remains valid as a quick-reference for subject patterns, but its role should be clarified as a transport-layer reference.

**Severity:** Low (no conflict, but clarification reduces confusion for new contributors).

### G3 — Observation partitioning is coarser than evidence

Observation events are partitioned by `source` only. Evidence events are partitioned by `source.symbol.timeframe`. This asymmetry is intentional and documented, but it means store cannot efficiently filter observation events by symbol if it ever needs to consume them directly.

**Severity:** None currently (store does not consume observation events). Worth noting for future families that might need symbol-level observation access.

### G4 — No projection events exist yet

The PROJECTION_EVENTS stream is documented as planned in actor-ownership.md and now in stream-families.md, but no registry spec, consumer, or producer exists. Gateway currently has no mechanism for cache invalidation or reactive updates.

**Severity:** Low (gateway is stateless today; projection events become relevant when gateway adds caching or when external consumers need materialization notifications).

### G5 — Signal family blocked by config-driven activation

S25 identified config-driven activation as a prerequisite for signal. This remains the primary blocker for the next mesh expansion. The signal family is fully planned but cannot be implemented without the activation mechanism.

**Severity:** Medium (blocks the next major mesh evolution step).

### G6 — raccoon-cli does not validate mesh-level invariants

raccoon-cli validates contracts, drift, and topology at the code level. It does not validate mesh-level rules like "every stream has exactly one writer" or "family names are reserved". These rules are currently enforced by architecture review only.

**Severity:** Low (the team is small enough for review-based enforcement, but this should evolve with team growth).

## MarketMonkey Absorption

This stage absorbed the following conceptual strengths from MarketMonkey:

| MM Strength | Foundry Absorption |
|-------------|-------------------|
| Stream mesh as explicit architecture | Dedicated mesh model document with layers, dimensions, and properties |
| Source/symbol/timeframe scoping | Partition key dimensions with per-family granularity (observation=source, evidence=source+symbol+timeframe) |
| Stream families as first-class concept | Family catalog with classification, invariants, and addition checklist |
| Natural domain progression | Nine reserved family names following the observation→portfolio progression |
| Separation of concerns between mesh and transport | Dedicated mesh-vs-transport document with vocabulary, mapping, and decision framework |

What was **not** absorbed:
- MarketMonkey's flat subject structure (Foundry uses hierarchical `{family}.{surface}.{aggregate}.{verb}`)
- MarketMonkey's per-symbol streams (Foundry uses per-family streams with subject filters)
- MarketMonkey's implicit ownership (Foundry requires explicit single-writer declarations)

## Recommendations for S27

### R1 — Clarify stream-taxonomy.md role

Position `stream-taxonomy.md` as the transport-layer quick-reference. Add a header noting that `stream-mesh-model.md` is the authoritative architectural document and `stream-taxonomy.md` is the subject encoding reference.

### R2 — Add mesh invariant assertions to raccoon-cli

Extend raccoon-cli's contract audit to validate:
- Every JetStream stream has exactly one producing binary
- Family names in subjects match the reserved list
- No cross-family imports exist in domain code

### R3 — Design signal family contracts

With the mesh model now explicit, the signal family can be designed at the contract level. This would produce: signal event types, subject encoding, consumer specs, query surface, and KV projection design — without implementing code.

### R4 — Evaluate observation partitioning evolution

If future families (signal, decision) need symbol-level observation access, evaluate whether observation subjects should evolve to `observation.events.market.trade.{source}.{symbol}`. This would be an additive subject change (current consumers use `>` wildcards) but needs careful analysis of subject cardinality impact.

### R5 — Prototype projection events

Design the PROJECTION_EVENTS stream and its first consumer (gateway cache invalidation). This is low-priority but would complete the four-surface mesh model.

## Files Modified

No code files were modified. This stage produced architecture documents only.

| File | Action |
|------|--------|
| `docs/architecture/stream-mesh-model.md` | Created |
| `docs/architecture/stream-families.md` | Created |
| `docs/architecture/mesh-vs-transport.md` | Created |
| `docs/stages/stage-s26-stream-mesh-canonicalization-report.md` | Created |

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Stream mesh is explicit and architecturally clear | Met — three documents define mesh model, families, and transport separation |
| Documents help orient future code | Met — family addition checklist, evolution rules, and decision framework provided |
| MarketMonkey relationship absorbed conceptually | Met — absorption table documents what was taken and what was rejected |
| Foundry gains better language for growing new flows | Met — mesh vocabulary, family classification, and surface layers provide shared language |
| No new stream family implemented | Met — no code changes |
| No deep code refactoring | Met — no code changes |
| Documents are not generic lists | Met — every section has architectural implications and concrete rules |
| Current, future, and prohibited clearly separated | Met — active/planned/future/reserved status per family |
