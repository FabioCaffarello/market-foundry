# S28 — Derive Refactor by Stream Families

**Stage:** S28
**Type:** Refactor + Architecture
**Status:** Complete
**Date:** 2025-03-17
**Depends on:** S27 (Stream Family Catalog and Ownership)

## Objective

Refactor the derive binary so its structure explicitly reflects stream families and follows a canonical family processor pattern, rather than hardcoding evidence types in actor spawning logic.

## Problem Statement

Before this refactor, `SourceScopeActor.onActivateSampler` directly constructed both `SamplerActor` (candle) and `TradeBurstSamplerActor` in its body. Adding a new evidence type required:
1. Modifying `SourceScopeActor.onActivateSampler` to add spawn logic
2. Interleaving the new sampler creation with existing types
3. No clear indication of where new families should be registered

The spawning logic was correct but implicit — the list of evidence families existed only as code flow, not as a declared concept.

## Solution

Introduced `FamilyProcessor` — a declarative struct that describes one evidence family's processing pipeline:

```go
type FamilyProcessor struct {
    Family      string
    ActorPrefix string
    NewActor    func(source, symbol string, timeframe time.Duration, publisherPID *actor.PID) actor.Producer
}
```

### Key Changes

1. **`FamilyProcessor` type** added to `source_scope_actor.go` — a simple struct, not an interface.

2. **`SourceScopeConfig.Processors`** — new field carrying the registered processor list.

3. **`SourceScopeActor.onActivateSampler`** — refactored from hardcoded dual-spawn to processor iteration:
   ```
   Before: for each timeframe → spawn SamplerActor, spawn TradeBurstSamplerActor
   After:  for each processor → for each timeframe → spawn via proc.NewActor
   ```

4. **`DeriveSupervisor.start`** — now builds the canonical processor list as the single registration point.

5. **`DeriveSupervisor.processors`** — new field storing the registered families, passed to every SourceScopeActor.

6. **Logging** — both supervisor and source scope now log registered family names at startup.

### What Did NOT Change

- **Sampler actors** — `SamplerActor` and `TradeBurstSamplerActor` remain unchanged. They are concrete, type-safe actors.
- **Messages** — `publishCandleMessage` and `publishTradeBurstMessage` remain separate, typed messages.
- **Publisher** — `EvidencePublisherActor` keeps explicit per-type message handling.
- **NATS adapters** — `EvidencePublisher`, `EvidenceRegistry` unchanged.
- **Application logic** — `CandleSampler`, `TradeBurstSampler` unchanged.
- **Domain types** — `EvidenceCandle`, `EvidenceTradeBurst` unchanged.
- **Consumer** — `ConsumerActor` unchanged (trades still flow to supervisor).
- **Binding watcher** — `BindingWatcherActor` unchanged.

## Files Changed

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/source_scope_actor.go` | Added `FamilyProcessor` type, `Processors` config field, refactored `onActivateSampler` to iterate processors, updated logging |
| `internal/actors/scopes/derive/derive_supervisor.go` | Added `processors` field, built processor list in `start()`, passed to `ensureSourceScope`, updated logging |

## Files Created

| File | Purpose |
|------|---------|
| `docs/architecture/derive-family-processor-pattern.md` | Canonical pattern documentation with step-by-step guide for adding new evidence types |
| `docs/stages/stage-s28-derive-refactor-by-stream-families-report.md` | This report |

## Pattern Consolidated

The derive binary now follows a three-layer pattern for evidence families:

```
Layer 1 — Registration (DeriveSupervisor.start)
    Declares which FamilyProcessors exist.
    Single point of truth for evidence families.

Layer 2 — Spawning (SourceScopeActor.onActivateSampler)
    Iterates registered processors.
    Spawns one sampler actor per (processor × symbol × timeframe).
    Family-agnostic — works with any registered processor.

Layer 3 — Processing (SamplerActor, TradeBurstSamplerActor, ...)
    Per-family, type-safe, self-contained actors.
    Own pure application logic (CandleSampler, TradeBurstSampler).
    Send typed publish messages to shared publisher.
```

### What Adding a New Evidence Type Touches

| Component | Action | Existing code modified? |
|-----------|--------|------------------------|
| Domain type | New file | No |
| Application sampler | New file + test | No |
| Sampler actor | New file | No |
| Publish message | One line in messages.go | Yes (additive) |
| Publisher actor | One case in Receive | Yes (additive) |
| NATS publisher | One method | Yes (additive) |
| Registry | One field | Yes (additive) |
| **FamilyProcessor** | One entry in processors list | Yes (additive) |
| SourceScopeActor | — | **No** |
| DeriveSupervisor routing | — | **No** |
| ConsumerActor | — | **No** |

The critical improvement: **SourceScopeActor no longer needs to know which evidence families exist.** It spawns whatever processors are registered.

## Test Results

All existing tests pass:
- `internal/application/derive` — 11/11 passed (candle sampler + trade burst sampler)
- `internal/domain/evidence` — all passed
- `internal/domain/observation` — all passed
- `cmd/derive` — builds successfully

One pre-existing test failure in `internal/application/configctl` (`TestCompileUseCaseBuildsDefaultArtifactMetadata`) is unrelated to this refactor.

## Risks and Limitations

### R1 — Publisher remains explicitly typed (accepted risk)

The `EvidencePublisherActor` still has one message case per evidence type. This is a deliberate trade-off: type safety at the NATS boundary is more valuable than a generic publish interface. Each new evidence type adds ~12 lines to the publisher actor. At the current growth rate (1-2 types per quarter), this scales well up to 10+ types.

### R2 — No runtime processor discovery

Processors are compiled in. There is no mechanism to enable/disable evidence families at runtime via configuration. This is a separate concern from the FamilyProcessor pattern — config-driven activation would layer on top, not replace it.

### R3 — Processor order affects actor naming

Actors are named sequentially by processor order. Reordering processors would change actor PIDs but not behavior. This is cosmetic and has no runtime impact.

### R4 — No cross-family dependencies

The pattern assumes each family processes trades independently. If a future evidence type depends on another type's output (e.g., a "signal" family reading from candles), it would not fit this pattern and would need its own consumer from EVIDENCE_EVENTS, not a FamilyProcessor.

## Recommendations for S29

### R1 — Update actor-ownership.md

S27 identified that actor-ownership.md is stale (I-01, I-02). With the derive refactor complete, the document should be updated to reflect:
- The FamilyProcessor pattern
- TradeBurstSamplerActor in the derive actor tree
- Store's full actor tree (trade burst projection, candle history)

### R2 — Design evidence.volume contracts

The FamilyProcessor pattern is now proven with two types. The next evidence type (volume) can be designed at the contract level, following the 7-step guide in derive-family-processor-pattern.md.

### R3 — Consider store-side evidence type registry

Store's `StoreSupervisor.start()` similarly hardcodes projection pipeline creation. The same data-driven pattern could be applied: register projection pipelines declaratively. However, store's pipelines are more heterogeneous (some have history buckets, some don't), so the value is lower.

### R4 — Extend raccoon-cli validation

raccoon-cli should validate that every FamilyProcessor entry has a corresponding:
- Domain event type in `internal/domain/evidence/events.go`
- Publish method in `EvidencePublisher`
- Registry spec in `EvidenceRegistry`
- Store consumer spec

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Derive reflects families, not just flows | Met — FamilyProcessor makes families explicit, registered, and logged |
| Processors per family are clear | Met — one FamilyProcessor per evidence type, single registration point |
| Consume/process/publish is more canonical | Met — three-layer pattern (registration → spawning → processing) |
| Pattern ready for new evidence types without excessive duplication | Met — 7-step guide, SourceScopeActor untouched for new types |
| Simplicity preserved | Met — FamilyProcessor is 6 lines, no interfaces, no generics |
| No generic framework created | Met — concrete types, explicit message handling, no plugin system |
| Signal not opened | Met — no signal-related changes |
| Processor and publisher ownership not mixed | Met — processors own sampling, publisher owns NATS encoding |
