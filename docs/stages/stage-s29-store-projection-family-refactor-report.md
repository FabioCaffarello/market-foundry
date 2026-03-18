# S29 — Store Projection Family Refactor

**Stage:** S29
**Type:** Refactor + Architecture
**Status:** Complete
**Date:** 2025-03-17
**Depends on:** S28 (Derive Refactor by Stream Families)

## Objective

Refactor the store binary so its projection pipelines are organized by explicit projection families, mirroring the derive binary's FamilyProcessor pattern.

## Problem Statement

Before this refactor, `StoreSupervisor.start()` hardcoded two projection pipeline blocks inline. `ProjectionTrackers` was a struct with named fields per type (`CandleProjection`, `CandleConsumer`, `BurstProjection`, `BurstConsumer`). Adding a new evidence type required:

1. Adding fields to `ProjectionTrackers` struct
2. Adding tracker creation in `cmd/store/run.go`
3. Adding a pipeline block in `StoreSupervisor.start()`
4. Updating health server tracker list in `run.go`

Four touch points for what is structurally the same operation: spawn a consumer + projection actor pair.

## Solution

### ProjectionPipeline

Introduced `ProjectionPipeline` — a declarative struct that describes one evidence type's complete projection pipeline:

```go
type ProjectionPipeline struct {
    Family         string
    ProjectionName string
    ConsumerName   string
    Buckets        []string
    ConsumerSpec   adapternats.ConsumerSpec
    NewProjection  func(natsURL string, tracker *healthz.Tracker) actor.Producer
    NewConsumer    func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.EvidenceRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
}
```

### Dynamic Trackers

Replaced `ProjectionTrackers` struct with `map[string]*healthz.Tracker`. Keys match pipeline actor names by convention (`"candle-projection"`, `"candle-consumer"`, etc.). This eliminates the need to modify a struct definition when adding families.

### Spawning Loop

The supervisor now iterates over registered pipelines:

```go
for _, p := range s.pipelines {
    projPID := ctx.SpawnChild(p.NewProjection(natsURL, projTracker), p.ProjectionName)
    ctx.SpawnChild(p.NewConsumer(natsURL, p.ConsumerSpec, registry, projPID, consTracker), p.ConsumerName)
}
```

## Key Changes

### store_supervisor.go

| Change | Description |
|--------|-------------|
| `+ProjectionPipeline` type | Declarative struct for one projection pipeline |
| `ProjectionTrackers` → `map[string]*healthz.Tracker` | Dynamic tracker lookup by name |
| `StoreSupervisor.pipelines` field | Stores registered projection families |
| `start()` refactored | Builds pipeline list, iterates to spawn, logs families dynamically |

### cmd/store/run.go

| Change | Description |
|--------|-------------|
| Tracker creation | Changed from 4 named variables to a `map[string]*healthz.Tracker` |
| Supervisor construction | Passes map instead of `ProjectionTrackers` struct |
| Health server trackers | Collected dynamically from map values |

### What Did NOT Change

- **CandleProjectionActor** — unchanged, type-safe projection logic
- **TradeBurstProjectionActor** — unchanged
- **EvidenceConsumerActor** — unchanged
- **TradeBurstConsumerActor** — unchanged
- **QueryResponderActor** — unchanged, remains explicitly typed
- **Messages** — `candleReceivedMessage`, `tradeBurstReceivedMessage` unchanged
- **KV stores** — `CandleKVStore`, `TradeBurstKVStore` unchanged
- **NATS adapters** — `EvidenceConsumer`, `TradeBurstConsumer` unchanged
- **Evidence registry** — unchanged
- **HTTP routes/handlers** — unchanged
- **Use cases** — unchanged
- **Domain types** — unchanged

## Files Changed

| File | Change |
|------|--------|
| `internal/actors/scopes/store/store_supervisor.go` | +`ProjectionPipeline` type, `ProjectionTrackers`→map, pipeline registration and spawning loop |
| `cmd/store/run.go` | Tracker creation as map, dynamic collection for health server |

## Files Created

| File | Purpose |
|------|---------|
| `docs/architecture/projection-families-model.md` | Canonical model for store projection families with invariants and comparison tables |
| `docs/architecture/latest-history-by-family.md` | Read model strategy per family: latest-only vs. latest+history, with decision framework |
| `docs/stages/stage-s29-store-projection-family-refactor-report.md` | This report |

## Projection Families Consolidated

| Family | Class | Consumer | Projection Actor | Buckets | Query Routes |
|--------|-------|----------|-----------------|---------|-------------|
| candle | Latest + History | `store-candle` | CandleProjectionActor | CANDLE_LATEST, CANDLE_HISTORY | candle.latest, candle.history |
| tradeburst | Latest-Only | `store-trade-burst` | TradeBurstProjectionActor | TRADE_BURST_LATEST | tradeburst.latest |

### Adding a New Evidence Type Now Requires

| Component | Action | Existing code modified? |
|-----------|--------|------------------------|
| Domain type | New file | No |
| Consumer adapter | New file | No |
| Consumer actor | New file | No |
| Projection actor | New file | No |
| KV store adapter | New file | No |
| Receive message | One line in messages.go | Yes (additive) |
| Registry | One spec + one consumer func | Yes (additive) |
| **Pipeline entry** | One entry in pipelines list | Yes (additive) |
| **Tracker entries** | Two entries in tracker map | Yes (additive) |
| Query route | One handler + route in QueryResponderActor | Yes (additive) |
| **StoreSupervisor** spawning | — | **No** |

The critical improvement: **StoreSupervisor's spawning loop does not need to know which evidence families exist.** It spawns whatever pipelines are registered.

## Symmetry: Derive ↔ Store

| Aspect | Derive (S28) | Store (S29) |
|--------|-------------|-------------|
| Registration type | `FamilyProcessor` | `ProjectionPipeline` |
| Registration point | `DeriveSupervisor.start()` | `StoreSupervisor.start()` |
| Spawning actor | `SourceScopeActor` (family-agnostic) | `StoreSupervisor` (family-agnostic loop) |
| Per-family actors | SamplerActor, TradeBurstSamplerActor | CandleProjectionActor+Consumer, TradeBurstProjectionActor+Consumer |
| Shared resource | EvidencePublisherActor (per source) | QueryResponderActor (shared across families) |
| Trackers | Single tracker (publisher) | Map of trackers (per pipeline × 2) |

Both binaries now follow the same principle: **families are declared, not hardcoded.**

## Test Results

All tests pass:
- `internal/actors/...` — all passed
- `internal/application/derive/...` — all passed
- `internal/application/evidenceclient/...` — all passed
- `internal/domain/evidence/...` — all passed
- `internal/adapters/nats/...` — all passed

One pre-existing test failure in `internal/application/configctl` is unrelated.

## Gaps Still Existing

### G1 — QueryResponderActor remains explicitly typed (accepted)

The QueryResponderActor opens KV stores and registers query routes per type in its `start()` method. Adding a new evidence type still requires adding a KV store, a handler, and a route to this actor. This is the right trade-off: the query boundary is where type safety matters most. Each new type adds ~15 lines to the responder.

### G2 — actor-ownership.md still stale

The canonical ownership document has not been updated since S12. It does not reflect:
- Trade burst actors in derive and store
- Candle history bucket
- The FamilyProcessor and ProjectionPipeline patterns
- Updated cross-binary matrix

### G3 — No projection pipeline tests

There are no integration tests that validate the projection pipeline end-to-end (consumer → projection → KV). Individual actors are tested via their domain logic, but the spawning and wiring is verified only by running the system.

### G4 — Health tracker map iteration order

`cmd/store/run.go` collects trackers from a map for the health server. Go map iteration is non-deterministic, so `/statusz` tracker ordering may vary between restarts. This is cosmetic — it does not affect health checks.

## Recommendations for S30

### R1 — Update actor-ownership.md (HIGH PRIORITY)

This is now the third stage recommending this update. The document is the canonical reference for the entire system's actor/stream/projection relationships. It should reflect:
- Derive: FamilyProcessor pattern, TradeBurstSamplerActor, dual-sampler spawning
- Store: ProjectionPipeline pattern, all 5 actors, all 3 KV buckets
- Cross-binary matrix: correct consumer lists, correct phases

### R2 — Design evidence.volume contracts

Both derive (FamilyProcessor) and store (ProjectionPipeline) are now ready for a third evidence type. Volume is the next natural candidate per stream-family-catalog.md (CF-07).

### R3 — Evaluate QueryResponderActor extraction

As evidence types grow, QueryResponderActor will accumulate KV stores and handlers. At 5+ types, consider whether per-family query actors (splitting the shared responder) would be cleaner. This is not urgent at 2 types.

### R4 — Add projection pipeline smoke test

A lightweight integration test that validates: event published → consumer receives → projection writes to KV → query returns result. This would catch wiring errors early.

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Store clearly oriented by projection families | Met — ProjectionPipeline declares families, supervisor iterates |
| Latest/history per family explicit | Met — dedicated architecture doc with decision framework |
| Query ownership clear | Met — QueryResponderActor serves all, per-family routes documented |
| Pattern ready for new families | Met — adding a family does not modify supervisor spawning loop |
| Architecture gains scalability without losing simplicity | Met — 12-line struct, spawning loop, no framework |
| No data platform created | Met — no analytics, no generic projection framework |
| No analytics expansion | Met — only existing families touched |
| Gateway not coupled to store internals | Met — gateway unchanged, accesses only via NATS query subjects |
| Scope limited to existing families | Met — candle and tradeburst only, no new families added |
