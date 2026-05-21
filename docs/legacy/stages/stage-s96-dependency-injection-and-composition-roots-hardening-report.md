# Stage S96 — Dependency Injection and Composition Roots Hardening

**Status:** complete
**Date:** 2026-03-19

## Executive Summary

Hardened the dependency injection discipline and composition root structure across the three densest runtimes (gateway, store, execute). The refactoring eliminated structural duplication, separated composition phases, and introduced a declarative pipeline catalog that collapses 6 per-scope pipeline types into one. No external DI framework was introduced — the architecture remains explicit and constructor-based.

## DI / Composition Root Hardening Applied

### Gateway Runtime

**Before:** `run.go` was a 231-line monolithic function interleaving connection creation, use case wiring, and route assembly. 7 blocks of identical "create gateway → defer close → nil-check → create use cases" pattern.

**After:** Composition split into clear phases:
- `compose.go` — `gatewayConns` struct owns all NATS connections and lifecycle. `buildGatewayConns()` creates connections. `buildRouteDependencies()` wires use cases.
- `run.go` — reduced to 35 lines with 3 visible phases: connections → dependencies → routes+spawn.
- `gateway.go` — factory functions retained as-is (each is clear and self-contained).

### Store Runtime

**Before:** `run.go` had 6 identical struct types (`trackerDef`, `signalTrackerDef`, `decisionTrackerDef`, `strategyTrackerDef`, `riskTrackerDef`, `executionTrackerDef`) and 6 near-identical loops, each doing the same thing with a different family checker.

**After:** Single `trackerDef` struct with an `isEnabled func(PipelineConfig) bool` predicate. All 9 tracker pairs declared in one `allTrackerDefs` slice. Single loop creates all enabled trackers.

**Before:** `store_supervisor.go` had 6 pipeline struct types (`ProjectionPipeline`, `SignalPipeline`, `DecisionPipeline`, `StrategyPipeline`, `RiskPipeline`, `ExecutionPipeline`) differing only in registry type parameter. 6 separate filter-and-spawn blocks. `start()` was 370 lines.

**After:** Single `Pipeline` struct with registry bound via closure in `NewConsumer`. `declarePipelines()` returns all pipelines + registries as a declarative catalog. `start()` is a single filter-and-spawn loop (~80 lines). `PipelineScope` tag enables conditional registry injection into the query responder.

### Execute Runtime

Already clean (93 lines). No structural changes needed — serves as the reference implementation for a well-structured composition root.

### Derive Runtime

Already clean (57 lines). No structural changes needed.

## Files Changed

| File | Action | Description |
|------|--------|-------------|
| `cmd/gateway/compose.go` | **NEW** | Gateway composition root: `gatewayConns`, `buildGatewayConns()`, `buildRouteDependencies()` |
| `cmd/gateway/run.go` | **REWRITTEN** | Reduced from 231 to 35 lines; 3 clear phases |
| `cmd/store/run.go` | **REWRITTEN** | Unified 6 tracker struct types into 1; single `buildTrackers()` function |
| `internal/actors/scopes/store/store_supervisor.go` | **REWRITTEN** | 6 pipeline types → 1 `Pipeline` type; declarative catalog; single spawn loop |
| `docs/architecture/dependency-injection-and-composition-roots.md` | **NEW** | Canonical DI patterns reference |
| `docs/architecture/runtime-assembly-guidelines.md` | **NEW** | Runtime assembly structure and conventions |

## Structural Gains

1. **Gateway `run.go`**: 231 → 35 lines. Phase separation makes the composition root scannable in seconds.
2. **Store `run.go`**: 162 → 108 lines. 6 struct types → 1. 6 loops → 1.
3. **Store supervisor**: 508 → 280 lines. 6 pipeline types → 1. 6 filter-and-spawn blocks → 1. Adding a new pipeline scope requires zero new types.
4. **Consistency**: All runtimes now follow the same 6-phase lifecycle documented in `runtime-assembly-guidelines.md`.
5. **Testability**: `buildGatewayConns()` and `buildRouteDependencies()` are independently callable, enabling future composition tests.
6. **Extensibility**: Adding a new pipeline family to the store requires 1 entry in `declarePipelines()` + 1 entry in `allTrackerDefs` — no new types, no new loops.

## Limits Maintained

- **No DI framework introduced.** All composition remains explicit and constructor-based.
- **No hidden registration.** No `init()` side effects, no reflection, no service locators.
- **Factory functions in `gateway.go` retained.** Each creates its own NATS connection for failure isolation — this is intentional.
- **Actor constructor patterns unchanged.** Hollywood actor Producer/Receiver lifecycle is preserved exactly.
- **Venue adapter selection unchanged.** The activation gate ceremony in `buildVenueAdapter()` retains its explicit security posture.
- **No cosmetic-only changes.** Every modification reduces structural duplication or improves phase separation.

## Preparation for S97

Recommended next steps:

1. **Derive supervisor pipeline catalog.** The derive supervisor's `FamilyProcessor` types could benefit from the same closure-based unification applied to the store. Lower priority since derive has fewer scopes.
2. **Composition root testing.** With `buildGatewayConns()` and `buildRouteDependencies()` extracted, integration tests can verify wiring without spawning a full HTTP server.
3. **Health tracker co-declaration.** The store's `allTrackerDefs` and `declarePipelines()` share the same family metadata. A future stage could generate tracker defs from the pipeline catalog to eliminate the dual declaration.
4. **Runtime lifecycle model documentation.** The shutdown sequence (signal → poison → health server close → defer closers) should be validated under load to confirm ordering guarantees.
